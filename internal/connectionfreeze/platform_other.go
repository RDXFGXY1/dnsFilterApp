//go:build !windows && !linux

package connectionfreeze

import "fmt"

func supported() bool { return false }
func startPlatform() (backend, error) {
	return nil, fmt.Errorf("Connection Freeze currently requires Windows or Linux with nftables")
}
