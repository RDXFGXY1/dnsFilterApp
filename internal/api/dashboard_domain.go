package api

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/RDXFGXY1/dns-filter-app/internal/localdomain"
	"github.com/gin-gonic/gin"
)

func validLocalDashboardDomain(input string) (string, bool) {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(input), "."))
	if len(domain) < 4 || len(domain) > 253 || !strings.Contains(domain, ".") || net.ParseIP(domain) != nil {
		return "", false
	}
	for _, blocked := range []string{".localhost", ".local", ".test", ".invalid", ".example"} {
		if strings.HasSuffix(domain, blocked) {
			return "", false
		}
	}
	for _, label := range strings.Split(domain, ".") {
		if len(label) < 1 || len(label) > 63 || !asciiAlnum(label[0]) || !asciiAlnum(label[len(label)-1]) {
			return "", false
		}
		for i := 1; i < len(label)-1; i++ {
			if !asciiAlnum(label[i]) && label[i] != '-' {
				return "", false
			}
		}
	}
	return domain, true
}

func asciiAlnum(c byte) bool { return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' }

func (s *Server) getDashboardDomain(c *gin.Context) {
	domain := s.dashboardDomain()
	url := ""
	if domain != "" {
		url = "https://" + domain + "/"
	}
	c.JSON(http.StatusOK, gin.H{"domain": domain, "url": url, "hosts_ready": domain != "" && localdomain.Ready(domain), "supported": runtime.GOOS == "windows", "fallback": fmt.Sprintf("http://127.0.0.1:%d/", s.cfg.Server.APIPort)})
}

func (s *Server) setDashboardDomain(c *gin.Context) {
	var input struct {
		Domain string `json:"domain"`
	}
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter a domain name"})
		return
	}
	domain, ok := validLocalDashboardDomain(input.Domain)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter a valid domain you own, such as guard.example.org (no scheme or port)"})
		return
	}
	s.configMu.Lock()
	previous := s.cfg.Server.DashboardDomain
	if domain == previous && localdomain.Ready(domain) {
		s.configMu.Unlock()
		c.JSON(http.StatusOK, gin.H{"success": true, "domain": domain, "url": "https://" + domain + "/"})
		return
	}
	if err := localdomain.Apply(domain); err != nil {
		s.configMu.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update the local Windows hosts entry: " + err.Error()})
		return
	}
	s.cfg.Server.DashboardDomain = domain
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Server.DashboardDomain = previous
		rollback := localdomain.Apply(previous)
		s.configMu.Unlock()
		message := "Could not save dashboard domain; local hosts entry was restored"
		if rollback != nil {
			message = "Could not save dashboard domain and could not restore the previous hosts entry: " + rollback.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}
	s.configMu.Unlock()
	if s.filter != nil {
		s.filter.SetDashboardAlias(domain)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "domain": domain, "url": "https://" + domain + "/", "sign_in_again": true})
}

func (s *Server) getDashboardLAN(c *gin.Context) {
	s.configMu.Lock()
	enabled := s.cfg.Server.DashboardLANEnabled
	allowed := append([]string{}, s.cfg.Server.DashboardAllowedIPs...)
	domain := s.cfg.Server.DashboardDomain
	s.configMu.Unlock()
	c.JSON(http.StatusOK, gin.H{"enabled": enabled, "allowed_ips": allowed, "domain": domain, "internet_available": false})
}

func (s *Server) setDashboardLAN(c *gin.Context) {
	var input struct {
		Enabled    bool     `json:"enabled"`
		AllowedIPs []string `json:"allowed_ips"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.AllowedIPs) > 16 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter up to 16 allowed private IP addresses"})
		return
	}
	seen := make(map[string]bool)
	allowed := make([]string, 0, len(input.AllowedIPs))
	for _, raw := range input.AllowedIPs {
		ip := net.ParseIP(strings.TrimSpace(raw))
		if ip == nil || !ip.IsPrivate() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only exact private LAN IP addresses are allowed; ranges and public IPs are not accepted"})
			return
		}
		canonical := ip.String()
		if !seen[canonical] {
			seen[canonical] = true
			allowed = append(allowed, canonical)
		}
	}
	s.configMu.Lock()
	if input.Enabled && (s.cfg.Server.DashboardDomain == "" || len(allowed) == 0) {
		s.configMu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Set a dashboard domain and at least one allowed device IP before enabling LAN access"})
		return
	}
	previousEnabled := s.cfg.Server.DashboardLANEnabled
	previousAllowed := s.cfg.Server.DashboardAllowedIPs
	s.cfg.Server.DashboardLANEnabled = input.Enabled
	s.cfg.Server.DashboardAllowedIPs = allowed
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Server.DashboardLANEnabled = previousEnabled
		s.cfg.Server.DashboardAllowedIPs = previousAllowed
		s.configMu.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save LAN access settings"})
		return
	}
	s.configMu.Unlock()
	if s.filter != nil {
		s.filter.SetDashboardLAN(input.Enabled, allowed)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "enabled": input.Enabled, "allowed_ips": allowed})
}
