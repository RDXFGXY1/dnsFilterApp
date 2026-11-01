package filter

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/pkg/logger"

	// ✨ NEW IMPORTS - Add these 5 features
	"github.com/RDXFGXY1/dns-filter-app/internal/aiblock"
	"github.com/RDXFGXY1/dns-filter-app/internal/apps"
	"github.com/RDXFGXY1/dns-filter-app/internal/blockpage"
	"github.com/RDXFGXY1/dns-filter-app/internal/categories"
	"github.com/RDXFGXY1/dns-filter-app/internal/gamification"
	"github.com/RDXFGXY1/dns-filter-app/internal/keywords"
	"github.com/RDXFGXY1/dns-filter-app/internal/securitypacks"
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
	securityPacks   *securitypacks.Manager

	// Track current user for gamification (can be set per-device later)
	currentUserID string

	// Detection module toggles (runtime-settable).
	useBlocklist  atomic.Bool
	useCategories atomic.Bool
	useKeywords   atomic.Bool
	useAI         atomic.Bool

	// App Blocker: domain -> appID for currently blocked app bundles.
	appDomains  map[string]string
	blockedApps map[string]bool
	// Focus Mode: domain -> appID active until focusUntil (unix seconds).
	focusDomains map[string]string
	focusUntil   atomic.Int64
	// Static index of ALL bundle domains -> appID, and appID -> category.
	appIndex map[string]string
	appCat   map[string]string
	// Screen-Time Bank: appID -> unlock-expiry (unix seconds).
	tempAllow map[string]int64
	appMu     sync.RWMutex

	// Doom-scroll Guard.
	guardEnabled   atomic.Bool
	guardThreshold atomic.Int64
	guardWindow    atomic.Int64 // seconds
	guardMinutes   atomic.Int64 // block duration
	guardMu        sync.Mutex
	guardHits      map[string][]int64 // clientIP -> hit timestamps
	guardBlocked   map[string]int64   // appID -> expiry
	guardTrips     []GuardEvent
}

// GuardEvent records a doom-scroll intervention.
type GuardEvent struct {
	Time     int64  `json:"time"`
	ClientIP string `json:"client_ip"`
	App      string `json:"app"`
	Minutes  int    `json:"minutes"`
	Message  string `json:"message"`
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

	// Initialize detection-module toggles from config
	engine.useBlocklist.Store(cfg.Filtering.UseBlocklist)
	engine.useCategories.Store(cfg.Filtering.UseCategories)
	engine.useKeywords.Store(cfg.Filtering.UseKeywords)
	engine.useAI.Store(cfg.Filtering.UseAI)

	engine.appDomains = make(map[string]string)
	engine.blockedApps = make(map[string]bool)
	engine.focusDomains = make(map[string]string)
	engine.tempAllow = make(map[string]int64)
	engine.guardHits = make(map[string][]int64)
	engine.guardBlocked = make(map[string]int64)

	// Build the static app index (all bundles) for classification.
	engine.appIndex = make(map[string]string)
	engine.appCat = make(map[string]string)
	for _, b := range apps.All {
		engine.appCat[b.ID] = b.Category
		for _, d := range b.Domains {
			engine.appIndex[normalizeDomain(d)] = b.ID
		}
	}

	// Guard defaults + persisted settings.
	engine.guardThreshold.Store(40)
	engine.guardWindow.Store(60)
	engine.guardMinutes.Store(10)
	engine.loadGuardSettings()

	engine.loadBlockedApps()

	if err := engine.loadPersistentRules(); err != nil {
		return nil, fmt.Errorf("failed to load persistent rules: %w", err)
	}

	packManager, err := securitypacks.New(cfg.SecurityPacks.Directory, cfg.SecurityPacks.TrustedKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize signed security packs: %w", err)
	}
	engine.securityPacks = packManager

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
	e.blockPageServer.SetAllowedCheck(func(domain string) bool {
		blocked, _ := e.evaluate(normalizeDomain(domain))
		return !blocked
	})
	e.blockPageServer.SetDashboardAlias(e.cfg.Server.DashboardDomain, e.cfg.Server.APIPort)
	e.blockPageServer.SetDashboardLAN(e.cfg.Server.DashboardLANEnabled, e.cfg.Server.DashboardAllowedIPs)
	e.blockPageServer.Start()
	e.log.Info("✓ Block page server started on :80")

	// HTTPS block page: local CA + dynamic per-domain certs on :443, so the
	// block page shows for HTTPS sites too (once the CA is trusted).
	if err := e.blockPageServer.EnableTLS("./data/ca", 443); err != nil {
		e.log.Warnf("HTTPS block page unavailable: %v", err)
	} else {
		e.log.Infof("✓ HTTPS block page on :443 — trust the CA at %s", e.blockPageServer.CACertPath())
	}

	e.log.Info("🎉 All new features initialized successfully!")
	return nil
}

