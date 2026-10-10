// Package discipline powers the /discipline productivity page: projects, tasks,
// and an event-driven activity system. Every action logs a discipline_event, so
// the heatmap, streaks, and stats are all just queries over one table.
package discipline

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Manager owns the discipline tables and logic.
type Manager struct {
	db       *sql.DB
	focusMu  sync.RWMutex
	focusID  int64
	focusEnd chan struct{}
}

type Goal struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	TargetDate  string  `json:"target_date"`
	Status      string  `json:"status"` // active | done | paused
	Target      int     `json:"target"` // for count-based goals (e.g. read 12 books)
	Current     int     `json:"current"`
	Progress    int     `json:"progress"`    // computed
	ProjectIDs  []int64 `json:"project_ids"` // linked projects
	CreatedAt   string  `json:"created_at"`
}

type Project struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Objective   string `json:"objective"`
	Notes       string `json:"notes"`
	Links       string `json:"links"`
	Color       string `json:"color"`
	Status      string `json:"status"`
	StartDate   string `json:"start_date"`
	TargetDate  string `json:"target_date"`
	CreatedAt   string `json:"created_at"`
	Total       int    `json:"total"`
	Done        int    `json:"done"`
	InProgress  int    `json:"in_progress"`
	Progress    int    `json:"progress"`
}

type Task struct {
	ID             int64  `json:"id"`
	ProjectID      int64  `json:"project_id"` // 0 = no project
	ParentID       int64  `json:"parent_id"`  // 0 = top-level task
	Title          string `json:"title"`
	Notes          string `json:"notes"`
	Status         string `json:"status"`   // todo | in_progress | done
	Priority       string `json:"priority"` // low | medium | high
	Size           string `json:"size"`     // XS | S | M | L | XL | ""
	Labels         string `json:"labels"`
	DueDate        string `json:"due_date"`
	StartDate      string `json:"start_date"`
	EstimateMin    int    `json:"estimate_minutes"`
	ActualMin      int    `json:"actual_minutes"`
	Energy         string `json:"energy"`
	MilestoneID    int64  `json:"milestone_id"`
	BlockedReason  string `json:"blocked_reason"`
	Recurrence     string `json:"recurrence"`
	ReminderAt     string `json:"reminder_at"`
	Links          string `json:"links"`
	Backlog        bool   `json:"backlog"`
	Archived       bool   `json:"archived"`
	Position       int    `json:"position"`
	MyDay          bool   `json:"my_day"`
	ScheduledAt    string `json:"scheduled_at"`
	WaitingFor     string `json:"waiting_for"`
	Evidence       string `json:"evidence"`
	FocusProfileID int64  `json:"focus_profile_id"`
	Done           bool   `json:"done"`
	CreatedAt      string `json:"created_at"`
	CompletedAt    string `json:"completed_at"`
}

