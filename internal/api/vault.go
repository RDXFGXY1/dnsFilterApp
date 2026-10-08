package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/vault"
	"github.com/gin-gonic/gin"
)

var vaultExempt = map[string]bool{
	"/api/vault/status": true, "/api/vault/setup": true, "/api/vault/unlock": true,
	"/api/vault/recover": true, "/api/vault/lock": true,
	"/api/connection-freeze/stop": true, // Recovery remains available while locked.
}

func (s *Server) vaultMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.vault == nil || c.Request.Method == http.MethodGet || vaultExempt[c.Request.URL.Path] || !vaultProtectsPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		if err := s.vault.RequireUnlocked(); err != nil {
			c.AbortWithStatusJSON(http.StatusLocked, gin.H{"error": "Secure configuration vault is locked", "code": "vault_locked"})
			return
		}
		c.Next()
		if c.Writer.Status() < 400 {
			_ = s.vault.Audit("configuration.change", c.Request.Method+" "+c.Request.URL.Path)
		}
	}
}

// vaultProtectsPath limits the vault to security and configuration changes.
// Personal productivity data (todos, notes, discipline, focus sessions) must
// remain usable while the configuration vault is locked.
func vaultProtectsPath(path string) bool {
	protected := []string{
		"/api/connection-freeze",
		"/api/whitelist", "/api/custom-blocklist", "/api/blocklist",
		"/api/settings", "/api/filtering", "/api/categories", "/api/keywords",
		"/api/apps", "/api/guard", "/api/blockpage", "/api/autostart",
		"/api/app-network",
		"/api/dns-bypass",
		"/api/downloads",
		"/api/network-profiles",
		"/api/network-intelligence/privacy/baseline",
		"/api/network-intelligence/destination-alerts/trust",
		"/api/network-intelligence/history",
		"/api/network-intelligence/notifications",
		"/api/network-intelligence/automatic-response",
		"/api/security-packs", "/api/system/dns", "/api/protection",
		"/api/plugin-registry",
		"/api/discipline/ai-providers",
	}
	for _, prefix := range protected {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func (s *Server) registerVaultRoutes(api *gin.RouterGroup) {
	api.GET("/vault/status", s.vaultStatus)
	api.POST("/vault/setup", s.vaultSetup)
	api.POST("/vault/unlock", s.vaultUnlock)
	api.POST("/vault/recover", s.vaultRecover)
	api.POST("/vault/lock", s.vaultLock)
	api.POST("/vault/backup", s.vaultBackup)
	api.POST("/vault/restore", s.vaultRestore)
	api.GET("/vault/audit", s.vaultAudit)
}

func (s *Server) vaultStatus(c *gin.Context) {
	if s.vault == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Vault unavailable"})
		return
	}
	c.JSON(http.StatusOK, s.vault.Status())
}

func (s *Server) vaultSetup(c *gin.Context) {
	if !s.requireVault(c) {
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password is required"})
		return
	}
	code, err := s.vault.Setup(in.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "recovery_code": code, "warning": "This recovery code is shown only once. Store it offline."})
}

func (s *Server) vaultUnlock(c *gin.Context) {
	if !s.requireVault(c) {
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "Password is required"})
		return
	}
	if err := s.vault.Unlock(in.Password); err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, vault.ErrLockedOut) {
			status = http.StatusTooManyRequests
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func (s *Server) vaultRecover(c *gin.Context) {
	if !s.requireVault(c) {
		return
	}
	var in struct {
		RecoveryCode string `json:"recovery_code"`
		NewPassword  string `json:"new_password"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "Recovery code and new password are required"})
		return
	}
	if err := s.vault.Recover(in.RecoveryCode, in.NewPassword); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, vault.ErrLockedOut) {
			status = http.StatusTooManyRequests
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func (s *Server) vaultLock(c *gin.Context) {
	if !s.requireVault(c) {
		return
	}
	s.vault.Lock()
	c.JSON(200, gin.H{"success": true})
}

func (s *Server) requireVault(c *gin.Context) bool {
	if s.vault != nil {
		return true
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Secure Vault is unavailable", "code": "vault_unavailable"})
	return false
}

type vaultBackupDocument struct {
	Version   int           `json:"version"`
	CreatedAt time.Time     `json:"created_at"`
	Config    config.Config `json:"config"`
}

func (s *Server) vaultBackup(c *gin.Context) {
	if !s.requireVault(c) {
		return
	}
	plain, err := json.Marshal(vaultBackupDocument{Version: 1, CreatedAt: time.Now().UTC(), Config: *s.cfg})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	encrypted, err := s.vault.Encrypt(plain)
	if err != nil {
		c.JSON(http.StatusLocked, gin.H{"error": err.Error()})
		return
	}
	_ = s.vault.Audit("backup.create", "encrypted configuration backup created")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=dns-filter-vault-%s.json", time.Now().Format("20060102-150405")))
	c.Data(200, "application/json", encrypted)
}

func (s *Server) vaultRestore(c *gin.Context) {
	if !s.requireVault(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)
	file, _, err := c.Request.FormFile("backup")
	if err != nil {
		c.JSON(400, gin.H{"error": "Encrypted backup file is required"})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(400, gin.H{"error": "Cannot read backup"})
		return
	}
	plain, err := s.vault.Decrypt(data)
	if err != nil {
		c.JSON(400, gin.H{"error": "Backup authentication failed or file is damaged"})
		return
	}
	var doc vaultBackupDocument
	d := json.NewDecoder(bytes.NewReader(plain))
	d.DisallowUnknownFields()
	if d.Decode(&doc) != nil || doc.Version != 1 {
		c.JSON(400, gin.H{"error": "Unsupported backup format"})
		return
	}
	// The current vault location and timeout policy belong to this installation.
	doc.Config.Vault = s.cfg.Vault
	s.configMu.Lock()
	defer s.configMu.Unlock()
	if err = s.cfg.ReplaceWith(doc.Config); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	_ = s.vault.Audit("backup.restore", "encrypted configuration backup restored")
	c.JSON(200, gin.H{"success": true, "restart_required": true})
}

func (s *Server) vaultAudit(c *gin.Context) {
	if !s.requireVault(c) {
		return
	}
	records, err := s.vault.AuditRecords(100)
	if err != nil {
		code := 500
		if errors.Is(err, vault.ErrLocked) {
			code = http.StatusLocked
		}
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, records)
}

func sanitizedAuditDetail(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}
