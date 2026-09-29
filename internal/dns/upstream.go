package dns

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mdns "github.com/miekg/dns"
)

const maxDNSMessageSize = 65535

type upstreamProtocol string

const (
	protocolUDP upstreamProtocol = "udp"
	protocolTCP upstreamProtocol = "tcp"
	protocolDoT upstreamProtocol = "dot"
	protocolDoH upstreamProtocol = "doh"
)

type upstreamEndpoint struct {
	raw        string
	protocol   upstreamProtocol
	address    string
	url        *url.URL
	serverName string
}

func (e upstreamEndpoint) encrypted() bool {
	return e.protocol == protocolDoH || e.protocol == protocolDoT
}

type UpstreamPool struct {
	endpoints  []upstreamEndpoint
	index      uint32
	mu         sync.RWMutex
	secureOnly bool
	httpClient *http.Client
	parseErrs  []error
}

// NewUpstreamPool builds a round-robin upstream pool. Entries may be ordinary
// host:port values, udp:// or tcp:// addresses, tls://host:853 (DoT), or an
// https://.../dns-query URL (DoH). In secure-only mode plaintext entries are
// ignored rather than used as a silent fallback.
func NewUpstreamPool(servers []string, secureOnly ...bool) *UpstreamPool {
	requireEncryption := len(secureOnly) > 0 && secureOnly[0]
	if len(servers) == 0 {
		if requireEncryption {
			servers = []string{"https://1.1.1.1/dns-query", "tls://1.1.1.1:853"}
		} else {
			servers = []string{"8.8.8.8:53"}
		}
	}

	p := &UpstreamPool{secureOnly: requireEncryption, httpClient: newDoHClient()}
	for _, server := range servers {
		endpoint, err := parseUpstreamEndpoint(server)
		if err != nil {
			p.parseErrs = append(p.parseErrs, err)
			continue
		}
		if requireEncryption && !endpoint.encrypted() {
			continue
		}
		p.endpoints = append(p.endpoints, endpoint)
	}
	return p
}

func newDoHClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("DoH redirects are disabled")
		},
	}
}

func parseUpstreamEndpoint(value string) (upstreamEndpoint, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return upstreamEndpoint{}, errors.New("empty upstream DNS endpoint")
	}

	if strings.HasPrefix(strings.ToLower(raw), "https://") {
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" {
			return upstreamEndpoint{}, fmt.Errorf("invalid DoH endpoint %q", raw)
		}
		if u.User != nil {
			return upstreamEndpoint{}, fmt.Errorf("DoH endpoint %q must not contain credentials", raw)
		}
		return upstreamEndpoint{raw: raw, protocol: protocolDoH, url: u}, nil
	}

	lower := strings.ToLower(raw)
	protocol := protocolUDP
	address := raw
	switch {
	case strings.HasPrefix(lower, "tls://"):
		protocol = protocolDoT
		address = raw[len("tls://"):]
	case strings.HasPrefix(lower, "tcp://"):
		protocol = protocolTCP
		address = raw[len("tcp://"):]
	case strings.HasPrefix(lower, "udp://"):
		address = raw[len("udp://"):]
	case strings.Contains(raw, "://"):
		return upstreamEndpoint{}, fmt.Errorf("unsupported DNS upstream scheme in %q", raw)
	}

	defaultPort := "53"
	if protocol == protocolDoT {
		defaultPort = "853"
	}
	host, port, err := splitHostPort(address, defaultPort)
	if err != nil {
		return upstreamEndpoint{}, fmt.Errorf("invalid DNS upstream %q: %w", raw, err)
	}
	endpoint := upstreamEndpoint{raw: raw, protocol: protocol, address: net.JoinHostPort(host, port)}
	if protocol == protocolDoT {
		endpoint.serverName = host
	}
	return endpoint, nil
}

func splitHostPort(address, defaultPort string) (string, string, error) {
	if host, port, err := net.SplitHostPort(address); err == nil {
		if host == "" || port == "" {
			return "", "", errors.New("host and port are required")
		}
		return host, port, nil
	}

	trimmed := strings.Trim(address, "[]")
	if net.ParseIP(trimmed) != nil || !strings.Contains(trimmed, ":") {
		if trimmed == "" {
			return "", "", errors.New("host is required")
		}
		return trimmed, defaultPort, nil
	}
	return "", "", errors.New("IPv6 addresses with a port must use [address]:port")
}

