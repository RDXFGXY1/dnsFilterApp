package api

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/appnetwork"
	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/connectionfreeze"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
	"github.com/RDXFGXY1/dns-filter-app/internal/dns"
	"github.com/RDXFGXY1/dns-filter-app/internal/dnsbypass"
	"github.com/RDXFGXY1/dns-filter-app/internal/dnswatch"
	"github.com/RDXFGXY1/dns-filter-app/internal/downloadguard"
	"github.com/RDXFGXY1/dns-filter-app/internal/filter"
	"github.com/RDXFGXY1/dns-filter-app/internal/hardsystem"
	"github.com/RDXFGXY1/dns-filter-app/internal/netdns"
	"github.com/RDXFGXY1/dns-filter-app/internal/networkintel"
	"github.com/RDXFGXY1/dns-filter-app/internal/plugins"
	"github.com/RDXFGXY1/dns-filter-app/internal/vault"
	appversion "github.com/RDXFGXY1/dns-filter-app/internal/version"
	"github.com/RDXFGXY1/dns-filter-app/pkg/logger"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	cfg              *config.Config
	db               *database.DB
	filter           *filter.Engine
	dnsServer        *dns.Server
	router           *gin.Engine
	server           *http.Server
	log              *logger.Logger
	discipline       *discipline.Manager
	vault            *vault.Manager
	freeze           *connectionfreeze.Manager
	hardSystem       *hardsystem.Manager
	appNetwork       *appnetwork.Manager
	networkIntel     *networkintel.Manager
	dnsWatch         *dnswatch.Manager
	dnsBypass        *dnsbypass.Manager
	downloadGuard    *downloadguard.Manager
	plugins          *plugins.Manager
	disciplineDNS    chan dns.QueryEvent
	authMu           sync.Mutex
	setupMu          sync.Mutex
	configMu         sync.Mutex
	sessions         map[string]time.Time
	csrfTokens       map[string]string
	loginAttempts    map[string]authAttempt
	recoveryAttempts map[string]authAttempt
}

type authAttempt struct {
	Failures    int
	LockedUntil time.Time
}

