//go:build !windows

package systemcheck

import "os"

func IsAdministrator() (bool, string) {
	if os.Geteuid() == 0 {
		return true, "Service is running with administrator rights"
	}
	return false, "Administrator rights are required to configure system DNS"
}

func FirewallStatus() (bool, string) {
	return true, "Firewall verification is handled by the operating system"
}
