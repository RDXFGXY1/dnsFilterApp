//go:build windows

package systemcheck

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// IsAdministrator reports whether the current process is running elevated.
func IsAdministrator() (bool, string) {
	token := windows.GetCurrentProcessToken()
	if token.IsElevated() {
		return true, "Service is running with Administrator rights"
	}
	return false, "Administrator rights are required to configure Windows DNS and firewall protection"
}

// FirewallStatus reports whether every Windows Firewall profile is enabled.
func FirewallStatus() (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "@((Get-NetFirewallProfile -ErrorAction Stop) | Where-Object { -not $_.Enabled }).Count")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, "Windows Firewall status could not be verified"
	}
	disabled, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return false, "Windows Firewall returned an unknown status"
	}
	if disabled > 0 {
		return false, "One or more Windows Firewall profiles are disabled"
	}
	return true, "Windows Firewall is enabled"
}
