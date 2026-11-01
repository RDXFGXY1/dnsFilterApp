<#
  Emergency DNS recovery for Windows 11.

  This script does not depend on dns-filter.exe. It stops DNS Filter temporarily,
  restores automatic DNS on connected adapters, flushes the resolver cache, and
  leaves application data and configuration untouched.
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Test-IsAdministrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-IsAdministrator)) {
    Write-Host 'Requesting Administrator permission for emergency DNS recovery...'
    Start-Process powershell -Verb RunAs -ArgumentList @(
        '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', ('"' + $PSCommandPath + '"')
    )
    exit 0
}

Write-Host ''
Write-Host 'DNS Filter - Emergency DNS Recovery' -ForegroundColor Cyan
Write-Host 'This restores normal automatic DNS without deleting your data.'
Write-Host ''

# Stop every supported startup path first. Otherwise DNS Route Watch could
# immediately point the adapters back to the unavailable local resolver.
if (Get-ScheduledTask -TaskName 'DNSFilter' -ErrorAction SilentlyContinue) {
    Stop-ScheduledTask -TaskName 'DNSFilter' -ErrorAction SilentlyContinue
    Disable-ScheduledTask -TaskName 'DNSFilter' -ErrorAction SilentlyContinue | Out-Null
}
if (Get-ScheduledTask -TaskName 'DNSFilterRecovery' -ErrorAction SilentlyContinue) {
    Stop-ScheduledTask -TaskName 'DNSFilterRecovery' -ErrorAction SilentlyContinue
    Disable-ScheduledTask -TaskName 'DNSFilterRecovery' -ErrorAction SilentlyContinue | Out-Null
}
foreach ($serviceName in @('DNS-Filter', 'DNSFilter')) {
    if (Get-Service -Name $serviceName -ErrorAction SilentlyContinue) {
        Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        Set-Service -Name $serviceName -StartupType Manual -ErrorAction SilentlyContinue
    }
}
Get-Process -Name 'dns-filter', 'dns-filter-dev' -ErrorAction SilentlyContinue |
    Stop-Process -Force -ErrorAction SilentlyContinue

$indexes = @(
    Get-NetIPConfiguration -ErrorAction Stop |
        Where-Object { $_.NetAdapter.Status -eq 'Up' -and ($_.IPv4DefaultGateway -or $_.IPv6DefaultGateway) } |
        ForEach-Object { $_.InterfaceIndex } |
        Sort-Object -Unique
)
if ($indexes.Count -eq 0) {
    throw 'No connected adapter with a default gateway was found. Connect to Wi-Fi/Ethernet and run this file again.'
}

foreach ($index in $indexes) {
    $alias = (Get-NetAdapter -InterfaceIndex $index -ErrorAction SilentlyContinue).Name
    Write-Host "Restoring automatic DNS on $alias (interface $index)..."
    Set-DnsClientServerAddress -InterfaceIndex $index -ResetServerAddresses -ErrorAction Stop
}

Clear-DnsClientCache -ErrorAction SilentlyContinue
& ipconfig /flushdns | Out-Null

$remainingLocal = @(
    foreach ($index in $indexes) {
        Get-DnsClientServerAddress -InterfaceIndex $index -ErrorAction SilentlyContinue |
            Where-Object { $_.ServerAddresses -contains '127.0.0.1' -or $_.ServerAddresses -contains '::1' }
    }
)
if ($remainingLocal.Count -gt 0) {
    throw 'Automatic DNS was requested, but one or more adapters still use localhost. Restart Windows or set that adapter DNS to Automatic manually.'
}

Write-Host ''
Write-Host 'Internet DNS has been restored to automatic configuration.' -ForegroundColor Green
Write-Host 'DNS Filter automatic startup was paused and user data was not deleted.'
Write-Host 'Run Install or Update DNS Filter again when you want to re-enable protection.'
Write-Host ''
Read-Host 'Press Enter to close'
