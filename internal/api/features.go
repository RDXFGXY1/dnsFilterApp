package api

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/RDXFGXY1/dns-filter-app/internal/autostart"
	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/netdns"
	appversion "github.com/RDXFGXY1/dns-filter-app/internal/version"
)

// registerFeatureRoutes wires the newer engine features (categories, keywords,
// gamification, AI blocking) into the protected API group.
func (s *Server) registerFeatureRoutes(api *gin.RouterGroup) {
	s.registerVaultRoutes(api)
	// Explainable decisions, safe rule previews, conflicts, and system health.
	api.GET("/decision/inspect", s.inspectDomainDecision)
	api.POST("/decision/simulate", s.simulateRule)
	api.GET("/decision/conflicts", s.listRuleConflicts)
	api.GET("/protection/health", s.protectionHealth)
	api.POST("/protection/repair", s.repairProtection)
	api.GET("/protection/lockdown", s.getEmergencyLockdown)
	api.POST("/protection/lockdown/start", s.startEmergencyLockdown)
	api.POST("/protection/lockdown/stop", s.stopEmergencyLockdown)

	// Categories
	api.GET("/categories", s.getCategories)
	api.POST("/categories/:id/toggle", s.toggleCategory)

	// Keyword lists
	api.GET("/keywords", s.getKeywordLists)
	api.POST("/keywords/:id/toggle", s.toggleKeywordList)
	api.GET("/keywords/matches", s.getKeywordMatches)

	// Gamification
	api.GET("/gamification/profile", s.getUserProfile)
	api.GET("/gamification/leaderboard", s.getLeaderboard)

	// AI blocker
	api.GET("/ai/stats", s.getAIStats)
	api.GET("/ai/predictions", s.getAIPredictions)
	api.POST("/ai/analyze", s.analyzeDomain)
	api.GET("/hard-system/recent", s.hardSystemRecent)

	// Middleman mode (system DNS control)
	api.GET("/system/dns/status", s.dnsStatus)
	api.POST("/system/dns/setup", s.dnsSetup)
	api.POST("/system/dns/reset", s.dnsReset)

	// Settings / controls (CLI parity + more)
	api.GET("/test", s.testDomain)
	api.GET("/filtering", s.getFiltering)
	api.POST("/filtering", s.setFiltering)
	api.POST("/filtering/pause", s.pauseFiltering)
	api.GET("/system/info", s.systemInfo)

	// Detection modules + block action
	api.GET("/filtering/modules", s.getModules)
	api.POST("/filtering/module", s.setModule)
	api.POST("/filtering/block-action", s.setBlockAction)

	// App Blocker
	api.GET("/apps", s.getApps)
	api.POST("/apps/:id/toggle", s.toggleApp)
	api.GET("/app-network/apps", s.appNetworkApps)
	api.GET("/app-network/events", s.appNetworkEvents)
	api.POST("/app-network/block", s.appNetworkBlock)
	api.POST("/app-network/allow", s.appNetworkAllow)
	api.GET("/app-network/quarantine", s.appNetworkQuarantineList)
	api.POST("/app-network/quarantine", s.appNetworkQuarantine)
	api.POST("/app-network/quarantine/release", s.appNetworkQuarantineRelease)
	api.GET("/app-network/schedules", s.appNetworkSchedules)
	api.POST("/app-network/schedules", s.appNetworkScheduleAdd)
	api.DELETE("/app-network/schedules/:id", s.appNetworkScheduleDelete)
	api.GET("/network-intelligence/timeline", s.networkTimeline)
	api.GET("/network-intelligence/alerts", s.networkAlerts)
	api.POST("/network-intelligence/alerts/review", s.networkAlertReview)
	api.GET("/network-intelligence/privacy", s.networkPrivacy)
	api.GET("/network-intelligence/privacy/details", s.networkPrivacyDetails)
	api.POST("/network-intelligence/privacy/baseline", s.networkPrivacyBaseline)
	api.GET("/network-intelligence/destination-alerts", s.networkDestinationAlerts)
	api.POST("/network-intelligence/destination-alerts/trust", s.networkDestinationTrust)
	api.GET("/network-intelligence/history", s.networkHistoryStatus)
	api.POST("/network-intelligence/history", s.networkHistoryRetention)
	api.DELETE("/network-intelligence/history", s.networkHistoryClear)
	api.GET("/network-intelligence/export", s.networkConnectionExport)
	api.GET("/network-intelligence/notifications", s.networkNotificationSettings)
	api.POST("/network-intelligence/notifications", s.networkNotificationSettingsUpdate)
	api.GET("/network-intelligence/automatic-response", s.networkAutomaticResponse)
	api.POST("/network-intelligence/automatic-response", s.networkAutomaticResponseUpdate)
	api.GET("/network-intelligence/analytics", s.networkSecurityAnalytics)
	api.GET("/network-intelligence/learning", s.networkLearning)
	api.POST("/network-intelligence/learning/start", s.networkLearningStart)
	api.GET("/network-intelligence/learning/:id/baseline", s.networkLearningBaseline)
	api.POST("/network-intelligence/learning/:id/stop", s.networkLearningStop)
	api.POST("/network-test/start", s.networkTestStart)
	api.GET("/network-test/:id", s.networkTestStatus)
	api.DELETE("/network-test/:id", s.networkTestCancel)
	api.GET("/network-profiles", s.listNetworkProfiles)
	api.POST("/network-profiles", s.createNetworkProfile)
	api.GET("/network-profiles/:id/preview", s.previewNetworkProfile)
	api.POST("/network-profiles/:id/apply", s.applyNetworkProfile)
	api.DELETE("/network-profiles/:id", s.deleteNetworkProfile)
	api.GET("/dns-route-watch", s.dnsRouteWatchStatus)
	api.POST("/dns-route-watch/check", s.dnsRouteWatchCheck)
	api.GET("/dns-bypass", s.dnsBypassStatus)
	api.POST("/dns-bypass", s.dnsBypassUpdate)
	api.GET("/downloads", s.downloadEvents)
	api.GET("/downloads/analytics", s.downloadAnalytics)
	api.POST("/downloads/:id/quarantine", s.downloadQuarantine)
	api.POST("/downloads/:id/restore", s.downloadRestore)

	// Focus Mode
	api.GET("/focus", s.getFocus)
	api.POST("/focus/start", s.startFocus)
	api.POST("/focus/stop", s.stopFocus)

	// Doom-scroll Guard
	api.GET("/guard", s.getGuard)
	api.POST("/guard", s.setGuard)

	// Screen-Time Bank
	api.GET("/bank", s.getBank)
	api.POST("/bank/spend", s.spendBank)

	// Block page appearance
	api.GET("/blockpage", s.getBlockPage)
	api.POST("/blockpage", s.setBlockPage)
	api.GET("/blockpage/ca", s.downloadCA)

	// Start on login
	api.GET("/autostart", s.getAutostart)
	api.POST("/autostart", s.setAutostart)

	// Admin password
	api.POST("/settings/password", s.changePassword)

	// Live activity monitor (Server-Sent Events)
	api.GET("/stream", s.streamEvents)

	// Blocklist sources
	api.GET("/blocklist/sources", s.getSources)
	api.POST("/blocklist/sources", s.addSource)
	api.POST("/blocklist/sources/toggle", s.toggleSource)
	api.DELETE("/blocklist/sources/:index", s.removeSource)
	api.POST("/blocklist/auto-update", s.setAutoUpdate)
	api.GET("/security-packs", s.listSecurityPacks)
	api.POST("/security-packs/verify", s.verifySecurityPack)
	api.POST("/security-packs/install", s.installSecurityPack)
	api.POST("/security-packs/trusted-keys", s.trustSecurityPackKey)
	api.DELETE("/security-packs/:id", s.removeSecurityPack)
}

