package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) serviceAction(action string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch action {
		case "start":
			err = m.controller.Start()
		case "stop":
			err = m.controller.Stop(true)
		case "setup-dns":
			err = m.controller.SetupDNS()
		case "restore-dns":
			err = m.controller.RestoreDNS()
		case "auto-on":
			err = m.controller.SetAutoLaunch(true)
		case "auto-off":
			err = m.controller.SetAutoLaunch(false)
		case "filter-on":
			m.controller.SetFiltering(true)
		case "filter-off":
			m.controller.SetFiltering(false)
		case "finish-setup":
			if !m.controller.Status().Running {
				err = m.controller.Start()
			}
			if err == nil {
				err = m.controller.SetupDNS()
			}
			if err == nil && m.settings.AutoLaunch {
				err = m.controller.SetAutoLaunch(true)
			}
			if err == nil {
				m.settings.SetupComplete = true
				m.settings.DNSManaged = true
				err = saveConsoleSettings(m.controller.Root(), m.settings)
			}
		default:
			err = fmt.Errorf("unknown action %q", action)
		}
		return serviceActionMsg{action: action, err: err}
	}
}

func (m Model) saveSettingsCmd() tea.Cmd {
	return func() tea.Msg {
		return serviceActionMsg{action: "save-settings", err: saveConsoleSettings(m.controller.Root(), m.settings)}
	}
}

func (m *Model) toggleSetting() tea.Cmd {
	status := m.controller.Status()
	switch m.settingCursor {
	case 0:
		if status.Running {
			return m.serviceAction("stop")
		}
		return m.serviceAction("start")
	case 1:
		if status.Filtering {
			return m.serviceAction("filter-off")
		}
		return m.serviceAction("filter-on")
	case 2:
		if status.DNSRouted {
			return m.serviceAction("restore-dns")
		}
		return m.serviceAction("setup-dns")
	case 3:
		m.settings.AutoLaunch = !status.AutoLaunch
		if m.settings.AutoLaunch {
			return m.serviceAction("auto-on")
		}
		return m.serviceAction("auto-off")
	case 4:
		m.settings.StartServerOnLaunch = !m.settings.StartServerOnLaunch
	case 5:
		m.settings.StartMinimized = !m.settings.StartMinimized
	case 6:
		m.settings.CloseToTray = !m.settings.CloseToTray
	case 7:
		m.showSettings = false
		m.showSetup = true
		m.setupStep = 0
		return nil
	}
	return m.saveSettingsCmd()
}
