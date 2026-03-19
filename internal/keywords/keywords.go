package keywords

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

// ════════════════════════════════════════════════════════════════
//  KEYWORD BLOCKING SYSTEM - DNS Filter v2.5
//  Block domains containing specific keywords
// ════════════════════════════════════════════════════════════════

type KeywordList struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Enabled     bool     `json:"enabled"`
	Category    string   `json:"category"`
	Severity    string   `json:"severity"` // low, medium, high, critical
}

type KeywordManager struct {
	db       *sql.DB
	mu       sync.RWMutex
	lists    map[string]*KeywordList
	keywords map[string][]string // keyword -> list IDs
}

func NewKeywordManager(db *sql.DB) (*KeywordManager, error) {
	km := &KeywordManager{
		db:       db,
		lists:    make(map[string]*KeywordList),
		keywords: make(map[string][]string),
	}

	if err := km.initTables(); err != nil {
		return nil, err
	}

	if err := km.loadKeywordLists(); err != nil {
		return nil, err
	}

	return km, nil
}

// ── Database Schema ──────────────────────────────────────────────

func (km *KeywordManager) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS keyword_lists (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			enabled BOOLEAN DEFAULT TRUE,
			category TEXT,
			severity TEXT
		)`,
		
		`CREATE TABLE IF NOT EXISTS keywords (
			list_id TEXT,
			keyword TEXT,
			PRIMARY KEY (list_id, keyword),
			FOREIGN KEY (list_id) REFERENCES keyword_lists(id)
		)`,
		
		`CREATE INDEX IF NOT EXISTS idx_keywords_keyword 
		 ON keywords(keyword)`,
		
		`CREATE TABLE IF NOT EXISTS keyword_matches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT,
			keyword TEXT,
			list_id TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := km.db.Exec(query); err != nil {
			return err
		}
	}

	return km.initDefaultLists()
}

// ── Default Keyword Lists ────────────────────────────────────────

func (km *KeywordManager) initDefaultLists() error {
	defaultLists := []KeywordList{
		{
			ID:          "adult",
			Name:        "Adult Content",
			Description: "Explicit adult keywords",
			Enabled:     true,
			Category:    "adult",
			Severity:    "critical",
			Keywords: []string{
				// Explicit terms (abbreviated for safety)
				"porn", "xxx", "sex", "nude", "naked", "adult",
				"erotic", "nsfw", "hentai", "cam", "strip",
				"escort", "hookup", "milf", "teen", "amateur",
				// Common patterns
				"redtube", "xvideo", "xhamster", "youporn",
				"pornhub", "xnxx", "chaturbate", "onlyfans",
			},
		},
		{
			ID:          "gambling",
			Name:        "Gambling",
			Description: "Gambling and betting keywords",
			Enabled:     true,
			Category:    "gambling",
			Severity:    "high",
			Keywords: []string{
				"casino", "poker", "bet", "betting", "gamble",
				"slots", "jackpot", "lottery", "roulette",
				"blackjack", "baccarat", "sportsbook",
				"odds", "wager", "bookmaker",
			},
		},
		{
			ID:          "drugs",
			Name:        "Drugs",
			Description: "Drug-related keywords",
			Enabled:     true,
			Category:    "drugs",
			Severity:    "critical",
			Keywords: []string{
				"weed", "marijuana", "cannabis", "cocaine",
				"heroin", "meth", "lsd", "ecstasy", "mdma",
				"drug", "dealer", "high", "stoner", "420",
				"bong", "vape", "cbd", "thc",
			},
		},
		{
			ID:          "violence",
			Name:        "Violence",
			Description: "Violent content keywords",
			Enabled:     true,
			Category:    "violence",
			Severity:    "high",
			Keywords: []string{
				"gore", "blood", "kill", "murder", "death",
				"weapon", "gun", "knife", "torture", "brutal",
				"execution", "massacre", "terrorist",
			},
		},
		{
			ID:          "hate",
			Name:        "Hate Speech",
			Description: "Hate speech and extremism",
			Enabled:     true,
			Category:    "hate",
			Severity:    "critical",
			Keywords: []string{
				"nazi", "kkk", "racist", "fascist", "extremist",
				"supremacist", "genocide", "hate",
			},
		},
		{
			ID:          "piracy",
			Name:        "Piracy",
			Description: "Piracy and illegal downloads",
			Enabled:     false,
			Category:    "piracy",
			Severity:    "medium",
			Keywords: []string{
				"torrent", "pirate", "crack", "keygen", "warez",
				"illegal", "download", "free", "movie", "stream",
				"magnet", "tracker",
			},
		},
		{
			ID:          "dating",
			Name:        "Dating",
			Description: "Dating and hookup sites",
			Enabled:     false,
			Category:    "dating",
			Severity:    "low",
			Keywords: []string{
				"dating", "hookup", "singles", "match", "flirt",
				"romance", "meet", "chat", "date",
			},
		},
		{
			ID:          "weapons",
			Name:        "Weapons",
			Description: "Weapon sales and information",
			Enabled:     true,
			Category:    "weapons",
			Severity:    "high",
			Keywords: []string{
				"gun", "rifle", "pistol", "firearm", "ammunition",
				"ammo", "bullet", "weapon", "arms", "explosive",
			},
		},
		{
			ID:          "suicide",
			Name:        "Self-Harm",
			Description: "Suicide and self-harm content",
			Enabled:     true,
			Category:    "safety",
			Severity:    "critical",
			Keywords: []string{
				"suicide", "selfharm", "cutting", "depression",
				"kill myself", "end it all", "die",
			},
		},
		{
			ID:          "hacking",
			Name:        "Hacking",
			Description: "Hacking and illegal activities",
			Enabled:     false,
			Category:    "security",
			Severity:    "medium",
			Keywords: []string{
				"hack", "exploit", "crack", "malware", "virus",
				"ddos", "botnet", "ransomware", "phishing",
			},
		},
	}

	for _, list := range defaultLists {
		// Check if exists
		var exists bool
		km.db.QueryRow("SELECT COUNT(*) > 0 FROM keyword_lists WHERE id = ?", 
		               list.ID).Scan(&exists)
		
		if !exists {
			// Insert list
			_, err := km.db.Exec(`
				INSERT INTO keyword_lists (id, name, description, enabled, category, severity)
				VALUES (?, ?, ?, ?, ?, ?)
			`, list.ID, list.Name, list.Description, list.Enabled, 
			   list.Category, list.Severity)
			
			if err != nil {
				return err
			}

			// Insert keywords
			for _, keyword := range list.Keywords {
				km.db.Exec(`
					INSERT INTO keywords (list_id, keyword)
					VALUES (?, ?)
				`, list.ID, strings.ToLower(keyword))
			}
		}
	}

	return nil
}

