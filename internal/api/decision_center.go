package api

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/aiblock"
	"github.com/RDXFGXY1/dns-filter-app/internal/netdns"
	"github.com/gin-gonic/gin"
	mdns "github.com/miekg/dns"
)

func normalizeDecisionDomain(raw string) string {
	domain := strings.ToLower(strings.TrimSpace(raw))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	if i := strings.IndexAny(domain, "/?#"); i >= 0 {
		domain = domain[:i]
	}
	return strings.TrimSuffix(domain, ".")
}

func domainRuleMatches(domain, rule string) bool {
	rule = strings.TrimPrefix(normalizeDecisionDomain(rule), "*.")
	return domain == rule || strings.HasSuffix(domain, "."+rule)
}

func matchingRules(domain string, rules []string) []string {
	matches := []string{}
	for _, rule := range rules {
		if domainRuleMatches(domain, rule) {
			matches = append(matches, rule)
		}
	}
	sort.Strings(matches)
	return matches
}

func decisionLayer(reason string) string {
	if reason == "whitelisted" || strings.HasPrefix(reason, "unlocked:") {
		return "trusted exception"
	}
	if i := strings.IndexByte(reason, ':'); i >= 0 {
		return reason[:i]
	}
	if reason == "" {
		return "default allow"
	}
	return reason
}

func decisionGuidance(blocked bool, ai aiblock.PredictionResult) (string, string) {
	if blocked {
		return "high", "Keep this domain blocked unless you recognize it and can verify why it is required."
	}
	if ai.Blocked || (ai.Category != "safe" && ai.Category != "unknown" && ai.Confidence >= 0.75) {
		return "review", "The active policy allows this domain, but the local model found a signal worth reviewing before you trust it."
	}
	return "low", "No active blocking rule or strong local risk signal matched this domain. Continue monitoring if it is unfamiliar."
}

