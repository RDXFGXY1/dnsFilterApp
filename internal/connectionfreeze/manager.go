// Package connectionfreeze manages temporary, fail-open connection freezing.
package connectionfreeze

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

type Event struct {
	ID       int64  `json:"id"`
	Time     string `json:"time"`
	Action   string `json:"action"`
	App      string `json:"app"`
	Local    string `json:"local"`
	Remote   string `json:"remote"`
	Protocol string `json:"protocol"`
	Reason   string `json:"reason"`
}
type Status struct {
	Supported bool      `json:"supported"`
	Active    bool      `json:"active"`
	Manual    bool      `json:"manual"`
	Until     time.Time `json:"until"`
	Error     string    `json:"error,omitempty"`
}
type backend interface {
	Close() error
	Events() ([]Event, error)
}
type Manager struct {
	mu         sync.Mutex
	db         *sql.DB
	session    backend
	until      time.Time
	lastError  string
	timer      *time.Timer
	done       chan struct{}
	closed     bool
	generation uint64
	start      func() (backend, error)
}

func New(db *sql.DB) (*Manager, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS connection_freeze_events (
 id INTEGER PRIMARY KEY AUTOINCREMENT, time TEXT NOT NULL, action TEXT NOT NULL,
 app TEXT NOT NULL, local TEXT NOT NULL, remote TEXT NOT NULL, protocol TEXT NOT NULL, reason TEXT NOT NULL)`)
	if err != nil {
		return nil, err
	}
	m := &Manager{db: db, done: make(chan struct{}), start: startPlatform}
	go m.poll()
	return m, nil
}
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Status{Supported: supported(), Active: m.session != nil, Manual: m.session != nil && m.until.IsZero(), Until: m.until, Error: m.lastError}
}
func (m *Manager) Start(minutes int) error {
	if minutes < 0 || minutes > 120 {
		return fmt.Errorf("choose 0 for manual stop, or a duration between 1 and 120 minutes")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return fmt.Errorf("connection freeze is shutting down")
	}
	if m.session != nil {
		return fmt.Errorf("connection freeze is already active")
	}
	b, err := m.start()
	if err != nil {
		m.lastError = err.Error()
		return err
	}
	m.session = b
	m.lastError = ""
	m.until = time.Time{}
	m.generation++
	generation := m.generation
	duration := "until manually stopped (or server exits)"
	if minutes > 0 {
		m.until = time.Now().Add(time.Duration(minutes) * time.Minute)
		m.timer = time.AfterFunc(time.Until(m.until), func() { m.expire(generation) })
		duration = fmt.Sprintf("for %d minutes", minutes)
	}
	m.record(Event{Action: "enabled", Reason: "New external connections frozen " + duration + "; loopback, existing flows, DNS Filter and DHCP remain available"})
	return nil
}

func (m *Manager) expire(generation uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// A callback already queued by a previous session cannot stop a new one.
	if m.session != nil && !m.until.IsZero() && m.generation == generation {
		_ = m.stopLocked("Timer expired")
	}
}
func (m *Manager) Stop(reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked(reason)
}
func (m *Manager) stopLocked(reason string) error {
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
	if m.session == nil {
		return nil
	}
	m.collect()
	if err := m.session.Close(); err != nil {
		m.lastError = err.Error()
		return err
	}
	m.session = nil
	m.until = time.Time{}
	m.record(Event{Action: "disabled", Reason: reason})
	return nil
}
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.done)
	}
	return m.stopLocked("Server shutting down")
}
func (m *Manager) poll() {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-m.done:
			return
		case <-tick.C:
			m.mu.Lock()
			if m.session != nil {
				m.collect()
			}
			m.mu.Unlock()
		}
	}
}
func (m *Manager) collect() {
	events, err := m.session.Events()
	if err != nil {
		m.lastError = "Connection log unavailable: " + err.Error()
		return
	}
	for _, e := range events {
		m.record(e)
	}
}
func (m *Manager) record(e Event) {
	if e.Time == "" {
		e.Time = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if _, err := m.db.Exec(`INSERT INTO connection_freeze_events(time,action,app,local,remote,protocol,reason) VALUES(?,?,?,?,?,?,?)`, e.Time, e.Action, e.App, e.Local, e.Remote, e.Protocol, e.Reason); err != nil {
		m.lastError = "Could not save connection log: " + err.Error()
		return
	}
	_, _ = m.db.Exec(`DELETE FROM connection_freeze_events WHERE id <= (SELECT MAX(id)-1000 FROM connection_freeze_events)`)
}
func (m *Manager) Events() ([]Event, error) {
	rows, err := m.db.Query(`SELECT id,time,action,app,local,remote,protocol,reason FROM connection_freeze_events ORDER BY id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Time, &e.Action, &e.App, &e.Local, &e.Remote, &e.Protocol, &e.Reason); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
