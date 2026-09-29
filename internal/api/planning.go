package api

import (
	"net/http"
	"strconv"

	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
	"github.com/gin-gonic/gin"
)

func idParam(c *gin.Context) int64 { n, _ := strconv.ParseInt(c.Param("id"), 10, 64); return n }
func (s *Server) registerPlanningRoutes(d *gin.RouterGroup) {
	d.GET("/milestones", s.planMilestones)
	d.POST("/milestones", s.planCreateMilestone)
	d.PATCH("/milestones/:id", s.planUpdateMilestone)
	d.DELETE("/milestones/:id", s.planDeleteMilestone)
	d.GET("/cycles", s.planCycles)
	d.POST("/cycles", s.planCreateCycle)
	d.PATCH("/cycles/:id", s.planUpdateCycle)
	d.DELETE("/cycles/:id", s.planDeleteCycle)
	d.GET("/cycle-assignments", s.planCycleAssignments)
	d.POST("/tasks/:id/cycle", s.planAssignCycle)
	d.GET("/tasks/:id/checklist", s.planChecklist)
	d.POST("/tasks/:id/checklist", s.planAddChecklist)
	d.PATCH("/checklist/:id", s.planToggleChecklist)
	d.DELETE("/checklist/:id", s.planDeleteChecklist)
	d.GET("/dependencies", s.planDependencies)
	d.POST("/tasks/:id/dependencies", s.planSetDependency)
	d.POST("/tasks/:id/duplicate", s.planDuplicateTask)
	d.GET("/tasks/:id/activity", s.planTaskActivity)
	d.GET("/tasks/:id/related", s.planRelatedTasks)
	d.POST("/tasks/:id/related", s.planSetRelatedTask)
	d.PATCH("/bulk/tasks", s.planBulkTasks)
}
func respondErr(c *gin.Context, e error) { c.JSON(http.StatusBadRequest, gin.H{"error": e.Error()}) }
func (s *Server) planMilestones(c *gin.Context) {
	x, e := s.discipline.ListMilestones(queryInt64(c, "project"))
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(http.StatusOK, x)
}
func (s *Server) planCreateMilestone(c *gin.Context) {
	var x struct {
		ProjectID   int64  `json:"project_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		StartDate   string `json:"start_date"`
		TargetDate  string `json:"target_date"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	id, e := s.discipline.CreateMilestone(x.ProjectID, x.Name, x.Description, x.StartDate, x.TargetDate)
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id})
}
func (s *Server) planUpdateMilestone(c *gin.Context) {
	var p discipline.MilestonePatch
	if c.ShouldBindJSON(&p) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if e := s.discipline.UpdateMilestone(idParam(c), p); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) planDeleteMilestone(c *gin.Context) {
	if e := s.discipline.DeleteMilestone(idParam(c)); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) planCycles(c *gin.Context) {
	x, e := s.discipline.ListCycles(queryInt64(c, "project"))
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, x)
}
func (s *Server) planCreateCycle(c *gin.Context) {
	var x struct {
		ProjectID int64  `json:"project_id"`
		Name      string `json:"name"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	id, e := s.discipline.CreateCycle(x.ProjectID, x.Name, x.StartDate, x.EndDate)
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true, "id": id})
}
func (s *Server) planUpdateCycle(c *gin.Context) {
	var p discipline.CyclePatch
	if c.ShouldBindJSON(&p) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if e := s.discipline.UpdateCycle(idParam(c), p); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) planDeleteCycle(c *gin.Context) {
	if e := s.discipline.DeleteCycle(idParam(c)); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) planCycleAssignments(c *gin.Context) {
	x, e := s.discipline.CycleAssignments(queryInt64(c, "project"))
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, x)
}
func (s *Server) planAssignCycle(c *gin.Context) {
	var x struct {
		CycleID int64 `json:"cycle_id"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if e := s.discipline.AssignTaskCycle(idParam(c), x.CycleID); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) planChecklist(c *gin.Context) {
	x, e := s.discipline.Checklist(idParam(c))
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, x)
}
func (s *Server) planAddChecklist(c *gin.Context) {
	var x struct {
		Text string `json:"text"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	id, e := s.discipline.AddChecklist(idParam(c), x.Text)
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true, "id": id})
}
func (s *Server) planToggleChecklist(c *gin.Context) {
	var x struct {
		Done bool `json:"done"`
	}
	_ = c.ShouldBindJSON(&x)
	if e := s.discipline.ToggleChecklist(idParam(c), x.Done); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) planDeleteChecklist(c *gin.Context) {
	if e := s.discipline.DeleteChecklist(idParam(c)); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) planDependencies(c *gin.Context) {
	x, e := s.discipline.ProjectDependencies(queryInt64(c, "project"))
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, x)
}
func (s *Server) planSetDependency(c *gin.Context) {
	var x struct {
		DependsOnID int64 `json:"depends_on_id"`
		Link        bool  `json:"link"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if e := s.discipline.SetDependency(idParam(c), x.DependsOnID, x.Link); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) planDuplicateTask(c *gin.Context) {
	id, e := s.discipline.DuplicateTask(idParam(c))
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true, "id": id})
}
func (s *Server) planTaskActivity(c *gin.Context) {
	x, e := s.discipline.TaskEvents(idParam(c))
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, x)
}
func (s *Server) planRelatedTasks(c *gin.Context) {
	x, e := s.discipline.RelatedTasks(idParam(c))
	if e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, x)
}
func (s *Server) planSetRelatedTask(c *gin.Context) {
	var x struct {
		RelatedID int64 `json:"related_id"`
		Link      bool  `json:"link"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	if e := s.discipline.SetRelatedTask(idParam(c), x.RelatedID, x.Link); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) planBulkTasks(c *gin.Context) {
	var x struct {
		IDs   []int64              `json:"ids"`
		Patch discipline.TaskPatch `json:"patch"`
	}
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	for _, id := range x.IDs {
		if _, _, e := s.discipline.UpdateTask(id, x.Patch); e != nil {
			respondErr(c, e)
			return
		}
	}
	c.JSON(200, gin.H{"success": true, "count": len(x.IDs)})
}
