package main

import (
	"net"
	"os"
	"strings"
	"testing"

	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/charmbracelet/lipgloss"
)

func TestSparklineUsesTimelineValues(t *testing.T) {
	got := sparkline([]TimelinePoint{{Count: 0}, {Count: 5}, {Count: 10}}, 10)
	if got != "▁▄█" {
		t.Fatalf("unexpected sparkline %q", got)
	}
}

func TestPreflightRejectsOccupiedDNSPort(t *testing.T) {
	listener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.LocalAddr().(*net.UDPAddr).Port
	cfg := &config.Config{Server: config.ServerConfig{DNSHost: "127.0.0.1", DNSPort: port, APIHost: "127.0.0.1", APIPort: 0}}
	if err = preflightPorts(cfg); err == nil {
		t.Fatal("expected occupied DNS port to be rejected")
	}
}

func TestConsoleSettingsRoundTrip(t *testing.T) {
	root := t.TempDir()
	expected := defaultConsoleSettings()
	expected.SetupComplete = true
	expected.AutoLaunch = true
	if err := saveConsoleSettings(root, expected); err != nil {
		t.Fatal(err)
	}
	actual, err := loadConsoleSettings(root)
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("settings mismatch: %#v", actual)
	}
	if info, err := os.Stat(consoleSettingsPath(root)); err != nil || info.Size() == 0 {
		t.Fatalf("settings file not created: %v", err)
	}
}

func TestTrayIconIsICO(t *testing.T) {
	icon := trayIcon()
	if len(icon) < 100 || icon[2] != 1 || icon[4] != 1 {
		t.Fatalf("invalid generated ICO (%d bytes)", len(icon))
	}
}

func TestResponsiveModalUsesFullscreenSpace(t *testing.T) {
	if got := responsiveModalWidth(200, 56, 132); got != 132 {
		t.Fatalf("fullscreen modal width = %d", got)
	}
	if got := responsiveModalWidth(120, 56, 132); got < 85 {
		t.Fatalf("desktop modal remained too narrow: %d", got)
	}
	if got := responsiveModalWidth(50, 56, 132); got > 42 {
		t.Fatalf("small terminal modal overflowed: %d", got)
	}
}

func TestPolledTerminalSizeReplacesStaleWindowSize(t *testing.T) {
	m := NewModel("http://127.0.0.1:8080", &ServiceController{}, defaultConsoleSettings(), nil)
	m.width, m.height = 90, 30
	updated, _ := m.Update(terminalSizeMsg{width: 190, height: 58})
	got := updated.(Model)
	if got.width != 190 || got.height != 58 {
		t.Fatalf("terminal stayed stale at %dx%d", got.width, got.height)
	}
}

func TestHelpViewUsesRealLineBreaks(t *testing.T) {
	m := NewModel("http://127.0.0.1:8080", &ServiceController{}, defaultConsoleSettings(), nil)
	m.width, m.height = 190, 58
	rendered := m.helpView()
	if strings.Contains(rendered, `\n`) {
		t.Fatal("help view rendered literal newline escapes")
	}
	card := strings.TrimLeft(rendered, "\n")
	if lipgloss.Height(card) > 22 {
		t.Fatalf("help card unexpectedly consumed %d lines", lipgloss.Height(card))
	}
}

func TestReadOnlyViewsHaveStableRows(t *testing.T) {
	m := NewModel("http://127.0.0.1:8080", &ServiceController{}, defaultConsoleSettings(), nil)
	m.snapshot.Queries = []Query{{Domain: "blocked.example"}}
	m.currentView = viewTraffic
	if m.rowCount() != 1 {
		t.Fatalf("traffic row count = %d", m.rowCount())
	}
	m.changeView(viewSystem)
	if m.cursor != 0 || m.currentView != viewSystem {
		t.Fatalf("view change did not reset navigation")
	}
}
