package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) networkTestStart(c *gin.Context) {
	if s.networkTest == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "network test is unavailable"})
		return
	}
	var body struct {
		Mode string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid test settings are required"})
		return
	}
	c.JSON(http.StatusAccepted, s.networkTest.Start(body.Mode))
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
