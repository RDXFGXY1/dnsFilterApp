package main

type Stats struct {
	BlockedDomains int `json:"blocked_domains"`
	TotalBlocked   int `json:"total_blocked"`
	UniqueClients  int `json:"unique_clients"`
	UniqueDomains  int `json:"unique_domains"`
}

type statsResponse struct {
	BlockedDomains int   `json:"blocked_domains"`
	Stats          Stats `json:"stats"`
}

type Query struct {
	Domain    string `json:"domain"`
	ClientIP  string `json:"client_ip"`
	Blocked   bool   `json:"blocked"`
	Timestamp int64  `json:"timestamp"`
}

type BlockedDomain struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

type TimelinePoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type Category struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type SystemInfo struct {
	Version     string   `json:"version"`
	DNSPort     int      `json:"dns_port"`
	DNSHost     string   `json:"dns_host"`
	APIPort     int      `json:"api_port"`
	UpstreamDNS []string `json:"upstream_dns"`
	CacheTTL    int      `json:"cache_ttl"`
}

type HealthCheck struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type ProtectionHealth struct {
	Score       int           `json:"score"`
	Enabled     bool          `json:"enabled"`
	PausedUntil int64         `json:"paused_until"`
	Interface   string        `json:"interface"`
	DNS         string        `json:"dns"`
	Queries     uint64        `json:"queries"`
	Blocked     uint64        `json:"blocked"`
	Cached      uint64        `json:"cached"`
	CheckedAt   string        `json:"checked_at"`
	Checks      []HealthCheck `json:"checks"`
}

type FilteringStatus struct {
	Enabled     bool  `json:"enabled"`
	PausedUntil int64 `json:"paused_until"`
}

type Snapshot struct {
	Stats      Stats
	Queries    []Query
	TopBlocked []BlockedDomain
	Timeline   []TimelinePoint
	Whitelist  []string
	Categories []Category
	System     SystemInfo
	Health     ProtectionHealth
	Filtering  FilteringStatus
}

type View int

const (
	viewOverview View = iota
	viewTraffic
	viewThreats
	viewPolicy
	viewSystem
)

var viewNames = []string{"Overview", "Traffic", "Threats", "Policy", "System"}
