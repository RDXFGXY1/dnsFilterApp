package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/RDXFGXY1/dns-filter-app/internal/securitypacks"
	"github.com/gin-gonic/gin"
)

func (s *Server) listSecurityPacks(c *gin.Context) {
	packs, trustedKeys := s.filter.ListSecurityPacks()
	c.JSON(http.StatusOK, gin.H{"packs": packs, "trusted_key_count": trustedKeys})
}

func (s *Server) verifySecurityPack(c *gin.Context) {
	data, ok := readSecurityPackUpload(c)
	if !ok {
		return
	}
	info, err := s.filter.VerifySecurityPack(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true, "pack": info})
}

func (s *Server) installSecurityPack(c *gin.Context) {
	data, ok := readSecurityPackUpload(c)
	if !ok {
		return
	}
	info, err := s.filter.InstallSecurityPack(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"installed": false, "error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"installed": true, "pack": info})
}

func (s *Server) removeSecurityPack(c *gin.Context) {
	if err := s.filter.RemoveSecurityPack(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) trustSecurityPackKey(c *gin.Context) {
	var body struct {
		ID        string `json:"id"`
		PublicKey string `json:"public_key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.ID) == "" || strings.TrimSpace(body.PublicKey) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key id and base64 public key are required"})
		return
	}
	if err := s.filter.TrustSecurityPackKey(strings.TrimSpace(body.ID), strings.TrimSpace(body.PublicKey)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func readSecurityPackUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, securitypacks.MaxPackBytes+1<<20)
	header, err := c.FormFile("pack")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a security pack JSON file is required"})
		return nil, false
	}
	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not open uploaded security pack"})
		return nil, false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, securitypacks.MaxPackBytes+1))
	if err != nil || len(data) > securitypacks.MaxPackBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("security pack exceeds the %d-byte limit", securitypacks.MaxPackBytes)})
		return nil, false
	}
	return data, true
}
