//go:build windows

// Package autostart manages Windows background startup and independent DNS recovery.
package autostart

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
)

const taskName = "DNSFilter"
const recoveryTaskName = "DNSFilterRecovery"

func Supported() bool             { return true }
func IsEnabled() bool             { return Inspect().Enabled }
func psQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
func encodedPowerShell(script string) string {
	encoded := utf16.Encode([]rune(script))
	raw := make([]byte, len(encoded)*2)
	for i, v := range encoded {
		raw[i*2] = byte(v)
		raw[i*2+1] = byte(v >> 8)
	}
	return base64.StdEncoding.EncodeToString(raw)
}
func powershell(script string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodedPowerShell("$ProgressPreference = 'SilentlyContinue'\n[Console]::OutputEncoding = [Text.Encoding]::UTF8\n"+script))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	// Encoded PowerShell can write '#< CLIXML' progress records to stderr even
	// on success. Only stdout is the JSON protocol; keep stderr for failures.
	out, e := cmd.Output()
	if e != nil {
		var exitError *exec.ExitError
		detail := out
		if errors.As(e, &exitError) && len(exitError.Stderr) > 0 {
			detail = exitError.Stderr
		}
		return nil, fmt.Errorf("manage Windows startup: %w: %s", e, strings.TrimSpace(string(detail)))
	}
	return out, nil
}
func serviceWorkingDirectory(exe string) string {
	dir := filepath.Dir(exe)
	// Covers installed/portable layouts and nested repository builds such as build/dev.
	for i := 0; i < 5; i++ {
		if _, e := os.Stat(filepath.Join(dir, "configs", "config.yaml")); e == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Dir(exe)
}
func taskRegistrationScript(exe string) string {
	return configurationScript(exe, Options{Mode: ModeBoot, RecoverDNS: true})
}
func configurationScript(exe string, options Options) string {
	dir := serviceWorkingDirectory(exe)
	script := `$ErrorActionPreference = 'Stop'
$settings = New-ScheduledTaskSettingsSet -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1) -StartWhenAvailable -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew
$settings.DisallowStartIfOnBatteries = $false
$settings.StopIfGoingOnBatteries = $false
`
	if options.Mode == ModeManual {
		script += `if (Get-ScheduledTask -TaskName 'DNSFilter' -ErrorAction SilentlyContinue) { Unregister-ScheduledTask -TaskName 'DNSFilter' -Confirm:$false }
`
	} else {
		trigger := `@((New-ScheduledTaskTrigger -AtStartup), (New-ScheduledTaskTrigger -AtLogOn))`
		if options.Mode == ModeLogin {
			trigger = `@((New-ScheduledTaskTrigger -AtLogOn))`
		}
		script += `$action = New-ScheduledTaskAction -Execute ` + psQuote(exe) + ` -WorkingDirectory ` + psQuote(dir) + `
$triggers = ` + trigger + `
Register-ScheduledTask -TaskName 'DNSFilter' -Action $action -Trigger $triggers -Settings $settings -User 'SYSTEM' -RunLevel Highest -Force | Out-Null
`
	}
	if options.RecoverDNS {
		script += `$recoveryAction = New-ScheduledTaskAction -Execute ` + psQuote(exe) + ` -Argument 'recover-dns' -WorkingDirectory ` + psQuote(dir) + `
$repeat = New-ScheduledTaskTrigger -Once -At (Get-Date).AddMinutes(1) -RepetitionInterval (New-TimeSpan -Minutes 1)
$recoveryTriggers = @((New-ScheduledTaskTrigger -AtStartup), $repeat)
$recoverySettings = New-ScheduledTaskSettingsSet -StartWhenAvailable -ExecutionTimeLimit (New-TimeSpan -Minutes 1) -MultipleInstances IgnoreNew
$recoverySettings.DisallowStartIfOnBatteries = $false
$recoverySettings.StopIfGoingOnBatteries = $false
Register-ScheduledTask -TaskName 'DNSFilterRecovery' -Action $recoveryAction -Trigger $recoveryTriggers -Settings $recoverySettings -User 'SYSTEM' -RunLevel Highest -Force | Out-Null
`
	} else {
		script += `if (Get-ScheduledTask -TaskName 'DNSFilterRecovery' -ErrorAction SilentlyContinue) { Unregister-ScheduledTask -TaskName 'DNSFilterRecovery' -Confirm:$false }
`
	}
	return script
}
func Configure(options Options) error {
	if e := options.Validate(); e != nil {
		return e
	}
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	dir := serviceWorkingDirectory(exe)
	if options.Mode != ModeManual || options.RecoverDNS {
		if _, e = os.Stat(filepath.Join(dir, "configs", "config.yaml")); e != nil {
			return fmt.Errorf("cannot find configs/config.yaml for %s; install or build the application in its normal folder", exe)
		}
	}
	_, e = powershell(configurationScript(exe, options))
	return e
}
func Enable() error  { return Configure(Options{Mode: ModeBoot, RecoverDNS: true}) }
func Disable() error { return Configure(Options{Mode: ModeManual}) }
func Inspect() Status {
	state := Status{Supported: true, Mode: "unknown", RecoverySupported: true, LoginSupported: true}
	out, e := powershell(`$ErrorActionPreference = 'Stop'
$t = Get-ScheduledTask -TaskName 'DNSFilter' -ErrorAction SilentlyContinue
$r = Get-ScheduledTask -TaskName 'DNSFilterRecovery' -ErrorAction SilentlyContinue
$mode = 'manual'
$enabled = $false
$exe = ''
$dir = ''
$result = 0
$problem = ''
if ($t) {
 $enabled = [bool]$t.Settings.Enabled
 if ($enabled) {
  $mode = 'unknown'
  foreach ($trigger in $t.Triggers) {
   if ($trigger.Enabled -eq $false) { continue }
   if ($trigger.CimClass.CimClassName -eq 'MSFT_TaskBootTrigger') { $mode = 'boot'; break }
   if ($trigger.CimClass.CimClassName -eq 'MSFT_TaskLogonTrigger') { $mode = 'login' }
  }
 }
 $exe = [string]$t.Actions[0].Execute
 $dir = [string]$t.Actions[0].WorkingDirectory
 $info = Get-ScheduledTaskInfo -TaskName 'DNSFilter'
 $result = [uint32]$info.LastTaskResult
 if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) { $problem = 'The startup executable is missing. Save these settings again to repair the task.' }
 elseif (-not (Test-Path -LiteralPath (Join-Path $dir 'configs\config.yaml') -PathType Leaf)) { $problem = 'The startup working folder is invalid. Save these settings again to repair the task.' }
}
[pscustomobject]@{supported=$true;enabled=$enabled;mode=$mode;recover_dns=([bool]$r -and [bool]$r.Settings.Enabled);recovery_supported=$true;login_supported=$true;executable=$exe;working_directory=$dir;last_result=$result;problem=$problem} | ConvertTo-Json -Compress`)
	if e != nil {
		state.Problem = e.Error()
		return state
	}
	if e = json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(string(out)), "\ufeff")), &state); e != nil {
		state.Problem = "Could not read Windows startup configuration: " + e.Error()
		state.Mode = "unknown"
		return state
	}
	state.ReadOK = true
	return state
}