type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type Event struct {
	Type      string `json:"type"`
	ProjectID int64  `json:"project_id"`
	TaskID    int64  `json:"task_id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

type Stats struct {
	CurrentStreak     int            `json:"current_streak"`
	LongestStreak     int            `json:"longest_streak"`
	ActiveDays        int            `json:"active_days"`
	TasksCompleted    int            `json:"tasks_completed"`
	SubtasksCompleted int            `json:"subtasks_completed"`
	BugsFixed         int            `json:"bugs_fixed"`
	Total             int            `json:"total"`
	ByType            map[string]int `json:"by_type"`
}

// New creates the manager and ensures tables exist.
func New(db *sql.DB) (*Manager, error) {
	m := &Manager{db: db}
	if err := m.init(); err != nil {
		return nil, err
	}
	m.restoreFocusSession()
	return m, nil
}

func (m *Manager) init() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS discipline_projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			color TEXT DEFAULT '#d1553b',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER DEFAULT 0,
			title TEXT NOT NULL,
			notes TEXT DEFAULT '',
			status TEXT DEFAULT 'todo',
			priority TEXT DEFAULT 'medium',
			labels TEXT DEFAULT '',
			due_date TEXT DEFAULT '',
			done INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TEXT DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			project_id INTEGER DEFAULT 0,
			task_id INTEGER DEFAULT 0,
			day TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_disc_events_day ON discipline_events(day)`,
		`CREATE INDEX IF NOT EXISTS idx_disc_tasks_project ON discipline_tasks(project_id)`,
		`CREATE TABLE IF NOT EXISTS discipline_goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			category TEXT DEFAULT '',
			target_date TEXT DEFAULT '',
			status TEXT DEFAULT 'active',
			target INTEGER DEFAULT 0,
			current INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_goal_projects (
			goal_id INTEGER,
			project_id INTEGER,
			UNIQUE(goal_id, project_id)
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_habits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			frequency TEXT DEFAULT 'daily',
			color TEXT DEFAULT '#d1553b',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_habit_checkins (
			habit_id INTEGER,
			day TEXT,
			UNIQUE(habit_id, day)
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_reminders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			remind_at TEXT NOT NULL,
			done INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_milestones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			status TEXT DEFAULT 'planned',
			start_date TEXT DEFAULT '',
			target_date TEXT DEFAULT '',
			position INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_cycles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			start_date TEXT DEFAULT '',
			end_date TEXT DEFAULT '',
			status TEXT DEFAULT 'planned',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_checklist (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL,
			text TEXT NOT NULL,
			done INTEGER DEFAULT 0,
			position INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_task_dependencies (
			task_id INTEGER NOT NULL,
			depends_on_id INTEGER NOT NULL,
			UNIQUE(task_id, depends_on_id)
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_task_cycles (
			task_id INTEGER NOT NULL,
			cycle_id INTEGER NOT NULL,
			UNIQUE(task_id, cycle_id)
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_task_relations (
			task_id INTEGER NOT NULL,
			related_task_id INTEGER NOT NULL,
			UNIQUE(task_id, related_task_id)
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_task_templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			title TEXT NOT NULL,
			notes TEXT DEFAULT '',
			priority TEXT DEFAULT 'medium',
			labels TEXT DEFAULT '',
			estimate_minutes INTEGER DEFAULT 0,
			energy TEXT DEFAULT 'medium',
			checklist TEXT DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_journal_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL COLLATE NOCASE UNIQUE,
			color TEXT DEFAULT '#4f73d9',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_journal_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type_id INTEGER DEFAULT 0,
			title TEXT NOT NULL,
			content TEXT DEFAULT '',
			day TEXT NOT NULL,
			duration_minutes INTEGER DEFAULT 0,
			mood INTEGER DEFAULT 0,
			focus INTEGER DEFAULT 0,
			tags TEXT DEFAULT '',
			project_id INTEGER DEFAULT 0,
			pinned INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_disc_journal_day ON discipline_journal_entries(day)`,
		`CREATE INDEX IF NOT EXISTS idx_disc_journal_type ON discipline_journal_entries(type_id)`,
		`CREATE TABLE IF NOT EXISTS discipline_focus_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER DEFAULT 0,
			task_title TEXT DEFAULT '',
			planned_minutes INTEGER DEFAULT 25,
			actual_seconds INTEGER DEFAULT 0,
			status TEXT DEFAULT 'active',
			activity_logging INTEGER DEFAULT 0,
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			ended_at TEXT DEFAULT '',
			reflection TEXT DEFAULT '',
			focus_score INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_focus_activity (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			seconds INTEGER DEFAULT 0,
			observations INTEGER DEFAULT 1,
			first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(session_id,kind,name)
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_distractions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL,
			label TEXT NOT NULL,
			note TEXT DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_disc_focus_activity_session ON discipline_focus_activity(session_id)`,
		`CREATE TABLE IF NOT EXISTS discipline_focus_capsules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			allowed_apps TEXT DEFAULT '[]',
			allowed_domains TEXT DEFAULT '[]',
			youtube_channels TEXT DEFAULT '[]',
			block_style TEXT DEFAULT 'motivation',
			strict INTEGER DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS discipline_inbox_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender TEXT NOT NULL DEFAULT 'Discipline Intelligence',
			subject TEXT NOT NULL,
			body TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT 'insight',
			severity TEXT NOT NULL DEFAULT 'normal',
			action_label TEXT NOT NULL DEFAULT '',
			action_view TEXT NOT NULL DEFAULT '',
			is_read INTEGER NOT NULL DEFAULT 0,
			starred INTEGER NOT NULL DEFAULT 0,
			archived INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_disc_inbox_state ON discipline_inbox_messages(archived,is_read,created_at)`,
		`CREATE TABLE IF NOT EXISTS ai_provider_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			name TEXT NOT NULL,
			model TEXT NOT NULL,
			base_url TEXT NOT NULL DEFAULT '',
			api_key_encrypted BLOB NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			is_default INTEGER NOT NULL DEFAULT 0,
			last_status TEXT NOT NULL DEFAULT 'not_tested',
			last_error TEXT NOT NULL DEFAULT '',
			last_tested_at TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, s := range stmts {
		if _, err := m.db.Exec(s); err != nil {
			return err
		}
	}
	// Migrations: add columns if the table predates them (ignore if they exist).
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN size TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN parent_id INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_projects ADD COLUMN objective TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_projects ADD COLUMN status TEXT DEFAULT 'active'`)
	m.db.Exec(`ALTER TABLE discipline_projects ADD COLUMN start_date TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_projects ADD COLUMN target_date TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_projects ADD COLUMN notes TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_projects ADD COLUMN links TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN start_date TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN estimate_minutes INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN actual_minutes INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN energy TEXT DEFAULT 'medium'`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN milestone_id INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN blocked_reason TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN recurrence TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN reminder_at TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN links TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN backlog INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN archived INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN position INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN my_day INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN scheduled_at TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN waiting_for TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN evidence TEXT DEFAULT ''`)
	m.db.Exec(`ALTER TABLE discipline_tasks ADD COLUMN focus_profile_id INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_focus_sessions ADD COLUMN capsule_id INTEGER DEFAULT 0`)
	m.db.Exec(`ALTER TABLE discipline_focus_sessions ADD COLUMN locked_youtube_channel TEXT DEFAULT ''`)
	return nil
}

// LogEvent records an activity event (the backbone of all stats/heatmaps).
func (m *Manager) LogEvent(eventType string, projectID, taskID int64) {
	day := time.Now().Format("2006-01-02")
	m.db.Exec(`INSERT INTO discipline_events (type, project_id, task_id, day) VALUES (?, ?, ?, ?)`,
		eventType, projectID, taskID, day)
}

// ─── Projects ─────────────────────────────────────────────────────────────────

func (m *Manager) ListProjects() ([]Project, error) {
	rows, err := m.db.Query(`SELECT id, name, description, objective, notes, links, color, status, start_date, target_date, created_at FROM discipline_projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Objective, &p.Notes, &p.Links, &p.Color, &p.Status, &p.StartDate, &p.TargetDate, &p.CreatedAt); err != nil {
			return nil, err
		}
		m.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(done),0) FROM discipline_tasks WHERE project_id = ? AND archived = 0 AND backlog = 0`, p.ID).
			Scan(&p.Total, &p.Done)
		m.db.QueryRow(`SELECT COUNT(*) FROM discipline_tasks WHERE project_id = ? AND archived = 0 AND backlog = 0 AND status = 'in_progress'`, p.ID).
			Scan(&p.InProgress)
		if p.Total > 0 {
			p.Progress = p.Done * 100 / p.Total
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ProjectPatch carries editable command-header fields without overwriting
// values that were not present in the request.
type ProjectPatch struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Objective   *string `json:"objective"`
	Status      *string `json:"status"`
	StartDate   *string `json:"start_date"`
	TargetDate  *string `json:"target_date"`
	Notes       *string `json:"notes"`
	Links       *string `json:"links"`
}

func (m *Manager) UpdateProject(id int64, p ProjectPatch) error {
	set := []string{}
	args := []interface{}{}
	add := func(column string, value interface{}) { set = append(set, column+" = ?"); args = append(args, value) }
	if p.Name != nil {
		if strings.TrimSpace(*p.Name) == "" {
			return fmt.Errorf("name required")
		}
		add("name", strings.TrimSpace(*p.Name))
	}
	if p.Description != nil {
		add("description", strings.TrimSpace(*p.Description))
	}
	if p.Objective != nil {
		add("objective", strings.TrimSpace(*p.Objective))
	}
	if p.Status != nil {
		valid := map[string]bool{"planning": true, "active": true, "paused": true, "at_risk": true, "completed": true}
		if !valid[*p.Status] {
			return fmt.Errorf("invalid project status")
		}
		add("status", *p.Status)
	}
	if p.StartDate != nil {
		add("start_date", *p.StartDate)
	}
	if p.TargetDate != nil {
		add("target_date", *p.TargetDate)
	}
	if p.Notes != nil {
		add("notes", strings.TrimSpace(*p.Notes))
	}
	if p.Links != nil {
		add("links", strings.TrimSpace(*p.Links))
	}
	if len(set) == 0 {
		return nil
	}
	args = append(args, id)
	_, err := m.db.Exec(`UPDATE discipline_projects SET `+strings.Join(set, ", ")+` WHERE id = ?`, args...)
	if err == nil {
		m.LogEvent("project_updated", id, 0)
	}
	return err
}

func (m *Manager) CreateProject(name, description, color string) (int64, error) {
	if strings.TrimSpace(name) == "" {
		return 0, fmt.Errorf("name required")
	}
	if color == "" {
		color = "#d1553b"
	}
	res, err := m.db.Exec(`INSERT INTO discipline_projects (name, description, color) VALUES (?, ?, ?)`,
		name, description, color)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	m.LogEvent("project_created", id, 0)
	return id, nil
}

func (m *Manager) DeleteProject(id int64) error {
	m.db.Exec(`DELETE FROM discipline_tasks WHERE project_id = ?`, id)
	_, err := m.db.Exec(`DELETE FROM discipline_projects WHERE id = ?`, id)
	return err
}

// ─── Goals ────────────────────────────────────────────────────────────────────

func (m *Manager) ListGoals() ([]Goal, error) {
	rows, err := m.db.Query(`SELECT id, name, description, category, target_date, status, target, current, created_at
		FROM discipline_goals ORDER BY status='done', created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Goal
	for rows.Next() {
		var g Goal
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.Category, &g.TargetDate, &g.Status, &g.Target, &g.Current, &g.CreatedAt); err != nil {
			return nil, err
		}
		g.ProjectIDs = m.goalProjects(g.ID)
		g.Progress = m.goalProgress(g)
		out = append(out, g)
	}
	return out, rows.Err()
}

