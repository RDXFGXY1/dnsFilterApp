package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s *Server) dnsBypassStatus(c *gin.Context) {
	if s.dnsBypass == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "DNS bypass protection unavailable"})
		return
	}
	status, err := s.dnsBypass.Status()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, status)
}
func (s *Server) dnsBypassUpdate(c *gin.Context) {
	if s.dnsBypass == nil {
		c.JSON(503, gin.H{"error": "DNS bypass protection unavailable"})
		return
	}
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "valid state required"})
		return
	}
	if err := s.dnsBypass.SetEnabled(in.Enabled); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	status, _ := s.dnsBypass.Status()
	c.JSON(200, status)
}
