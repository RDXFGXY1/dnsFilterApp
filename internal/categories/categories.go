package categories

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

// ════════════════════════════════════════════════════════════════
//  CATEGORY SYSTEM - DNS Filter v2.5
//  Organize domains into toggleable categories
// ════════════════════════════════════════════════════════════════

type Category struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Enabled     bool     `json:"enabled"`
	DomainCount int      `json:"domain_count"`
	Color       string   `json:"color"`
	Domains     []string `json:"domains,omitempty"`
}

type CategoryManager struct {
	db         *sql.DB
	mu         sync.RWMutex
	categories map[string]*Category
}

func NewCategoryManager(db *sql.DB) (*CategoryManager, error) {
	cm := &CategoryManager{
		db:         db,
		categories: make(map[string]*Category),
	}

	if err := cm.initTables(); err != nil {
		return nil, err
	}

	if err := cm.loadCategories(); err != nil {
		return nil, err
	}

	return cm, nil
}

// ── Database Schema ──────────────────────────────────────────────

func (cm *CategoryManager) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			icon TEXT,
			enabled BOOLEAN DEFAULT TRUE,
			color TEXT
		)`,
		
		`CREATE TABLE IF NOT EXISTS category_domains (
			category_id TEXT,
			domain TEXT,
			PRIMARY KEY (category_id, domain),
			FOREIGN KEY (category_id) REFERENCES categories(id)
		)`,
		
		`CREATE INDEX IF NOT EXISTS idx_category_domains_domain 
		 ON category_domains(domain)`,
	}

	for _, query := range queries {
		if _, err := cm.db.Exec(query); err != nil {
			return err
		}
	}

	// Initialize default categories
	return cm.initDefaultCategories()
}

// ── Default Categories ───────────────────────────────────────────

func (cm *CategoryManager) initDefaultCategories() error {
	defaultCategories := []Category{
		{
			ID:          "adult",
			Name:        "Adult Content",
			Description: "Pornography and adult websites",
			Icon:        "🔞",
			Enabled:     true,
			Color:       "#FF0000",
		},
		{
			ID:          "social",
			Name:        "Social Media",
			Description: "Facebook, Instagram, TikTok, Twitter, etc.",
			Icon:        "📱",
			Enabled:     false,
			Color:       "#3B5998",
		},
		{
			ID:          "gaming",
			Name:        "Gaming",
			Description: "Gaming sites and platforms",
			Icon:        "🎮",
			Enabled:     false,
			Color:       "#9146FF",
		},
		{
			ID:          "shopping",
			Name:        "Shopping",
			Description: "E-commerce and shopping sites",
			Icon:        "🛒",
			Enabled:     false,
			Color:       "#FF9900",
		},
		{
			ID:          "streaming",
			Name:        "Streaming",
			Description: "Netflix, YouTube, Twitch, etc.",
			Icon:        "📺",
			Enabled:     false,
			Color:       "#E50914",
		},
		{
			ID:          "gambling",
			Name:        "Gambling",
			Description: "Online casinos and betting sites",
			Icon:        "🎰",
			Enabled:     true,
			Color:       "#FFD700",
		},
		{
			ID:          "news",
			Name:        "News",
			Description: "News websites and media",
			Icon:        "📰",
			Enabled:     false,
			Color:       "#000000",
		},
		{
			ID:          "dating",
			Name:        "Dating",
			Description: "Dating apps and websites",
			Icon:        "💕",
			Enabled:     false,
			Color:       "#FF1493",
		},
		{
			ID:          "crypto",
			Name:        "Cryptocurrency",
			Description: "Crypto trading and mining",
			Icon:        "₿",
			Enabled:     false,
			Color:       "#F7931A",
		},
		{
			ID:          "weapons",
			Name:        "Weapons",
			Description: "Weapon sales and information",
			Icon:        "⚔️",
			Enabled:     true,
			Color:       "#8B0000",
		},
		{
			ID:          "drugs",
			Name:        "Drugs",
			Description: "Drug-related content",
			Icon:        "💊",
			Enabled:     true,
			Color:       "#800080",
		},
		{
			ID:          "piracy",
			Name:        "Piracy",
			Description: "Torrent and piracy sites",
			Icon:        "🏴‍☠️",
			Enabled:     false,
			Color:       "#000000",
		},
	}

	for _, cat := range defaultCategories {
		// Check if exists
		var exists bool
		cm.db.QueryRow("SELECT COUNT(*) > 0 FROM categories WHERE id = ?", cat.ID).Scan(&exists)
		
		if !exists {
			_, err := cm.db.Exec(`
				INSERT INTO categories (id, name, description, icon, enabled, color)
				VALUES (?, ?, ?, ?, ?, ?)
			`, cat.ID, cat.Name, cat.Description, cat.Icon, cat.Enabled, cat.Color)
			
			if err != nil {
				return err
			}
		}
	}

	// Add default domains for each category
	return cm.addDefaultDomains()
}

func (cm *CategoryManager) addDefaultDomains() error {
	domainSets := map[string][]string{
		"adult": {
			"pornhub.com", "xvideos.com", "xnxx.com", "xhamster.com",
			"youporn.com", "redtube.com", "tube8.com", "spankbang.com",
			"eporner.com", "vporn.com", "chaturbate.com", "onlyfans.com",
		},
		"social": {
			"facebook.com", "instagram.com", "tiktok.com", "twitter.com",
			"x.com", "snapchat.com", "reddit.com", "linkedin.com",
			"pinterest.com", "tumblr.com", "whatsapp.com", "telegram.org",
		},
		"gaming": {
			"steam.com", "epicgames.com", "twitch.tv", "roblox.com",
			"minecraft.net", "fortnite.com", "leagueoflegends.com",
			"valorant.com", "playstation.com", "xbox.com", "origin.com",
		},
		"shopping": {
			"amazon.com", "ebay.com", "aliexpress.com", "walmart.com",
			"target.com", "etsy.com", "wish.com", "bestbuy.com",
			"shopify.com", "alibaba.com",
		},
		"streaming": {
			"netflix.com", "youtube.com", "twitch.tv", "hulu.com",
			"disneyplus.com", "primevideo.com", "hbomax.com",
			"crunchyroll.com", "spotify.com", "soundcloud.com",
		},
		"gambling": {
			"bet365.com", "pokerstars.com", "888casino.com",
			"williamhill.com", "betway.com", "draftkings.com",
			"fanduel.com", "bovada.lv",
		},
		"news": {
			"cnn.com", "bbc.com", "nytimes.com", "foxnews.com",
			"theguardian.com", "reuters.com", "apnews.com",
			"washingtonpost.com", "wsj.com",
		},
		"dating": {
			"tinder.com", "bumble.com", "match.com", "okcupid.com",
			"pof.com", "hinge.co", "badoo.com", "eharmony.com",
		},
		"crypto": {
			"binance.com", "coinbase.com", "kraken.com",
			"crypto.com", "kucoin.com", "gate.io", "bitfinex.com",
		},
		"piracy": {
			"thepiratebay.org", "1337x.to", "rarbg.to",
			"yts.mx", "kickasstorrents.to", "torrentz2.eu",
		},
	}

	for categoryID, domains := range domainSets {
		for _, domain := range domains {
			cm.db.Exec(`
				INSERT OR IGNORE INTO category_domains (category_id, domain)
				VALUES (?, ?)
			`, categoryID, domain)
		}
	}

	return nil
}

// ── Category Management ──────────────────────────────────────────

func (cm *CategoryManager) loadCategories() error {
	rows, err := cm.db.Query(`
		SELECT id, name, description, icon, enabled, color
		FROM categories
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	for rows.Next() {
		var cat Category
		err := rows.Scan(&cat.ID, &cat.Name, &cat.Description, 
		                 &cat.Icon, &cat.Enabled, &cat.Color)
		if err != nil {
			continue
		}

		// Count domains
		cm.db.QueryRow(`
			SELECT COUNT(*) FROM category_domains WHERE category_id = ?
		`, cat.ID).Scan(&cat.DomainCount)

		cm.categories[cat.ID] = &cat
	}

	return nil
}

