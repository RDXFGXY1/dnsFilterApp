package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/getlantern/systray"
)

func runTray(controller *ServiceController, program *tea.Program) {
	systray.Run(func() {
		systray.SetIcon(trayIcon())
		systray.SetTitle("DNS Filter")
		systray.SetTooltip("DNS Filter protection console")
		statusItem := systray.AddMenuItem("Protection status", "Current local protection state")
		statusItem.Disable()
		showItem := systray.AddMenuItem("Open console", "Show the terminal control center")
		dashboardItem := systray.AddMenuItem("Open dashboard", "Configure DNS Filter in the browser")
		systray.AddSeparator()
		serviceItem := systray.AddMenuItem("Stop service", "Start or stop DNS and dashboard services")
		filterItem := systray.AddMenuItem("Pause filtering", "Allow traffic without stopping the dashboard")
		restoreItem := systray.AddMenuItem("Restore normal Windows DNS", "Return the active adapter to DHCP DNS")
		systray.AddSeparator()
		exitItem := systray.AddMenuItem("Exit DNS Filter", "Stop protection, restore DNS, and close")
		refreshTitles := func() {
			status := controller.Status()
			if status.Running && status.DNSServer {
				statusItem.SetTitle("Protected — service running")
				serviceItem.SetTitle("Stop service")
			} else {
				statusItem.SetTitle("Protection stopped")
				serviceItem.SetTitle("Start service")
			}
			if status.Filtering {
				filterItem.SetTitle("Pause filtering")
			} else {
				filterItem.SetTitle("Resume filtering")
			}
		}
		refreshTitles()
		go func() {
			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				refreshTitles()
			}
		}()
		go func() {
			for range showItem.ClickedCh {
				showConsoleWindow()
			}
		}()
		go func() {
			for range dashboardItem.ClickedCh {
				_ = openURL(controller.BrowserURL())
			}
		}()
		go func() {
			for range serviceItem.ClickedCh {
				status := controller.Status()
				if status.Running {
					_ = controller.Stop(true)
				} else {
					_ = controller.Start()
				}
				refreshTitles()
			}
		}()
		go func() {
			for range filterItem.ClickedCh {
				status := controller.Status()
				controller.SetFiltering(!status.Filtering)
				refreshTitles()
			}
		}()
		go func() {
			for range restoreItem.ClickedCh {
				_ = controller.RestoreDNS()
				refreshTitles()
			}
		}()
		go func() {
			<-exitItem.ClickedCh
			if err := controller.Stop(true); err != nil {
				statusItem.SetTitle("Exit blocked — Windows DNS could not be restored")
				showConsoleWindow()
				return
			}
			program.Quit()
			systray.Quit()
		}()
	}, func() {})
}
