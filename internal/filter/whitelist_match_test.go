package filter

import "testing"

func TestDomainMatchesWhitelist(t *testing.T) {
	tests := []struct {
		domain string
		rule   string
		want   bool
	}{
		{"spotify.com", "*.spotify.com", true},
		{"dealer.spotify.com", "*.spotify.com", true},
		{"gew1-dealer.g2.spotify.com", "*.spotify.com", true},
		{"evilspotify.com", "*.spotify.com", false},
		{"spotify.com.attacker.test", "*.spotify.com", false},
		{"dealer.spotify.com", "dealer.spotify.com", true},
		{"other.spotify.com", "dealer.spotify.com", false},
	}
	for _, test := range tests {
		if got := domainMatchesWhitelist(test.domain, test.rule); got != test.want {
			t.Errorf("domainMatchesWhitelist(%q, %q) = %v, want %v", test.domain, test.rule, got, test.want)
		}
	}
}