func NewServer(cfg *config.Config, db *database.DB, filterEngine *filter.Engine, dnsServer *dns.Server) *Server {
	if cfg.Logging.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	s := &Server{
		cfg:              cfg,
		db:               db,
		filter:           filterEngine,
		dnsServer:        dnsServer,
		router:           router,
		log:              logger.Get(),
		sessions:         make(map[string]time.Time),
		csrfTokens:       make(map[string]string),
		loginAttempts:    make(map[string]authAttempt),
		recoveryAttempts: make(map[string]authAttempt),
	}
	router.Use(gin.Recovery(), s.securityMiddleware(), requestSizeMiddleware())

	if mgr, err := discipline.New(db.GetDB()); err != nil {
		s.log.Warnf("Discipline module unavailable: %v", err)
	} else {
		s.discipline = mgr
		if dnsServer != nil {
			s.disciplineDNS = dnsServer.Subscribe()
			go func(ch <-chan dns.QueryEvent) {
				for event := range ch {
					mgr.RecordFocusActivity("domain", event.Domain, 0)
				}
			}(s.disciplineDNS)
		}
	}
	if mgr, err := vault.New(cfg.Vault.Directory, time.Duration(cfg.Vault.UnlockMinutes)*time.Minute, cfg.Vault.MaxFailures, time.Duration(cfg.Vault.LockoutMinutes)*time.Minute); err != nil {
		s.log.Warnf("Secure configuration vault unavailable: %v", err)
	} else {
		s.vault = mgr
	}

	if mgr, err := connectionfreeze.New(db.GetDB()); err != nil {
		s.log.Warnf("Connection Freeze unavailable: %v", err)
	} else {
		s.freeze = mgr
	}
	if mgr, err := hardsystem.New(db.GetDB()); err != nil {
		s.log.Warnf("Hard System unavailable: %v", err)
	} else {
		s.hardSystem = mgr
	}
	if mgr, err := appnetwork.New(db.GetDB()); err != nil {
		s.log.Warnf("App Network Control unavailable: %v", err)
	} else {
		s.appNetwork = mgr
	}
	if mgr, err := networkintel.New(db.GetDB()); err != nil {
		s.log.Warnf("Connection Intelligence unavailable: %v", err)
	} else {
		s.networkIntel = mgr
	}
	if s.networkIntel != nil && s.appNetwork != nil {
		go s.runAutomaticResponses()
	}
	if !cfg.HasSystemDNSPreference() {
		if _, _, active, err := netdns.Status(); err == nil && active {
			cfg.Server.ManageSystemDNS = true
			if err := cfg.SaveDefault(); err != nil {
				s.log.Warnf("Could not migrate the existing local DNS routing preference: %v", err)
				cfg.Server.ManageSystemDNS = false
			} else {
				s.log.Infof("Migrated the existing local DNS routing preference to automatic route repair")
			}
		}
	}
	if mgr, err := dnswatch.NewWithAutoRepair(db.GetDB(), cfg.Server.ManageSystemDNS); err != nil {
		s.log.Warnf("DNS Route Watch unavailable: %v", err)
	} else {
		s.dnsWatch = mgr
	}
	if mgr, err := dnsbypass.New(db.GetDB()); err != nil {
		s.log.Warnf("DNS Bypass Kill Switch unavailable: %v", err)
	} else {
		s.dnsBypass = mgr
	}
	if mgr, err := downloadguard.New(db.GetDB(), "./data/quarantine/downloads"); err != nil {
		s.log.Warnf("Download Guardian unavailable: %v", err)
	} else {
		s.downloadGuard = mgr
	}
	if mgr, err := plugins.New(db.GetDB()); err != nil {
		s.log.Warnf("Plugin Center unavailable: %v", err)
	} else {
		s.plugins = mgr
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Static("/static", "./web/static")
	s.router.LoadHTMLGlob("./web/templates/*")

	// Public routes
	s.router.GET("/healthz", s.handleHealth)
	s.router.GET("/setup", s.handleFirstRunSetupPage)
	s.router.POST("/setup", s.handleFirstRunSetup)
	s.router.GET("/setup/preflight", s.handleFirstRunPreflight)
	s.router.GET("/login", s.handleLoginPage)
	s.router.POST("/login", s.handleLogin)
	s.router.POST("/recover-account", s.handleAccountRecovery)
	s.router.GET("/logout", s.handleLogout)
	s.router.OPTIONS("/extension/v1/analyze", s.hardSystemPreflight)
	s.router.POST("/extension/v1/analyze", s.hardSystemAnalyze)

	// Protected page routes
	s.router.GET("/", s.authCheck(), s.handleDashboard)
	s.router.GET("/discipline", s.authCheck(), s.handleDiscipline)
	s.router.GET("/discipline/", s.authCheck(), s.handleDiscipline)

	// Protected API routes
	api := s.router.Group("/api")
	api.Use(s.authMiddleware())
	api.Use(s.vaultMiddleware())
	{
		api.GET("/stats", s.getStats)
		api.GET("/stats/blocked", s.getBlockedStats)
		api.GET("/stats/top-blocked", s.getTopBlocked)
		api.GET("/stats/timeline", s.getTimeline)
		api.GET("/recent", s.getRecentBlocked)
		api.GET("/whitelist", s.getWhitelist)
		api.GET("/whitelist/discover", s.discoverWhitelistSubdomains)
		api.POST("/whitelist", s.addToWhitelist)
		api.DELETE("/whitelist/:domain", s.removeFromWhitelist)
		api.GET("/requests", s.getUnblockRequests)
		api.POST("/requests/:id/approve", s.approveUnblockRequest)
		api.POST("/requests/:id/deny", s.denyUnblockRequest)
		api.POST("/blocklist/update", s.updateBlocklists)
		api.GET("/blocklist/count", s.getBlocklistCount)
		api.GET("/settings", s.getSettings)
		api.POST("/settings", s.updateSettings)
		api.POST("/system/restart", s.restartService)
		api.POST("/system/clear-cache", s.clearCache)
		api.GET("/custom-blocklist", s.getCustomBlocklist)
		api.POST("/custom-blocklist/add", s.addToCustomBlocklist)
		api.DELETE("/custom-blocklist/:domain", s.removeFromCustomBlocklist)
		api.POST("/blocklist/reload-custom", s.reloadCustomBlocklists)

		// Newer features: categories, keywords, gamification, AI
		s.registerFeatureRoutes(api)
		s.registerConnectionFreezeRoutes(api)
		s.registerOrganizerRoutes(api)
		s.registerDisciplineRoutes(api)
		s.registerPluginRoutes(api)
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"service": "dns-filter", "version": appversion.Current, "ready": true})
}

