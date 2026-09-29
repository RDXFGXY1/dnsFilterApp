package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/vault"
	"github.com/gin-gonic/gin"
)

func TestRecoveryKeyGenerationAndNormalization(t *testing.T) {
	key, hash, err := newRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(key) < 32 || !strings.Contains(key, "-") {
		t.Fatalf("unexpected recovery key format: %q", key)
	}
	if !verifyPassword(normalizeRecoveryKey(strings.ToLower(key)), hash) {
		t.Fatal("generated recovery key does not match its hash")
	}
	if verifyPassword(normalizeRecoveryKey(key)+"X", hash) {
		t.Fatal("incorrect recovery key matched")
	}
}

func TestFirstRunSetupCreatesCredentialsAndClosesEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("security:\n  admin_username: admin\n  admin_password_hash: "+config.LegacyExamplePasswordHash+"\n  session_timeout: 60\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	vaultManager, err := vault.New(filepath.Join(dir, "vault"), 15*time.Minute, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{cfg: cfg, vault: vaultManager, sessions: make(map[string]time.Time), recoveryAttempts: make(map[string]authAttempt)}
	router := gin.New()
	router.POST("/setup", s.handleFirstRunSetup)

	body := `{"username":"owner","password":"VeryStrong!Pass2026","vault_password":"Separate!Vault2026","vault_unlock_minutes":30}`
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:43210"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("setup returned %d: %s", response.Code, response.Body.String())
	}
	var result struct {
		RecoveryKey      string `json:"recovery_key"`
		VaultRecoveryKey string `json:"vault_recovery_code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.RecoveryKey == "" || result.VaultRecoveryKey == "" {
		t.Fatalf("setup did not return both recovery credentials: %s", response.Body.String())
	}
	if !cfg.Security.IsSetupComplete() || cfg.Security.AdminUsername != "owner" {
		t.Fatal("setup credentials were not activated")
	}
	if !verifyPassword("VeryStrong!Pass2026", cfg.Security.AdminPasswordHash) || !verifyPassword(normalizeRecoveryKey(result.RecoveryKey), cfg.Security.RecoveryKeyHash) {
		t.Fatal("stored credential hashes do not match generated credentials")
	}
	if len(response.Result().Cookies()) == 0 {
		t.Fatal("setup did not create an authenticated session")
	}
	if status := vaultManager.Status(); !status.Configured || !status.Unlocked {
		t.Fatalf("vault was not configured and unlocked: %+v", status)
	}
	if cfg.Vault.UnlockMinutes != 30 {
		t.Fatalf("vault auto-lock policy was not saved: %d", cfg.Vault.UnlockMinutes)
	}

	second := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(body))
	secondRequest.Header.Set("Content-Type", "application/json")
	secondRequest.RemoteAddr = "127.0.0.1:43210"
	router.ServeHTTP(second, secondRequest)
	if second.Code != http.StatusConflict {
		t.Fatalf("completed setup endpoint remained open: %d", second.Code)
	}
}

func TestFirstRunSetupRejectsRemoteRequests(t *testing.T) {
	s := &Server{cfg: &config.Config{Security: config.SecurityConfig{AdminPasswordHash: config.LegacyExamplePasswordHash}}, sessions: make(map[string]time.Time)}
	router := gin.New()
	router.POST("/setup", s.handleFirstRunSetup)
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(`{"username":"owner","password":"VeryStrong!Pass2026"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.10:40000"
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusForbidden {
		t.Fatalf("remote setup returned %d", response.Code)
	}
}

func TestStrongPasswordPolicy(t *testing.T) {
	if strongPasswordError("owner", "VeryStrong!Pass2026") != "" {
		t.Fatal("strong password was rejected")
	}
	for _, password := range []string{"short!A1", "alllowercasepassword", "Owner!Password2026"} {
		if strongPasswordError("owner", password) == "" {
			t.Fatalf("weak password %q was accepted", password)
		}
	}
}

func TestRecoveryAttemptLockout(t *testing.T) {
	s := &Server{recoveryAttempts: make(map[string]authAttempt)}
	for i := 0; i < 5; i++ {
		s.recordRecoveryFailure("127.0.0.1")
	}
	if _, locked := s.recoveryLocked("127.0.0.1"); !locked {
		t.Fatal("expected recovery lockout")
	}
	s.authMu.Lock()
	a := s.recoveryAttempts["127.0.0.1"]
	a.LockedUntil = time.Now().Add(-time.Second)
	s.recoveryAttempts["127.0.0.1"] = a
	s.authMu.Unlock()
	if _, locked := s.recoveryLocked("127.0.0.1"); locked {
		t.Fatal("expired recovery lockout remained active")
	}
}

func TestLoginAttemptLockoutAndSuccessfulReset(t *testing.T) {
	s := &Server{loginAttempts: make(map[string]authAttempt)}
	key := "127.0.0.1|owner"
	for i := 0; i < 4; i++ {
		s.recordLoginFailure(key)
	}
	if _, locked := s.loginLocked(key); locked {
		t.Fatal("login locked before the fifth failure")
	}
	s.recordLoginFailure(key)
	if wait, locked := s.loginLocked(key); !locked || wait < 1 || wait > 5 {
		t.Fatalf("expected a temporary login lock, got locked=%v wait=%d", locked, wait)
	}
	s.clearLoginFailures(key)
	if _, locked := s.loginLocked(key); locked {
		t.Fatal("successful login did not clear the lock state")
	}
}

func TestLoginRateLimitResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{}
	router := gin.New()
	router.GET("/locked", func(c *gin.Context) { s.loginRateLimited(c, "application/json", 3) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/locked", nil))
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit returned %d", response.Code)
	}
	if response.Header().Get("Retry-After") != "180" || !strings.Contains(response.Body.String(), "login_locked") {
		t.Fatalf("rate limit response is incomplete: headers=%v body=%s", response.Header(), response.Body.String())
	}
}

func TestVaultProtectionScope(t *testing.T) {
	for _, path := range []string{"/api/settings/password", "/api/filtering", "/api/security-packs/install"} {
		if !vaultProtectsPath(path) {
			t.Errorf("expected vault to protect %s", path)
		}
	}
	for _, path := range []string{"/api/organizer/todos/4", "/api/organizer/notes/2", "/api/discipline/tasks/8", "/api/focus/start"} {
		if vaultProtectsPath(path) {
			t.Errorf("vault should not block productivity action %s", path)
		}
	}
}

func TestAuthenticationVaultAndRecoveryEndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("server:\n  api_host: 127.0.0.1\nsecurity:\n  admin_username: admin\n  admin_password_hash: "+config.LegacyExamplePasswordHash+"\n  session_timeout: 60\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	vaultManager, err := vault.New(filepath.Join(dir, "vault"), time.Minute, 5, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{
		cfg:              cfg,
		vault:            vaultManager,
		sessions:         make(map[string]time.Time),
		csrfTokens:       make(map[string]string),
		loginAttempts:    make(map[string]authAttempt),
		recoveryAttempts: make(map[string]authAttempt),
	}
	router := gin.New()
	router.Use(s.securityMiddleware(), requestSizeMiddleware())
	router.POST("/setup", s.handleFirstRunSetup)
	router.POST("/login", s.handleLogin)
	router.POST("/recover-account", s.handleAccountRecovery)
	router.GET("/logout", s.handleLogout)
	api := router.Group("/api", s.authMiddleware(), s.vaultMiddleware())
	api.POST("/vault/lock", s.vaultLock)
	api.POST("/vault/unlock", s.vaultUnlock)
	api.POST("/settings/test", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"success": true}) })

	request := func(method, path, body string, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Host = "127.0.0.1:8080"
		req.RemoteAddr = "127.0.0.1:43210"
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	setupBody := `{"username":"owner","password":"VeryStrong!Pass2026","vault_password":"Separate!Vault2026","vault_unlock_minutes":15}`
	setup := request(http.MethodPost, "/setup", setupBody, nil, "")
	if setup.Code != http.StatusOK {
		t.Fatalf("setup failed: %d %s", setup.Code, setup.Body.String())
	}
	var setupResult struct {
		RecoveryKey string `json:"recovery_key"`
		CSRFToken   string `json:"csrf_token"`
	}
	if err := json.Unmarshal(setup.Body.Bytes(), &setupResult); err != nil || setupResult.RecoveryKey == "" || setupResult.CSRFToken == "" {
		t.Fatalf("setup response incomplete: %s", setup.Body.String())
	}
	cookies := setup.Result().Cookies()

	locked := request(http.MethodPost, "/api/vault/lock", `{}`, cookies, setupResult.CSRFToken)
	if locked.Code != http.StatusOK {
		t.Fatalf("vault lock failed: %d %s", locked.Code, locked.Body.String())
	}
	protected := request(http.MethodPost, "/api/settings/test", `{}`, cookies, setupResult.CSRFToken)
	if protected.Code != http.StatusLocked {
		t.Fatalf("locked vault allowed protected mutation: %d %s", protected.Code, protected.Body.String())
	}
	wrongUnlock := request(http.MethodPost, "/api/vault/unlock", `{"password":"wrong password"}`, cookies, setupResult.CSRFToken)
	if wrongUnlock.Code != http.StatusUnauthorized {
		t.Fatalf("wrong vault password returned %d", wrongUnlock.Code)
	}
	unlocked := request(http.MethodPost, "/api/vault/unlock", `{"password":"Separate!Vault2026"}`, cookies, setupResult.CSRFToken)
	if unlocked.Code != http.StatusOK {
		t.Fatalf("vault unlock failed: %d %s", unlocked.Code, unlocked.Body.String())
	}
	protected = request(http.MethodPost, "/api/settings/test", `{}`, cookies, setupResult.CSRFToken)
	if protected.Code != http.StatusOK {
		t.Fatalf("unlocked vault blocked protected mutation: %d %s", protected.Code, protected.Body.String())
	}

	recoveryBody := `{"username":"owner","recovery_key":"` + setupResult.RecoveryKey + `","new_password":"NewDashboard!Pass2027"}`
	recovered := request(http.MethodPost, "/recover-account", recoveryBody, nil, "")
	if recovered.Code != http.StatusOK {
		t.Fatalf("account recovery failed: %d %s", recovered.Code, recovered.Body.String())
	}
	var recoveryResult struct {
		NewRecoveryKey string `json:"new_recovery_key"`
	}
	if err := json.Unmarshal(recovered.Body.Bytes(), &recoveryResult); err != nil || recoveryResult.NewRecoveryKey == "" || recoveryResult.NewRecoveryKey == setupResult.RecoveryKey {
		t.Fatalf("account recovery key was not rotated: %s", recovered.Body.String())
	}
	staleSession := request(http.MethodPost, "/api/settings/test", `{}`, cookies, setupResult.CSRFToken)
	if staleSession.Code != http.StatusUnauthorized {
		t.Fatalf("recovery did not invalidate old session: %d", staleSession.Code)
	}
	oldLogin := request(http.MethodPost, "/login", `{"username":"owner","password":"VeryStrong!Pass2026"}`, nil, "")
	if oldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("old password still worked after recovery: %d", oldLogin.Code)
	}
	newLogin := request(http.MethodPost, "/login", `{"username":"owner","password":"NewDashboard!Pass2027"}`, nil, "")
	if newLogin.Code != http.StatusOK {
		t.Fatalf("new password did not work: %d %s", newLogin.Code, newLogin.Body.String())
	}
	var newLoginResult struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(newLogin.Body.Bytes(), &newLoginResult); err != nil || newLoginResult.CSRFToken == "" {
		t.Fatalf("new login did not return a CSRF token: %s", newLogin.Body.String())
	}
	newCookies := newLogin.Result().Cookies()
	logout := request(http.MethodGet, "/logout", "", newCookies, "")
	if logout.Code != http.StatusFound {
		t.Fatalf("logout returned %d", logout.Code)
	}
	afterLogout := request(http.MethodPost, "/api/settings/test", `{}`, newCookies, newLoginResult.CSRFToken)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("logout did not invalidate the session: %d", afterLogout.Code)
	}
	oldRecovery := request(http.MethodPost, "/recover-account", `{"username":"owner","recovery_key":"`+setupResult.RecoveryKey+`","new_password":"AnotherSecure!Pass2028"}`, nil, "")
	if oldRecovery.Code != http.StatusUnauthorized {
		t.Fatalf("old recovery key still worked after rotation: %d", oldRecovery.Code)
	}
}

func TestUnavailableVaultReturnsServiceUnavailable(t *testing.T) {
	s := &Server{}
	router := gin.New()
	router.POST("/unlock", s.vaultUnlock)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/unlock", strings.NewReader(`{"password":"anything"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "vault_unavailable") {
		t.Fatalf("unavailable vault returned %d: %s", w.Code, w.Body.String())
	}
}
