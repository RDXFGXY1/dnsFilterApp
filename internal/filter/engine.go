package filter

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/pkg/logger"

	// ✨ NEW IMPORTS - Add these 5 features
	"github.com/RDXFGXY1/dns-filter-app/internal/aiblock"
	"github.com/RDXFGXY1/dns-filter-app/internal/blockpage"
	"github.com/RDXFGXY1/dns-filter-app/internal/categories"
	"github.com/RDXFGXY1/dns-filter-app/internal/gamification"
	"github.com/RDXFGXY1/dns-filter-app/internal/keywords"
)

// CustomBlocklistEntry represents a single entry in a custom YAML blocklist
type CustomBlocklistEntry struct {
	Domain   string `yaml:"domain"`
	Category string `yaml:"category"`
	Note     string `yaml:"note"`
	Enabled  bool   `yaml:"enabled"`
}

// CustomBlocklist is the top-level structure of custom*.yaml files
type CustomBlocklist struct {
	Version     string                 `yaml:"version"`
	LastUpdated string                 `yaml:"last_updated"`
	Domains     []CustomBlocklistEntry `yaml:"domains"`
}

type Engine struct {
	cfg            *config.Config
	db             *database.DB
	log            *logger.Logger
	blockedDomains map[string]bool
	customBlocked  map[string]bool
	whitelist      map[string]bool
	mu             sync.RWMutex
	httpClient     *http.Client

	// ✨ NEW FEATURES - Add these fields
	categoryMgr     *categories.CategoryManager
	keywordMgr      *keywords.KeywordManager
	gamificationMgr *gamification.Engine
	aiBlocker       *aiblock.AIBlocker
	blockPageServer *blockpage.BlockPageServer

	// Track current user for gamification (can be set per-device later)
	currentUserID string
}

func New(cfg *config.Config, db *database.DB) (*Engine, error) {
	log := logger.Get()

	// Create HTTP client for downloading blocklists
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	engine := &Engine{
		cfg:            cfg,
		db:             db,
		log:            log,
		blockedDomains: make(map[string]bool),
		customBlocked:  make(map[string]bool),
		whitelist:      make(map[string]bool),
		httpClient:     httpClient,
		currentUserID:  "default_user", // Default user, can be changed per device
	}

	// Load whitelist from config
	for _, domain := range cfg.Whitelist.Domains {
		engine.whitelist[normalizeDomain(domain)] = true
	}

	// Load blocklists from database
	if err := engine.loadBlocklists(); err != nil {
		return nil, fmt.Errorf("failed to load blocklists: %w", err)
	}

	// If database is empty, fetch default blocklists
	if len(engine.blockedDomains) == 0 {
		log.Info("No blocklists found in database, fetching default lists...")
		engine.UpdateBlocklists()
	}

	// ✨ INITIALIZE NEW FEATURES
	if err := engine.initializeNewFeatures(); err != nil {
		log.Warnf("Failed to initialize some new features: %v", err)
		// Don't fail completely, continue with basic functionality
	}

	return engine, nil
}

// ✨ NEW METHOD - Initialize all 5 new features
func (e *Engine) initializeNewFeatures() error {
	// Get SQL database from your database wrapper
	sqlDB := e.db.GetDB() // You'll need to expose this - see database.go changes below

	var err error

	// 1. Initialize Categories
	e.log.Info("Initializing category system...")
	e.categoryMgr, err = categories.NewCategoryManager(sqlDB)
	if err != nil {
		return fmt.Errorf("failed to init categories: %w", err)
	}
	e.log.Infof("✓ Categories loaded: %d categories", len(e.categoryMgr.GetAllCategories()))

	// 2. Initialize Keywords
	e.log.Info("Initializing keyword blocking...")
	e.keywordMgr, err = keywords.NewKeywordManager(sqlDB)
	if err != nil {
		return fmt.Errorf("failed to init keywords: %w", err)
	}
	e.log.Infof("✓ Keywords loaded: %d lists", len(e.keywordMgr.GetAllLists()))

	// 3. Initialize Gamification
	e.log.Info("Initializing gamification system...")
	e.gamificationMgr, err = gamification.NewEngine(sqlDB)
	if err != nil {
		return fmt.Errorf("failed to init gamification: %w", err)
	}
	// Create default user if doesn't exist
	e.gamificationMgr.CreateUser(e.currentUserID, "Default User", "00:00:00:00:00:00")
	e.log.Info("✓ Gamification ready (points, levels, achievements)")

	// 4. Initialize AI Blocker
	e.log.Info("Initializing AI-powered blocking...")
	e.aiBlocker, err = aiblock.NewAIBlocker(sqlDB)
	if err != nil {
		return fmt.Errorf("failed to init AI blocker: %w", err)
	}
	e.log.Info("✓ AI blocker ready (pattern recognition)")

	// 5. Initialize Block Page Server
	e.log.Info("Initializing custom block page server...")
	e.blockPageServer, err = blockpage.NewBlockPageServer(sqlDB, 80)
	if err != nil {
		e.log.Warnf("Failed to initialize block page server: %v", err)
		return fmt.Errorf("failed to init block page server: %w", err)
	}
	e.blockPageServer.Start()
	e.log.Info("✓ Block page server started on :80")

	e.log.Info("🎉 All new features initialized successfully!")
	return nil
}