func (s *Server) handleDiscipline(c *gin.Context) {
	c.HTML(http.StatusOK, "discipline.html", gin.H{"title": "Discipline", "csp_nonce": c.GetString(cspNonceContextKey)})
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.APIHost, s.cfg.Server.APIPort)
	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.plugins != nil {
		s.plugins.Shutdown()
	}
	if s.dnsServer != nil && s.disciplineDNS != nil {
		s.dnsServer.Unsubscribe(s.disciplineDNS)
		s.disciplineDNS = nil
	}
	if s.freeze != nil {
		_ = s.freeze.Close()
	}
	if s.downloadGuard != nil {
		s.downloadGuard.Close()
	}
	if s.dnsWatch != nil {
		s.dnsWatch.Close()
	}
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// authCheck redirects to /login for page routes
func (s *Server) authCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.cfg.Security.IsSetupComplete() {
			c.Redirect(http.StatusFound, "/setup")
			c.Abort()
			return
		}
		if !s.validSession(c) {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set("authenticated", true)
		c.Next()
	}
}

// authMiddleware returns 401 for API routes
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.cfg.Security.IsSetupComplete() {
			c.JSON(http.StatusPreconditionRequired, gin.H{"error": "First-run setup is required", "code": "setup_required"})
			c.Abort()
			return
		}
		if !s.validSession(c) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Set("authenticated", true)
		c.Next()
	}
}

func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *Server) handleDashboard(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	configured, unlocked := false, false
	if s.vault != nil {
		status := s.vault.Status()
		configured, unlocked = status.Configured, status.Unlocked
	}
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":            "DNS Filter Dashboard",
		"vault_configured": configured,
		"vault_unlocked":   unlocked,
		"csp_nonce":        c.GetString(cspNonceContextKey),
	})
}

func (s *Server) handleLoginPage(c *gin.Context) {
	if !s.cfg.Security.IsSetupComplete() {
		c.Redirect(http.StatusFound, "/setup")
		return
	}
	if s.validSession(c) {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":     "Login",
		"error":     "",
		"csp_nonce": c.GetString(cspNonceContextKey),
	})
}

func (s *Server) handleLogin(c *gin.Context) {
	if !s.cfg.Security.IsSetupComplete() {
		c.JSON(http.StatusPreconditionRequired, gin.H{"error": "Complete first-run security setup before signing in", "code": "setup_required", "redirect": "/setup"})
		return
	}
	// Support both JSON and HTML form
	var username, password string
	contentType := c.GetHeader("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var d struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&d); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		username = d.Username
		password = d.Password
	} else {
		username = c.PostForm("username")
		password = c.PostForm("password")
	}

	if username == "" || password == "" {
		s.loginFailed(c, contentType, "Username and password required")
		return
	}
	loginKey := c.ClientIP() + "|" + strings.ToLower(username)
	if wait, locked := s.loginLocked(loginKey); locked {
		s.loginRateLimited(c, contentType, wait)
		return
	}

	validUser := username == s.cfg.Security.AdminUsername
	validPass := verifyPassword(password, s.cfg.Security.AdminPasswordHash)

	if validUser && validPass {
		s.clearLoginFailures(loginKey)
		response := gin.H{"success": true, "redirect": "/"}
		s.configMu.Lock()
		if s.cfg.Security.RecoveryKeyHash == "" {
			key, keyHash, err := newRecoveryKey()
			if err != nil {
				s.configMu.Unlock()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create a recovery key"})
				return
			}
			s.cfg.Security.RecoveryKeyHash = keyHash
			if err := s.cfg.SaveDefault(); err != nil {
				s.cfg.Security.RecoveryKeyHash = ""
				s.configMu.Unlock()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save the recovery key"})
				return
			}
			response["recovery_key"] = key
			response["recovery_warning"] = "Save this recovery key now. It will not be shown again."
		}
		s.configMu.Unlock()
		token, err := s.createSession()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create a secure session"})
			return
		}
		s.setSessionCookie(c, token)
		response["csrf_token"] = s.csrfToken(token)
		if strings.HasPrefix(contentType, "application/json") {
			c.JSON(http.StatusOK, response)
		} else {
			c.Redirect(http.StatusFound, "/")
		}
	} else {
		s.recordLoginFailure(loginKey)
		time.Sleep(250 * time.Millisecond)
		s.loginFailed(c, contentType, "Invalid username or password")
	}
}

