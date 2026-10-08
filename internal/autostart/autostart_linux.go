//go:build linux

package autostart

import (
	"fmt"
	"os/exec"
)

const linuxServiceName = "dns-filter.service"

// Supported reports whether this Linux installation is managed by systemd.
func Supported() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// IsEnabled reports whether DNS Filter starts automatically at system boot.
func IsEnabled() bool {
	if !Supported() {
		return false
	}
	return exec.Command("systemctl", "is-enabled", "--quiet", linuxServiceName).Run() == nil
}

// Enable configures the already-installed service to start automatically.
func Enable() error {
	if !Supported() {
		return fmt.Errorf("systemd is required to manage automatic startup")
	}
	if out, err := exec.Command("systemctl", "enable", linuxServiceName).CombinedOutput(); err != nil {
		return fmt.Errorf("enable %s: %w: %s", linuxServiceName, err, string(out))
	}
	return nil
}

// Disable leaves the current process running but prevents startup on the next boot.
func Disable() error {
	if !Supported() {
		return fmt.Errorf("systemd is required to manage automatic startup")
	}
	if out, err := exec.Command("systemctl", "disable", linuxServiceName).CombinedOutput(); err != nil {
		return fmt.Errorf("disable %s: %w: %s", linuxServiceName, err, string(out))
	}
	return nil
}
