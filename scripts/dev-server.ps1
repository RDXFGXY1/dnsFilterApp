<#
  Development runner for DNS Filter.

  It builds and runs directly from the repository. It does not create a portable
  package and it does not copy files into Program Files.

  Examples:
    .\scripts\dev-server.ps1 restart
    .\scripts\dev-server.ps1 restart -NoBuild
    .\scripts\dev-server.ps1 stop
    .\scripts\dev-server.ps1 status
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('start', 'restart', 'stop', 'status')]
    [string]$Action = 'restart',
    [switch]$NoBuild
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$buildDir = Join-Path $root 'build\dev'
$exe = Join-Path $buildDir 'dns-filter-dev.exe'
$pidFile = Join-Path $root 'data\dns-filter-dev.pid'
$logDir = Join-Path $root 'data\logs'
$stdoutLog = Join-Path $logDir 'dev-server.stdout.log'
$stderrLog = Join-Path $logDir 'dev-server.stderr.log'
$healthURL = 'http://127.0.0.1:8080/healthz'

function Test-Administrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Restart-Elevated {
    $arguments = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', ('"' + $PSCommandPath + '"'), $Action)
    if ($NoBuild) { $arguments += '-NoBuild' }
    Start-Process powershell -Verb RunAs -WindowStyle Hidden -ArgumentList $arguments -WorkingDirectory $root
}

function Get-DevProcess {
    if (Test-Path -LiteralPath $pidFile) {
        $devPID = 0
        if ([int]::TryParse((Get-Content -Raw -LiteralPath $pidFile).Trim(), [ref]$devPID)) {
            $process = Get-Process -Id $devPID -ErrorAction SilentlyContinue
            if ($null -ne $process -and $process.ProcessName -eq 'dns-filter-dev' -and $process.Path -eq $exe) { return $process }
        }
    }
    # Scheduled startup creates a new PID after reboot; do not rely on an old file.
    return Get-Process -Name 'dns-filter-dev' -ErrorAction SilentlyContinue |
        Where-Object Path -eq $exe | Select-Object -First 1
}

function Get-ManagedStartupTask {
    $task = Get-ScheduledTask -TaskName 'DNSFilter' -ErrorAction SilentlyContinue
    if ($null -ne $task -and $task.Actions[0].Execute -eq $exe) { return $task }
    return $null
}
function Stop-DnsFilterProcesses {
    if ($null -ne (Get-ManagedStartupTask)) { Stop-ScheduledTask -TaskName 'DNSFilter' -ErrorAction SilentlyContinue }
    $ids = [Collections.Generic.HashSet[int]]::new()
    $dev = Get-DevProcess
    if ($null -ne $dev) { [void]$ids.Add($dev.Id) }

    foreach ($port in @(53, 80, 443, 8080)) {
        Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue | ForEach-Object {
            $process = Get-Process -Id $_.OwningProcess -ErrorAction SilentlyContinue
            if ($null -ne $process -and $process.ProcessName -like 'dns-filter*') {
                [void]$ids.Add($process.Id)
            }
        }
        Get-NetUDPEndpoint -LocalPort $port -ErrorAction SilentlyContinue | ForEach-Object {
            $process = Get-Process -Id $_.OwningProcess -ErrorAction SilentlyContinue
            if ($null -ne $process -and $process.ProcessName -like 'dns-filter*') {
                [void]$ids.Add($process.Id)
            }
        }
    }

    foreach ($processID in $ids) {
        Write-Host "Stopping DNS Filter process $processID..." -ForegroundColor Yellow
        Stop-Process -Id $processID -Force -ErrorAction SilentlyContinue
    }
    if ($ids.Count -gt 0) { Start-Sleep -Milliseconds 900 }
    Remove-Item -LiteralPath $pidFile -Force -ErrorAction SilentlyContinue
}

function Build-DevServer {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw 'Go was not found in PATH.'
    }
    New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
    $env:GOCACHE = Join-Path $root '.tmp\gocache-dev'
    $env:CGO_ENABLED = '0'
    Write-Host 'Building development server...' -ForegroundColor Cyan
    Push-Location $root
    try {
        & go build -o $exe ./cmd/server
        if ($LASTEXITCODE -ne 0) { throw "Go build failed with exit code $LASTEXITCODE." }
    } finally {
        Pop-Location
    }
}

