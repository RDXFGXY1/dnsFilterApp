package dns

import (
	"testing"
	"time"
)

func TestEmergencyLockdownDecision(t *testing.T) {
	server := &Server{}
	if err := server.StartEmergencyLockdown(time.Hour, []string{"Example.com", "https://updates.example.org/path", "example.com"}); err != nil {
		t.Fatalf("StartEmergencyLockdown() error = %v", err)
	}

	status := server.EmergencyLockdownStatus()
	if !status.Active || len(status.AllowedDomains) != 2 {
		t.Fatalf("status = %#v", status)
	}
	tests := []struct {
		domain  string
		blocked bool
	}{
		{"example.com.", false},
		{"cdn.example.com.", false},
		{"updates.example.org.", false},
		{"notexample.com.", true},
		{"malware.test.", true},
	}
	for _, tt := range tests {
		blocked, active := server.lockdownDecision(tt.domain)
		if !active || blocked != tt.blocked {
			t.Errorf("lockdownDecision(%q) = (%v, %v), want (%v, true)", tt.domain, blocked, active, tt.blocked)
		}
	}
}

func TestEmergencyLockdownStopsAndExpires(t *testing.T) {
	server := &Server{}
	if err := server.StartEmergencyLockdown(time.Hour, nil); err != nil {
		t.Fatal(err)
	}
	server.StopEmergencyLockdown()
	if _, active := server.lockdownDecision("example.com"); active {
		t.Fatal("lockdown remained active after stop")
	}

	server.lockdownMu.Lock()
	server.lockdownUntil = time.Now().Add(-time.Second).Unix()
	server.lockdownMu.Unlock()
	if status := server.EmergencyLockdownStatus(); status.Active || status.Until != 0 {
		t.Fatalf("expired status = %#v", status)
	}
}

func TestEmergencyLockdownRejectsUnsafeDuration(t *testing.T) {
	server := &Server{}
	if err := server.StartEmergencyLockdown(25*time.Hour, nil); err == nil {
		t.Fatal("duration above 24 hours was accepted")
	}
}
