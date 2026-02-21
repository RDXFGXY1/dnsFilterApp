package dns

import (
	"sync"
	"sync/atomic"
)

type UpstreamPool struct {
	servers []string
	index   uint32
	mu      sync.RWMutex
}

func NewUpstreamPool(servers []string) *UpstreamPool {
	if len(servers) == 0 {
		servers = []string{"8.8.8.8:53"} // Fallback to Google DNS
	}

	return &UpstreamPool{
		servers: servers,
		index:   0,
	}
}

// Get returns the next upstream DNS server using round-robin
func (p *UpstreamPool) Get() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.servers) == 0 {
		return "8.8.8.8:53"
	}

	if len(p.servers) == 1 {
		return p.servers[0]
	}

	// Round-robin selection
	idx := atomic.AddUint32(&p.index, 1)
	return p.servers[idx%uint32(len(p.servers))]
}

// Add adds a new upstream server to the pool
func (p *UpstreamPool) Add(server string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.servers = append(p.servers, server)
}

// Remove removes an upstream server from the pool
func (p *UpstreamPool) Remove(server string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, s := range p.servers {
		if s == server {
			p.servers = append(p.servers[:i], p.servers[i+1:]...)
			break
		}
	}
}

// List returns all upstream servers
func (p *UpstreamPool) List() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	servers := make([]string, len(p.servers))
	copy(servers, p.servers)
	return servers
}
