package api

import "testing"

func TestAIProviderEndpointsArePinnedOrSafe(t *testing.T) {
	cases := []struct {
		provider, raw, want string
		ok                  bool
	}{
		{"openai", "http://evil.test", "https://api.openai.com/v1", true},
		{"xai", "", "https://api.x.ai/v1", true},
		{"gemini", "", "https://generativelanguage.googleapis.com/v1beta", true},
		{"compatible", "https://models.example/v1/", "https://models.example/v1", true},
		{"compatible", "http://127.0.0.1:11434/v1", "http://127.0.0.1:11434/v1", true},
		{"compatible", "http://192.168.1.7/v1", "", false},
	}
	for _, tc := range cases {
		got, err := aiProviderEndpoint(tc.provider, tc.raw)
		if (err == nil) != tc.ok || got != tc.want {
			t.Fatalf("%s %q: got=%q err=%v", tc.provider, tc.raw, got, err)
		}
	}
}
