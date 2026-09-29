//go:build !windows

package dnsbypass

import "errors"

type unsupportedPlatform struct{}

func newPlatform() Platform                 { return unsupportedPlatform{} }
func (unsupportedPlatform) Supported() bool { return false }
func (unsupportedPlatform) Enable() error {
	return errors.New("DNS bypass protection requires Windows")
}
func (unsupportedPlatform) Disable() error {
	return errors.New("DNS bypass protection requires Windows")
}
func (unsupportedPlatform) Active() (bool, error) { return false, nil }
