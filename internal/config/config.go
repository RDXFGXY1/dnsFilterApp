package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Filtering  FilteringConfig  `yaml:"filtering"`
	Database   DatabaseConfig   `yaml:"database"`
	Logging    LoggingConfig    `yaml:"logging"`
	Security   SecurityConfig   `yaml:"security"`
	Blocklists BlocklistsConfig `yaml:"blocklists"`
	Whitelist  WhitelistConfig  `yaml:"whitelist"`
	Advanced   AdvancedConfig   `yaml:"advanced"`
}

type ServerConfig struct {
	DNSPort     int      `yaml:"dns_port"`
	DNSHost     string   `yaml:"dns_host"`
	APIPort     int      `yaml:"api_port"`
	APIHost     string   `yaml:"api_host"`
	UpstreamDNS []string `yaml:"upstream_dns"`
	Workers     int      `yaml:"workers"`
	CacheSize   int      `yaml:"cache_size"`
	CacheTTL    int      `yaml:"cache_ttl"`
}

type FilteringConfig struct {
	Enabled          bool             `yaml:"enabled"`
	BlockAction      string           `yaml:"block_action"`
	RedirectIP       string           `yaml:"redirect_ip"`
	BlockCategories  []string         `yaml:"block_categories"`
	SafeSearch       bool             `yaml:"safe_search"`
	YoutubeRestrict  bool             `yaml:"youtube_restricted"`
	Schedule         ScheduleConfig   `yaml:"schedule"`
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
	Path              string `yaml:"path"`
	MaxLogEntries     int    `yaml:"max_log_entries"`
	LogRetentionDays  int    `yaml:"log_retention_days"`
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
	AdminUsername    string `yaml:"admin_username"`
	AdminPasswordHash string `yaml:"admin_password_hash"`
	JWTSecret        string `yaml:"jwt_secret"`
	SessionTimeout   int    `yaml:"session_timeout"`
	HTTPSEnabled     bool   `yaml:"https_enabled"`
	HTTPSCert        string `yaml:"https_cert"`
	HTTPSKey         string `yaml:"https_key"`
}

type BlocklistsConfig struct {
	AutoUpdateInterval int               `yaml:"auto_update_interval"`
	Sources            []BlocklistSource  `yaml:"sources"`
	CustomPath         string             `yaml:"custom_path"`
}

type BlocklistSource struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Category string `yaml:"category"`
	Enabled  bool   `yaml:"enabled"`
}

type WhitelistConfig struct {
	Domains []string `yaml:"domains"`
}

type AdvancedConfig struct {
	DOHEnabled      bool `yaml:"doh_enabled"`
	DOHPort         int  `yaml:"doh_port"`
	DOTEnabled      bool `yaml:"dot_enabled"`
	DOTPort         int  `yaml:"dot_port"`
	DNSSECEnabled   bool `yaml:"dnssec_enabled"`
	IPv6Enabled     bool `yaml:"ipv6_enabled"`
	BlockPrivateIP  bool `yaml:"block_private_ip"`
	RateLimit       int  `yaml:"rate_limit"`
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
	if cfg.Blocklists.CustomPath == "" {
		cfg.Blocklists.CustomPath = "./configs/custom*.yaml"
	}

	return &cfg, nil
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
