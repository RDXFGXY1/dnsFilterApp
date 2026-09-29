//go:build windows

package netdns

import (
	"context"
	"fmt"
	dnsproto "github.com/miekg/dns"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// setDNS uses netsh to set a static primary DNS server on the interface.
func setDNS(iface, dns string) error {
	query := new(dnsproto.Msg)
	query.SetQuestion(".", dnsproto.TypeNS)
	answer, _, err := (&dnsproto.Client{Timeout: 3 * time.Second}).Exchange(query, "[::1]:53")
	if err != nil || answer == nil || answer.Rcode != dnsproto.RcodeSuccess {
		return fmt.Errorf("IPv6 local DNS is not ready; start the updated server before enabling DNS")
	}
	// Primary DNS (IPv4)
	if out, err := run("netsh", "interface", "ipv4", "set", "dnsservers",
		fmt.Sprintf("name=%s", iface), "static", dns, "primary"); err != nil {
		return fmt.Errorf("netsh set dns failed: %v: %s", err, out)
	}
	if out, err := run("netsh", "interface", "ipv6", "set", "dnsservers", "name="+iface, "source=static", "address=::1", "validate=no"); err != nil {
		return fmt.Errorf("IPv4 was configured but IPv6 failed; protection is incomplete: %v: %s", err, out)
	}
	if out, err := run("ipconfig", "/flushdns"); err != nil {
		return fmt.Errorf("DNS configured but cache flush failed: %v: %s", err, out)
	}
	return nil
}

// resetDNS returns the interface to DHCP-provided DNS.
func resetDNS(iface string) error {
	if out, err := run("netsh", "interface", "ipv4", "set", "dnsservers",
		fmt.Sprintf("name=%s", iface), "dhcp"); err != nil {
		return fmt.Errorf("netsh reset dns failed: %v: %s", err, out)
	}
	if out, err := run("netsh", "interface", "ipv6", "set", "dnsservers", "name="+iface, "source=dhcp"); err != nil {
		return fmt.Errorf("IPv4 reset, but IPv6 reset failed: %v: %s", err, out)
	}
	_, _ = run("ipconfig", "/flushdns")
	return nil
}

// currentDNS returns the configured DNS servers for the interface.
func currentDNS(iface string) (string, error) {
	out, err := run("netsh", "interface", "ipv4", "show", "dnsservers",
		fmt.Sprintf("name=%s", iface))
	if err != nil {
		return "", fmt.Errorf("netsh show dnsservers failed: %v: %s", err, out)
	}
	v4 := firstIPs(out)
	out6, err := run("netsh", "interface", "ipv6", "show", "dnsservers", "name="+iface)
	if err != nil {
		return "", fmt.Errorf("IPv6 DNS status unavailable: %w", err)
	}
	values := []string{}
	if v4 != "" {
		values = append(values, v4)
	}
	for _, field := range strings.Fields(out6) {
		value := strings.Trim(field, ",;[]")
		if i := strings.IndexByte(value, '%'); i >= 0 {
			value = value[:i]
		}
		if ip := net.ParseIP(value); ip != nil {
			values = append(values, ip.String())
		}
	}
	return strings.Join(values, ", "), nil
}

func run(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
