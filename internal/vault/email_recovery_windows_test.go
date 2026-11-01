//go:build windows

package vault

import (
	"strings"
	"testing"
	"time"
)

func TestEmailCodeResetsVaultPassword(t *testing.T) {
	m, err := New(t.TempDir(), time.Hour, 5, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Setup("old-strong-vault-password"); err != nil {
		t.Fatal(err)
	}
	original := emailSender
	defer func() { emailSender = original }()
	var code string
	emailSender = func(_ EmailRecoverySettings, subject, body string) error {
		if strings.Contains(subject, "recovery code") {
			code = strings.TrimSpace(strings.Split(strings.Split(body, "code is: ")[1], "\r\n")[0])
		}
		return nil
	}
	settings := EmailRecoverySettings{Host: "smtp.example.com", Port: 2525, TLSMode: "starttls", From: "sender@example.com", To: "owner@example.com", Username: "sender@example.com", Password: "app-password"}
	if err := m.EnrollEmailRecovery(settings); err != nil {
		t.Fatal(err)
	}
	m.Lock()
	if err := m.RequestEmailRecovery(); err != nil {
		t.Fatal(err)
	}
	if len(code) != 8 {
		t.Fatalf("bad code: %q", code)
	}
	if err := m.RecoverWithEmailCode("00000000", "new-strong-vault-password"); err == nil {
		t.Fatal("wrong code accepted")
	}
	if err := m.RecoverWithEmailCode(code, "new-strong-vault-password"); err != nil {
		t.Fatal(err)
	}
	m.Lock()
	if err := m.Unlock("old-strong-vault-password"); err == nil {
		t.Fatal("old password accepted")
	}
	if err := m.Unlock("new-strong-vault-password"); err != nil {
		t.Fatal(err)
	}
	if err := m.RecoverWithEmailCode(code, "another-strong-password"); err == nil {
		t.Fatal("code reused")
	}
}