// ─── Live activity monitor (SSE) ──────────────────────────────────────────────

func (s *Server) streamEvents(c *gin.Context) {
	if s.dnsServer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "DNS server unavailable"})
		return
	}
	ch := s.dnsServer.Subscribe()
	defer s.dnsServer.Unsubscribe(ch)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	ctx := c.Request.Context()
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case ev, ok := <-ch:
			if !ok {
				return false
			}
			c.SSEvent("query", ev)
			return true
		case <-keepalive.C:
			c.SSEvent("ping", "")
			return true
		case <-ctx.Done():
			return false
		}
	})
}

// ─── App Blocker ──────────────────────────────────────────────────────────────

func (s *Server) getApps(c *gin.Context) {
	c.JSON(http.StatusOK, s.filter.GetApps())
}

func (s *Server) toggleApp(c *gin.Context) {
	var data struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.filter.SetAppBlocked(c.Param("id"), data.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": c.Param("id"), "blocked": data.Enabled})
}

// ─── Focus Mode ───────────────────────────────────────────────────────────────

func (s *Server) getFocus(c *gin.Context) {
	active, until := s.filter.FocusStatus()
	c.JSON(http.StatusOK, gin.H{"active": active, "until": until})
}

func (s *Server) startFocus(c *gin.Context) {
	var data struct {
		Minutes int      `json:"minutes"`
		Apps    []string `json:"apps"`
	}
	if err := c.ShouldBindJSON(&data); err != nil || data.Minutes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "positive 'minutes' required"})
		return
	}
	s.filter.StartFocus(data.Minutes, data.Apps)
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	_, until := s.filter.FocusStatus()
	c.JSON(http.StatusOK, gin.H{"success": true, "until": until})
}

