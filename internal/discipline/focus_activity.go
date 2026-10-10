package discipline

import (
	"errors"
	"strings"
	"time"
)

type FocusSession struct {
	ID                   int64           `json:"id"`
	CapsuleID            int64           `json:"capsule_id"`
	LockedYouTubeChannel string          `json:"locked_youtube_channel"`
	TaskID               int64           `json:"task_id"`
	TaskTitle            string          `json:"task_title"`
	PlannedMinutes       int             `json:"planned_minutes"`
	ActualSeconds        int             `json:"actual_seconds"`
	Status               string          `json:"status"`
	ActivityLogging      bool            `json:"activity_logging"`
	StartedAt            string          `json:"started_at"`
	EndedAt              string          `json:"ended_at"`
	Reflection           string          `json:"reflection"`
	FocusScore           int             `json:"focus_score"`
	Activity             []FocusActivity `json:"activity,omitempty"`
	Distractions         []Distraction   `json:"distractions,omitempty"`
}

type FocusActivity struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Seconds      int    `json:"seconds"`
	Observations int    `json:"observations"`
	FirstSeen    string `json:"first_seen"`
	LastSeen     string `json:"last_seen"`
}

type Distraction struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
}

func (m *Manager) StartFocusSession(taskID int64, taskTitle string, minutes int, logging bool, capsuleID ...int64) (int64, error) {
	if minutes < 1 || minutes > 720 {
		return 0, errors.New("focus duration must be between 1 and 720 minutes")
	}
	m.focusMu.Lock()
	defer m.focusMu.Unlock()
	if m.focusID != 0 {
		return 0, errors.New("a focus session is already active")
	}
	selectedCapsule := int64(0)
	if len(capsuleID) > 0 {
		selectedCapsule = capsuleID[0]
		if selectedCapsule > 0 {
			if _, err := m.FocusCapsule(selectedCapsule); err != nil {
				return 0, errors.New("focus capsule not found")
			}
		}
	}
	r, err := m.db.Exec(`INSERT INTO discipline_focus_sessions(task_id,task_title,planned_minutes,activity_logging,capsule_id) VALUES(?,?,?,?,?)`, taskID, strings.TrimSpace(taskTitle), minutes, logging, selectedCapsule)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}
	m.focusID = id
	m.focusEnd = make(chan struct{})
	if logging {
		go m.trackForegroundApps(id, m.focusEnd)
	}
	m.scheduleFocusExpiry(id, time.Duration(minutes)*time.Minute, m.focusEnd)
	return id, nil
}

func (m *Manager) scheduleFocusExpiry(id int64, remaining time.Duration, stop <-chan struct{}) {
	if remaining <= 0 {
		remaining = time.Second
	}
	go func() {
		timer := time.NewTimer(remaining)
		defer timer.Stop()
		select {
		case <-timer.C:
			_, _ = m.EndFocusSession(id, "completed", "Session ended automatically when its planned time expired.", 0, false)
		case <-stop:
		}
	}()
}

// restoreFocusSession resumes an unfinished session after a local service or
// computer restart. Expired sessions are closed; live ones restart the private
// foreground activity recorder when consent was enabled.
func (m *Manager) restoreFocusSession() {
	var id int64
	var planned int
	var logging bool
	var started string
	err := m.db.QueryRow(`SELECT id,planned_minutes,activity_logging,started_at FROM discipline_focus_sessions WHERE status='active' ORDER BY id DESC LIMIT 1`).Scan(&id, &planned, &logging, &started)
	if err != nil {
		return
	}
	start, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSuffix(started, "Z"), time.Local)
	if err != nil {
		start, err = time.Parse(time.RFC3339, started)
	}
	if err != nil || time.Since(start) >= time.Duration(planned)*time.Minute {
		_, _ = m.db.Exec(`UPDATE discipline_focus_sessions SET status='completed',actual_seconds=MAX(1,planned_minutes*60),ended_at=CURRENT_TIMESTAMP WHERE id=?`, id)
		return
	}
	m.focusMu.Lock()
	m.focusID = id
	m.focusEnd = make(chan struct{})
	m.focusMu.Unlock()
	if logging {
		go m.trackForegroundApps(id, m.focusEnd)
	}
	m.scheduleFocusExpiry(id, time.Duration(planned)*time.Minute-time.Since(start), m.focusEnd)
}

func (m *Manager) ActiveFocusSessionID() int64 {
	m.focusMu.RLock()
	defer m.focusMu.RUnlock()
	return m.focusID
}

