package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) dnsRouteWatchStatus(c *gin.Context) {
	if s.dnsWatch == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "DNS Route Watch unavailable"})
		return
	}
	events, err := s.dnsWatch.Events()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"current": s.dnsWatch.Current(), "events": events, "interval_seconds": 30, "auto_repair": s.dnsWatch.AutoRepair()})
}

func (s *Server) dnsRouteWatchCheck(c *gin.Context) {
	if s.dnsWatch == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "DNS Route Watch unavailable"})
		return
	}
	_ = s.dnsWatch.Check()
	c.JSON(http.StatusOK, gin.H{"current": s.dnsWatch.Current()})
}