func (m *Manager) goalProjects(goalID int64) []int64 {
	rows, err := m.db.Query(`SELECT project_id FROM discipline_goal_projects WHERE goal_id = ?`, goalID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

// goalProgress computes progress from linked projects' task completion, or from
// the count target for count-based goals. Never manually entered.
func (m *Manager) goalProgress(g Goal) int {
	if len(g.ProjectIDs) > 0 {
		ph := make([]string, len(g.ProjectIDs))
		args := make([]interface{}, len(g.ProjectIDs))
		for i, id := range g.ProjectIDs {
			ph[i] = "?"
			args[i] = id
		}
		var done, total int
		m.db.QueryRow(`SELECT COALESCE(SUM(done),0), COUNT(*) FROM discipline_tasks WHERE project_id IN (`+strings.Join(ph, ",")+`)`, args...).
			Scan(&done, &total)
		if total > 0 {
			return done * 100 / total
		}
		return 0
	}
	if g.Target > 0 {
		p := g.Current * 100 / g.Target
		if p > 100 {
			p = 100
		}
		return p
	}
	if g.Status == "done" {
		return 100
	}
	return 0
}

func (m *Manager) CreateGoal(name, description, category, targetDate string, target int, projectIDs []int64) (int64, error) {
	if strings.TrimSpace(name) == "" {
		return 0, fmt.Errorf("name required")
	}
	res, err := m.db.Exec(`INSERT INTO discipline_goals (name, description, category, target_date, target) VALUES (?, ?, ?, ?, ?)`,
		name, description, category, targetDate, target)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	for _, pid := range projectIDs {
		m.db.Exec(`INSERT OR IGNORE INTO discipline_goal_projects (goal_id, project_id) VALUES (?, ?)`, id, pid)
	}
	m.LogEvent("goal_created", 0, 0)
	return id, nil
}

// GoalPatch carries optional goal updates.
type GoalPatch struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Category    *string `json:"category"`
	TargetDate  *string `json:"target_date"`
	Status      *string `json:"status"`
	Target      *int    `json:"target"`
	Current     *int    `json:"current"`
}

func (m *Manager) UpdateGoal(id int64, p GoalPatch) error {
	set := []string{}
	args := []interface{}{}
	add := func(col string, v interface{}) { set = append(set, col+"=?"); args = append(args, v) }
	if p.Name != nil {
		add("name", *p.Name)
	}
	if p.Description != nil {
		add("description", *p.Description)
	}
	if p.Category != nil {
		add("category", *p.Category)
	}
	if p.TargetDate != nil {
		add("target_date", *p.TargetDate)
	}
	if p.Status != nil {
		add("status", *p.Status)
		if *p.Status == "done" {
			m.LogEvent("goal_completed", 0, 0)
		}
	}
	if p.Target != nil {
		add("target", *p.Target)
	}
	if p.Current != nil {
		add("current", *p.Current)
	}
	if len(set) == 0 {
		return nil
	}
	args = append(args, id)
	_, err := m.db.Exec(`UPDATE discipline_goals SET `+strings.Join(set, ", ")+` WHERE id=?`, args...)
	return err
}

func (m *Manager) DeleteGoal(id int64) error {
	m.db.Exec(`DELETE FROM discipline_goal_projects WHERE goal_id = ?`, id)
	_, err := m.db.Exec(`DELETE FROM discipline_goals WHERE id = ?`, id)
	return err
}

func (m *Manager) LinkGoalProject(goalID, projectID int64, link bool) error {
	var err error
	if link {
		_, err = m.db.Exec(`INSERT OR IGNORE INTO discipline_goal_projects (goal_id, project_id) VALUES (?, ?)`, goalID, projectID)
	} else {
		_, err = m.db.Exec(`DELETE FROM discipline_goal_projects WHERE goal_id = ? AND project_id = ?`, goalID, projectID)
	}
	return err
}

// ─── Habits ───────────────────────────────────────────────────────────────────

type Habit struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Frequency     string     `json:"frequency"`
	Color         string     `json:"color"`
	CreatedAt     string     `json:"created_at"`
	DoneToday     bool       `json:"done_today"`
	CurrentStreak int        `json:"current_streak"`
	LongestStreak int        `json:"longest_streak"`
	WeekDone      int        `json:"week_done"`
	Days          []DayCount `json:"days"` // last 84 days check-in heatmap (0/1)
}

