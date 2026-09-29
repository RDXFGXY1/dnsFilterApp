package discipline

import (
	"fmt"
	"strings"
)

type Milestone struct {
	ID           int64  `json:"id"`
	ProjectID    int64  `json:"project_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	StartDate    string `json:"start_date"`
	TargetDate   string `json:"target_date"`
	Position     int    `json:"position"`
	CreatedAt    string `json:"created_at"`
	Total        int    `json:"total"`
	Done         int    `json:"done"`
	Progress     int    `json:"progress"`
	EstimateMin  int    `json:"estimate_minutes"`
	RemainingMin int    `json:"remaining_minutes"`
	Blockers     int    `json:"blockers"`
}

type MilestonePatch struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	StartDate   *string `json:"start_date"`
	TargetDate  *string `json:"target_date"`
	Position    *int    `json:"position"`
}

func (m *Manager) ListMilestones(projectID int64) ([]Milestone, error) {
	rows, err := m.db.Query(`SELECT id, project_id, name, description, status, start_date, target_date, position, created_at FROM discipline_milestones WHERE project_id=? ORDER BY position,id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Milestone{}
	for rows.Next() {
		var x Milestone
		if err := rows.Scan(&x.ID, &x.ProjectID, &x.Name, &x.Description, &x.Status, &x.StartDate, &x.TargetDate, &x.Position, &x.CreatedAt); err != nil {
			return nil, err
		}
		m.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(done),0),COALESCE(SUM(estimate_minutes),0),COALESCE(SUM(CASE WHEN done=0 THEN estimate_minutes ELSE 0 END),0),COALESCE(SUM(CASE WHEN blocked_reason<>'' THEN 1 ELSE 0 END),0) FROM discipline_tasks WHERE milestone_id=? AND archived=0`, x.ID).Scan(&x.Total, &x.Done, &x.EstimateMin, &x.RemainingMin, &x.Blockers)
		if x.Total > 0 {
			x.Progress = x.Done * 100 / x.Total
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (m *Manager) CreateMilestone(projectID int64, name, description, start, target string) (int64, error) {
	if strings.TrimSpace(name) == "" {
		return 0, fmt.Errorf("name required")
	}
	var pos int
	m.db.QueryRow(`SELECT COALESCE(MAX(position),0)+1 FROM discipline_milestones WHERE project_id=?`, projectID).Scan(&pos)
	r, err := m.db.Exec(`INSERT INTO discipline_milestones(project_id,name,description,start_date,target_date,position) VALUES(?,?,?,?,?,?)`, projectID, strings.TrimSpace(name), description, start, target, pos)
	if err != nil {
		return 0, err
	}
	id, _ := r.LastInsertId()
	m.LogEvent("milestone_created", projectID, 0)
	return id, nil
}

func (m *Manager) UpdateMilestone(id int64, p MilestonePatch) error {
	set := []string{}
	args := []interface{}{}
	add := func(k string, v interface{}) { set = append(set, k+"=?"); args = append(args, v) }
	if p.Name != nil {
		if strings.TrimSpace(*p.Name) == "" {
			return fmt.Errorf("name required")
		}
		add("name", strings.TrimSpace(*p.Name))
	}
	if p.Description != nil {
		add("description", *p.Description)
	}
	if p.Status != nil {
		add("status", *p.Status)
	}
	if p.StartDate != nil {
		add("start_date", *p.StartDate)
	}
	if p.TargetDate != nil {
		add("target_date", *p.TargetDate)
	}
	if p.Position != nil {
		add("position", *p.Position)
	}
	if len(set) == 0 {
		return nil
	}
	args = append(args, id)
	_, err := m.db.Exec(`UPDATE discipline_milestones SET `+strings.Join(set, ",")+` WHERE id=?`, args...)
	return err
}
func (m *Manager) DeleteMilestone(id int64) error {
	m.db.Exec(`UPDATE discipline_tasks SET milestone_id=0 WHERE milestone_id=?`, id)
	_, err := m.db.Exec(`DELETE FROM discipline_milestones WHERE id=?`, id)
	return err
}

type Cycle struct {
	ID        int64  `json:"id"`
	ProjectID int64  `json:"project_id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	TaskCount int    `json:"task_count"`
}

