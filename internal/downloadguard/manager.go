package downloadguard

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Event struct {
	ID              int64     `json:"id"`
	Path            string    `json:"path"`
	Filename        string    `json:"filename"`
	SHA256          string    `json:"sha256"`
	SizeBytes       int64     `json:"size_bytes"`
	Extension       string    `json:"extension"`
	DetectedType    string    `json:"detected_type"`
	SignatureStatus string    `json:"signature_status"`
	Publisher       string    `json:"publisher"`
	RiskScore       int       `json:"risk_score"`
	RiskLevel       string    `json:"risk_level"`
	Reasons         []string  `json:"reasons"`
	State           string    `json:"state"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
type Analytics struct {
	Total       int   `json:"total"`
	Today       int   `json:"today"`
	Executables int   `json:"executables"`
	HighRisk    int   `json:"high_risk"`
	Quarantined int   `json:"quarantined"`
	TotalBytes  int64 `json:"total_bytes"`
}
type observed struct {
	size   int64
	mod    time.Time
	stable int
}
type Manager struct {
	db                 *sql.DB
	folder, quarantine string
	mu                 sync.Mutex
	seen               map[string]observed
	cancel             context.CancelFunc
	done               chan struct{}
}

func New(db *sql.DB, quarantine string) (*Manager, error) {
	if db == nil {
		return nil, errors.New("database unavailable")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	folder := filepath.Join(home, "Downloads")
	if quarantine == "" {
		quarantine = filepath.Join("data", "quarantine", "downloads")
	}
	if err = os.MkdirAll(quarantine, 0700); err != nil {
		return nil, fmt.Errorf("create download quarantine: %w", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS download_events(id INTEGER PRIMARY KEY AUTOINCREMENT,path TEXT NOT NULL,filename TEXT NOT NULL,sha256 TEXT NOT NULL,size_bytes INTEGER NOT NULL,extension TEXT NOT NULL,detected_type TEXT NOT NULL,signature_status TEXT NOT NULL,publisher TEXT NOT NULL,risk_score INTEGER NOT NULL,risk_level TEXT NOT NULL,reasons TEXT NOT NULL,state TEXT NOT NULL,quarantine_path TEXT NOT NULL DEFAULT '',created_at TIMESTAMP NOT NULL,updated_at TIMESTAMP NOT NULL); CREATE INDEX IF NOT EXISTS idx_download_events_created ON download_events(created_at DESC); CREATE INDEX IF NOT EXISTS idx_download_events_hash ON download_events(sha256);`)
	if err != nil {
		return nil, err
	}
	m := &Manager{db: db, folder: folder, quarantine: quarantine, seen: map[string]observed{}, done: make(chan struct{})}
	m.baseline()
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.loop(ctx)
	return m, nil
}
func (m *Manager) Folder() string { return m.folder }
func (m *Manager) Close() {
	if m.cancel != nil {
		m.cancel()
		<-m.done
	}
}
func (m *Manager) baseline() {
	entries, _ := os.ReadDir(m.folder)
	for _, e := range entries {
		if e.IsDir() || ignore(e.Name()) {
			continue
		}
		if i, err := e.Info(); err == nil {
			m.seen[filepath.Join(m.folder, e.Name())] = observed{size: i.Size(), mod: i.ModTime(), stable: 2}
		}
	}
}
func ignore(name string) bool {
	n := strings.ToLower(name)
	return strings.HasSuffix(n, ".crdownload") || strings.HasSuffix(n, ".part") || strings.HasSuffix(n, ".partial") || strings.HasSuffix(n, ".tmp")
}
func (m *Manager) loop(ctx context.Context) {
	defer close(m.done)
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.scan()
		}
	}
}
func (m *Manager) scan() {
	entries, err := os.ReadDir(m.folder)
	if err != nil {
		return
	}
	present := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || ignore(e.Name()) {
			continue
		}
		path := filepath.Join(m.folder, e.Name())
		present[path] = true
		i, err := e.Info()
		if err != nil {
			continue
		}
		m.mu.Lock()
		old, ok := m.seen[path]
		if !ok || old.size != i.Size() || !old.mod.Equal(i.ModTime()) {
			m.seen[path] = observed{size: i.Size(), mod: i.ModTime()}
			m.mu.Unlock()
			continue
		}
		old.stable++
		m.seen[path] = old
		m.mu.Unlock()
		if old.stable == 2 {
			go m.analyze(path)
		}
	}
	m.mu.Lock()
	for p := range m.seen {
		if !present[p] {
			delete(m.seen, p)
		}
	}
	m.mu.Unlock()
}

