package dns

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
)

func TestParseUpstreamEndpoint(t *testing.T) {
	tests := []struct {
		input     string
		protocol  upstreamProtocol
		address   string
		encrypted bool
	}{
		{"8.8.8.8:53", protocolUDP, "8.8.8.8:53", false},
		{"tcp://9.9.9.9", protocolTCP, "9.9.9.9:53", false},
		{"tls://1.1.1.1", protocolDoT, "1.1.1.1:853", true},
		{"https://1.1.1.1/dns-query", protocolDoH, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseUpstreamEndpoint(tt.input)
			if err != nil {
				t.Fatalf("parseUpstreamEndpoint() error = %v", err)
			}
			if got.protocol != tt.protocol || got.address != tt.address || got.encrypted() != tt.encrypted {
				t.Fatalf("endpoint = %#v, want protocol=%q address=%q encrypted=%v", got, tt.protocol, tt.address, tt.encrypted)
			}
		})
	}
}

func TestSecureOnlyRejectsPlaintextOnlyConfiguration(t *testing.T) {
	pool := NewUpstreamPool([]string{"8.8.8.8:53", "tcp://1.1.1.1:53"}, true)
	if err := pool.Validate(); err == nil {
		t.Fatal("Validate() succeeded with only plaintext upstreams in secure-only mode")
	}
	if got := pool.List(); len(got) != 0 {
		t.Fatalf("List() = %v, want no plaintext endpoints", got)
	}
}

func TestDoHExchange(t *testing.T) {
	server := newTestDoHServer(t, true)
	defer server.Close()

	pool := NewUpstreamPool([]string{server.URL + "/dns-query"}, true)
	pool.httpClient = server.Client()
	query := new(mdns.Msg)
	query.SetQuestion("example.com.", mdns.TypeA)

	response, used, err := pool.Exchange(t.Context(), query, time.Second)
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if used != server.URL+"/dns-query" {
		t.Fatalf("used endpoint = %q", used)
	}
	if len(response.Answer) != 1 || response.Answer[0].(*mdns.A).A.String() != "192.0.2.1" {
		t.Fatalf("unexpected response: %v", response)
	}
}

func TestDoHFailsOverToNextEndpoint(t *testing.T) {
	server := newTestDoHServer(t, false)
	defer server.Close()

	pool := NewUpstreamPool([]string{"https://127.0.0.1:1/dns-query", server.URL + "/dns-query"}, true)
	pool.httpClient = server.Client()
	query := new(mdns.Msg)
	query.SetQuestion("example.com.", mdns.TypeA)

	_, used, err := pool.Exchange(t.Context(), query, 250*time.Millisecond)
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if used != server.URL+"/dns-query" {
		t.Fatalf("used endpoint = %q, want fallback endpoint", used)
	}
}

func newTestDoHServer(t *testing.T, withAnswer bool) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/dns-message" {
			t.Errorf("Content-Type = %q", got)
		}
		wire, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
			return
		}
		query := new(mdns.Msg)
		if err := query.Unpack(wire); err != nil {
			t.Errorf("unpack request: %v", err)
			return
		}
		response := new(mdns.Msg)
		response.SetReply(query)
		if withAnswer {
			response.Answer = []mdns.RR{&mdns.A{
				Hdr: mdns.RR_Header{Name: query.Question[0].Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 60},
				A:   net.ParseIP("192.0.2.1"),
			}}
		}
		packed, err := response.Pack()
		if err != nil {
			t.Errorf("pack response: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(packed)
	}))
}
