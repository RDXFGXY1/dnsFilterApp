package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseConfigTemplateIsLoadableAndSanitized(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "config.example.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load release config template: %v", err)
	}

	if cfg.Security.RecoveryKeyHash != "" {
		t.Fatal("release config template must not contain a recovery key hash")
	}
	if cfg.Security.JWTSecret != "" {
		t.Fatal("release config template must not contain a shared JWT secret")
	}
	if cfg.Security.HTTPSEnabled {
		t.Fatal("release config template must not claim HTTPS is enabled")
	}
	if cfg.Server.APIHost != "127.0.0.1" {
		t.Fatalf("dashboard must default to loopback, got %q", cfg.Server.APIHost)
	}
	if cfg.Security.AdminPasswordHash == "" {
		t.Fatal("release config template must retain its setup sentinel")
	}
	if cfg.Security.IsSetupComplete() {
		t.Fatal("release config template must require first-run setup")
	}
}

func TestExistingCredentialsRemainConfigured(t *testing.T) {
	for _, security := range []SecurityConfig{
		{SetupComplete: true},
		{RecoveryKeyHash: "existing-recovery-hash"},
		{AdminPasswordHash: "custom-password-hash"},
	} {
		if !security.IsSetupComplete() {
			t.Fatalf("existing installation was treated as unconfigured: %+v", security)
		}
	}
}

func TestSaveDefaultUsesEachConfigsOwnPath(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.yaml")
	secondPath := filepath.Join(dir, "second.yaml")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte("security:\n  admin_username: initial\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	first, err := Load(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Load(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	first.Security.AdminUsername = "first-owner"
	second.Security.AdminUsername = "second-owner"
	if err := first.SaveDefault(); err != nil {
		t.Fatal(err)
	}
	if err := second.SaveDefault(); err != nil {
		t.Fatal(err)
	}
	firstData, _ := os.ReadFile(firstPath)
	secondData, _ := os.ReadFile(secondPath)
	if !strings.Contains(string(firstData), "first-owner") || strings.Contains(string(firstData), "second-owner") {
		t.Fatalf("first config was written to the wrong path: %s", firstData)
	}
	if !strings.Contains(string(secondData), "second-owner") || strings.Contains(string(secondData), "first-owner") {
		t.Fatalf("second config was written to the wrong path: %s", secondData)
	}
}

func TestReplaceWithRetainsInstallationPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("security:\n  admin_username: initial\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	next := Config{Security: SecurityConfig{AdminUsername: "restored-owner"}}
	if err := cfg.ReplaceWith(next); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveDefault(); err != nil {
		t.Fatalf("replacement lost its installation path: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "restored-owner") {
		t.Fatalf("replacement was not persisted: %s", data)
	}
}

func TestSystemDNSPreferenceDistinguishesLegacyAndExplicitOptOut(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "legacy.yaml")
	explicitPath := filepath.Join(dir, "explicit.yaml")
	if err := os.WriteFile(legacyPath, []byte("server:\n  api_host: 127.0.0.1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(explicitPath, []byte("server:\n  manage_system_dns: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	legacy, err := Load(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := Load(explicitPath)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.HasSystemDNSPreference() {
		t.Fatal("legacy config unexpectedly has an explicit routing preference")
	}
	if !explicit.HasSystemDNSPreference() || explicit.Server.ManageSystemDNS {
		t.Fatal("explicit false routing preference was not preserved")
	}
}
