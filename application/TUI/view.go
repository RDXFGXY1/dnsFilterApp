package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Starting DNS Filter Console…"
	}
	if m.showSetup {
		return m.setupView()
	}
	if m.showSettings {
		return m.settingsView()
	}
	if !m.authenticated {
		return m.authView()
	}
	if m.showHelp {
		return m.helpView()
	}
	header := m.headerView()
	navWidth := 20
	contentWidth := maxInt(40, m.width-navWidth-3)
	bodyHeight := maxInt(8, m.height-lipgloss.Height(header)-3)
	nav := m.navView(navWidth, bodyHeight)
	content := m.contentView(contentWidth, bodyHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, nav, " ", content)
	return header + body + "\n" + m.footerView()
}

func (m Model) authView() string {
	status := "Connecting securely to the local service…"
	style := mutedStyle
	if !m.controller.Status().Running {
		status = "Service is stopped — press S to open settings and start it"
		style = warnStyle
	}
	if m.errorMsg != "" {
		status, style = m.errorMsg, badStyle
	}
	box := panelStyle.Width(minInt(58, maxInt(28, m.width-8))).Render(
		logoStyle.Render("◆ DNS FILTER CONSOLE") + "\n" +
			mutedStyle.Render("Local protection companion") + "\n\n" + style.Render(status) + "\n\n" +
			mutedStyle.Render("Make sure DNS Filter is running at ") + valueStyle.Render(m.apiURL))
	return strings.Repeat("\n", maxInt(1, m.height/2-5)) + lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box)
}

func (m Model) headerView() string {
	state, stateStyle := "PROTECTED", goodStyle
	if !m.snapshot.Health.Enabled {
		state, stateStyle = "FILTERING OFF", badStyle
	}
	if m.snapshot.Health.PausedUntil > time.Now().Unix() {
		state, stateStyle = "PAUSED", warnStyle
	}
	left := logoStyle.Render("◆ DNS FILTER") + "  " + mutedStyle.Render("READ-ONLY CONSOLE")
	right := stateStyle.Render("● "+state) + "  " + mutedStyle.Render(fmt.Sprintf("Health %d/100", m.snapshot.Health.Score))
	gap := strings.Repeat(" ", maxInt(1, m.width-lipgloss.Width(left)-lipgloss.Width(right)))
	sub := mutedStyle.Render("Local service  ") + valueStyle.Render(m.apiURL) + mutedStyle.Render("  •  Changes are made in the web dashboard")
	return left + gap + right + "\n" + sub + "\n" + lipgloss.NewStyle().Foreground(colors.border).Render(strings.Repeat("─", m.width)) + "\n"
}

