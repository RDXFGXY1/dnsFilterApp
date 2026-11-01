package dns

import (
	"fmt"
	"net"
)

func bindDNSPacket(network, address string, windows bool, listen func(string, string) (net.PacketConn, error)) (net.PacketConn, error) {
	conn, err := listen(network, address)
	if err == nil {
		return conn, nil
	}
	host, port, splitErr := net.SplitHostPort(address)
	ip := net.ParseIP(host)
	if !windows || splitErr != nil || (host != "" && (ip == nil || !ip.IsUnspecified())) {
		return nil, fmt.Errorf("DNS listener %s: %w", address, err)
	}
	loopback := "127.0.0.1"
	if network == "udp6" {
		loopback = "::1"
	}
	local := net.JoinHostPort(loopback, port)
	conn, localErr := listen(network, local)
	if localErr != nil {
		return nil, fmt.Errorf("DNS listener %s failed (%v); local fallback %s also failed: %w", address, err, local, localErr)
	}
	return conn, nil
}
