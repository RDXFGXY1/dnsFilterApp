//go:build windows

package connectionfreeze

import (
	"crypto/tls"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Opt-in: briefly applies real WFP rules. Run elevated with a test timeout.
func TestLiveFreeze(t *testing.T) {
	if os.Getenv("DNS_FILTER_WFP_TEST") != "1" {
		t.Skip("requires explicit elevated live test")
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 3 * time.Second}, "tcp", "1.1.1.1:443", &tls.Config{ServerName: "cloudflare-dns.com", MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatal("baseline connection", err)
	}
	defer conn.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	b, err := startWindows(filepath.Join(os.Getenv("SystemRoot"), "System32", "notepad.exe"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	c, err := net.DialTimeout("tcp", "1.1.1.1:443", time.Second)
	if err == nil {
		c.Close()
		t.Fatal("new external connection was allowed")
	}
	t.Log("New external TCP connection blocked")
	local, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal("loopback blocked", err)
	}
	local.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err = conn.Write([]byte("HEAD / HTTP/1.1\r\nHost: cloudflare-dns.com\r\nConnection: close\r\n\r\n")); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, 256)
	if n, err := conn.Read(response); err != nil || n == 0 {
		t.Fatalf("established connection did not survive: %v", err)
	}
	t.Log("Established TLS connection and new loopback connection survived")
	var events []Event
	for i := 0; i < 10; i++ {
		events, err = b.Events()
		if err != nil {
			t.Fatal(err)
		}
		if len(events) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if len(events) == 0 {
		t.Error("No blocked connection events received")
	}
	for _, e := range events {
		t.Logf("Logged %s %s → %s", e.Protocol, e.App, e.Remote)
	}
	if err = b.Close(); err != nil {
		t.Fatal(err)
	}
	c, err = net.DialTimeout("tcp", "1.1.1.1:443", 3*time.Second)
	if err != nil {
		t.Fatal("connection not restored", err)
	}
	c.Close()
	t.Log("New connections restored after stop")
}
