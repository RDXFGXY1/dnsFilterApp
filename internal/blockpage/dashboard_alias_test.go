package blockpage

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDashboardAliasRedirectsOnlyLocalExactHost(t *testing.T) {
	b := &BlockPageServer{}
	b.SetDashboardAlias("guard.nullstudio.org", 8080)
	for _, tc := range []struct {
		host, remote string
		handled      bool
		status       int
	}{
		{"guard.nullstudio.org", "127.0.0.1:1234", true, http.StatusPermanentRedirect},
		{"guard.nullstudio.org", "192.0.2.10:1234", true, http.StatusForbidden},
		{"guard.nullstudio.org.attacker.example", "127.0.0.1:1234", false, http.StatusOK},
	} {
		r := httptest.NewRequest(http.MethodGet, "http://"+tc.host+"/login", nil)
		r.RemoteAddr = tc.remote
		w := httptest.NewRecorder()
		if handled := b.serveDashboardAlias(w, r); handled != tc.handled {
			t.Fatalf("%s handled=%v", tc.host, handled)
		}
		if tc.handled && w.Code != tc.status {
			t.Fatalf("%s status=%d", tc.host, w.Code)
		}
		if tc.status == http.StatusPermanentRedirect && w.Header().Get("Location") != "https://guard.nullstudio.org/login" {
			t.Fatal("wrong redirect")
		}
	}
}