func (e *Engine) SetDashboardAlias(domain string) {
	if e.blockPageServer != nil {
		e.blockPageServer.SetDashboardAlias(domain, e.cfg.Server.APIPort)
	}
}

func (e *Engine) SetDashboardLAN(enabled bool, allowedIPs []string) {
	if e.blockPageServer != nil {
		e.blockPageServer.SetDashboardLAN(enabled, allowedIPs)
	}
}

// ShouldBlock decides whether a domain is blocked and records the attempt for
// gamification/AI. Used on the live DNS path.
func (e *Engine) ShouldBlock(domain string, clientIP string) (bool, string) {
	blocked, reason := e.evaluate(domain)
	e.trackBlockAttempt(normalizeDomain(domain), blocked, reason)
	return blocked, reason
}

// TestDomain runs the same block logic read-only (no stats/gamification writes).
// Used by the dashboard "test a domain" tool.
func (e *Engine) TestDomain(domain string) (bool, string) {
	return e.evaluate(domain)
}

// evaluate is the pure block-decision logic with no side effects.
func (e *Engine) evaluate(domain string) (bool, string) {
	domain = normalizeDomain(domain)

	// Never block empty domains
	if domain == "" {
		return false, ""
	}

	// Check whitelist first (highest priority)
	if e.isWhitelisted(domain) {
		return false, "whitelisted"
	}

	// Screen-Time Bank: a paid unlock overrides all blocks for that app.
	if id, ok := matchDomainSet(domain, e.appIndex); ok {
		e.appMu.RLock()
		until := e.tempAllow[id]
		e.appMu.RUnlock()
		if until > time.Now().Unix() {
			return false, "unlocked:" + id
		}
	}

	// Doom-scroll Guard: temporary auto-block after a binge.
	if id, blocked := e.guardCheck(domain); blocked {
		return true, "doomscroll:" + id
	}

	// Focus Mode (blocks distraction apps while a session is running)
	if e.FocusActive() {
		e.appMu.RLock()
		id, ok := matchDomainSet(domain, e.focusDomains)
		e.appMu.RUnlock()
		if ok {
			return true, "focus:" + id
		}
	}

	// App Blocker (explicit 1-click app bundles)
	e.appMu.RLock()
	appID, appOK := matchDomainSet(domain, e.appDomains)
	e.appMu.RUnlock()
	if appOK {
		return true, "app:" + appID
	}

	// Check schedule
	if e.cfg.Filtering.Schedule.Enabled && !e.isInAllowedTime() {
		if len(e.cfg.Filtering.Schedule.Rules) > 0 && e.cfg.Filtering.Schedule.Rules[0].StrictMode {
			return true, "schedule"
		}
	}

	// Categories (fast, high priority)
	if e.useCategories.Load() && e.categoryMgr != nil {
		if blocked, category := e.categoryMgr.IsBlocked(domain); blocked {
			return true, "category:" + category
		}
	}

	// Keywords (fast, catches patterns)
	if e.useKeywords.Load() && e.keywordMgr != nil {
		if blocked, keywords, listID := e.keywordMgr.CheckDomain(domain); blocked {
			if len(keywords) > 0 {
				return true, fmt.Sprintf("keyword:%s:%s", listID, keywords[0])
			}
			return true, fmt.Sprintf("keyword:%s", listID)
		}
	}

	// Custom + downloaded blocklists (including subdomain match)
	if e.useBlocklist.Load() {
		if e.securityPacks != nil {
			if match, blocked := e.securityPacks.Match(domain); blocked {
				return true, fmt.Sprintf("security_pack:%s:%s", match.PackID, match.Category)
			}
		}
		e.mu.RLock()
		customBlocked := e.customBlocked[domain]
		directBlocked := e.blockedDomains[domain]
		e.mu.RUnlock()

		if customBlocked {
			return true, "custom"
		}
		if directBlocked {
			return true, "blocklist"
		}

		parts := strings.Split(domain, ".")
		for i := 1; i < len(parts); i++ {
			parent := strings.Join(parts[i:], ".")

			e.mu.RLock()
			parentBlocked := e.blockedDomains[parent] || e.customBlocked[parent]
			e.mu.RUnlock()

			if parentBlocked {
				return true, "blocklist:subdomain"
			}
		}
	}

	// AI as last resort (slower but catches new threats)
	if e.useAI.Load() && e.aiBlocker != nil {
		result := e.aiBlocker.Predict(domain)
		if result.Blocked {
			return true, fmt.Sprintf("ai:%.0f%%", result.Confidence)
		}
	}

	return false, ""
}