func (m *Manager) analyze(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return
	}
	h := sha256.New()
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	if _, err = f.Seek(0, 0); err != nil {
		return
	}
	if _, err = io.Copy(h, f); err != nil {
		return
	}
	hash := hex.EncodeToString(h.Sum(nil))
	var exists int
	if m.db.QueryRow(`SELECT count(*) FROM download_events WHERE sha256=? AND path=?`, hash, path).Scan(&exists) == nil && exists > 0 {
		return
	}
	ext := strings.ToLower(filepath.Ext(path))
	kind := detectType(head[:n], ext)
	sig, publisher := signatureInfo(path)
	score, reasons := scoreFile(filepath.Base(path), ext, kind, sig)
	level := riskLevel(score)
	now := time.Now().UTC()
	_, _ = m.db.Exec(`INSERT INTO download_events(path,filename,sha256,size_bytes,extension,detected_type,signature_status,publisher,risk_score,risk_level,reasons,state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, path, filepath.Base(path), hash, info.Size(), ext, kind, sig, publisher, score, level, mustJSON(reasons), "complete", now, now)
}
func detectType(head []byte, ext string) string {
	if len(head) >= 2 && head[0] == 'M' && head[1] == 'Z' {
		return "windows-executable"
	}
	if len(head) >= 4 && string(head[:4]) == "PK\x03\x04" {
		return "zip-archive"
	}
	if len(head) >= 4 && string(head[:4]) == "%PDF" {
		return "pdf-document"
	}
	if len(head) >= 8 && string(head[1:4]) == "PNG" {
		return "png-image"
	}
	if len(head) >= 3 && head[0] == 0xff && head[1] == 0xd8 && head[2] == 0xff {
		return "jpeg-image"
	}
	if ext == ".ps1" || ext == ".bat" || ext == ".cmd" || ext == ".vbs" || ext == ".js" {
		return "script"
	}
	return "unknown"
}
func scoreFile(name, ext, kind, sig string) (int, []string) {
	score := 0
	reasons := []string{}
	add := func(n int, s string) { score += n; reasons = append(reasons, fmt.Sprintf("%+d %s", n, s)) }
	danger := map[string]int{".scr": 40, ".ps1": 25, ".bat": 25, ".cmd": 25, ".vbs": 30, ".js": 20, ".msi": 15, ".exe": 15}
	if n := danger[ext]; n > 0 {
		add(n, "executable or command-capable file type")
	}
	parts := strings.Split(strings.ToLower(name), ".")
	if len(parts) > 2 && danger["."+parts[len(parts)-1]] > 0 {
		add(25, "misleading double extension")
	}
	if kind == "windows-executable" && ext != ".exe" && ext != ".scr" && ext != ".dll" {
		add(30, "file header does not match its extension")
	}
	if (kind == "windows-executable" || ext == ".msi") && sig == "unsigned" {
		add(15, "executable is not digitally signed")
	}
	if sig == "invalid" {
		add(35, "digital signature is invalid")
	}
	if sig == "valid" {
		score -= 15
		reasons = append(reasons, "-15 valid Authenticode signature")
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "No high-risk file signals were detected")
	}
	return score, reasons
}
func riskLevel(n int) string {
	if n >= 75 {
		return "critical"
	}
	if n >= 50 {
		return "high"
	}
	if n >= 25 {
		return "review"
	}
	return "low"
}
func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func (m *Manager) List(limit int) ([]Event, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := m.db.Query(`SELECT id,path,filename,sha256,size_bytes,extension,detected_type,signature_status,publisher,risk_score,risk_level,reasons,state,created_at,updated_at FROM download_events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var reasons string
		if err := rows.Scan(&e.ID, &e.Path, &e.Filename, &e.SHA256, &e.SizeBytes, &e.Extension, &e.DetectedType, &e.SignatureStatus, &e.Publisher, &e.RiskScore, &e.RiskLevel, &reasons, &e.State, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(reasons), &e.Reasons)
		out = append(out, e)
	}
	return out, rows.Err()
}
func (m *Manager) Analytics() (Analytics, error) {
	var a Analytics
	err := m.db.QueryRow(`SELECT count(*),coalesce(sum(CASE WHEN date(created_at)=date('now') THEN 1 ELSE 0 END),0),coalesce(sum(CASE WHEN detected_type='windows-executable' OR extension IN('.exe','.msi','.scr') THEN 1 ELSE 0 END),0),coalesce(sum(CASE WHEN risk_score>=50 THEN 1 ELSE 0 END),0),coalesce(sum(CASE WHEN state='quarantined' THEN 1 ELSE 0 END),0),coalesce(sum(size_bytes),0) FROM download_events`).Scan(&a.Total, &a.Today, &a.Executables, &a.HighRisk, &a.Quarantined, &a.TotalBytes)
	return a, err
}
func (m *Manager) Quarantine(id int64) error {
	var path, hash, state string
	if err := m.db.QueryRow(`SELECT path,sha256,state FROM download_events WHERE id=?`, id).Scan(&path, &hash, &state); err != nil {
		return errors.New("download event not found")
	}
	if state == "quarantined" {
		return nil
	}
	actual, err := hashFile(path)
	if err != nil {
		return err
	}
	if actual != hash {
		return errors.New("file changed after analysis; refresh before quarantining")
	}
	dest := filepath.Join(m.quarantine, fmt.Sprintf("%d-%s.object", id, hash[:16]))
	if err = os.Rename(path, dest); err != nil {
		return fmt.Errorf("quarantine move failed: %w", err)
	}
	_, err = m.db.Exec(`UPDATE download_events SET state='quarantined',quarantine_path=?,updated_at=? WHERE id=?`, dest, time.Now().UTC(), id)
	if err != nil {
		_ = os.Rename(dest, path)
	}
	return err
}
func (m *Manager) Restore(id int64) error {
	var path, q, state string
	if err := m.db.QueryRow(`SELECT path,quarantine_path,state FROM download_events WHERE id=?`, id).Scan(&path, &q, &state); err != nil {
		return errors.New("download event not found")
	}
	if state != "quarantined" {
		return errors.New("file is not quarantined")
	}
	if _, err := os.Stat(path); err == nil {
		return errors.New("original path is occupied; move that file before restoring")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.Rename(q, path); err != nil {
		return err
	}
	_, err := m.db.Exec(`UPDATE download_events SET state='restored',quarantine_path='',updated_at=? WHERE id=?`, time.Now().UTC(), id)
	return err
}
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func SortByRisk(events []Event) {
	sort.Slice(events, func(i, j int) bool { return events[i].RiskScore > events[j].RiskScore })
}
