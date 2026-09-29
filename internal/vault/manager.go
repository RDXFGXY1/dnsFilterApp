package vault

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

var (
	ErrNotConfigured = errors.New("vault is not configured")
	ErrLocked        = errors.New("vault is locked")
	ErrLockedOut     = errors.New("vault is temporarily locked after too many failed attempts")
)

type wrapper struct {
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type envelope struct {
	Version  int       `json:"version"`
	Created  time.Time `json:"created_at"`
	Password wrapper   `json:"password"`
	Recovery wrapper   `json:"recovery"`
}

type Status struct {
	Configured     bool       `json:"configured"`
	Unlocked       bool       `json:"unlocked"`
	UnlockedUntil  *time.Time `json:"unlocked_until,omitempty"`
	FailedAttempts int        `json:"failed_attempts"`
	LockedOutUntil *time.Time `json:"locked_out_until,omitempty"`
	AuditEntries   int        `json:"audit_entries"`
}

type auditRecord struct {
	Time     time.Time `json:"time"`
	Action   string    `json:"action"`
	Detail   string    `json:"detail,omitempty"`
	Previous string    `json:"previous_hash,omitempty"`
	Hash     string    `json:"hash"`
}

type Manager struct {
	mu              sync.Mutex
	directory       string
	envelopePath    string
	auditPath       string
	auditHeadPath   string
	master          []byte
	unlockedUntil   time.Time
	unlockDuration  time.Duration
	maxFailures     int
	lockoutDuration time.Duration
	failures        int
	lockedOutUntil  time.Time
}

func New(directory string, unlockDuration time.Duration, maxFailures int, lockoutDuration time.Duration) (*Manager, error) {
	if unlockDuration <= 0 {
		unlockDuration = 15 * time.Minute
	}
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if lockoutDuration <= 0 {
		lockoutDuration = 5 * time.Minute
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	return &Manager{directory: directory, envelopePath: filepath.Join(directory, "vault.json"), auditPath: filepath.Join(directory, "audit.jsonl"), auditHeadPath: filepath.Join(directory, "audit.head"), unlockDuration: unlockDuration, maxFailures: maxFailures, lockoutDuration: lockoutDuration}, nil
}

func (m *Manager) Configured() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := os.Stat(m.envelopePath)
	return err == nil
}

// SetUnlockDuration applies the first-run auto-lock choice immediately without
// requiring a service restart.
func (m *Manager) SetUnlockDuration(duration time.Duration) {
	if duration <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unlockDuration = duration
	if len(m.master) == 32 {
		m.unlockedUntil = time.Now().Add(duration)
	}
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expireLocked()
	s := Status{Configured: fileExists(m.envelopePath), Unlocked: len(m.master) == 32, FailedAttempts: m.failures, AuditEntries: countLines(m.auditPath)}
	if s.Unlocked {
		t := m.unlockedUntil
		s.UnlockedUntil = &t
	}
	if time.Now().Before(m.lockedOutUntil) {
		t := m.lockedOutUntil
		s.LockedOutUntil = &t
	}
	return s
}

func (m *Manager) Setup(password string) (string, error) {
	if len(password) < 12 {
		return "", errors.New("vault password must contain at least 12 characters")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if fileExists(m.envelopePath) {
		return "", errors.New("vault is already configured")
	}
	master, err := randomBytes(32)
	if err != nil {
		return "", err
	}
	defer zero(master)
	recoveryRaw, err := randomBytes(20)
	if err != nil {
		return "", err
	}
	recovery := groupCode(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(recoveryRaw))
	pw, err := wrap(master, password)
	if err != nil {
		return "", err
	}
	rw, err := wrap(master, normalizeRecovery(recovery))
	if err != nil {
		return "", err
	}
	e := envelope{Version: 1, Created: time.Now().UTC(), Password: pw, Recovery: rw}
	if err := writeJSONAtomic(m.envelopePath, e, 0600); err != nil {
		return "", err
	}
	m.setMaster(master)
	_ = m.auditLocked("vault.setup", "secure configuration vault enabled")
	return recovery, nil
}

func (m *Manager) Unlock(password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !fileExists(m.envelopePath) {
		return ErrNotConfigured
	}
	if time.Now().Before(m.lockedOutUntil) {
		return ErrLockedOut
	}
	e, err := m.readEnvelope()
	if err != nil {
		return err
	}
	master, err := unwrap(e.Password, password)
	if err != nil {
		m.failures++
		if m.failures >= m.maxFailures {
			m.lockedOutUntil = time.Now().Add(m.lockoutDuration)
			m.failures = 0
			return ErrLockedOut
		}
		return errors.New("incorrect vault password")
	}
	defer zero(master)
	m.failures = 0
	m.lockedOutUntil = time.Time{}
	m.setMaster(master)
	return m.auditLocked("vault.unlock", "vault unlocked with password")
}

func (m *Manager) Recover(code, newPassword string) error {
	if len(newPassword) < 12 {
		return errors.New("new vault password must contain at least 12 characters")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if time.Now().Before(m.lockedOutUntil) {
		return ErrLockedOut
	}
	e, err := m.readEnvelope()
	if err != nil {
		return err
	}
	master, err := unwrap(e.Recovery, normalizeRecovery(code))
	if err != nil {
		m.failures++
		if m.failures >= m.maxFailures {
			m.lockedOutUntil = time.Now().Add(m.lockoutDuration)
			m.failures = 0
			return ErrLockedOut
		}
		return errors.New("invalid recovery code")
	}
	defer zero(master)
	e.Password, err = wrap(master, newPassword)
	if err != nil {
		return err
	}
	if err := writeJSONAtomic(m.envelopePath, e, 0600); err != nil {
		return err
	}
	m.failures = 0
	m.lockedOutUntil = time.Time{}
	m.setMaster(master)
	return m.auditLocked("vault.recover", "vault password reset with recovery code")
}

func (m *Manager) Lock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.master) == 32 {
		_ = m.auditLocked("vault.lock", "vault locked")
	}
	zero(m.master)
	m.master = nil
	m.unlockedUntil = time.Time{}
}

func (m *Manager) RequireUnlocked() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expireLocked()
	if !fileExists(m.envelopePath) {
		return nil
	}
	if len(m.master) != 32 {
		return ErrLocked
	}
	return nil
}

