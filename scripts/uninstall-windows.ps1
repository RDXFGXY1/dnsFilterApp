<# Safe DNS Filter uninstaller. User state is preserved unless explicitly confirmed. #>
[CmdletBinding()]
param(
    [string]$InstallDir = (Join-Path $env:ProgramFiles 'DNS Filter'),
    [switch]$RemoveUserData,
    [string]$DataRemovalConfirmation = '',
    [switch]$PlanOnly
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Test-IsAdministrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Resolve-SafeInstallPath([string]$Path) {
    $full = [IO.Path]::GetFullPath($Path)
    $root = [IO.Path]::GetPathRoot($full)
    if ($full -eq $root -or $full.Length -lt 8) { throw "Unsafe installation path: $full" }
    return $full
}

function Stop-ProcessAtPath([string]$Path) {
    $expected = [IO.Path]::GetFullPath($Path)
    Get-Process -ErrorAction SilentlyContinue | Where-Object {
        try { $_.Path -and ([IO.Path]::GetFullPath($_.Path) -eq $expected) } catch { $false }
    } | Stop-Process -Force
}

function Remove-OwnedFirewallRules {
    $exact = @('DNSFilter-Bypass-UDP53', 'DNSFilter-Bypass-TCP53', 'DNS Filter - DNS', 'DNS Filter - Web')
    Get-NetFirewallRule -ErrorAction SilentlyContinue | Where-Object {
        $_.Name -in $exact -or $_.Name -like 'DNSFilter-App-*' -or $_.Name -like 'DNSFilter-Schedule-*' -or $_.Name -like 'DNSFilter-Quarantine-*'
    } | Remove-NetFirewallRule -ErrorAction Continue
}

function Restore-AutomaticDNS {
    $routes = @(Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue | Sort-Object RouteMetric, InterfaceMetric)
    if ($routes.Count -eq 0) { throw 'No active default-route adapter was found; reset DNS manually.' }
    $indexes = @($routes | Select-Object -ExpandProperty InterfaceIndex -Unique)
    foreach ($index in $indexes) {
        Set-DnsClientServerAddress -InterfaceIndex $index -ResetServerAddresses -ErrorAction Stop
    }
    & ipconfig /flushdns | Out-Null
}

function Remove-OwnedRootCertificate([string]$CertificatePath) {
    if (-not (Test-Path -LiteralPath $CertificatePath -PathType Leaf)) { return }
    $certificate = [Security.Cryptography.X509Certificates.X509Certificate2]::new($CertificatePath)
    $thumbprint = $certificate.Thumbprint
    Get-ChildItem Cert:\LocalMachine\Root | Where-Object Thumbprint -eq $thumbprint | Remove-Item -Force
}

$install = Resolve-SafeInstallPath $InstallDir
if ($RemoveUserData -and $DataRemovalConfirmation -cne 'DELETE LOCAL DNS FILTER DATA') {
    throw 'User-data removal requires -DataRemovalConfirmation "DELETE LOCAL DNS FILTER DATA".'
}

$plan = [ordered]@{
    InstallDir = $install
    StopExecutables = @((Join-Path $install 'dns-filter.exe'), (Join-Path $install 'desktop\DNSFilter.exe'))
    RestoreAutomaticDNS = $true
    ScheduledTask = 'DNSFilter'
    FirewallRules = @('DNSFilter-Bypass-*', 'DNSFilter-App-*', 'DNSFilter-Schedule-*', 'DNSFilter-Quarantine-*', 'DNS Filter - DNS', 'DNS Filter - Web')
    RemoveMatchingLocalCA = Test-Path -LiteralPath (Join-Path $install 'data\ca\ca.crt')
    Preserve = $(if ($RemoveUserData) { @() } else { @('configs', 'data') })
    RemoveUserData = [bool]$RemoveUserData
}
if ($PlanOnly) { [pscustomobject]$plan; exit 0 }
if (-not (Test-IsAdministrator)) { throw 'Run the uninstaller from an Administrator PowerShell window.' }
if (-not (Test-Path -LiteralPath $install -PathType Container)) { throw "Installation directory not found: $install" }

$logDir = Join-Path $install 'data\logs'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$logFile = Join-Path $logDir 'uninstall.log'
function Log([string]$Message) {
    Write-Host $Message
    Add-Content -LiteralPath $logFile -Value ("{0:o} {1}" -f [DateTimeOffset]::Now, $Message) -Encoding UTF8
}

Log 'Starting safe uninstall.'
Stop-ProcessAtPath (Join-Path $install 'desktop\DNSFilter.exe')
Stop-ProcessAtPath (Join-Path $install 'dns-filter.exe')

foreach ($serviceName in @('DNS-Filter', 'DNSFilter')) {
    $service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
    if ($service) {
        Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        & sc.exe delete $serviceName | Out-Null
    }
}
if (Get-ScheduledTask -TaskName 'DNSFilter' -ErrorAction SilentlyContinue) {
    Unregister-ScheduledTask -TaskName 'DNSFilter' -Confirm:$false
}

Restore-AutomaticDNS
Log 'System DNS restored to automatic configuration.'
Remove-OwnedFirewallRules
Log 'DNS Filter-owned firewall rules removed.'
Remove-OwnedRootCertificate (Join-Path $install 'data\ca\ca.crt')
Log 'Matching DNS Filter local CA certificate removed when present.'

$immutable = @(
    'dns-filter.exe', 'web', 'desktop', '.rollback', '.installed-version',
    'update-windows.ps1', 'uninstall-windows.ps1', 'emergency-restore-dns.ps1',
    'Install or Update DNS Filter.bat', 'Uninstall DNS Filter.bat',
    '1. Start DNS Filter.bat', '2. Enable Protection.bat', '3. Disable Protection.bat',
    'EMERGENCY - Restore Normal DNS.bat', 'README.txt', 'RUNNING.md', 'VERSION'
)
foreach ($relative in $immutable) {
    $target = Join-Path $install $relative
    if (Test-Path -LiteralPath $target) { Remove-Item -LiteralPath $target -Recurse -Force }
}

if ($RemoveUserData) {
    foreach ($relative in @('configs', 'data')) {
        $target = Join-Path $install $relative
        if (Test-Path -LiteralPath $target) { Remove-Item -LiteralPath $target -Recurse -Force }
    }
    Write-Host 'DNS Filter and explicitly confirmed local user data were removed.'
} else {
    Log 'Application removed. configs and data were preserved.'
    Write-Host "DNS Filter was removed. Local data remains in: $install"
}
