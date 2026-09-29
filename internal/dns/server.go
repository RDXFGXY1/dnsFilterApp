package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/internal/filter"
	"github.com/RDXFGXY1/dns-filter-app/pkg/logger"
	"github.com/miekg/dns"
)

// QueryEvent is a single DNS query outcome, streamed to live-monitor clients.
type QueryEvent struct {
	Time     int64  `json:"time"`
	Domain   string `json:"domain"`
	ClientIP string `json:"client_ip"`
	Type     string `json:"type"`
	Blocked  bool   `json:"blocked"`
	Reason   string `json:"reason"`
	Cached   bool   `json:"cached"`
}

type Server struct {
	cfg          *config.Config
	filter       *filter.Engine
	db           *database.DB
	dnsServer    *dns.Server
	localIPv6    *dns.Server
	cache        *DNSCache
	upstreamPool *UpstreamPool
	log          *logger.Logger
	stats        *Statistics

	filterEnabled atomic.Bool  // runtime on/off toggle
	pauseUntil    atomic.Int64 // unix seconds; 0 = not paused
	blockAction   atomic.Value // string: nxdomain | redirect | block_page
	lockdownMu    sync.RWMutex
	lockdownUntil int64
	lockdownAllow []string

	// Live-query pub/sub for the activity monitor.
	subs   map[chan QueryEvent]struct{}
	subsMu sync.Mutex
}

// Subscribe returns a channel that receives live query events. Call Unsubscribe
// when done.
func (s *Server) Subscribe() chan QueryEvent {
	ch := make(chan QueryEvent, 128)
	s.subsMu.Lock()
	if s.subs == nil {
		s.subs = make(map[chan QueryEvent]struct{})
	}
	s.subs[ch] = struct{}{}
	s.subsMu.Unlock()
	return ch
}

// Unsubscribe removes and closes a subscriber channel.
func (s *Server) Unsubscribe(ch chan QueryEvent) {
	s.subsMu.Lock()
	if _, ok := s.subs[ch]; ok {
		delete(s.subs, ch)
		close(ch)
	}
	s.subsMu.Unlock()
}

// publish fans an event out to subscribers, dropping it for any slow consumer.
func (s *Server) publish(ev QueryEvent) {
	s.subsMu.Lock()
	if len(s.subs) == 0 {
		s.subsMu.Unlock()
		return
	}
	for ch := range s.subs {
		select {
		case ch <- ev:
		default: // subscriber is slow; drop this event for it
		}
	}
	s.subsMu.Unlock()
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
	upstreamPool := NewUpstreamPool(cfg.Server.UpstreamDNS, cfg.Advanced.SecureDNSOnly)
	if err := upstreamPool.Validate(); err != nil {
		return nil, fmt.Errorf("invalid upstream DNS configuration: %w", err)
	}

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

	server.filterEnabled.Store(cfg.Filtering.Enabled)
	action := cfg.Filtering.BlockAction
	if action == "" {
		action = "nxdomain"
	}
	server.blockAction.Store(action)

	// Setup DNS server
	dns.HandleFunc(".", server.handleDNSRequest)

	server.dnsServer = &dns.Server{
		Addr: net.JoinHostPort(cfg.Server.DNSHost, fmt.Sprint(cfg.Server.DNSPort)),
		Net:  "udp",
	}
	if ip := net.ParseIP(cfg.Server.DNSHost); ip != nil && ip.To4() != nil {
		server.dnsServer.Net = "udp4"
		server.localIPv6 = &dns.Server{Addr: net.JoinHostPort("::1", fmt.Sprint(cfg.Server.DNSPort)), Net: "udp6"}
	}

	return server, nil
}

func (s *Server) Start() error {
	s.log.Infof("DNS server listening on %s", s.dnsServer.Addr)
	if s.localIPv6 != nil {
		// Bind IPv6 before starting either listener, so startup cannot claim
		// success while Windows is routed to an unavailable ::1 resolver.
		conn, err := net.ListenPacket("udp6", s.localIPv6.Addr)
		if err != nil {
			return fmt.Errorf("local IPv6 DNS listener: %w", err)
		}
		s.localIPv6.PacketConn = conn
		go func() {
			if err := s.localIPv6.ActivateAndServe(); err != nil {
				s.log.Errorf("IPv6 DNS listener failed: %v", err)
			}
		}()
		defer conn.Close()
	}
	return s.dnsServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down DNS server...")
	if s.localIPv6 != nil {
		_ = s.localIPv6.ShutdownContext(ctx)
	}
	return s.dnsServer.ShutdownContext(ctx)
}

// FilteringActive reports whether filtering is currently applied (enabled and
// not within a temporary pause window).
func (s *Server) FilteringActive() bool {
	if !s.filterEnabled.Load() {
		return false
	}
	if pu := s.pauseUntil.Load(); pu > 0 && time.Now().Unix() < pu {
		return false
	}
	return true
}

// SetFiltering turns filtering on or off and clears any active pause.
func (s *Server) SetFiltering(on bool) {
	s.filterEnabled.Store(on)
	s.pauseUntil.Store(0)
	s.ClearCache()
}

