package contentinspector

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ════════════════════════════════════════════════════════════════
//  CONTENT INSPECTOR - DNS Filter v2.5
//  Fetches actual website content and scans for adult keywords
//  More accurate than domain-only filtering
// ════════════════════════════════════════════════════════════════

type ContentInspector struct {
	db              *sql.DB
	httpClient      *http.Client
	suspiciousWords []string
	mu              sync.RWMutex
	cache           map[string]*CacheEntry
	enabled         bool
}

type CacheEntry struct {
	IsAdult   bool
	Reason    string
	Timestamp time.Time
}

type InspectionResult struct {
	Domain      string   `json:"domain"`
	IsAdult     bool     `json:"is_adult"`
	Confidence  float64  `json:"confidence"`
	MatchedWords []string `json:"matched_words"`
	Reason      string   `json:"reason"`
}

func NewContentInspector(db *sql.DB) (*ContentInspector, error) {
	ci := &ContentInspector{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't follow redirects, just inspect the first page
				return http.ErrUseLastResponse
			},
		},
		cache:   make(map[string]*CacheEntry),
		enabled: true,
	}

	// Load suspicious words
	ci.loadSuspiciousWords()

	// Initialize database
	if err := ci.initTables(); err != nil {
		return nil, err
	}

	// Start cache cleanup
	go ci.cleanupCache()

	return ci, nil
}

// ── Database Schema ──────────────────────────────────────────────

