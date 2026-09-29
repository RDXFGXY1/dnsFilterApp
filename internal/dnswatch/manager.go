package dnswatch

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/netdns"
)

type Event struct {
	ID        int64     `json:"id"`
	Interface string    `json:"interface"`
	DNS       string    `json:"dns"`
	Protected bool      `json:"protected"`
	Error     string    `json:"error,omitempty"`
	ChangedAt time.Time `json:"changed_at"`
}

type Manager struct {
	db        *sql.DB
	checkMu   sync.Mutex
	mu        sync.RWMutex
	last      Event
	repair    bool
	status    func() (string, string, bool, error)
	setDNS    func() error
	stop      chan struct{}
	closeOnce sync.Once
}

func New(db *sql.DB) (*Manager, error) {
	return NewWithAutoRepair(db, false)
}

func NewWithAutoRepair(db *sql.DB, repair bool) (*Manager, error) {
	return newManager(db, repair, netdns.Status, netdns.SetDNS, 30*time.Second)
}

func newManager(db *sql.DB, repair bool, status func() (string, string, bool, error), setDNS func() error, interval time.Duration) (*Manager, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS dns_route_events(
		id INTEGER PRIMARY KEY AUTOINCREMENT,interface TEXT NOT NULL,dns TEXT NOT NULL,
		protected INTEGER NOT NULL,error TEXT NOT NULL DEFAULT '',changed_at TIMESTAMP NOT NULL)`)
	if err != nil {
		return nil, err
	}
	m := &Manager{db: db, repair: repair, status: status, setDNS: setDNS, stop: make(chan struct{})}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	_ = db.QueryRow(`SELECT id,interface,dns,protected,error,changed_at FROM dns_route_events ORDER BY id DESC LIMIT 1`).Scan(&m.last.ID, &m.last.Interface, &m.last.DNS, &m.last.Protected, &m.last.Error, &m.last.ChangedAt)
	_ = m.Check()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = m.Check()
			case <-m.stop:
				return
			}
		}
	}()
	return m, nil
}

func signature(e Event) string {
	return strings.Join([]string{e.Interface, e.DNS, e.Error, map[bool]string{true: "1", false: "0"}[e.Protected]}, "|")
}

func (m *Manager) Check() error {
	m.checkMu.Lock()
	defer m.checkMu.Unlock()
	iface, dns, active, err := m.status()
	m.mu.RLock()
	repair := m.repair
	m.mu.RUnlock()
	if err == nil && !active && repair {
		if repairErr := m.setDNS(); repairErr != nil {
			err = repairErr
		} else {
			iface, dns, active, err = m.status()
		}
	}
	event := Event{Interface: iface, DNS: dns, Protected: active, ChangedAt: time.Now().UTC()}
	if err != nil {
		event.Error = err.Error()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if signature(event) == signature(m.last) {
		return err
	}
	result, insertErr := m.db.Exec(`INSERT INTO dns_route_events(interface,dns,protected,error,changed_at) VALUES(?,?,?,?,?)`, event.Interface, event.DNS, event.Protected, event.Error, event.ChangedAt)
	if insertErr != nil {
		return insertErr
	}
	event.ID, _ = result.LastInsertId()
	m.last = event
	_, _ = m.db.Exec(`DELETE FROM dns_route_events WHERE id NOT IN (SELECT id FROM dns_route_events ORDER BY id DESC LIMIT 200)`)
	return err
}

func (m *Manager) Current() Event { m.mu.RLock(); defer m.mu.RUnlock(); return m.last }

func (m *Manager) SetAutoRepair(enabled bool) {
	m.mu.Lock()
	m.repair = enabled
	m.mu.Unlock()
}

func (m *Manager) AutoRepair() bool { m.mu.RLock(); defer m.mu.RUnlock(); return m.repair }

func (m *Manager) Close() { m.closeOnce.Do(func() { close(m.stop) }) }

func (m *Manager) Events() ([]Event, error) {
	rows, err := m.db.Query(`SELECT id,interface,dns,protected,error,changed_at FROM dns_route_events ORDER BY id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Interface, &e.DNS, &e.Protected, &e.Error, &e.ChangedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
