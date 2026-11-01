package dns

import (
	"errors"
	"net"
	"testing"
)

func TestWindowsWildcardDNSCanFallBackToLoopback(t *testing.T) {
	calls := []string{}
	conn, e := bindDNSPacket("udp4", "0.0.0.0:53", true, func(network, address string) (net.PacketConn, error) {
		calls = append(calls, address)
		if len(calls) == 1 {
			return nil, errors.New("wildcard occupied by ICS")
		}
		return net.ListenPacket("udp4", "127.0.0.1:0")
	})
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	if len(calls) != 2 || calls[1] != "127.0.0.1:53" {
		t.Fatalf("wrong fallback: %v", calls)
	}
}
func TestDNSFallbackDoesNotChangeExplicitBindingsOrOtherPlatforms(t *testing.T) {
	for _, tc := range []struct {
		windows bool
		address string
	}{{false, "0.0.0.0:53"}, {true, "192.0.2.1:53"}, {true, "127.0.0.1:53"}} {
		calls := 0
		_, e := bindDNSPacket("udp4", tc.address, tc.windows, func(string, string) (net.PacketConn, error) { calls++; return nil, errors.New("occupied") })
		if e == nil || calls != 1 {
			t.Fatalf("unexpected fallback for %+v (%d calls)", tc, calls)
		}
	}
}
func TestDNSListenerKeepsSuccessfulWildcard(t *testing.T) {
	calls := 0
	conn, e := bindDNSPacket("udp4", "0.0.0.0:0", true, func(network, address string) (net.PacketConn, error) {
		calls++
		return net.ListenPacket(network, address)
	})
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	if calls != 1 {
		t.Fatal("successful listener was retried")
	}
}
