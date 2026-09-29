package hardsystem

import (
	"database/sql"
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	DefaultThreshold = 70
	MaxTextBytes     = 120000
)

type Request struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Text        string `json:"text"`
}

type Signal struct {
	Name   string `json:"name"`
	Points int    `json:"points"`
	Count  int    `json:"count"`
}

type Result struct {
	Domain     string    `json:"domain"`
	Score      int       `json:"score"`
	Action     string    `json:"action"`
	Category   string    `json:"category"`
	Reasons    []string  `json:"reasons"`
	Signals    []Signal  `json:"signals"`
	Threshold  int       `json:"threshold"`
	AnalyzedAt time.Time `json:"analyzed_at"`
	Version    string    `json:"version"`
}

type Event struct {
	ID         int64     `json:"id"`
	Domain     string    `json:"domain"`
	Score      int       `json:"score"`
	Action     string    `json:"action"`
	Category   string    `json:"category"`
	Reasons    string    `json:"reasons"`
	AnalyzedAt time.Time `json:"analyzed_at"`
}

type Manager struct{ db *sql.DB }

func New(db *sql.DB) (*Manager, error) {
	if db == nil {
		return nil, errors.New("database unavailable")
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS hard_system_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain TEXT NOT NULL, score INTEGER NOT NULL, action TEXT NOT NULL,
		category TEXT NOT NULL, reasons TEXT NOT NULL, analyzed_at TIMESTAMP NOT NULL
	)`)
	if err != nil {
		return nil, err
	}
	return &Manager{db: db}, nil
}

var wordRE = regexp.MustCompile(`[a-z0-9]+`)

var strongAdult = []string{"porn", "pornographic", "xxx", "hentai", "camgirl", "adult video", "sex video", "nude video", "explicit video"}
var adultTerms = []string{"nude", "nudity", "sexual", "erotic", "fetish", "hardcore", "nsfw", "adult content", "explicit content"}
var suspiciousTerms = []string{"free crack", "cracked software", "keygen", "steal password", "verify your wallet", "seed phrase", "urgent account verification", "disable antivirus", "malware download"}
var contextTerms = []string{"education", "educational", "medical", "health", "research", "safety", "prevention", "dictionary", "encyclopedia"}

func (m *Manager) Analyze(in Request) (Result, error) {
	if len(in.Text) > MaxTextBytes {
		in.Text = in.Text[:MaxTextBytes]
	}
	if len(in.Title) > 500 {
		in.Title = in.Title[:500]
	}
	if len(in.Description) > 2000 {
		in.Description = in.Description[:2000]
	}
	u, err := url.Parse(strings.TrimSpace(in.URL))
	if err != nil || u.Hostname() == "" {
		return Result{}, errors.New("valid page URL required")
	}
	domain := strings.ToLower(u.Hostname())
	body := normalize(in.Text)
	meta := normalize(in.Title + " " + in.Description)
	domainText := normalize(domain)
	wordCount := len(wordRE.FindAllString(body, -1))
	if wordCount == 0 {
		wordCount = 1
	}
	signals := []Signal{}
	reasons := []string{}
	score := 0
	adultScore := 0
	suspiciousScore := 0
	add := func(name string, points, count int) {
		if points <= 0 || count <= 0 {
			return
		}
		signals = append(signals, Signal{name, points, count})
		score += points
		reasons = append(reasons, name)
	}
	strongCount := hits(body, strongAdult)
	strongPoints := min(55, strongCount*14)
	add("Strong adult-content phrases", strongPoints, strongCount)
	adultScore += strongPoints
	adultCount := hits(body, adultTerms)
	densityPoints := min(28, adultCount*7)
	if adultCount*1000/wordCount >= 12 {
		densityPoints += 10
	}
	add("Adult-content language density", densityPoints, adultCount)
	adultScore += densityPoints
	metaCount := hits(meta, append(strongAdult, adultTerms...))
	metaPoints := min(24, metaCount*12)
	add("Sensitive title or metadata", metaPoints, metaCount)
	adultScore += metaPoints
	domainCount := hits(domainText, strongAdult)
	domainPoints := min(24, domainCount*18)
	add("Sensitive domain pattern", domainPoints, domainCount)
	adultScore += domainPoints
	suspiciousCount := hits(body+" "+meta, suspiciousTerms)
	suspiciousPoints := min(55, suspiciousCount*18)
	add("Suspicious or harmful instructions", suspiciousPoints, suspiciousCount)
	suspiciousScore += suspiciousPoints
	contextCount := hits(body+" "+meta, contextTerms)
	if contextCount > 0 && adultScore > 0 && strongCount == 0 && domainCount == 0 {
		reduction := min(25, contextCount*6)
		score -= reduction
		signals = append(signals, Signal{"Educational or medical context", -reduction, contextCount})
		reasons = append(reasons, "Educational/medical context reduced risk")
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	category := "safe"
	if adultScore >= suspiciousScore && adultScore > 0 {
		category = "adult"
	} else if suspiciousScore > 0 {
		category = "suspicious"
	}
	action := "allow"
	if score >= DefaultThreshold {
		action = "block"
	} else if score >= 40 {
		action = "warn"
	}
	if len(reasons) == 0 {
		reasons = []string{"No strong local content-risk signals detected"}
	}
	sort.Slice(signals, func(i, j int) bool { return signals[i].Points > signals[j].Points })
	result := Result{domain, score, action, category, reasons, signals, DefaultThreshold, time.Now().UTC(), "hard-system-v1"}
	_, _ = m.db.Exec(`INSERT INTO hard_system_events(domain,score,action,category,reasons,analyzed_at) VALUES(?,?,?,?,?,?)`, domain, score, action, category, strings.Join(reasons, " · "), result.AnalyzedAt)
	_, _ = m.db.Exec(`DELETE FROM hard_system_events WHERE id NOT IN (SELECT id FROM hard_system_events ORDER BY id DESC LIMIT 500)`)
	return result, nil
}

func (m *Manager) Recent(limit int) ([]Event, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := m.db.Query(`SELECT id,domain,score,action,category,reasons,analyzed_at FROM hard_system_events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Domain, &e.Score, &e.Action, &e.Category, &e.Reasons, &e.AnalyzedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func normalize(s string) string {
	return strings.Join(wordRE.FindAllString(strings.ToLower(s), -1), " ")
}
func hits(s string, terms []string) int {
	padded := " " + strings.TrimSpace(s) + " "
	total := 0
	for _, term := range terms {
		total += strings.Count(padded, " "+normalize(term)+" ")
	}
	return total
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
