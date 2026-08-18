//go:build windows

package netdns

import (
	"fmt"
	"os/exec"
	"strings"
)

// setDNS uses netsh to set a static primary DNS server on the interface.
func setDNS(iface, dns string) error {
	// Primary DNS (IPv4)
	if out, err := run("netsh", "interface", "ipv4", "set", "dnsservers",
		fmt.Sprintf("name=%s", iface), "static", dns, "primary"); err != nil {
		return fmt.Errorf("netsh set dns failed: %v: %s", err, out)
	}
	return nil
}

// resetDNS returns the interface to DHCP-provided DNS.
func resetDNS(iface string) error {
	if out, err := run("netsh", "interface", "ipv4", "set", "dnsservers",
		fmt.Sprintf("name=%s", iface), "dhcp"); err != nil {
		return fmt.Errorf("netsh reset dns failed: %v: %s", err, out)
	}
	return nil
}

// currentDNS returns the configured DNS servers for the interface.
func currentDNS(iface string) (string, error) {
	out, err := run("netsh", "interface", "ipv4", "show", "dnsservers",
		fmt.Sprintf("name=%s", iface))
	if err != nil {
		return "", fmt.Errorf("netsh show dnsservers failed: %v: %s", err, out)
	}
	return firstIPs(out), nil
}

func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