func (m Model) navView(width, height int) string {
	var b strings.Builder
	b.WriteString(labelStyle.Render("MONITOR") + "\n\n")
	icons := []string{"▦", "↯", "◆", "◫", "⚙"}
	for i, name := range viewNames {
		line := fmt.Sprintf("%s  %d  %s", icons[i], i+1, name)
		if View(i) == m.currentView {
			b.WriteString(activeNav.Width(width - 2).Render(line))
		} else {
			b.WriteString(inactiveNav.Width(width - 2).Render(line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + labelStyle.Render("ACTIONS") + "\n\n")
	b.WriteString(inactiveNav.Render("O  Open dashboard") + "\n")
	b.WriteString(inactiveNav.Render("R  Refresh now") + "\n")
	b.WriteString(inactiveNav.Render("␠  Pause updates") + "\n")
	b.WriteString(inactiveNav.Render("S  Settings") + "\n")
	bottom := "\n" + mutedStyle.Render("Configuration is locked\nin this console.")
	return panelStyle.Width(width).Height(maxInt(1, height-2)).Render(b.String() + strings.Repeat("\n", maxInt(0, height-lipgloss.Height(b.String())-6)) + bottom)
}

func (m Model) contentView(width, height int) string {
	innerWidth := maxInt(32, width-4)
	var content string
	switch m.currentView {
	case viewTraffic:
		content = m.trafficView(innerWidth, height)
	case viewThreats:
		content = m.threatsView(innerWidth, height)
	case viewPolicy:
		content = m.policyView(innerWidth, height)
	case viewSystem:
		content = m.systemView(innerWidth, height)
	default:
		content = m.overviewView(innerWidth, height)
	}
	return panelStyle.Width(width).Height(maxInt(1, height-2)).Render(content)
}

func titleBlock(title, subtitle string) string {
	return valueStyle.Render(title) + "\n" + mutedStyle.Render(subtitle) + "\n\n"
}

func metric(label, value, note string, width int) string {
	body := labelStyle.Render(strings.ToUpper(label)) + "\n" + lipgloss.NewStyle().Foreground(colors.text).Bold(true).Render(value) + "\n" + mutedStyle.Render(note)
	return lipgloss.NewStyle().Background(colors.panelAlt).Padding(0, 1).Width(width).Render(body)
}

func (m Model) overviewView(width, height int) string {
	s := m.snapshot.Stats
	cardWidth := maxInt(13, (width-6)/4)
	cards := lipgloss.JoinHorizontal(lipgloss.Top,
		metric("Blocked 24h", comma(s.TotalBlocked), "requests stopped", cardWidth), " ",
		metric("Protected list", comma(s.BlockedDomains), "blocked domains", cardWidth), " ",
		metric("Unique domains", comma(s.UniqueDomains), "seen in 24 hours", cardWidth), " ",
		metric("Clients", comma(s.UniqueClients), "protected devices", cardWidth))
	chart := labelStyle.Render("24-HOUR BLOCK ACTIVITY") + "\n" + lipgloss.NewStyle().Foreground(colors.accent2).Render(sparkline(m.snapshot.Timeline, maxInt(10, width-2)))
	checks := m.healthRows(width, 4)
	recent := m.queryRows(width, maxInt(3, height-15))
	return titleBlock("Protection overview", "Live health, impact, and the newest blocked requests") + cards + "\n\n" + chart + "\n\n" + checks + "\n\n" + labelStyle.Render("LATEST EVENTS") + "\n" + recent
}

func (m Model) trafficView(width, height int) string {
	return titleBlock("Live blocked traffic", "Newest DNS decisions • automatically refreshes every three seconds") +
		mutedStyle.Render(pad("TIME", 10)+pad("DOMAIN", maxInt(16, width-30))+pad("CLIENT", 18)+"DECISION") + "\n" +
		m.queryRows(width, maxInt(4, height-7))
}

func (m Model) queryRows(width, limit int) string {
	if len(m.snapshot.Queries) == 0 {
		return mutedStyle.Render("No blocked requests recorded yet.")
	}
	domainWidth := maxInt(16, width-38)
	var b strings.Builder
	for i, q := range m.snapshot.Queries {
		if i >= limit {
			break
		}
		stamp := "—"
		if q.Timestamp > 0 {
			stamp = time.Unix(q.Timestamp, 0).Format("15:04:05")
		}
		line := pad(stamp, 10) + pad(q.Domain, domainWidth) + pad(q.ClientIP, 18) + "BLOCKED"
		style := lipgloss.NewStyle().Foreground(colors.text)
		if m.currentView == viewTraffic && i == m.cursor {
			style = selectedLine.Width(width)
		}
		b.WriteString(style.Render(line) + "\n")
	}
	return b.String()
}

func (m Model) threatsView(width, height int) string {
	var b strings.Builder
	b.WriteString(titleBlock("Threat concentration", "Domains generating the most blocked traffic in the last 24 hours"))
	maxCount := 1
	for _, item := range m.snapshot.TopBlocked {
		if item.Count > maxCount {
			maxCount = item.Count
		}
	}
	limit := maxInt(3, height-7)
	for i, item := range m.snapshot.TopBlocked {
		if i >= limit {
			break
		}
		barWidth := maxInt(8, minInt(30, width/3))
		filled := item.Count * barWidth / maxCount
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		line := fmt.Sprintf("%2d  %s %s %6d", i+1, pad(item.Domain, maxInt(16, width-barWidth-16)), bar, item.Count)
		style := lipgloss.NewStyle().Foreground(colors.text)
		if i == m.cursor {
			style = selectedLine.Width(width)
		}
		b.WriteString(style.Render(line) + "\n")
	}
	if len(m.snapshot.TopBlocked) == 0 {
		b.WriteString(mutedStyle.Render("No concentrated threats detected."))
	}
	return b.String()
}

func (m Model) policyView(width, height int) string {
	enabled := 0
	for _, c := range m.snapshot.Categories {
		if c.Enabled {
			enabled++
		}
	}
	var b strings.Builder
	b.WriteString(titleBlock("Policy inventory", "Read-only visibility into active categories and trusted exceptions"))
	b.WriteString(metric("Categories", fmt.Sprintf("%d / %d", enabled, len(m.snapshot.Categories)), "enabled", 20) + "  " + metric("Whitelist", fmt.Sprintf("%d", len(m.snapshot.Whitelist)), "trusted domains", 20) + "\n\n")
	b.WriteString(labelStyle.Render("CATEGORY COVERAGE") + "\n")
	limit := maxInt(2, (height-13)/2)
	for i, c := range m.snapshot.Categories {
		if i >= limit {
			break
		}
		state := goodStyle.Render("ENABLED")
		if !c.Enabled {
			state = mutedStyle.Render("OFF")
		}
		b.WriteString(pad(c.Name, maxInt(18, width-15)) + state + "\n")
	}
	b.WriteString("\n" + labelStyle.Render("TRUSTED DOMAINS") + "\n")
	for i, d := range m.snapshot.Whitelist {
		if i >= limit {
			break
		}
		b.WriteString(goodStyle.Render("✓ ") + truncate(d, width-3) + "\n")
	}
	if len(m.snapshot.Whitelist) == 0 {
		b.WriteString(mutedStyle.Render("No whitelist exceptions."))
	}
	return b.String()
}

func (m Model) systemView(width, height int) string {
	s := m.snapshot.System
	h := m.snapshot.Health
	status := "ACTIVE"
	if !m.snapshot.Filtering.Enabled {
		status = "DISABLED"
	}
	var b strings.Builder
	b.WriteString(titleBlock("Local service", "Server identity, routing, cache, and protection checks"))
	cardWidth := maxInt(12, (width-4)/3)
	b.WriteString(metric("Filtering", status, "engine state", cardWidth) + " " + metric("Health", fmt.Sprintf("%d/100", h.Score), "protection score", cardWidth) + " " + metric("Cache", fmt.Sprintf("%d", h.Cached), "DNS responses", cardWidth) + "\n\n")
	b.WriteString(labelStyle.Render("SERVICE DETAILS") + "\n")
	b.WriteString(fmt.Sprintf("Version          %s\nDNS listener     %s:%d\nWeb dashboard    127.0.0.1:%d\nUpstream DNS     %s\nCache TTL        %d seconds\nInterface        %s\n\n", s.Version, s.DNSHost, s.DNSPort, s.APIPort, strings.Join(s.UpstreamDNS, ", "), s.CacheTTL, h.Interface))
	b.WriteString(labelStyle.Render("PROTECTION CHECKS") + "\n" + m.healthRows(width, maxInt(2, height-16)))
	return b.String()
}

func (m Model) healthRows(width, limit int) string {
	var b strings.Builder
	for i, check := range m.snapshot.Health.Checks {
		if i >= limit {
			break
		}
		icon := goodStyle.Render("●")
		if !check.OK {
			icon = badStyle.Render("●")
		}
		b.WriteString(icon + " " + pad(check.Label, 22) + mutedStyle.Render(truncate(check.Detail, maxInt(10, width-26))) + "\n")
	}
	return b.String()
}

func (m Model) footerView() string {
	refresh := "AUTO ON"
	if !m.autoRefresh {
		refresh = "AUTO PAUSED"
	}
	left := fmt.Sprintf(" %s • updated %s", refresh, since(m.lastUpdate))
	if m.loading {
		left = " REFRESHING…"
	}
	if m.errorMsg != "" {
		left = " ERROR • " + truncate(m.errorMsg, maxInt(10, m.width/2))
	}
	if m.notice != "" {
		left = " " + m.notice
	}
	right := "O Dashboard   R Refresh   Space Auto   ? Help   Q Quit "
	gap := strings.Repeat(" ", maxInt(1, m.width-lipgloss.Width(left)-lipgloss.Width(right)))
	return mutedStyle.Render(left + gap + right)
}

func (m Model) helpView() string {
	boxWidth := responsiveModalWidth(m.width, 60, 118)
	navigation := labelStyle.Render("NAVIGATION") + "\n\n" +
		valueStyle.Render("1–5") + "       Switch monitoring views\n" +
		valueStyle.Render("Tab / ←→") + "  Move between views\n" +
		valueStyle.Render("↑↓ / J K") + "  Navigate rows and lists\n" +
		valueStyle.Render("Mouse") + "     Click views, actions, and scroll"
	actions := labelStyle.Render("ACTIONS") + "\n\n" +
		valueStyle.Render("R") + "         Refresh all analytics\n" +
		valueStyle.Render("Space") + "     Pause or resume auto-refresh\n" +
		valueStyle.Render("O") + "         Open the localhost dashboard\n" +
		valueStyle.Render("S") + "         Server and application settings\n" +
		valueStyle.Render("F1 / ?") + "    Toggle this help\n" +
		valueStyle.Render("Q") + "         Hide in the system tray"
	shortcutBody := navigation + "\n\n" + actions
	if boxWidth >= 88 {
		shortcutBody = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width((boxWidth-8)/2).Render(navigation), "   ", lipgloss.NewStyle().Width((boxWidth-8)/2).Render(actions))
	}
	content := logoStyle.Render("◆ DNS FILTER CONSOLE — HELP") + "\n\n" + shortcutBody + "\n\n" +
		mutedStyle.Render("The console manages service lifecycle and Windows integration.\nDetailed filtering rules remain in the localhost dashboard.")
	box := panelStyle.Width(boxWidth).Padding(1, 2).Render(content)
	top := maxInt(1, (m.height-lipgloss.Height(box))/2)
	return strings.Repeat("\n", top) + lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box)
}

func toggleText(enabled bool) string {
	if enabled {
		return goodStyle.Render("● ON ")
	}
	return mutedStyle.Render("○ OFF")
}

func (m Model) setupView() string {
	boxWidth := responsiveModalWidth(m.width, 56, 132)
	steps := []string{"WELCOME", "SYSTEM CHECK", "PREFERENCES", "ACTIVATE"}
	var progress []string
	for i, name := range steps {
		style := mutedStyle
		marker := "○ "
		if i < m.setupStep {
			style = goodStyle
			marker = "✓ "
		}
		if i == m.setupStep {
			style = logoStyle
			marker = "● "
		}
		progress = append(progress, style.Render(marker+name))
	}
	var body string
	switch m.setupStep {
	case 0:
		body = valueStyle.Render("Welcome to DNS Filter") + "\n\nThis guided setup starts the local DNS engine and dashboard, then safely routes Windows DNS through the filter. Your previous network configuration can be restored at any time."
	case 1:
		status := m.controller.Status()
		body = valueStyle.Render("System readiness") + "\n\n" + toggleText(status.DNSServer) + " DNS Filter service\n" + toggleText(status.Dashboard) + " Local dashboard\n" + toggleText(status.DNSRouted) + " Windows DNS route\n\n" + mutedStyle.Render("Administrator permission is required when Windows DNS and startup are configured.")
	case 2:
		body = valueStyle.Render("Startup preferences") + "\n\n" + toggleText(m.settings.AutoLaunch) + " Launch automatically with Windows       [A]\n" + toggleText(m.settings.StartMinimized) + " Start hidden in the system tray          [M]\n\nYou can change these options later from Console Settings."
	default:
		body = valueStyle.Render("Ready to activate protection") + "\n\nThe application will start its DNS and dashboard services, save your startup preferences, and point the active Windows network connection to 127.0.0.1.\n\n" + warnStyle.Render("If setup cannot finish, no completed setup state will be saved.")
	}
	footer := "Enter / →  Continue"
	body += "\n\nDashboard: " + m.controller.BrowserURL() + "\nFallback: " + m.controller.DashboardURL()
	if m.setupStep == 3 {
		footer = "Enter  Activate protection"
	}
	mainBody := lipgloss.NewStyle().Width(maxInt(32, boxWidth-6)).Render(body)
	if boxWidth >= 96 {
		leftWidth := boxWidth * 58 / 100
		summary := lipgloss.NewStyle().Background(colors.panelAlt).Padding(1, 2).Width(boxWidth - leftWidth - 10).Render(
			labelStyle.Render("SETUP WILL CONFIGURE") + "\n\n" +
				goodStyle.Render("● ") + "Local DNS engine\n" +
				goodStyle.Render("● ") + "Private web dashboard\n" +
				goodStyle.Render("● ") + "Windows DNS recovery\n" +
				goodStyle.Render("● ") + "Startup and tray behavior")
		mainBody = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(leftWidth).Render(body), "   ", summary)
	}
	box := panelStyle.Width(boxWidth).Padding(1, 2).Render(logoStyle.Render("◆ FIRST-RUN SETUP") + "\n\n" + strings.Join(progress, "  ") + "\n\n" + mainBody + "\n\n" + goodStyle.Render("[ "+footer+" ]") + "    " + mutedStyle.Render("← Back   Q Exit"))
	return strings.Repeat("\n", maxInt(1, (m.height-lipgloss.Height(box))/2)) + lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box)
}

