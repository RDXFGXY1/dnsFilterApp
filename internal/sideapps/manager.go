package sideapps

import (
	"database/sql"
	"encoding/json"
	"errors"
)

type Manager struct{ db *sql.DB }
type App struct {
	Manifest    Manifest `json:"manifest"`
	SHA256      string   `json:"sha256"`
	Enabled     bool     `json:"enabled"`
	InstalledAt string   `json:"installed_at"`
}

func New(db *sql.DB) (*Manager, error) {
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS side_app_packages(id TEXT PRIMARY KEY,manifest TEXT NOT NULL,sha256 TEXT NOT NULL,archive BLOB NOT NULL,enabled INTEGER NOT NULL DEFAULT 1,installed_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS side_app_storage(app_id TEXT PRIMARY KEY,document TEXT NOT NULL)`,
	} {
		if _, e := db.Exec(q); e != nil {
			return nil, e
		}
	}
	return &Manager{db: db}, nil
}
func (m *Manager) List() ([]App, error) {
	rows, e := m.db.Query(`SELECT manifest,sha256,enabled,installed_at FROM side_app_packages ORDER BY installed_at,id`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []App{}
	for rows.Next() {
		var a App
		var raw string
		if e = rows.Scan(&raw, &a.SHA256, &a.Enabled, &a.InstalledAt); e != nil {
			return nil, e
		}
		if e = json.Unmarshal([]byte(raw), &a.Manifest); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (m *Manager) Install(raw []byte, digest string, replace bool) (*Package, error) {
	p, e := Parse(raw)
	if e != nil {
		return nil, e
	}
	if digest == "" || digest != p.SHA256 {
		return nil, errors.New("package changed or was not reviewed; review it again before installing")
	}
	manifest, e := json.Marshal(p.Manifest)
	if e != nil {
		return nil, e
	}
	tx, e := m.db.Begin()
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	var n int
	if e = tx.QueryRow(`SELECT COUNT(*) FROM side_app_packages WHERE id=?`, p.Manifest.ID).Scan(&n); e != nil {
		return nil, e
	}
	if n > 0 && !replace {
		return nil, ErrConflict
	}
	_, e = tx.Exec(`INSERT INTO side_app_packages(id,manifest,sha256,archive) VALUES(?,?,?,?) ON CONFLICT(id) DO UPDATE SET manifest=excluded.manifest,sha256=excluded.sha256,archive=excluded.archive,enabled=1,installed_at=CURRENT_TIMESTAMP`, p.Manifest.ID, string(manifest), p.SHA256, raw)
	if e != nil {
		return nil, e
	}
	return p, tx.Commit()
}
func (m *Manager) SetEnabled(id string, enabled bool) error {
	r, e := m.db.Exec(`UPDATE side_app_packages SET enabled=? WHERE id=?`, enabled, id)
	return changed(r, e)
}
func changed(r sql.Result, e error) error {
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (m *Manager) Remove(id string) error {
	tx, e := m.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	r, e := tx.Exec(`DELETE FROM side_app_packages WHERE id=?`, id)
	if e = changed(r, e); e != nil {
		return e
	}
	if _, e = tx.Exec(`DELETE FROM side_app_storage WHERE app_id=?`, id); e != nil {
		return e
	}
	return tx.Commit()
}
func (m *Manager) Package(id string) (*Package, error) {
	var raw []byte
	var enabled bool
	if e := m.db.QueryRow(`SELECT archive,enabled FROM side_app_packages WHERE id=?`, id).Scan(&raw, &enabled); e != nil {
		return nil, e
	}
	if !enabled {
		return nil, ErrForbidden
	}
	return Parse(raw)
}
func storageAllowed(tx *sql.Tx, id string) error {
	var raw string
	var enabled bool
	if e := tx.QueryRow(`SELECT manifest,enabled FROM side_app_packages WHERE id=?`, id).Scan(&raw, &enabled); e != nil {
		return e
	}
	var a Manifest
	if e := json.Unmarshal([]byte(raw), &a); e != nil {
		return e
	}
	if enabled {
		for _, permission := range a.Permissions {
			if permission == "storage" {
				return nil
			}
		}
	}
	return ErrForbidden
}
func (m *Manager) ReadStorage(id string) (json.RawMessage, error) {
	tx, e := m.db.Begin()
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if e = storageAllowed(tx, id); e != nil {
		return nil, e
	}
	var raw string
	e = tx.QueryRow(`SELECT document FROM side_app_storage WHERE app_id=?`, id).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return json.RawMessage("null"), nil
	}
	if e != nil {
		return nil, e
	}
	return json.RawMessage(raw), nil
}
func (m *Manager) WriteStorage(id string, raw json.RawMessage) error {
	if len(raw) > MaxStorageBytes || !json.Valid(raw) {
		return errors.New("storage must be valid JSON up to 1 MB")
	}
	tx, e := m.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = storageAllowed(tx, id); e != nil {
		return e
	}
	if _, e = tx.Exec(`INSERT INTO side_app_storage(app_id,document) VALUES(?,?) ON CONFLICT(app_id) DO UPDATE SET document=excluded.document`, id, string(raw)); e != nil {
		return e
	}
	return tx.Commit()
}