func (m *Manager) ListHabits() ([]Habit, error) {
	rows, err := m.db.Query(`SELECT id, name, frequency, color, created_at FROM discipline_habits ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	today := time.Now().Format("2006-01-02")
	// Monday of the current week.
	now := time.Now()
	weekday := int(now.Weekday()) // Sun=0
	offset := (weekday + 6) % 7
	monday := now.AddDate(0, 0, -offset).Format("2006-01-02")

	var out []Habit
	for rows.Next() {
		var h Habit
		if err := rows.Scan(&h.ID, &h.Name, &h.Frequency, &h.Color, &h.CreatedAt); err != nil {
			return nil, err
		}
		crows, _ := m.db.Query(`SELECT day FROM discipline_habit_checkins WHERE habit_id = ? ORDER BY day DESC`, h.ID)
		var days []string
		set := map[string]bool{}
		for crows.Next() {
			var d string
			crows.Scan(&d)
			days = append(days, d)
			set[d] = true
			if d >= monday {
				h.WeekDone++
			}
		}
		crows.Close()
		h.DoneToday = set[today]
		h.CurrentStreak, h.LongestStreak = computeStreaks(days)
		h.Days = make([]DayCount, 0, 84)
		for i := 83; i >= 0; i-- {
			d := now.AddDate(0, 0, -i).Format("2006-01-02")
			c := 0
			if set[d] {
				c = 1
			}
			h.Days = append(h.Days, DayCount{Day: d, Count: c})
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (m *Manager) CreateHabit(name, frequency, color string) (int64, error) {
	if strings.TrimSpace(name) == "" {
		return 0, fmt.Errorf("name required")
	}
	if frequency == "" {
		frequency = "daily"
	}
	if color == "" {
		color = "#d1553b"
	}
	res, err := m.db.Exec(`INSERT INTO discipline_habits (name, frequency, color) VALUES (?, ?, ?)`, name, frequency, color)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (m *Manager) DeleteHabit(id int64) error {
	m.db.Exec(`DELETE FROM discipline_habit_checkins WHERE habit_id = ?`, id)
	_, err := m.db.Exec(`DELETE FROM discipline_habits WHERE id = ?`, id)
	return err
}

// ToggleCheckin toggles a habit check-in for a day (default today). Returns the
// new done state.
func (m *Manager) ToggleCheckin(habitID int64, day string) (bool, error) {
	if day == "" {
		day = time.Now().Format("2006-01-02")
	}
	var cnt int
	m.db.QueryRow(`SELECT COUNT(*) FROM discipline_habit_checkins WHERE habit_id = ? AND day = ?`, habitID, day).Scan(&cnt)
	if cnt > 0 {
		_, err := m.db.Exec(`DELETE FROM discipline_habit_checkins WHERE habit_id = ? AND day = ?`, habitID, day)
		return false, err
	}
	_, err := m.db.Exec(`INSERT OR IGNORE INTO discipline_habit_checkins (habit_id, day) VALUES (?, ?)`, habitID, day)
	m.LogEvent("habit_checkin", 0, 0)
	return true, err
}

// ─── Reminders ────────────────────────────────────────────────────────────────

type Reminder struct {
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	RemindAt  string `json:"remind_at"` // "2006-01-02 15:04"
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
}

func (m *Manager) ListReminders() ([]Reminder, error) {
	rows, err := m.db.Query(`SELECT id, text, remind_at, done, created_at FROM discipline_reminders ORDER BY done, remind_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Reminder
	for rows.Next() {
		var r Reminder
		var done int
		if err := rows.Scan(&r.ID, &r.Text, &r.RemindAt, &done, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Done = done == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

func (m *Manager) CreateReminder(text, remindAt string) (int64, error) {
	if strings.TrimSpace(text) == "" || strings.TrimSpace(remindAt) == "" {
		return 0, fmt.Errorf("text and time required")
	}
	res, err := m.db.Exec(`INSERT INTO discipline_reminders (text, remind_at) VALUES (?, ?)`, text, remindAt)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (m *Manager) SetReminderDone(id int64, done bool) error {
	d := 0
	if done {
		d = 1
	}
	_, err := m.db.Exec(`UPDATE discipline_reminders SET done = ? WHERE id = ?`, d, id)
	return err
}

func (m *Manager) DeleteReminder(id int64) error {
	_, err := m.db.Exec(`DELETE FROM discipline_reminders WHERE id = ?`, id)
	return err
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

func (m *Manager) ListTasks(projectID int64, all bool) ([]Task, error) {
	q := `SELECT id, project_id, title, notes, status, priority, labels, due_date, done, created_at, completed_at, size, parent_id,
		start_date, estimate_minutes, actual_minutes, energy, milestone_id, blocked_reason, recurrence, reminder_at, links, backlog, archived, position, my_day, scheduled_at, waiting_for, evidence, focus_profile_id
	      FROM discipline_tasks`
	var args []interface{}
	if !all {
		q += ` WHERE project_id = ?`
		args = append(args, projectID)
	}
	q += ` ORDER BY backlog ASC, parent_id ASC, position ASC, done ASC, CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 ELSE 2 END, created_at ASC`

	rows, err := m.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanTask(rows *sql.Rows) (Task, error) {
	var t Task
	var done, backlog, archived, myDay int
	err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Status, &t.Priority,
		&t.Labels, &t.DueDate, &done, &t.CreatedAt, &t.CompletedAt, &t.Size, &t.ParentID,
		&t.StartDate, &t.EstimateMin, &t.ActualMin, &t.Energy, &t.MilestoneID, &t.BlockedReason,
		&t.Recurrence, &t.ReminderAt, &t.Links, &backlog, &archived, &t.Position, &myDay, &t.ScheduledAt, &t.WaitingFor, &t.Evidence, &t.FocusProfileID)
	t.Done = done == 1
	t.Backlog = backlog == 1
	t.Archived = archived == 1
	t.MyDay = myDay == 1
	return t, err
}

type TaskInput struct {
	ProjectID      int64  `json:"project_id"`
	ParentID       int64  `json:"parent_id"`
	Title          string `json:"title"`
	Notes          string `json:"notes"`
	Priority       string `json:"priority"`
	Labels         string `json:"labels"`
	DueDate        string `json:"due_date"`
	StartDate      string `json:"start_date"`
	EstimateMin    int    `json:"estimate_minutes"`
	ActualMin      int    `json:"actual_minutes"`
	Energy         string `json:"energy"`
	MilestoneID    int64  `json:"milestone_id"`
	BlockedReason  string `json:"blocked_reason"`
	Recurrence     string `json:"recurrence"`
	ReminderAt     string `json:"reminder_at"`
	Links          string `json:"links"`
	Backlog        bool   `json:"backlog"`
	Position       int    `json:"position"`
	MyDay          bool   `json:"my_day"`
	ScheduledAt    string `json:"scheduled_at"`
	WaitingFor     string `json:"waiting_for"`
	Evidence       string `json:"evidence"`
	FocusProfileID int64  `json:"focus_profile_id"`
}

func (m *Manager) CreateTask(in TaskInput) (int64, error) {
	if strings.TrimSpace(in.Title) == "" {
		return 0, fmt.Errorf("title required")
	}
	if in.Priority == "" {
		in.Priority = "medium"
	}
	if in.Energy == "" {
		in.Energy = "medium"
	}
	backlog := 0
	if in.Backlog {
		backlog = 1
	}
	myDay := 0
	if in.MyDay {
		myDay = 1
	}
	res, err := m.db.Exec(`INSERT INTO discipline_tasks (project_id, parent_id, title, notes, priority, labels, due_date, start_date, estimate_minutes, actual_minutes, energy, milestone_id, blocked_reason, recurrence, reminder_at, links, backlog, position, my_day, scheduled_at, waiting_for, evidence, focus_profile_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, in.ProjectID, in.ParentID, strings.TrimSpace(in.Title), in.Notes, in.Priority, in.Labels, in.DueDate, in.StartDate, in.EstimateMin, in.ActualMin, in.Energy, in.MilestoneID, in.BlockedReason, in.Recurrence, in.ReminderAt, in.Links, backlog, in.Position, myDay, in.ScheduledAt, in.WaitingFor, in.Evidence, in.FocusProfileID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	m.LogEvent("task_created", in.ProjectID, id)
	return id, nil
}

// TaskPatch carries optional task field updates.
type TaskPatch struct {
	ProjectID      *int64  `json:"project_id"`
	Title          *string `json:"title"`
	Notes          *string `json:"notes"`
	Status         *string `json:"status"`
	Priority       *string `json:"priority"`
	Size           *string `json:"size"`
	Labels         *string `json:"labels"`
	DueDate        *string `json:"due_date"`
	StartDate      *string `json:"start_date"`
	EstimateMin    *int    `json:"estimate_minutes"`
	ActualMin      *int    `json:"actual_minutes"`
	Energy         *string `json:"energy"`
	MilestoneID    *int64  `json:"milestone_id"`
	BlockedReason  *string `json:"blocked_reason"`
	Recurrence     *string `json:"recurrence"`
	ReminderAt     *string `json:"reminder_at"`
	Links          *string `json:"links"`
	Backlog        *bool   `json:"backlog"`
	Archived       *bool   `json:"archived"`
	Position       *int    `json:"position"`
	MyDay          *bool   `json:"my_day"`
	ScheduledAt    *string `json:"scheduled_at"`
	WaitingFor     *string `json:"waiting_for"`
	Evidence       *string `json:"evidence"`
	FocusProfileID *int64  `json:"focus_profile_id"`
}

// UpdateTask applies a partial update. Returns the updated task and whether it
// was just completed (for XP awards).
func (m *Manager) UpdateTask(id int64, p TaskPatch) (Task, bool, error) {
	cur, err := m.getTask(id)
	if err != nil {
		return Task{}, false, err
	}
	oldStatus := cur.Status
	oldPriority := cur.Priority
	if p.ProjectID != nil {
		cur.ProjectID = *p.ProjectID
	}

	if p.Title != nil {
		cur.Title = *p.Title
	}
	if p.Notes != nil {
		cur.Notes = *p.Notes
	}
	if p.Priority != nil {
		cur.Priority = *p.Priority
	}
	if p.Size != nil {
		cur.Size = *p.Size
	}
	if p.Labels != nil {
		cur.Labels = *p.Labels
	}
	if p.DueDate != nil {
		cur.DueDate = *p.DueDate
	}
	if p.StartDate != nil {
		cur.StartDate = *p.StartDate
	}
	if p.EstimateMin != nil {
		cur.EstimateMin = *p.EstimateMin
	}
	if p.ActualMin != nil {
		cur.ActualMin = *p.ActualMin
	}
	if p.Energy != nil {
		cur.Energy = *p.Energy
	}
	if p.MilestoneID != nil {
		cur.MilestoneID = *p.MilestoneID
	}
	if p.BlockedReason != nil {
		cur.BlockedReason = *p.BlockedReason
	}
	if p.Recurrence != nil {
		cur.Recurrence = *p.Recurrence
	}
	if p.ReminderAt != nil {
		cur.ReminderAt = *p.ReminderAt
	}
	if p.Links != nil {
		cur.Links = *p.Links
	}
	if p.Backlog != nil {
		cur.Backlog = *p.Backlog
	}
	if p.Archived != nil {
		cur.Archived = *p.Archived
	}
	if p.Position != nil {
		cur.Position = *p.Position
	}
	if p.MyDay != nil {
		cur.MyDay = *p.MyDay
	}
	if p.ScheduledAt != nil {
		cur.ScheduledAt = *p.ScheduledAt
	}
	if p.WaitingFor != nil {
		cur.WaitingFor = *p.WaitingFor
	}
	if p.Evidence != nil {
		cur.Evidence = *p.Evidence
	}
	if p.FocusProfileID != nil {
		cur.FocusProfileID = *p.FocusProfileID
	}

	completedNow := false
	if p.Status != nil {
		cur.Status = *p.Status
	}
	if cur.Status == "done" && !cur.Done {
		cur.Done = true
		cur.CompletedAt = time.Now().Format("2006-01-02 15:04:05")
		completedNow = true
	} else if cur.Status != "done" && cur.Done {
		cur.Done = false
		cur.CompletedAt = ""
	}

	done := 0
	if cur.Done {
		done = 1
	}
	backlog, archived, myDay := 0, 0, 0
	if cur.Backlog {
		backlog = 1
	}
	if cur.Archived {
		archived = 1
	}
	if cur.MyDay {
		myDay = 1
	}
	_, err = m.db.Exec(`UPDATE discipline_tasks SET project_id=?, title=?, notes=?, status=?, priority=?, size=?, labels=?, due_date=?, done=?, completed_at=?, start_date=?, estimate_minutes=?, actual_minutes=?, energy=?, milestone_id=?, blocked_reason=?, recurrence=?, reminder_at=?, links=?, backlog=?, archived=?, position=?, my_day=?, scheduled_at=?, waiting_for=?, evidence=?, focus_profile_id=? WHERE id=?`,
		cur.ProjectID, cur.Title, cur.Notes, cur.Status, cur.Priority, cur.Size, cur.Labels, cur.DueDate, done, cur.CompletedAt, cur.StartDate, cur.EstimateMin, cur.ActualMin, cur.Energy, cur.MilestoneID, cur.BlockedReason, cur.Recurrence, cur.ReminderAt, cur.Links, backlog, archived, cur.Position, myDay, cur.ScheduledAt, cur.WaitingFor, cur.Evidence, cur.FocusProfileID, cur.ID)
	if err != nil {
		return Task{}, false, err
	}

	m.emitTaskEvents(cur, oldStatus, oldPriority, completedNow, p)
	if completedNow && strings.TrimSpace(cur.Recurrence) != "" {
		m.createNextRecurringTask(cur)
	}
	return cur, completedNow, nil
}

// createNextRecurringTask turns recurrence into working behavior: completing an
// occurrence schedules a fresh task while preserving its planning metadata.
func (m *Manager) createNextRecurringTask(t Task) {
	nextDate := func(raw string) string {
		if raw == "" {
			return ""
		}
		d, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return raw
		}
		switch strings.ToLower(strings.TrimSpace(t.Recurrence)) {
		case "daily", "day":
			d = d.AddDate(0, 0, 1)
		case "weekdays", "weekday":
			for d = d.AddDate(0, 0, 1); d.Weekday() == time.Saturday || d.Weekday() == time.Sunday; d = d.AddDate(0, 0, 1) {
			}
		case "monthly", "month":
			d = d.AddDate(0, 1, 0)
		case "yearly", "year":
			d = d.AddDate(1, 0, 0)
		default:
			d = d.AddDate(0, 0, 7)
		}
		return d.Format("2006-01-02")
	}
	_, _ = m.CreateTask(TaskInput{
		ProjectID: t.ProjectID, ParentID: t.ParentID, Title: t.Title, Notes: t.Notes,
		Priority: t.Priority, Labels: t.Labels, DueDate: nextDate(t.DueDate), StartDate: nextDate(t.StartDate),
		EstimateMin: t.EstimateMin, Energy: t.Energy, MilestoneID: t.MilestoneID,
		Recurrence: t.Recurrence, Links: t.Links, Position: t.Position,
		ScheduledAt: t.ScheduledAt, WaitingFor: t.WaitingFor, FocusProfileID: t.FocusProfileID,
	})
}

// emitTaskEvents records granular activity events for what actually changed.
func (m *Manager) emitTaskEvents(t Task, oldStatus, oldPriority string, completedNow bool, p TaskPatch) {
	if completedNow {
		if t.ParentID > 0 {
			m.LogEvent("subtask_completed", t.ProjectID, t.ID)
		} else {
			m.LogEvent("task_completed", t.ProjectID, t.ID)
		}
		if strings.Contains(strings.ToLower(t.Labels), "bug") {
			m.LogEvent("bug_fixed", t.ProjectID, t.ID)
		}
		return
	}
	if p.Status != nil && t.Status != oldStatus {
		m.LogEvent("status_changed", t.ProjectID, t.ID)
	}
	if p.Priority != nil && t.Priority != oldPriority {
		m.LogEvent("priority_changed", t.ProjectID, t.ID)
	}
	if p.ProjectID != nil || p.Title != nil || p.Notes != nil || p.Size != nil || p.Labels != nil || p.DueDate != nil || p.StartDate != nil || p.EstimateMin != nil || p.ActualMin != nil || p.MilestoneID != nil || p.BlockedReason != nil || p.Backlog != nil || p.Archived != nil || p.MyDay != nil || p.ScheduledAt != nil || p.WaitingFor != nil || p.Evidence != nil || p.FocusProfileID != nil {
		m.LogEvent("task_updated", t.ProjectID, t.ID)
	}
}

func (m *Manager) getTask(id int64) (Task, error) {
	rows, err := m.db.Query(`SELECT id, project_id, title, notes, status, priority, labels, due_date, done, created_at, completed_at, size, parent_id,
		start_date, estimate_minutes, actual_minutes, energy, milestone_id, blocked_reason, recurrence, reminder_at, links, backlog, archived, position, my_day, scheduled_at, waiting_for, evidence, focus_profile_id
		FROM discipline_tasks WHERE id = ?`, id)
	if err != nil {
		return Task{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Task{}, fmt.Errorf("task not found")
	}
	return scanTask(rows)
}

func (m *Manager) DeleteTask(id int64) error {
	// Deleting a parent removes its subtasks too.
	_, err := m.db.Exec(`DELETE FROM discipline_tasks WHERE id = ? OR parent_id = ?`, id, id)
	return err
}

// ─── Activity: heatmap, stats, day drilldown ──────────────────────────────────

// Heatmap returns per-day event counts for the last `days` days (project 0 = all).
func (m *Manager) Heatmap(projectID int64, days int) ([]DayCount, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	q := `SELECT day, COUNT(*) FROM discipline_events WHERE day >= ?`
	args := []interface{}{since}
	if projectID > 0 {
		q += ` AND project_id = ?`
		args = append(args, projectID)
	}
	q += ` GROUP BY day`

	rows, err := m.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var d string
		var c int
		if err := rows.Scan(&d, &c); err != nil {
			return nil, err
		}
		counts[d] = c
	}

	out := make([]DayCount, 0, days)
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, DayCount{Day: d, Count: counts[d]})
	}
	return out, nil
}

// Stats computes streaks and counts from the event log (project 0 = all).
func (m *Manager) Stats(projectID int64) (Stats, error) {
	var s Stats
	s.ByType = map[string]int{}

	q := `SELECT type, COUNT(*) FROM discipline_events`
	args := []interface{}{}
	if projectID > 0 {
		q += ` WHERE project_id = ?`
		args = append(args, projectID)
	}
	q += ` GROUP BY type`
	rows, err := m.db.Query(q, args...)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var t string
		var c int
		rows.Scan(&t, &c)
		s.ByType[t] = c
		s.Total += c
		switch t {
		case "task_completed":
			s.TasksCompleted = c
		case "subtask_completed":
			s.SubtasksCompleted = c
		case "bug_fixed":
			s.BugsFixed = c
		}
	}
	rows.Close()

	// Distinct active days (desc) → streaks.
	dq := `SELECT DISTINCT day FROM discipline_events`
	if projectID > 0 {
		dq += ` WHERE project_id = ?`
	}
	dq += ` ORDER BY day DESC`
	drows, err := m.db.Query(dq, args...)
	if err != nil {
		return s, err
	}
	var days []string
	for drows.Next() {
		var d string
		drows.Scan(&d)
		days = append(days, d)
	}
	drows.Close()

	s.ActiveDays = len(days)
	s.CurrentStreak, s.LongestStreak = computeStreaks(days)
	return s, nil
}

// computeStreaks takes active days (desc) and returns current + longest streak.
func computeStreaks(daysDesc []string) (current, longest int) {
	if len(daysDesc) == 0 {
		return 0, 0
	}
	set := map[string]bool{}
	for _, d := range daysDesc {
		set[d] = true
	}

	// Current streak: today or yesterday backwards.
	today := time.Now()
	start := today
	if !set[today.Format("2006-01-02")] {
		start = today.AddDate(0, 0, -1)
	}
	for set[start.Format("2006-01-02")] {
		current++
		start = start.AddDate(0, 0, -1)
	}

	// Longest streak across all active days.
	sorted := make([]string, len(daysDesc))
	copy(sorted, daysDesc)
	sort.Strings(sorted) // ascending
	run := 0
	var prev time.Time
	for i, ds := range sorted {
		d, _ := time.Parse("2006-01-02", ds)
		if i > 0 && d.Sub(prev) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > longest {
			longest = run
		}
		prev = d
	}
	return current, longest
}

// DayEvents returns the events logged on a given date (YYYY-MM-DD).
func (m *Manager) DayEvents(date string, projectID int64) ([]Event, error) {
	q := `SELECT e.type, e.project_id, e.task_id, COALESCE(t.title,''), e.created_at
	      FROM discipline_events e LEFT JOIN discipline_tasks t ON t.id = e.task_id
	      WHERE e.day = ?`
	args := []interface{}{date}
	if projectID > 0 {
		q += ` AND e.project_id = ?`
		args = append(args, projectID)
	}
	q += ` ORDER BY e.created_at DESC`
	return m.queryEvents(q, args...)
}

// RecentEvents returns the newest events (for an activity feed).
func (m *Manager) RecentEvents(projectID int64, limit int) ([]Event, error) {
	q := `SELECT e.type, e.project_id, e.task_id, COALESCE(t.title,''), e.created_at
	      FROM discipline_events e LEFT JOIN discipline_tasks t ON t.id = e.task_id`
	args := []interface{}{}
	if projectID > 0 {
		q += ` WHERE e.project_id = ?`
		args = append(args, projectID)
	}
	q += ` ORDER BY e.id DESC LIMIT ?`
	args = append(args, limit)
	return m.queryEvents(q, args...)
}

func (m *Manager) queryEvents(q string, args ...interface{}) ([]Event, error) {
	rows, err := m.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.Type, &e.ProjectID, &e.TaskID, &e.Title, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SeriesByType returns a continuous per-day count of one event type (e.g.
// task_completed) over the last `days` days — for progress charts.
func (m *Manager) SeriesByType(projectID int64, eventType string, days int) ([]DayCount, error) {
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	q := `SELECT day, COUNT(*) FROM discipline_events WHERE type = ? AND day >= ?`
	args := []interface{}{eventType, since}
	if projectID > 0 {
		q += ` AND project_id = ?`
		args = append(args, projectID)
	}
	q += ` GROUP BY day`
	rows, err := m.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var d string
		var c int
		rows.Scan(&d, &c)
		counts[d] = c
	}
	out := make([]DayCount, 0, days)
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, DayCount{Day: d, Count: counts[d]})
	}
	return out, nil
}