func (s *Server) stopFocus(c *gin.Context) {
	s.filter.StopFocus()
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Doom-scroll Guard ────────────────────────────────────────────────────────

func (s *Server) getGuard(c *gin.Context) {
	c.JSON(http.StatusOK, s.filter.GuardStatus())
}

func (s *Server) setGuard(c *gin.Context) {
	var data struct {
		Enabled   bool `json:"enabled"`
		Threshold int  `json:"threshold"`
		Minutes   int  `json:"minutes"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	s.filter.SetGuard(data.Enabled, data.Threshold, data.Minutes)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Screen-Time Bank ─────────────────────────────────────────────────────────

func (s *Server) getBank(c *gin.Context) {
	c.JSON(http.StatusOK, s.filter.BankStatus())
}

func (s *Server) spendBank(c *gin.Context) {
	var data struct {
		App     string `json:"app"`
		Minutes int    `json:"minutes"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.filter.SpendUnlock(data.App, data.Minutes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Block page appearance ────────────────────────────────────────────────────

func (s *Server) getBlockPage(c *gin.Context) {
	mode, _ := s.db.GetSetting("blockpage_mode")
	msg, _ := s.db.GetSetting("blockpage_message")
	if mode == "" {
		mode = "motivational"
	}
	c.JSON(http.StatusOK, gin.H{"mode": mode, "message": msg})
}

func (s *Server) downloadCA(c *gin.Context) {
	path := s.filter.GetCACertPath()
	if path == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "HTTPS block page not enabled"})
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CA certificate not found"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=dns-filter-ca.crt")
	c.Data(http.StatusOK, "application/x-x509-ca-cert", data)
}

func (s *Server) setBlockPage(c *gin.Context) {
	var data struct {
		Mode    string `json:"mode"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	switch data.Mode {
	case "motivational", "roast", "meme":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be motivational, roast, or meme"})
		return
	}
	s.db.SetSetting("blockpage_mode", data.Mode)
	s.db.SetSetting("blockpage_message", data.Message)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Detection modules + block action ─────────────────────────────────────────

func (s *Server) getModules(c *gin.Context) {
	resp := gin.H{"modules": s.filter.GetModules()}
	if s.dnsServer != nil {
		resp["block_action"] = s.dnsServer.BlockAction()
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) setModule(c *gin.Context) {
	var data struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	previous, known := s.filter.GetModules()[data.Name]
	if !known || !s.filter.SetModule(data.Name, data.Enabled) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown module: " + data.Name})
		return
	}
	if err := s.cfg.SaveDefault(); err != nil {
		s.filter.SetModule(data.Name, previous)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save module setting: " + err.Error(), "code": "config_save_failed"})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "name": data.Name, "enabled": data.Enabled})
}

func (s *Server) setBlockAction(c *gin.Context) {
	var data struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	switch data.Action {
	case "nxdomain", "redirect", "block_page":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be nxdomain, redirect, or block_page"})
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	previous := s.cfg.Filtering.BlockAction
	s.cfg.Filtering.BlockAction = data.Action
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Filtering.BlockAction = previous
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save block action: " + err.Error(), "code": "config_save_failed"})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.SetBlockAction(data.Action)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "action": data.Action})
}

// ─── Start on login ───────────────────────────────────────────────────────────

func (s *Server) getAutostart(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"supported": autostart.Supported(), "enabled": autostart.IsEnabled()})
}

func (s *Server) setAutostart(c *gin.Context) {
	var data struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	var err error
	if data.Enabled {
		err = autostart.Enable()
	} else {
		err = autostart.Disable()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"hint":  "The server must run as Administrator to manage the login task.",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "enabled": data.Enabled})
}

// ─── Change admin password ────────────────────────────────────────────────────

func (s *Server) changePassword(c *gin.Context) {
	var data struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if !verifyPassword(data.Current, s.cfg.Security.AdminPasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}
	if len(data.New) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be at least 6 characters"})
		return
	}
	hash, err := config.HashPassword(data.New)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	previous := s.cfg.Security.AdminPasswordHash
	s.cfg.Security.AdminPasswordHash = hash
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Security.AdminPasswordHash = previous
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Blocklist sources manager ────────────────────────────────────────────────

func (s *Server) getSources(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"sources":              s.cfg.Blocklists.Sources,
		"auto_update_interval": s.cfg.Blocklists.AutoUpdateInterval,
	})
}

func (s *Server) addSource(c *gin.Context) {
	var src config.BlocklistSource
	if err := c.ShouldBindJSON(&src); err != nil || src.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and url required"})
		return
	}
	if src.Name == "" {
		src.Name = src.URL
	}
	if src.Category == "" {
		src.Category = "custom"
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	previous := append([]config.BlocklistSource(nil), s.cfg.Blocklists.Sources...)
	src.Enabled = true
	s.cfg.Blocklists.Sources = append(s.cfg.Blocklists.Sources, src)
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Blocklists.Sources = previous
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save blocklist source: " + err.Error(), "code": "config_save_failed"})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) toggleSource(c *gin.Context) {
	var data struct {
		Index   int  `json:"index"`
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	if data.Index < 0 || data.Index >= len(s.cfg.Blocklists.Sources) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "index out of range"})
		return
	}
	previous := s.cfg.Blocklists.Sources[data.Index].Enabled
	s.cfg.Blocklists.Sources[data.Index].Enabled = data.Enabled
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Blocklists.Sources[data.Index].Enabled = previous
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save blocklist source: " + err.Error(), "code": "config_save_failed"})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) removeSource(c *gin.Context) {
	idx, err := strconv.Atoi(c.Param("index"))
	if err != nil || idx < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid index"})
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	if idx >= len(s.cfg.Blocklists.Sources) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid index"})
		return
	}
	previous := append([]config.BlocklistSource(nil), s.cfg.Blocklists.Sources...)
	s.cfg.Blocklists.Sources = append(s.cfg.Blocklists.Sources[:idx], s.cfg.Blocklists.Sources[idx+1:]...)
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Blocklists.Sources = previous
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove blocklist source: " + err.Error(), "code": "config_save_failed"})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) setAutoUpdate(c *gin.Context) {
	var data struct {
		Hours int `json:"hours"`
	}
	if err := c.ShouldBindJSON(&data); err != nil || data.Hours < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "non-negative 'hours' required"})
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	previous := s.cfg.Blocklists.AutoUpdateInterval
	s.cfg.Blocklists.AutoUpdateInterval = data.Hours
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Blocklists.AutoUpdateInterval = previous
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save auto-update interval: " + err.Error(), "code": "config_save_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "hours": data.Hours})
}

// ─── Domain tester (CLI: `test <domain>`) ─────────────────────────────────────

func (s *Server) testDomain(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain query param required"})
		return
	}
	blocked, reason := s.filter.TestDomain(domain)
	c.JSON(http.StatusOK, gin.H{"domain": domain, "blocked": blocked, "reason": reason})
}

// ─── Global filtering toggle / pause (like Pi-hole) ───────────────────────────

func (s *Server) getFiltering(c *gin.Context) {
	if s.dnsServer == nil {
		c.JSON(http.StatusOK, gin.H{"enabled": true, "paused_until": 0})
		return
	}
	enabled, pausedUntil := s.dnsServer.FilteringStatus()
	c.JSON(http.StatusOK, gin.H{"enabled": enabled, "paused_until": pausedUntil})
}

func (s *Server) setFiltering(c *gin.Context) {
	var data struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.SetFiltering(data.Enabled)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "enabled": data.Enabled})
}

func (s *Server) pauseFiltering(c *gin.Context) {
	var data struct {
		Minutes int `json:"minutes"`
	}
	if err := c.ShouldBindJSON(&data); err != nil || data.Minutes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "positive 'minutes' required"})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.PauseFiltering(time.Duration(data.Minutes) * time.Minute)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "minutes": data.Minutes})
}

// ─── System info (CLI: `info`) ────────────────────────────────────────────────

func (s *Server) systemInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":      appversion.Current,
		"dns_port":     s.cfg.Server.DNSPort,
		"dns_host":     s.cfg.Server.DNSHost,
		"api_port":     s.cfg.Server.APIPort,
		"upstream_dns": s.cfg.Server.UpstreamDNS,
		"secure_dns":   s.cfg.Advanced.SecureDNSOnly,
		"cache_ttl":    s.cfg.Server.CacheTTL,
	})
}

// ─── Middleman mode ───────────────────────────────────────────────────────────

func (s *Server) dnsStatus(c *gin.Context) {
	iface, dns, active, err := netdns.Status()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"active": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"active": active, "interface": iface, "dns": dns})
}

func (s *Server) dnsSetup(c *gin.Context) {
	if err := netdns.SetDNS(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"hint":  "The server must be running as Administrator/root to change system DNS.",
		})
		return
	}
	s.configMu.Lock()
	previous := s.cfg.Server.ManageSystemDNS
	s.cfg.Server.ManageSystemDNS = true
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Server.ManageSystemDNS = previous
		s.configMu.Unlock()
		if !previous {
			_ = netdns.ResetDNS()
		}
		if s.dnsWatch != nil {
			s.dnsWatch.SetAutoRepair(previous)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "System DNS changed but the routing preference could not be saved; DNS was restored"})
		return
	}
	s.configMu.Unlock()
	if s.dnsWatch != nil {
		s.dnsWatch.SetAutoRepair(true)
		_ = s.dnsWatch.Check()
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "active": true})
}

func (s *Server) dnsReset(c *gin.Context) {
	if s.dnsWatch != nil {
		s.dnsWatch.SetAutoRepair(false)
	}
	if err := netdns.ResetDNS(); err != nil {
		if s.dnsWatch != nil {
			s.dnsWatch.SetAutoRepair(s.cfg.Server.ManageSystemDNS)
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"hint":  "The server must be running as Administrator/root to change system DNS.",
		})
		return
	}
	s.configMu.Lock()
	previous := s.cfg.Server.ManageSystemDNS
	s.cfg.Server.ManageSystemDNS = false
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Server.ManageSystemDNS = previous
		s.configMu.Unlock()
		if previous {
			_ = netdns.SetDNS()
			if s.dnsWatch != nil {
				s.dnsWatch.SetAutoRepair(true)
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Automatic DNS was applied but the routing preference could not be saved; the previous route was restored"})
		return
	}
	s.configMu.Unlock()
	c.JSON(http.StatusOK, gin.H{"success": true, "active": false})
}

// queryLimit parses a ?limit= query param with a fallback default.
func queryLimit(c *gin.Context, def int) int {
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// ─── Categories ───────────────────────────────────────────────────────────────

func (s *Server) getCategories(c *gin.Context) {
	c.JSON(http.StatusOK, s.filter.GetAllCategories())
}

func (s *Server) toggleCategory(c *gin.Context) {
	var data struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.filter.ToggleCategory(c.Param("id"), data.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": c.Param("id"), "enabled": data.Enabled})
}

// ─── Keywords ─────────────────────────────────────────────────────────────────

func (s *Server) getKeywordLists(c *gin.Context) {
	c.JSON(http.StatusOK, s.filter.GetAllKeywordLists())
}

func (s *Server) toggleKeywordList(c *gin.Context) {
	var data struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.filter.ToggleKeywordList(c.Param("id"), data.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": c.Param("id"), "enabled": data.Enabled})
}

func (s *Server) getKeywordMatches(c *gin.Context) {
	matches, err := s.filter.GetKeywordMatchStats(queryLimit(c, 20))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, matches)
}

// ─── Gamification ─────────────────────────────────────────────────────────────

func (s *Server) getUserProfile(c *gin.Context) {
	userID := c.DefaultQuery("user", "default_user")
	profile, err := s.filter.GetUserProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (s *Server) getLeaderboard(c *gin.Context) {
	board, err := s.filter.GetLeaderboard(queryLimit(c, 10))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, board)
}

// ─── AI Blocker ───────────────────────────────────────────────────────────────

func (s *Server) getAIStats(c *gin.Context) {
	stats := s.filter.GetAIStats()
	if stats == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI blocker not initialized"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (s *Server) getAIPredictions(c *gin.Context) {
	preds, err := s.filter.GetAIRecentPredictions(queryLimit(c, 20))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, preds)
}

func (s *Server) analyzeDomain(c *gin.Context) {
	var body struct {
		Domain string `json:"domain"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Domain) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain is required"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(body.Domain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	if i := strings.IndexByte(domain, '/'); i >= 0 {
		domain = domain[:i]
	}
	if len(domain) > 253 || strings.ContainsAny(domain, " @\\") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain"})
		return
	}
	c.JSON(http.StatusOK, s.filter.AnalyzeDomain(domain))
}
