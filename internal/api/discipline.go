package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RDXFGXY1/dns-filter-app/internal/discipline"
	"github.com/RDXFGXY1/dns-filter-app/internal/vault"
)

// registerDisciplineRoutes wires the /discipline productivity API.
func (s *Server) registerDisciplineRoutes(api *gin.RouterGroup) {
	if s.discipline == nil {
		return
	}
	d := api.Group("/discipline")
	{
		d.GET("/projects", s.discProjects)
		d.POST("/projects", s.discCreateProject)
		d.PATCH("/projects/:id", s.discUpdateProject)
		d.DELETE("/projects/:id", s.discDeleteProject)

		d.GET("/goals", s.discGoals)
		d.POST("/goals", s.discCreateGoal)
		d.PATCH("/goals/:id", s.discUpdateGoal)
		d.DELETE("/goals/:id", s.discDeleteGoal)
		d.POST("/goals/:id/link", s.discLinkGoal)

		d.GET("/habits", s.discHabits)
		d.POST("/habits", s.discCreateHabit)
		d.DELETE("/habits/:id", s.discDeleteHabit)
		d.POST("/habits/:id/checkin", s.discCheckin)

		d.GET("/reminders", s.discReminders)
		d.POST("/reminders", s.discCreateReminder)
		d.PATCH("/reminders/:id", s.discSetReminderDone)
		d.DELETE("/reminders/:id", s.discDeleteReminder)

		d.GET("/tasks", s.discTasks)
		d.POST("/tasks", s.discCreateTask)
		d.PATCH("/tasks/:id", s.discUpdateTask)
		d.DELETE("/tasks/:id", s.discDeleteTask)
		d.GET("/task-templates", s.taskTemplates)
		d.POST("/task-templates", s.taskTemplateCreate)
		d.DELETE("/task-templates/:id", s.taskTemplateDelete)
		d.POST("/task-templates/:id/apply", s.taskTemplateApply)

		d.GET("/heatmap", s.discHeatmap)
		d.GET("/stats", s.discStats)
		d.GET("/day", s.discDay)
		d.GET("/series", s.discSeries)
		d.GET("/recent", s.discRecent)
		d.GET("/journal/types", s.discJournalTypes)
		d.POST("/journal/types", s.discCreateJournalType)
		d.DELETE("/journal/types/:id", s.discDeleteJournalType)
		d.GET("/journal/entries", s.discJournalEntries)
		d.POST("/journal/entries", s.discCreateJournalEntry)
		d.PUT("/journal/entries/:id", s.discUpdateJournalEntry)
		d.DELETE("/journal/entries/:id", s.discDeleteJournalEntry)
		d.GET("/journal/stats", s.discJournalStats)
		d.GET("/inbox", s.discInboxMessages)
		d.GET("/inbox/stats", s.discInboxStats)
		d.POST("/inbox", s.discCreateInboxMessage)
		d.POST("/inbox/briefing", s.discGenerateBriefing)
		d.PATCH("/inbox/:id", s.discUpdateInboxMessage)
		d.DELETE("/inbox/:id", s.discDeleteInboxMessage)
		d.GET("/ai-providers", s.aiProviderList)
		d.POST("/ai-providers", s.aiProviderSave)
		d.POST("/ai-providers/:id/test", s.aiProviderTest)
		d.DELETE("/ai-providers/:id", s.aiProviderDelete)
		d.POST("/focus-sessions/start", s.discStartFocusSession)
		d.GET("/focus-sessions/active", s.discActiveFocusSession)
		d.POST("/focus-sessions/:id/distraction", s.discFocusDistraction)
		d.POST("/focus-sessions/:id/end", s.discEndFocusSession)
		d.GET("/focus-sessions", s.discFocusSessions)
		d.GET("/focus-sessions/:id", s.discFocusSession)
		d.POST("/focus-sessions/:id/coach", s.discFocusCoach)
		d.GET("/focus-capsules", s.discFocusCapsules)
		d.POST("/focus-capsules", s.discSaveFocusCapsule)
		d.PUT("/focus-capsules/:id", s.discSaveFocusCapsule)
		d.DELETE("/focus-capsules/:id", s.discDeleteFocusCapsule)
		s.registerPlanningRoutes(d)
		s.registerProjectTransferRoutes(d)
	}
}

func (s *Server) discInboxMessages(c *gin.Context) {
	x, e := s.discipline.ListInboxMessages(c.DefaultQuery("scope", "inbox"), c.Query("q"))
	if e != nil {
		c.JSON(500, gin.H{"error": e.Error()})
		return
	}
	c.JSON(200, x)
}
func (s *Server) discInboxStats(c *gin.Context) {
	x, e := s.discipline.InboxStats()
	if e != nil {
		c.JSON(500, gin.H{"error": e.Error()})
		return
	}
	c.JSON(200, x)
}
func (s *Server) discCreateInboxMessage(c *gin.Context) {
	var x discipline.InboxInput
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	id, e := s.discipline.CreateInboxMessage(x)
	if e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "id": id})
}
func (s *Server) discGenerateBriefing(c *gin.Context) {
	id, mode, e := s.generateIntelligenceBriefing(c.Request.Context())
	if e != nil {
		status, code := http.StatusBadGateway, "ai_generation_failed"
		if errors.Is(e, vault.ErrLocked) {
			status, code = http.StatusLocked, "vault_locked"
		}
		c.JSON(status, gin.H{"error": e.Error(), "code": code})
		return
	}
	c.JSON(200, gin.H{"success": true, "id": id, "mode": mode})
}
func (s *Server) discUpdateInboxMessage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var x discipline.InboxPatch
	if c.ShouldBindJSON(&x) != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	if e := s.discipline.UpdateInboxMessage(id, x); e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}
