package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/getlantern/systray"
)

func main() {
	controller, err := NewServiceController()
	if err != nil {
		fmt.Fprintf(os.Stderr, "DNS Filter could not locate its application files: %v\n", err)
		os.Exit(1)
	}
	settings, _ := loadConsoleSettings(controller.Root())
	apiURL := controller.DashboardURL()
	if len(os.Args) > 1 {
		apiURL = os.Args[1]
	}
	var startErr error
	if settings.StartServerOnLaunch {
		startErr = controller.Start()
	}
	if startErr != nil && settings.DNSManaged {
		_ = controller.RestoreDNS()
		settings.DNSManaged = false
		_ = saveConsoleSettings(controller.Root(), settings)
	}
	settings.LastCleanExit = false
	_ = saveConsoleSettings(controller.Root(), settings)
	p := tea.NewProgram(NewModel(apiURL, controller, settings, startErr), tea.WithAltScreen(), tea.WithMouseCellMotion())
	errCh := make(chan error, 1)
	go func() { errCh <- p.Start(); systray.Quit() }()
	if settings.StartMinimized {
		hideConsoleWindow()
	}
	runTray(controller, p)
	if err := <-errCh; err != nil {
		fmt.Fprintf(os.Stderr, "DNS Filter console failed: %v\n", err)
		os.Exit(1)
	}
	_ = controller.Stop(true)
	latest, _ := loadConsoleSettings(controller.Root())
	latest.DNSManaged = false
	latest.LastCleanExit = true
	_ = saveConsoleSettings(controller.Root(), latest)
}
