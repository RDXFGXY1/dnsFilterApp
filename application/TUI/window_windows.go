//go:build windows

package main

import "golang.org/x/sys/windows"

var (
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	user32              = windows.NewLazySystemDLL("user32.dll")
	getConsoleWindow    = kernel32.NewProc("GetConsoleWindow")
	showWindow          = user32.NewProc("ShowWindow")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
)

func consoleHandle() uintptr { handle, _, _ := getConsoleWindow.Call(); return handle }
func hideConsoleWindow() {
	if handle := consoleHandle(); handle != 0 {
		showWindow.Call(handle, 0)
	}
}
func showConsoleWindow() {
	if handle := consoleHandle(); handle != 0 {
		showWindow.Call(handle, 5)
		setForegroundWindow.Call(handle)
	}
}