func (p *UpstreamPool) Validate() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.endpoints) > 0 {
		return nil
	}
	if p.secureOnly {
		return errors.New("secure DNS mode requires at least one https:// (DoH) or tls:// (DoT) upstream")
	}
	if len(p.parseErrs) > 0 {
		return fmt.Errorf("no valid upstream DNS endpoints: %w", p.parseErrs[0])
	}
	return errors.New("no upstream DNS endpoints configured")
}

// Exchange tries every configured endpoint, beginning at a rotating position.
// This supplies failover without ever downgrading when secure-only mode is on.
func (p *UpstreamPool) Exchange(ctx context.Context, request *mdns.Msg, timeout time.Duration) (*mdns.Msg, string, error) {
	p.mu.RLock()
	endpoints := append([]upstreamEndpoint(nil), p.endpoints...)
	p.mu.RUnlock()
	if len(endpoints) == 0 {
		return nil, "", p.Validate()
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	start := int(atomic.AddUint32(&p.index, 1)-1) % len(endpoints)
	errs := make([]error, 0, len(endpoints))
	for offset := 0; offset < len(endpoints); offset++ {
		endpoint := endpoints[(start+offset)%len(endpoints)]
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		response, err := p.exchange(attemptCtx, endpoint, request, timeout)
		cancel()
		if err == nil && response != nil {
			return response, endpoint.raw, nil
		}
		if err == nil {
			err = errors.New("empty DNS response")
		}
		errs = append(errs, fmt.Errorf("%s: %w", endpoint.raw, err))
	}
	return nil, "", fmt.Errorf("all DNS upstreams failed: %w", errors.Join(errs...))
}

func (p *UpstreamPool) exchange(ctx context.Context, endpoint upstreamEndpoint, request *mdns.Msg, timeout time.Duration) (*mdns.Msg, error) {
	switch endpoint.protocol {
	case protocolDoH:
		return p.exchangeDoH(ctx, endpoint, request)
	case protocolDoT:
		client := &mdns.Client{
			Net: "tcp-tls", Timeout: timeout,
			TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: endpoint.serverName},
		}
		response, _, err := client.ExchangeContext(ctx, request, endpoint.address)
		return response, err
	case protocolTCP, protocolUDP:
		client := &mdns.Client{Net: string(endpoint.protocol), Timeout: timeout}
		response, _, err := client.ExchangeContext(ctx, request, endpoint.address)
		return response, err
	default:
		return nil, fmt.Errorf("unsupported upstream protocol %q", endpoint.protocol)
	}
}

func (p *UpstreamPool) exchangeDoH(ctx context.Context, endpoint upstreamEndpoint, message *mdns.Msg) (*mdns.Msg, error) {
	wire, err := message.Pack()
	if err != nil {
		return nil, fmt.Errorf("encode DNS request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.url.String(), bytes.NewReader(wire))
	if err != nil {
		return nil, fmt.Errorf("create DoH request: %w", err)
	}
	req.Header.Set("Accept", "application/dns-message")
	req.Header.Set("Content-Type", "application/dns-message")

	response, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send DoH request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("DoH server returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxDNSMessageSize+1))
	if err != nil {
		return nil, fmt.Errorf("read DoH response: %w", err)
	}
	if len(body) > maxDNSMessageSize {
		return nil, errors.New("DoH response exceeds the DNS message size limit")
	}
	result := new(mdns.Msg)
	if err := result.Unpack(body); err != nil {
		return nil, fmt.Errorf("decode DoH response: %w", err)
	}
	if !result.Response {
		return nil, errors.New("DoH server returned a DNS query instead of a response")
	}
	if result.Id != message.Id {
		return nil, errors.New("DoH response ID does not match the request")
	}
	return result, nil
}

// Get returns the next upstream's display value. DNS requests should use Exchange.
func (p *UpstreamPool) Get() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.endpoints) == 0 {
		return ""
	}
	idx := atomic.AddUint32(&p.index, 1) - 1
	return p.endpoints[idx%uint32(len(p.endpoints))].raw
}

func (p *UpstreamPool) Add(server string) {
	endpoint, err := parseUpstreamEndpoint(server)
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.parseErrs = append(p.parseErrs, err)
		return
	}
	if p.secureOnly && !endpoint.encrypted() {
		return
	}
	p.endpoints = append(p.endpoints, endpoint)
}

func (p *UpstreamPool) Remove(server string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, endpoint := range p.endpoints {
		if endpoint.raw == server {
			p.endpoints = append(p.endpoints[:i], p.endpoints[i+1:]...)
			break
		}
	}
}

func (p *UpstreamPool) List() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	servers := make([]string, len(p.endpoints))
	for i, endpoint := range p.endpoints {
		servers[i] = endpoint.raw
	}
	return servers
}