// PauseFiltering disables filtering for the given duration, then it resumes.
func (s *Server) PauseFiltering(d time.Duration) {
	s.pauseUntil.Store(time.Now().Add(d).Unix())
	s.ClearCache()
}

// SetBlockAction changes how blocked domains are answered at runtime.
func (s *Server) SetBlockAction(action string) {
	s.blockAction.Store(action)
	s.ClearCache()
}

// BlockAction returns the current block action.
func (s *Server) BlockAction() string {
	a, _ := s.blockAction.Load().(string)
	return a
}

// FilteringStatus returns whether filtering is enabled and, if paused, the unix
// timestamp when it resumes (0 = not paused).
func (s *Server) FilteringStatus() (enabled bool, pausedUntil int64) {
	pu := s.pauseUntil.Load()
	if pu > 0 && time.Now().Unix() >= pu {
		pu = 0
	}
	return s.filterEnabled.Load(), pu
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
	qtype := dns.TypeToString[question.Qtype]

	// Log query if enabled
	if s.cfg.Logging.LogQueries {
		s.log.Debugf("DNS Query: %s from %s (type: %s)", domain, clientIP, qtype)
	}

	// Emergency lockdown is evaluated before cache and normal filtering so a
	// previously cached answer cannot bypass an active lockdown.
	if blocked, active := s.lockdownDecision(domain); active && blocked {
		const reason = "emergency_lockdown"
		s.log.Warnf("EMERGENCY LOCKDOWN: blocked %s from %s", domain, clientIP)
		s.publish(QueryEvent{Time: time.Now().Unix(), Domain: strings.TrimSuffix(domain, "."), ClientIP: clientIP, Type: qtype, Blocked: true, Reason: reason})
		s.handleBlockedDomain(w, r, m, domain, clientIP, reason)
		return
	}

	// Check cache first
	if cachedResponse := s.cache.Get(domain, question.Qtype); cachedResponse != nil {
		s.stats.mu.Lock()
		s.stats.CachedResponses++
		s.stats.mu.Unlock()

		cachedResponse.SetReply(r)
		w.WriteMsg(cachedResponse)
		s.publish(QueryEvent{Time: time.Now().Unix(), Domain: strings.TrimSuffix(domain, "."), ClientIP: clientIP, Type: qtype, Cached: true})
		return
	}

	// Check if domain should be blocked
	if s.FilteringActive() {
		blocked, reason := s.filter.ShouldBlock(domain, clientIP)
		if blocked {
			s.log.Infof("🛡️  BLOCKED: %s (reason: %s)", domain, reason)
			s.publish(QueryEvent{Time: time.Now().Unix(), Domain: strings.TrimSuffix(domain, "."), ClientIP: clientIP, Type: qtype, Blocked: true, Reason: reason})
			s.handleBlockedDomain(w, r, m, domain, clientIP, reason)
			return
		}
	}

	s.publish(QueryEvent{Time: time.Now().Unix(), Domain: strings.TrimSuffix(domain, "."), ClientIP: clientIP, Type: qtype})
	s.filter.NoteActivity(clientIP, strings.TrimSuffix(domain, "."))

	// Forward to upstream DNS
	s.forwardToUpstream(w, r, m, domain, question.Qtype)
}

func (s *Server) handleBlockedDomain(w dns.ResponseWriter, r *dns.Msg, m *dns.Msg, domain string, clientIP string, reason string) {
	// Increment blocked queries
	s.stats.mu.Lock()
	s.stats.BlockedQueries++
	s.stats.mu.Unlock()

	// Log blocked query
	s.log.Infof("BLOCKED: %s from %s", domain, clientIP)

	// Save to database
	s.db.LogBlockedQuery(strings.TrimSuffix(domain, "."), clientIP, time.Now())

	// Handle based on block action (runtime-settable)
	action, _ := s.blockAction.Load().(string)
	switch action {
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
					// Block answers must expire almost immediately. A long TTL pins
					// the browser to the redirect IP after the rule is removed.
					Ttl: 1,
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
					// Keep the block-page redirect deliberately short-lived so an
					// unblock takes effect without restarting the browser.
					Ttl: 1,
				},
				A: net.ParseIP("127.0.0.1"), // Assuming block page server runs on this IP:port
			}
			m.Answer = append(m.Answer, rr)
		}
	}

	w.WriteMsg(m)
}

func (s *Server) forwardToUpstream(w dns.ResponseWriter, r *dns.Msg, m *dns.Msg, domain string, qtype uint16) {
	timeout := time.Duration(s.cfg.Advanced.UpstreamTimeoutSeconds) * time.Second
	response, upstream, err := s.upstreamPool.Exchange(context.Background(), r, timeout)
	if err != nil {
		s.log.Errorf("Failed to forward DNS query: %v", err)
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		return
	}
	s.log.Debugf("DNS query forwarded through %s", upstream)

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
		"total_queries":      s.stats.TotalQueries,
		"blocked_queries":    s.stats.BlockedQueries,
		"cached_responses":   s.stats.CachedResponses,
		"block_rate":         fmt.Sprintf("%.2f%%", blockRate),
		"uptime_seconds":     uptime.Seconds(),
		"uptime_human":       uptime.String(),
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
