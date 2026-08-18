// Package netdns configures the operating system's DNS resolver to point at
// (or away from) the local DNS filter. It provides a small cross-platform
// surface; the actual mechanism is implemented per-OS in the *_<goos>.go files.
package netdns

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// LocalDNS is the address the filter listens on. Pointing the OS resolver here
// makes the filter the "middleman" for every DNS lookup on the machine.
const LocalDNS = "127.0.0.1"

// SetDNS points the active network interface's DNS at the local filter.
// It typically requires administrator/root privileges.
func SetDNS() error {
	iface, err := activeInterface()
	if err != nil {
		return fmt.Errorf("could not determine active network interface: %w", err)
	}
	return setDNS(iface, LocalDNS)
}

// ResetDNS restores the active interface to automatic (DHCP-provided) DNS.
func ResetDNS() error {
	iface, err := activeInterface()
	if err != nil {
		return fmt.Errorf("could not determine active network interface: %w", err)
	}
	return resetDNS(iface)
}

// Status reports the active interface's current DNS server and whether it
// currently points at the local filter ("middleman mode" on).
func Status() (iface, dns string, active bool, err error) {
	iface, err = activeInterface()
	if err != nil {
		return "", "", false, err
	}
	dns, err = currentDNS(iface)
	if err != nil {
		return iface, "", false, err
	}
	return iface, dns, strings.Contains(dns, LocalDNS), nil
}

var ipRe = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)

// firstIPs extracts up to a few IPv4 addresses from arbitrary command output.
func firstIPs(s string) string {
	matches := ipRe.FindAllString(s, -1)
	if len(matches) == 0 {
		return ""
	}
	return strings.Join(matches, ", ")
}

// activeInterface returns the name of the network interface that carries the
// default route. It works without elevated privileges by opening a UDP socket
// toward a public address (no packets are actually sent) and matching the
// chosen local address back to a named interface.
func activeInterface() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localIP := conn.LocalAddr().(*net.UDPAddr).IP

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.Equal(localIP) {
				return iface.Name, nil
			}
		}
	}
	return "", fmt.Errorf("no active interface found for local IP %s", localIP)
}