func (ci *ContentInspector) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS content_inspection_cache (
			domain TEXT PRIMARY KEY,
			is_adult BOOLEAN,
			matched_words TEXT,
			confidence REAL,
			inspected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS content_inspection_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT,
			is_adult BOOLEAN,
			matched_words TEXT,
			content_sample TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := ci.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

// ── Suspicious Words Database ───────────────────────────────────

func (ci *ContentInspector) loadSuspiciousWords() {
	ci.suspiciousWords = []string{
		// Live streaming adult content
		"live sex", "live cam", "live porn", "webcam sex", "cam girls",
		"live girls", "sex cam", "adult cam", "xxx cam", "porn cam",
		"chaturbate", "stripchat", "cam4", "myfreecams", "bongacams",
		
		// Sex dolls and products
		"sex doll", "sex toy", "adult toy", "realistic doll", "love doll",
		"silicone doll", "tpe doll", "sex robot", "adult doll",
		"masturbator", "dildo", "vibrator", "fleshlight",
		
		// Escort and dating
		"escort service", "call girl", "prostitute", "sex worker",
		"hookup", "sugar daddy", "sugar baby", "affair dating",
		
		// Explicit content
		"xxx videos", "porn videos", "adult videos", "fuck videos",
		"sex videos", "naked girls", "nude girls", "naked women",
		"hardcore porn", "anal sex", "oral sex", "group sex",
		"gangbang", "milf porn", "teen porn", "lesbian porn",
		
		// Adult services
		"adult entertainment", "gentlemen's club", "strip club",
		"erotic massage", "happy ending", "sensual massage",
		
		// Dating/hookup specific
		"no strings attached", "casual hookup", "one night stand",
		"fuck buddy", "friends with benefits", "hookup tonight",
		
		// Common patterns
		"18+", "adults only", "must be 18", "age verification",
		"enter if 18", "nsfw", "not safe for work",
		
		// Euphemisms
		"escort", "companion", "intimate", "sensual", "erotic",
		"naughty", "kinky", "fetish", "bdsm",
	}
}

// ── Content Inspection ───────────────────────────────────────────

func (ci *ContentInspector) Inspect(domain string) InspectionResult {
	if !ci.enabled {
		return InspectionResult{
			Domain:  domain,
			IsAdult: false,
			Reason:  "content inspector disabled",
		}
	}

	// Check cache first
	ci.mu.RLock()
	if cached, exists := ci.cache[domain]; exists {
		// Cache valid for 24 hours
		if time.Since(cached.Timestamp) < 24*time.Hour {
			ci.mu.RUnlock()
			return InspectionResult{
				Domain:  domain,
				IsAdult: cached.IsAdult,
				Reason:  cached.Reason,
			}
		}
	}
	ci.mu.RUnlock()

	// Fetch and inspect content
	result := ci.fetchAndInspect(domain)

	// Update cache
	ci.mu.Lock()
	ci.cache[domain] = &CacheEntry{
		IsAdult:   result.IsAdult,
		Reason:    result.Reason,
		Timestamp: time.Now(),
	}
	ci.mu.Unlock()

	// Save to database
	ci.saveInspection(result)

	return result
}

func (ci *ContentInspector) fetchAndInspect(domain string) InspectionResult {
	result := InspectionResult{
		Domain:       domain,
		IsAdult:      false,
		Confidence:   0,
		MatchedWords: []string{},
	}

	// Prepare URL (try both HTTP and HTTPS)
	urls := []string{
		"https://" + strings.TrimSuffix(domain, "."),
		"http://" + strings.TrimSuffix(domain, "."),
	}

	var content string
	var fetchErr error

	for _, url := range urls {
		content, fetchErr = ci.fetchContent(url)
		if fetchErr == nil && content != "" {
			break
		}
	}

	if fetchErr != nil || content == "" {
		result.Reason = "failed to fetch content"
		return result
	}

	// Convert to lowercase for matching
	contentLower := strings.ToLower(content)

	// Count matches
	matchCount := 0
	matchedWords := make(map[string]bool)

	for _, word := range ci.suspiciousWords {
		if strings.Contains(contentLower, strings.ToLower(word)) {
			matchCount++
			matchedWords[word] = true
			
			// Stop after finding 5 matches (sufficient evidence)
			if matchCount >= 5 {
				break
			}
		}
	}

	// Convert map to slice
	for word := range matchedWords {
		result.MatchedWords = append(result.MatchedWords, word)
	}

	// Calculate confidence (0-100%)
	// 1 match = 20%, 2 = 40%, 3 = 60%, 4 = 80%, 5+ = 100%
	result.Confidence = float64(matchCount) * 20
	if result.Confidence > 100 {
		result.Confidence = 100
	}

	// Determine if adult (threshold: 40% = 2+ matches)
	if result.Confidence >= 40 {
		result.IsAdult = true
		result.Reason = fmt.Sprintf("content:%d_matches", matchCount)
	} else {
		result.Reason = "content:safe"
	}

	return result
}

func (ci *ContentInspector) fetchContent(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set user agent to look like a regular browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml")

	resp, err := ci.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Only process successful responses
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Read content (limit to 1MB to avoid memory issues)
	limitedReader := io.LimitReader(resp.Body, 1024*1024)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

// ── Database Operations ──────────────────────────────────────────

func (ci *ContentInspector) saveInspection(result InspectionResult) {
	matchedWordsStr := strings.Join(result.MatchedWords, ", ")
	
	// Save to cache table
	ci.db.Exec(`
		INSERT OR REPLACE INTO content_inspection_cache 
		(domain, is_adult, matched_words, confidence, inspected_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, result.Domain, result.IsAdult, matchedWordsStr, result.Confidence)

	// Save to log table
	ci.db.Exec(`
		INSERT INTO content_inspection_log
		(domain, is_adult, matched_words)
		VALUES (?, ?, ?)
	`, result.Domain, result.IsAdult, matchedWordsStr)
}

// ── Cache Management ─────────────────────────────────────────────

func (ci *ContentInspector) cleanupCache() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ci.mu.Lock()
		now := time.Now()
		for domain, entry := range ci.cache {
			// Remove entries older than 24 hours
			if now.Sub(entry.Timestamp) > 24*time.Hour {
				delete(ci.cache, domain)
			}
		}
		ci.mu.Unlock()
	}
}

// ── Configuration ────────────────────────────────────────────────

func (ci *ContentInspector) Enable() {
	ci.enabled = true
}

func (ci *ContentInspector) Disable() {
	ci.enabled = false
}

func (ci *ContentInspector) IsEnabled() bool {
	return ci.enabled
}

func (ci *ContentInspector) AddSuspiciousWord(word string) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	
	word = strings.ToLower(strings.TrimSpace(word))
	if word != "" {
		ci.suspiciousWords = append(ci.suspiciousWords, word)
	}
}

func (ci *ContentInspector) GetSuspiciousWords() []string {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	
	words := make([]string, len(ci.suspiciousWords))
	copy(words, ci.suspiciousWords)
	return words
}

// ── Statistics ───────────────────────────────────────────────────

func (ci *ContentInspector) GetStats() map[string]interface{} {
	var totalInspections, adultFound int
	
	ci.db.QueryRow(`
		SELECT COUNT(*), SUM(CASE WHEN is_adult THEN 1 ELSE 0 END)
		FROM content_inspection_cache
	`).Scan(&totalInspections, &adultFound)

	ci.mu.RLock()
	cacheSize := len(ci.cache)
	ci.mu.RUnlock()

	return map[string]interface{}{
		"total_inspections": totalInspections,
		"adult_found":       adultFound,
		"cache_size":        cacheSize,
		"enabled":           ci.enabled,
		"suspicious_words":  len(ci.suspiciousWords),
	}
}

// ── Batch Inspection ─────────────────────────────────────────────

func (ci *ContentInspector) InspectBatch(domains []string) map[string]InspectionResult {
	results := make(map[string]InspectionResult)
	
	for _, domain := range domains {
		results[domain] = ci.Inspect(domain)
		
		// Small delay to avoid hammering servers
		time.Sleep(100 * time.Millisecond)
	}
	
	return results
}

// ── Clear Cache ──────────────────────────────────────────────────

func (ci *ContentInspector) ClearCache() {
	ci.mu.Lock()
	ci.cache = make(map[string]*CacheEntry)
	ci.mu.Unlock()

	ci.db.Exec("DELETE FROM content_inspection_cache")
}