func (s *Server) loginRateLimited(c *gin.Context, contentType string, wait int) {
	c.Header("Retry-After", fmt.Sprint(wait*60))
	message := fmt.Sprintf("Too many sign-in attempts. Try again in %d minutes.", wait)
	if strings.HasPrefix(contentType, "application/json") {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": message, "code": "login_locked", "retry_after_minutes": wait})
		return
	}
	c.HTML(http.StatusTooManyRequests, "login.html", gin.H{"title": "Login", "error": message, "csp_nonce": c.GetString(cspNonceContextKey)})
}

func (s *Server) loginFailed(c *gin.Context, contentType, msg string) {
	if strings.HasPrefix(contentType, "application/json") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
	} else {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"title":     "Login",
			"error":     msg,
			"csp_nonce": c.GetString(cspNonceContextKey),
		})
	}
}

func (s *Server) handleLogout(c *gin.Context) {
	if token, err := c.Cookie("session"); err == nil {
		s.authMu.Lock()
		delete(s.sessions, token)
		delete(s.csrfTokens, token)
		s.authMu.Unlock()
	}
	secure := s.cfg.Security.HTTPSEnabled
	c.SetCookie("session", "", -1, "/", "", secure, true)
	c.SetCookie("csrf_token", "", -1, "/", "", secure, false)
	c.Redirect(http.StatusFound, "/login")
}

