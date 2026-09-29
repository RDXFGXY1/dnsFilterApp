package main

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// Windows Terminal does not consistently emit a resize event when its window
// is maximized. Polling the actual console viewport keeps Bubble Tea's model in
// sync with the visible terminal instead of rendering against a stale size.
func pollTerminalSize() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
		width, height, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || width < 1 || height < 1 {
			return terminalSizeMsg{}
		}
		return terminalSizeMsg{width: width, height: height}
	})
}