func (m Model) settingsView() string {
	boxWidth := responsiveModalWidth(m.width, 64, 142)
	status := m.controller.Status()
	items := []struct {
		label, detail string
		enabled       bool
	}{
		{"DNS Filter service", "Hosts DNS filtering and the dashboard", status.Running},
		{"Filtering engine", "Allow traffic without closing the dashboard", status.Filtering},
		{"Windows DNS route", "Use 127.0.0.1 or restore normal DHCP DNS", status.DNSRouted},
		{"Launch with Windows", "Create an elevated login task", status.AutoLaunch},
		{"Start server on launch", "Bring protection online when this console starts", m.settings.StartServerOnLaunch},
		{"Start minimized", "Open directly in the tray", m.settings.StartMinimized},
		{"Close to tray", "Keep protection running when the console closes", m.settings.CloseToTray},
		{"Run setup again", "Re-check and repair the initial configuration", false},
	}
	var rows strings.Builder
	labelWidth := 26
	detailWidth := maxInt(22, boxWidth-labelWidth-20)
	for i, item := range items {
		marker := toggleText(item.enabled)
		if i == 7 {
			marker = logoStyle.Render("↻ RUN")
		}
		line := fmt.Sprintf("%-3s %s %s %s", fmt.Sprintf("%d", i+1), pad(item.label, labelWidth), pad(item.detail, detailWidth), marker)
		style := lipgloss.NewStyle().Foreground(colors.text)
		if i == m.settingCursor {
			style = selectedLine.Width(minInt(105, m.width-10))
		}
		rows.WriteString(style.Render(line) + "\n")
	}
	box := panelStyle.Width(boxWidth).Padding(1, 2).Render(logoStyle.Render("◆ SERVER & APPLICATION SETTINGS") + "\n" + mutedStyle.Render("Lifecycle controls live here. Detailed rules stay in the hosted dashboard.") + "\n\n" + rows.String() + "\n" + mutedStyle.Render("↑↓ Select   Enter/Click Change   S/Esc Close"))
	return strings.Repeat("\n", maxInt(1, (m.height-lipgloss.Height(box))/2)) + lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box)
}

func responsiveModalWidth(terminalWidth, minimum, maximum int) int {
	available := maxInt(24, terminalWidth-8)
	width := available * 90 / 100
	if width < minimum {
		width = minimum
	}
	if width > maximum {
		width = maximum
	}
	if width > available {
		width = available
	}
	return width
}