func (m *Manager) ListCycles(projectID int64) ([]Cycle, error) {
	rows, err := m.db.Query(`SELECT id,project_id,name,start_date,end_date,status,created_at FROM discipline_cycles WHERE project_id=? ORDER BY start_date DESC,id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Cycle{}
	for rows.Next() {
		var c Cycle
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.StartDate, &c.EndDate, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		m.db.QueryRow(`SELECT COUNT(*) FROM discipline_task_cycles WHERE cycle_id=?`, c.ID).Scan(&c.TaskCount)
		out = append(out, c)
	}
	return out, rows.Err()
}
func (m *Manager) CreateCycle(projectID int64, name, start, end string) (int64, error) {
	if strings.TrimSpace(name) == "" {
		return 0, fmt.Errorf("name required")
	}
	r, e := m.db.Exec(`INSERT INTO discipline_cycles(project_id,name,start_date,end_date) VALUES(?,?,?,?)`, projectID, name, start, end)
	if e != nil {
		return 0, e
	}
	id, _ := r.LastInsertId()
	return id, nil
}

type CyclePatch struct {
	Name      *string `json:"name"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Status    *string `json:"status"`
}

func (m *Manager) UpdateCycle(id int64, p CyclePatch) error {
	set, args := []string{}, []interface{}{}
	add := func(k string, v interface{}) { set = append(set, k+"=?"); args = append(args, v) }
	if p.Name != nil {
		if strings.TrimSpace(*p.Name) == "" {
			return fmt.Errorf("name required")
		}
		add("name", strings.TrimSpace(*p.Name))
	}
	if p.StartDate != nil {
		add("start_date", *p.StartDate)
	}
	if p.EndDate != nil {
		add("end_date", *p.EndDate)
	}
	if p.Status != nil {
		add("status", *p.Status)
	}
	if len(set) == 0 {
		return nil
	}
	args = append(args, id)
	_, e := m.db.Exec(`UPDATE discipline_cycles SET `+strings.Join(set, ",")+` WHERE id=?`, args...)
	return e
}

func (m *Manager) DeleteCycle(id int64) error {
	m.db.Exec(`DELETE FROM discipline_task_cycles WHERE cycle_id=?`, id)
	_, e := m.db.Exec(`DELETE FROM discipline_cycles WHERE id=?`, id)
	return e
}

type CycleAssignment struct {
	TaskID  int64 `json:"task_id"`
	CycleID int64 `json:"cycle_id"`
}

func (m *Manager) CycleAssignments(projectID int64) ([]CycleAssignment, error) {
	rows, e := m.db.Query(`SELECT tc.task_id,tc.cycle_id FROM discipline_task_cycles tc JOIN discipline_tasks t ON t.id=tc.task_id WHERE t.project_id=?`, projectID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []CycleAssignment{}
	for rows.Next() {
		var x CycleAssignment
		if e := rows.Scan(&x.TaskID, &x.CycleID); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (m *Manager) AssignTaskCycle(taskID, cycleID int64) error {
	m.db.Exec(`DELETE FROM discipline_task_cycles WHERE task_id=?`, taskID)
	if cycleID == 0 {
		return nil
	}
	_, e := m.db.Exec(`INSERT INTO discipline_task_cycles(task_id,cycle_id) VALUES(?,?)`, taskID, cycleID)
	return e
}

type RelatedTask struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

func (m *Manager) RelatedTasks(taskID int64) ([]RelatedTask, error) {
	rows, e := m.db.Query(`SELECT t.id,t.title,t.status FROM discipline_task_relations r JOIN discipline_tasks t ON t.id=r.related_task_id WHERE r.task_id=? ORDER BY t.title`, taskID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []RelatedTask{}
	for rows.Next() {
		var x RelatedTask
		if e := rows.Scan(&x.ID, &x.Title, &x.Status); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (m *Manager) SetRelatedTask(taskID, relatedID int64, link bool) error {
	if taskID == relatedID {
		return fmt.Errorf("task cannot relate to itself")
	}
	if link {
		tx, e := m.db.Begin()
		if e != nil {
			return e
		}
		defer tx.Rollback()
		if _, e = tx.Exec(`INSERT OR IGNORE INTO discipline_task_relations(task_id,related_task_id) VALUES(?,?),(?,?)`, taskID, relatedID, relatedID, taskID); e != nil {
			return e
		}
		return tx.Commit()
	}
	_, e := m.db.Exec(`DELETE FROM discipline_task_relations WHERE (task_id=? AND related_task_id=?) OR (task_id=? AND related_task_id=?)`, taskID, relatedID, relatedID, taskID)
	return e
}

type ChecklistItem struct {
	ID       int64  `json:"id"`
	TaskID   int64  `json:"task_id"`
	Text     string `json:"text"`
	Done     bool   `json:"done"`
	Position int    `json:"position"`
}

func (m *Manager) Checklist(taskID int64) ([]ChecklistItem, error) {
	rows, e := m.db.Query(`SELECT id,task_id,text,done,position FROM discipline_checklist WHERE task_id=? ORDER BY position,id`, taskID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ChecklistItem{}
	for rows.Next() {
		var x ChecklistItem
		var d int
		if e := rows.Scan(&x.ID, &x.TaskID, &x.Text, &d, &x.Position); e != nil {
			return nil, e
		}
		x.Done = d == 1
		out = append(out, x)
	}
	return out, rows.Err()
}
func (m *Manager) AddChecklist(taskID int64, text string) (int64, error) {
	if strings.TrimSpace(text) == "" {
		return 0, fmt.Errorf("text required")
	}
	var p int
	m.db.QueryRow(`SELECT COALESCE(MAX(position),0)+1 FROM discipline_checklist WHERE task_id=?`, taskID).Scan(&p)
	r, e := m.db.Exec(`INSERT INTO discipline_checklist(task_id,text,position) VALUES(?,?,?)`, taskID, text, p)
	if e != nil {
		return 0, e
	}
	id, _ := r.LastInsertId()
	return id, nil
}
func (m *Manager) ToggleChecklist(id int64, done bool) error {
	d := 0
	if done {
		d = 1
	}
	_, e := m.db.Exec(`UPDATE discipline_checklist SET done=? WHERE id=?`, d, id)
	return e
}
func (m *Manager) DeleteChecklist(id int64) error {
	_, e := m.db.Exec(`DELETE FROM discipline_checklist WHERE id=?`, id)
	return e
}

type Dependency struct {
	TaskID         int64  `json:"task_id"`
	DependsOnID    int64  `json:"depends_on_id"`
	TaskTitle      string `json:"task_title"`
	DependsOnTitle string `json:"depends_on_title"`
	Resolved       bool   `json:"resolved"`
}

func (m *Manager) ProjectDependencies(projectID int64) ([]Dependency, error) {
	rows, e := m.db.Query(`SELECT d.task_id,d.depends_on_id,COALESCE(t.title,''),COALESCE(b.title,''),COALESCE(b.done,0) FROM discipline_task_dependencies d JOIN discipline_tasks t ON t.id=d.task_id JOIN discipline_tasks b ON b.id=d.depends_on_id WHERE t.project_id=?`, projectID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Dependency{}
	for rows.Next() {
		var x Dependency
		var done int
		if e := rows.Scan(&x.TaskID, &x.DependsOnID, &x.TaskTitle, &x.DependsOnTitle, &done); e != nil {
			return nil, e
		}
		x.Resolved = done == 1
		out = append(out, x)
	}
	return out, rows.Err()
}
func (m *Manager) SetDependency(taskID, dependsOn int64, link bool) error {
	if taskID == dependsOn {
		return fmt.Errorf("task cannot depend on itself")
	}
	if link {
		var sameProject int
		m.db.QueryRow(`SELECT COUNT(*) FROM discipline_tasks a JOIN discipline_tasks b ON a.project_id=b.project_id WHERE a.id=? AND b.id=?`, taskID, dependsOn).Scan(&sameProject)
		if sameProject == 0 {
			return fmt.Errorf("dependencies must belong to the same project")
		}
		var createsCycle int
		m.db.QueryRow(`WITH RECURSIVE chain(id) AS (SELECT depends_on_id FROM discipline_task_dependencies WHERE task_id=? UNION SELECT d.depends_on_id FROM discipline_task_dependencies d JOIN chain c ON d.task_id=c.id) SELECT COUNT(*) FROM chain WHERE id=?`, dependsOn, taskID).Scan(&createsCycle)
		if createsCycle > 0 {
			return fmt.Errorf("dependency would create a cycle")
		}
		_, e := m.db.Exec(`INSERT OR IGNORE INTO discipline_task_dependencies(task_id,depends_on_id) VALUES(?,?)`, taskID, dependsOn)
		return e
	}
	_, e := m.db.Exec(`DELETE FROM discipline_task_dependencies WHERE task_id=? AND depends_on_id=?`, taskID, dependsOn)
	return e
}

func (m *Manager) DuplicateTask(id int64) (int64, error) {
	t, e := m.getTask(id)
	if e != nil {
		return 0, e
	}
	t.Title += " (copy)"
	newID, e := m.CreateTask(TaskInput{ProjectID: t.ProjectID, ParentID: t.ParentID, Title: t.Title, Notes: t.Notes, Priority: t.Priority, Labels: t.Labels, DueDate: t.DueDate, StartDate: t.StartDate, EstimateMin: t.EstimateMin, Energy: t.Energy, MilestoneID: t.MilestoneID, BlockedReason: t.BlockedReason, Recurrence: t.Recurrence, ReminderAt: t.ReminderAt, Links: t.Links, Backlog: t.Backlog})
	if e != nil {
		return 0, e
	}
	m.db.Exec(`INSERT INTO discipline_checklist(task_id,text,done,position) SELECT ?,text,0,position FROM discipline_checklist WHERE task_id=?`, newID, id)
	m.db.Exec(`INSERT INTO discipline_task_dependencies(task_id,depends_on_id) SELECT ?,depends_on_id FROM discipline_task_dependencies WHERE task_id=?`, newID, id)
	m.db.Exec(`INSERT INTO discipline_task_cycles(task_id,cycle_id) SELECT ?,cycle_id FROM discipline_task_cycles WHERE task_id=?`, newID, id)
	return newID, nil
}
func (m *Manager) TaskEvents(taskID int64) ([]Event, error) {
	return m.queryEvents(`SELECT e.type,e.project_id,e.task_id,COALESCE(t.title,''),e.created_at FROM discipline_events e LEFT JOIN discipline_tasks t ON t.id=e.task_id WHERE e.task_id=? ORDER BY e.id DESC LIMIT 60`, taskID)
}
