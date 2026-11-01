package api

import "testing"

func TestValidLocalDashboardDomain(t *testing.T) {
	for _, input := range []string{"guard.nullstudio.org", "My-Guard.example.org", "console.example.co.uk"} {
		if _, ok := validLocalDashboardDomain(input); !ok {
			t.Fatalf("rejected %q", input)
		}
	}
	for _, input := range []string{"http://guard.nullstudio.org", "guard.nullstudio.org:8080", "localhost", "foo.local", "127.0.0.1", "-bad.example.org", "bad..example.org", "example.org/path"} {
		if _, ok := validLocalDashboardDomain(input); ok {
			t.Fatalf("accepted %q", input)
		}
	}
}
