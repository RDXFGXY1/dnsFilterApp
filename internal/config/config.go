package config

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Filtering     FilteringConfig     `yaml:"filtering"`
	Database      DatabaseConfig      `yaml:"database"`
	Logging       LoggingConfig       `yaml:"logging"`
	Security      SecurityConfig      `yaml:"security"`
	Blocklists    BlocklistsConfig    `yaml:"blocklists"`
	SecurityPacks SecurityPacksConfig `yaml:"security_packs"`
	Vault         VaultConfig         `yaml:"vault"`
	Whitelist     WhitelistConfig     `yaml:"whitelist"`
	Advanced      AdvancedConfig      `yaml:"advanced"`
	loadedPath    string
	dnsIntentSet  bool
}

type ServerConfig struct {
	DNSPort         int      `yaml:"dns_port"`
	DNSHost         string   `yaml:"dns_host"`
	APIPort         int      `yaml:"api_port"`
	APIHost         string   `yaml:"api_host"`
	UpstreamDNS     []string `yaml:"upstream_dns"`
	Workers         int      `yaml:"workers"`
	CacheSize       int      `yaml:"cache_size"`
	CacheTTL        int      `yaml:"cache_ttl"`
	ManageSystemDNS bool     `yaml:"manage_system_dns"`
}

type FilteringConfig struct {
	Enabled         bool     `yaml:"enabled"`
	BlockAction     string   `yaml:"block_action"`
	RedirectIP      string   `yaml:"redirect_ip"`
	BlockCategories []string `yaml:"block_categories"`
	SafeSearch      bool     `yaml:"safe_search"`
	YoutubeRestrict bool     `yaml:"youtube_restricted"`

	// Detection modules — which engines contribute to a block decision.
	UseBlocklist         bool `yaml:"use_blocklist"`
	UseCategories        bool `yaml:"use_categories"`
	UseKeywords          bool `yaml:"use_keywords"`
	UseAI                bool `yaml:"use_ai"`
	UseContentInspection bool `yaml:"use_content_inspection"`

	Schedule ScheduleConfig `yaml:"schedule"`
}

type ScheduleConfig struct {
	Enabled bool           `yaml:"enabled"`
	Rules   []ScheduleRule `yaml:"rules"`
}

type ScheduleRule struct {
	Name       string   `yaml:"name"`
	Days       []string `yaml:"days"`
	StartTime  string   `yaml:"start_time"`
	EndTime    string   `yaml:"end_time"`
	StrictMode bool     `yaml:"strict_mode"`
}

type DatabaseConfig struct {
	Path             string `yaml:"path"`
	MaxLogEntries    int    `yaml:"max_log_entries"`
	LogRetentionDays int    `yaml:"log_retention_days"`
}

type LoggingConfig struct {
	Level          string `yaml:"level"`
	File           string `yaml:"file"`
	MaxSizeMB      int    `yaml:"max_size_mb"`
	MaxBackups     int    `yaml:"max_backups"`
	MaxAgeDays     int    `yaml:"max_age_days"`
	LogQueries     bool   `yaml:"log_queries"`
	LogBlockedOnly bool   `yaml:"log_blocked_only"`
}

type SecurityConfig struct {
	AdminUsername     string `yaml:"admin_username"`
	AdminPasswordHash string `yaml:"admin_password_hash"`
	RecoveryKeyHash   string `yaml:"recovery_key_hash,omitempty"`
	SetupComplete     bool   `yaml:"setup_complete"`
	JWTSecret         string `yaml:"jwt_secret"`
	SessionTimeout    int    `yaml:"session_timeout"`
	HTTPSEnabled      bool   `yaml:"https_enabled"`
	HTTPSCert         string `yaml:"https_cert"`
	HTTPSKey          string `yaml:"https_key"`
}

const LegacyExamplePasswordHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// IsSetupComplete keeps existing installations working while requiring a
// clean release template to create its own administrator credentials.
func (s SecurityConfig) IsSetupComplete() bool {
	if s.SetupComplete || s.RecoveryKeyHash != "" {
		return true
	}
	return s.AdminPasswordHash != "" && s.AdminPasswordHash != LegacyExamplePasswordHash
}

type BlocklistsConfig struct {
	AutoUpdateInterval int               `yaml:"auto_update_interval"`
	Sources            []BlocklistSource `yaml:"sources"`
	CustomPath         string            `yaml:"custom_path"`
}

type BlocklistSource struct {
	Name     string `yaml:"name" json:"name"`
	URL      string `yaml:"url" json:"url"`
	Category string `yaml:"category" json:"category"`
	Enabled  bool   `yaml:"enabled" json:"enabled"`
}

