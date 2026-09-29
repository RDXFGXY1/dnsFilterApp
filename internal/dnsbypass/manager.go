package dnsbypass

import (
	"database/sql"
	"errors"
	"time"
)

type Platform interface {
	Enable() error
	Disable() error
	Active() (bool, error)
	Supported() bool
}

type Manager struct {
	db       *sql.DB
	platform Platform
}

type Status struct {
	Supported      bool     `json:"supported"`
	Enabled        bool     `json:"enabled"`
	Enforced       bool     `json:"enforced"`
	ProtectedPorts []string `json:"protected_ports"`
	UpdatedAt      string   `json:"updated_at,omitempty"`
}

func New(db *sql.DB) (*Manager, error) { return newWithPlatform(db, newPlatform()) }

func newWithPlatform(db *sql.DB, p Platform) (*Manager, error) {
	if db == nil {
		return nil, errors.New("database unavailable")
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS dns_bypass_settings(id INTEGER PRIMARY KEY CHECK(id=1),enabled INTEGER NOT NULL DEFAULT 0,updated_at TIMESTAMP NOT NULL);
		CREATE TABLE IF NOT EXISTS dns_bypass_events(id INTEGER PRIMARY KEY AUTOINCREMENT,action TEXT NOT NULL,success INTEGER NOT NULL,detail TEXT NOT NULL,created_at TIMESTAMP NOT NULL);
		INSERT OR IGNORE INTO dns_bypass_settings(id,enabled,updated_at) VALUES(1,0,CURRENT_TIMESTAMP)`)
	if err != nil {
		return nil, err
	}
	return &Manager{db: db, platform: p}, nil
}

func (m *Manager) Status() (Status, error) {
	var enabled bool
	var updated time.Time
	if err := m.db.QueryRow(`SELECT enabled,updated_at FROM dns_bypass_settings WHERE id=1`).Scan(&enabled, &updated); err != nil {
		return Status{}, err
	}
	enforced, err := m.platform.Active()
	if err != nil && enabled {
		return Status{}, err
	}
	return Status{Supported: m.platform.Supported(), Enabled: enabled, Enforced: enforced, ProtectedPorts: []string{"UDP 53", "TCP 53"}, UpdatedAt: updated.Format(time.RFC3339)}, nil
}

func (m *Manager) SetEnabled(enabled bool) error {
	if !m.platform.Supported() {
		return errors.New("DNS bypass protection requires Windows")
	}
	var err error
	action := "disable"
	if enabled {
		action = "enable"
		err = m.platform.Enable()
	} else {
		err = m.platform.Disable()
	}
	detail := "direct plaintext DNS firewall rules updated"
	if err != nil {
		detail = err.Error()
	}
	_, _ = m.db.Exec(`INSERT INTO dns_bypass_events(action,success,detail,created_at) VALUES(?,?,?,?)`, action, err == nil, detail, time.Now().UTC())
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`UPDATE dns_bypass_settings SET enabled=?,updated_at=? WHERE id=1`, enabled, time.Now().UTC())
	return err
}
