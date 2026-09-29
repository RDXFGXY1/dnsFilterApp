package dns

import (
	"errors"
	"sort"
	"strings"
	"time"
)

const maxEmergencyLockdownDuration = 24 * time.Hour

type EmergencyLockdownStatus struct {
	Active         bool     `json:"active"`
	Until          int64    `json:"until"`
	AllowedDomains []string `json:"allowed_domains"`
}

// StartEmergencyLockdown blocks every DNS name except the supplied domains and
// their subdomains. It is deliberately runtime-only and expires automatically.
func (s *Server) StartEmergencyLockdown(duration time.Duration, allowedDomains []string) error {
	if duration < time.Minute || duration > maxEmergencyLockdownDuration {
		return errors.New("emergency lockdown duration must be between 1 minute and 24 hours")
	}

	seen := make(map[string]struct{}, len(allowedDomains))
	allowed := make([]string, 0, len(allowedDomains))
	for _, domain := range allowedDomains {
		domain = normalizeLockdownDomain(domain)
		if domain == "" {
			continue
		}
		if _, exists := seen[domain]; exists {
			continue
		}
		seen[domain] = struct{}{}
		allowed = append(allowed, domain)
	}
	sort.Strings(allowed)

	s.lockdownMu.Lock()
	s.lockdownUntil = time.Now().Add(duration).Unix()
	s.lockdownAllow = allowed
	s.lockdownMu.Unlock()
	if s.cache != nil {
		s.ClearCache()
	}
	return nil
}

func (s *Server) StopEmergencyLockdown() {
	s.lockdownMu.Lock()
	s.lockdownUntil = 0
	s.lockdownAllow = nil
	s.lockdownMu.Unlock()
	if s.cache != nil {
		s.ClearCache()
	}
}

func (s *Server) EmergencyLockdownStatus() EmergencyLockdownStatus {
	now := time.Now().Unix()
	s.lockdownMu.RLock()
	until := s.lockdownUntil
	allowed := append([]string(nil), s.lockdownAllow...)
	s.lockdownMu.RUnlock()
	if until <= now {
		if until != 0 {
			s.lockdownMu.Lock()
			if s.lockdownUntil <= now {
				s.lockdownUntil = 0
				s.lockdownAllow = nil
			}
			s.lockdownMu.Unlock()
		}
		return EmergencyLockdownStatus{AllowedDomains: []string{}}
	}
	return EmergencyLockdownStatus{Active: true, Until: until, AllowedDomains: allowed}
}

// lockdownDecision returns whether a domain is blocked and whether lockdown is
// currently active. Allowed parents also allow their subdomains.
func (s *Server) lockdownDecision(rawDomain string) (blocked bool, active bool) {
	status := s.EmergencyLockdownStatus()
	if !status.Active {
		return false, false
	}
	domain := normalizeLockdownDomain(rawDomain)
	for _, allowed := range status.AllowedDomains {
		if domain == allowed || strings.HasSuffix(domain, "."+allowed) {
			return false, true
		}
	}
	return true, true
}

func normalizeLockdownDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	if i := strings.IndexAny(domain, "/?#"); i >= 0 {
		domain = domain[:i]
	}
	return strings.Trim(strings.TrimSpace(domain), ".")
}
