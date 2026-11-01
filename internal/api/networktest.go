package api

import (
	"net/http"

	"github.com/RDXFGXY1/dns-filter-app/internal/nettest"
	"github.com/gin-gonic/gin"
)

func (s *Server) networkTestStart(c *gin.Context) {
	if s.networkTest == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "network test is unavailable"})
		return
	}
	var body nettest.Options
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid test settings are required"})
		return
	}
	result, err := s.networkTest.StartWithOptions(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, result)
}

func (s *Server) networkTestHistory(c *gin.Context) {
	items, err := s.db.GetNetworkTestHistory(30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load Network Test history"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) networkTestStatus(c *gin.Context) {
	if s.networkTest == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "network test is unavailable"})
		return
	}
	result, ok := s.networkTest.Get(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "network test not found"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) networkTestCancel(c *gin.Context) {
	if s.networkTest == nil || !s.networkTest.Cancel(c.Param("id")) {
		c.JSON(http.StatusNotFound, gin.H{"error": "network test not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
