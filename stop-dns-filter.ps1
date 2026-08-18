# ════════════════════════════════════════════════════════════════
#  DNS Filter - Stop Service Script
#  Stops the DNS Filter and restores system DNS
# ════════════════════════════════════════════════════════════════

function Test-AdminPrivileges {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Write-Log {
    param([string]$Message)
    $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
    $dataDir = Join-Path $scriptDir "data"
    $logFile = Join-Path $dataDir "logs\dns-filter-startup.log"
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    "$timestamp - $Message" | Add-Content -Path $logFile -ErrorAction SilentlyContinue
}

Clear-Host
Write-Host @"
╔══════════════════════════════════════════════════════════════╗
║         DNS FILTER - Stop Service                           ║
╚══════════════════════════════════════════════════════════════╝
"@

# Check admin privileges
if (-not (Test-AdminPrivileges)) {
    Write-Host "✗ ERROR: This script requires Administrator privileges"
    Write-Host "  Please run PowerShell as Administrator"
    Read-Host "Press Enter to exit"
    exit 1
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$dnsFilterExe = Join-Path $scriptDir "build\dns-filter.exe"

# Check if DNS Filter executable exists
if (-not (Test-Path $dnsFilterExe)) {
    Write-Host "✗ ERROR: DNS Filter executable not found at: $dnsFilterExe"
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host ""
Write-Host "1. Stopping DNS Filter process..."

# Stop the process
$process = Get-Process dns-filter -ErrorAction SilentlyContinue
if ($process) {
    try {
        Stop-Process -InputObject $process -Force
        Write-Host "✓ DNS Filter process stopped"
        Write-Log "DNS Filter process stopped"
        Start-Sleep -Seconds 1
    }
    catch {
        Write-Host "✗ Failed to stop process: $_"
        Write-Log "ERROR: Failed to stop process - $_"
    }
} else {
    Write-Host "ℹ DNS Filter is not running"
    Write-Log "DNS Filter was not running when stop was called"
}

Write-Host ""
Write-Host "2. Restoring system DNS..."

try {
    & $dnsFilterExe reset-dns
    Write-Host "✓ System DNS restored"
    Write-Log "System DNS restored"
}
catch {
    Write-Host "⚠ Warning: Could not automatically restore DNS"
    Write-Host "  You may need to manually set DNS in Network Settings"
    Write-Log "WARNING: Failed to auto-restore DNS - $_"
}

Write-Host ""
Write-Host "3. Flushing DNS cache..."

try {
    ipconfig /flushdns | Out-Null
    Write-Host "✓ DNS cache flushed"
    Write-Log "DNS cache flushed"
}
catch {
    Write-Host "✗ Failed to flush DNS cache"
}

Write-Host ""
Write-Host @"
╔══════════════════════════════════════════════════════════════╗
║              ✓ DNS FILTER STOPPED                           ║
╚══════════════════════════════════════════════════════════════╝

✓ DNS Filter service stopped
✓ System DNS restored
✓ DNS cache flushed

⚠ Important:
   - Close all browsers and reopen them for DNS to take effect
   - Your internet should work normally now

To restart DNS Filter, run:
   start-dns-filter.bat

"@

Write-Log "DNS Filter stop completed successfully"
Read-Host "Press Enter to exit"
