package main

import tea "github.com/charmbracelet/bubbletea"

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case terminalSizeMsg:
		if msg.width > 0 && msg.height > 0 && (msg.width != m.width || msg.height != m.height) {
			m.width, m.height = msg.width, msg.height
		}
		return m, pollTerminalSize()
	case authenticatedMsg:
		m.authenticated, m.loading, m.errorMsg = true, true, ""
		return m, m.fetchSnapshot()
	case snapshotMsg:
		m.snapshot, m.loading, m.errorMsg = Snapshot(msg), false, ""
		m.lastUpdate = now()
		m.clampCursor()
	case dashboardOpenedMsg:
		m.notice = "Dashboard opened — configuration changes stay in the browser"
	case serviceActionMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.notice = ""
		} else {
			m.errorMsg = ""
			m.notice = "Action completed: " + msg.action
			if msg.action == "finish-setup" {
				m.settings.SetupComplete = true
				m.showSetup = false
				m.authenticated = false
				return m, m.login()
			}
			if msg.action == "start" {
				m.authenticated = false
				return m, m.login()
			}
			if msg.action == "stop" {
				m.authenticated = false
			}
			if msg.action == "setup-dns" {
				m.settings.DNSManaged = true
				_ = saveConsoleSettings(m.controller.Root(), m.settings)
			}
			if msg.action == "restore-dns" {
				m.settings.DNSManaged = false
				_ = saveConsoleSettings(m.controller.Root(), m.settings)
			}
			if msg.action == "auto-on" {
				m.settings.AutoLaunch = true
				_ = saveConsoleSettings(m.controller.Root(), m.settings)
			}
			if msg.action == "auto-off" {
				m.settings.AutoLaunch = false
				_ = saveConsoleSettings(m.controller.Root(), m.settings)
			}
		}
	case errMsg:
		m.loading, m.errorMsg = false, msg.Error()
	case tickMsg:
		if m.authenticated && m.autoRefresh {
			return m, tea.Batch(m.fetchSnapshot(), nextTick())
		}
		return m, nextTick()
	case tea.KeyMsg:
		if m.showSetup {
			return m.updateSetup(msg)
		}
		if m.showSettings {
			return m.updateSettings(msg)
		}
		if m.showHelp {
			if oneOfKey(msg.String(), "?", "f1", "esc") {
				m.showHelp = false
			}
			if oneOfKey(msg.String(), "q", "ctrl+c") {
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "q":
			if m.settings.CloseToTray {
				hideConsoleWindow()
				m.notice = "Console hidden — DNS Filter is still running in the tray"
				return m, nil
			}
			return m, tea.Quit
		case "ctrl+c":
			return m, tea.Quit
		case "?", "f1":
			m.showHelp = true
		case "s":
			m.showSettings = true
			m.settingCursor = 0
		case "1":
			m.changeView(viewOverview)
		case "2":
			m.changeView(viewTraffic)
		case "3":
			m.changeView(viewThreats)
		case "4":
			m.changeView(viewPolicy)
		case "5":
			m.changeView(viewSystem)
		case "tab", "right", "l":
			m.changeView(View((int(m.currentView) + 1) % len(viewNames)))
		case "shift+tab", "left", "h":
			m.changeView(View((int(m.currentView) + len(viewNames) - 1) % len(viewNames)))
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor+1 < m.rowCount() {
				m.cursor++
			}
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = maxInt(0, m.rowCount()-1)
		case "r":
			m.loading = true
			return m, m.fetchSnapshot()
		case "space":
			m.autoRefresh = !m.autoRefresh
		case "o":
			return m, m.openDashboard()
		}
	case tea.MouseMsg:
		return m.updateMouse(tea.MouseEvent(msg))
	}
	return m, nil
}

func (m Model) updateSetup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "left", "shift+tab":
		if m.setupStep > 0 {
			m.setupStep--
		}
	case "right", "tab", "enter":
		if m.setupStep < 3 {
			m.setupStep++
		} else {
			return m, m.serviceAction("finish-setup")
		}
	case "a":
		m.settings.AutoLaunch = !m.settings.AutoLaunch
	case "m":
		m.settings.StartMinimized = !m.settings.StartMinimized
	}
	return m, nil
}

func (m Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "s":
		m.showSettings = false
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.settingCursor > 0 {
			m.settingCursor--
		}
	case "down", "j":
		if m.settingCursor < 7 {
			m.settingCursor++
		}
	case "enter", "space":
		return m, m.toggleSetting()
	}
	return m, nil
}

func (m Model) updateMouse(msg tea.MouseEvent) (tea.Model, tea.Cmd) {
	if msg.Type == tea.MouseWheelUp {
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}
	if msg.Type == tea.MouseWheelDown {
		if m.cursor+1 < m.rowCount() {
			m.cursor++
		}
		return m, nil
	}
	if msg.Type != tea.MouseLeft {
		return m, nil
	}
	if m.showSetup {
		if msg.Y >= m.height-5 {
			if m.setupStep < 3 {
				m.setupStep++
			} else {
				return m, m.serviceAction("finish-setup")
			}
		}
		return m, nil
	}
	if m.showSettings {
		row := msg.Y - 8
		if row >= 0 && row < 8 {
			m.settingCursor = row
			return m, m.toggleSetting()
		}
		return m, nil
	}
	if msg.X <= 22 && msg.Y >= 6 && msg.Y <= 10 {
		m.changeView(View(msg.Y - 6))
		return m, nil
	}
	if msg.X <= 22 && msg.Y == 14 {
		return m, m.openDashboard()
	}
	return m, nil
}

func (m *Model) changeView(view View) { m.currentView, m.cursor, m.notice = view, 0, "" }
func (m *Model) clampCursor() {
	if m.cursor >= m.rowCount() {
		m.cursor = maxInt(0, m.rowCount()-1)
	}
}
func (m Model) rowCount() int {
	switch m.currentView {
	case viewTraffic:
		return len(m.snapshot.Queries)
	case viewThreats:
		return len(m.snapshot.TopBlocked)
	case viewPolicy:
		return len(m.snapshot.Categories) + len(m.snapshot.Whitelist)
	case viewSystem:
		return len(m.snapshot.Health.Checks)
	default:
		return 0
	}
}
