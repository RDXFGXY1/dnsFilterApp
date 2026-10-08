package api

import (
	"net/http"
	"strconv"

	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
	"github.com/gin-gonic/gin"
)

func journalID(c *gin.Context) int64 {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	return id
}

func (s *Server) discJournalTypes(c *gin.Context) {
	items, err := s.discipline.ListJournalTypes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) discCreateJournalType(c *gin.Context) {
	var input struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	id, err := s.discipline.CreateJournalType(input.Name, input.Color)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

func (s *Server) discDeleteJournalType(c *gin.Context) {
	if err := s.discipline.DeleteJournalType(journalID(c)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) discJournalEntries(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := s.discipline.ListJournalEntries(discipline.JournalFilters{Query: c.Query("q"), TypeID: queryInt64(c, "type"), From: c.Query("from"), To: c.Query("to"), Limit: limit})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) discCreateJournalEntry(c *gin.Context) {
	var input discipline.JournalEntryInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	id, err := s.discipline.CreateJournalEntry(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id})
}

func (s *Server) discUpdateJournalEntry(c *gin.Context) {
	var input discipline.JournalEntryInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.discipline.UpdateJournalEntry(journalID(c), input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) discDeleteJournalEntry(c *gin.Context) {
	if err := s.discipline.DeleteJournalEntry(journalID(c)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) discJournalStats(c *gin.Context) {
	stats, err := s.discipline.JournalStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
