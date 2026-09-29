package appnetwork

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type App struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	PID       int    `json:"pid"`
	Running   bool   `json:"running"`
	Blocked   bool   `json:"blocked"`
	RuleName  string `json:"rule_name,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type Event struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Action    string `json:"action"`
	Success   bool   `json:"success"`
	Detail    string `json:"detail"`
	CreatedAt string `json:"created_at"`
}

type platform interface {
	RunningApps() ([]App, error)
	Block(path, ruleName string) error
	Allow(ruleName string) error
	Supported() bool
}

type Manager struct {
	db       *sql.DB
	platform platform
}

func New(db *sql.DB) (*Manager, error) {
	m, err := newWithPlatform(db, newPlatform())
	if err != nil {
		return nil, err
	}
	_ = m.EvaluateSchedules(time.Now())
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_ = m.EvaluateSchedules(time.Now())
		}
	}()
	return m, nil
}

func newWithPlatform(db *sql.DB, p platform) (*Manager, error) {
	if db == nil {
		return nil, errors.New("database unavailable")
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS app_network_rules (
		path TEXT PRIMARY KEY, name TEXT NOT NULL, rule_name TEXT NOT NULL,
		blocked INTEGER NOT NULL DEFAULT 1, updated_at TIMESTAMP NOT NULL
	); CREATE TABLE IF NOT EXISTS app_network_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, path TEXT NOT NULL,
		action TEXT NOT NULL, success INTEGER NOT NULL, detail TEXT NOT NULL, created_at TIMESTAMP NOT NULL
	)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS app_network_schedules(id INTEGER PRIMARY KEY AUTOINCREMENT,path TEXT NOT NULL,name TEXT NOT NULL,days TEXT NOT NULL,start_minute INTEGER NOT NULL,end_minute INTEGER NOT NULL,enabled INTEGER NOT NULL DEFAULT 1,active INTEGER NOT NULL DEFAULT 0,updated_at TIMESTAMP NOT NULL)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS app_quarantine(path TEXT PRIMARY KEY,name TEXT NOT NULL,rule_name TEXT NOT NULL,reason TEXT NOT NULL,risk INTEGER NOT NULL,quarantined_at TIMESTAMP NOT NULL)`)
	if err != nil {
		return nil, err
	}
	return &Manager{db: db, platform: p}, nil
}

type QuarantineEntry struct {
	Path          string `json:"path"`
	Name          string `json:"name"`
	Reason        string `json:"reason"`
	Risk          int    `json:"risk"`
	QuarantinedAt string `json:"quarantined_at"`
}

