//go:build windows

package dnsbypass

import (
	"fmt"
	"os/exec"
	"strings"
)

type windowsPlatform struct{}

func newPlatform() Platform             { return windowsPlatform{} }
func (windowsPlatform) Supported() bool { return true }

func dnsFirewall(script string) error {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		msg = err.Error()
	}
	return fmt.Errorf("Windows Firewall: %s", msg)
}
func (windowsPlatform) Enable() error {
	return dnsFirewall(`$ErrorActionPreference='Stop'
$rules=@(@{Name='DNSFilter-Bypass-UDP53';Protocol='UDP'},@{Name='DNSFilter-Bypass-TCP53';Protocol='TCP'})
foreach($r in $rules){Get-NetFirewallRule -Name $r.Name -ErrorAction SilentlyContinue|Remove-NetFirewallRule -ErrorAction Stop;New-NetFirewallRule -Name $r.Name -DisplayName $r.Name -Direction Outbound -Action Block -Protocol $r.Protocol -RemotePort 53 -RemoteAddress Internet -Profile Any -Enabled True -ErrorAction Stop|Out-Null}`)
}
func (windowsPlatform) Disable() error {
	return dnsFirewall(`$ErrorActionPreference='Stop';@('DNSFilter-Bypass-UDP53','DNSFilter-Bypass-TCP53')|ForEach-Object{Get-NetFirewallRule -Name $_ -ErrorAction SilentlyContinue|Remove-NetFirewallRule -ErrorAction Stop}`)
}
func (windowsPlatform) Active() (bool, error) {
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", `$n=@('DNSFilter-Bypass-UDP53','DNSFilter-Bypass-TCP53');$found=@(Get-NetFirewallRule -Name $n -ErrorAction SilentlyContinue|Where-Object Enabled -eq 'True');if($found.Count -eq 2){'true'}else{'false'}`).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("Windows Firewall: %s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)) == "true", nil
}
