package main

import (
	"github.com/charmbracelet/lipgloss"
)

var colors = struct {
	bg, panel, panelAlt, border, text, muted, accent, accent2, good, warn, bad lipgloss.Color
}{
	"#0B1017", "#111923", "#17212D", "#263445", "#E8EEF5", "#8391A2",
	"#52D6C7", "#6CA8FF", "#70E1A1", "#F2C66D", "#FF7B83",
}

var (
	logoStyle    = lipgloss.NewStyle().Foreground(colors.accent).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(colors.muted)
	labelStyle   = lipgloss.NewStyle().Foreground(colors.muted).Bold(true)
	valueStyle   = lipgloss.NewStyle().Foreground(colors.text).Bold(true)
	panelStyle   = lipgloss.NewStyle().Background(colors.panel).Border(lipgloss.RoundedBorder()).BorderForeground(colors.border).Padding(0, 1)
	activeNav    = lipgloss.NewStyle().Background(colors.accent).Foreground(colors.bg).Bold(true).Padding(0, 1)
	inactiveNav  = lipgloss.NewStyle().Foreground(colors.muted).Padding(0, 1)
	goodStyle    = lipgloss.NewStyle().Foreground(colors.good).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(colors.warn).Bold(true)
	badStyle     = lipgloss.NewStyle().Foreground(colors.bad).Bold(true)
	selectedLine = lipgloss.NewStyle().Background(colors.panelAlt).Foreground(colors.text).Bold(true)
)
