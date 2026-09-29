package vault

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

func TestVaultLifecycleRecoveryAndEncryption(t *testing.T) {
	m, err := New(t.TempDir(), time.Minute, 3, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	code, err := m.Setup("a-very-strong-password")
	if err != nil || code == "" {
		t.Fatalf("setup: code=%q err=%v", code, err)
	}
	if !m.Status().Unlocked {
		t.Fatal("vault should be unlocked after setup")
	}

	plain := []byte(`{"safe":true}`)
	sealed, err := m.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := m.Decrypt(sealed)
	if err != nil || !bytes.Equal(opened, plain) {
		t.Fatalf("decrypt: %q %v", opened, err)
	}
	var backup map[string]any
	if err := json.Unmarshal(sealed, &backup); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(backup["ciphertext"].(string))
	if err != nil || len(ciphertext) == 0 {
		t.Fatalf("decode ciphertext: %v", err)
	}
	ciphertext[0] ^= 1
	backup["ciphertext"] = base64.RawStdEncoding.EncodeToString(ciphertext)
	tampered, err := json.Marshal(backup)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Decrypt(tampered); err == nil {
		t.Fatal("tampered backup was accepted")
	}

	m.Lock()
	if !errors.Is(m.RequireUnlocked(), ErrLocked) {
		t.Fatal("expected locked vault")
	}
	if err := m.Unlock("wrong password"); err == nil {
		t.Fatal("wrong password accepted")
	}
	if err := m.Unlock("a-very-strong-password"); err != nil {
		t.Fatal(err)
	}
	m.Lock()
	if err := m.Recover(code, "a-new-strong-password"); err != nil {
		t.Fatal(err)
	}
	m.Lock()
	if err := m.Unlock("a-new-strong-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AuditRecords(100); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.auditPath, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.AuditRecords(100); err == nil {
		t.Fatal("deleted audit history was accepted")
	}
}

func TestVaultAttemptLockout(t *testing.T) {
	m, _ := New(t.TempDir(), time.Minute, 2, 25*time.Millisecond)
	if _, err := m.Setup("another-strong-password"); err != nil {
		t.Fatal(err)
	}
	m.Lock()
	_ = m.Unlock("bad-one")
	if err := m.Unlock("bad-two"); !errors.Is(err, ErrLockedOut) {
		t.Fatalf("expected lockout, got %v", err)
	}
	if err := m.Unlock("another-strong-password"); !errors.Is(err, ErrLockedOut) {
		t.Fatal("lockout was bypassed")
	}
	time.Sleep(30 * time.Millisecond)
	if err := m.Unlock("another-strong-password"); err != nil {
		t.Fatal(err)
	}
}

func TestVaultRecoveryAttemptLockout(t *testing.T) {
	m, _ := New(t.TempDir(), time.Minute, 2, 25*time.Millisecond)
	code, err := m.Setup("another-strong-password")
	if err != nil {
		t.Fatal(err)
	}
	m.Lock()
	if err := m.Recover("bad-code-one", "replacement-password"); err == nil {
		t.Fatal("wrong recovery code was accepted")
	}
	if err := m.Recover("bad-code-two", "replacement-password"); !errors.Is(err, ErrLockedOut) {
		t.Fatalf("expected recovery lockout, got %v", err)
	}
	if err := m.Recover(code, "replacement-password"); !errors.Is(err, ErrLockedOut) {
		t.Fatal("valid recovery code bypassed active lockout")
	}
	time.Sleep(30 * time.Millisecond)
	if err := m.Recover(code, "replacement-password"); err != nil {
		t.Fatal(err)
	}
}
