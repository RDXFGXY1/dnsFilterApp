package api

import (
	"net/http"

	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
	"github.com/gin-gonic/gin"
)

func (s *Server) taskTemplates(c *gin.Context) {
	x, e := s.discipline.ListTaskTemplates()
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(http.StatusOK, x)
}
func (s *Server) taskTemplateCreate(c *gin.Context) {
	var x discipline.TaskTemplate
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	id, e := s.discipline.CreateTaskTemplate(x)
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true, "id": id})
}
func (s *Server) taskTemplateDelete(c *gin.Context) {
	if e := s.discipline.DeleteTaskTemplate(idParam(c)); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) taskTemplateApply(c *gin.Context) {
	var x struct {
		ProjectID int64  `json:"project_id"`
		DueDate   string `json:"due_date"`
		MyDay     bool   `json:"my_day"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	id, e := s.discipline.ApplyTaskTemplate(idParam(c), x.ProjectID, x.DueDate, x.MyDay)
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true, "id": id})
}
