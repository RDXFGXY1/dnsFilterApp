package database

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn        *sql.DB
	liveMu      sync.Mutex
	livePrivacy LivePrivacy
}

type LivePrivacy struct {
	RetentionHours int  `json:"retention_hours"`
	HideClientIP   bool `json:"hide_client_ip"`
	BlockedOnly    bool `json:"blocked_only"`
}

type BlockedQuery struct {
	ID        int64
	Domain    string
	ClientIP  string
	Timestamp time.Time
}

type ObservedDomain struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

type LiveQuery struct {
	Time     int64  `json:"time"`
	Domain   string `json:"domain"`
	ClientIP string `json:"client_ip"`
	Type     string `json:"type"`
	Blocked  bool   `json:"blocked"`
	Reason   string `json:"reason"`
	Cached   bool   `json:"cached"`
}

func (db *DB) GetDB() *sql.DB {
	return db.conn // Assuming your DB struct has a field called 'conn' of type *sql.DB
}

func New(dbPath string) (*DB, error) {
	// WAL mode allows concurrent readers with a writer, and busy_timeout makes
	// writers wait for a lock instead of failing immediately. This keeps the DNS
	// hot path from stalling under concurrent blocked-query writes.
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)", dbPath)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn, livePrivacy: LivePrivacy{RetentionHours: 168}}

	if err := db.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS blocked_queries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain TEXT NOT NULL,
		client_ip TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_blocked_timestamp ON blocked_queries(timestamp);
	CREATE INDEX IF NOT EXISTS idx_blocked_domain ON blocked_queries(domain);

	CREATE TABLE IF NOT EXISTS live_queries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		time INTEGER NOT NULL,
		domain TEXT NOT NULL,
		client_ip TEXT NOT NULL,
		qtype TEXT NOT NULL,
		blocked BOOLEAN NOT NULL,
		reason TEXT NOT NULL,
		cached BOOLEAN NOT NULL
	);

	CREATE TABLE IF NOT EXISTS network_test_history (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		mode TEXT NOT NULL,
		limit_mb INTEGER NOT NULL,
		duration_seconds INTEGER NOT NULL,
		ping_ms REAL NOT NULL,
		jitter_ms REAL NOT NULL,
		packet_loss REAL NOT NULL,
		download_mbps REAL NOT NULL,
		upload_mbps REAL NOT NULL,
		bytes_downloaded INTEGER NOT NULL,
		bytes_uploaded INTEGER NOT NULL,
		error TEXT NOT NULL,
		started_at DATETIME NOT NULL,
		finished_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS dns_health_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kind TEXT NOT NULL,
		status TEXT NOT NULL,
		title TEXT NOT NULL,
		detail TEXT NOT NULL,
		at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS blocklist (
		domain TEXT PRIMARY KEY,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS whitelist (
		domain TEXT PRIMARY KEY,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS custom_blocklist (
		domain TEXT PRIMARY KEY,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS blocklist_sources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		category TEXT,
		enabled BOOLEAN DEFAULT 1,
		last_updated DATETIME,
		domain_count INTEGER DEFAULT 0
	);
	`

	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) LogBlockedQuery(domain, clientIP string, timestamp time.Time) error {
	query := `INSERT INTO blocked_queries (domain, client_ip, timestamp) VALUES (?, ?, ?)`
	_, err := db.conn.Exec(query, domain, clientIP, timestamp)
	return err
}

// SaveLiveQueries stores only the latest 200 DNS outcomes. A batch keeps writes
// off the DNS request path and makes insertion and pruning atomic.
func (db *DB) SaveLiveQueries(items []LiveQuery) error {
	if len(items) == 0 {
		return nil
	}
	db.liveMu.Lock()
	defer db.liveMu.Unlock()
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO live_queries (time, domain, client_ip, qtype, blocked, reason, cached) VALUES (?, ?, ?, ?, ?, ?, ?) `)
	if err != nil {
		return err
	}
	for _, item := range items {
		if db.livePrivacy.BlockedOnly && !item.Blocked {
			continue
		}
		if db.livePrivacy.HideClientIP {
			item.ClientIP = ""
		}
		if _, err = stmt.Exec(item.Time, item.Domain, item.ClientIP, item.Type, item.Blocked, item.Reason, item.Cached); err != nil {
			stmt.Close()
			return err
		}
	}
	if err = stmt.Close(); err != nil {
		return err
	}
	cutoff := time.Now().Add(-time.Duration(db.livePrivacy.RetentionHours) * time.Hour).Unix()
	if _, err = tx.Exec(`DELETE FROM live_queries WHERE time < ? OR id NOT IN (SELECT id FROM live_queries ORDER BY id DESC LIMIT 200)`, cutoff); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) GetLiveQueries(limit int) ([]LiveQuery, error) {
	db.liveMu.Lock()
	defer db.liveMu.Unlock()
	if limit < 1 || limit > 200 {
		limit = 200
	}
	cutoff := time.Now().Add(-time.Duration(db.livePrivacy.RetentionHours) * time.Hour).Unix()
	rows, err := db.conn.Query(`SELECT time, domain, client_ip, qtype, blocked, reason, cached FROM live_queries WHERE time >= ? ORDER BY id DESC LIMIT ?`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]LiveQuery, 0, limit)
	for rows.Next() {
		var item LiveQuery
		if err := rows.Scan(&item.Time, &item.Domain, &item.ClientIP, &item.Type, &item.Blocked, &item.Reason, &item.Cached); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ClearLiveQueries() error {
	db.liveMu.Lock()
	defer db.liveMu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM live_queries`)
	return err
}

func (db *DB) LivePrivacy() LivePrivacy {
	db.liveMu.Lock()
	defer db.liveMu.Unlock()
	return db.livePrivacy
}

func (db *DB) SetLivePrivacy(settings LivePrivacy) error {
	if settings.RetentionHours < 1 || settings.RetentionHours > 720 {
		return fmt.Errorf("invalid live history retention")
	}
	db.liveMu.Lock()
	defer db.liveMu.Unlock()
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if settings.HideClientIP {
		if _, err = tx.Exec(`UPDATE live_queries SET client_ip = ''`); err != nil {
			return err
		}
	}
	if settings.BlockedOnly {
		if _, err = tx.Exec(`DELETE FROM live_queries WHERE blocked = 0`); err != nil {
			return err
		}
	}
	cutoff := time.Now().Add(-time.Duration(settings.RetentionHours) * time.Hour).Unix()
	if _, err = tx.Exec(`DELETE FROM live_queries WHERE time < ?`, cutoff); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	db.livePrivacy = settings
	return nil
}

func (db *DB) GetRecentBlocked(limit int) ([]BlockedQuery, error) {
	query := `
		SELECT id, domain, client_ip, timestamp 
		FROM blocked_queries 
		ORDER BY timestamp DESC 
		LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []BlockedQuery
	for rows.Next() {
		var bq BlockedQuery
		if err := rows.Scan(&bq.ID, &bq.Domain, &bq.ClientIP, &bq.Timestamp); err != nil {
			return nil, err
		}
		results = append(results, bq)
	}

	return results, rows.Err()
}

// GetObservedSubdomains returns domains already seen in the local blocked-query
// history. It intentionally performs no third-party lookup or traffic upload.
func (db *DB) GetObservedSubdomains(root string, limit int) ([]ObservedDomain, error) {
	if limit < 1 || limit > 500 {
		limit = 200
	}
	rows, err := db.conn.Query(`
		SELECT domain, COUNT(*) AS hits
		FROM blocked_queries
		WHERE domain = ? OR domain LIKE ?
		GROUP BY domain
		ORDER BY hits DESC, domain ASC
		LIMIT ?`, root, "%."+root, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ObservedDomain{}
	for rows.Next() {
		var item ObservedDomain
		if err := rows.Scan(&item.Domain, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) GetBlockedStats(hours int) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(DISTINCT domain) as unique_domains,
			COUNT(DISTINCT client_ip) as unique_clients
		FROM blocked_queries
		WHERE timestamp > datetime('now', '-' || ? || ' hours')
	`

	var stats map[string]interface{} = make(map[string]interface{})
	var total, uniqueDomains, uniqueClients int64

	err := db.conn.QueryRow(query, hours).Scan(&total, &uniqueDomains, &uniqueClients)
	if err != nil {
		return nil, err
	}

	stats["total_blocked"] = total
	stats["unique_domains"] = uniqueDomains
	stats["unique_clients"] = uniqueClients

	return stats, nil
}

func (db *DB) GetTopBlockedDomains(limit int) (map[string]int, error) {
	query := `
		SELECT domain, COUNT(*) as count
		FROM blocked_queries
		WHERE timestamp > datetime('now', '-24 hours')
		GROUP BY domain
		ORDER BY count DESC
		LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]int)
	for rows.Next() {
		var domain string
		var count int
		if err := rows.Scan(&domain, &count); err != nil {
			return nil, err
		}
		results[domain] = count
	}

	return results, rows.Err()
}

// TimelinePoint is one hourly bucket of blocked queries.
type TimelinePoint struct {
	Label string `json:"label"` // "15:00"
	Count int    `json:"count"`
}

// GetBlockedTimeline returns a continuous per-hour series of blocked-query
// counts for the last `hours` hours (empty hours included as zero).
func (db *DB) GetBlockedTimeline(hours int) ([]TimelinePoint, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	rows, err := db.conn.Query(
		`SELECT timestamp FROM blocked_queries WHERE timestamp > ?`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buckets := make(map[string]int)
	for rows.Next() {
		var t time.Time
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		buckets[t.Local().Format("2006-01-02T15")]++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	now := time.Now()
	points := make([]TimelinePoint, 0, hours)
	for i := hours - 1; i >= 0; i-- {
		h := now.Add(-time.Duration(i) * time.Hour)
		key := h.Format("2006-01-02T15")
		points = append(points, TimelinePoint{
			Label: h.Format("15:04"),
			Count: buckets[key],
		})
	}
	return points, nil
}

func (db *DB) SaveBlocklist(domains map[string]bool) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing blocklist
	if _, err := tx.Exec("DELETE FROM blocklist"); err != nil {
		return err
	}

	// Insert new blocklist
	stmt, err := tx.Prepare("INSERT INTO blocklist (domain) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for domain := range domains {
		if _, err := stmt.Exec(domain); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) LoadBlocklist() (map[string]bool, error) {
	query := "SELECT domain FROM blocklist"
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	domains := make(map[string]bool)
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains[domain] = true
	}

	return domains, rows.Err()
}

func (db *DB) AddToWhitelist(domain string) error {
	query := "INSERT OR REPLACE INTO whitelist (domain) VALUES (?)"
	_, err := db.conn.Exec(query, domain)
	return err
}

func (db *DB) RemoveFromWhitelist(domain string) error {
	query := "DELETE FROM whitelist WHERE domain = ?"
	_, err := db.conn.Exec(query, domain)
	return err
}

func (db *DB) GetWhitelist() ([]string, error) {
	query := "SELECT domain FROM whitelist ORDER BY domain"
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}

	return domains, rows.Err()
}

func (db *DB) AddToCustomBlocklist(domain string) error {
	_, err := db.conn.Exec("INSERT OR REPLACE INTO custom_blocklist (domain) VALUES (?)", domain)
	return err
}

func (db *DB) RemoveFromCustomBlocklist(domain string) error {
	_, err := db.conn.Exec("DELETE FROM custom_blocklist WHERE domain = ?", domain)
	return err
}

func (db *DB) GetCustomBlocklist() ([]string, error) {
	rows, err := db.conn.Query("SELECT domain FROM custom_blocklist ORDER BY domain")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, rows.Err()
}

func (db *DB) CleanupOldLogs(days int) error {
	query := "DELETE FROM blocked_queries WHERE timestamp < datetime('now', '-' || ? || ' days')"
	_, err := db.conn.Exec(query, days)
	return err
}

func (db *DB) GetSetting(key string) (string, error) {
	query := "SELECT value FROM settings WHERE key = ?"
	var value string
	err := db.conn.QueryRow(query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (db *DB) SetSetting(key, value string) error {
	query := "INSERT OR REPLACE INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)"
	_, err := db.conn.Exec(query, key, value)
	return err
}
