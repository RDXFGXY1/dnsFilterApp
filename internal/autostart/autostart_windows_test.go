//go:build windows

package autostart

import (
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
		`'C:\Program Files\DNS Filter\dns-filter.exe'`,
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("registration script is missing %q", expected)
		}
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
