// Package deadlines prepares local desktop reminders without changing user data.
package deadlines

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Notification struct {
	Key   string    `json:"key"`
	Title string    `json:"title"`
	Body  string    `json:"body"`
	Kind  string    `json:"kind"`
	DueAt time.Time `json:"due_at"`
}

func parseDate(raw string, location *time.Location) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if value, e := time.Parse(time.RFC3339, raw); e == nil {
		return value, true
	}
	for _, format := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"} {
		if value, e := time.ParseInLocation(format, raw, location); e == nil {
			if format == "2006-01-02" {
				value = value.Add(9 * time.Hour)
			}
			return value, true
		}
	}
	return time.Time{}, false
}
func Collect(db *sql.DB, now time.Time) ([]Notification, error) {
	queries := []struct{ table, kind, title, query string }{
		{"discipline_reminders", "reminder", "Reminder", `SELECT id,text,remind_at FROM discipline_reminders WHERE done=0 AND remind_at<>''`},
		{"discipline_tasks", "task-reminder", "Task reminder", `SELECT id,title,reminder_at FROM discipline_tasks WHERE done=0 AND archived=0 AND reminder_at<>''`},
		{"discipline_tasks", "task", "Task deadline", `SELECT id,title,due_date FROM discipline_tasks WHERE done=0 AND archived=0 AND due_date<>''`},
		{"discipline_projects", "project", "Project deadline", `SELECT id,name,target_date FROM discipline_projects WHERE status NOT IN ('done','paused','archived','cancelled') AND target_date<>''`},
		{"discipline_goals", "goal", "Goal deadline", `SELECT id,name,target_date FROM discipline_goals WHERE status NOT IN ('done','paused','archived','cancelled') AND target_date<>''`},
		{"discipline_milestones", "milestone", "Milestone deadline", `SELECT m.id,m.name,m.target_date FROM discipline_milestones m JOIN discipline_projects p ON p.id=m.project_id WHERE m.status NOT IN ('done','cancelled') AND p.status NOT IN ('done','paused','archived','cancelled') AND m.target_date<>''`},
	}
	out := []Notification{}
	seen := map[string]bool{}
	for _, q := range queries {
		var exists int
		if e := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, q.table).Scan(&exists); e != nil {
			return nil, e
		}
		if exists == 0 {
			continue
		}
		rows, e := db.Query(q.query)
		if e != nil {
			return nil, fmt.Errorf("read %s reminders: %w", q.kind, e)
		}
		for rows.Next() {
			var id int64
			var name, raw string
			if e = rows.Scan(&id, &name, &raw); e != nil {
				rows.Close()
				return nil, e
			}
			due, valid := parseDate(raw, now.Location())
			if !valid || due.Before(now.Add(-24*time.Hour)) || due.After(now.Add(15*time.Minute)) {
				continue
			}
			stage, heading := "due", q.title+" is due"
			if due.After(now) {
				stage = "soon"
				heading = q.title + " in " + fmt.Sprint(int(due.Sub(now).Minutes()+0.999)) + " minutes"
			}
			family := q.kind
			if family == "task-reminder" {
				family = "task"
			}
			identity := fmt.Sprintf("%s:%d:%s:%s", family, id, due.UTC().Format(time.RFC3339), stage)
			if seen[identity] {
				continue
			}
			seen[identity] = true
			sum := sha256.Sum256([]byte(identity))
			out = append(out, Notification{Key: hex.EncodeToString(sum[:16]), Title: heading, Body: strings.TrimSpace(name), Kind: q.kind, DueAt: due})
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DueAt.After(out[j].DueAt) })
	return out, nil
}