func (m *Manager) Encrypt(plain []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expireLocked()
	if len(m.master) != 32 {
		return nil, ErrLocked
	}
	nonce, ciphertext, err := seal(m.master, plain)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Version    int       `json:"version"`
		Created    time.Time `json:"created_at"`
		Nonce      string    `json:"nonce"`
		Ciphertext string    `json:"ciphertext"`
	}{1, time.Now().UTC(), enc(nonce), enc(ciphertext)})
}

func (m *Manager) Decrypt(data []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expireLocked()
	if len(m.master) != 32 {
		return nil, ErrLocked
	}
	var b struct {
		Version    int    `json:"version"`
		Nonce      string `json:"nonce"`
		Ciphertext string `json:"ciphertext"`
	}
	if err := json.Unmarshal(data, &b); err != nil || b.Version != 1 {
		return nil, errors.New("invalid encrypted backup")
	}
	return open(m.master, b.Nonce, b.Ciphertext)
}

func (m *Manager) Audit(action, detail string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expireLocked()
	if len(m.master) != 32 {
		return ErrLocked
	}
	return m.auditLocked(action, detail)
}

func (m *Manager) AuditRecords(limit int) ([]map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expireLocked()
	if len(m.master) != 32 {
		return nil, ErrLocked
	}
	f, err := os.Open(m.auditPath)
	if errors.Is(err, os.ErrNotExist) {
		if headErr := m.verifyAuditHead(""); headErr != nil {
			return nil, headErr
		}
		return []map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []map[string]any
	var previous string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var r auditRecord
		if json.Unmarshal(scanner.Bytes(), &r) != nil {
			return nil, errors.New("audit log is damaged")
		}
		expected := m.auditHash(r.Time, r.Action, r.Detail, r.Previous)
		if r.Previous != previous || !hmac.Equal([]byte(expected), []byte(r.Hash)) {
			return nil, errors.New("audit log integrity check failed")
		}
		previous = r.Hash
		out = append(out, map[string]any{"time": r.Time, "action": r.Action, "detail": r.Detail})
	}
	if err := m.verifyAuditHead(previous); err != nil {
		return nil, err
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, scanner.Err()
}

func (m *Manager) setMaster(master []byte) {
	zero(m.master)
	m.master = append([]byte(nil), master...)
	m.unlockedUntil = time.Now().Add(m.unlockDuration)
}
func (m *Manager) expireLocked() {
	if len(m.master) > 0 && time.Now().After(m.unlockedUntil) {
		zero(m.master)
		m.master = nil
		m.unlockedUntil = time.Time{}
	}
}
func (m *Manager) readEnvelope() (envelope, error) {
	var e envelope
	data, err := os.ReadFile(m.envelopePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return e, ErrNotConfigured
		}
		return e, err
	}
	if json.Unmarshal(data, &e) != nil || e.Version != 1 {
		return e, errors.New("vault metadata is invalid")
	}
	return e, nil
}
func (m *Manager) auditLocked(action, detail string) error {
	previous := lastAuditHash(m.auditPath)
	now := time.Now().UTC()
	r := auditRecord{Time: now, Action: action, Detail: detail, Previous: previous}
	r.Hash = m.auditHash(now, action, detail, previous)
	data, _ := json.Marshal(r)
	f, err := os.OpenFile(m.auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(append(data, '\n')); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	return m.writeAuditHead(r.Hash)
}

func (m *Manager) writeAuditHead(hash string) error {
	h := hmac.New(sha256.New, m.master)
	h.Write([]byte("audit-head\n" + hash))
	return writeJSONAtomic(m.auditHeadPath, map[string]string{"hash": hash, "mac": hex.EncodeToString(h.Sum(nil))}, 0600)
}

func (m *Manager) verifyAuditHead(last string) error {
	data, err := os.ReadFile(m.auditHeadPath)
	if err != nil {
		return errors.New("audit log anchor is missing")
	}
	var head map[string]string
	if json.Unmarshal(data, &head) != nil || head["hash"] != last {
		return errors.New("audit log was truncated or deleted")
	}
	h := hmac.New(sha256.New, m.master)
	h.Write([]byte("audit-head\n" + head["hash"]))
	expected := hex.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(head["mac"])) {
		return errors.New("audit log anchor authentication failed")
	}
	return nil
}
func (m *Manager) auditHash(t time.Time, action, detail, previous string) string {
	h := hmac.New(sha256.New, m.master)
	fmt.Fprintf(h, "%s\n%s\n%s\n%s", t.Format(time.RFC3339Nano), action, detail, previous)
	return hex.EncodeToString(h.Sum(nil))
}