func (s *Server) handleAccountRecovery(c *gin.Context) {
	if !s.cfg.Security.IsSetupComplete() {
		c.JSON(http.StatusPreconditionRequired, gin.H{"error": "First-run setup is required", "code": "setup_required"})
		return
	}
	var data struct {
		Username    string `json:"username"`
		RecoveryKey string `json:"recovery_key"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&data); err != nil || data.Username == "" || data.RecoveryKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username, recovery key, and new password are required"})
		return
	}
	if len(data.NewPassword) < 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must contain at least 12 characters"})
		return
	}
	if message := strongPasswordError(data.Username, data.NewPassword); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	ip := c.ClientIP()
	if wait, locked := s.recoveryLocked(ip); locked {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": fmt.Sprintf("Too many attempts. Try again in %d minutes.", wait)})
		return
	}
	s.configMu.Lock()
	valid := data.Username == s.cfg.Security.AdminUsername && s.cfg.Security.RecoveryKeyHash != "" &&
		verifyPassword(normalizeRecoveryKey(data.RecoveryKey), s.cfg.Security.RecoveryKeyHash)
	if !valid {
		s.configMu.Unlock()
		s.recordRecoveryFailure(ip)
		time.Sleep(250 * time.Millisecond)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "The recovery information is invalid"})
		return
	}
	passwordHash, err := config.HashPassword(data.NewPassword)
	if err != nil {
		s.configMu.Unlock()
		c.JSON(500, gin.H{"error": "Could not secure the new password"})
		return
	}
	nextKey, nextHash, err := newRecoveryKey()
	if err != nil {
		s.configMu.Unlock()
		c.JSON(500, gin.H{"error": "Could not rotate the recovery key"})
		return
	}
	oldPassword, oldRecovery := s.cfg.Security.AdminPasswordHash, s.cfg.Security.RecoveryKeyHash
	s.cfg.Security.AdminPasswordHash, s.cfg.Security.RecoveryKeyHash = passwordHash, nextHash
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Security.AdminPasswordHash, s.cfg.Security.RecoveryKeyHash = oldPassword, oldRecovery
		s.configMu.Unlock()
		c.JSON(500, gin.H{"error": "Could not save the new credentials"})
		return
	}
	s.configMu.Unlock()
	s.authMu.Lock()
	s.sessions = make(map[string]time.Time)
	s.csrfTokens = make(map[string]string)
	s.loginAttempts = make(map[string]authAttempt)
	delete(s.recoveryAttempts, ip)
	s.authMu.Unlock()
	c.JSON(http.StatusOK, gin.H{"success": true, "new_recovery_key": nextKey, "warning": "Your old recovery key is no longer valid. Save this new key now."})
}

func (s *Server) createSession() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	csrfRaw := make([]byte, 32)
	if _, err := rand.Read(csrfRaw); err != nil {
		return "", err
	}
	csrfToken := base64.RawURLEncoding.EncodeToString(csrfRaw)
	s.authMu.Lock()
	if s.sessions == nil {
		s.sessions = make(map[string]time.Time)
	}
	if s.csrfTokens == nil {
		s.csrfTokens = make(map[string]string)
	}
	s.sessions[token] = time.Now().Add(time.Duration(s.cfg.Security.SessionTimeout) * time.Minute)
	s.csrfTokens[token] = csrfToken
	s.authMu.Unlock()
	return token, nil
}

func (s *Server) validSession(c *gin.Context) bool {
	token, err := c.Cookie("session")
	if err != nil || token == "" {
		return false
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	expires, ok := s.sessions[token]
	if !ok || time.Now().After(expires) {
		delete(s.sessions, token)
		delete(s.csrfTokens, token)
		return false
	}
	return true
}

func (s *Server) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	secure := s.cfg.Security.HTTPSEnabled
	c.SetCookie("session", token, s.cfg.Security.SessionTimeout*60, "/", "", secure, true)
	c.SetCookie("csrf_token", s.csrfToken(token), s.cfg.Security.SessionTimeout*60, "/", "", secure, false)
}

func (s *Server) csrfToken(session string) string {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	return s.csrfTokens[session]
}

func newRecoveryKey() (string, string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	plain := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	groups := make([]string, 0, 7)
	for len(plain) > 4 {
		groups = append(groups, plain[:4])
		plain = plain[4:]
	}
	if plain != "" {
		groups = append(groups, plain)
	}
	key := strings.Join(groups, "-")
	hash, err := config.HashPassword(normalizeRecoveryKey(key))
	return key, hash, err
}

func normalizeRecoveryKey(key string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(key), "-", ""))
}

func (s *Server) recoveryLocked(ip string) (int, bool) {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	a := s.recoveryAttempts[ip]
	if time.Now().Before(a.LockedUntil) {
		return max(1, int(math.Ceil(time.Until(a.LockedUntil).Minutes()))), true
	}
	if !a.LockedUntil.IsZero() {
		delete(s.recoveryAttempts, ip)
	}
	return 0, false
}

func (s *Server) loginLocked(key string) (int, bool) {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	a := s.loginAttempts[key]
	if time.Now().Before(a.LockedUntil) {
		return max(1, int(math.Ceil(time.Until(a.LockedUntil).Minutes()))), true
	}
	if !a.LockedUntil.IsZero() {
		delete(s.loginAttempts, key)
	}
	return 0, false
}

func (s *Server) recordLoginFailure(key string) {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	if s.loginAttempts == nil {
		s.loginAttempts = make(map[string]authAttempt)
	}
	a := s.loginAttempts[key]
	a.Failures++
	if a.Failures >= 5 {
		a.Failures = 0
		a.LockedUntil = time.Now().Add(5 * time.Minute)
	}
	s.loginAttempts[key] = a
}

func (s *Server) clearLoginFailures(key string) {
	s.authMu.Lock()
	delete(s.loginAttempts, key)
	s.authMu.Unlock()
}

func (s *Server) recordRecoveryFailure(ip string) {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	a := s.recoveryAttempts[ip]
	a.Failures++
	if a.Failures >= 5 {
		a.Failures = 0
		a.LockedUntil = time.Now().Add(15 * time.Minute)
	}
	s.recoveryAttempts[ip] = a
}

func (s *Server) getStats(c *gin.Context) {
	blockedCount := s.filter.GetBlockedCount()
	dbStats, _ := s.db.GetBlockedStats(24)
	c.JSON(http.StatusOK, gin.H{
		"blocked_domains": blockedCount,
		"stats":           dbStats,
		"timestamp":       time.Now().Unix(),
	})
}

func (s *Server) getBlockedStats(c *gin.Context) {
	stats, err := s.db.GetBlockedStats(24)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (s *Server) getTopBlocked(c *gin.Context) {
	topBlocked, err := s.db.GetTopBlockedDomains(10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Convert the map to a sorted [{domain, count}] array the dashboard expects.
	out := make([]gin.H, 0, len(topBlocked))
	for domain, count := range topBlocked {
		out = append(out, gin.H{"domain": domain, "count": count})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i]["count"].(int) > out[j]["count"].(int)
	})
	c.JSON(http.StatusOK, out)
}

func (s *Server) getTimeline(c *gin.Context) {
	points, err := s.db.GetBlockedTimeline(24)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, points)
}

func (s *Server) getRecentBlocked(c *gin.Context) {
	recent, err := s.db.GetRecentBlocked(50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Map to the lowercase keys + unix timestamp the dashboard expects.
	out := make([]gin.H, 0, len(recent))
	for _, r := range recent {
		out = append(out, gin.H{
			"domain":    r.Domain,
			"client_ip": r.ClientIP,
			"timestamp": r.Timestamp.Unix(),
		})
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) getWhitelist(c *gin.Context) {
	c.JSON(http.StatusOK, s.filter.GetWhitelist())
}

func (s *Server) addToWhitelist(c *gin.Context) {
	var data struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	wildcard := strings.HasPrefix(strings.TrimSpace(data.Domain), "*.")
	domain, err := normalizeRootDomain(data.Domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if wildcard {
		domain = "*." + domain
	}

	fmt.Print("Adding to whitelist: ", domain)
	if err := s.filter.AddToWhitelist(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Mutate policy first, then invalidate cached answers. Clearing before the
	// mutation leaves a race where the old blocked answer can be cached again.
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "domain": domain})
}

func (s *Server) discoverWhitelistSubdomains(c *gin.Context) {
	root, err := normalizeRootDomain(c.Query("domain"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items, err := s.db.GetObservedSubdomains(root, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"root":     root,
		"wildcard": "*." + root,
		"domains":  items,
		"source":   "local blocked DNS history",
	})
}

func normalizeRootDomain(value string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(value))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	if slash := strings.IndexByte(domain, '/'); slash >= 0 {
		domain = domain[:slash]
	}
	domain = strings.TrimPrefix(strings.Trim(domain, "."), "*.")
	if domain == "" || len(domain) > 253 || !strings.Contains(domain, ".") {
		return "", fmt.Errorf("enter a valid root domain, for example spotify.com")
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", fmt.Errorf("invalid domain")
		}
		for _, ch := range label {
			if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
				return "", fmt.Errorf("invalid domain")
			}
		}
	}
	return domain, nil
}

func (s *Server) removeFromWhitelist(c *gin.Context) {
	if err := s.filter.RemoveFromWhitelist(c.Param("domain")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	fmt.Print("Removing from whitelist: ", c.Param("domain"))
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) updateBlocklists(c *gin.Context) {
	go func() {
		s.filter.UpdateBlocklists()
		if s.dnsServer != nil {
			s.dnsServer.ClearCache()
		}
	}()
	c.JSON(http.StatusOK, gin.H{"message": "Blocklist update started, cache will be cleared after update"})
}

func (s *Server) getBlocklistCount(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"count": s.filter.GetBlockedCount()})
}

func (s *Server) getSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"dns_port":  s.cfg.Server.DNSPort,
		"api_port":  s.cfg.Server.APIPort,
		"filtering": s.cfg.Filtering.Enabled,
	})
}

func (s *Server) updateSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) restartService(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Restart initiated"})
}

func (s *Server) clearCache(c *gin.Context) {
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"message": "DNS cache cleared"})
}

func (s *Server) getCustomBlocklist(c *gin.Context) {
	c.JSON(http.StatusOK, s.filter.GetCustomBlocklist())
}

func (s *Server) addToCustomBlocklist(c *gin.Context) {
	var data struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.filter.AddToCustomBlocklist(data.Domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "domain": data.Domain})
}

func (s *Server) removeFromCustomBlocklist(c *gin.Context) {
	if err := s.filter.RemoveFromCustomBlocklist(c.Param("domain")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) reloadCustomBlocklists(c *gin.Context) {
	count, err := s.filter.ReloadCustomBlocklists()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Custom blocklists reloaded and cache cleared",
		"count":   count,
	})
}
