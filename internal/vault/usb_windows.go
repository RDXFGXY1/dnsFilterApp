//go:build windows

package vault

import (
	"fmt"
	"golang.org/x/sys/windows"
	"strings"
)

func RemovableRoots() []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	roots := []string{}
	for i := 0; i < 26; i++ {
		if mask&(1<<i) == 0 {
			continue
		}
		root := fmt.Sprintf("%c:\\", 'A'+i)
		ptr, err := windows.UTF16PtrFromString(root)
		if err == nil && windows.GetDriveType(ptr) == windows.DRIVE_REMOVABLE {
			roots = append(roots, root)
		}
	}
	return roots
}

func validRemovableRoot(root string) bool {
	root = strings.ToUpper(strings.TrimSpace(root))
	for _, candidate := range RemovableRoots() {
		if root == strings.ToUpper(candidate) {
			return true
		}
	}
	return false
}
