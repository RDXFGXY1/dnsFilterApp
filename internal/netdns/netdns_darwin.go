//go:build darwin

package netdns

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// setDNS sets the DNS server for the network service backing the interface.
func setDNS(iface, dns string) error {
	service, err := serviceForInterface(iface)
	if err != nil {
		return err
	}
	if out, err := run("networksetup", "-setdnsservers", service, dns); err != nil {
		return fmt.Errorf("networksetup failed: %v: %s", err, out)
	}
	return nil
}

// resetDNS clears DNS overrides (restores DHCP-provided servers).
func resetDNS(iface string) error {
	service, err := serviceForInterface(iface)
	if err != nil {
		return err
	}
	if out, err := run("networksetup", "-setdnsservers", service, "Empty"); err != nil {
		return fmt.Errorf("networksetup reset failed: %v: %s", err, out)
	}
	return nil
}

// serviceForInterface maps a BSD device name (e.g. en0) to a networksetup
// service name (e.g. "Wi-Fi") by parsing -listnetworkserviceorder.
func serviceForInterface(iface string) (string, error) {
	out, err := run("networksetup", "-listnetworkserviceorder")
	if err != nil {
		return "", fmt.Errorf("could not list network services: %v: %s", err, out)
	}
	// Blocks look like:
	//   (1) Wi-Fi
	//   (Hardware Port: Wi-Fi, Device: en0)
	re := regexp.MustCompile(`\(\d+\)\s+(.+)\n\(Hardware Port:.*Device:\s*(\w+)\)`)
	for _, m := range re.FindAllStringSubmatch(out, -1) {
		if strings.TrimSpace(m[2]) == iface {
			return strings.TrimSpace(m[1]), nil
		}
	}
	return "", fmt.Errorf("no network service found for interface %s", iface)
}

// currentDNS returns the DNS servers configured for the interface's service.
func currentDNS(iface string) (string, error) {
	service, err := serviceForInterface(iface)
	if err != nil {
		return "", err
	}
	out, err := run("networksetup", "-getdnsservers", service)
	if err != nil {
		return "", err
	}
	return firstIPs(out), nil // empty when "There aren't any DNS Servers set"
}

func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
