package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/networkintel"
	"github.com/gin-gonic/gin"
)

func (s *Server) networkTimeline(c *gin.Context) {
	if s.networkIntel == nil || !s.networkIntel.Supported() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Connection Intelligence requires Windows"})
		return
	}
	refreshErr := s.networkIntel.Refresh()
	items, err := s.networkIntel.Recent(200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response := gin.H{"connections": items, "supported": true}
	if refreshErr != nil {
		response["warning"] = refreshErr.Error()
	}
	c.JSON(http.StatusOK, response)
}
func (s *Server) networkAlerts(c *gin.Context) {
	if s.networkIntel == nil {
		c.JSON(503, gin.H{"error": "Connection Intelligence unavailable"})
		return
	}
	_ = s.networkIntel.Refresh()
	items, err := s.networkIntel.Alerts()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, items)
}
func (s *Server) networkAlertReview(c *gin.Context) {
	var in struct {
		Path string `json:"path"`
	}
	if c.ShouldBindJSON(&in) != nil || in.Path == "" {
		c.JSON(400, gin.H{"error": "application path required"})
		return
	}
	if err := s.networkIntel.Review(in.Path); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) networkPrivacy(c *gin.Context) {
	if s.networkIntel == nil {
		c.JSON(503, gin.H{"error": "Connection Intelligence unavailable"})
		return
	}
	_ = s.networkIntel.Refresh()
	items, err := s.networkIntel.Privacy()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, items)
}

func (s *Server) networkPrivacyDetails(c *gin.Context) {
	if s.networkIntel == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Connection Intelligence unavailable"})
		return
	}
	path := c.Query("path")
	items, err := s.networkIntel.AppConnections(path, 300)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": path, "connections": items})
}

func (s *Server) networkPrivacyBaseline(c *gin.Context) {
	if s.networkIntel == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Connection Intelligence unavailable"})
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if c.ShouldBindJSON(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application path required"})
		return
	}
	count, err := s.networkIntel.TrustCurrentDestinations(body.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "trusted": count})
}

func (s *Server) networkDestinationAlerts(c *gin.Context) {
	if s.networkIntel == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Connection Intelligence unavailable"})
		return
	}
	items, err := s.networkIntel.NewDestinations(300)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"connections": items, "count": len(items)})
}

func (s *Server) networkDestinationTrust(c *gin.Context) {
	if s.networkIntel == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Connection Intelligence unavailable"})
		return
	}
	var body struct {
		Path       string `json:"path"`
		RemoteIP   string `json:"remote_ip"`
		RemotePort int    `json:"remote_port"`
		Protocol   string `json:"protocol"`
	}
	if c.ShouldBindJSON(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application and destination details are required"})
		return
	}
	if err := s.networkIntel.TrustDestination(body.Path, body.RemoteIP, body.RemotePort, body.Protocol); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) networkHistoryStatus(c *gin.Context) {
	status, err := s.networkIntel.HistoryStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (s *Server) networkHistoryRetention(c *gin.Context) {
	var body struct {
		Days int `json:"days"`
	}
	if c.ShouldBindJSON(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "retention days are required"})
		return
	}
	if err := s.networkIntel.SetRetentionDays(body.Days); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "retention_days": body.Days})
}

func (s *Server) networkHistoryClear(c *gin.Context) {
	if err := s.networkIntel.ClearHistory(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) networkConnectionExport(c *gin.Context) {
	items, err := s.networkIntel.ConnectionReport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	format := c.DefaultQuery("format", "json")
	stamp := time.Now().Format("20060102-150405")
	if format == "json" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=dns-filter-connections-%s.json", stamp))
		c.JSON(http.StatusOK, gin.H{"generated_at": time.Now().UTC(), "connections": items})
		return
	}
	if format != "csv" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format must be csv or json"})
		return
	}
	var output bytes.Buffer
	w := csv.NewWriter(&output)
	_ = w.Write([]string{"application", "path", "hostname", "remote_ip", "remote_port", "protocol", "risk", "reason", "observations", "trusted", "first_seen", "last_seen"})
	for _, item := range items {
		_ = w.Write([]string{item.Application, item.Path, item.Hostname, item.RemoteIP, strconv.Itoa(item.RemotePort), item.Protocol, strconv.Itoa(item.Risk), item.Reasons, strconv.Itoa(item.Seen), strconv.FormatBool(item.Trusted), item.FirstSeen.Format(time.RFC3339), item.LastSeen.Format(time.RFC3339)})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=dns-filter-connections-%s.csv", stamp))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", output.Bytes())
}

func (s *Server) networkNotificationSettings(c *gin.Context) {
	settings, err := s.networkIntel.NotificationSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (s *Server) networkNotificationSettingsUpdate(c *gin.Context) {
	var settings networkintel.NotificationSettings
	if c.ShouldBindJSON(&settings) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid notification settings are required"})
		return
	}
	if err := s.networkIntel.SetNotificationSettings(settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) networkAutomaticResponse(c *gin.Context) {
	settings, err := s.networkIntel.AutomaticResponseSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	events, err := s.networkIntel.AutomaticResponseEvents(50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings, "events": events})
}

func (s *Server) networkAutomaticResponseUpdate(c *gin.Context) {
	var settings networkintel.AutomaticResponseSettings
	if c.ShouldBindJSON(&settings) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid automatic response settings are required"})
		return
	}
	if err := s.networkIntel.SetAutomaticResponseSettings(settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) networkSecurityAnalytics(c *gin.Context) {
	analytics, err := s.networkIntel.SecurityAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, analytics)
}

func (s *Server) networkLearning(c *gin.Context) {
	if s.networkIntel == nil || !s.networkIntel.Supported() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Learning Mode requires Windows"})
		return
	}
	_ = s.networkIntel.Refresh()
	items, err := s.networkIntel.LearningSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) networkLearningStart(c *gin.Context) {
	if s.networkIntel == nil || !s.networkIntel.Supported() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Learning Mode requires Windows"})
		return
	}
	var body struct {
		Path    string `json:"path"`
		Minutes int    `json:"minutes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	id, err := s.networkIntel.StartLearning(body.Path, body.Minutes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

func (s *Server) networkLearningStop(c *gin.Context) {
	if s.networkIntel == nil || !s.networkIntel.Supported() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Learning Mode requires Windows"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid learning session"})
		return
	}
	if err := s.networkIntel.StopLearning(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) networkLearningBaseline(c *gin.Context) {
	if s.networkIntel == nil || !s.networkIntel.Supported() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Learning Mode requires Windows"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid learning session"})
		return
	}
	session, connections, err := s.networkIntel.LearningBaseline(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "learning session not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session":     session,
		"connections": connections,
		"mode":        "review_only",
		"note":        "This is a learned isolation plan. It is not enforced until a safe Windows Filtering Platform provider is installed.",
	})
}
