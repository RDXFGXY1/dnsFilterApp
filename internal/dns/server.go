package dns

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/yourusername/dns-filter-app/internal/config"
	"github.com/yourusername/dns-filter-app/internal/database"
	"github.com/yourusername/dns-filter-app/internal/filter"
	"github.com/yourusername/dns-filter-app/pkg/logger"
)

type Server struct {
	cfg          *config.Config
	filter       *filter.Engine
	db           *database.DB
	dnsServer    *dns.Server
	cache        *DNSCache
	upstreamPool *UpstreamPool
	log          *logger.Logger
	stats        *Statistics
}

type Statistics struct {
	mu              sync.RWMutex
	TotalQueries    uint64
	BlockedQueries  uint64
	CachedResponses uint64
	StartTime       time.Time
}

func NewServer(cfg *config.Config, filterEngine *filter.Engine, db *database.DB) (*Server, error) {
	log := logger.Get()

	// Create upstream DNS pool
	upstreamPool := NewUpstreamPool(cfg.Server.UpstreamDNS)

	// Create DNS cache
	cache := NewDNSCache(cfg.Server.CacheSize, time.Duration(cfg.Server.CacheTTL)*time.Second)

	server := &Server{
		cfg:          cfg,
		filter:       filterEngine,
		db:           db,
		cache:        cache,
		upstreamPool: upstreamPool,
		log:          log,
		stats: &Statistics{
			StartTime: time.Now(),
		},
	}

	// Setup DNS server
	dns.HandleFunc(".", server.handleDNSRequest)

	server.dnsServer = &dns.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Server.DNSHost, cfg.Server.DNSPort),
		Net:  "udp",
	}

	return server, nil
}

func (s *Server) Start() error {
	s.log.Infof("DNS server listening on %s", s.dnsServer.Addr)
	return s.dnsServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down DNS server...")
	return s.dnsServer.ShutdownContext(ctx)
}

// ClearCache removes all cached DNS responses
// Call this after updating blocklists so blocked domains take effect immediately
func (s *Server) ClearCache() {
	s.cache.Clear()
	s.log.Info("DNS cache cleared")
}

// GetStats returns current server statistics
func (s *Server) GetStats() (total, blocked, cached uint64) {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()
	return s.stats.TotalQueries, s.stats.BlockedQueries, s.stats.CachedResponses
}

func (s *Server) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	// Increment total queries
	s.stats.mu.Lock()
	s.stats.TotalQueries++
	s.stats.mu.Unlock()

	// Create response message
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	// Get client IP
	clientIP := getClientIP(w)

	// Extract query domain
	if len(r.Question) == 0 {
		w.WriteMsg(m)
		return
	}

	question := r.Question[0]
	domain := question.Name

	// Log query if enabled
	if s.cfg.Logging.LogQueries {
		s.log.Debugf("DNS Query: %s from %s (type: %s)", domain, clientIP, dns.TypeToString[question.Qtype])
	}

	// Check cache first
	if cachedResponse := s.cache.Get(domain, question.Qtype); cachedResponse != nil {
		s.stats.mu.Lock()
		s.stats.CachedResponses++
		s.stats.mu.Unlock()

		cachedResponse.SetReply(r)
		w.WriteMsg(cachedResponse)
		return
	}

	// Check if domain should be blocked
	if s.cfg.Filtering.Enabled && s.filter.ShouldBlock(domain, clientIP) {
		s.handleBlockedDomain(w, r, m, domain, clientIP)
		return
	}

	// Forward to upstream DNS
	s.forwardToUpstream(w, r, m, domain, question.Qtype)
}

func (s *Server) handleBlockedDomain(w dns.ResponseWriter, r *dns.Msg, m *dns.Msg, domain string, clientIP string) {
	// Increment blocked queries
	s.stats.mu.Lock()
	s.stats.BlockedQueries++
	s.stats.mu.Unlock()

	// Log blocked query
	s.log.Infof("BLOCKED: %s from %s", domain, clientIP)

	// Save to database
	s.db.LogBlockedQuery(domain, clientIP, time.Now())

	// Handle based on block action
	switch s.cfg.Filtering.BlockAction {
	case "nxdomain":
		// Return NXDOMAIN (domain not found)
		m.SetRcode(r, dns.RcodeNameError)

	case "redirect":
		// Redirect to specified IP
		if len(r.Question) > 0 && r.Question[0].Qtype == dns.TypeA {
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   r.Question[0].Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    300,
				},
				A: net.ParseIP(s.cfg.Filtering.RedirectIP),
			}
			m.Answer = append(m.Answer, rr)
		}

	case "block_page":
		// Redirect to local block page (127.0.0.1)
		if len(r.Question) > 0 && r.Question[0].Qtype == dns.TypeA {
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   r.Question[0].Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    300,
				},
				A: net.ParseIP("127.0.0.1"),
			}
			m.Answer = append(m.Answer, rr)
		}
	}

	w.WriteMsg(m)
}

func (s *Server) forwardToUpstream(w dns.ResponseWriter, r *dns.Msg, m *dns.Msg, domain string, qtype uint16) {
	// Get upstream DNS server
	upstream := s.upstreamPool.Get()

	// Create DNS client
	client := &dns.Client{
		Timeout: 5 * time.Second,
	}

	// Forward query
	response, _, err := client.Exchange(r, upstream)
	if err != nil {
		s.log.Errorf("Failed to forward DNS query to %s: %v", upstream, err)
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		return
	}

	// Cache successful response
	if response.Rcode == dns.RcodeSuccess && len(response.Answer) > 0 {
		s.cache.Set(domain, qtype, response)
	}

	// Send response
	w.WriteMsg(response)
}

func (s *Server) GetStatistics() map[string]interface{} {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	uptime := time.Since(s.stats.StartTime)
	blockRate := float64(0)
	if s.stats.TotalQueries > 0 {
		blockRate = (float64(s.stats.BlockedQueries) / float64(s.stats.TotalQueries)) * 100
	}

	return map[string]interface{}{
		"total_queries":     s.stats.TotalQueries,
		"blocked_queries":   s.stats.BlockedQueries,
		"cached_responses":  s.stats.CachedResponses,
		"block_rate":        fmt.Sprintf("%.2f%%", blockRate),
		"uptime_seconds":    uptime.Seconds(),
		"uptime_human":      uptime.String(),
		"queries_per_minute": float64(s.stats.TotalQueries) / uptime.Minutes(),
	}
}

func getClientIP(w dns.ResponseWriter) string {
	if addr, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	if addr, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		return addr.IP.String()
	}
	return "unknown"
}
