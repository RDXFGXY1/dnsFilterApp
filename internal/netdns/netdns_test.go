package netdns

import "testing"

func TestOnlyLocalResolvers(t *testing.T) {
	for _, tc := range []struct {
		values string
		want   bool
	}{
		{"127.0.0.1, ::1", true}, {"127.0.0.1, fe80::1", false},
		{"127.0.0.1, 192.168.1.1", false}, {"", false}, {"::1", true},
	} {
		if got := onlyLocalResolvers(tc.values); got != tc.want {
			t.Errorf("%q: got %v, want %v", tc.values, got, tc.want)
		}
	}
}