function Start-DevServer {
    if (-not (Test-Path -LiteralPath $exe)) {
        throw "Development executable was not found: $exe"
    }
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $pidFile), $logDir | Out-Null
    Remove-Item -LiteralPath $stdoutLog, $stderrLog -Force -ErrorAction SilentlyContinue

    $managed = Get-ManagedStartupTask
    if ($null -ne $managed -and $managed.Settings.Enabled) {
        # Keep the SYSTEM identity used by startup; an interactive duplicate can
        # conflict with Windows ICS even when the SYSTEM loopback listener works.
        Start-ScheduledTask -TaskName 'DNSFilter'
        for ($attempt = 1; $attempt -le 90; $attempt++) {
            Start-Sleep -Milliseconds 500
            try {
                $health = Invoke-RestMethod -Uri $healthURL -TimeoutSec 2
                $scheduledProcess = Get-DevProcess
                if ($health.ready -and $null -ne $scheduledProcess) {
                    Set-Content -LiteralPath $pidFile -Value $scheduledProcess.Id -Encoding ASCII
                    Write-Host 'DNS Filter is running through its Windows startup task.' -ForegroundColor Green
                    Write-Host 'Dashboard: http://dnsfilter.localhost:8080'
                    return
                }
            } catch {}
        }
        throw 'The scheduled development server did not become healthy. Check the Windows Startup diagnostics and data/logs/dns-filter.log.'
    }
    Write-Host 'Starting directly from the repository...' -ForegroundColor Cyan
    $process = Start-Process -FilePath $exe -ArgumentList '-dev' -WorkingDirectory $root -WindowStyle Hidden -PassThru `
        -RedirectStandardOutput $stdoutLog -RedirectStandardError $stderrLog
    Set-Content -LiteralPath $pidFile -Value $process.Id -Encoding ASCII

    for ($attempt = 1; $attempt -le 30; $attempt++) {
        Start-Sleep -Milliseconds 500
        if ($process.HasExited) {
            $details = if (Test-Path $stderrLog) { (Get-Content -Raw $stderrLog).Trim() } else { '' }
            throw "Development server exited before becoming ready. $details"
        }
        try {
            $health = Invoke-RestMethod -Uri $healthURL -TimeoutSec 2
            if ($health.ready -eq $true) {
                Write-Host ''
                Write-Host 'DNS Filter development server is ready.' -ForegroundColor Green
                Write-Host "PID:       $($process.Id)"
                Write-Host 'Dashboard: http://dnsfilter.localhost:8080'
                Write-Host "Binary:    $exe"
                Write-Host 'Assets:    web\ (served directly from the repository)'
                return
            }
        } catch {}
    }
    throw 'Development server did not become healthy on port 8080.'
}

if ($Action -eq 'status') {
    $dev = Get-DevProcess
    if ($null -ne $dev) {
        Write-Host "Development server is running (PID $($dev.Id))." -ForegroundColor Green
        try { Invoke-RestMethod -Uri $healthURL -TimeoutSec 2 | ConvertTo-Json -Compress }
        catch { Write-Warning 'The process exists, but the health endpoint is unavailable.' }
        exit 0
    }
    Write-Host 'Development server is not running.' -ForegroundColor Yellow
    exit 1
}

if (-not (Test-Administrator)) {
    Write-Host 'Administrator permission is required for DNS port 53. Opening an elevated development runner...'
    Restart-Elevated
    exit 0
}

Set-Location $root

if ($Action -in @('stop', 'restart')) {
    Stop-DnsFilterProcesses
}
if ($Action -eq 'stop') {
    Write-Host 'DNS Filter development server stopped.' -ForegroundColor Green
    exit 0
}

$alreadyRunning = Get-DevProcess
if ($Action -eq 'start' -and $null -ne $alreadyRunning) {
    Write-Host "Development server is already running (PID $($alreadyRunning.Id))." -ForegroundColor Green
    exit 0
}

if (-not $NoBuild) { Build-DevServer }
Start-DevServer
