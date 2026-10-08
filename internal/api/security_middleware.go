package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxDashboardRequestBytes int64 = 6 << 20
const maxPluginFileRequestBytes int64 = 52 << 20
const cspNonceContextKey = "csp_nonce"

func requestSizeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requestMayHaveBody(c.Request.Method) {
			c.Next()
			return
		}
		limit := maxDashboardRequestBytes
		if strings.HasPrefix(c.Request.URL.Path, "/api/plugins/") && strings.HasSuffix(c.Request.URL.Path, "/file") {
			limit = maxPluginFileRequestBytes
		}
		if c.Request.ContentLength > limit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body is too large",
				"code":  "request_too_large",
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func requestMayHaveBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (s *Server) securityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce := newCSPNonce()
		c.Set(cspNonceContextKey, nonce)
		setSecurityHeaders(c, nonce)
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "no-store")
		}

		if !allowedDashboardHost(c.Request.Host, s.cfg.Server.APIHost) {
			c.AbortWithStatusJSON(http.StatusMisdirectedRequest, gin.H{
				"error": "This local dashboard address is not trusted",
				"code":  "untrusted_host",
			})
			return
		}

		if requiresCSRF(c.Request.Method, c.Request.URL.Path) && s.validSession(c) {
			session, _ := c.Cookie("session")
			cookieToken, cookieErr := c.Cookie("csrf_token")
			headerToken := c.GetHeader("X-CSRF-Token")
			expected := s.csrfToken(session)
			if cookieErr != nil || !secureEqual(expected, cookieToken) || !secureEqual(expected, headerToken) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "Security token missing or invalid. Refresh the page and try again.",
					"code":  "csrf_failed",
				})
				return
			}
		}

		c.Next()
	}
}

func setSecurityHeaders(c *gin.Context, nonce string) {
	c.Header("Content-Security-Policy", contentSecurityPolicy(nonce))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	c.Header("Cross-Origin-Opener-Policy", "same-origin")
}

func newCSPNonce() string {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		panic("secure CSP nonce generation failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func contentSecurityPolicy(nonce string) string {
	// Event and style attributes are temporarily scoped to their own directives
	// while legacy templates are migrated to delegated listeners and CSS classes.
	// Inline script/style elements are allowed only with this response's nonce.
	return "default-src 'none'; " +
		"script-src 'self' 'nonce-" + nonce + "'; script-src-elem 'self' 'nonce-" + nonce + "'; script-src-attr 'unsafe-inline'; " +
		"style-src 'self'; style-src-elem 'self'; style-src-attr 'unsafe-inline'; " +
		"img-src 'self' data:; font-src 'self' data:; connect-src 'self'; " +
		"object-src 'none'; media-src 'none'; worker-src 'none'; manifest-src 'none'; " +
		"base-uri 'none'; frame-src http://127.0.0.1:* http://localhost:*; frame-ancestors 'none'; form-action 'self'"
}

func requiresCSRF(method, path string) bool {
	if !strings.HasPrefix(path, "/api/") {
		return false
	}
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func secureEqual(expected, actual string) bool {
	if expected == "" || len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func allowedDashboardHost(requestHost, configuredHost string) bool {
	host := normalizeRequestHost(requestHost)
	if host == "" {
		return false
	}
	if host == "localhost" || host == "dnsfilter.localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	configured := normalizeRequestHost(configuredHost)
	if configured == "" || configured == "0.0.0.0" || configured == "::" {
		return false
	}
	return host == configured
}

func normalizeRequestHost(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	} else if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.Trim(value, "[]")
	}
	return strings.TrimSuffix(value, ".")
}
