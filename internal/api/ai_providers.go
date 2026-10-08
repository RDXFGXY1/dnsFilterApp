package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type aiProviderView struct {
	ID           int64  `json:"id"`
	Provider     string `json:"provider"`
	Name         string `json:"name"`
	Model        string `json:"model"`
	BaseURL      string `json:"base_url"`
	Enabled      bool   `json:"enabled"`
	Default      bool   `json:"is_default"`
	HasKey       bool   `json:"has_key"`
	LastStatus   string `json:"last_status"`
	LastError    string `json:"last_error"`
	LastTestedAt string `json:"last_tested_at"`
}
type aiProviderInput struct {
	ID       int64  `json:"id"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Enabled  bool   `json:"enabled"`
	Default  bool   `json:"is_default"`
}

func aiProviderEndpoint(provider, custom string) (string, error) {
	switch provider {
	case "openai":
		return "https://api.openai.com/v1", nil
	case "xai":
		return "https://api.x.ai/v1", nil
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta", nil
	}
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(custom), "/"))
	if err != nil || u.Host == "" {
		return "", errors.New("valid base URL required")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	local := strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())
	if u.Scheme != "https" && !local {
		return "", errors.New("custom providers require HTTPS or a loopback address")
	}
	if provider != "compatible" {
		return "", errors.New("unsupported provider")
	}
	return u.String(), nil
}

func (s *Server) aiProviderList(c *gin.Context) {
	rows, e := s.db.GetDB().Query(`SELECT id,provider,name,model,base_url,enabled,is_default,length(api_key_encrypted)>0,last_status,last_error,last_tested_at FROM ai_provider_configs ORDER BY is_default DESC,name`)
	if e != nil {
		c.JSON(500, gin.H{"error": e.Error()})
		return
	}
	defer rows.Close()
	out := []aiProviderView{}
	for rows.Next() {
		var x aiProviderView
		if e = rows.Scan(&x.ID, &x.Provider, &x.Name, &x.Model, &x.BaseURL, &x.Enabled, &x.Default, &x.HasKey, &x.LastStatus, &x.LastError, &x.LastTestedAt); e != nil {
			c.JSON(500, gin.H{"error": e.Error()})
			return
		}
		out = append(out, x)
	}
	c.JSON(200, out)
}

func (s *Server) aiProviderSave(c *gin.Context) {
	if s.vault == nil {
		c.JSON(503, gin.H{"error": "Secure Vault unavailable"})
		return
	}
	var x aiProviderInput
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "Invalid provider settings"})
		return
	}
	x.Provider = strings.ToLower(strings.TrimSpace(x.Provider))
	x.Name = strings.TrimSpace(x.Name)
	x.Model = strings.TrimSpace(x.Model)
	base, e := aiProviderEndpoint(x.Provider, x.BaseURL)
	if e != nil || x.Name == "" || x.Model == "" {
		if e == nil {
			e = errors.New("name and model are required")
		}
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	var sealed []byte
	if strings.TrimSpace(x.APIKey) != "" {
		sealed, e = s.vault.Encrypt([]byte(strings.TrimSpace(x.APIKey)))
		if e != nil {
			c.JSON(423, gin.H{"error": "Unlock Secure Vault before saving an API key", "code": "vault_locked"})
			return
		}
	} else if x.ID > 0 {
		e = s.db.GetDB().QueryRow(`SELECT api_key_encrypted FROM ai_provider_configs WHERE id=?`, x.ID).Scan(&sealed)
		if e != nil {
			c.JSON(400, gin.H{"error": "API key is required"})
			return
		}
	} else {
		c.JSON(400, gin.H{"error": "API key is required"})
		return
	}
	tx, e := s.db.GetDB().Begin()
	if e != nil {
		c.JSON(500, gin.H{"error": e.Error()})
		return
	}
	defer tx.Rollback()
	if x.Default {
		_, _ = tx.Exec(`UPDATE ai_provider_configs SET is_default=0`)
	}
	if x.ID > 0 {
		_, e = tx.Exec(`UPDATE ai_provider_configs SET provider=?,name=?,model=?,base_url=?,api_key_encrypted=?,enabled=?,is_default=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, x.Provider, x.Name, x.Model, base, sealed, x.Enabled, x.Default, x.ID)
	} else {
		_, e = tx.Exec(`INSERT INTO ai_provider_configs(provider,name,model,base_url,api_key_encrypted,enabled,is_default) VALUES(?,?,?,?,?,?,?)`, x.Provider, x.Name, x.Model, base, sealed, x.Enabled, x.Default)
	}
	if e != nil {
		c.JSON(500, gin.H{"error": e.Error()})
		return
	}
	if e = tx.Commit(); e != nil {
		c.JSON(500, gin.H{"error": e.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func (s *Server) aiProviderDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, e := s.db.GetDB().Exec(`DELETE FROM ai_provider_configs WHERE id=?`, id)
	if e != nil {
		c.JSON(500, gin.H{"error": e.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func (s *Server) aiProviderTest(c *gin.Context) {
	if s.vault == nil {
		c.JSON(503, gin.H{"error": "Secure Vault unavailable"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var provider, model, base string
	var sealed []byte
	e := s.db.GetDB().QueryRow(`SELECT provider,model,base_url,api_key_encrypted FROM ai_provider_configs WHERE id=?`, id).Scan(&provider, &model, &base, &sealed)
	if e != nil {
		c.JSON(404, gin.H{"error": "Provider not found"})
		return
	}
	plain, e := s.vault.Decrypt(sealed)
	if e != nil {
		c.JSON(423, gin.H{"error": "Unlock Secure Vault before testing the provider", "code": "vault_locked"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	var req *http.Request
	if provider == "gemini" {
		req, _ = http.NewRequestWithContext(ctx, "GET", base+"/models", nil)
		req.Header.Set("x-goog-api-key", string(plain))
	} else {
		req, _ = http.NewRequestWithContext(ctx, "GET", strings.TrimRight(base, "/")+"/models", nil)
		req.Header.Set("Authorization", "Bearer "+string(plain))
	}
	req.Header.Set("Accept", "application/json")
	resp, e := http.DefaultClient.Do(req)
	status, detail := "connected", ""
	if e != nil {
		status = "failed"
		detail = e.Error()
	} else {
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			status = "failed"
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 1200))
			detail = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(bytes.TrimSpace(b))))
		}
	}
	if len(detail) > 700 {
		detail = detail[:700]
	}
	_, _ = s.db.GetDB().Exec(`UPDATE ai_provider_configs SET last_status=?,last_error=?,last_tested_at=CURRENT_TIMESTAMP WHERE id=?`, status, detail, id)
	if status == "failed" {
		c.JSON(400, gin.H{"error": "Connection failed", "detail": detail})
		return
	}
	c.JSON(200, gin.H{"success": true, "status": status, "model": model})
}
