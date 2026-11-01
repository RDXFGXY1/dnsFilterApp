package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/gin-gonic/gin"
)

func TestRequestSizeMiddlewareRejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestSizeMiddleware())
	r.POST("/upload", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(make([]byte, maxDashboardRequestBytes+1)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request returned %d: %s", w.Code, w.Body.String())
	}
}

func TestRequestSizeMiddlewareAllowsUploadCeiling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestSizeMiddleware())
	r.POST("/upload", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(make([]byte, maxDashboardRequestBytes)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("request at limit returned %d", w.Code)
	}
}

func TestAllowedDashboardHost(t *testing.T) {
	tests := []struct {
		host       string
		configured string
		want       bool
	}{
		{"localhost:8080", "127.0.0.1", true},
		{"dnsfilter.localhost:8080", "127.0.0.1", true},
		{"guard.nullstudio.org", "127.0.0.1", true},
		{"127.0.0.1:8080", "127.0.0.1", true},
		{"[::1]:8080", "127.0.0.1", true},
		{"192.168.1.9:8080", "192.168.1.9", true},
		{"attacker.example", "127.0.0.1", false},
		{"localhost.attacker.example", "127.0.0.1", false},
		{"192.168.1.9", "0.0.0.0", false},
	}
	for _, tt := range tests {
		if got := allowedDashboardHost(tt.host, tt.configured, "guard.nullstudio.org"); got != tt.want {
			t.Errorf("allowedDashboardHost(%q, %q) = %v, want %v", tt.host, tt.configured, got, tt.want)
		}
	}
}

func TestSecurityMiddlewareHeadersAndHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testSecurityServer()
	r := gin.New()
	r.Use(s.securityMiddleware())
	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("trusted host returned %d", w.Code)
	}
	for _, header := range []string{"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Permissions-Policy"} {
		if w.Header().Get(header) == "" {
			t.Errorf("missing %s", header)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Host = "attacker.example"
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusMisdirectedRequest {
		t.Fatalf("untrusted host returned %d, want %d", w.Code, http.StatusMisdirectedRequest)
	}
}

func TestContentSecurityPolicyUsesPerResponseNonce(t *testing.T) {
	s := testSecurityServer()
	r := gin.New()
	r.Use(s.securityMiddleware())
	r.GET("/nonce", func(c *gin.Context) {
		c.Header("X-Test-CSP-Nonce", c.GetString(cspNonceContextKey))
		c.Status(http.StatusNoContent)
	})
	request := func() (string, string) {
		req := httptest.NewRequest(http.MethodGet, "/nonce", nil)
		req.Host = "localhost:8080"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Header().Get("Content-Security-Policy"), w.Header().Get("X-Test-CSP-Nonce")
	}
	firstPolicy, firstNonce := request()
	secondPolicy, secondNonce := request()
	if firstNonce == "" || secondNonce == "" || firstNonce == secondNonce {
		t.Fatalf("CSP nonces must be non-empty and unique: %q %q", firstNonce, secondNonce)
	}
	if !strings.Contains(firstPolicy, "'nonce-"+firstNonce+"'") || !strings.Contains(secondPolicy, "'nonce-"+secondNonce+"'") {
		t.Fatal("response CSP does not contain its generated nonce")
	}
	if strings.Contains(firstPolicy, "script-src 'self' 'unsafe-inline'") || strings.Contains(firstPolicy, "style-src 'self' 'unsafe-inline'") {
		t.Fatalf("broad inline execution remained enabled: %s", firstPolicy)
	}
	for _, directive := range []string{"default-src 'none'", "object-src 'none'", "frame-ancestors 'none'", "base-uri 'none'", "form-action 'self'"} {
		if !strings.Contains(firstPolicy, directive) {
			t.Errorf("CSP missing %q", directive)
		}
	}
}

func TestSecurityMiddlewareCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := testSecurityServer()
	r := gin.New()
	r.Use(s.securityMiddleware())
	r.POST("/api/change", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/api/read", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := func(method, path, cookieToken, headerToken string) int {
		req := httptest.NewRequest(method, path, nil)
		req.Host = "localhost:8080"
		req.AddCookie(&http.Cookie{Name: "session", Value: "session-one"})
		if cookieToken != "" {
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: cookieToken})
		}
		if headerToken != "" {
			req.Header.Set("X-CSRF-Token", headerToken)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	if got := request(http.MethodPost, "/api/change", "", ""); got != http.StatusForbidden {
		t.Fatalf("missing token returned %d", got)
	}
	if got := request(http.MethodPost, "/api/change", "csrf-one", "wrong"); got != http.StatusForbidden {
		t.Fatalf("mismatched token returned %d", got)
	}
	if got := request(http.MethodPost, "/api/change", "csrf-one", "csrf-one"); got != http.StatusNoContent {
		t.Fatalf("valid token returned %d", got)
	}
	if got := request(http.MethodGet, "/api/read", "", ""); got != http.StatusNoContent {
		t.Fatalf("safe request returned %d", got)
	}
}

func testSecurityServer() *Server {
	return &Server{
		cfg: &config.Config{
			Server:   config.ServerConfig{APIHost: "127.0.0.1"},
			Security: config.SecurityConfig{SessionTimeout: 60},
		},
		sessions:   map[string]time.Time{"session-one": time.Now().Add(time.Hour)},
		csrfTokens: map[string]string{"session-one": "csrf-one"},
	}
}
