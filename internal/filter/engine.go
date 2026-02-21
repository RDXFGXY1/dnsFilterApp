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
	cfg             *config.Config
	db              *database.DB
	log             *logger.Logger
	blockedDomains  map[string]bool
	customBlocked   map[string]bool
	whitelist       map[string]bool
	mu              sync.RWMutex
	httpClient      *http.Client
}

func New(cfg *config.Config, db *database.DB) (*Engine, error) {
	log := logger.Get()

	// Create HTTP client for downloading blocklists
	// Uses a standard dialer - bootstrap DNS is set in main.go
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

	return engine, nil
}

func (e *Engine) ShouldBlock(domain string, clientIP string) bool {
	domain = normalizeDomain(domain)

	// Never block empty domains
	if domain == "" {
		return false
	}

	// Check whitelist first
	if e.isWhitelisted(domain) {
		return false
	}

	// Check schedule
	if e.cfg.Filtering.Schedule.Enabled && !e.isInAllowedTime() {
		if len(e.cfg.Filtering.Schedule.Rules) > 0 && e.cfg.Filtering.Schedule.Rules[0].StrictMode {
			return true
		}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check custom blocklist
	if e.customBlocked[domain] {
		return true
	}

	// Direct match
	if e.blockedDomains[domain] {
		return true
	}

	// Check subdomains (e.g., ads.example.com -> example.com)
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[i:], ".")
		if e.blockedDomains[parent] || e.customBlocked[parent] {
			return true
		}
	}

	return false
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
// This is faster and used when the user edits custom-blocklist.yaml directly
func (e *Engine) ReloadCustomBlocklists() (int, error) {
	customDomains, count := e.loadCustomYAMLBlocklists()

	e.mu.Lock()
	// Keep existing remote blocklist, just update custom entries in customBlocked
	for domain := range customDomains {
		e.customBlocked[domain] = true
	}
	e.mu.Unlock()

	e.log.Infof("Reloaded %d custom blocklist domains", count)
	return count, nil
}

func (e *Engine) fetchBlocklist(url string) ([]string, error) {
	// Handle local file URLs
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
	// Increase scanner buffer for large files
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
	return len(e.blockedDomains) + len(e.customBlocked)
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
			// Skip localhost entries
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
		// Remove any path or query
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