func (s *Server) discDeleteInboxMessage(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if e := s.discipline.DeleteInboxMessage(id); e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func discPriorityXP(priority string) int {
	switch priority {
	case "high":
		return 30
	case "low":
		return 5
	default:
		return 15
	}
}

func queryInt64(c *gin.Context, key string) int64 {
	n, _ := strconv.ParseInt(c.Query(key), 10, 64)
	return n
}

// ─── Projects ─────────────────────────────────────────────────────────────────

func (s *Server) discProjects(c *gin.Context) {
	projects, err := s.discipline.ListProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, projects)
}

func (s *Server) discCreateProject(c *gin.Context) {
	var data struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	id, err := s.discipline.CreateProject(data.Name, data.Description, data.Color)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id})
}

func (s *Server) discUpdateProject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var patch discipline.ProjectPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.discipline.UpdateProject(id, patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) discDeleteProject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := s.discipline.DeleteProject(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Goals ────────────────────────────────────────────────────────────────────

func (s *Server) discGoals(c *gin.Context) {
	goals, err := s.discipline.ListGoals()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, goals)
}

func (s *Server) discCreateGoal(c *gin.Context) {
	var data struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Category    string  `json:"category"`
		TargetDate  string  `json:"target_date"`
		Target      int     `json:"target"`
		ProjectIDs  []int64 `json:"project_ids"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	id, err := s.discipline.CreateGoal(data.Name, data.Description, data.Category, data.TargetDate, data.Target, data.ProjectIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id})
}

func (s *Server) discUpdateGoal(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var patch discipline.GoalPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.discipline.UpdateGoal(id, patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) discDeleteGoal(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := s.discipline.DeleteGoal(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) discLinkGoal(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var data struct {
		ProjectID int64 `json:"project_id"`
		Link      bool  `json:"link"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if err := s.discipline.LinkGoalProject(id, data.ProjectID, data.Link); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Habits ───────────────────────────────────────────────────────────────────

func (s *Server) discHabits(c *gin.Context) {
	habits, err := s.discipline.ListHabits()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, habits)
}

func (s *Server) discCreateHabit(c *gin.Context) {
	var data struct {
		Name      string `json:"name"`
		Frequency string `json:"frequency"`
		Color     string `json:"color"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	id, err := s.discipline.CreateHabit(data.Name, data.Frequency, data.Color)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id})
}

func (s *Server) discDeleteHabit(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := s.discipline.DeleteHabit(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) discCheckin(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var data struct {
		Day string `json:"day"`
	}
	c.ShouldBindJSON(&data)
	done, err := s.discipline.ToggleCheckin(id, data.Day)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "done": done})
}

// ─── Reminders ────────────────────────────────────────────────────────────────

func (s *Server) discReminders(c *gin.Context) {
	rem, err := s.discipline.ListReminders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rem)
}

func (s *Server) discCreateReminder(c *gin.Context) {
	var data struct {
		Text     string `json:"text"`
		RemindAt string `json:"remind_at"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	id, err := s.discipline.CreateReminder(data.Text, data.RemindAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id})
}

func (s *Server) discSetReminderDone(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var data struct {
		Done bool `json:"done"`
	}
	c.ShouldBindJSON(&data)
	if err := s.discipline.SetReminderDone(id, data.Done); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) discDeleteReminder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := s.discipline.DeleteReminder(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

func (s *Server) discTasks(c *gin.Context) {
	// ?all=1 returns every task; otherwise filter by ?project=ID (0 = no project).
	all := c.Query("all") == "1"
	tasks, err := s.discipline.ListTasks(queryInt64(c, "project"), all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (s *Server) discCreateTask(c *gin.Context) {
	var data discipline.TaskInput
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	id, err := s.discipline.CreateTask(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": id})
}

func (s *Server) discUpdateTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var patch discipline.TaskPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	task, completed, err := s.discipline.UpdateTask(id, patch)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	xp := 0
	if completed {
		xp = discPriorityXP(task.Priority)
		s.filter.AwardPoints(xp, "task_completed")
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "task": task, "xp": xp})
}

func (s *Server) discDeleteTask(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := s.discipline.DeleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ─── Activity ─────────────────────────────────────────────────────────────────

func (s *Server) discHeatmap(c *gin.Context) {
	days := 364
	if v := c.Query("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 366 {
			days = n
		}
	}
	data, err := s.discipline.Heatmap(queryInt64(c, "project"), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) discStats(c *gin.Context) {
	stats, err := s.discipline.Stats(queryInt64(c, "project"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (s *Server) discSeries(c *gin.Context) {
	typ := c.DefaultQuery("type", "task_completed")
	days := 21
	if v := c.Query("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 366 {
			days = n
		}
	}
	data, err := s.discipline.SeriesByType(queryInt64(c, "project"), typ, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) discRecent(c *gin.Context) {
	limit := 15
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	events, err := s.discipline.RecentEvents(queryInt64(c, "project"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

func (s *Server) discDay(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date required"})
		return
	}
	events, err := s.discipline.DayEvents(date, queryInt64(c, "project"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}