// GetModules returns the current detection-module toggle states.
func (e *Engine) GetModules() map[string]bool {
	return map[string]bool{
		"blocklist":  e.useBlocklist.Load(),
		"categories": e.useCategories.Load(),
		"keywords":   e.useKeywords.Load(),
		"ai":         e.useAI.Load(),
	}
}

// SetModule toggles a detection module at runtime. Returns false if the name is
// unknown.
func (e *Engine) SetModule(name string, on bool) bool {
	switch name {
	case "blocklist":
		e.useBlocklist.Store(on)
		e.cfg.Filtering.UseBlocklist = on
	case "categories":
		e.useCategories.Store(on)
		e.cfg.Filtering.UseCategories = on
	case "keywords":
		e.useKeywords.Store(on)
		e.cfg.Filtering.UseKeywords = on
	case "ai":
		e.useAI.Store(on)
		e.cfg.Filtering.UseAI = on
	default:
		return false
	}
	return true
}

// ✨ NEW METHOD - Track block attempts for gamification.
// Runs asynchronously so the DNS hot path never waits on database writes
// (gamification points, achievements, streaks, AI learning). Blocking on these
// under concurrent queries caused the resolver to stall on blocked domains.
func (e *Engine) trackBlockAttempt(domain string, blocked bool, reason string) {
	if e.gamificationMgr == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				e.log.Warnf("trackBlockAttempt recovered from panic: %v", r)
			}
		}()

		// Award points for successfully resisting blocked sites
		e.gamificationMgr.OnBlockAttempt(e.currentUserID, domain, blocked)

		// If a non-AI system caught it, teach the AI for future predictions.
		if blocked && e.aiBlocker != nil && !strings.HasPrefix(reason, "ai:") {
			category := "suspicious"
			if strings.Contains(reason, "adult") {
				category = "adult"
			} else if strings.Contains(reason, "gambling") {
				category = "gambling"
			}
			e.aiBlocker.LearnFromBlock(domain, category)
		}
	}()
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

// GetAIRecentPredictions returns recent AI predictions for the dashboard
func (e *Engine) GetAIRecentPredictions(limit int) ([]aiblock.PredictionResult, error) {
	if e.aiBlocker == nil {
		return nil, fmt.Errorf("AI blocker not initialized")
	}
	return e.aiBlocker.GetRecentPredictions(limit)
}

func (e *Engine) AnalyzeDomain(domain string) aiblock.PredictionResult {
	if e.aiBlocker == nil {
		return aiblock.PredictionResult{Domain: domain, Category: "unknown", Reason: "Local model unavailable"}
	}
	return e.aiBlocker.Predict(normalizeDomain(domain))
}

// GetKeywordMatchStats returns recent keyword match stats for the dashboard
func (e *Engine) GetKeywordMatchStats(limit int) ([]map[string]interface{}, error) {
	if e.keywordMgr == nil {
		return nil, fmt.Errorf("keywords not initialized")
	}
	return e.keywordMgr.GetMatchStats(limit)
}

// AwardPoints grants gamification points to the current user (used by the
// discipline task system when tasks are completed).
func (e *Engine) AwardPoints(points int, reason string) {
	if e.gamificationMgr != nil {
		e.gamificationMgr.AddPoints(e.currentUserID, points, reason)
	}
}

