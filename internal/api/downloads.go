package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func (s *Server) downloadEvents(c *gin.Context) {
	if s.downloadGuard == nil {
		c.JSON(503, gin.H{"error": "Download Guardian unavailable"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := s.downloadGuard.List(limit)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"folder": s.downloadGuard.Folder(), "events": items, "count": len(items)})
}
func (s *Server) downloadAnalytics(c *gin.Context) {
	if s.downloadGuard == nil {
		c.JSON(503, gin.H{"error": "Download Guardian unavailable"})
		return
	}
	v, err := s.downloadGuard.Analytics()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, v)
}
func (s *Server) downloadQuarantine(c *gin.Context) { s.downloadAction(c, true) }
func (s *Server) downloadRestore(c *gin.Context)    { s.downloadAction(c, false) }
func (s *Server) downloadAction(c *gin.Context, quarantine bool) {
	if s.downloadGuard == nil {
		c.JSON(503, gin.H{"error": "Download Guardian unavailable"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid download event"})
		return
	}
	if quarantine {
		err = s.downloadGuard.Quarantine(id)
	} else {
		err = s.downloadGuard.Restore(id)
	}
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
