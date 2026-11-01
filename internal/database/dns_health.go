package database

import "time"

type DNSHealthEvent struct {
	ID     int64     `json:"id"`
	Kind   string    `json:"kind"`
	Status string    `json:"status"`
	Title  string    `json:"title"`
	Detail string    `json:"detail"`
	At     time.Time `json:"at"`
}

func (db *DB) LogDNSHealth(kind, status, title, detail string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO dns_health_events(kind,status,title,detail,at) VALUES(?,?,?,?,?)`, kind, status, title, detail, time.Now().UTC()); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM dns_health_events WHERE id NOT IN (SELECT id FROM dns_health_events ORDER BY id DESC LIMIT 200)`); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) GetDNSHealthEvents(limit int) ([]DNSHealthEvent, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := db.conn.Query(`SELECT id,kind,status,title,detail,at FROM dns_health_events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]DNSHealthEvent, 0, limit)
	for rows.Next() {
		var x DNSHealthEvent
		if err := rows.Scan(&x.ID, &x.Kind, &x.Status, &x.Title, &x.Detail, &x.At); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
