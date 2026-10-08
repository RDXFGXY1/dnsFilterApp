//go:build windows

package appnetwork

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type windowsPlatform struct{}

func validatePathPlatform(path string) error {
	if !strings.EqualFold(filepath.Ext(path), ".exe") {
		return fmt.Errorf("select a valid Windows .exe file")
	}
	base := strings.ToLower(filepath.Base(path))
	critical := map[string]bool{"svchost.exe": true, "lsass.exe": true, "services.exe": true, "wininit.exe": true, "csrss.exe": true}
	if critical[base] {
		return fmt.Errorf("this critical Windows process cannot be blocked")
	}
	return nil
}

func newPlatform() platform             { return windowsPlatform{} }
func (windowsPlatform) Supported() bool { return true }

func run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("Windows Firewall: %s", message)
	}
	return nil
}

func runFirewallScript(script, ruleName, programPath string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "DNSFILTER_RULE_NAME="+ruleName, "DNSFILTER_PROGRAM_PATH="+programPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(out))
	if marker := strings.Index(message, "At line:"); marker >= 0 {
		message = strings.TrimSpace(message[:marker])
	}
	if message == "" {
		message = err.Error()
	}
	return fmt.Errorf("Windows Firewall: %s", message)
}

func (windowsPlatform) Block(path, name string) error {
	script := `$ErrorActionPreference='Stop'
$ruleName=$env:DNSFILTER_RULE_NAME
$programPath=$env:DNSFILTER_PROGRAM_PATH
if([string]::IsNullOrWhiteSpace($ruleName)){throw 'Firewall rule name is missing'}
if(-not (Test-Path -LiteralPath $programPath -PathType Leaf)){throw 'Application executable was not found'}
Get-NetFirewallRule -Name $ruleName -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction Stop
New-NetFirewallRule -Name $ruleName -DisplayName $ruleName -Direction Outbound -Action Block -Program $programPath -Profile Any -Enabled True -ErrorAction Stop | Out-Null`
	return runFirewallScript(script, name, path)
}

func (windowsPlatform) Allow(name string) error {
	script := `$ErrorActionPreference='Stop'
$ruleName=$env:DNSFILTER_RULE_NAME
if([string]::IsNullOrWhiteSpace($ruleName)){throw 'Firewall rule name is missing'}
Get-NetFirewallRule -Name $ruleName -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction Stop`
	return runFirewallScript(script, name, "")
}

func (windowsPlatform) RunningApps() ([]App, error) {
	script := `Get-CimInstance Win32_Process | Where-Object {$_.ExecutablePath -and $_.ExecutablePath.EndsWith('.exe')} | Select-Object Name,ExecutablePath,ProcessId | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("could not read running applications: %s", strings.TrimSpace(string(out)))
	}
	if strings.TrimSpace(string(out)) == "" {
		return []App{}, nil
	}
	var raw json.RawMessage = out
	var list []struct {
		Name           string
		ExecutablePath string
		ProcessID      int
	}
	if len(raw) > 0 && raw[0] == '{' {
		var one struct {
			Name           string
			ExecutablePath string
			ProcessID      int
		}
		if err := json.Unmarshal(raw, &one); err != nil {
			return nil, err
		}
		list = append(list, one)
	} else if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	apps := make([]App, 0, len(list))
	seen := map[string]bool{}
	for _, item := range list {
		key := strings.ToLower(item.ExecutablePath)
		if seen[key] {
			continue
		}
		seen[key] = true
		apps = append(apps, App{Name: item.Name, Path: item.ExecutablePath, PID: item.ProcessID, Running: true})
	}
	return apps, nil
}
