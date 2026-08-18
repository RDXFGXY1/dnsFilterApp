# ====================================================================
#  DNS Filter - Background Runner Script
#  Runs DNS filter hidden in background with first-run approval
# ====================================================================

param(
    [switch]$Force = $false
)

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$dnsFilterExe = Join-Path $scriptDir "build\dns-filter.exe"
$dataDir = Join-Path $scriptDir "data"
$approvalFlagFile = Join-Path $dataDir ".dns-filter-approved"
$logFile = Join-Path $dataDir "logs\dns-filter-startup.log"

function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    "$timestamp - $Message" | Add-Content -Path $logFile
}

function Test-AdminPrivileges {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Show-ApprovalDialog {
    [System.Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms") | Out-Null
    
    $form = New-Object System.Windows.Forms.Form
    $form.Text = "DNS Filter - Start Service?"
    $form.Width = 500
    $form.Height = 300
    $form.StartPosition = [System.Windows.Forms.FormStartPosition]::CenterScreen
    $form.TopMost = $true
    
    $titleLabel = New-Object System.Windows.Forms.Label
    $titleLabel.Text = "DNS Filter - Background Service"
    $titleLabel.Font = New-Object System.Drawing.Font("Arial", 14, [System.Drawing.FontStyle]::Bold)
    $titleLabel.Location = New-Object System.Drawing.Point(20, 20)
    $titleLabel.AutoSize = $true
    $form.Controls.Add($titleLabel)
    
    $descLabel = New-Object System.Windows.Forms.Label
    $descLabel.Text = "Start DNS Filter in the background?`n`n- Runs hidden`n- Protects your network`n- Dashboard: http://127.0.0.1:8080`n- Configures system DNS automatically`n`nThis will start the DNS Filter service and keep it running."
    $descLabel.Location = New-Object System.Drawing.Point(20, 60)
    $descLabel.Size = New-Object System.Drawing.Size(450, 140)
    $descLabel.AutoSize = $false
    $form.Controls.Add($descLabel)
    
    $yesButton = New-Object System.Windows.Forms.Button
    $yesButton.Text = "Yes, Start DNS Filter"
    $yesButton.Location = New-Object System.Drawing.Point(80, 220)
    $yesButton.Size = New-Object System.Drawing.Size(150, 40)
    $yesButton.BackColor = [System.Drawing.Color]::LimeGreen
    $yesButton.ForeColor = [System.Drawing.Color]::White
    $yesButton.Font = New-Object System.Drawing.Font("Arial", 10, [System.Drawing.FontStyle]::Bold)
    $yesButton.DialogResult = [System.Windows.Forms.DialogResult]::Yes
    $form.Controls.Add($yesButton)
    
    $noButton = New-Object System.Windows.Forms.Button
    $noButton.Text = "No, Not Now"
    $noButton.Location = New-Object System.Drawing.Point(280, 220)
    $noButton.Size = New-Object System.Drawing.Size(150, 40)
    $noButton.BackColor = [System.Drawing.Color]::Red
    $noButton.ForeColor = [System.Drawing.Color]::White
    $noButton.Font = New-Object System.Drawing.Font("Arial", 10, [System.Drawing.FontStyle]::Bold)
    $noButton.DialogResult = [System.Windows.Forms.DialogResult]::No
    $form.Controls.Add($noButton)
    
    $form.AcceptButton = $yesButton
    $form.CancelButton = $noButton
    
    $result = $form.ShowDialog()
    $form.Dispose()
    
    return $result -eq [System.Windows.Forms.DialogResult]::Yes
}

function Start-DnsFilter {
    Write-Host "[*] Starting DNS Filter in background..."
    Write-Log "Starting DNS Filter service..."
    
    if (-not (Test-Path $dataDir)) {
        New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
    }
    if (-not (Test-Path (Join-Path $dataDir "logs"))) {
        New-Item -ItemType Directory -Path (Join-Path $dataDir "logs") -Force | Out-Null
    }
    
    $existingProcess = Get-Process dns-filter -ErrorAction SilentlyContinue
    if ($existingProcess) {
        Write-Host "[+] DNS Filter is already running (PID: $($existingProcess.Id))"
        Write-Log "DNS Filter already running with PID $($existingProcess.Id)"
        return $true
    }
    
    try {
        $processInfo = New-Object System.Diagnostics.ProcessStartInfo
        $processInfo.FileName = $dnsFilterExe
        $processInfo.UseShellExecute = $false
        $processInfo.RedirectStandardOutput = $true
        $processInfo.RedirectStandardError = $true
        $processInfo.CreateNoWindow = $true
        $processInfo.WorkingDirectory = $scriptDir
        
        $process = [System.Diagnostics.Process]::Start($processInfo)
        $processId = $process.Id
        
        Write-Host "[+] DNS Filter started (PID: $processId)"
        Write-Log "DNS Filter started successfully with PID $processId"
        
        Start-Sleep -Seconds 2
        
        if ((Get-Process -Id $processId -ErrorAction SilentlyContinue)) {
            Write-Host "[+] DNS Filter is running"
            return $true
        } else {
            Write-Host "[-] DNS Filter process exited unexpectedly"
            Write-Log "ERROR: DNS Filter process exited unexpectedly"
            return $false
        }
    }
    catch {
        Write-Host "[-] Failed to start DNS Filter: $_"
        Write-Log "ERROR: Failed to start DNS Filter - $_"
        return $false
    }
}

function Setup-SystemDNS {
    Write-Host "[*] Configuring system DNS..."
    Write-Log "Running setup-dns..."
    
    try {
        & $dnsFilterExe setup-dns
        Write-Host "[+] System DNS configured"
        Write-Log "System DNS configured successfully"
        return $true
    }
    catch {
        Write-Host "[-] Failed to setup DNS: $_"
        Write-Log "ERROR: Failed to setup DNS - $_"
        return $false
    }
}

function Show-FirstRunMessage {
    Write-Host ""
    Write-Host "===================================================="
    Write-Host "   DNS FILTER - FIRST RUN APPROVAL GRANTED"
    Write-Host "===================================================="
    Write-Host ""
    Write-Host "[+] Approval recorded"
    Write-Host "[+] Will start on next launch"
    Write-Host "[+] Dashboard: http://127.0.0.1:8080"
    Write-Host "    Username: admin"
    Write-Host "    Password: changeme"
    Write-Host ""
    Write-Host "Next time you run this script, DNS Filter will start automatically."
    Write-Host ""
}

# Main Script
Clear-Host
Write-Host ""
Write-Host "===================================================="
Write-Host "   DNS FILTER - Background Service Launcher"
Write-Host "===================================================="
Write-Host ""

Write-Log "Script started"

if (-not (Test-AdminPrivileges)) {
    Write-Host "[-] ERROR: This script requires Administrator privileges"
    Write-Host "    Please run PowerShell as Administrator"
    Write-Log "ERROR: Script not running as Administrator"
    Read-Host "Press Enter to exit"
    exit 1
}

if (-not (Test-Path $dnsFilterExe)) {
    Write-Host "[-] ERROR: DNS Filter executable not found"
    Write-Host "    Expected path: $dnsFilterExe"
    Write-Host "    Build command: go build -o build\dns-filter.exe .\cmd\server"
    Write-Log "ERROR: DNS Filter executable not found at $dnsFilterExe"
    Read-Host "Press Enter to exit"
    exit 1
}

if ((-not $Force) -and (-not (Test-Path $approvalFlagFile))) {
    Write-Host "First run detected - asking for approval..."
    
    if (Show-ApprovalDialog) {
        New-Item -Path $approvalFlagFile -Force | Out-Null
        Show-FirstRunMessage
        Write-Host "Press Enter to continue..."
        Read-Host
    } else {
        Write-Host "User declined. Exiting..."
        Write-Log "User declined to start DNS Filter"
        exit 0
    }
}

Write-Host ""
if (-not (Start-DnsFilter)) {
    Write-Host "Failed to start DNS Filter"
    Write-Log "Failed to start DNS Filter"
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host ""
if (-not (Setup-SystemDNS)) {
    Write-Host "[!] Warning: Failed to setup system DNS"
    Write-Host "    You can setup manually later via the dashboard"
}

Write-Host ""
Write-Host "===================================================="
Write-Host "   DNS FILTER ACTIVE - Running in Background"
Write-Host "===================================================="
Write-Host ""
Write-Host "Dashboard:      http://127.0.0.1:8080"
Write-Host "Username:       admin"
Write-Host "Password:       changeme"
Write-Host ""
Write-Host "Logs:           data/logs/dns-filter.log"
Write-Host ""
Write-Host "The DNS Filter is running hidden in the background."
Write-Host "This window can now be closed."
Write-Host ""
Write-Host "IMPORTANT:"
Write-Host "- Disable Secure DNS in your browser"
Write-Host "- Chrome/Edge: Settings -> Security -> Use secure DNS -> OFF"
Write-Host "- Firefox: about:preferences -> DNS over HTTPS -> OFF"
Write-Host ""
Write-Host "To stop DNS Filter:"
Write-Host "  1. Run: stop-dns-filter.bat"
Write-Host "  2. Or use Task Manager to end dns-filter.exe"
Write-Host ""

Write-Log "DNS Filter service startup completed successfully"
Start-Sleep -Seconds 3