// ── Load Keywords ────────────────────────────────────────────────

func (km *KeywordManager) loadKeywordLists() error {
	// Load lists
	rows, err := km.db.Query(`
		SELECT id, name, description, enabled, category, severity
		FROM keyword_lists
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	km.mu.Lock()
	defer km.mu.Unlock()

	for rows.Next() {
		var list KeywordList
		err := rows.Scan(&list.ID, &list.Name, &list.Description,
		                 &list.Enabled, &list.Category, &list.Severity)
		if err != nil {
			continue
		}

		// Load keywords for this list
		kwRows, err := km.db.Query(`
			SELECT keyword FROM keywords WHERE list_id = ?
		`, list.ID)
		if err != nil {
			continue
		}

		var keywords []string
		for kwRows.Next() {
			var kw string
			kwRows.Scan(&kw)
			keywords = append(keywords, kw)
			
			// Add to reverse lookup map
			km.keywords[kw] = append(km.keywords[kw], list.ID)
		}
		kwRows.Close()

		list.Keywords = keywords
		km.lists[list.ID] = &list
	}

	return nil
}

// ── Keyword Matching ─────────────────────────────────────────────

func (km *KeywordManager) CheckDomain(domain string) (bool, []string, string) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	
	km.mu.RLock()
	defer km.mu.RUnlock()

	var matchedKeywords []string
	var matchedList string
	var severity string

	// Check each enabled list
	for listID, list := range km.lists {
		if !list.Enabled {
			continue
		}

		// Check each keyword in the list
		for _, keyword := range list.Keywords {
			if strings.Contains(domain, keyword) {
				matchedKeywords = append(matchedKeywords, keyword)
				matchedList = listID
				severity = list.Severity

				// Log match
				go km.logMatch(domain, keyword, listID)

				// Return immediately on critical severity
				if severity == "critical" {
					return true, matchedKeywords, matchedList
				}
			}
		}
	}

	if len(matchedKeywords) > 0 {
		return true, matchedKeywords, matchedList
	}

	return false, nil, ""
}

func (km *KeywordManager) logMatch(domain, keyword, listID string) {
	km.db.Exec(`
		INSERT INTO keyword_matches (domain, keyword, list_id)
		VALUES (?, ?, ?)
	`, domain, keyword, listID)
}

// ── List Management ──────────────────────────────────────────────

func (km *KeywordManager) GetAllLists() []*KeywordList {
	km.mu.RLock()
	defer km.mu.RUnlock()

	lists := make([]*KeywordList, 0, len(km.lists))
	for _, list := range km.lists {
		lists = append(lists, list)
	}

	return lists
}

func (km *KeywordManager) GetList(id string) (*KeywordList, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	list, exists := km.lists[id]
	if !exists {
		return nil, fmt.Errorf("list not found: %s", id)
	}

	return list, nil
}

func (km *KeywordManager) ToggleList(id string, enabled bool) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	_, err := km.db.Exec(`
		UPDATE keyword_lists SET enabled = ? WHERE id = ?
	`, enabled, id)

	if err != nil {
		return err
	}

	if list, exists := km.lists[id]; exists {
		list.Enabled = enabled
	}

	return nil
}

func (km *KeywordManager) AddKeyword(listID, keyword string) error {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	
	_, err := km.db.Exec(`
		INSERT OR IGNORE INTO keywords (list_id, keyword)
		VALUES (?, ?)
	`, listID, keyword)

	if err != nil {
		return err
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	if list, exists := km.lists[listID]; exists {
		list.Keywords = append(list.Keywords, keyword)
	}

	km.keywords[keyword] = append(km.keywords[keyword], listID)

	return nil
}

func (km *KeywordManager) RemoveKeyword(listID, keyword string) error {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	
	_, err := km.db.Exec(`
		DELETE FROM keywords WHERE list_id = ? AND keyword = ?
	`, listID, keyword)

	if err != nil {
		return err
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	// Remove from list
	if list, exists := km.lists[listID]; exists {
		for i, kw := range list.Keywords {
			if kw == keyword {
				list.Keywords = append(list.Keywords[:i], list.Keywords[i+1:]...)
				break
			}
		}
	}

	// Remove from reverse lookup
	if listIDs, exists := km.keywords[keyword]; exists {
		for i, id := range listIDs {
			if id == listID {
				km.keywords[keyword] = append(listIDs[:i], listIDs[i+1:]...)
				break
			}
		}
	}

	return nil
}

// ── Statistics ───────────────────────────────────────────────────

func (km *KeywordManager) GetMatchStats(limit int) ([]map[string]interface{}, error) {
	rows, err := km.db.Query(`
		SELECT domain, keyword, list_id, COUNT(*) as count
		FROM keyword_matches
		GROUP BY domain, keyword, list_id
		ORDER BY count DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var domain, keyword, listID string
		var count int
		rows.Scan(&domain, &keyword, &listID, &count)

		stats = append(stats, map[string]interface{}{
			"domain":  domain,
			"keyword": keyword,
			"list_id": listID,
			"count":   count,
		})
	}

	return stats, nil
}

