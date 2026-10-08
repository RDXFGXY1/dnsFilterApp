package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	id, err := s.discipline.StartFocusSession(x.TaskID, x.TaskTitle, x.Minutes, x.ActivityLogging)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
	c.JSON(http.StatusOK, gin.H{"success": true, "session": item})
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
