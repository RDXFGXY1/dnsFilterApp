//go:build !windows

package autostart

import "fmt"

// Supported reports whether autostart can be managed on this OS.
func Supported() bool { return false }

// IsEnabled always returns false on non-Windows platforms.
func IsEnabled() bool { return false }

// Enable is not implemented outside Windows.
func Enable() error {
	return fmt.Errorf("start-on-login is currently supported on Windows only")
}

// Disable is not implemented outside Windows.
func Disable() error {
	return fmt.Errorf("start-on-login is currently supported on Windows only")
}