func (km *KeywordManager) GetTotalMatches() int {
	var total int
	km.db.QueryRow("SELECT COUNT(*) FROM keyword_matches").Scan(&total)
	return total
}

// ── Advanced Matching ────────────────────────────────────────────

func (km *KeywordManager) CheckWithWildcard(domain string) (bool, []string) {
	domain = strings.ToLower(domain)
	
	km.mu.RLock()
	defer km.mu.RUnlock()

	var matches []string

	for listID, list := range km.lists {
		if !list.Enabled {
			continue
		}

		for _, keyword := range list.Keywords {
			// Support wildcards
			if strings.HasPrefix(keyword, "*") {
				suffix := strings.TrimPrefix(keyword, "*")
				if strings.HasSuffix(domain, suffix) {
					matches = append(matches, fmt.Sprintf("%s (%s)", keyword, listID))
				}
			} else if strings.HasSuffix(keyword, "*") {
				prefix := strings.TrimSuffix(keyword, "*")
				if strings.HasPrefix(domain, prefix) {
					matches = append(matches, fmt.Sprintf("%s (%s)", keyword, listID))
				}
			} else if strings.Contains(domain, keyword) {
				matches = append(matches, fmt.Sprintf("%s (%s)", keyword, listID))
			}
		}
	}

	return len(matches) > 0, matches
}

// ── Import/Export ────────────────────────────────────────────────

func (km *KeywordManager) ExportList(listID string) ([]string, error) {
	rows, err := km.db.Query(`
		SELECT keyword FROM keywords WHERE list_id = ?
	`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keywords []string
	for rows.Next() {
		var kw string
		rows.Scan(&kw)
		keywords = append(keywords, kw)
	}

	return keywords, nil
}

func (km *KeywordManager) ImportKeywords(listID string, keywords []string) error {
	tx, err := km.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO keywords (list_id, keyword)
		VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" {
			stmt.Exec(listID, keyword)
		}
	}

	return tx.Commit()
}

// ── Bulk Operations ──────────────────────────────────────────────

func (km *KeywordManager) EnableMultiple(listIDs []string) error {
	for _, id := range listIDs {
		if err := km.ToggleList(id, true); err != nil {
			return err
		}
	}
	return nil
}

func (km *KeywordManager) DisableMultiple(listIDs []string) error {
	for _, id := range listIDs {
		if err := km.ToggleList(id, false); err != nil {
			return err
		}
	}
	return nil
}

// ── Clean Up ─────────────────────────────────────────────────────

func (km *KeywordManager) CleanOldMatches(days int) error {
	_, err := km.db.Exec(`
		DELETE FROM keyword_matches 
		WHERE timestamp < datetime('now', '-' || ? || ' days')
	`, days)
	return err
}