func (m *Manager) RecordFocusActivity(kind, name string, seconds int) {
	m.focusMu.RLock()
	id := m.focusID
	m.focusMu.RUnlock()
	if id == 0 {
		return
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || len(name) > 255 {
		return
	}
	if kind != "app" && kind != "domain" {
		return
	}
	if seconds < 0 {
		seconds = 0
	}
	m.db.Exec(`INSERT INTO discipline_focus_activity(session_id,kind,name,seconds,observations) VALUES(?,?,?,?,1) ON CONFLICT(session_id,kind,name) DO UPDATE SET seconds=seconds+excluded.seconds,observations=observations+1,last_seen=CURRENT_TIMESTAMP`, id, kind, name, seconds)
}

func (m *Manager) AddDistraction(sessionID int64, label, note string) error {
	if sessionID == 0 {
		sessionID = m.ActiveFocusSessionID()
	}
	if sessionID == 0 {
		return errors.New("no active focus session")
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return errors.New("distraction label is required")
	}
	_, err := m.db.Exec(`INSERT INTO discipline_distractions(session_id,label,note) VALUES(?,?,?)`, sessionID, label, strings.TrimSpace(note))
	return err
}

func (m *Manager) EndFocusSession(sessionID int64, status, reflection string, score int, createJournal bool) (FocusSession, error) {
	m.focusMu.Lock()
	if sessionID == 0 {
		sessionID = m.focusID
	}
	if sessionID == 0 {
		m.focusMu.Unlock()
		return FocusSession{}, errors.New("no active focus session")
	}
	if m.focusID == sessionID && m.focusEnd != nil {
		close(m.focusEnd)
		m.focusEnd = nil
		m.focusID = 0
	}
	m.focusMu.Unlock()
	if status != "completed" && status != "cancelled" {
		status = "completed"
	}
	score = normalizeJournalScore(score)
	_, err := m.db.Exec(`UPDATE discipline_focus_sessions SET status=?,actual_seconds=MAX(1,CAST((julianday('now')-julianday(started_at))*86400 AS INTEGER)),ended_at=CURRENT_TIMESTAMP,reflection=?,focus_score=? WHERE id=?`, status, strings.TrimSpace(reflection), score, sessionID)
	if err != nil {
		return FocusSession{}, err
	}
	s, err := m.FocusSessionDetails(sessionID)
	if err != nil {
		return s, err
	}
	if createJournal && status == "completed" {
		content := strings.TrimSpace(reflection)
		if content == "" {
			content = "Completed a focused work session."
		}
		_, _ = m.CreateJournalEntry(JournalEntryInput{Title: "Focus · " + s.TaskTitle, Content: content, Day: time.Now().Format("2006-01-02"), DurationMinutes: max(1, s.ActualSeconds/60), Focus: score, Tags: "focus-session", ProjectID: 0})
	}
	return s, nil
}

func (m *Manager) FocusSessionDetails(id int64) (FocusSession, error) {
	var s FocusSession
	err := m.db.QueryRow(`SELECT id,capsule_id,locked_youtube_channel,task_id,task_title,planned_minutes,actual_seconds,status,activity_logging,started_at,ended_at,reflection,focus_score FROM discipline_focus_sessions WHERE id=?`, id).Scan(&s.ID, &s.CapsuleID, &s.LockedYouTubeChannel, &s.TaskID, &s.TaskTitle, &s.PlannedMinutes, &s.ActualSeconds, &s.Status, &s.ActivityLogging, &s.StartedAt, &s.EndedAt, &s.Reflection, &s.FocusScore)
	if err != nil {
		return s, err
	}
	rows, err := m.db.Query(`SELECT kind,name,seconds,observations,first_seen,last_seen FROM discipline_focus_activity WHERE session_id=? ORDER BY seconds DESC,observations DESC`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a FocusActivity
			if rows.Scan(&a.Kind, &a.Name, &a.Seconds, &a.Observations, &a.FirstSeen, &a.LastSeen) == nil {
				s.Activity = append(s.Activity, a)
			}
		}
	}
	dr, err := m.db.Query(`SELECT id,label,note,created_at FROM discipline_distractions WHERE session_id=? ORDER BY id`, id)
	if err == nil {
		defer dr.Close()
		for dr.Next() {
			var d Distraction
			if dr.Scan(&d.ID, &d.Label, &d.Note, &d.CreatedAt) == nil {
				s.Distractions = append(s.Distractions, d)
			}
		}
	}
	return s, nil
}

func (m *Manager) RecentFocusSessions(limit int) ([]FocusSession, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := m.db.Query(`SELECT id FROM discipline_focus_sessions ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FocusSession{}
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			s, e := m.FocusSessionDetails(id)
			if e == nil {
				out = append(out, s)
			}
		}
	}
	return out, rows.Err()
}

func (m *Manager) trackForegroundApps(sessionID int64, stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			name := foregroundApplication()
			if name != "" {
				m.RecordFocusActivity("app", name, 5)
			}
		}
	}
}
