package discipline

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type InboxMessage struct {
	ID          int64  `json:"id"`
	Sender      string `json:"sender"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	ActionLabel string `json:"action_label"`
	ActionView  string `json:"action_view"`
	IsRead      bool   `json:"is_read"`
	Starred     bool   `json:"starred"`
	Archived    bool   `json:"archived"`
	CreatedAt   string `json:"created_at"`
}

type InboxInput struct {
	Sender      string `json:"sender"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	ActionLabel string `json:"action_label"`
	ActionView  string `json:"action_view"`
}

type InboxPatch struct {
	IsRead   *bool `json:"is_read"`
	Starred  *bool `json:"starred"`
	Archived *bool `json:"archived"`
}

type InboxStats struct {
	Total    int `json:"total"`
	Unread   int `json:"unread"`
	Starred  int `json:"starred"`
	Archived int `json:"archived"`
}

func validInboxView(view string) bool {
	switch view {
	case "today", "tasks", "goals", "habits", "reminders", "journal", "analytics", "projects", "activity", "command", "home":
		return true
	}
	return false
}

func (m *Manager) CreateInboxMessage(input InboxInput) (int64, error) {
	input.Subject = strings.TrimSpace(input.Subject)
	if input.Subject == "" {
		return 0, errors.New("message subject is required")
	}
	if strings.TrimSpace(input.Sender) == "" {
		input.Sender = "Discipline Intelligence"
	}
	if input.Category == "" {
		input.Category = "insight"
	}
	if input.Severity != "attention" && input.Severity != "success" {
		input.Severity = "normal"
	}
	if !validInboxView(input.ActionView) {
		input.ActionView = ""
	}
	r, err := m.db.Exec(`INSERT INTO discipline_inbox_messages(sender,subject,body,category,severity,action_label,action_view) VALUES(?,?,?,?,?,?,?)`, input.Sender, input.Subject, strings.TrimSpace(input.Body), input.Category, input.Severity, input.ActionLabel, input.ActionView)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	if err == nil {
		m.LogEvent("inbox_message", 0, 0)
	}
	return id, err
}

func (m *Manager) ListInboxMessages(scope, query string) ([]InboxMessage, error) {
	where := []string{"1=1"}
	args := []any{}
	switch scope {
	case "unread":
		where = append(where, "is_read=0", "archived=0")
	case "starred":
		where = append(where, "starred=1", "archived=0")
	case "archived":
		where = append(where, "archived=1")
	default:
		where = append(where, "archived=0")
	}
	if query = strings.TrimSpace(query); query != "" {
		where = append(where, "(subject LIKE ? OR body LIKE ? OR sender LIKE ?)")
		q := "%" + query + "%"
		args = append(args, q, q, q)
	}
	rows, err := m.db.Query(`SELECT id,sender,subject,body,category,severity,action_label,action_view,is_read,starred,archived,created_at FROM discipline_inbox_messages WHERE `+strings.Join(where, " AND ")+` ORDER BY starred DESC,created_at DESC,id DESC LIMIT 250`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []InboxMessage{}
	for rows.Next() {
		var x InboxMessage
		if err = rows.Scan(&x.ID, &x.Sender, &x.Subject, &x.Body, &x.Category, &x.Severity, &x.ActionLabel, &x.ActionView, &x.IsRead, &x.Starred, &x.Archived, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (m *Manager) InboxStats() (InboxStats, error) {
	var s InboxStats
	err := m.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(CASE WHEN is_read=0 AND archived=0 THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN starred=1 AND archived=0 THEN 1 ELSE 0 END),0),COALESCE(SUM(archived),0) FROM discipline_inbox_messages`).Scan(&s.Total, &s.Unread, &s.Starred, &s.Archived)
	return s, err
}

func (m *Manager) UpdateInboxMessage(id int64, p InboxPatch) error {
	sets := []string{}
	args := []any{}
	if p.IsRead != nil {
		sets = append(sets, "is_read=?")
		args = append(args, *p.IsRead)
	}
	if p.Starred != nil {
		sets = append(sets, "starred=?")
		args = append(args, *p.Starred)
	}
	if p.Archived != nil {
		sets = append(sets, "archived=?")
		args = append(args, *p.Archived)
	}
	if len(sets) == 0 {
		return errors.New("no changes supplied")
	}
	args = append(args, id)
	r, err := m.db.Exec(`UPDATE discipline_inbox_messages SET `+strings.Join(sets, ",")+` WHERE id=?`, args...)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (m *Manager) DeleteInboxMessage(id int64) error {
	r, e := m.db.Exec(`DELETE FROM discipline_inbox_messages WHERE id=?`, id)
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GenerateDailyBriefing creates a deterministic, private briefing from local data.
// A future AI provider can use the same inbox contract without changing the UI.
func (m *Manager) GenerateDailyBriefing() (int64, error) {
	var open, overdue, myday, habits, journal int
	m.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(CASE WHEN due_date<>'' AND date(due_date)<date('now','localtime') THEN 1 ELSE 0 END),0),COALESCE(SUM(my_day),0) FROM discipline_tasks WHERE done=0 AND archived=0`).Scan(&open, &overdue, &myday)
	m.db.QueryRow(`SELECT COUNT(*) FROM discipline_habits`).Scan(&habits)
	m.db.QueryRow(`SELECT COUNT(*) FROM discipline_journal_entries WHERE day=date('now','localtime')`).Scan(&journal)
	subject := "Your discipline briefing · " + time.Now().Format("Monday")
	body := fmt.Sprintf("Here is your private local briefing.\n\n• %d open tasks\n• %d overdue decisions\n• %d tasks committed to My Day\n• %d active habits\n• %d journal entries today\n\n", open, overdue, myday, habits, journal)
	severity, action, view := "normal", "Open My Day", "today"
	if overdue > 0 {
		severity = "attention"
		action = "Review overdue work"
		view = "tasks"
		body += "Priority: resolve, reschedule, or remove overdue work before adding more commitments."
	} else if myday == 0 {
		body += "Your day has no deliberate commitment yet. Choose one meaningful task before reacting to new work."
	} else {
		severity = "success"
		body += "Your system is clear enough to execute. Protect a focus block and finish the most important commitment first."
	}
	return m.CreateInboxMessage(InboxInput{Subject: subject, Body: body, Category: "daily briefing", Severity: severity, ActionLabel: action, ActionView: view})
}