// GetCACertPath returns the local CA certificate path for the HTTPS block page.
func (e *Engine) GetCACertPath() string {
	if e.blockPageServer == nil {
		return ""
	}
	return e.blockPageServer.CACertPath()
}

// ─── App Blocker ──────────────────────────────────────────────────────────────

// matchDomainSet returns the appID if domain (or a parent domain) is in the set.
func matchDomainSet(domain string, set map[string]string) (string, bool) {
	if id, ok := set[domain]; ok {
		return id, true
	}
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		if id, ok := set[strings.Join(parts[i:], ".")]; ok {
			return id, true
		}
	}
	return "", false
}

// rebuildAppDomains recomputes the blocked-domain lookup from blockedApps.
// Caller must hold appMu.
func (e *Engine) rebuildAppDomains() {
	e.appDomains = make(map[string]string)
	for id := range e.blockedApps {
		if b := apps.ByID(id); b != nil {
			for _, d := range b.Domains {
				e.appDomains[normalizeDomain(d)] = id
			}
		}
	}
}

// loadBlockedApps restores blocked apps from the database on startup.
func (e *Engine) loadBlockedApps() {
	csv, err := e.db.GetSetting("blocked_apps")
	if err != nil || csv == "" {
		return
	}
	e.appMu.Lock()
	for _, id := range strings.Split(csv, ",") {
		id = strings.TrimSpace(id)
		if id != "" && apps.ByID(id) != nil {
			e.blockedApps[id] = true
		}
	}
	e.rebuildAppDomains()
	e.appMu.Unlock()
}

func (e *Engine) persistBlockedApps() {
	ids := make([]string, 0, len(e.blockedApps))
	for id := range e.blockedApps {
		ids = append(ids, id)
	}
	e.db.SetSetting("blocked_apps", strings.Join(ids, ","))
}

// SetAppBlocked blocks or unblocks an app bundle by id.
func (e *Engine) SetAppBlocked(id string, on bool) error {
	if apps.ByID(id) == nil {
		return fmt.Errorf("unknown app: %s", id)
	}
	e.appMu.Lock()
	if on {
		e.blockedApps[id] = true
	} else {
		delete(e.blockedApps, id)
	}
	e.rebuildAppDomains()
	e.persistBlockedApps()
	e.appMu.Unlock()
	return nil
}

// GetApps returns all bundles annotated with their current blocked state.
func (e *Engine) GetApps() []map[string]interface{} {
	e.appMu.RLock()
	defer e.appMu.RUnlock()
	out := make([]map[string]interface{}, 0, len(apps.All))
	for _, b := range apps.All {
		out = append(out, map[string]interface{}{
			"id": b.ID, "name": b.Name, "emoji": b.Emoji,
			"category": b.Category, "blocked": e.blockedApps[b.ID],
		})
	}
	return out
}

// ─── Focus Mode ───────────────────────────────────────────────────────────────

// FocusActive reports whether a focus session is currently running.
func (e *Engine) FocusActive() bool {
	u := e.focusUntil.Load()
	return u > 0 && time.Now().Unix() < u
}

// StartFocus blocks the given app bundles (or the default distraction set) for
// the given number of minutes.
func (e *Engine) StartFocus(minutes int, appIDs []string) {
	if len(appIDs) == 0 {
		appIDs = apps.FocusDefault
	}
	set := make(map[string]string)
	for _, id := range appIDs {
		if b := apps.ByID(id); b != nil {
			for _, d := range b.Domains {
				set[normalizeDomain(d)] = id
			}
		}
	}
	e.appMu.Lock()
	e.focusDomains = set
	e.appMu.Unlock()
	e.focusUntil.Store(time.Now().Add(time.Duration(minutes) * time.Minute).Unix())
}

// StopFocus ends the current focus session immediately.
func (e *Engine) StopFocus() {
	e.focusUntil.Store(0)
}

// FocusStatus returns whether focus is active and the unix time it ends.
func (e *Engine) FocusStatus() (active bool, until int64) {
	u := e.focusUntil.Load()
	if u > 0 && time.Now().Unix() >= u {
		u = 0
	}
	return u > 0, u
}