// ✨ UPDATED METHOD - Now returns reason for blocking
func (e *Engine) ShouldBlock(domain string, clientIP string) (bool, string) {
	domain = normalizeDomain(domain)

	// Never block empty domains
	if domain == "" {
		return false, ""
	}

	// Check whitelist first (highest priority)
	if e.isWhitelisted(domain) {
		return false, "whitelisted"
	}

	// Check schedule
	if e.cfg.Filtering.Schedule.Enabled && !e.isInAllowedTime() {
		if len(e.cfg.Filtering.Schedule.Rules) > 0 && e.cfg.Filtering.Schedule.Rules[0].StrictMode {
			e.trackBlockAttempt(domain, true, "schedule")
			return true, "schedule"
		}
	}

	// ✨ NEW - Check Categories (fast, high priority)
	if e.categoryMgr != nil {
		if blocked, category := e.categoryMgr.IsBlocked(domain); blocked {
			e.trackBlockAttempt(domain, true, "category:"+category)
			return true, "category:" + category
		}
	}

	// ✨ NEW - Check Keywords (fast, catches patterns)
	if e.keywordMgr != nil {
		if blocked, keywords, listID := e.keywordMgr.CheckDomain(domain); blocked {
			reason := fmt.Sprintf("keyword:%s", listID)
			if len(keywords) > 0 {
				reason = fmt.Sprintf("keyword:%s:%s", listID, keywords[0])
			}
			e.trackBlockAttempt(domain, true, reason)
			return true, reason
		}
	}

	e.mu.RLock()
	customBlocked := e.customBlocked[domain]
	directBlocked := e.blockedDomains[domain]
	e.mu.RUnlock()

	// Check custom blocklist
	if customBlocked {
		e.trackBlockAttempt(domain, true, "custom")
		return true, "custom"
	}

	// Direct match in main blocklist
	if directBlocked {
		e.trackBlockAttempt(domain, true, "blocklist")
		return true, "blocklist"
	}

	// Check subdomains (e.g., ads.example.com -> example.com)
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[i:], ".")

		e.mu.RLock()
		parentBlocked := e.blockedDomains[parent] || e.customBlocked[parent]
		e.mu.RUnlock()

		if parentBlocked {
			e.trackBlockAttempt(domain, true, "blocklist:subdomain")
			return true, "blocklist:subdomain"
		}
	}

	// ✨ NEW - Check AI as last resort (slower but catches new threats)
	if e.aiBlocker != nil {
		result := e.aiBlocker.Predict(domain)
		if result.Blocked {
			reason := fmt.Sprintf("ai:%.0f%%", result.Confidence)
			e.trackBlockAttempt(domain, true, reason)
			return true, reason
		}
	}

	// Not blocked - track as allowed
	e.trackBlockAttempt(domain, false, "")
	return false, ""
}

// ✨ NEW METHOD - Track block attempts for gamification
func (e *Engine) trackBlockAttempt(domain string, blocked bool, reason string) {
	if e.gamificationMgr != nil {
		// Award points for successfully resisting blocked sites
		e.gamificationMgr.OnBlockAttempt(e.currentUserID, domain, blocked)

		// If AI caught something new, teach it
		if blocked && e.aiBlocker != nil && strings.HasPrefix(reason, "ai:") {
			// AI already predicted it, no need to re-teach
		} else if blocked && e.aiBlocker != nil {
			// Other system caught it, teach AI for future
			category := "suspicious"
			if strings.Contains(reason, "adult") {
				category = "adult"
			} else if strings.Contains(reason, "gambling") {
				category = "gambling"
			}
			e.aiBlocker.LearnFromBlock(domain, category)
		}
	}
}

// ✨ NEW METHOD - Set user ID for per-device tracking
func (e *Engine) SetCurrentUser(userID string) {
	e.currentUserID = userID
}

// ✨ NEW METHOD - Get user profile for API
func (e *Engine) GetUserProfile(userID string) (*gamification.UserProfile, error) {
	if e.gamificationMgr == nil {
		return nil, fmt.Errorf("gamification not initialized")
	}
	return e.gamificationMgr.GetUserProfile(userID)
}

