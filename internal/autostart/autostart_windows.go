//go:build windows

// Package autostart manages the background DNS Filter service on Windows.
// A SYSTEM scheduled task starts at boot independently of the desktop GUI and
// retries after an unexpected process exit.
package autostart

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

const taskName = "DNSFilter"

// Supported reports whether autostart can be managed on this OS.
func Supported() bool { return true }

// IsEnabled reports whether the logon task exists.
func IsEnabled() bool {
	return exec.Command("schtasks", "/query", "/tn", taskName).Run() == nil
}

func psQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func encodedPowerShell(script string) string {
	encoded := utf16.Encode([]rune(script))
	bytes := make([]byte, len(encoded)*2)
	for i, value := range encoded {
		bytes[i*2] = byte(value)
		bytes[i*2+1] = byte(value >> 8)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

func serviceWorkingDirectory(exe string) string {
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "configs", "config.yaml")); err == nil {
		return dir
	}
	parent := filepath.Dir(dir)
	if _, err := os.Stat(filepath.Join(parent, "configs", "config.yaml")); err == nil {
		return parent
	}
	return dir
}

func taskRegistrationScript(exe string) string {
	dir := serviceWorkingDirectory(exe)
	return `$ErrorActionPreference = 'Stop'
$action = New-ScheduledTaskAction -Execute ` + psQuote(exe) + ` -WorkingDirectory ` + psQuote(dir) + `
$triggers = @((New-ScheduledTaskTrigger -AtStartup), (New-ScheduledTaskTrigger -AtLogOn))
$settings = New-ScheduledTaskSettingsSet -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -StartWhenAvailable -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew
$settings.DisallowStartIfOnBatteries = $false
$settings.StopIfGoingOnBatteries = $false
Register-ScheduledTask -TaskName 'DNSFilter' -Action $action -Trigger $triggers -Settings $settings -User 'SYSTEM' -RunLevel Highest -Force | Out-Null`
}

// Enable creates or replaces the resilient boot task for the current exe.
func Enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	script := taskRegistrationScript(exe)
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodedPowerShell(script)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Disable removes the logon task.
func Disable() error {
	out, err := exec.Command("schtasks", "/delete", "/tn", taskName, "/f").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