// ─── Doom-scroll Guard ────────────────────────────────────────────────────────

func (e *Engine) loadGuardSettings() {
	if v, _ := e.db.GetSetting("guard_enabled"); v == "1" {
		e.guardEnabled.Store(true)
	}
	if v, _ := e.db.GetSetting("guard_threshold"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			e.guardThreshold.Store(int64(n))
		}
	}
	if v, _ := e.db.GetSetting("guard_minutes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			e.guardMinutes.Store(int64(n))
		}
	}
}

// guardCheck reports whether a domain is currently guard-blocked.
func (e *Engine) guardCheck(domain string) (string, bool) {
	id, ok := matchDomainSet(domain, e.appIndex)
	if !ok {
		return "", false
	}
	e.guardMu.Lock()
	until := e.guardBlocked[id]
	e.guardMu.Unlock()
	if until > time.Now().Unix() {
		return id, true
	}
	return id, false
}

// NoteActivity records an allowed distraction hit and trips the guard on a binge.
func (e *Engine) NoteActivity(clientIP, domain string) {
	if !e.guardEnabled.Load() {
		return
	}
	id, ok := matchDomainSet(domain, e.appIndex)
	if !ok {
		return
	}
	if cat := e.appCat[id]; cat != "social" && cat != "video" {
		return
	}

	now := time.Now().Unix()
	window := e.guardWindow.Load()
	threshold := int(e.guardThreshold.Load())

	e.guardMu.Lock()
	defer e.guardMu.Unlock()

	hits := e.guardHits[clientIP]
	cutoff := now - window
	kept := hits[:0]
	for _, t := range hits {
		if t > cutoff {
			kept = append(kept, t)
		}
	}
	kept = append(kept, now)
	e.guardHits[clientIP] = kept

	if len(kept) >= threshold {
		mins := int(e.guardMinutes.Load())
		e.guardBlocked[id] = now + int64(mins)*60
		e.guardHits[clientIP] = nil
		ev := GuardEvent{
			Time: now, ClientIP: clientIP, App: id, Minutes: mins,
			Message: fmt.Sprintf("Whoa — you hit %s %d times in %ds. Taking a %d-min breather.", id, len(kept), window, mins),
		}
		e.guardTrips = append([]GuardEvent{ev}, e.guardTrips...)
		if len(e.guardTrips) > 20 {
			e.guardTrips = e.guardTrips[:20]
		}
		e.log.Infof("🌀 Doom-scroll guard tripped: %s (%s) blocked %d min", id, clientIP, mins)
	}
}

// GuardStatus returns the guard config, active blocks, and recent trips.
func (e *Engine) GuardStatus() map[string]interface{} {
	now := time.Now().Unix()
	e.guardMu.Lock()
	active := make([]map[string]interface{}, 0)
	for id, until := range e.guardBlocked {
		if until > now {
			active = append(active, map[string]interface{}{"app": id, "remaining": until - now})
		}
	}
	trips := make([]GuardEvent, len(e.guardTrips))
	copy(trips, e.guardTrips)
	e.guardMu.Unlock()

	return map[string]interface{}{
		"enabled":       e.guardEnabled.Load(),
		"threshold":     e.guardThreshold.Load(),
		"window":        e.guardWindow.Load(),
		"minutes":       e.guardMinutes.Load(),
		"active_blocks": active,
		"trips":         trips,
	}
}

// SetGuard updates the guard configuration and persists it.
func (e *Engine) SetGuard(enabled bool, threshold, minutes int) {
	e.guardEnabled.Store(enabled)
	if threshold > 0 {
		e.guardThreshold.Store(int64(threshold))
	}
	if minutes > 0 {
		e.guardMinutes.Store(int64(minutes))
	}
	if enabled {
		e.db.SetSetting("guard_enabled", "1")
	} else {
		e.db.SetSetting("guard_enabled", "0")
	}
	e.db.SetSetting("guard_threshold", fmt.Sprintf("%d", e.guardThreshold.Load()))
	e.db.SetSetting("guard_minutes", fmt.Sprintf("%d", e.guardMinutes.Load()))
}

// ─── Screen-Time Bank ─────────────────────────────────────────────────────────

