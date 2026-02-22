package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/internal/dns"
	"github.com/RDXFGXY1/dns-filter-app/internal/filter"
	"github.com/RDXFGXY1/dns-filter-app/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	cfg       *config.Config
	db        *database.DB
	filter    *filter.Engine
	dnsServer *dns.Server
	router    *gin.Engine
	server    *http.Server
	log       *logger.Logger
	
}

func NewServer(cfg *config.Config, db *database.DB, filterEngine *filter.Engine, dnsServer *dns.Server) *Server {
	if cfg.Logging.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		cfg:       cfg,
		db:        db,
		filter:    filterEngine,
		dnsServer: dnsServer,
		router:    router,
		log:       logger.Get(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Static("/static", "./web/static")
	s.router.LoadHTMLGlob("./web/templates/*")

	// Public routes
	s.router.GET("/login", s.handleLoginPage)
	s.router.POST("/login", s.handleLogin)
	s.router.GET("/logout", s.handleLogout)

	// Protected page routes
	s.router.GET("/", s.authCheck(), s.handleDashboard)

	// Protected API routes
	api := s.router.Group("/api")
	api.Use(s.authMiddleware())
	{
		api.GET("/stats", s.getStats)
		api.GET("/stats/blocked", s.getBlockedStats)
		api.GET("/stats/top-blocked", s.getTopBlocked)
		api.GET("/recent", s.getRecentBlocked)
		api.GET("/whitelist", s.getWhitelist)
		api.POST("/whitelist", s.addToWhitelist)
		api.DELETE("/whitelist/:domain", s.removeFromWhitelist)
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
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.APIHost, s.cfg.Server.APIPort)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// authCheck redirects to /login for page routes
func (s *Server) authCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := c.Cookie("session")
		if err != nil || session != "authenticated" {
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
		session, err := c.Cookie("session")
		if err != nil || session != "authenticated" {
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
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "DNS Filter Dashboard",
	})
}

func (s *Server) handleLoginPage(c *gin.Context) {
	session, err := c.Cookie("session")
	if err == nil && session == "authenticated" {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login",
		"error": "",
	})
}

func (s *Server) handleLogin(c *gin.Context) {
	// Support both JSON and HTML form
	var username, password string
	contentType := c.GetHeader("Content-Type")

	if contentType == "application/json" {
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

	validUser := username == s.cfg.Security.AdminUsername
	validPass := verifyPassword(password, s.cfg.Security.AdminPasswordHash)

	if validUser && validPass {
		c.SetCookie("session", "authenticated", s.cfg.Security.SessionTimeout*60, "/", "", false, true)
		if contentType == "application/json" {
			c.JSON(http.StatusOK, gin.H{"success": true, "redirect": "/"})
		} else {
			c.Redirect(http.StatusFound, "/")
		}
	} else {
		s.loginFailed(c, contentType, "Invalid username or password")
	}
}

func (s *Server) loginFailed(c *gin.Context, contentType, msg string) {
	if contentType == "application/json" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
	} else {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"title": "Login",
			"error": msg,
		})
	}
}

func (s *Server) handleLogout(c *gin.Context) {
	c.SetCookie("session", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
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
	c.JSON(http.StatusOK, topBlocked)
}

func (s *Server) getRecentBlocked(c *gin.Context) {
	recent, err := s.db.GetRecentBlocked(50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recent)
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
	s.filter.AddToWhitelist(data.Domain)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) removeFromWhitelist(c *gin.Context) {
	s.filter.RemoveFromWhitelist(c.Param("domain"))
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
	s.filter.AddToCustomBlocklist(data.Domain)
	c.JSON(http.StatusOK, gin.H{"success": true, "domain": data.Domain})
}

func (s *Server) removeFromCustomBlocklist(c *gin.Context) {
	s.filter.RemoveFromCustomBlocklist(c.Param("domain"))
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