func quarantineRuleName(path string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(path))))
	return "DNSFilter-Quarantine-" + hex.EncodeToString(sum[:8])
}
func (m *Manager) Quarantine(path, reason string, risk int) error {
	clean, err := validatePath(path)
	if err != nil {
		return err
	}
	if risk < 0 || risk > 100 {
		return errors.New("risk must be between 0 and 100")
	}
	if strings.TrimSpace(reason) == "" {
		reason = "Manual security quarantine"
	}
	rule := quarantineRuleName(clean)
	err = m.platform.Block(clean, rule)
	detail := "application isolated by Windows Firewall"
	if err != nil {
		detail = err.Error()
	}
	_, _ = m.db.Exec(`INSERT INTO app_network_events(name,path,action,success,detail,created_at) VALUES(?,?,?,?,?,?)`, filepath.Base(clean), clean, "quarantine", err == nil, detail, time.Now().UTC())
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`INSERT INTO app_quarantine(path,name,rule_name,reason,risk,quarantined_at) VALUES(?,?,?,?,?,?) ON CONFLICT(path) DO UPDATE SET reason=excluded.reason,risk=excluded.risk,quarantined_at=excluded.quarantined_at`, clean, filepath.Base(clean), rule, reason, risk, time.Now().UTC())
	return err
}
func (m *Manager) ReleaseQuarantine(path string) error {
	var clean, name, rule string
	if err := m.db.QueryRow(`SELECT path,name,rule_name FROM app_quarantine WHERE lower(path)=lower(?)`, strings.TrimSpace(strings.Trim(path, `"`))).Scan(&clean, &name, &rule); err != nil {
		return errors.New("application is not quarantined")
	}
	err := m.platform.Allow(rule)
	detail := "application released from quarantine"
	if err != nil {
		detail = err.Error()
	}
	_, _ = m.db.Exec(`INSERT INTO app_network_events(name,path,action,success,detail,created_at) VALUES(?,?,?,?,?,?)`, name, clean, "release", err == nil, detail, time.Now().UTC())
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`DELETE FROM app_quarantine WHERE path=?`, clean)
	return err
}
func (m *Manager) Quarantined() ([]QuarantineEntry, error) {
	rows, err := m.db.Query(`SELECT path,name,reason,risk,quarantined_at FROM app_quarantine ORDER BY quarantined_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QuarantineEntry{}
	for rows.Next() {
		var x QuarantineEntry
		var t time.Time
		if err := rows.Scan(&x.Path, &x.Name, &x.Reason, &x.Risk, &t); err != nil {
			return nil, err
		}
		x.QuarantinedAt = t.Format(time.RFC3339)
		out = append(out, x)
	}
	return out, rows.Err()
}

type Schedule struct {
	ID          int64  `json:"id"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Days        []int  `json:"days"`
	StartMinute int    `json:"start_minute"`
	EndMinute   int    `json:"end_minute"`
	Enabled     bool   `json:"enabled"`
	Active      bool   `json:"active"`
}

func daysText(days []int) string {
	parts := []string{}
	seen := map[int]bool{}
	for _, d := range days {
		if d >= 0 && d <= 6 && !seen[d] {
			parts = append(parts, strconv.Itoa(d))
			seen[d] = true
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
func parseDays(v string) []int {
	out := []int{}
	for _, p := range strings.Split(v, ",") {
		if n, e := strconv.Atoi(p); e == nil {
			out = append(out, n)
		}
	}
	return out
}
func (m *Manager) AddSchedule(s Schedule) error {
	clean, err := validatePath(s.Path)
	if err != nil {
		return err
	}
	if len(s.Days) == 0 || s.StartMinute < 0 || s.StartMinute > 1439 || s.EndMinute < 0 || s.EndMinute > 1439 || s.StartMinute == s.EndMinute {
		return errors.New("valid days and different start/end times are required")
	}
	_, err = m.db.Exec(`INSERT INTO app_network_schedules(path,name,days,start_minute,end_minute,enabled,active,updated_at) VALUES(?,?,?,?,?,1,0,?)`, clean, filepath.Base(clean), daysText(s.Days), s.StartMinute, s.EndMinute, time.Now().UTC())
	if err == nil {
		err = m.EvaluateSchedules(time.Now())
	}
	return err
}
func (m *Manager) Schedules() ([]Schedule, error) {
	rows, err := m.db.Query(`SELECT id,path,name,days,start_minute,end_minute,enabled,active FROM app_network_schedules ORDER BY name,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Schedule{}
	for rows.Next() {
		var s Schedule
		var days string
		if err := rows.Scan(&s.ID, &s.Path, &s.Name, &days, &s.StartMinute, &s.EndMinute, &s.Enabled, &s.Active); err != nil {
			return nil, err
		}
		s.Days = parseDays(days)
		out = append(out, s)
	}
	return out, rows.Err()
}
func (m *Manager) DeleteSchedule(id int64) error {
	var active bool
	m.db.QueryRow(`SELECT active FROM app_network_schedules WHERE id=?`, id).Scan(&active)
	if active {
		_ = m.platform.Allow("DNSFilter-Schedule-" + strconv.FormatInt(id, 10))
	}
	_, err := m.db.Exec(`DELETE FROM app_network_schedules WHERE id=?`, id)
	return err
}
func (m *Manager) EvaluateSchedules(now time.Time) error {
	items, err := m.Schedules()
	if err != nil {
		return err
	}
	minute := now.Hour()*60 + now.Minute()
	day := int(now.Weekday())
	for _, s := range items {
		wanted := false
		if s.Enabled {
			dayOK := false
			for _, d := range s.Days {
				if d == day {
					dayOK = true
				}
			}
			wanted = dayOK && func() bool {
				if s.StartMinute < s.EndMinute {
					return minute >= s.StartMinute && minute < s.EndMinute
				}
				return minute >= s.StartMinute || minute < s.EndMinute
			}()
		}
		if wanted == s.Active {
			continue
		}
		rule := "DNSFilter-Schedule-" + strconv.FormatInt(s.ID, 10)
		if wanted {
			err = m.platform.Block(s.Path, rule)
		} else {
			err = m.platform.Allow(rule)
		}
		if err != nil {
			return err
		}
		_, _ = m.db.Exec(`UPDATE app_network_schedules SET active=?,updated_at=? WHERE id=?`, wanted, time.Now().UTC(), s.ID)
	}
	return nil
}

func (m *Manager) Supported() bool { return m.platform.Supported() }

func validatePath(path string) (string, error) {
	path = strings.TrimSpace(strings.Trim(path, `"`))
	if path == "" {
		return "", errors.New("application path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil || !filepath.IsAbs(abs) || !strings.EqualFold(filepath.Ext(abs), ".exe") {
		return "", errors.New("select a valid Windows .exe file")
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return "", errors.New("application executable was not found")
	}
	base := strings.ToLower(filepath.Base(abs))
	critical := map[string]bool{"svchost.exe": true, "lsass.exe": true, "services.exe": true, "wininit.exe": true, "csrss.exe": true}
	if critical[base] {
		return "", errors.New("this critical Windows process cannot be blocked")
	}
	if self, selfErr := os.Executable(); selfErr == nil && strings.EqualFold(filepath.Clean(self), filepath.Clean(abs)) {
		return "", errors.New("DNS Filter cannot block its own network connection")
	}
	return filepath.Clean(abs), nil
}

func ruleName(path string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(path))))
	return "DNSFilter-App-" + hex.EncodeToString(sum[:8])
}

func (m *Manager) SetBlocked(path string, blocked bool) error {
	clean, err := validatePath(path)
	if err != nil {
		return err
	}
	name, rule := filepath.Base(clean), ruleName(clean)
	action := "allow"
	if blocked {
		action = "block"
		err = m.platform.Block(clean, rule)
	} else {
		err = m.platform.Allow(rule)
	}
	detail := "Windows Firewall rule updated"
	if err != nil {
		detail = err.Error()
	}
	_, _ = m.db.Exec(`INSERT INTO app_network_events(name,path,action,success,detail,created_at) VALUES(?,?,?,?,?,?)`, name, clean, action, err == nil, detail, time.Now().UTC())
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`INSERT INTO app_network_rules(path,name,rule_name,blocked,updated_at) VALUES(?,?,?,?,?)
		ON CONFLICT(path) DO UPDATE SET name=excluded.name,rule_name=excluded.rule_name,blocked=excluded.blocked,updated_at=excluded.updated_at`, clean, name, rule, blocked, time.Now().UTC())
	return err
}

func (m *Manager) Apps() ([]App, error) {
	running, runErr := m.platform.RunningApps()
	byPath := map[string]App{}
	for _, app := range running {
		if app.Path == "" {
			continue
		}
		app.Running = true
		byPath[strings.ToLower(filepath.Clean(app.Path))] = app
	}
	rows, err := m.db.Query(`SELECT path,name,rule_name,blocked,updated_at FROM app_network_rules ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var path, name, rule string
		var blocked bool
		var updated time.Time
		if err := rows.Scan(&path, &name, &rule, &blocked, &updated); err != nil {
			return nil, err
		}
		key := strings.ToLower(filepath.Clean(path))
		app := byPath[key]
		if app.Path == "" {
			app = App{Name: name, Path: path}
		}
		app.Blocked, app.RuleName, app.UpdatedAt = blocked, rule, updated.Format(time.RFC3339)
		byPath[key] = app
	}
	out := make([]App, 0, len(byPath))
	for _, app := range byPath {
		out = append(out, app)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Blocked != out[j].Blocked {
			return out[i].Blocked
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	if len(out) == 0 && runErr != nil {
		return nil, runErr
	}
	return out, nil
}

func (m *Manager) Events(limit int) ([]Event, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := m.db.Query(`SELECT id,name,path,action,success,detail,created_at FROM app_network_events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var event Event
		var created time.Time
		if err := rows.Scan(&event.ID, &event.Name, &event.Path, &event.Action, &event.Success, &event.Detail, &created); err != nil {
			return nil, err
		}
		event.CreatedAt = created.Format(time.RFC3339)
		out = append(out, event)
	}
	return out, rows.Err()
}
