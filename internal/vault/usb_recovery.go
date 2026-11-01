package vault

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var usbRootAllowed = validRemovableRoot

// CreateUSBRecovery writes a new, independent recovery code to the selected
// removable drive. Existing offline and USB codes remain valid.
func (m *Manager) CreateUSBRecovery(root string) (string, error) {
	if !usbRootAllowed(root) {
		return "", errors.New("select a connected removable USB drive")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expireLocked()
	if len(m.master) != 32 {
		return "", ErrLocked
	}
	e, err := m.readEnvelope()
	if err != nil {
		return "", err
	}
	if len(e.USB) >= 10 {
		return "", errors.New("USB recovery limit reached")
	}
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	code := groupCode(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret))
	idRaw := make([]byte, 8)
	if _, err := rand.Read(idRaw); err != nil {
		return "", err
	}
	id := hex.EncodeToString(idRaw)
	key, err := wrap(m.master, normalizeRecovery(code))
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "DNSFilter-Recovery")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "vault-recovery-"+id+".txt")
	contents := fmt.Sprintf("DNS Filter Vault USB recovery\nCreated: %s\nCode: %s\n\nKeep this USB private. Anyone with this code and access to your dashboard can reset the Vault password.\n", time.Now().Format(time.RFC3339), code)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	if _, err = file.WriteString(contents); err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}
	if err = file.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	e.USB = append(e.USB, usbWrapper{ID: id, Key: key})
	if err := writeJSONAtomic(m.envelopePath, e, 0600); err != nil {
		os.Remove(path)
		return "", err
	}
	_ = m.auditLocked("vault.usb.create", "USB recovery code created")
	return path, nil
}

func (m *Manager) RecoverWithUSBCode(code, newPassword string) error {
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
	var master []byte
	for _, item := range e.USB {
		if master, err = unwrap(item.Key, normalizeRecovery(code)); err == nil {
			break
		}
	}
	if len(master) != 32 {
		m.failures++
		if m.failures >= m.maxFailures {
			m.lockedOutUntil = time.Now().Add(m.lockoutDuration)
			m.failures = 0
			return ErrLockedOut
		}
		return errors.New("invalid USB recovery code")
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
	return m.auditLocked("vault.usb.recover", "vault password reset with USB code")
}
