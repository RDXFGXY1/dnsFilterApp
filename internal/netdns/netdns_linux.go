//go:build linux

package netdns

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	resolvConf   = "/etc/resolv.conf"
	resolvBackup = "/etc/resolv.conf.dnsfilter.bak"
)

// setDNS prefers systemd-resolved (resolvectl); if that is unavailable it
// backs up and rewrites /etc/resolv.conf.
func setDNS(iface, dns string) error {
	if hasResolvectl() {
		if out, err := run("resolvectl", "dns", iface, dns); err != nil {
			return fmt.Errorf("resolvectl failed: %v: %s", err, out)
		}
		return nil
	}
	if hasNetworkManager() {
		return setNetworkManagerDNS(iface, dns)
	}
	return writeResolvConf(dns)
}

// resetDNS reverts systemd-resolved or restores the resolv.conf backup.
func resetDNS(iface string) error {
	if hasResolvectl() {
		if out, err := run("resolvectl", "revert", iface); err != nil {
			return fmt.Errorf("resolvectl revert failed: %v: %s", err, out)
		}
		return nil
	}
	if hasNetworkManager() {
		return resetNetworkManagerDNS(iface)
	}
	return restoreResolvConf()
}

func hasResolvectl() bool {
	if _, err := exec.LookPath("resolvectl"); err != nil {
		return false
	}
	_, err := run("resolvectl", "status")
	return err == nil
}

func hasNetworkManager() bool {
	if _, err := exec.LookPath("nmcli"); err != nil {
		return false
	}
	out, err := run("nmcli", "-t", "-f", "RUNNING", "general")
	return err == nil && strings.EqualFold(strings.TrimSpace(out), "running")
}

func networkManagerConnection(iface string) (string, error) {
	out, err := run("nmcli", "-g", "GENERAL.CONNECTION", "device", "show", iface)
	if err != nil || strings.TrimSpace(out) == "" || strings.TrimSpace(out) == "--" {
		return "", fmt.Errorf("NetworkManager has no active connection for %s: %v", iface, err)
	}
	return strings.TrimSpace(strings.Split(out, "\n")[0]), nil
}

func setNetworkManagerDNS(iface, dns string) error {
	connection, err := networkManagerConnection(iface)
	if err != nil {
		return err
	}
	if out, err := run("nmcli", "connection", "modify", connection,
		"ipv4.dns", dns, "ipv4.ignore-auto-dns", "yes",
		"ipv6.dns", "", "ipv6.ignore-auto-dns", "yes"); err != nil {
		return fmt.Errorf("NetworkManager DNS setup failed: %v: %s", err, out)
	}
	if out, err := run("nmcli", "connection", "up", connection); err != nil {
		return fmt.Errorf("NetworkManager could not reactivate %s: %v: %s", connection, err, out)
	}
	return nil
}

func resetNetworkManagerDNS(iface string) error {
	connection, err := networkManagerConnection(iface)
	if err != nil {
		return err
	}
	if out, err := run("nmcli", "connection", "modify", connection,
		"ipv4.dns", "", "ipv4.ignore-auto-dns", "no",
		"ipv6.dns", "", "ipv6.ignore-auto-dns", "no"); err != nil {
		return fmt.Errorf("NetworkManager DNS reset failed: %v: %s", err, out)
	}
	if out, err := run("nmcli", "connection", "up", connection); err != nil {
		return fmt.Errorf("NetworkManager could not reactivate %s: %v: %s", connection, err, out)
	}
	return nil
}

func writeResolvConf(dns string) error {
	// Back up the current resolv.conf once so reset can restore it.
	if _, err := os.Stat(resolvBackup); os.IsNotExist(err) {
		if data, err := os.ReadFile(resolvConf); err == nil {
			_ = os.WriteFile(resolvBackup, data, 0644)
		}
	}
	content := fmt.Sprintf("# Managed by dns-filter\nnameserver %s\n", dns)
	if err := os.WriteFile(resolvConf, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s (need root?): %w", resolvConf, err)
	}
	return nil
}

func restoreResolvConf() error {
	data, err := os.ReadFile(resolvBackup)
	if err != nil {
		return fmt.Errorf("no backup at %s to restore: %w", resolvBackup, err)
	}
	if err := os.WriteFile(resolvConf, data, 0644); err != nil {
		return fmt.Errorf("failed to restore %s (need root?): %w", resolvConf, err)
	}
	_ = os.Remove(resolvBackup)
	return nil
}

// currentDNS returns the DNS servers currently in use for the interface.
func currentDNS(iface string) (string, error) {
	if hasResolvectl() {
		if out, err := run("resolvectl", "dns", iface); err == nil {
			return firstIPs(out), nil
		}
	}
	if hasNetworkManager() {
		connection, err := networkManagerConnection(iface)
		if err == nil {
			if out, err := run("nmcli", "-g", "ipv4.dns", "connection", "show", connection); err == nil {
				return firstIPs(out), nil
			}
		}
	}
	data, err := os.ReadFile(resolvConf)
	if err != nil {
		return "", err
	}
	return firstIPs(string(data)), nil
}

func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