// ✨ NEW METHOD - Get leaderboard
func (e *Engine) GetLeaderboard(limit int) ([]gamification.LeaderboardEntry, error) {
	if e.gamificationMgr == nil {
		return nil, fmt.Errorf("gamification not initialized")
	}
	return e.gamificationMgr.GetLeaderboard(limit)
}

// ✨ NEW METHOD - Toggle category
func (e *Engine) ToggleCategory(categoryID string, enabled bool) error {
	if e.categoryMgr == nil {
		return fmt.Errorf("categories not initialized")
	}
	return e.categoryMgr.ToggleCategory(categoryID, enabled)
}

// ✨ NEW METHOD - Toggle keyword list
func (e *Engine) ToggleKeywordList(listID string, enabled bool) error {
	if e.keywordMgr == nil {
		return fmt.Errorf("keywords not initialized")
	}
	return e.keywordMgr.ToggleList(listID, enabled)
}

// ✨ NEW METHOD - Get all categories
func (e *Engine) GetAllCategories() []*categories.Category {
	if e.categoryMgr == nil {
		return nil
	}
	return e.categoryMgr.GetAllCategories()
}

// ✨ NEW METHOD - Get all keyword lists
func (e *Engine) GetAllKeywordLists() []*keywords.KeywordList {
	if e.keywordMgr == nil {
		return nil
	}
	return e.keywordMgr.GetAllLists()
}

// ✨ NEW METHOD - Get AI stats
func (e *Engine) GetAIStats() map[string]interface{} {
	if e.aiBlocker == nil {
		return nil
	}
	return e.aiBlocker.GetModelStats()
}

func (e *Engine) isWhitelisted(domain string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.whitelist[domain] {
		return true
	}

	for whitelistedDomain := range e.whitelist {
		if strings.HasPrefix(whitelistedDomain, "*.") {
			pattern := strings.TrimPrefix(whitelistedDomain, "*.")
			if strings.HasSuffix(domain, pattern) {
				return true
			}
		}
	}

	return false
}

func (e *Engine) isInAllowedTime() bool {
	if len(e.cfg.Filtering.Schedule.Rules) == 0 {
		return true
	}

	now := time.Now()
	currentDay := strings.ToLower(now.Weekday().String())
	currentTime := now.Format("15:04")

	for _, rule := range e.cfg.Filtering.Schedule.Rules {
		dayMatch := false
		for _, day := range rule.Days {
			if strings.ToLower(day) == currentDay {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			continue
		}
		if currentTime >= rule.StartTime && currentTime <= rule.EndTime {
			return false
		}
	}

	return true
}

func (e *Engine) UpdateBlocklists() error {
	e.log.Info("Updating blocklists...")

	newBlocked := make(map[string]bool)
	totalDomains := 0

	for _, source := range e.cfg.Blocklists.Sources {
		if !source.Enabled {
			continue
		}

		e.log.Infof("Fetching blocklist: %s", source.Name)

		domains, err := e.fetchBlocklist(source.URL)
		if err != nil {
			e.log.Errorf("Failed to fetch %s: %v", source.Name, err)
			continue
		}

		for _, domain := range domains {
			newBlocked[domain] = true
		}

		totalDomains += len(domains)
		e.log.Infof("Loaded %d domains from %s", len(domains), source.Name)
	}

	// Load and merge custom YAML blocklists
	customDomains, customCount := e.loadCustomYAMLBlocklists()
	for domain := range customDomains {
		newBlocked[domain] = true
	}
	if customCount > 0 {
		e.log.Infof("Loaded %d domains from custom blocklists", customCount)
	}

	e.mu.Lock()
	e.blockedDomains = newBlocked
	e.mu.Unlock()

	if err := e.db.SaveBlocklist(newBlocked); err != nil {
		e.log.Errorf("Failed to save blocklist to database: %v", err)
	}

	e.log.Infof("Blocklist update complete: %d total domains blocked", totalDomains+customCount)
	return nil
}

// loadCustomYAMLBlocklists reads all custom*.yaml files and returns blocked domains
func (e *Engine) loadCustomYAMLBlocklists() (map[string]bool, int) {
	result := make(map[string]bool)
	count := 0

	files, err := filepath.Glob(e.cfg.Blocklists.CustomPath)
	if err != nil || len(files) == 0 {
		return result, 0
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			e.log.Warnf("Failed to read custom blocklist %s: %v", file, err)
			continue
		}

		var bl CustomBlocklist
		if err := yaml.Unmarshal(data, &bl); err != nil {
			e.log.Warnf("Failed to parse custom blocklist %s: %v", file, err)
			continue
		}

		for _, entry := range bl.Domains {
			if !entry.Enabled {
				continue
			}
			domain := normalizeDomain(entry.Domain)
			if domain != "" {
				result[domain] = true
				count++
			}
		}
		e.log.Infof("Loaded custom blocklist: %s (%d enabled domains)", file, count)
	}

	return result, count
}

