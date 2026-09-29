package api

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type projectTransferChecklist struct {
	Text string `yaml:"text" json:"text"`
	Done bool   `yaml:"done,omitempty" json:"done"`
}
type projectTransferTask struct {
	Key             string                     `yaml:"key" json:"key"`
	Title           string                     `yaml:"title" json:"title"`
	Notes           string                     `yaml:"notes,omitempty" json:"notes"`
	Status          string                     `yaml:"status,omitempty" json:"status"`
	Priority        string                     `yaml:"priority,omitempty" json:"priority"`
	Size            string                     `yaml:"size,omitempty" json:"size"`
	Energy          string                     `yaml:"energy,omitempty" json:"energy"`
	Labels          []string                   `yaml:"labels,omitempty" json:"labels"`
	StartDate       string                     `yaml:"start_date,omitempty" json:"start_date"`
	DueDate         string                     `yaml:"due_date,omitempty" json:"due_date"`
	EstimateMinutes int                        `yaml:"estimate_minutes,omitempty" json:"estimate_minutes"`
	ActualMinutes   int                        `yaml:"actual_minutes,omitempty" json:"actual_minutes"`
	Milestone       string                     `yaml:"milestone,omitempty" json:"milestone"`
	Cycle           string                     `yaml:"cycle,omitempty" json:"cycle"`
	BlockedReason   string                     `yaml:"blocked_reason,omitempty" json:"blocked_reason"`
	Recurrence      string                     `yaml:"recurrence,omitempty" json:"recurrence"`
	ReminderAt      string                     `yaml:"reminder_at,omitempty" json:"reminder_at"`
	Links           []string                   `yaml:"links,omitempty" json:"links"`
	Backlog         bool                       `yaml:"backlog,omitempty" json:"backlog"`
	Archived        bool                       `yaml:"archived,omitempty" json:"archived"`
	DependsOn       []string                   `yaml:"depends_on,omitempty" json:"depends_on"`
	Checklist       []projectTransferChecklist `yaml:"checklist,omitempty" json:"checklist"`
	Subtasks        []projectTransferTask      `yaml:"subtasks,omitempty" json:"subtasks"`
}
type projectTransferMilestone struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description"`
	Status      string `yaml:"status,omitempty" json:"status"`
	StartDate   string `yaml:"start_date,omitempty" json:"start_date"`
	TargetDate  string `yaml:"target_date,omitempty" json:"target_date"`
}
type projectTransferCycle struct {
	Name      string `yaml:"name" json:"name"`
	Status    string `yaml:"status,omitempty" json:"status"`
	StartDate string `yaml:"start_date,omitempty" json:"start_date"`
	EndDate   string `yaml:"end_date,omitempty" json:"end_date"`
}
type projectTransferProject struct {
	Name        string                     `yaml:"name" json:"name"`
	Description string                     `yaml:"description,omitempty" json:"description"`
	Objective   string                     `yaml:"objective,omitempty" json:"objective"`
	Status      string                     `yaml:"status,omitempty" json:"status"`
	Color       string                     `yaml:"color,omitempty" json:"color"`
	StartDate   string                     `yaml:"start_date,omitempty" json:"start_date"`
	TargetDate  string                     `yaml:"target_date,omitempty" json:"target_date"`
	Notes       string                     `yaml:"notes,omitempty" json:"notes"`
	Links       []string                   `yaml:"links,omitempty" json:"links"`
	Milestones  []projectTransferMilestone `yaml:"milestones,omitempty" json:"milestones"`
	Cycles      []projectTransferCycle     `yaml:"cycles,omitempty" json:"cycles"`
	Tasks       []projectTransferTask      `yaml:"tasks,omitempty" json:"tasks"`
}
type projectTransferDocument struct {
	Version  int                      `yaml:"version" json:"version"`
	Projects []projectTransferProject `yaml:"projects" json:"projects"`
}

func (s *Server) registerProjectTransferRoutes(d *gin.RouterGroup) {
	d.POST("/projects/validate-yaml", s.validateProjectsYAML)
	d.POST("/projects/import-yaml", s.importProjectsYAML)
	d.GET("/projects/export-yaml", s.exportProjectsYAML)
}

