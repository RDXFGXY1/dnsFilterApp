//go:build windows

// Desktop uses Go and native Windows controls, without a script host.
package main

import (
	"fmt"
	"github.com/RDXFGXY1/dns-filter-app/internal/netdns"
	"gopkg.in/yaml.v3"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

var (
	user                                      = syscall.NewLazyDLL("user32.dll")
	kernel                                    = syscall.NewLazyDLL("kernel32.dll")
	shell                                     = syscall.NewLazyDLL("shell32.dll")
	createWindow                              = user.NewProc("CreateWindowExW")
	sendMessage                               = user.NewProc("SendMessageW")
	mainWindow, statusLabel, dnsLabel, logBox uintptr
	buttons                                   = map[int]uintptr{}
	root, server, dashboardURL                string
	port                                      = 8080
	ready, checking, busy, starting           bool
	lastCheck                                 time.Time
	events                                    = make(chan update, 16)
)

type update struct {
	kind, text, dns string
	ready           bool
}
type point struct{ X, Y int32 }
type message struct {
	Window         uintptr
	ID             uint32
	WParam, LParam uintptr
	Time           uint32
	Point          point
	Private        uint32
}
type windowClass struct {
	Style                              uint32
	Callback                           uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Name                         *uint16
}
type shellInfo struct {
	Size, Mask                        uint32
	Window                            uintptr
	Verb, File, Parameters, Directory *uint16
	Show                              int32
	Instance, IDList                  uintptr
	Class                             *uint16
	ClassKey                          uintptr
	HotKey                            uint32
	Icon, Process                     uintptr
}

func wide(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func text(h uintptr, s string) {
	user.NewProc("SetWindowTextW").Call(h, uintptr(unsafe.Pointer(wide(s))))
}
func enabled(h uintptr, on bool) {
	var v uintptr
	if on {
		v = 1
	}
	user.NewProc("EnableWindow").Call(h, v)
}
func popup(s string, flags uintptr) uintptr {
	r, _, _ := user.NewProc("MessageBoxW").Call(mainWindow, uintptr(unsafe.Pointer(wide(s))), uintptr(unsafe.Pointer(wide("DNS Filter"))), flags)
	return r
}
func logLine(s string) {
	sendMessage.Call(logBox, 0xB1, ^uintptr(0), ^uintptr(0))
	sendMessage.Call(logBox, 0xC2, 0, uintptr(unsafe.Pointer(wide(time.Now().Format("15:04:05")+"  "+s+"\r\n"))))
}
func control(class, title string, x, y, w, h, id int, style uintptr) uintptr {
	handle, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(wide(class))), uintptr(unsafe.Pointer(wide(title))), 0x50000000|style, uintptr(x), uintptr(y), uintptr(w), uintptr(h), mainWindow, uintptr(id), 0, 0)
	if handle == 0 {
		panic(fmt.Sprintf("Cannot create %s: %v", title, err))
	}
	font, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("GetStockObject").Call(17)
	sendMessage.Call(handle, 0x30, font, 1)
	return handle
}
func button(title string, x, y, id int) {
	buttons[id] = control("BUTTON", title, x, y, 332, 44, id, 0x10000)
}
func syncButtons() {
	enabled(buttons[1], !ready && !starting && !busy)
	enabled(buttons[2], ready)
	enabled(buttons[3], ready && !busy)
	enabled(buttons[4], !busy)
	enabled(buttons[7], ready)
}
func refresh() {
	if checking {
		return
	}
	checking = true
	lastCheck = time.Now()
	go func() {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
		reachable := err == nil
		if conn != nil {
			conn.Close()
		}
		iface, dns, _, err := netdns.Status()
		detail := "IPv4 / IPv6 DNS: " + iface + " → " + dns
		if err != nil {
			detail = "DNS status unavailable: " + err.Error()
		}
		events <- update{kind: "status", dns: detail, ready: reachable}
	}()
}