func wrap(master []byte, secret string) (wrapper, error) {
	salt, e := randomBytes(16)
	if e != nil {
		return wrapper{}, e
	}
	key := derive(secret, salt)
	defer zero(key)
	nonce, ct, e := seal(key, master)
	return wrapper{enc(salt), enc(nonce), enc(ct)}, e
}
func unwrap(w wrapper, secret string) ([]byte, error) {
	salt, e := dec(w.Salt)
	if e != nil {
		return nil, e
	}
	key := derive(secret, salt)
	defer zero(key)
	return open(key, w.Nonce, w.Ciphertext)
}
func derive(secret string, salt []byte) []byte {
	return argon2.IDKey([]byte(secret), salt, 3, 32*1024, 2, 32)
}
func seal(key, plain []byte) ([]byte, []byte, error) {
	b, e := aes.NewCipher(key)
	if e != nil {
		return nil, nil, e
	}
	g, e := cipher.NewGCM(b)
	if e != nil {
		return nil, nil, e
	}
	n, e := randomBytes(g.NonceSize())
	if e != nil {
		return nil, nil, e
	}
	return n, g.Seal(nil, n, plain, nil), nil
}
func open(key []byte, ns, cs string) ([]byte, error) {
	n, e := dec(ns)
	if e != nil {
		return nil, e
	}
	ct, e := dec(cs)
	if e != nil {
		return nil, e
	}
	b, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	g, e := cipher.NewGCM(b)
	if e != nil {
		return nil, e
	}
	return g.Open(nil, n, ct, nil)
}
func randomBytes(n int) ([]byte, error) { b := make([]byte, n); _, e := rand.Read(b); return b, e }
func enc(b []byte) string               { return base64.RawStdEncoding.EncodeToString(b) }
func dec(s string) ([]byte, error)      { return base64.RawStdEncoding.DecodeString(s) }
func normalizeRecovery(s string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), "-", ""))
}
func groupCode(s string) string {
	var p []string
	for len(s) > 5 {
		p = append(p, s[:5])
		s = s[5:]
	}
	if s != "" {
		p = append(p, s)
	}
	return strings.Join(p, "-")
}
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
func fileExists(p string) bool { _, e := os.Stat(p); return e == nil }
func countLines(p string) int {
	f, e := os.Open(p)
	if e != nil {
		return 0
	}
	defer f.Close()
	n := 0
	s := bufio.NewScanner(f)
	for s.Scan() {
		n++
	}
	return n
}
func lastAuditHash(p string) string {
	f, e := os.Open(p)
	if e != nil {
		return ""
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	last := ""
	for s.Scan() {
		var r auditRecord
		if json.Unmarshal(s.Bytes(), &r) == nil {
			last = r.Hash
		}
	}
	return last
}
func writeJSONAtomic(path string, v any, mode os.FileMode) error {
	data, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	tmp := path + ".tmp"
	if e = os.WriteFile(tmp, data, mode); e != nil {
		return e
	}
	if e = os.Rename(tmp, path); e == nil {
		return nil
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}
