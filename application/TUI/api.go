package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) endpoint(path string) string { return m.apiURL + "/api/" + strings.TrimLeft(path, "/") }

func (m Model) get(path string, target any) error {
	resp, err := m.client.Get(m.endpoint(path))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (m Model) login() tea.Cmd {
	return func() tea.Msg {
		payload, _ := json.Marshal(map[string]string{"username": m.username, "password": m.password})
		resp, err := m.client.Post(m.apiURL+"/login", "application/json", strings.NewReader(string(payload)))
		if err != nil {
			return errMsg{fmt.Errorf("server unavailable: %w", err)}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return errMsg{fmt.Errorf("authentication failed (HTTP %d)", resp.StatusCode)}
		}
		return authenticatedMsg{}
	}
}

func (m Model) fetchSnapshot() tea.Cmd {
	return func() tea.Msg {
		var out Snapshot
		var stats statsResponse
		requests := []struct {
			path   string
			target any
		}{
			{"stats", &stats}, {"recent", &out.Queries}, {"stats/top-blocked", &out.TopBlocked},
			{"stats/timeline", &out.Timeline}, {"whitelist", &out.Whitelist}, {"categories", &out.Categories},
			{"system/info", &out.System}, {"protection/health", &out.Health}, {"filtering", &out.Filtering},
		}
		for _, request := range requests {
			if err := m.get(request.path, request.target); err != nil {
				return errMsg{fmt.Errorf("refresh %s failed: %w", request.path, err)}
			}
		}
		out.Stats = stats.Stats
		out.Stats.BlockedDomains = stats.BlockedDomains
		// /api/recent contains blocked events only, so mark them explicitly.
		for i := range out.Queries {
			out.Queries[i].Blocked = true
		}
		return snapshotMsg(out)
	}
}

func (m Model) openDashboard() tea.Cmd {
	return func() tea.Msg {
		browserURL := m.apiURL
		if m.controller != nil && m.apiURL == m.controller.DashboardURL() {
			browserURL = m.controller.BrowserURL()
		}
		if err := openURL(browserURL); err != nil {
			return errMsg{fmt.Errorf("could not open dashboard: %w", err)}
		}
		return dashboardOpenedMsg{}
	}
}

func openURL(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		command = exec.Command("open", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	return command.Start()
}
