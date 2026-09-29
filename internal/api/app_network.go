package api

import (
	"net/http"
	"strconv"

	"github.com/RDXFGXY1/dns-filter-app/internal/appnetwork"
	"github.com/gin-gonic/gin"
)

func (s *Server) appNetworkSchedules(c *gin.Context) {
	items, err := s.appNetwork.Schedules()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, items)
}
func (s *Server) appNetworkScheduleAdd(c *gin.Context) {
	var in appnetwork.Schedule
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "valid schedule required"})
		return
	}
	if err := s.appNetwork.AddSchedule(in); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) appNetworkScheduleDelete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid schedule"})
		return
	}
	if err = s.appNetwork.DeleteSchedule(id); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func (s *Server) appNetworkApps(c *gin.Context) {
	if s.appNetwork == nil || !s.appNetwork.Supported() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "App Network Control requires Windows"})
		return
	}
	apps, err := s.appNetwork.Apps()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"supported": true, "apps": apps})
}

func (s *Server) appNetworkEvents(c *gin.Context) {
	if s.appNetwork == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "App Network Control unavailable"})
		return
	}
	events, err := s.appNetwork.Events(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

func (s *Server) appNetworkBlock(c *gin.Context) { s.setAppNetworkState(c, true) }
func (s *Server) appNetworkAllow(c *gin.Context) { s.setAppNetworkState(c, false) }

func (s *Server) setAppNetworkState(c *gin.Context, blocked bool) {
	if s.appNetwork == nil || !s.appNetwork.Supported() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "App Network Control requires Windows"})
		return
	}
	var in struct {
		Path string `json:"path"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application path is required"})
		return
	}
	if err := s.appNetwork.SetBlocked(in.Path, blocked); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "blocked": blocked})
}

func (s *Server) appNetworkQuarantineList(c *gin.Context) {
	if s.appNetwork == nil {
		c.JSON(503, gin.H{"error": "Quarantine unavailable"})
		return
	}
	items, err := s.appNetwork.Quarantined()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"items": items, "supported": s.appNetwork.Supported()})
}
func (s *Server) appNetworkQuarantine(c *gin.Context) {
	if s.appNetwork == nil || !s.appNetwork.Supported() {
		c.JSON(503, gin.H{"error": "Quarantine requires Windows"})
		return
	}
	var in struct {
		Path   string `json:"path"`
		Reason string `json:"reason"`
		Risk   int    `json:"risk"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "application path is required"})
		return
	}
	if err := s.appNetwork.Quarantine(in.Path, in.Reason, in.Risk); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) appNetworkQuarantineRelease(c *gin.Context) {
	if s.appNetwork == nil {
		c.JSON(503, gin.H{"error": "Quarantine unavailable"})
		return
	}
	var in struct {
		Path string `json:"path"`
	}
	if c.ShouldBindJSON(&in) != nil {
		c.JSON(400, gin.H{"error": "application path is required"})
		return
	}
	if err := s.appNetwork.ReleaseQuarantine(in.Path); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
