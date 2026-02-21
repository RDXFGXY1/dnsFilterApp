package dns

import (
	"sync"
	"time"

	"github.com/miekg/dns"
)

type cacheEntry struct {
	response  *dns.Msg
	timestamp time.Time
}

type DNSCache struct {
	mu       sync.RWMutex
	entries  map[string]*cacheEntry
	maxSize  int
	ttl      time.Duration
}

func NewDNSCache(maxSize int, ttl time.Duration) *DNSCache {
	cache := &DNSCache{
		entries: make(map[string]*cacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

func (c *DNSCache) Get(domain string, qtype uint16) *dns.Msg {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.makeKey(domain, qtype)
	entry, exists := c.entries[key]

	if !exists {
		return nil
	}

	// Check if entry is expired
	if time.Since(entry.timestamp) > c.ttl {
		return nil
	}

	// Return a copy of the cached response
	return entry.response.Copy()
}

func (c *DNSCache) Set(domain string, qtype uint16, response *dns.Msg) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict old entries if cache is full
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	key := c.makeKey(domain, qtype)
	c.entries[key] = &cacheEntry{
		response:  response.Copy(),
		timestamp: time.Now(),
	}
}

func (c *DNSCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
}

func (c *DNSCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

func (c *DNSCache) makeKey(domain string, qtype uint16) string {
	return domain + ":" + dns.TypeToString[qtype]
}

func (c *DNSCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	first := true
	for key, entry := range c.entries {
		if first || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *DNSCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.Sub(entry.timestamp) > c.ttl {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}
