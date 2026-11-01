package database

import "time"

type NetworkTestRecord struct {
	ID              string    `json:"id"`
	Status          string    `json:"status"`
	Mode            string    `json:"mode"`
	LimitMB         int       `json:"limit_mb"`
	DurationSeconds int       `json:"duration_seconds"`
	PingMS          float64   `json:"ping_ms"`
	JitterMS        float64   `json:"jitter_ms"`
	PacketLoss      float64   `json:"packet_loss"`
	DownloadMbps    float64   `json:"download_mbps"`
	UploadMbps      float64   `json:"upload_mbps"`
	BytesDownloaded int64     `json:"bytes_downloaded"`
	BytesUploaded   int64     `json:"bytes_uploaded"`
	Error           string    `json:"error,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
}

func (db *DB) SaveNetworkTest(record NetworkTestRecord) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT OR REPLACE INTO network_test_history
		(id,status,mode,limit_mb,duration_seconds,ping_ms,jitter_ms,packet_loss,download_mbps,upload_mbps,bytes_downloaded,bytes_uploaded,error,started_at,finished_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, record.ID, record.Status, record.Mode, record.LimitMB, record.DurationSeconds, record.PingMS, record.JitterMS, record.PacketLoss, record.DownloadMbps, record.UploadMbps, record.BytesDownloaded, record.BytesUploaded, record.Error, record.StartedAt, record.FinishedAt)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM network_test_history WHERE id NOT IN (SELECT id FROM network_test_history ORDER BY started_at DESC, id DESC LIMIT 30)`); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) GetNetworkTestHistory(limit int) ([]NetworkTestRecord, error) {
	if limit < 1 || limit > 30 {
		limit = 30
	}
	rows, err := db.conn.Query(`SELECT id,status,mode,limit_mb,duration_seconds,ping_ms,jitter_ms,packet_loss,download_mbps,upload_mbps,bytes_downloaded,bytes_uploaded,error,started_at,finished_at FROM network_test_history ORDER BY started_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]NetworkTestRecord, 0, limit)
	for rows.Next() {
		var x NetworkTestRecord
		if err := rows.Scan(&x.ID, &x.Status, &x.Mode, &x.LimitMB, &x.DurationSeconds, &x.PingMS, &x.JitterMS, &x.PacketLoss, &x.DownloadMbps, &x.UploadMbps, &x.BytesDownloaded, &x.BytesUploaded, &x.Error, &x.StartedAt, &x.FinishedAt); err != nil {
			return nil, err
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