// Request elevation through Windows and retain the actual child process handle.
func launch(file, args, verb string, show int32) (syscall.Handle, error) {
	info := shellInfo{Mask: 0x40 | 0x100, Window: mainWindow, Verb: wide(verb), File: wide(file), Parameters: wide(args), Directory: wide(root), Show: show}
	info.Size = uint32(unsafe.Sizeof(info))
	ok, _, err := shell.NewProc("ShellExecuteExW").Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(info)
	if ok == 0 {
		return 0, fmt.Errorf("Windows could not open the application (permission may have been cancelled): %v", err)
	}
	return syscall.Handle(info.Process), nil
}
func open(file string) {
	h, err := launch(file, "", "open", 1)
	if err != nil {
		logLine(err.Error())
	}
	if h != 0 {
		syscall.CloseHandle(h)
	}
}
func run(action string) {
	if busy || (action == "" && starting) {
		return
	}
	if _, err := os.Stat(server); err != nil {
		popup("Server executable is missing:\n"+server, 0x10)
		return
	}
	h, err := launch(server, action, "runas", 0)
	if err != nil {
		logLine(err.Error())
		return
	}
	if h == 0 {
		logLine("Windows did not return a process handle. Check server logs.")
		return
	}
	if action == "" {
		starting = true
		logLine("Server starting. Loading blocklists can take a moment.")
	} else {
		busy = true
		logLine("Applying " + action + "…")
	}
	syncButtons()
	go func() {
		defer syscall.CloseHandle(h)
		result, err := syscall.WaitForSingleObject(h, syscall.INFINITE)
		var code uint32
		if err == nil && result == syscall.WAIT_OBJECT_0 {
			err = syscall.GetExitCodeProcess(h, &code)
		}
		kind := "operation"
		if action == "" {
			kind = "server"
			action = "Server"
		}
		detail := action + " completed successfully."
		if err != nil {
			detail = action + " could not be verified: " + err.Error()
		} else if code != 0 {
			detail = fmt.Sprintf("%s exited with code %d. Check server logs and local DNS availability.", action, code)
		}
		events <- update{kind: kind, text: detail}
	}()
}
func tick() {
	for {
		select {
		case e := <-events:
			switch e.kind {
			case "status":
				ready = e.ready
				checking = false
				s := "Server is not reachable. Start it before enabling local DNS."
				if ready {
					s = "Dashboard port is reachable. Open the dashboard to check protection."
				}
				text(statusLabel, s)
				text(dnsLabel, e.dns)
			case "operation":
				busy = false
				logLine(e.text)
				refresh()
			case "server":
				starting = false
				logLine(e.text)
				refresh()
			}
			syncButtons()
		default:
			if time.Since(lastCheck) > 5*time.Second {
				refresh()
			}
			return
		}
	}
}
func windowProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	switch msg {
	case 0x111:
		switch int(w & 0xffff) {
		case 1:
			run("")
		case 2:
			open(dashboardURL)
		case 3:
			run("setup-dns")
		case 4:
			if popup("Restore automatic DNS (DHCP) on the current adapter?\nThis replaces manually configured DNS addresses.", 0x24) == 6 {
				run("reset-dns")
			}
		case 5:
			open(filepath.Join(root, "data", "logs"))
		case 6:
			refresh()
		case 7:
			open(dashboardURL + "/#freeze")
		}
		return 0
	case 0x113:
		tick()
		return 0
	case 0x10:
		if busy {
			popup("Wait for the DNS operation to finish before closing.", 0x40)
			return 0
		}
		user.NewProc("DestroyWindow").Call(h)
		return 0
	case 2:
		user.NewProc("PostQuitMessage").Call(0)
		return 0
	}
	r, _, _ := user.NewProc("DefWindowProcW").Call(h, uintptr(msg), w, l)
	return r
}
func main() {
	runtime.LockOSThread()
	defer func() {
		if e := recover(); e != nil {
			popup(fmt.Sprint(e), 0x10)
		}
	}()
	exe, err := os.Executable()
	if err != nil {
		panic(err)
	}
	root = filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(root, "configs", "config.yaml")); err != nil {
		root = filepath.Dir(root)
	}
	data, err := os.ReadFile(filepath.Join(root, "configs", "config.yaml"))
	if err != nil {
		panic("Keep the desktop executable in the project's build folder. Configuration was not found.")
	}
	var cfg struct {
		Server struct {
			Port int `yaml:"api_port"`
		} `yaml:"server"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		panic(err)
	}
	if cfg.Server.Port > 0 {
		port = cfg.Server.Port
	}
	server = filepath.Join(root, "build", "dns-filter-gui-server.exe")
	dashboardURL = "http://dnsfilter.localhost:" + strconv.Itoa(port)
	instance, _, _ := kernel.NewProc("GetModuleHandleW").Call(0)
	cursor, _, _ := user.NewProc("LoadCursorW").Call(0, 32512)
	wc := windowClass{Callback: syscall.NewCallback(windowProc), Instance: instance, Cursor: cursor, Background: 16, Name: wide("DNSFilterGoDesktop")}
	if r, _, err := user.NewProc("RegisterClassW").Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		panic(err)
	}
	mainWindow, _, err = createWindow.Call(0, uintptr(unsafe.Pointer(wc.Name)), uintptr(unsafe.Pointer(wide("DNS Filter — Go Desktop"))), 0x00CA0000, 0x80000000, 0x80000000, 780, 674, 0, 0, instance, 0)
	if mainWindow == 0 {
		panic(err)
	}
	control("STATIC", "DNS FILTER  /  DESKTOP CONTROL", 28, 24, 700, 28, 0, 0)
	control("STATIC", "Local protection • Native Go application", 28, 60, 700, 24, 0, 0)
	statusLabel = control("STATIC", "Checking server…", 28, 104, 700, 28, 0, 0)
	dnsLabel = control("STATIC", "Checking DNS…", 28, 140, 700, 48, 0, 0)
	button("Start server", 28, 204, 1)
	button("Open dashboard", 390, 204, 2)
	button("Enable local DNS", 28, 262, 3)
	button("Restore automatic DNS (DHCP)", 390, 262, 4)
	control("STATIC", "Closing this window leaves the server running. DNS changes need Windows permission.", 28, 324, 700, 32, 0, 0)
	logBox = control("EDIT", "", 28, 365, 694, 132, 0, 0x00800844)
	button("Open server logs", 28, 516, 5)
	button("Refresh status", 390, 516, 6)
	buttons[7] = control("BUTTON", "Connection Freeze & logs — open dashboard controls", 28, 574, 694, 44, 7, 0x10000)
	logLine("Ready. Local DNS setup covers IPv4 and IPv6 on the current adapter.")
	logLine("Dashboard: " + dashboardURL + " | Fallback: http://127.0.0.1:" + strconv.Itoa(port))
	syncButtons()
	refresh()
	user.NewProc("SetTimer").Call(mainWindow, 1, 200, 0)
	user.NewProc("ShowWindow").Call(mainWindow, 5)
	user.NewProc("UpdateWindow").Call(mainWindow)
	var msg message
	for {
		result, _, err := user.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(result) == -1 {
			panic(err)
		}
		if result == 0 {
			break
		}
		if handled, _, _ := user.NewProc("IsDialogMessageW").Call(mainWindow, uintptr(unsafe.Pointer(&msg))); handled != 0 {
			continue
		}
		user.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&msg)))
		user.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&msg)))
	}
}
