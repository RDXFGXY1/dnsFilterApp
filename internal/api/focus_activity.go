package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
)

func focusSessionID(c *gin.Context) int64 {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	return id
}

func (s *Server) discStartFocusSession(c *gin.Context) {
	var x struct {
		TaskID          int64  `json:"task_id"`
		TaskTitle       string `json:"task_title"`
		Minutes         int    `json:"minutes"`
		ActivityLogging bool   `json:"activity_logging"`
		CapsuleID       int64  `json:"capsule_id"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	id, err := s.discipline.StartFocusSession(x.TaskID, x.TaskTitle, x.Minutes, x.ActivityLogging, x.CapsuleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if x.CapsuleID > 0 && s.appNetwork != nil {
		capsule, capsuleErr := s.discipline.FocusCapsule(x.CapsuleID)
		if capsuleErr == nil && len(capsule.AllowedApps) > 0 {
			if err := s.appNetwork.BeginFocusAllowlist(capsule.AllowedApps, time.Duration(x.Minutes)*time.Minute); err != nil {
				_, _ = s.discipline.EndFocusSession(id, "cancelled", "Application allowlist could not start", 0, false)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

func (s *Server) discActiveFocusSession(c *gin.Context) {
	id := s.discipline.ActiveFocusSessionID()
	if id == 0 {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}
	item, err := s.discipline.FocusSessionDetails(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"active": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"active": true, "session": item})
}

func (s *Server) discFocusDistraction(c *gin.Context) {
	var x struct {
		Label string `json:"label"`
		Note  string `json:"note"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.discipline.AddDistraction(focusSessionID(c), x.Label, x.Note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true})
}

func (s *Server) discEndFocusSession(c *gin.Context) {
	var x struct {
		Status        string `json:"status"`
		Reflection    string `json:"reflection"`
		FocusScore    int    `json:"focus_score"`
		CreateJournal bool   `json:"create_journal"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	item, err := s.discipline.EndFocusSession(focusSessionID(c), x.Status, x.Reflection, x.FocusScore, x.CreateJournal)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	warning := ""
	if s.appNetwork != nil {
		if releaseErr := s.appNetwork.EndFocusAllowlist(); releaseErr != nil {
			warning = releaseErr.Error()
		}
	}
	messageID := int64(0)
	if x.Status == "completed" {
		message := localFocusCoach(item)
		messageID, _ = s.discipline.CreateInboxMessage(discipline.InboxInput{
			Sender: "Discipline Session Coach", Subject: message.Subject, Body: message.Body,
			Category: message.Category, Severity: message.Severity,
			ActionLabel: message.ActionLabel, ActionView: message.ActionView,
		})
		if messageID > 0 {
			_ = s.discipline.SetFocusReviewState(item.ID, "ready", "local", messageID, "")
			item, _ = s.discipline.FocusSessionDetails(item.ID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "session": item, "warning": warning, "message_id": messageID})
}

func (s *Server) discFocusSessions(c *gin.Context) {
	items, err := s.discipline.RecentFocusSessions(30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}
func (s *Server) discFocusSession(c *gin.Context) {
	item, err := s.discipline.FocusSessionDetails(focusSessionID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) discFocusCoach(c *gin.Context) {
	var x struct {
		Regenerate bool `json:"regenerate"`
	}
	_ = c.ShouldBindJSON(&x)
	id, mode, err := s.generateFocusCoach(c.Request.Context(), focusSessionID(c), x.Regenerate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message_id": id, "mode": mode})
}
