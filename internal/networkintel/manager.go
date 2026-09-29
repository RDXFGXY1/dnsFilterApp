package networkintel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Connection struct {
	ID          int64     `json:"id"`
	Application string    `json:"application"`
	Path        string    `json:"path"`
	RemoteIP    string    `json:"remote_ip"`
	RemotePort  int       `json:"remote_port"`
	Protocol    string    `json:"protocol"`
	Risk        int       `json:"risk"`
	Reasons     string    `json:"reasons"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Seen        int       `json:"seen"`
	Trusted     bool      `json:"trusted"`
	Hostname    string    `json:"hostname,omitempty"`
}
type platform interface {
	Connections() ([]Connection, error)
	Supported() bool
}
type Manager struct {
	db             *sql.DB
	platform       platform
	nameMu         sync.Mutex
	destinationDNS map[string]string
	resolvingDNS   map[string]bool
}

func New(db *sql.DB) (*Manager, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS network_connection_timeline(id INTEGER PRIMARY KEY AUTOINCREMENT,application TEXT NOT NULL,path TEXT NOT NULL,remote_ip TEXT NOT NULL,remote_port INTEGER NOT NULL,protocol TEXT NOT NULL,risk INTEGER NOT NULL,reasons TEXT NOT NULL,first_seen TIMESTAMP NOT NULL,last_seen TIMESTAMP NOT NULL,seen INTEGER NOT NULL DEFAULT 1,UNIQUE(path,remote_ip,remote_port,protocol))`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS network_observed_apps(path TEXT PRIMARY KEY,application TEXT NOT NULL,first_seen TIMESTAMP NOT NULL,last_seen TIMESTAMP NOT NULL,reviewed INTEGER NOT NULL DEFAULT 0)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS network_learning_sessions(id INTEGER PRIMARY KEY AUTOINCREMENT,path TEXT NOT NULL,application TEXT NOT NULL,started_at TIMESTAMP NOT NULL,ends_at TIMESTAMP NOT NULL,stopped_at TIMESTAMP)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS network_learning_observations(
		session_id INTEGER NOT NULL,remote_ip TEXT NOT NULL,remote_port INTEGER NOT NULL,protocol TEXT NOT NULL,
		risk INTEGER NOT NULL,reasons TEXT NOT NULL,first_seen TIMESTAMP NOT NULL,last_seen TIMESTAMP NOT NULL,seen INTEGER NOT NULL DEFAULT 1,
		PRIMARY KEY(session_id,remote_ip,remote_port,protocol),
		FOREIGN KEY(session_id) REFERENCES network_learning_sessions(id) ON DELETE CASCADE)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS network_destination_baseline(
		path TEXT NOT NULL,remote_ip TEXT NOT NULL,remote_port INTEGER NOT NULL,protocol TEXT NOT NULL,trusted_at TIMESTAMP NOT NULL,
		PRIMARY KEY(path,remote_ip,remote_port,protocol))`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS network_intelligence_settings(
		id INTEGER PRIMARY KEY CHECK(id=1),retention_days INTEGER NOT NULL DEFAULT 7)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`INSERT OR IGNORE INTO network_intelligence_settings(id,retention_days) VALUES(1,7)`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS network_notification_settings(
		id INTEGER PRIMARY KEY CHECK(id=1),enabled INTEGER NOT NULL DEFAULT 1,min_risk INTEGER NOT NULL DEFAULT 20,applications TEXT NOT NULL DEFAULT '[]')`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`INSERT OR IGNORE INTO network_notification_settings(id,enabled,min_risk,applications) VALUES(1,1,20,'[]')`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS network_automatic_response_settings(
		id INTEGER PRIMARY KEY CHECK(id=1),enabled INTEGER NOT NULL DEFAULT 0,min_risk INTEGER NOT NULL DEFAULT 35,action TEXT NOT NULL DEFAULT 'review')`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`INSERT OR IGNORE INTO network_automatic_response_settings(id,enabled,min_risk,action) VALUES(1,0,35,'review')`)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS network_automatic_response_events(
		id INTEGER PRIMARY KEY AUTOINCREMENT,path TEXT NOT NULL,remote_ip TEXT NOT NULL,remote_port INTEGER NOT NULL,protocol TEXT NOT NULL,risk INTEGER NOT NULL,action TEXT NOT NULL,created_at TIMESTAMP NOT NULL,
		UNIQUE(path,remote_ip,remote_port,protocol))`)
	if err != nil {
		return nil, err
	}
	m := &Manager{
		db:             db,
		platform:       newPlatform(),
		destinationDNS: map[string]string{},
		resolvingDNS:   map[string]bool{},
	}
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_ = m.Refresh()
		}
	}()
	return m, nil
}
func (m *Manager) Supported() bool { return m.platform.Supported() }
func score(c Connection) (int, string) {
	risk := 0
	why := []string{}
	ip := net.ParseIP(c.RemoteIP)
	if ip != nil && !ip.IsPrivate() && !ip.IsLoopback() {
		risk += 10
	}
	if c.RemotePort != 80 && c.RemotePort != 443 && c.RemotePort != 53 {
		risk += 15
		why = append(why, "uncommon remote port")
	}
	lower := strings.ToLower(c.Path)
	if lower != "" && !strings.Contains(lower, `\program files\`) && !strings.Contains(lower, `\windows\`) {
		risk += 20
		why = append(why, "application runs outside trusted system folders")
	}
	if c.Application == "" || c.Path == "" {
		risk += 20
		why = append(why, "process identity is incomplete")
	}
	if len(why) == 0 {
		why = append(why, "no strong local risk signals")
	}
	if risk > 100 {
		risk = 100
	}
	return risk, strings.Join(why, " · ")
}
func (m *Manager) Refresh() error {
	items, err := m.platform.Connections()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, c := range items {
		c.Application = filepath.Base(c.Path)
		if c.Application == "." || c.Application == "" {
			c.Application = "Unknown process"
		}
		c.Risk, c.Reasons = score(c)
		_, _ = m.db.Exec(`INSERT INTO network_connection_timeline(application,path,remote_ip,remote_port,protocol,risk,reasons,first_seen,last_seen,seen) VALUES(?,?,?,?,?,?,?,?,?,1) ON CONFLICT(path,remote_ip,remote_port,protocol) DO UPDATE SET application=excluded.application,risk=excluded.risk,reasons=excluded.reasons,last_seen=excluded.last_seen,seen=seen+1`, c.Application, c.Path, c.RemoteIP, c.RemotePort, c.Protocol, c.Risk, c.Reasons, now, now)
		if c.Path != "" {
			_, _ = m.db.Exec(`INSERT INTO network_observed_apps(path,application,first_seen,last_seen,reviewed) VALUES(?,?,?,?,0) ON CONFLICT(path) DO UPDATE SET application=excluded.application,last_seen=excluded.last_seen`, c.Path, c.Application, now, now)
			_, _ = m.db.Exec(`INSERT INTO network_learning_observations(session_id,remote_ip,remote_port,protocol,risk,reasons,first_seen,last_seen,seen)
				SELECT id,?,?,?,?,?,?,?,1 FROM network_learning_sessions
				WHERE path=? AND stopped_at IS NULL AND started_at<=? AND ends_at>=?
				ON CONFLICT(session_id,remote_ip,remote_port,protocol) DO UPDATE SET risk=excluded.risk,reasons=excluded.reasons,last_seen=excluded.last_seen,seen=seen+1`,
				c.RemoteIP, c.RemotePort, c.Protocol, c.Risk, c.Reasons, now, now, c.Path, now, now)
		}
	}
	retentionDays, _ := m.RetentionDays()
	_, _ = m.db.Exec(`DELETE FROM network_connection_timeline WHERE last_seen < ?`, now.Add(-time.Duration(retentionDays)*24*time.Hour))
	return nil
}

type AppAlert struct {
	Path        string    `json:"path"`
	Application string    `json:"application"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Reviewed    bool      `json:"reviewed"`
}
type PrivacyProfile struct {
	Path         string `json:"path"`
	Application  string `json:"application"`
	Destinations int    `json:"destinations"`
	Observations int    `json:"observations"`
	AverageRisk  int    `json:"average_risk"`
	PrivacyScore int    `json:"privacy_score"`
	Level        string `json:"level"`
}

func (m *Manager) Alerts() ([]AppAlert, error) {
	rows, err := m.db.Query(`SELECT path,application,first_seen,last_seen,reviewed FROM network_observed_apps WHERE reviewed=0 ORDER BY first_seen DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AppAlert{}
	for rows.Next() {
		var x AppAlert
		if err := rows.Scan(&x.Path, &x.Application, &x.FirstSeen, &x.LastSeen, &x.Reviewed); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (m *Manager) Review(path string) error {
	_, err := m.db.Exec(`UPDATE network_observed_apps SET reviewed=1 WHERE path=?`, path)
	return err
}
func (m *Manager) Privacy() ([]PrivacyProfile, error) {
	rows, err := m.db.Query(`SELECT path,application,COUNT(DISTINCT remote_ip),SUM(seen),CAST(AVG(risk) AS INTEGER) FROM network_connection_timeline WHERE path<>'' GROUP BY path,application ORDER BY COUNT(DISTINCT remote_ip) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PrivacyProfile{}
	for rows.Next() {
		var x PrivacyProfile
		if err := rows.Scan(&x.Path, &x.Application, &x.Destinations, &x.Observations, &x.AverageRisk); err != nil {
			return nil, err
		}
		x.PrivacyScore = x.Destinations*4 + x.Observations/10 + x.AverageRisk
		if !strings.Contains(strings.ToLower(x.Path), `\program files\`) && !strings.Contains(strings.ToLower(x.Path), `\windows\`) {
			x.PrivacyScore += 15
		}
		if x.PrivacyScore > 100 {
			x.PrivacyScore = 100
		}
		x.Level = "Low"
		if x.PrivacyScore >= 65 {
			x.Level = "High"
		} else if x.PrivacyScore >= 35 {
			x.Level = "Review"
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (m *Manager) AppConnections(path string, limit int) ([]Connection, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("application path required")
	}
	if limit < 1 || limit > 500 {
		limit = 200
	}
	rows, err := m.db.Query(`SELECT t.id,t.application,t.path,t.remote_ip,t.remote_port,t.protocol,t.risk,t.reasons,t.first_seen,t.last_seen,t.seen,
		CASE WHEN b.path IS NULL THEN 0 ELSE 1 END
		FROM network_connection_timeline t LEFT JOIN network_destination_baseline b
		ON b.path=t.path AND b.remote_ip=t.remote_ip AND b.remote_port=t.remote_port AND b.protocol=t.protocol
		WHERE t.path=? ORDER BY t.last_seen DESC,t.seen DESC LIMIT ?`, path, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Connection{}
	for rows.Next() {
		var item Connection
		if err := rows.Scan(&item.ID, &item.Application, &item.Path, &item.RemoteIP, &item.RemotePort, &item.Protocol, &item.Risk, &item.Reasons, &item.FirstSeen, &item.LastSeen, &item.Seen, &item.Trusted); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *Manager) TrustCurrentDestinations(path string) (int64, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, errors.New("application path required")
	}
	result, err := m.db.Exec(`INSERT OR IGNORE INTO network_destination_baseline(path,remote_ip,remote_port,protocol,trusted_at)
		SELECT path,remote_ip,remote_port,protocol,? FROM network_connection_timeline WHERE path=?`, time.Now().UTC(), path)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (m *Manager) TrustDestination(path, remoteIP string, remotePort int, protocol string) error {
	path = strings.TrimSpace(path)
	remoteIP = strings.TrimSpace(remoteIP)
	protocol = strings.ToUpper(strings.TrimSpace(protocol))
	if path == "" {
		return errors.New("application path required")
	}
	if net.ParseIP(remoteIP) == nil {
		return errors.New("valid destination IP required")
	}
	if remotePort < 1 || remotePort > 65535 {
		return errors.New("valid destination port required")
	}
	if protocol == "" {
		return errors.New("destination protocol required")
	}
	var exists int
	if err := m.db.QueryRow(`SELECT COUNT(1) FROM network_connection_timeline
		WHERE path=? AND remote_ip=? AND remote_port=? AND UPPER(protocol)=?`, path, remoteIP, remotePort, protocol).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return errors.New("destination is not present in the observed connection timeline")
	}
	_, err := m.db.Exec(`INSERT OR IGNORE INTO network_destination_baseline(path,remote_ip,remote_port,protocol,trusted_at)
		VALUES(?,?,?,?,?)`, path, remoteIP, remotePort, protocol, time.Now().UTC())
	return err
}

func (m *Manager) NewDestinations(limit int) ([]Connection, error) {
	if limit < 1 || limit > 500 {
		limit = 200
	}
	rows, err := m.db.Query(`SELECT t.id,t.application,t.path,t.remote_ip,t.remote_port,t.protocol,t.risk,t.reasons,t.first_seen,t.last_seen,t.seen
		FROM network_connection_timeline t
		WHERE EXISTS(SELECT 1 FROM network_destination_baseline established WHERE established.path=t.path)
		AND NOT EXISTS(SELECT 1 FROM network_destination_baseline known WHERE known.path=t.path AND known.remote_ip=t.remote_ip AND known.remote_port=t.remote_port AND known.protocol=t.protocol)
		ORDER BY t.last_seen DESC,t.risk DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Connection{}
	for rows.Next() {
		var item Connection
		if err := rows.Scan(&item.ID, &item.Application, &item.Path, &item.RemoteIP, &item.RemotePort, &item.Protocol, &item.Risk, &item.Reasons, &item.FirstSeen, &item.LastSeen, &item.Seen); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	m.decorateAndResolve(items)
	return items, nil
}

type HistoryStatus struct {
	RetentionDays int   `json:"retention_days"`
	Connections   int64 `json:"connections"`
	Applications  int64 `json:"applications"`
}

type NotificationSettings struct {
	Enabled      bool     `json:"enabled"`
	MinRisk      int      `json:"min_risk"`
	Applications []string `json:"applications"`
}

type AutomaticResponseSettings struct {
	Enabled bool   `json:"enabled"`
	MinRisk int    `json:"min_risk"`
	Action  string `json:"action"`
}

type AutomaticResponseEvent struct {
	ID          int64     `json:"id"`
	Path        string    `json:"path"`
	Application string    `json:"application"`
	RemoteIP    string    `json:"remote_ip"`
	RemotePort  int       `json:"remote_port"`
	Protocol    string    `json:"protocol"`
	Risk        int       `json:"risk"`
	Action      string    `json:"action"`
	CreatedAt   time.Time `json:"created_at"`
}

func (m *Manager) AutomaticResponseSettings() (AutomaticResponseSettings, error) {
	var settings AutomaticResponseSettings
	if err := m.db.QueryRow(`SELECT enabled,min_risk,action FROM network_automatic_response_settings WHERE id=1`).Scan(&settings.Enabled, &settings.MinRisk, &settings.Action); err != nil {
		return AutomaticResponseSettings{}, err
	}
	return settings, nil
}

func (m *Manager) SetAutomaticResponseSettings(settings AutomaticResponseSettings) error {
	settings.Action = strings.ToLower(strings.TrimSpace(settings.Action))
	if settings.MinRisk < 0 || settings.MinRisk > 100 {
		return errors.New("minimum risk must be between 0 and 100")
	}
	if settings.Action != "review" && settings.Action != "quarantine" && settings.Action != "block" {
		return errors.New("automatic action must be review, quarantine, or block")
	}
	_, err := m.db.Exec(`UPDATE network_automatic_response_settings SET enabled=?,min_risk=?,action=? WHERE id=1`, settings.Enabled, settings.MinRisk, settings.Action)
	return err
}

func (m *Manager) AutomaticCandidates(minRisk int, limit int) ([]Connection, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}
	rows, err := m.db.Query(`SELECT t.id,t.application,t.path,t.remote_ip,t.remote_port,t.protocol,t.risk,t.reasons,t.first_seen,t.last_seen,t.seen
		FROM network_connection_timeline t
		WHERE t.risk>=? AND EXISTS(SELECT 1 FROM network_destination_baseline b WHERE b.path=t.path)
		AND NOT EXISTS(SELECT 1 FROM network_destination_baseline b WHERE b.path=t.path AND b.remote_ip=t.remote_ip AND b.remote_port=t.remote_port AND b.protocol=t.protocol)
		AND NOT EXISTS(SELECT 1 FROM network_automatic_response_events e WHERE e.path=t.path AND e.remote_ip=t.remote_ip AND e.remote_port=t.remote_port AND e.protocol=t.protocol)
		ORDER BY t.risk DESC,t.last_seen DESC LIMIT ?`, minRisk, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Connection{}
	for rows.Next() {
		var item Connection
		if err := rows.Scan(&item.ID, &item.Application, &item.Path, &item.RemoteIP, &item.RemotePort, &item.Protocol, &item.Risk, &item.Reasons, &item.FirstSeen, &item.LastSeen, &item.Seen); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *Manager) RecordAutomaticResponse(item Connection, action string) error {
	_, err := m.db.Exec(`INSERT OR IGNORE INTO network_automatic_response_events(path,remote_ip,remote_port,protocol,risk,action,created_at) VALUES(?,?,?,?,?,?,?)`, item.Path, item.RemoteIP, item.RemotePort, item.Protocol, item.Risk, action, time.Now().UTC())
	return err
}

func (m *Manager) AutomaticResponseEvents(limit int) ([]AutomaticResponseEvent, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := m.db.Query(`SELECT e.id,e.path,COALESCE(a.application,''),e.remote_ip,e.remote_port,e.protocol,e.risk,e.action,e.created_at
		FROM network_automatic_response_events e LEFT JOIN network_observed_apps a ON a.path=e.path ORDER BY e.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AutomaticResponseEvent{}
	for rows.Next() {
		var item AutomaticResponseEvent
		if err := rows.Scan(&item.ID, &item.Path, &item.Application, &item.RemoteIP, &item.RemotePort, &item.Protocol, &item.Risk, &item.Action, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type AnalyticsItem struct {
	Label        string `json:"label"`
	Destinations int    `json:"destinations"`
	Observations int    `json:"observations"`
	MaxRisk      int    `json:"max_risk"`
}

type SecurityAnalytics struct {
	Applications int             `json:"applications"`
	Destinations int             `json:"destinations"`
	Observations int             `json:"observations"`
	HighRisk     int             `json:"high_risk"`
	TopApps      []AnalyticsItem `json:"top_apps"`
	TopTargets   []AnalyticsItem `json:"top_targets"`
}

func (m *Manager) SecurityAnalytics() (SecurityAnalytics, error) {
	var result SecurityAnalytics
	if err := m.db.QueryRow(`SELECT COUNT(DISTINCT path),COUNT(*),COALESCE(SUM(seen),0),COALESCE(SUM(CASE WHEN risk>=35 THEN 1 ELSE 0 END),0) FROM network_connection_timeline`).Scan(&result.Applications, &result.Destinations, &result.Observations, &result.HighRisk); err != nil {
		return result, err
	}
	read := func(query string) ([]AnalyticsItem, error) {
		rows, err := m.db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := []AnalyticsItem{}
		for rows.Next() {
			var item AnalyticsItem
			if err := rows.Scan(&item.Label, &item.Destinations, &item.Observations, &item.MaxRisk); err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, rows.Err()
	}
	var err error
	result.TopApps, err = read(`SELECT application,COUNT(*),SUM(seen),MAX(risk) FROM network_connection_timeline GROUP BY path,application ORDER BY SUM(seen) DESC LIMIT 8`)
	if err != nil {
		return result, err
	}
	result.TopTargets, err = read(`SELECT remote_ip,COUNT(DISTINCT path),SUM(seen),MAX(risk) FROM network_connection_timeline GROUP BY remote_ip ORDER BY SUM(seen) DESC LIMIT 8`)
	return result, err
}

func (m *Manager) NotificationSettings() (NotificationSettings, error) {
	var enabled bool
	var minRisk int
	var raw string
	if err := m.db.QueryRow(`SELECT enabled,min_risk,applications FROM network_notification_settings WHERE id=1`).Scan(&enabled, &minRisk, &raw); err != nil {
		return NotificationSettings{}, err
	}
	settings := NotificationSettings{Enabled: enabled, MinRisk: minRisk, Applications: []string{}}
	if err := json.Unmarshal([]byte(raw), &settings.Applications); err != nil {
		return NotificationSettings{}, err
	}
	return settings, nil
}

func (m *Manager) SetNotificationSettings(settings NotificationSettings) error {
	if settings.MinRisk < 0 || settings.MinRisk > 100 {
		return errors.New("minimum risk must be between 0 and 100")
	}
	clean := make([]string, 0, len(settings.Applications))
	seen := map[string]bool{}
	for _, item := range settings.Applications {
		item = strings.TrimSpace(item)
		if item == "" || seen[strings.ToLower(item)] {
			continue
		}
		seen[strings.ToLower(item)] = true
		clean = append(clean, item)
	}
	if len(clean) > 100 {
		return errors.New("no more than 100 notification applications are allowed")
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`UPDATE network_notification_settings SET enabled=?,min_risk=?,applications=? WHERE id=1`, settings.Enabled, settings.MinRisk, string(raw))
	return err
}

func (m *Manager) RetentionDays() (int, error) {
	var days int
	if err := m.db.QueryRow(`SELECT retention_days FROM network_intelligence_settings WHERE id=1`).Scan(&days); err != nil {
		return 7, err
	}
	return days, nil
}

func (m *Manager) HistoryStatus() (HistoryStatus, error) {
	days, err := m.RetentionDays()
	if err != nil {
		return HistoryStatus{}, err
	}
	status := HistoryStatus{RetentionDays: days}
	if err := m.db.QueryRow(`SELECT COUNT(*),COUNT(DISTINCT path) FROM network_connection_timeline`).Scan(&status.Connections, &status.Applications); err != nil {
		return HistoryStatus{}, err
	}
	return status, nil
}

func (m *Manager) SetRetentionDays(days int) error {
	allowed := map[int]bool{1: true, 7: true, 14: true, 30: true, 90: true}
	if !allowed[days] {
		return errors.New("retention must be 1, 7, 14, 30, or 90 days")
	}
	_, err := m.db.Exec(`UPDATE network_intelligence_settings SET retention_days=? WHERE id=1`, days)
	return err
}

func (m *Manager) ClearHistory() error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range []string{
		`DELETE FROM network_automatic_response_events`,
		`DELETE FROM network_learning_observations`,
		`DELETE FROM network_observed_apps`,
		`DELETE FROM network_connection_timeline`,
	} {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (m *Manager) ConnectionReport() ([]Connection, error) {
	rows, err := m.db.Query(`SELECT t.id,t.application,t.path,t.remote_ip,t.remote_port,t.protocol,t.risk,t.reasons,t.first_seen,t.last_seen,t.seen,
		CASE WHEN b.path IS NULL THEN 0 ELSE 1 END
		FROM network_connection_timeline t LEFT JOIN network_destination_baseline b
		ON b.path=t.path AND b.remote_ip=t.remote_ip AND b.remote_port=t.remote_port AND b.protocol=t.protocol
		ORDER BY t.last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Connection{}
	for rows.Next() {
		var item Connection
		if err := rows.Scan(&item.ID, &item.Application, &item.Path, &item.RemoteIP, &item.RemotePort, &item.Protocol, &item.Risk, &item.Reasons, &item.FirstSeen, &item.LastSeen, &item.Seen, &item.Trusted); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	m.decorateAndResolve(items)
	return items, nil
}

func (m *Manager) decorateAndResolve(items []Connection) {
	m.nameMu.Lock()
	unknown := make([]string, 0, 8)
	for i := range items {
		ip := items[i].RemoteIP
		if hostname, known := m.destinationDNS[ip]; known {
			items[i].Hostname = hostname
			continue
		}
		if !m.resolvingDNS[ip] && len(unknown) < 8 {
			m.resolvingDNS[ip] = true
			unknown = append(unknown, ip)
		}
	}
	m.nameMu.Unlock()
	for _, ip := range unknown {
		go m.resolveDestination(ip)
	}
}

func (m *Manager) resolveDestination(ip string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	names, err := net.DefaultResolver.LookupAddr(ctx, ip)
	cancel()
	hostname := ""
	if err == nil && len(names) > 0 {
		hostname = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(names[0])), ".")
	}
	m.nameMu.Lock()
	m.destinationDNS[ip] = hostname
	delete(m.resolvingDNS, ip)
	m.nameMu.Unlock()
}

type LearningSession struct {
	ID           int64      `json:"id"`
	Path         string     `json:"path"`
	Application  string     `json:"application"`
	StartedAt    time.Time  `json:"started_at"`
	EndsAt       time.Time  `json:"ends_at"`
	StoppedAt    *time.Time `json:"stopped_at,omitempty"`
	Active       bool       `json:"active"`
	Destinations int        `json:"destinations"`
	Ports        int        `json:"ports"`
	Observations int        `json:"observations"`
}

type BaselineConnection struct {
	RemoteIP     string `json:"remote_ip"`
	RemotePort   int    `json:"remote_port"`
	Protocol     string `json:"protocol"`
	Risk         int    `json:"risk"`
	Reasons      string `json:"reasons"`
	Observations int    `json:"observations"`
}

func (m *Manager) StartLearning(path string, minutes int) (int64, error) {
	path = strings.TrimSpace(strings.Trim(path, `"`))
	if path == "" {
		return 0, errors.New("application path required")
	}
	if minutes < 1 || minutes > 1440 {
		return 0, errors.New("learning duration must be 1–1440 minutes")
	}
	now := time.Now().UTC()
	r, e := m.db.Exec(`INSERT INTO network_learning_sessions(path,application,started_at,ends_at) VALUES(?,?,?,?)`, path, filepath.Base(path), now, now.Add(time.Duration(minutes)*time.Minute))
	if e != nil {
		return 0, e
	}
	return r.LastInsertId()
}
func (m *Manager) StopLearning(id int64) error {
	_, e := m.db.Exec(`UPDATE network_learning_sessions SET stopped_at=? WHERE id=? AND stopped_at IS NULL`, time.Now().UTC(), id)
	return e
}
func (m *Manager) LearningSessions() ([]LearningSession, error) {
	rows, e := m.db.Query(`SELECT id,path,application,started_at,ends_at,stopped_at FROM network_learning_sessions ORDER BY id DESC LIMIT 50`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []LearningSession{}
	now := time.Now().UTC()
	for rows.Next() {
		var s LearningSession
		if e := rows.Scan(&s.ID, &s.Path, &s.Application, &s.StartedAt, &s.EndsAt, &s.StoppedAt); e != nil {
			return nil, e
		}
		s.Active = s.StoppedAt == nil && now.Before(s.EndsAt)
		_ = m.db.QueryRow(`SELECT COUNT(DISTINCT remote_ip),COUNT(DISTINCT remote_port),COALESCE(SUM(seen),0) FROM network_learning_observations WHERE session_id=?`, s.ID).Scan(&s.Destinations, &s.Ports, &s.Observations)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (m *Manager) LearningBaseline(id int64) (LearningSession, []BaselineConnection, error) {
	var session LearningSession
	err := m.db.QueryRow(`SELECT id,path,application,started_at,ends_at,stopped_at FROM network_learning_sessions WHERE id=?`, id).Scan(
		&session.ID, &session.Path, &session.Application, &session.StartedAt, &session.EndsAt, &session.StoppedAt,
	)
	if err != nil {
		return session, nil, err
	}
	now := time.Now().UTC()
	session.Active = session.StoppedAt == nil && now.Before(session.EndsAt)
	rows, err := m.db.Query(`SELECT remote_ip,remote_port,protocol,risk,reasons,seen
		FROM network_learning_observations WHERE session_id=?
		ORDER BY seen DESC,remote_ip,remote_port`, session.ID)
	if err != nil {
		return session, nil, err
	}
	defer rows.Close()
	items := []BaselineConnection{}
	for rows.Next() {
		var item BaselineConnection
		if err := rows.Scan(&item.RemoteIP, &item.RemotePort, &item.Protocol, &item.Risk, &item.Reasons, &item.Observations); err != nil {
			return session, nil, err
		}
		items = append(items, item)
	}
	return session, items, rows.Err()
}
func (m *Manager) Recent(limit int) ([]Connection, error) {
	if limit < 1 || limit > 500 {
		limit = 200
	}
	rows, err := m.db.Query(`SELECT id,application,path,remote_ip,remote_port,protocol,risk,reasons,first_seen,last_seen,seen FROM network_connection_timeline ORDER BY last_seen DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Connection{}
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ID, &c.Application, &c.Path, &c.RemoteIP, &c.RemotePort, &c.Protocol, &c.Risk, &c.Reasons, &c.FirstSeen, &c.LastSeen, &c.Seen); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
