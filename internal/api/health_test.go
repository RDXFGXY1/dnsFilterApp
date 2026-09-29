package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appversion "github.com/RDXFGXY1/dns-filter-app/internal/version"
	"github.com/gin-gonic/gin"
)

func TestHealthIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{}
	r := gin.New()
	r.GET("/healthz", s.handleHealth)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("health endpoint returned %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"service":"dns-filter"`) || !strings.Contains(body, `"version":"`+appversion.Current+`"`) || !strings.Contains(body, `"ready":true`) {
		t.Fatalf("health identity is incomplete: %s", body)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("health response can be cached")
	}
}
