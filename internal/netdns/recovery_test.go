package netdns

import (
	"errors"
	"testing"
)

func TestRecoveryDoesNotBypassAnAnsweringResolver(t *testing.T) {
	reset := false
	recovered, e := recoverIfOffline(func() (string, string, bool, error) { return "Ethernet", "127.0.0.1, ::1", true, nil }, func(string) bool { return true }, func() error { reset = true; return nil })
	if e != nil || recovered || reset {
		t.Fatalf("healthy resolver bypassed: %v %v", recovered, e)
	}
}
func TestRecoveryRestoresOnlyUnresponsiveLocalRouting(t *testing.T) {
	resets := 0
	recovered, e := recoverIfOffline(func() (string, string, bool, error) { return "Ethernet", "127.0.0.1, ::1", true, nil }, func(string) bool { return false }, func() error { resets++; return nil })
	if e != nil || !recovered || resets != 1 {
		t.Fatalf("dead resolver not recovered: %v %v %d", recovered, e, resets)
	}
	recovered, e = recoverIfOffline(func() (string, string, bool, error) { return "Ethernet", "192.0.2.53", false, nil }, func(string) bool { t.Fatal("external DNS must not be probed"); return false }, func() error { t.Fatal("external DNS must not be changed"); return nil })
	if e != nil || recovered {
		t.Fatal("nonlocal routing changed")
	}
}
func TestRecoveryRespectsConcurrentDNSChange(t *testing.T) {
	checks := 0
	recovered, e := recoverIfOffline(func() (string, string, bool, error) {
		checks++
		if checks == 1 {
			return "Ethernet", "127.0.0.1", true, nil
		}
		return "Wi-Fi", "192.0.2.53", false, nil
	}, func(string) bool { return false }, func() error { t.Fatal("manual DNS change overwritten"); return nil })
	if e != nil || recovered {
		t.Fatal("concurrent DNS change was not respected")
	}
}
func TestRecoveryPropagatesResetFailure(t *testing.T) {
	failure := errors.New("permission denied")
	_, e := recoverIfOffline(func() (string, string, bool, error) { return "Ethernet", "127.0.0.1", true, nil }, func(string) bool { return false }, func() error { return failure })
	if !errors.Is(e, failure) {
		t.Fatalf("lost reset error: %v", e)
	}
}
