//go:build windows

package autostart

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskRegistrationScriptIsBootResilient(t *testing.T) {
	script := taskRegistrationScript(`C:\Program Files\DNS Filter\dns-filter.exe`)
	for _, expected := range []string{
		"New-ScheduledTaskTrigger -AtStartup",
		"New-ScheduledTaskTrigger -AtLogOn",
		"RestartCount 999",
		"RestartInterval",
		"MultipleInstances IgnoreNew",
		"DisallowStartIfOnBatteries = $false",
		"-User 'SYSTEM'",
		"DNSFilterRecovery",
		"-Argument 'recover-dns'",
		`'C:\Program Files\DNS Filter\dns-filter.exe'`,
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("registration script is missing %q", expected)
		}
	}
}

func TestServiceWorkingDirectoryUsesRepositoryForNestedDevBuild(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "config.yaml"), []byte("server: {}"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := serviceWorkingDirectory(filepath.Join(root, "build", "dev", "dns-filter-dev.exe")); got != root {
		t.Fatalf("nested working folder = %q, want %q", got, root)
	}
}

func TestLoginAndManualStartupDoNotRegisterMainBootTrigger(t *testing.T) {
	exe := `C:\Apps\DNS Filter\dns-filter.exe`
	login := configurationScript(exe, Options{Mode: ModeLogin})
	if strings.Contains(login, "-AtStartup") || !strings.Contains(login, "-AtLogOn") {
		t.Fatal("login mode incorrectly starts before sign-in")
	}
	manual := configurationScript(exe, Options{Mode: ModeManual, RecoverDNS: true})
	if strings.Contains(manual, "Register-ScheduledTask -TaskName 'DNSFilter' ") || !strings.Contains(manual, "Register-ScheduledTask -TaskName 'DNSFilterRecovery'") {
		t.Fatal("manual mode should register only requested recovery")
	}
	if !strings.Contains(configurationScript(exe, Options{Mode: ModeManual}), "Unregister-ScheduledTask -TaskName 'DNSFilterRecovery'") {
		t.Fatal("disable does not remove recovery")
	}
}

func TestPowerShellQuoteEscapesApostrophes(t *testing.T) {
	if got := psQuote(`C:\Owner's Apps\dns-filter.exe`); got != `'C:\Owner''s Apps\dns-filter.exe'` {
		t.Fatalf("unexpected quoted path: %s", got)
	}
}

func TestServiceWorkingDirectoryUsesParentForDevelopmentBuild(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "config.yaml"), []byte("server: {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "build", "dns-filter-gui-server.exe")
	if got := serviceWorkingDirectory(exe); got != root {
		t.Fatalf("working directory = %q, want %q", got, root)
	}
}

// Encoded PowerShell progress/diagnostic records must never enter the JSON stream.
func TestPowerShellKeepsDiagnosticStreamOutOfJSON(t *testing.T) {
	out, err := powershell(`[Console]::Error.WriteLine('#< CLIXML'); '{"enabled":true,"mode":"boot"}'`)
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Enabled bool   `json:"enabled"`
		Mode    string `json:"mode"`
	}
	if err = json.Unmarshal(out, &state); err != nil {
		t.Fatalf("stderr polluted JSON output %q: %v", out, err)
	}
	if !state.Enabled || state.Mode != "boot" {
		t.Fatalf("wrong status: %+v", state)
	}
}
func TestPowerShellPreservesFailureDiagnostics(t *testing.T) {
	_, err := powershell(`[Console]::Error.WriteLine('startup permission denied'); exit 7`)
	if err == nil || !strings.Contains(err.Error(), "startup permission denied") {
		t.Fatalf("lost failure detail: %v", err)
	}
}
