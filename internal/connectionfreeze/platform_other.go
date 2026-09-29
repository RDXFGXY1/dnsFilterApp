//go:build !windows

package connectionfreeze

import "fmt"

func supported() bool { return false }
func startPlatform() (backend, error) {
	return nil, fmt.Errorf("Connection Freeze currently requires Windows")
}
