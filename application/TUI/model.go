package main

import (
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	apiURL, username, password string
	client                     *http.Client
	snapshot                   Snapshot
	currentView                View
	cursor, width, height      int
	authenticated, loading     bool
	autoRefresh, showHelp      bool
	errorMsg, notice           string
	lastUpdate                 time.Time
	controller                 *ServiceController
	settings                   ConsoleSettings
	showSetup, showSettings    bool
	setupStep, settingCursor   int
}

func NewModel(apiURL string, controller *ServiceController, settings ConsoleSettings, startErr error) Model {
	jar, _ := cookiejar.New(nil)
	m := Model{
		apiURL:   strings.TrimSuffix(strings.TrimRight(apiURL, "/"), "/api"),
		client:   &http.Client{Jar: jar, Timeout: 5 * time.Second},
		username: envOr("DNS_FILTER_USERNAME", "admin"), password: envOr("DNS_FILTER_PASSWORD", "changeme"),
		loading: true, autoRefresh: true, controller: controller, settings: settings, showSetup: !settings.SetupComplete,
	}
	if startErr != nil {
		m.errorMsg = startErr.Error()
	}
	return m
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (m Model) Init() tea.Cmd {
	if !m.controller.Status().Dashboard {
		return tea.Batch(nextTick(), pollTerminalSize())
	}
	return tea.Batch(m.login(), nextTick(), pollTerminalSize())
}

func nextTick() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
