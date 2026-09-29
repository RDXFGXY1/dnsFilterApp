package api

import "github.com/gin-gonic/gin"

func (s *Server) registerConnectionFreezeRoutes(api *gin.RouterGroup) {
	api.GET("/connection-freeze", func(c *gin.Context) {
		if s.freeze == nil {
			c.JSON(503, gin.H{"error": "Connection Freeze unavailable"})
			return
		}
		events, err := s.freeze.Events()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": s.freeze.Status(), "events": events})
	})
	api.POST("/connection-freeze/start", func(c *gin.Context) {
		if s.freeze == nil {
			c.JSON(503, gin.H{"error": "Connection Freeze unavailable"})
			return
		}
		var body struct {
			Minutes *int `json:"minutes"`
		}
		if c.ShouldBindJSON(&body) != nil || body.Minutes == nil {
			c.JSON(400, gin.H{"error": "Choose a duration in minutes"})
			return
		}
		if err := s.freeze.Start(*body.Minutes); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"success": true, "status": s.freeze.Status()})
	})
	api.POST("/connection-freeze/stop", func(c *gin.Context) {
		if s.freeze == nil {
			c.JSON(503, gin.H{"error": "Connection Freeze unavailable"})
			return
		}
		if err := s.freeze.Stop("Stopped from dashboard"); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"success": true, "status": s.freeze.Status()})
	})
}
