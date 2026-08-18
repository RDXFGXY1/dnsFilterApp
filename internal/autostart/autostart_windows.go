//go:build windows

// Package autostart manages launching DNS Filter automatically at login.
// On Windows it uses a Scheduled Task set to run at logon with the highest
// privileges, so the app starts elevated (ports 53/80/443) without a UAC prompt.
package autostart

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const taskName = "DNSFilter"

// Supported reports whether autostart can be managed on this OS.
func Supported() bool { return true }

// IsEnabled reports whether the logon task exists.
func IsEnabled() bool {
	return exec.Command("schtasks", "/query", "/tn", taskName).Run() == nil
}

// Enable creates (or replaces) a logon scheduled task for the current exe.
func Enable() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	out, err := exec.Command("schtasks", "/create", "/tn", taskName,
		"/tr", `"`+exe+`"`, "/sc", "onlogon", "/rl", "highest", "/f").CombinedOutput()
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
