package vault

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestUSBRecoveryCreatesSeparateCodeAndResetsPassword(t *testing.T) {
	m, err := New(t.TempDir(), time.Hour, 5, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Setup("old-strong-vault-password"); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	old := usbRootAllowed
	usbRootAllowed = func(path string) bool { return path == root }
	defer func() { usbRootAllowed = old }()
	path, err := m.CreateUSBRecovery(root)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	code := strings.TrimSpace(strings.Split(strings.Split(string(data), "Code: ")[1], "\n")[0])
	if code == "" {
		t.Fatal("no code on USB")
	}
	m.Lock()
	if err := m.RecoverWithUSBCode("wrong", "new-strong-vault-password"); err == nil {
		t.Fatal("wrong USB code accepted")
	}
	if err := m.RecoverWithUSBCode(code, "new-strong-vault-password"); err != nil {
		t.Fatal(err)
	}
	m.Lock()
	if err := m.Unlock("old-strong-vault-password"); err == nil {
		t.Fatal("old password accepted")
	}
	if err := m.Unlock("new-strong-vault-password"); err != nil {
		t.Fatal(err)
	}
}
