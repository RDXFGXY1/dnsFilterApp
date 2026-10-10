package api

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
	"github.com/gin-gonic/gin"
)

func (s *Server) discFocusCapsules(c *gin.Context) {
	items, err := s.discipline.ListFocusCapsules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) discSaveFocusCapsule(c *gin.Context) {
	var input discipline.FocusCapsuleInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid capsule"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	item, err := s.discipline.SaveFocusCapsule(id, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "capsule": item})
}

func (s *Server) discDeleteFocusCapsule(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid capsule"})
		return
	}
	if err := s.discipline.DeleteFocusCapsule(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type extensionFocusRequest struct {
	URL      string   `json:"url"`
	Title    string   `json:"title"`
	Channels []string `json:"channels"`
}

func (s *Server) extensionFocusCheck(c *gin.Context) {
	ip := net.ParseIP(c.ClientIP())
	origin := c.GetHeader("Origin")
	if ip == nil || !ip.IsLoopback() || !extensionOriginAllowed(origin) || s.discipline == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "local browser extension required"})
		return
	}
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Vary", "Origin")
	var input extensionFocusRequest
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid page information required"})
		return
	}
	decision := s.discipline.EvaluateFocusPage(input.URL, input.Channels)
	if decision.Active && strings.TrimSpace(input.URL) != "" {
		if parsed, err := url.Parse(input.URL); err == nil && parsed.Hostname() != "" {
			s.discipline.RecordFocusActivity("domain", parsed.Hostname(), 0)
		}
		if !decision.Allowed {
			_ = s.discipline.AddDistraction(decision.SessionID, "Blocked navigation", input.Title)
		}
	}
	c.JSON(http.StatusOK, decision)
}