const pointsPerToken = 50
const minutesPerToken = 15

// BankStatus returns points, available/spent tokens, and active unlocks.
func (e *Engine) BankStatus() map[string]interface{} {
	points := 0
	if e.gamificationMgr != nil {
		if p, err := e.gamificationMgr.GetUserProfile(e.currentUserID); err == nil && p != nil {
			points = p.Points
		}
	}
	spent := 0
	if v, _ := e.db.GetSetting("tokens_spent"); v != "" {
		spent, _ = strconv.Atoi(v)
	}
	available := points/pointsPerToken - spent
	if available < 0 {
		available = 0
	}

	now := time.Now().Unix()
	e.appMu.RLock()
	unlocks := make([]map[string]interface{}, 0)
	for id, until := range e.tempAllow {
		if until > now {
			unlocks = append(unlocks, map[string]interface{}{"app": id, "remaining": until - now})
		}
	}
	e.appMu.RUnlock()

	return map[string]interface{}{
		"points":            points,
		"tokens_available":  available,
		"tokens_spent":      spent,
		"minutes_per_token": minutesPerToken,
		"active_unlocks":    unlocks,
	}
}

// SpendUnlock spends tokens to temporarily allow an app for the given minutes.
func (e *Engine) SpendUnlock(appID string, minutes int) error {
	if apps.ByID(appID) == nil {
		return fmt.Errorf("unknown app: %s", appID)
	}
	if minutes <= 0 {
		return fmt.Errorf("minutes must be positive")
	}
	cost := (minutes + minutesPerToken - 1) / minutesPerToken // ceil

	status := e.BankStatus()
	if status["tokens_available"].(int) < cost {
		return fmt.Errorf("not enough tokens: need %d, have %d", cost, status["tokens_available"].(int))
	}

	spent := status["tokens_spent"].(int) + cost
	e.db.SetSetting("tokens_spent", fmt.Sprintf("%d", spent))

	e.appMu.Lock()
	e.tempAllow[appID] = time.Now().Add(time.Duration(minutes) * time.Minute).Unix()
	e.appMu.Unlock()
	e.log.Infof("💰 Bank: unlocked %s for %d min (cost %d tokens)", appID, minutes, cost)
	return nil
}

func (e *Engine) isWhitelisted(domain string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.whitelist[domain] {
		return true
	}

	for whitelistedDomain := range e.whitelist {
		if domainMatchesWhitelist(domain, whitelistedDomain) {
			return true
		}
	}

	return false
}