func decodeProjectsYAML(content string) (projectTransferDocument, error) {
	var doc projectTransferDocument
	dec := yaml.NewDecoder(bytes.NewBufferString(content))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return doc, fmt.Errorf("YAML could not be read: %w", err)
	}
	if doc.Version != 1 {
		return doc, fmt.Errorf("version must be 1")
	}
	if len(doc.Projects) == 0 {
		return doc, fmt.Errorf("the file does not contain any projects")
	}
	if len(doc.Projects) > 50 {
		return doc, fmt.Errorf("one import can contain at most 50 projects")
	}
	total := 0
	for pi := range doc.Projects {
		p := &doc.Projects[pi]
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			return doc, fmt.Errorf("project %d is missing a name", pi+1)
		}
		if p.Status == "" {
			p.Status = "planning"
		}
		if !oneOf(p.Status, "planning", "active", "paused", "at_risk", "completed") {
			return doc, fmt.Errorf("project %q has invalid status %q", p.Name, p.Status)
		}
		if p.Color == "" {
			p.Color = "#d1553b"
		}
		milestones := map[string]bool{}
		for i := range p.Milestones {
			m := &p.Milestones[i]
			m.Name = strings.TrimSpace(m.Name)
			if m.Name == "" {
				return doc, fmt.Errorf("project %q has a milestone without a name", p.Name)
			}
			if milestones[strings.ToLower(m.Name)] {
				return doc, fmt.Errorf("project %q repeats milestone %q", p.Name, m.Name)
			}
			milestones[strings.ToLower(m.Name)] = true
			if m.Status == "" {
				m.Status = "planned"
			}
		}
		cycles := map[string]bool{}
		for i := range p.Cycles {
			x := &p.Cycles[i]
			x.Name = strings.TrimSpace(x.Name)
			if x.Name == "" {
				return doc, fmt.Errorf("project %q has a cycle without a name", p.Name)
			}
			cycles[strings.ToLower(x.Name)] = true
			if x.Status == "" {
				x.Status = "planned"
			}
		}
		keys := map[string]bool{}
		var checkTask func(*projectTransferTask, string) error
		checkTask = func(t *projectTransferTask, path string) error {
			total++
			if total > 2000 {
				return fmt.Errorf("one import can contain at most 2,000 tasks")
			}
			t.Key = strings.TrimSpace(t.Key)
			t.Title = strings.TrimSpace(t.Title)
			if t.Title == "" {
				return fmt.Errorf("%s is missing a title", path)
			}
			if t.Key == "" {
				t.Key = nextPortableTaskKey(t.Title, keys)
			}
			if keys[t.Key] {
				return fmt.Errorf("project %q repeats task key %q", p.Name, t.Key)
			}
			keys[t.Key] = true
			if t.Status == "" {
				t.Status = "todo"
			}
			if !oneOf(t.Status, "todo", "in_progress", "done") {
				return fmt.Errorf("task %q has invalid status %q", t.Key, t.Status)
			}
			if t.Priority == "" {
				t.Priority = "medium"
			}
			if !oneOf(t.Priority, "low", "medium", "high") {
				return fmt.Errorf("task %q has invalid priority %q", t.Key, t.Priority)
			}
			if t.Energy == "" {
				t.Energy = "medium"
			}
			if t.Milestone != "" && !milestones[strings.ToLower(t.Milestone)] {
				return fmt.Errorf("task %q references unknown milestone %q", t.Key, t.Milestone)
			}
			if t.Cycle != "" && !cycles[strings.ToLower(t.Cycle)] {
				return fmt.Errorf("task %q references unknown cycle %q", t.Key, t.Cycle)
			}
			for i := range t.Subtasks {
				if len(t.Subtasks[i].Subtasks) > 0 {
					return fmt.Errorf("subtask %q cannot contain another subtask level", t.Subtasks[i].Title)
				}
				if err := checkTask(&t.Subtasks[i], "subtask under "+t.Key); err != nil {
					return err
				}
			}
			return nil
		}
		for i := range p.Tasks {
			if err := checkTask(&p.Tasks[i], fmt.Sprintf("task %d", i+1)); err != nil {
				return doc, err
			}
		}
		var checkDeps func(projectTransferTask) error
		checkDeps = func(t projectTransferTask) error {
			for _, dep := range t.DependsOn {
				if !keys[dep] {
					return fmt.Errorf("task %q depends on unknown task key %q", t.Key, dep)
				}
				if dep == t.Key {
					return fmt.Errorf("task %q cannot depend on itself", t.Key)
				}
			}
			for _, sub := range t.Subtasks {
				if err := checkDeps(sub); err != nil {
					return err
				}
			}
			return nil
		}
		for _, t := range p.Tasks {
			if err := checkDeps(t); err != nil {
				return doc, err
			}
		}
	}
	return doc, nil
}
func oneOf(value string, options ...string) bool {
	for _, x := range options {
		if value == x {
			return true
		}
	}
	return false
}

