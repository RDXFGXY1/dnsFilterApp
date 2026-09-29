package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/netdns"
	"github.com/RDXFGXY1/dns-filter-app/internal/systemcheck"
	"github.com/gin-gonic/gin"
	dnsproto "github.com/miekg/dns"
)

type setupPreflightCheck struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	OK       bool   `json:"ok"`
	Required bool   `json:"required"`
	Detail   string `json:"detail"`
}

func (s *Server) handleFirstRunSetupPage(c *gin.Context) {
	if s.cfg.Security.IsSetupComplete() {
		if s.validSession(c) {
			c.Redirect(http.StatusFound, "/")
		} else {
			c.Redirect(http.StatusFound, "/login")
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "setup.html", gin.H{"title": "Secure setup", "csp_nonce": c.GetString(cspNonceContextKey)})
}

func (s *Server) handleFirstRunPreflight(c *gin.Context) {
	if !directLoopbackRequest(c.Request.RemoteAddr) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Initial setup checks are available only from this computer"})
		return
	}
	if s.cfg.Security.IsSetupComplete() {
		c.JSON(http.StatusConflict, gin.H{"error": "Initial setup has already been completed"})
		return
	}

	adminOK, adminDetail := systemcheck.IsAdministrator()
	firewallOK, firewallDetail := systemcheck.FirewallStatus()
	dnsReady, dnsDetail := setupDNSReady(s.cfg.Server.DNSPort)
	iface, resolvers, routed, routeErr := netdns.Status()
	routeDetail := "System DNS is routed through the local filter"
	if routeErr != nil {
		routeDetail = "System DNS route could not be verified: " + routeErr.Error()
	} else if !routed {
		routeDetail = "System DNS is not routed through the local filter yet"
	} else if iface != "" {
		routeDetail += " on " + iface
	}

	checks := []setupPreflightCheck{
		{ID: "service", Label: "Local protection service", OK: true, Required: true, Detail: "The local setup service is responding"},
		{ID: "dns_listener", Label: "DNS listener", OK: dnsReady, Required: true, Detail: dnsDetail},
		{ID: "administrator", Label: "Administrator rights", OK: adminOK, Required: false, Detail: adminDetail},
		{ID: "firewall", Label: "Windows Firewall", OK: firewallOK, Required: false, Detail: firewallDetail},
		{ID: "dns_route", Label: "System DNS route", OK: routed, Required: false, Detail: routeDetail},
	}
	ready := true
	for _, check := range checks {
		if check.Required && !check.OK {
			ready = false
		}
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ready": ready, "checks": checks, "dns": resolvers, "checked_at": time.Now().UTC().Format(time.RFC3339)})
}

func setupDNSReady(port int) (bool, string) {
	query := new(dnsproto.Msg)
	query.SetQuestion(".", dnsproto.TypeNS)
	client := &dnsproto.Client{Timeout: 2 * time.Second}
	answer, _, err := client.Exchange(query, net.JoinHostPort("127.0.0.1", fmt.Sprint(port)))
	if err != nil {
		return false, "The local DNS listener did not answer on port " + fmt.Sprint(port)
	}
	if answer == nil || answer.Rcode != dnsproto.RcodeSuccess {
		return false, "The local DNS listener answered but its upstream resolver is not ready"
	}
	return true, "The local DNS listener is answering securely on port " + fmt.Sprint(port)
}

func (s *Server) handleFirstRunSetup(c *gin.Context) {
	if !directLoopbackRequest(c.Request.RemoteAddr) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Initial setup is available only from this computer"})
		return
	}

	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	if s.cfg.Security.IsSetupComplete() {
		c.JSON(http.StatusConflict, gin.H{"error": "Initial setup has already been completed"})
		return
	}

	var input struct {
		Username           string `json:"username"`
		Password           string `json:"password"`
		VaultPassword      string `json:"vault_password"`
		VaultUnlockMinutes int    `json:"vault_unlock_minutes"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if !validAdminUsername(input.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username must contain 3–32 letters, numbers, dots, dashes, or underscores"})
		return
	}
	if message := strongPasswordError(input.Username, input.Password); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	if s.vault == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Secure Vault is unavailable; setup cannot safely continue"})
		return
	}
	if s.vault.Configured() {
		c.JSON(http.StatusConflict, gin.H{"error": "A Vault already exists while account setup is incomplete; recover or remove the orphaned Vault first"})
		return
	}
	if len(input.VaultPassword) < 12 || len(input.VaultPassword) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vault password must contain 12–128 characters"})
		return
	}
	if input.VaultUnlockMinutes != 5 && input.VaultUnlockMinutes != 15 && input.VaultUnlockMinutes != 30 && input.VaultUnlockMinutes != 60 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Choose a Vault auto-lock period of 5, 15, 30, or 60 minutes"})
		return
	}

	passwordHash, err := config.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not secure the administrator password"})
		return
	}
	recoveryKey, recoveryHash, err := newRecoveryKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate the recovery key"})
		return
	}
	token, err := s.createSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create a secure session"})
		return
	}
	forgetSession := func() {
		s.authMu.Lock()
		delete(s.sessions, token)
		delete(s.csrfTokens, token)
		s.authMu.Unlock()
	}

	previous := s.cfg.Security
	previousUnlockMinutes := s.cfg.Vault.UnlockMinutes
	s.cfg.Security.AdminUsername = input.Username
	s.cfg.Security.AdminPasswordHash = passwordHash
	s.cfg.Security.RecoveryKeyHash = recoveryHash
	s.cfg.Security.SetupComplete = true
	s.cfg.Security.JWTSecret = ""
	s.cfg.Vault.UnlockMinutes = input.VaultUnlockMinutes
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Security = previous
		s.cfg.Vault.UnlockMinutes = previousUnlockMinutes
		forgetSession()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save the secure setup"})
		return
	}
	s.vault.SetUnlockDuration(time.Duration(input.VaultUnlockMinutes) * time.Minute)
	vaultRecoveryCode, err := s.vault.Setup(input.VaultPassword)
	if err != nil {
		s.cfg.Security = previous
		s.cfg.Vault.UnlockMinutes = previousUnlockMinutes
		if rollbackErr := s.cfg.SaveDefault(); rollbackErr != nil {
			if s.log != nil {
				s.log.Errorf("critical setup rollback failure after Vault error: %v", rollbackErr)
			}
			forgetSession()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Secure Vault creation failed and the previous configuration could not be restored", "code": "setup_rollback_failed"})
			return
		}
		forgetSession()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create the Secure Vault: " + err.Error()})
		return
	}
	s.setSessionCookie(c, token)
	c.JSON(http.StatusOK, gin.H{
		"success":              true,
		"csrf_token":           s.csrfToken(token),
		"redirect":             "/",
		"recovery_key":         recoveryKey,
		"vault_recovery_code":  vaultRecoveryCode,
		"vault_unlock_minutes": input.VaultUnlockMinutes,
		"recovery_warning":     "Save this account recovery key now. It will not be shown again.",
	})
}

func directLoopbackRequest(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func validAdminUsername(username string) bool {
	if len(username) < 3 || len(username) > 32 {
		return false
	}
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func strongPasswordError(username, password string) string {
	if len(password) < 12 {
		return "Password must contain at least 12 characters"
	}
	if len(password) > 128 {
		return "Password must contain no more than 128 characters"
	}
	if strings.Contains(strings.ToLower(password), strings.ToLower(username)) {
		return "Password must not contain the administrator username"
	}
	classes := 0
	var lower, upper, digit, symbol bool
	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			lower = true
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			symbol = true
		}
	}
	for _, present := range []bool{lower, upper, digit, symbol} {
		if present {
			classes++
		}
	}
	if classes < 3 {
		return "Password must combine at least three of: lowercase, uppercase, numbers, and symbols"
	}
	return ""
}