// domainMatchesWhitelist keeps wildcard matching on DNS label boundaries.
// A rule such as *.spotify.com covers spotify.com and every nested subdomain,
// but never a lookalike such as evilspotify.com.
func domainMatchesWhitelist(domain, rule string) bool {
	domain = normalizeDomain(domain)
	rule = normalizeDomain(rule)
	if domain == rule {
		return true
	}
	if !strings.HasPrefix(rule, "*.") {
		return false
	}
	root := strings.TrimPrefix(rule, "*.")
	return domain == root || strings.HasSuffix(domain, "."+root)
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

func (e *Engine) loadPersistentRules() error {
	// Config entries remain a supported baseline, while dashboard-created
	// entries live in SQLite so they survive service and machine restarts.
	for _, domain := range e.cfg.Whitelist.Domains {
		domain = normalizeDomain(domain)
		if domain != "" {
			e.whitelist[domain] = true
		}
	}

	whitelist, err := e.db.GetWhitelist()
	if err != nil {
		return fmt.Errorf("load whitelist: %w", err)
	}
	for _, domain := range whitelist {
		domain = normalizeDomain(domain)
		if domain != "" {
			e.whitelist[domain] = true
		}
	}

	customBlocklist, err := e.db.GetCustomBlocklist()
	if err != nil {
		return fmt.Errorf("load custom blocklist: %w", err)
	}
	for _, domain := range customBlocklist {
		domain = normalizeDomain(domain)
		if domain != "" {
			e.customBlocked[domain] = true
		}
	}
	return nil
}

func (e *Engine) AddToWhitelist(domain string) error {
	domain = normalizeDomain(domain)
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	if err := e.db.AddToWhitelist(domain); err != nil {
		return fmt.Errorf("persist whitelist rule: %w", err)
	}
	e.mu.Lock()
	e.whitelist[domain] = true
	e.mu.Unlock()
	return nil
}

func (e *Engine) RemoveFromWhitelist(domain string) error {
	domain = normalizeDomain(domain)
	if err := e.db.RemoveFromWhitelist(domain); err != nil {
		return fmt.Errorf("remove persisted whitelist rule: %w", err)
	}
	e.mu.Lock()
	delete(e.whitelist, domain)
	e.mu.Unlock()
	return nil
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

func (e *Engine) ListUnblockRequests(status string) ([]blockpage.UnblockRequest, error) {
	if e.blockPageServer == nil {
		return []blockpage.UnblockRequest{}, nil
	}
	return e.blockPageServer.ListUnblockRequests(status)
}

func (e *Engine) GetUnblockRequest(id int) (blockpage.UnblockRequest, error) {
	if e.blockPageServer == nil {
		return blockpage.UnblockRequest{}, fmt.Errorf("block page server unavailable")
	}
	return e.blockPageServer.GetUnblockRequest(id)
}

func (e *Engine) DecideUnblockRequest(id int, status, note string) error {
	if e.blockPageServer == nil {
		return fmt.Errorf("block page server unavailable")
	}
	return e.blockPageServer.DecideUnblockRequest(id, status, note)
}

// ─── Custom Blocklist Methods ─────────────────────────────────────────────────

func (e *Engine) AddToCustomBlocklist(domain string) error {
	domain = normalizeDomain(domain)
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	if err := e.db.AddToCustomBlocklist(domain); err != nil {
		return fmt.Errorf("persist custom block rule: %w", err)
	}
	e.mu.Lock()
	e.customBlocked[domain] = true
	e.mu.Unlock()
	e.log.Infof("Added %s to custom blocklist", domain)
	return nil
}

func (e *Engine) RemoveFromCustomBlocklist(domain string) error {
	domain = normalizeDomain(domain)
	if err := e.db.RemoveFromCustomBlocklist(domain); err != nil {
		return fmt.Errorf("remove persisted custom block rule: %w", err)
	}
	e.mu.Lock()
	delete(e.customBlocked, domain)
	e.mu.Unlock()
	e.log.Infof("Removed %s from custom blocklist", domain)
	return nil
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

func (e *Engine) ListSecurityPacks() ([]securitypacks.Info, int) {
	if e.securityPacks == nil {
		return []securitypacks.Info{}, 0
	}
	return e.securityPacks.List(), e.securityPacks.TrustedKeyCount()
}

func (e *Engine) VerifySecurityPack(data []byte) (securitypacks.Info, error) {
	if e.securityPacks == nil {
		return securitypacks.Info{}, fmt.Errorf("security pack manager is unavailable")
	}
	_, info, err := e.securityPacks.Verify(data)
	return info, err
}

func (e *Engine) InstallSecurityPack(data []byte) (securitypacks.Info, error) {
	if e.securityPacks == nil {
		return securitypacks.Info{}, fmt.Errorf("security pack manager is unavailable")
	}
	return e.securityPacks.Install(data)
}

func (e *Engine) RemoveSecurityPack(id string) error {
	if e.securityPacks == nil {
		return fmt.Errorf("security pack manager is unavailable")
	}
	return e.securityPacks.Remove(id)
}

func (e *Engine) TrustSecurityPackKey(id, publicKey string) error {
	if e.securityPacks == nil {
		return fmt.Errorf("security pack manager is unavailable")
	}
	if err := e.securityPacks.AddTrustedKey(id, publicKey); err != nil {
		return err
	}
	if e.cfg.SecurityPacks.TrustedKeys == nil {
		e.cfg.SecurityPacks.TrustedKeys = make(map[string]string)
	}
	e.cfg.SecurityPacks.TrustedKeys[id] = publicKey
	return e.cfg.SaveDefault()
}

// ─── Stats ────────────────────────────────────────────────────────────────────

func (e *Engine) GetBlockedCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	total := len(e.blockedDomains) + len(e.customBlocked)
	if e.securityPacks != nil {
		total += e.securityPacks.DomainCount()
	}

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
