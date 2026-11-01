package api

import (
	"net/http"

	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/gin-gonic/gin"
)

func (s *Server) livePrivacyStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.LivePrivacy())
}

func (s *Server) updateLivePrivacy(c *gin.Context) {
	var settings database.LivePrivacy
	if err := c.ShouldBindJSON(&settings); err != nil || settings.RetentionHours < 1 || settings.RetentionHours > 720 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Choose a retention period between 1 and 720 hours"})
		return
	}
	s.configMu.Lock()
	defer s.configMu.Unlock()
	previous := s.db.LivePrivacy()
	if err := s.db.SetLivePrivacy(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update saved Live history"})
		return
	}
	old := s.cfg.Logging
	s.cfg.Logging.LiveRetentionHours = settings.RetentionHours
	s.cfg.Logging.LiveHideClientIP = settings.HideClientIP
	s.cfg.Logging.LiveBlockedOnly = settings.BlockedOnly
	if err := s.cfg.SaveDefault(); err != nil {
		s.cfg.Logging = old
		_ = s.db.SetLivePrivacy(previous)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save Live privacy settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "settings": settings})
}