func nextPortableTaskKey(title string, used map[string]bool) string {
	base := safeFilename(title)
	if base == "project" || base == "" {
		base = "task"
	}
	key := base
	for n := 2; used[key]; n++ {
		key = fmt.Sprintf("%s-%d", base, n)
	}
	return key
}

func (s *Server) validateProjectsYAML(c *gin.Context) {
	var b struct {
		YAML string `json:"yaml"`
	}
	if c.ShouldBindJSON(&b) != nil || strings.TrimSpace(b.YAML) == "" {
		c.JSON(400, gin.H{"error": "YAML content is required"})
		return
	}
	doc, e := decodeProjectsYAML(b.YAML)
	if e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	tasks := 0
	milestones := 0
	for _, p := range doc.Projects {
		milestones += len(p.Milestones)
		for _, t := range p.Tasks {
			tasks += 1 + len(t.Subtasks)
		}
	}
	c.JSON(200, gin.H{"valid": true, "document": doc, "projects": len(doc.Projects), "tasks": tasks, "milestones": milestones})
}

func deleteProjectTx(tx *sql.Tx, id int64) error {
	queries := []string{`DELETE FROM discipline_checklist WHERE task_id IN(SELECT id FROM discipline_tasks WHERE project_id=?)`, `DELETE FROM discipline_task_dependencies WHERE task_id IN(SELECT id FROM discipline_tasks WHERE project_id=?) OR depends_on_id IN(SELECT id FROM discipline_tasks WHERE project_id=?)`, `DELETE FROM discipline_task_cycles WHERE task_id IN(SELECT id FROM discipline_tasks WHERE project_id=?)`, `DELETE FROM discipline_task_relations WHERE task_id IN(SELECT id FROM discipline_tasks WHERE project_id=?) OR related_task_id IN(SELECT id FROM discipline_tasks WHERE project_id=?)`, `DELETE FROM discipline_events WHERE project_id=?`, `DELETE FROM discipline_goal_projects WHERE project_id=?`, `DELETE FROM discipline_tasks WHERE project_id=?`, `DELETE FROM discipline_milestones WHERE project_id=?`, `DELETE FROM discipline_cycles WHERE project_id=?`, `DELETE FROM discipline_projects WHERE id=?`}
	for _, q := range queries {
		args := []any{id}
		if strings.Count(q, "?") == 2 {
			args = append(args, id)
		}
		if _, e := tx.Exec(q, args...); e != nil {
			return e
		}
	}
	return nil
}