func (cm *CategoryManager) GetAllCategories() []*Category {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	categories := make([]*Category, 0, len(cm.categories))
	for _, cat := range cm.categories {
		categories = append(categories, cat)
	}

	return categories
}

func (cm *CategoryManager) GetCategory(id string) (*Category, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cat, exists := cm.categories[id]
	if !exists {
		return nil, fmt.Errorf("category not found: %s", id)
	}

	// Load domains
	rows, err := cm.db.Query(`
		SELECT domain FROM category_domains WHERE category_id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		rows.Scan(&domain)
		domains = append(domains, domain)
	}

	cat.Domains = domains
	return cat, nil
}

func (cm *CategoryManager) ToggleCategory(id string, enabled bool) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	_, err := cm.db.Exec(`
		UPDATE categories SET enabled = ? WHERE id = ?
	`, enabled, id)

	if err != nil {
		return err
	}

	if cat, exists := cm.categories[id]; exists {
		cat.Enabled = enabled
	}

	return nil
}

func (cm *CategoryManager) AddDomainToCategory(categoryID, domain string) error {
	domain = strings.ToLower(strings.TrimSpace(domain))
	
	_, err := cm.db.Exec(`
		INSERT OR IGNORE INTO category_domains (category_id, domain)
		VALUES (?, ?)
	`, categoryID, domain)

	if err != nil {
		return err
	}

	// Update count
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cat, exists := cm.categories[categoryID]; exists {
		cat.DomainCount++
	}

	return nil
}

func (cm *CategoryManager) RemoveDomainFromCategory(categoryID, domain string) error {
	_, err := cm.db.Exec(`
		DELETE FROM category_domains 
		WHERE category_id = ? AND domain = ?
	`, categoryID, domain)

	if err != nil {
		return err
	}

	// Update count
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if cat, exists := cm.categories[categoryID]; exists {
		cat.DomainCount--
	}

	return nil
}

// ── Domain Checking ──────────────────────────────────────────────

func (cm *CategoryManager) IsBlocked(domain string) (bool, string) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	
	// Remove trailing dot
	domain = strings.TrimSuffix(domain, ".")

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Check if domain is in any enabled category
	var categoryID string
	var enabled bool
	
	err := cm.db.QueryRow(`
		SELECT cd.category_id, c.enabled
		FROM category_domains cd
		JOIN categories c ON cd.category_id = c.id
		WHERE cd.domain = ? AND c.enabled = TRUE
		LIMIT 1
	`, domain).Scan(&categoryID, &enabled)

	if err == nil && enabled {
		return true, categoryID
	}

	// Check subdomains (e.g., www.example.com matches example.com)
	parts := strings.Split(domain, ".")
	for i := 0; i < len(parts)-1; i++ {
		parentDomain := strings.Join(parts[i+1:], ".")
		
		err := cm.db.QueryRow(`
			SELECT cd.category_id, c.enabled
			FROM category_domains cd
			JOIN categories c ON cd.category_id = c.id
			WHERE cd.domain = ? AND c.enabled = TRUE
			LIMIT 1
		`, parentDomain).Scan(&categoryID, &enabled)

		if err == nil && enabled {
			return true, categoryID
		}
	}

	return false, ""
}

func (cm *CategoryManager) GetDomainCategory(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	
	var categoryID string
	cm.db.QueryRow(`
		SELECT category_id FROM category_domains WHERE domain = ?
	`, domain).Scan(&categoryID)

	return categoryID
}

// ── Statistics ───────────────────────────────────────────────────

func (cm *CategoryManager) GetCategoryStats() map[string]int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := make(map[string]int)
	
	for id, cat := range cm.categories {
		stats[id] = cat.DomainCount
	}

	return stats
}

func (cm *CategoryManager) GetEnabledCategories() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var enabled []string
	for id, cat := range cm.categories {
		if cat.Enabled {
			enabled = append(enabled, id)
		}
	}

	return enabled
}

// ── Bulk Operations ──────────────────────────────────────────────

func (cm *CategoryManager) EnableMultiple(categoryIDs []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	tx, err := cm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, id := range categoryIDs {
		_, err := tx.Exec("UPDATE categories SET enabled = TRUE WHERE id = ?", id)
		if err != nil {
			return err
		}
		
		if cat, exists := cm.categories[id]; exists {
			cat.Enabled = true
		}
	}

	return tx.Commit()
}

func (cm *CategoryManager) DisableMultiple(categoryIDs []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	tx, err := cm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, id := range categoryIDs {
		_, err := tx.Exec("UPDATE categories SET enabled = FALSE WHERE id = ?", id)
		if err != nil {
			return err
		}
		
		if cat, exists := cm.categories[id]; exists {
			cat.Enabled = false
		}
	}

	return tx.Commit()
}

// ── Import/Export ────────────────────────────────────────────────

func (cm *CategoryManager) ExportCategory(categoryID string) ([]string, error) {
	rows, err := cm.db.Query(`
		SELECT domain FROM category_domains WHERE category_id = ?
	`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		rows.Scan(&domain)
		domains = append(domains, domain)
	}

	return domains, nil
}

func (cm *CategoryManager) ImportDomains(categoryID string, domains []string) error {
	tx, err := cm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO category_domains (category_id, domain)
		VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain != "" {
			stmt.Exec(categoryID, domain)
		}
	}

	return tx.Commit()
}
