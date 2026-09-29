package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type ConsoleSettings struct {
	SetupComplete       bool `json:"setup_complete"`
	StartServerOnLaunch bool `json:"start_server_on_launch"`
	AutoLaunch          bool `json:"auto_launch"`
	StartMinimized      bool `json:"start_minimized"`
	CloseToTray         bool `json:"close_to_tray"`
	DNSManaged          bool `json:"dns_managed"`
	LastCleanExit       bool `json:"last_clean_exit"`
}

func defaultConsoleSettings() ConsoleSettings {
	return ConsoleSettings{StartServerOnLaunch: true, CloseToTray: true}
}
func consoleSettingsPath(root string) string {
	return filepath.Join(root, "data", "console-settings.json")
}
func loadConsoleSettings(root string) (ConsoleSettings, error) {
	settings := defaultConsoleSettings()
	data, err := os.ReadFile(consoleSettingsPath(root))
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	return settings, json.Unmarshal(data, &settings)
}
func saveConsoleSettings(root string, settings ConsoleSettings) error {
	path := consoleSettingsPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
