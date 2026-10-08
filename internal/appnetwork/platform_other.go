//go:build !windows && !linux

package appnetwork

import "errors"

type unsupportedPlatform struct{}

func newPlatform() platform                 { return unsupportedPlatform{} }
func (unsupportedPlatform) Supported() bool { return false }
func (unsupportedPlatform) RunningApps() ([]App, error) {
	return nil, errors.New("App Network Control requires Windows")
}
func (unsupportedPlatform) Block(string, string) error {
	return errors.New("App Network Control requires Windows")
}
func (unsupportedPlatform) Allow(string) error {
	return errors.New("App Network Control requires Windows")
}
func validatePathPlatform(string) error {
	return errors.New("App Network Control requires Windows or Linux with cgroup v2 and nftables")
}
