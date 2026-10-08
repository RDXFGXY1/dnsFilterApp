//go:build windows

package discipline

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procForeground  = user32.NewProc("GetForegroundWindow")
	procWindowPID   = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess = kernel32.NewProc("OpenProcess")
	procQueryImage  = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle = kernel32.NewProc("CloseHandle")
)

func foregroundApplication() string {
	hwnd, _, _ := procForeground.Call()
	if hwnd == 0 {
		return ""
	}
	var pid uint32
	procWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	const queryLimited = 0x1000
	h, _, _ := procOpenProcess.Call(queryLimited, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer procCloseHandle.Call(h)
	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	ok, _, _ := procQueryImage.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ok == 0 {
		return ""
	}
	return filepath.Base(syscall.UTF16ToString(buf[:size]))
}