type SecurityPacksConfig struct {
	Directory   string            `yaml:"directory"`
	TrustedKeys map[string]string `yaml:"trusted_keys"`
}

type VaultConfig struct {
	Directory      string `yaml:"directory"`
	UnlockMinutes  int    `yaml:"unlock_minutes"`
	MaxFailures    int    `yaml:"max_failures"`
	LockoutMinutes int    `yaml:"lockout_minutes"`
}

type WhitelistConfig struct {
	Domains []string `yaml:"domains"`
}

type AdvancedConfig struct {
	DOHEnabled             bool `yaml:"doh_enabled"`
	DOHPort                int  `yaml:"doh_port"`
	DOTEnabled             bool `yaml:"dot_enabled"`
	DOTPort                int  `yaml:"dot_port"`
	DNSSECEnabled          bool `yaml:"dnssec_enabled"`
	IPv6Enabled            bool `yaml:"ipv6_enabled"`
	BlockPrivateIP         bool `yaml:"block_private_ip"`
	RateLimit              int  `yaml:"rate_limit"`
	SecureDNSOnly          bool `yaml:"secure_dns_only"`
	UpstreamTimeoutSeconds int  `yaml:"upstream_timeout_seconds"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	var raw struct {
		Server map[string]any `yaml:"server"`
	}
	if err := yaml.Unmarshal(data, &raw); err == nil {
		_, cfg.dnsIntentSet = raw.Server["manage_system_dns"]
	}
	cfg.loadedPath = path

	// If no detection module is enabled (e.g. an older config without these
	// keys), default to the sensible set so filtering still works.
	if !cfg.Filtering.UseBlocklist && !cfg.Filtering.UseCategories &&
		!cfg.Filtering.UseKeywords && !cfg.Filtering.UseAI {
		cfg.Filtering.UseBlocklist = true
		cfg.Filtering.UseCategories = true
		cfg.Filtering.UseKeywords = true
	}

	// Set defaults
	if cfg.Server.DNSPort == 0 {
		cfg.Server.DNSPort = 53
	}
	if cfg.Server.APIPort == 0 {
		cfg.Server.APIPort = 8080
	}
	if cfg.Server.Workers == 0 {
		cfg.Server.Workers = 4
	}
	if cfg.Server.CacheSize == 0 {
		cfg.Server.CacheSize = 10000
	}
	if cfg.Server.CacheTTL == 0 {
		cfg.Server.CacheTTL = 3600
	}
	if cfg.Advanced.UpstreamTimeoutSeconds <= 0 {
		cfg.Advanced.UpstreamTimeoutSeconds = 5
	}
	if cfg.Blocklists.CustomPath == "" {
		cfg.Blocklists.CustomPath = "./configs/custom*.yaml"
	}
	if cfg.SecurityPacks.Directory == "" {
		cfg.SecurityPacks.Directory = "./data/security-packs"
	}
	if cfg.SecurityPacks.TrustedKeys == nil {
		cfg.SecurityPacks.TrustedKeys = make(map[string]string)
	}
	if cfg.Vault.Directory == "" {
		cfg.Vault.Directory = "./data/vault"
	}
	if cfg.Vault.UnlockMinutes <= 0 {
		cfg.Vault.UnlockMinutes = 15
	}
	if cfg.Vault.MaxFailures <= 0 {
		cfg.Vault.MaxFailures = 5
	}
	if cfg.Vault.LockoutMinutes <= 0 {
		cfg.Vault.LockoutMinutes = 5
	}

	return &cfg, nil
}

// HasSystemDNSPreference reports whether this installation explicitly stores
// the routing preference. It lets older installations migrate once without
// treating the default false value as an explicit opt-out.
func (c *Config) HasSystemDNSPreference() bool { return c.dnsIntentSet }

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".dns-filter-config-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary config file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to secure temporary config file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temporary config file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to flush temporary config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temporary config file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace config file: %w", err)
	}
	c.loadedPath = path
	c.dnsIntentSet = true
	return nil
}

// SaveDefault persists the config back to the file it was loaded from.
func (c *Config) SaveDefault() error {
	if c.loadedPath == "" {
		return fmt.Errorf("no config path known")
	}
	return c.Save(c.loadedPath)
}

// ReplaceWith atomically persists and activates a replacement configuration
// while retaining the installation-specific file path.
func (c *Config) ReplaceWith(next Config) error {
	next.loadedPath = c.loadedPath
	if err := next.SaveDefault(); err != nil {
		return err
	}
	*c = next
	return nil
}

// HashPassword returns a bcrypt hash for the given plaintext password.
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(h), err
}
