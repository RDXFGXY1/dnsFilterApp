package api

import "testing"

func TestExtensionOriginAllowed(t *testing.T) {
	tests := map[string]bool{
		"chrome-extension://abcdefghijklmnop": true,
		"moz-extension://a-local-id":          true,
		"https://example.com":                 false,
		"http://127.0.0.1:8080":               false,
		"":                                    false,
	}
	for origin, want := range tests {
		if got := extensionOriginAllowed(origin); got != want {
			t.Fatalf("origin %q: got %v, want %v", origin, got, want)
		}
	}
}
