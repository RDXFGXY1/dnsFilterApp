package securitypacks

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	mdns "github.com/miekg/dns"
)

const (
	FormatVersion = 1
	MaxPackBytes  = 10 << 20
	MaxDomains    = 500000
)

var safeID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

type Manifest struct {
	FormatVersion int       `json:"format_version"`
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Version       int       `json:"version"`
	Category      string    `json:"category"`
	KeyID         string    `json:"key_id"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Domains       []string  `json:"domains"`
}

type Pack struct {
	Manifest  Manifest `json:"manifest"`
	Signature string   `json:"signature"`
}

type Info struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     int       `json:"version"`
	Category    string    `json:"category"`
	KeyID       string    `json:"key_id"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	DomainCount int       `json:"domain_count"`
}

type Match struct {
	PackID    string    `json:"pack_id"`
	PackName  string    `json:"pack_name"`
	Category  string    `json:"category"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Manager struct {
	directory string
	mu        sync.RWMutex
	keys      map[string]ed25519.PublicKey
	packs     map[string]Info
	domains   map[string]Match
}

func New(directory string, encodedKeys map[string]string) (*Manager, error) {
	if directory == "" {
		return nil, errors.New("security pack directory is required")
	}
	m := &Manager{directory: directory, keys: make(map[string]ed25519.PublicKey), packs: make(map[string]Info), domains: make(map[string]Match)}
	for id, encoded := range encodedKeys {
		if err := m.addTrustedKey(id, encoded); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("create security pack directory: %w", err)
	}
	if err := m.reload(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) AddTrustedKey(id, encoded string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.addTrustedKey(id, encoded); err != nil {
		return err
	}
	return m.reloadLocked()
}

func (m *Manager) addTrustedKey(id, encoded string) error {
	id = strings.TrimSpace(id)
	if !safeID.MatchString(id) {
		return fmt.Errorf("invalid security pack key id %q", id)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return fmt.Errorf("trusted key %q is not a valid base64 Ed25519 public key", id)
	}
	m.keys[id] = append(ed25519.PublicKey(nil), decoded...)
	return nil
}

func (m *Manager) TrustedKeyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.keys)
}

func (m *Manager) Verify(data []byte) (Pack, Info, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.verifyLocked(data, time.Now())
}

func (m *Manager) verifyLocked(data []byte, now time.Time) (Pack, Info, error) {
	if len(data) == 0 || len(data) > MaxPackBytes {
		return Pack{}, Info{}, fmt.Errorf("security pack must be between 1 byte and %d bytes", MaxPackBytes)
	}
	var pack Pack
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pack); err != nil {
		return Pack{}, Info{}, fmt.Errorf("invalid security pack JSON: %w", err)
	}
	manifest := &pack.Manifest
	if manifest.FormatVersion != FormatVersion || !safeID.MatchString(manifest.ID) || manifest.Name == "" || manifest.Version < 1 {
		return Pack{}, Info{}, errors.New("security pack manifest is incomplete or uses an unsupported format")
	}
	if len(manifest.Domains) == 0 || len(manifest.Domains) > MaxDomains {
		return Pack{}, Info{}, fmt.Errorf("security pack must contain 1–%d domains", MaxDomains)
	}
	if manifest.CreatedAt.IsZero() || manifest.ExpiresAt.IsZero() || !manifest.ExpiresAt.After(manifest.CreatedAt) {
		return Pack{}, Info{}, errors.New("security pack timestamps are invalid")
	}
	if !manifest.ExpiresAt.After(now) {
		return Pack{}, Info{}, errors.New("security pack has expired")
	}
	key, trusted := m.keys[manifest.KeyID]
	if !trusted {
		return Pack{}, Info{}, fmt.Errorf("security pack key %q is not trusted", manifest.KeyID)
	}
	signature, err := base64.StdEncoding.DecodeString(pack.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return Pack{}, Info{}, errors.New("security pack signature is malformed")
	}
	payload, err := json.Marshal(pack.Manifest)
	if err != nil || !ed25519.Verify(key, payload, signature) {
		return Pack{}, Info{}, errors.New("security pack signature verification failed")
	}

	seen := make(map[string]struct{}, len(manifest.Domains))
	for _, raw := range manifest.Domains {
		domain := normalizeDomain(raw)
		if _, valid := mdns.IsDomainName(domain); !valid {
			return Pack{}, Info{}, fmt.Errorf("security pack contains invalid domain %q", raw)
		}
		if raw != domain {
			return Pack{}, Info{}, fmt.Errorf("security pack domain %q is not normalized", raw)
		}
		if _, exists := seen[domain]; exists {
			return Pack{}, Info{}, fmt.Errorf("security pack contains duplicate domain %q", domain)
		}
		seen[domain] = struct{}{}
	}
	info := packInfo(pack)
	return pack, info, nil
}

func (m *Manager) Install(data []byte) (Info, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pack, info, err := m.verifyLocked(data, time.Now())
	if err != nil {
		return Info{}, err
	}
	if current, exists := m.packs[info.ID]; exists && info.Version < current.Version {
		return Info{}, fmt.Errorf("refusing security pack downgrade from version %d to %d", current.Version, info.Version)
	}
	canonical, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return Info{}, err
	}
	target := filepath.Join(m.directory, info.ID+".json")
	temp, err := os.CreateTemp(m.directory, ".pack-*.tmp")
	if err != nil {
		return Info{}, err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return Info{}, err
	}
	if _, err := temp.Write(canonical); err != nil {
		temp.Close()
		return Info{}, err
	}
	if err := temp.Close(); err != nil {
		return Info{}, err
	}
	if err := replaceFile(tempName, target); err != nil {
		return Info{}, err
	}
	if err := m.reloadLocked(); err != nil {
		return Info{}, err
	}
	return info, nil
}

func (m *Manager) Remove(id string) error {
	if !safeID.MatchString(id) {
		return errors.New("invalid security pack id")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	err := os.Remove(filepath.Join(m.directory, id+".json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return m.reloadLocked()
}

func (m *Manager) List() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := make([]Info, 0, len(m.packs))
	for _, info := range m.packs {
		items = append(items, info)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (m *Manager) Match(domain string) (Match, bool) {
	domain = normalizeDomain(domain)
	now := time.Now()
	m.mu.RLock()
	defer m.mu.RUnlock()
	if match, ok := m.domains[domain]; ok {
		if match.ExpiresAt.After(now) {
			return match, true
		}
		return Match{}, false
	}
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		if match, ok := m.domains[strings.Join(parts[i:], ".")]; ok {
			if match.ExpiresAt.After(now) {
				return match, true
			}
			return Match{}, false
		}
	}
	return Match{}, false
}

func (m *Manager) DomainCount() int {
	now := time.Now()
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, match := range m.domains {
		if match.ExpiresAt.After(now) {
			count++
		}
	}
	return count
}

func (m *Manager) reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reloadLocked()
}

func (m *Manager) reloadLocked() error {
	entries, err := os.ReadDir(m.directory)
	if err != nil {
		return err
	}
	packs := make(map[string]Info)
	domains := make(map[string]Match)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(m.directory, entry.Name()))
		if err != nil {
			continue
		}
		pack, info, err := m.verifyLocked(data, time.Now())
		if err != nil {
			continue
		}
		packs[info.ID] = info
		for _, domain := range pack.Manifest.Domains {
			domains[normalizeDomain(domain)] = Match{PackID: info.ID, PackName: info.Name, Category: info.Category, ExpiresAt: info.ExpiresAt}
		}
	}
	m.packs = packs
	m.domains = domains
	return nil
}

func packInfo(pack Pack) Info {
	m := pack.Manifest
	return Info{ID: m.ID, Name: m.Name, Version: m.Version, Category: m.Category, KeyID: m.KeyID, CreatedAt: m.CreatedAt, ExpiresAt: m.ExpiresAt, DomainCount: len(m.Domains)}
}

func normalizeDomain(domain string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(domain)), ".")
}

func replaceFile(source, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	backup := target + ".previous"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(source, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}
	_ = os.Remove(backup)
	return nil
}
