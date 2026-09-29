package api

import (
	"net"
	"net/http"
	"strings"

	"github.com/RDXFGXY1/dns-filter-app/internal/hardsystem"
	"github.com/gin-gonic/gin"
)

func extensionOriginAllowed(origin string) bool {
	return strings.HasPrefix(origin, "chrome-extension://") || strings.HasPrefix(origin, "moz-extension://")
}

func (s *Server) hardSystemPreflight(c *gin.Context) {
	if !extensionOriginAllowed(c.GetHeader("Origin")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	c.Header("Access-Control-Allow-Origin", c.GetHeader("Origin"))
	c.Header("Vary", "Origin")
	c.Header("Access-Control-Allow-Headers", "Content-Type")
	c.Header("Access-Control-Allow-Methods", "POST, OPTIONS")
	c.Status(http.StatusNoContent)
}

func (s *Server) hardSystemAnalyze(c *gin.Context) {
	ip := net.ParseIP(c.ClientIP())
	origin := c.GetHeader("Origin")
	if ip == nil || !ip.IsLoopback() || !extensionOriginAllowed(origin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "local browser extension required"})
		return
	}
	if s.hardSystem == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Hard System unavailable"})
		return
	}
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Vary", "Origin")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, hardsystem.MaxTextBytes+8192)
	var in hardsystem.Request
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid page content required"})
		return
	}
	result, err := s.hardSystem.Analyze(in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) hardSystemRecent(c *gin.Context) {
	if s.hardSystem == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Hard System unavailable"})
		return
	}
	items, err := s.hardSystem.Recent(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": true, "threshold": hardsystem.DefaultThreshold, "events": items})
}
