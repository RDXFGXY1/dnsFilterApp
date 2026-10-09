//go:build linux

package connectionfreeze

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const linuxFreezeTable = "dnsfilter_freeze"

type linuxBackend struct {
	mu          sync.Mutex
	closed      bool
	lastPackets uint64
}

var nftBinary = "nft"

func supported() bool {
	_, err := exec.LookPath(nftBinary)
	return err == nil
}

func startPlatform() (backend, error) {
	if !supported() {
		return nil, fmt.Errorf("Connection Freeze requires nftables; install it with: sudo pacman -S nftables")
	}
	if os.Geteuid() != 0 {
		return nil, fmt.Errorf("Connection Freeze requires the DNS Filter system service to run as root")
	}

	// A stale table can remain only after an unclean process termination. Removing
	// this private table cannot affect distro or user firewall rules.
	_ = runNftCommand("delete", "table", "inet", linuxFreezeTable)

	script := `add table inet dnsfilter_freeze
add chain inet dnsfilter_freeze output { type filter hook output priority -10; policy accept; }
add rule inet dnsfilter_freeze output oifname "lo" accept
add rule inet dnsfilter_freeze output ip daddr 127.0.0.0/8 accept
add rule inet dnsfilter_freeze output ip6 daddr ::1 accept
add rule inet dnsfilter_freeze output ct state established,related accept
add rule inet dnsfilter_freeze output ip protocol icmp accept
add rule inet dnsfilter_freeze output meta l4proto ipv6-icmp accept
add rule inet dnsfilter_freeze output udp dport { 53, 67, 68, 546, 547 } accept
add rule inet dnsfilter_freeze output tcp dport { 53, 853 } accept
add rule inet dnsfilter_freeze output counter drop comment "dns-filter-connection-freeze"
`
	if err := runNftScript(script); err != nil {
		_ = runNftCommand("delete", "table", "inet", linuxFreezeTable)
		return nil, err
	}
	// Do not report an active freeze unless the kernel can list the installed
	// tagged drop rule. This catches partial/unsupported nftables transactions.
	if _, found, err := linuxDropCounter(); err != nil || !found {
		_ = runNftCommand("delete", "table", "inet", linuxFreezeTable)
		if err != nil {
			return nil, fmt.Errorf("verify Connection Freeze rule: %w", err)
		}
		return nil, fmt.Errorf("verify Connection Freeze rule: tagged drop rule was not installed")
	}
	return &linuxBackend{}, nil
}

func runNftScript(script string) error {
	cmd := exec.Command(nftBinary, "-f", "-")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("nftables: %s", message)
	}
	return nil
}

func runNftCommand(args ...string) error {
	out, err := exec.Command(nftBinary, args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("nftables: %s", message)
	}
	return nil
}

func (b *linuxBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	err := runNftCommand("delete", "table", "inet", linuxFreezeTable)
	if err == nil {
		b.closed = true
	}
	return err
}

func (b *linuxBackend) Events() ([]Event, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, nil
	}
	packets, err := linuxDropPackets()
	if err != nil {
		return nil, err
	}
	if packets <= b.lastPackets {
		b.lastPackets = packets
		return nil, nil
	}
	delta := packets - b.lastPackets
	b.lastPackets = packets
	return []Event{{
		Time:     time.Now().UTC().Format(time.RFC3339Nano),
		Action:   "blocked",
		App:      "system",
		Remote:   "multiple destinations",
		Protocol: "IP",
		Reason:   fmt.Sprintf("Connection Freeze blocked %d new outbound packet(s) on Linux", delta),
	}}, nil
}

func linuxDropPackets() (uint64, error) {
	packets, found, err := linuxDropCounter()
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, fmt.Errorf("nftables connection log: tagged drop rule is missing")
	}
	return packets, nil
}

func linuxDropCounter() (uint64, bool, error) {
	out, err := exec.Command(nftBinary, "-j", "list", "chain", "inet", linuxFreezeTable, "output").CombinedOutput()
	if err != nil {
		return 0, false, fmt.Errorf("nftables connection log: %s", strings.TrimSpace(string(out)))
	}
	var document struct {
		Nftables []json.RawMessage `json:"nftables"`
	}
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&document); err != nil {
		return 0, false, fmt.Errorf("decode nftables counters: %w", err)
	}
	var total uint64
	found := false
	for _, item := range document.Nftables {
		var wrapped struct {
			Rule struct {
				Comment string            `json:"comment"`
				Expr    []json.RawMessage `json:"expr"`
			} `json:"rule"`
		}
		if json.Unmarshal(item, &wrapped) != nil || wrapped.Rule.Comment != "dns-filter-connection-freeze" {
			continue
		}
		found = true
		for _, expression := range wrapped.Rule.Expr {
			var counter struct {
				Counter *struct {
					Packets uint64 `json:"packets"`
				} `json:"counter"`
			}
			if json.Unmarshal(expression, &counter) == nil && counter.Counter != nil {
				total += counter.Counter.Packets
			}
		}
	}
	return total, found, nil
}
