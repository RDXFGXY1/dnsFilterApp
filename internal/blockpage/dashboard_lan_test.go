package blockpage

import (
	"net"
	"testing"
)

func TestDashboardLANClientAllowed(t *testing.T) {
	allowed := []string{"192.168.1.25"}
	for _, tc := range []struct {
		name    string
		ip      string
		enabled bool
		tls     bool
		want    bool
	}{
		{"allowed private client", "192.168.1.25", true, true, true},
		{"disabled", "192.168.1.25", false, true, false},
		{"http", "192.168.1.25", true, false, false},
		{"unlisted private client", "192.168.1.26", true, true, false},
		{"public client", "8.8.8.8", true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := dashboardLANClientAllowed(net.ParseIP(tc.ip), tc.enabled, allowed, tc.tls); got != tc.want {
				t.Fatalf("allowed=%v, want %v", got, tc.want)
			}
		})
	}
}
