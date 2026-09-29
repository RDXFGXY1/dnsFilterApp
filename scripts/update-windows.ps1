<#
Safely installs or updates DNS Filter on Windows.

Mutable machine state is never replaced during an update:
  configs\, data\, logs, database, Vault, recovery state, and certificates.

The executable and web assets are staged, the previous version is retained under
.rollback, and a failed health check automatically restores the previous version.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$PackageDir,
    [string]$InstallDir = (Join-Path $env:ProgramFiles 'DNS Filter'),
    [string]$HealthUrl = 'http://127.0.0.1:8080/healthz',
    [ValidateRange(5, 120)]
    [int]$HealthTimeoutSeconds = 30,
    [switch]$ValidateOnly
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Resolve-FullPath([string]$Path, [bool]$MustExist) {
    if ($MustExist) {
        return (Resolve-Path -LiteralPath $Path -ErrorAction Stop).Path
    }
    return [IO.Path]::GetFullPath($Path)
}

function Test-IsAdministrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Assert-SafeInstallPath([string]$Path) {
    $root = [IO.Path]::GetPathRoot($Path)
    if ([string]::IsNullOrWhiteSpace($Path) -or $Path -eq $root -or $Path.Length -lt 8) {
        throw "Unsafe installation path: $Path"
    }
}

function Write-UpdateLog([string]$Message) {
    $line = "{0:o} {1}" -f [DateTimeOffset]::Now, $Message
    Write-Host $Message
    if ($script:logFile) { Add-Content -LiteralPath $script:logFile -Value $line -Encoding UTF8 }
}

function Stop-InstalledProcess([string]$ExecutablePath) {
    $expected = [IO.Path]::GetFullPath($ExecutablePath)
    Get-Process -ErrorAction SilentlyContinue | Where-Object {
        try { $_.Path -and ([IO.Path]::GetFullPath($_.Path) -eq $expected) } catch { $false }
    } | Stop-Process -Force
    Start-Sleep -Milliseconds 700
}

function Wait-ServiceReady([string]$Url, [int]$TimeoutSeconds, [string]$ExpectedVersion) {
    $deadline = [DateTimeOffset]::Now.AddSeconds($TimeoutSeconds)
    while ([DateTimeOffset]::Now -lt $deadline) {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Uri $Url -MaximumRedirection 0 -SkipHttpErrorCheck -TimeoutSec 2
            if ([int]$response.StatusCode -eq 200) {
                $health = $response.Content | ConvertFrom-Json
                if ($health.service -eq 'dns-filter' -and $health.ready -eq $true -and $health.version -eq $ExpectedVersion) { return $true }
            }
        } catch {}
        Start-Sleep -Milliseconds 500
    }
    return $false
}

$package = Resolve-FullPath $PackageDir $true
$install = Resolve-FullPath $InstallDir $false
Assert-SafeInstallPath $install

$packageExe = Join-Path $package 'dns-filter.exe'
$packageWeb = Join-Path $package 'web'
$packageConfig = Join-Path $package 'configs'
$packageVersionFile = Join-Path $package 'VERSION'
if (-not (Test-Path -LiteralPath $packageExe -PathType Leaf)) { throw "Package is missing dns-filter.exe: $package" }
if (-not (Test-Path -LiteralPath (Join-Path $packageWeb 'templates') -PathType Container)) { throw 'Package is missing web templates.' }
if (-not (Test-Path -LiteralPath (Join-Path $packageWeb 'static') -PathType Container)) { throw 'Package is missing web static assets.' }
if (-not (Test-Path -LiteralPath $packageVersionFile -PathType Leaf)) { throw 'Package is missing its VERSION identity file.' }
$expectedVersion = (Get-Content -LiteralPath $packageVersionFile -Raw).Trim()
if ([string]::IsNullOrWhiteSpace($expectedVersion)) { throw 'Package VERSION is empty.' }
if ($package -eq $install -or $package.StartsWith($install + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'The update package must be outside the installation directory.'
}

if ($ValidateOnly) {
    [pscustomobject]@{ Valid = $true; Version = $expectedVersion; Package = $package; Install = $install; Preserved = @('configs', 'data') }
    exit 0
}
if (-not (Test-IsAdministrator)) {
    throw 'Run this installer from an Administrator PowerShell window.'
}

New-Item -ItemType Directory -Force -Path $install | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $install 'data\logs') | Out-Null
$script:logFile = Join-Path $install 'data\logs\update.log'

