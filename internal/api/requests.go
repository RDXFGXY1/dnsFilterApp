package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) getUnblockRequests(c *gin.Context) {
	status := strings.ToLower(strings.TrimSpace(c.DefaultQuery("status", "all")))
	if status != "all" && status != "pending" && status != "approved" && status != "denied" && status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status filter"})
		return
	}
	items, err := s.filter.ListUnblockRequests(status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) approveUnblockRequest(c *gin.Context) {
	s.decideUnblockRequest(c, "approved")
}

func (s *Server) denyUnblockRequest(c *gin.Context) {
	s.decideUnblockRequest(c, "denied")
}

func (s *Server) decideUnblockRequest(c *gin.Context, decision string) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)
	if len(body.Note) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decision note is too long"})
		return
	}
	item, err := s.filter.GetUnblockRequest(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}
	if item.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "request has already been decided"})
		return
	}
	if decision == "approved" {
		s.filter.AddToWhitelist(item.Domain)
		if s.dnsServer != nil {
			s.dnsServer.ClearCache()
		}
	}
	if err := s.filter.DecideUnblockRequest(id, decision, body.Note); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "status": decision, "domain": item.Domain})
}