// ReloadCustomBlocklists reloads only custom YAML blocklists without fetching remote sources
func (e *Engine) ReloadCustomBlocklists() (int, error) {
	customDomains, count := e.loadCustomYAMLBlocklists()

	e.mu.Lock()
	for domain := range customDomains {
		e.customBlocked[domain] = true
	}
	e.mu.Unlock()

	e.log.Infof("Reloaded %d custom blocklist domains", count)
	return count, nil
}

func (e *Engine) fetchBlocklist(url string) ([]string, error) {
	if strings.HasPrefix(url, "file://") {
		return e.fetchLocalBlocklist(strings.TrimPrefix(url, "file://"))
	}

	resp, err := e.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var domains []string
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, len(buf))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		domain := parseDomainFromLine(line)
		if domain != "" {
			domains = append(domains, normalizeDomain(domain))
		}
	}

	return domains, scanner.Err()
}

func (e *Engine) fetchLocalBlocklist(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var domains []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		domain := parseDomainFromLine(line)
		if domain != "" {
			domains = append(domains, normalizeDomain(domain))
		}
	}
	return domains, scanner.Err()
}

func (e *Engine) loadBlocklists() error {
	domains, err := e.db.LoadBlocklist()
	if err != nil {
		return err
	}

	e.mu.Lock()
	e.blockedDomains = domains
	e.mu.Unlock()

	return nil
}

// ─── Whitelist Methods ────────────────────────────────────────────────────────

func (e *Engine) AddToWhitelist(domain string) {
	domain = normalizeDomain(domain)
	e.mu.Lock()
	e.whitelist[domain] = true
	e.mu.Unlock()
	e.db.AddToWhitelist(domain)
}

func (e *Engine) RemoveFromWhitelist(domain string) {
	domain = normalizeDomain(domain)
	e.mu.Lock()
	delete(e.whitelist, domain)
	e.mu.Unlock()
	e.db.RemoveFromWhitelist(domain)
}

func (e *Engine) GetWhitelist() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	list := make([]string, 0, len(e.whitelist))
	for domain := range e.whitelist {
		list = append(list, domain)
	}
	return list
}

// ─── Custom Blocklist Methods ─────────────────────────────────────────────────

func (e *Engine) AddToCustomBlocklist(domain string) {
	domain = normalizeDomain(domain)
	e.mu.Lock()
	e.customBlocked[domain] = true
	e.mu.Unlock()
	e.log.Infof("Added %s to custom blocklist", domain)
}

func (e *Engine) RemoveFromCustomBlocklist(domain string) {
	domain = normalizeDomain(domain)
	e.mu.Lock()
	delete(e.customBlocked, domain)
	e.mu.Unlock()
	e.log.Infof("Removed %s from custom blocklist", domain)
}

func (e *Engine) GetCustomBlocklist() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	list := make([]string, 0, len(e.customBlocked))
	for domain := range e.customBlocked {
		list = append(list, domain)
	}
	return list
}

// ─── Stats ────────────────────────────────────────────────────────────────────

func (e *Engine) GetBlockedCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	total := len(e.blockedDomains) + len(e.customBlocked)

	// ✨ Add category domains to count
	if e.categoryMgr != nil {
		stats := e.categoryMgr.GetCategoryStats()
		for _, count := range stats {
			total += count
		}
	}

	return total
}

func (e *Engine) StartAutoUpdate(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		e.log.Info("Starting scheduled blocklist update...")
		if err := e.UpdateBlocklists(); err != nil {
			e.log.Errorf("Auto-update failed: %v", err)
		}
	}
}

// ✨ NEW METHOD - Cleanup on shutdown
func (e *Engine) Close() error {
	if e.blockPageServer != nil {
		return e.blockPageServer.Stop()
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimSuffix(domain, ".")
	return domain
}

func parseDomainFromLine(line string) string {
	// Hosts file format: 0.0.0.0 example.com or 127.0.0.1 example.com
	if strings.HasPrefix(line, "0.0.0.0") || strings.HasPrefix(line, "127.0.0.1") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			d := fields[1]
			if d == "localhost" || d == "0.0.0.0" || d == "127.0.0.1" {
				return ""
			}
			return d
		}
	}

	// AdBlock format: ||example.com^
	if strings.HasPrefix(line, "||") {
		domain := strings.TrimPrefix(line, "||")
		domain = strings.TrimSuffix(domain, "^")
		if idx := strings.IndexAny(domain, "/^?"); idx != -1 {
			domain = domain[:idx]
		}
		return domain
	}

	// Plain domain
	if !strings.Contains(line, " ") && strings.Contains(line, ".") {
		return line
	}

	return ""
}
