//go:build !windows

package networkintel

import "errors"

type unsupported struct{}

func newPlatform() platform         { return unsupported{} }
func (unsupported) Supported() bool { return false }
func (unsupported) Connections() ([]Connection, error) {
	return nil, errors.New("connection intelligence requires Windows")
}