func (s *Server) importProjectsYAML(c *gin.Context) {
	var b struct {
		YAML string `json:"yaml"`
		Mode string `json:"mode"`
	}
	if c.ShouldBindJSON(&b) != nil || strings.TrimSpace(b.YAML) == "" {
		c.JSON(400, gin.H{"error": "YAML content is required"})
		return
	}
	if !oneOf(b.Mode, "add", "replace", "skip", "merge") {
		c.JSON(400, gin.H{"error": "mode must be add, replace, skip, or merge"})
		return
	}
	doc, e := decodeProjectsYAML(b.YAML)
	if e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	tx, e := s.db.GetDB().Begin()
	if e != nil {
		respondErr(c, e)
		return
	}
	defer tx.Rollback()
	imported, merged, skipped, tasks := 0, 0, 0, 0
	for _, p := range doc.Projects {
		var existing int64
		e = tx.QueryRow(`SELECT id FROM discipline_projects WHERE lower(name)=lower(?) LIMIT 1`, p.Name).Scan(&existing)
		if e != nil && e != sql.ErrNoRows {
			respondErr(c, e)
			return
		}
		if existing > 0 && b.Mode == "skip" {
			skipped++
			continue
		}
		if existing > 0 && b.Mode == "replace" {
			if e = deleteProjectTx(tx, existing); e != nil {
				respondErr(c, e)
				return
			}
			existing = 0
		}
		projectID := existing
		var r sql.Result
		if existing > 0 && b.Mode == "merge" {
			merged++
		} else {
			r, e = tx.Exec(`INSERT INTO discipline_projects(name,description,objective,status,color,start_date,target_date,notes,links) VALUES(?,?,?,?,?,?,?,?,?)`, p.Name, p.Description, p.Objective, p.Status, p.Color, p.StartDate, p.TargetDate, p.Notes, strings.Join(p.Links, "\n"))
			if e != nil {
				respondErr(c, e)
				return
			}
			projectID, _ = r.LastInsertId()
			imported++
		}
		milestoneIDs := map[string]int64{}
		if existing > 0 && b.Mode == "merge" {
			rows, queryErr := tx.Query(`SELECT id,name FROM discipline_milestones WHERE project_id=?`, projectID)
			if queryErr != nil {
				respondErr(c, queryErr)
				return
			}
			for rows.Next() {
				var id int64
				var name string
				_ = rows.Scan(&id, &name)
				milestoneIDs[strings.ToLower(name)] = id
			}
			rows.Close()
		}
		for pos, m := range p.Milestones {
			if milestoneIDs[strings.ToLower(m.Name)] > 0 {
				continue
			}
			r, e = tx.Exec(`INSERT INTO discipline_milestones(project_id,name,description,status,start_date,target_date,position) VALUES(?,?,?,?,?,?,?)`, projectID, m.Name, m.Description, m.Status, m.StartDate, m.TargetDate, pos)
			if e != nil {
				respondErr(c, e)
				return
			}
			id, _ := r.LastInsertId()
			milestoneIDs[strings.ToLower(m.Name)] = id
		}
		cycleIDs := map[string]int64{}
		if existing > 0 && b.Mode == "merge" {
			rows, queryErr := tx.Query(`SELECT id,name FROM discipline_cycles WHERE project_id=?`, projectID)
			if queryErr != nil {
				respondErr(c, queryErr)
				return
			}
			for rows.Next() {
				var id int64
				var name string
				_ = rows.Scan(&id, &name)
				cycleIDs[strings.ToLower(name)] = id
			}
			rows.Close()
		}
		for _, x := range p.Cycles {
			if cycleIDs[strings.ToLower(x.Name)] > 0 {
				continue
			}
			r, e = tx.Exec(`INSERT INTO discipline_cycles(project_id,name,start_date,end_date,status) VALUES(?,?,?,?,?)`, projectID, x.Name, x.StartDate, x.EndDate, x.Status)
			if e != nil {
				respondErr(c, e)
				return
			}
			id, _ := r.LastInsertId()
			cycleIDs[strings.ToLower(x.Name)] = id
		}
		taskIDs := map[string]int64{}
		allTasks := []projectTransferTask{}
		var insertTask func(projectTransferTask, int64, int) error
		insertTask = func(t projectTransferTask, parent int64, pos int) error {
			done := t.Status == "done"
			r, e := tx.Exec(`INSERT INTO discipline_tasks(project_id,parent_id,title,notes,status,priority,size,labels,due_date,done,completed_at,start_date,estimate_minutes,actual_minutes,energy,milestone_id,blocked_reason,recurrence,reminder_at,links,backlog,archived,position) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, projectID, parent, t.Title, t.Notes, t.Status, t.Priority, t.Size, strings.Join(t.Labels, ","), t.DueDate, done, map[bool]string{true: "imported", false: ""}[done], t.StartDate, t.EstimateMinutes, t.ActualMinutes, t.Energy, milestoneIDs[strings.ToLower(t.Milestone)], t.BlockedReason, t.Recurrence, t.ReminderAt, strings.Join(t.Links, "\n"), t.Backlog, t.Archived, pos)
			if e != nil {
				return e
			}
			id, _ := r.LastInsertId()
			taskIDs[t.Key] = id
			allTasks = append(allTasks, t)
			for i, ch := range t.Checklist {
				if _, e = tx.Exec(`INSERT INTO discipline_checklist(task_id,text,done,position) VALUES(?,?,?,?)`, id, ch.Text, ch.Done, i); e != nil {
					return e
				}
			}
			for i, sub := range t.Subtasks {
				if e = insertTask(sub, id, i); e != nil {
					return e
				}
			}
			if cycle := cycleIDs[strings.ToLower(t.Cycle)]; cycle > 0 {
				if _, e = tx.Exec(`INSERT OR IGNORE INTO discipline_task_cycles(task_id,cycle_id) VALUES(?,?)`, id, cycle); e != nil {
					return e
				}
			}
			tasks++
			return nil
		}
		for i, t := range p.Tasks {
			if e = insertTask(t, 0, i); e != nil {
				respondErr(c, e)
				return
			}
		}
		var addDeps func(projectTransferTask) error
		addDeps = func(t projectTransferTask) error {
			for _, dep := range t.DependsOn {
				if _, e := tx.Exec(`INSERT OR IGNORE INTO discipline_task_dependencies(task_id,depends_on_id) VALUES(?,?)`, taskIDs[t.Key], taskIDs[dep]); e != nil {
					return e
				}
			}
			for _, sub := range t.Subtasks {
				if e := addDeps(sub); e != nil {
					return e
				}
			}
			return nil
		}
		for _, t := range p.Tasks {
			if e = addDeps(t); e != nil {
				respondErr(c, e)
				return
			}
		}
	}
	if e = tx.Commit(); e != nil {
		respondErr(c, e)
		return
	}
	c.JSON(200, gin.H{"success": true, "projects": imported, "merged": merged, "tasks": tasks, "skipped": skipped})
}

func (s *Server) exportProjectsYAML(c *gin.Context) {
	projectID := queryInt64(c, "project")
	q := `SELECT id,name,description,objective,status,color,start_date,target_date,notes,links FROM discipline_projects`
	args := []any{}
	if projectID > 0 {
		q += ` WHERE id=?`
		args = append(args, projectID)
	} else if raw := strings.TrimSpace(c.Query("projects")); raw != "" {
		ids := []int64{}
		for _, part := range strings.Split(raw, ",") {
			if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && id > 0 {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "select at least one valid project"})
			return
		}
		marks := make([]string, len(ids))
		for i, id := range ids {
			marks[i] = "?"
			args = append(args, id)
		}
		q += ` WHERE id IN (` + strings.Join(marks, ",") + `)`
	}
	q += ` ORDER BY id`
	rows, e := s.db.GetDB().Query(q, args...)
	if e != nil {
		respondErr(c, e)
		return
	}
	defer rows.Close()
	doc := projectTransferDocument{Version: 1, Projects: []projectTransferProject{}}
	for rows.Next() {
		var id int64
		var p projectTransferProject
		var links string
		if e = rows.Scan(&id, &p.Name, &p.Description, &p.Objective, &p.Status, &p.Color, &p.StartDate, &p.TargetDate, &p.Notes, &links); e != nil {
			respondErr(c, e)
			return
		}
		p.Links = splitNonempty(links)
		if p.Color == "#d1553b" {
			p.Color = ""
		}
		p.Milestones = []projectTransferMilestone{}
		mIDs := map[int64]string{}
		mr, _ := s.db.GetDB().Query(`SELECT id,name,description,status,start_date,target_date FROM discipline_milestones WHERE project_id=? ORDER BY position,id`, id)
		for mr.Next() {
			var mid int64
			var m projectTransferMilestone
			_ = mr.Scan(&mid, &m.Name, &m.Description, &m.Status, &m.StartDate, &m.TargetDate)
			mIDs[mid] = m.Name
			if m.Status == "planned" {
				m.Status = ""
			}
			p.Milestones = append(p.Milestones, m)
		}
		mr.Close()
		p.Cycles = []projectTransferCycle{}
		cycleNames := map[int64]string{}
		cr, _ := s.db.GetDB().Query(`SELECT id,name,status,start_date,end_date FROM discipline_cycles WHERE project_id=? ORDER BY id`, id)
		for cr.Next() {
			var cid int64
			var x projectTransferCycle
			_ = cr.Scan(&cid, &x.Name, &x.Status, &x.StartDate, &x.EndDate)
			cycleNames[cid] = x.Name
			if x.Status == "planned" {
				x.Status = ""
			}
			p.Cycles = append(p.Cycles, x)
		}
		cr.Close()
		type taskRow struct {
			id, parent int64
			t          projectTransferTask
		}
		trs := []taskRow{}
		tr, e := s.db.GetDB().Query(`SELECT id,parent_id,title,notes,status,priority,size,labels,start_date,due_date,estimate_minutes,actual_minutes,energy,milestone_id,blocked_reason,recurrence,reminder_at,links,backlog,archived FROM discipline_tasks WHERE project_id=? ORDER BY parent_id,position,id`, id)
		if e != nil {
			respondErr(c, e)
			return
		}
		for tr.Next() {
			var r taskRow
			var labels, taskLinks string
			var milestoneID int64
			if e = tr.Scan(&r.id, &r.parent, &r.t.Title, &r.t.Notes, &r.t.Status, &r.t.Priority, &r.t.Size, &labels, &r.t.StartDate, &r.t.DueDate, &r.t.EstimateMinutes, &r.t.ActualMinutes, &r.t.Energy, &milestoneID, &r.t.BlockedReason, &r.t.Recurrence, &r.t.ReminderAt, &taskLinks, &r.t.Backlog, &r.t.Archived); e != nil {
				tr.Close()
				respondErr(c, e)
				return
			}
			r.t.Labels = splitCSV(labels)
			r.t.Links = splitNonempty(taskLinks)
			r.t.Milestone = mIDs[milestoneID]
			var cycleID int64
			_ = s.db.GetDB().QueryRow(`SELECT cycle_id FROM discipline_task_cycles WHERE task_id=? LIMIT 1`, r.id).Scan(&cycleID)
			r.t.Cycle = cycleNames[cycleID]
			ch, _ := s.discipline.Checklist(r.id)
			for _, x := range ch {
				r.t.Checklist = append(r.t.Checklist, projectTransferChecklist{Text: x.Text, Done: x.Done})
			}
			trs = append(trs, r)
		}
		tr.Close()
		keyByID := map[int64]string{}
		usedKeys := map[string]bool{}
		for i := range trs {
			trs[i].t.Key = nextPortableTaskKey(trs[i].t.Title, usedKeys)
			usedKeys[trs[i].t.Key] = true
			keyByID[trs[i].id] = trs[i].t.Key
		}
		for i := range trs {
			dr, _ := s.db.GetDB().Query(`SELECT depends_on_id FROM discipline_task_dependencies WHERE task_id=?`, trs[i].id)
			for dr.Next() {
				var dep int64
				_ = dr.Scan(&dep)
				if keyByID[dep] != "" {
					trs[i].t.DependsOn = append(trs[i].t.DependsOn, keyByID[dep])
				}
			}
			dr.Close()
		}
		children := map[int64][]projectTransferTask{}
		for _, r := range trs {
			if r.parent > 0 {
				children[r.parent] = append(children[r.parent], r.t)
			}
		}
		for _, r := range trs {
			if r.parent == 0 {
				t := r.t
				t.Subtasks = children[r.id]
				cleanExportTask(&t)
				p.Tasks = append(p.Tasks, t)
			}
		}
		doc.Projects = append(doc.Projects, p)
	}
	data, e := yaml.Marshal(doc)
	if e != nil {
		respondErr(c, e)
		return
	}
	name := "projects.yaml"
	if len(doc.Projects) == 1 {
		name = safeFilename(doc.Projects[0].Name) + ".yaml"
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", data)
}

func cleanExportTask(t *projectTransferTask) {
	if t.Status == "todo" {
		t.Status = ""
	}
	if t.Priority == "medium" {
		t.Priority = ""
	}
	if t.Energy == "medium" {
		t.Energy = ""
	}
	for i := range t.Subtasks {
		cleanExportTask(&t.Subtasks[i])
	}
}
func splitNonempty(v string) []string {
	parts := strings.FieldsFunc(v, func(r rune) bool { return r == '\n' || r == '\r' })
	out := []string{}
	for _, x := range parts {
		if strings.TrimSpace(x) != "" {
			out = append(out, strings.TrimSpace(x))
		}
	}
	return out
}
func splitCSV(v string) []string {
	out := []string{}
	for _, x := range strings.Split(v, ",") {
		if strings.TrimSpace(x) != "" {
			out = append(out, strings.TrimSpace(x))
		}
	}
	return out
}
func safeFilename(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, v)
	v = strings.Trim(v, "-")
	if v == "" {
		return "project"
	}
	return v
}
