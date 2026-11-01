//go:build windows

package localdomain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func hostsPath() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "drivers", "etc", "hosts")
}

func Ready(domain string) bool {
	data, err := os.ReadFile(hostsPath())
	return err == nil && containsLocalEntry(data, domain)
}

func Apply(domain string) error {
	path := hostsPath()
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	updated, err := rewriteHosts(original, domain)
	if err != nil {
		return err
	}
	if string(updated) == string(original) {
		return nil
	}
	backupDir := filepath.Join("data", "hosts-backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return err
	}
	backup := filepath.Join(backupDir, fmt.Sprintf("hosts-%d.bak", time.Now().UnixNano()))
	if err := os.WriteFile(backup, original, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(path, updated, 0644); err != nil {
		_ = os.WriteFile(path, original, 0644)
		return err
	}
	if domain != "" && !Ready(domain) {
		_ = os.WriteFile(path, original, 0644)
		return fmt.Errorf("Windows did not accept the local hosts entry")
	}
	_ = exec.Command("ipconfig", "/flushdns").Run()
	return nil
}