func (s *Server) inspectDomainDecision(c *gin.Context) {
	domain := normalizeDecisionDomain(c.Query("domain"))
	if domain == "" || len(domain) > 253 || strings.ContainsAny(domain, " @\\") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid domain is required"})
		return
	}

	blocked, reason := s.filter.TestDomain(domain)
	allowMatches := matchingRules(domain, s.filter.GetWhitelist())
	blockMatches := matchingRules(domain, s.filter.GetCustomBlocklist())
	conflicts := []string{}
	if len(allowMatches) > 0 && len(blockMatches) > 0 {
		conflicts = append(conflicts, "A trusted exception overrides a direct block rule")
	}
	if !blocked && len(blockMatches) > 0 {
		conflicts = append(conflicts, "A higher-priority rule currently overrides the direct block")
	}
	if blocked && len(allowMatches) > 0 {
		conflicts = append(conflicts, "An exception exists but a higher-scope decision still blocked this host")
	}

	var visits int
	var lastBlocked string
	_ = s.db.GetDB().QueryRow(`SELECT COUNT(*), COALESCE(MAX(timestamp), '') FROM blocked_queries WHERE domain = ? OR domain LIKE ?`, domain, "%."+domain).Scan(&visits, &lastBlocked)
	ai := s.filter.AnalyzeDomain(domain)
	risk, recommendation := decisionGuidance(blocked, ai)
	c.JSON(http.StatusOK, gin.H{
		"domain": domain, "decision": map[bool]string{true: "blocked", false: "allowed"}[blocked],
		"blocked": blocked, "reason": reason, "layer": decisionLayer(reason),
		"allow_matches": allowMatches, "block_matches": blockMatches, "conflicts": conflicts,
		"ai": ai, "blocked_visits": visits, "last_blocked": lastBlocked,
		"risk": risk, "recommendation": recommendation,
		"checked_at": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) simulateRule(c *gin.Context) {
	var body struct {
		Pattern string   `json:"pattern"`
		Action  string   `json:"action"`
		Samples []string `json:"samples"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || (body.Action != "block" && body.Action != "allow") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pattern and action (block or allow) are required"})
		return
	}
	pattern := normalizeDecisionDomain(body.Pattern)
	if pattern == "" || len(body.Samples) == 0 || len(body.Samples) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide a pattern and 1–50 sample domains"})
		return
	}
	results := make([]gin.H, 0, len(body.Samples))
	changed := 0
	for _, raw := range body.Samples {
		domain := normalizeDecisionDomain(raw)
		if domain == "" {
			continue
		}
		blocked, reason := s.filter.TestDomain(domain)
		matches := domainRuleMatches(domain, pattern)
		afterBlocked := blocked
		if matches {
			afterBlocked = body.Action == "block"
		}
		if afterBlocked != blocked {
			changed++
		}
		results = append(results, gin.H{"domain": domain, "matches": matches, "before_blocked": blocked, "before_reason": reason, "after_blocked": afterBlocked})
	}
	c.JSON(http.StatusOK, gin.H{"pattern": pattern, "action": body.Action, "changed": changed, "results": results})
}

func (s *Server) listRuleConflicts(c *gin.Context) {
	white := s.filter.GetWhitelist()
	blocks := s.filter.GetCustomBlocklist()
	items := []gin.H{}
	for _, allowed := range white {
		for _, blocked := range blocks {
			if domainRuleMatches(allowed, blocked) || domainRuleMatches(blocked, allowed) {
				items = append(items, gin.H{"allowed": allowed, "blocked": blocked, "winner": "trusted exception", "severity": "warning"})
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"count": len(items), "items": items, "precedence": []string{"trusted exception", "temporary unlock", "focus/guard", "application", "schedule", "category", "keyword", "direct blocklist", "AI", "default allow"}})
}

func (s *Server) protectionHealth(c *gin.Context) {
	enabled, pausedUntil := true, int64(0)
	queries, blocked, cached := uint64(0), uint64(0), uint64(0)
	if s.dnsServer != nil {
		enabled, pausedUntil = s.dnsServer.FilteringStatus()
		queries, blocked, cached = s.dnsServer.GetStats()
	}
	iface, dnsValues, routed, routeErr := netdns.Status()
	checks := []gin.H{
		{"id": "engine", "label": "Filtering engine", "ok": enabled, "detail": map[bool]string{true: "Active and evaluating traffic", false: "Filtering is disabled"}[enabled]},
		{"id": "route", "label": "System DNS route", "ok": routed, "detail": map[bool]string{true: "Current adapter DNS is local; browser secure DNS and other adapters are not verified", false: "External or unavailable DNS detected; check IPv4 and IPv6 routing"}[routed]},
		{"id": "blockpage", "label": "Block-page recovery", "ok": s.filter != nil, "detail": "Live re-check and stale-connection recovery enabled"},
		{"id": "telemetry", "label": "Traffic telemetry", "ok": queries > 0, "detail": map[bool]string{true: "DNS queries are being observed", false: "No queries observed since startup"}[queries > 0]},
	}
	score := 0
	for _, check := range checks {
		if check["ok"].(bool) {
			score += 25
		}
	}
	response := gin.H{"score": score, "checks": checks, "enabled": enabled, "paused_until": pausedUntil, "interface": iface, "dns": dnsValues, "queries": queries, "blocked": blocked, "cached": cached, "checked_at": time.Now().Format(time.RFC3339)}
	if routeErr != nil {
		response["route_error"] = routeErr.Error()
	}
	c.JSON(http.StatusOK, response)
}

func (s *Server) repairProtection(c *gin.Context) {
	steps := []string{}
	if s.dnsServer != nil {
		s.dnsServer.ClearCache()
		s.dnsServer.SetFiltering(true)
		steps = append(steps, "DNS response cache cleared", "Filtering engine enabled")
	}
	if _, _, active, _ := netdns.Status(); !active {
		if err := netdns.SetDNS(); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "partial": true, "steps": steps, "error": err.Error(), "hint": "Run the dashboard as Administrator to repair system DNS routing."})
			return
		}
		steps = append(steps, "System DNS routed through the local filter")
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "steps": steps})
}

func (s *Server) getEmergencyLockdown(c *gin.Context) {
	if s.dnsServer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "DNS server is unavailable"})
		return
	}
	c.JSON(http.StatusOK, s.dnsServer.EmergencyLockdownStatus())
}

func (s *Server) startEmergencyLockdown(c *gin.Context) {
	if s.dnsServer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "DNS server is unavailable"})
		return
	}
	var body struct {
		Minutes        int      `json:"minutes"`
		AllowedDomains []string `json:"allowed_domains"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Minutes < 1 || body.Minutes > 1440 || len(body.AllowedDomains) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide 1–1440 minutes and no more than 100 allowed domains"})
		return
	}
	allowed := make([]string, 0, len(body.AllowedDomains))
	for _, raw := range body.AllowedDomains {
		domain := normalizeDecisionDomain(raw)
		if domain == "" {
			continue
		}
		if _, valid := mdns.IsDomainName(domain); !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid allowed domain: " + raw})
			return
		}
		allowed = append(allowed, domain)
	}
	if err := s.dnsServer.StartEmergencyLockdown(time.Duration(body.Minutes)*time.Minute, allowed); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s.dnsServer.EmergencyLockdownStatus())
}

func (s *Server) stopEmergencyLockdown(c *gin.Context) {
	if s.dnsServer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "DNS server is unavailable"})
		return
	}
	s.dnsServer.StopEmergencyLockdown()
	c.JSON(http.StatusOK, s.dnsServer.EmergencyLockdownStatus())
}