$lockPath = Join-Path $install '.update.lock'
$lock = $null
$staging = Join-Path ([IO.Path]::GetTempPath()) ('dns-filter-update-' + [guid]::NewGuid().ToString('N'))
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$rollback = Join-Path $install ('.rollback\' + $stamp)
$targetExe = Join-Path $install 'dns-filter.exe'
$targetWeb = Join-Path $install 'web'
$wasUpgrade = Test-Path -LiteralPath $targetExe -PathType Leaf
$newStarted = $false
$maintenanceFiles = @('update-windows.ps1', 'uninstall-windows.ps1', 'Uninstall DNS Filter.bat')

try {
    $lock = [IO.File]::Open($lockPath, [IO.FileMode]::OpenOrCreate, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
    New-Item -ItemType Directory -Force -Path $staging | Out-Null
    Copy-Item -LiteralPath $packageExe -Destination (Join-Path $staging 'dns-filter.exe') -Force
    Copy-Item -LiteralPath $packageWeb -Destination (Join-Path $staging 'web') -Recurse -Force
    if (Test-Path -LiteralPath (Join-Path $package 'desktop')) {
        Copy-Item -LiteralPath (Join-Path $package 'desktop') -Destination (Join-Path $staging 'desktop') -Recurse -Force
    }
    foreach ($name in $maintenanceFiles) {
        $source = Join-Path $package $name
        if (Test-Path -LiteralPath $source -PathType Leaf) { Copy-Item -LiteralPath $source -Destination (Join-Path $staging $name) -Force }
    }

    Write-UpdateLog 'Package staged and validated.'
    Stop-InstalledProcess $targetExe
    New-Item -ItemType Directory -Force -Path $rollback | Out-Null
    if (Test-Path -LiteralPath $targetExe) { Move-Item -LiteralPath $targetExe -Destination (Join-Path $rollback 'dns-filter.exe') }
    if (Test-Path -LiteralPath $targetWeb) { Move-Item -LiteralPath $targetWeb -Destination (Join-Path $rollback 'web') }
    if (Test-Path -LiteralPath (Join-Path $install 'desktop')) { Move-Item -LiteralPath (Join-Path $install 'desktop') -Destination (Join-Path $rollback 'desktop') }
    foreach ($name in $maintenanceFiles) {
        $target = Join-Path $install $name
        if (Test-Path -LiteralPath $target) { Move-Item -LiteralPath $target -Destination (Join-Path $rollback $name) }
    }

    Move-Item -LiteralPath (Join-Path $staging 'dns-filter.exe') -Destination $targetExe
    Move-Item -LiteralPath (Join-Path $staging 'web') -Destination $targetWeb
    if (Test-Path -LiteralPath (Join-Path $staging 'desktop')) { Move-Item -LiteralPath (Join-Path $staging 'desktop') -Destination (Join-Path $install 'desktop') }
    foreach ($name in $maintenanceFiles) {
        $stagedFile = Join-Path $staging $name
        if (Test-Path -LiteralPath $stagedFile) { Move-Item -LiteralPath $stagedFile -Destination (Join-Path $install $name) }
    }

    if (-not (Test-Path -LiteralPath (Join-Path $install 'configs')) -and (Test-Path -LiteralPath $packageConfig)) {
        Copy-Item -LiteralPath $packageConfig -Destination (Join-Path $install 'configs') -Recurse
        Write-UpdateLog 'Fresh sanitized configuration installed.'
    } else {
        Write-UpdateLog 'Existing configuration and machine data preserved.'
    }

    Start-Process -FilePath $targetExe -WorkingDirectory $install -WindowStyle Hidden
    $newStarted = $true
    if (-not (Wait-ServiceReady $HealthUrl $HealthTimeoutSeconds $expectedVersion)) { throw "The updated service failed its identity and health check for version $expectedVersion." }

    Set-Content -LiteralPath (Join-Path $install '.installed-version') -Value $expectedVersion -Encoding ASCII
    Write-UpdateLog ($(if ($wasUpgrade) { 'Update completed successfully.' } else { 'Installation completed successfully.' }))
    Write-Output ([pscustomobject]@{ Success = $true; Updated = $wasUpgrade; InstallDir = $install; Rollback = $rollback })
} catch {
    $failure = $_.Exception.Message
    Write-UpdateLog "Update failed: $failure"
    if ($newStarted) { Stop-InstalledProcess $targetExe }
    if ($wasUpgrade -and (Test-Path -LiteralPath (Join-Path $rollback 'dns-filter.exe'))) {
        if (Test-Path -LiteralPath $targetExe) { Remove-Item -LiteralPath $targetExe -Force }
        if (Test-Path -LiteralPath $targetWeb) { Remove-Item -LiteralPath $targetWeb -Recurse -Force }
        if (Test-Path -LiteralPath (Join-Path $install 'desktop')) { Remove-Item -LiteralPath (Join-Path $install 'desktop') -Recurse -Force }
        foreach ($name in $maintenanceFiles) { $target = Join-Path $install $name; if (Test-Path -LiteralPath $target) { Remove-Item -LiteralPath $target -Force } }
        Move-Item -LiteralPath (Join-Path $rollback 'dns-filter.exe') -Destination $targetExe
        if (Test-Path -LiteralPath (Join-Path $rollback 'web')) { Move-Item -LiteralPath (Join-Path $rollback 'web') -Destination $targetWeb }
        if (Test-Path -LiteralPath (Join-Path $rollback 'desktop')) { Move-Item -LiteralPath (Join-Path $rollback 'desktop') -Destination (Join-Path $install 'desktop') }
        foreach ($name in $maintenanceFiles) { $old = Join-Path $rollback $name; if (Test-Path -LiteralPath $old) { Move-Item -LiteralPath $old -Destination (Join-Path $install $name) } }
        Start-Process -FilePath $targetExe -WorkingDirectory $install -WindowStyle Hidden
        Write-UpdateLog 'Previous executable and web assets were restored automatically.'
    } elseif (-not $wasUpgrade) {
        if (Test-Path -LiteralPath $targetExe) { Remove-Item -LiteralPath $targetExe -Force }
        if (Test-Path -LiteralPath $targetWeb) { Remove-Item -LiteralPath $targetWeb -Recurse -Force }
        if (Test-Path -LiteralPath (Join-Path $install 'desktop')) { Remove-Item -LiteralPath (Join-Path $install 'desktop') -Recurse -Force }
        foreach ($name in $maintenanceFiles) { $target = Join-Path $install $name; if (Test-Path -LiteralPath $target) { Remove-Item -LiteralPath $target -Force } }
        Write-UpdateLog 'Failed fresh-install binaries were removed; configuration and data were preserved.'
    }
    throw
} finally {
    if ($lock) { $lock.Dispose() }
    if (Test-Path -LiteralPath $lockPath) { Remove-Item -LiteralPath $lockPath -Force -ErrorAction SilentlyContinue }
    if (Test-Path -LiteralPath $staging) { Remove-Item -LiteralPath $staging -Recurse -Force -ErrorAction SilentlyContinue }
}
