<#
  build-portable.ps1 — package DNS Filter as a portable Windows folder + zip.

  Produces dist\dns-filter-portable\ (extract-and-run) and a matching .zip.
  Run from the project root:  powershell -ExecutionPolicy Bypass -File scripts\build-portable.ps1
#>

$ErrorActionPreference = "Stop"

# ── Locate Go ────────────────────────────────────────────────────────────────
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    $env:Path += ";C:\Program Files\Go\bin"
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go not found. Install Go or add it to PATH."
}

$root    = Split-Path -Parent $PSScriptRoot
$version = "2.6.0"
$outName = "dns-filter-portable"
$dist    = Join-Path $root "dist\$outName"
Set-Location $root

Write-Host "==> Building dns-filter.exe (CGO disabled, static)..." -ForegroundColor Cyan
$env:CGO_ENABLED = "0"; $env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -ldflags "-s -w" -o "build\dns-filter.exe" ./cmd/server
if ($LASTEXITCODE -ne 0) { throw "build failed" }

Write-Host "==> Assembling portable folder..." -ForegroundColor Cyan
if (Test-Path $dist) { Remove-Item $dist -Recurse -Force }
New-Item -ItemType Directory -Force -Path $dist | Out-Null

Copy-Item "build\dns-filter.exe" $dist
Copy-Item "configs" $dist -Recurse
Copy-Item "web" $dist -Recurse
if (Test-Path "docs\RUNNING.md") { Copy-Item "docs\RUNNING.md" $dist }
New-Item -ItemType Directory -Force -Path (Join-Path $dist "data\logs") | Out-Null

# ── Launcher: Start ──────────────────────────────────────────────────────────
$start = @'
@echo off
title DNS Filter
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Requesting administrator privileges...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)
cd /d "%~dp0"
echo Starting DNS Filter... Dashboard: http://127.0.0.1:8080  (admin / changeme)
dns-filter.exe
pause
'@
Set-Content -Path (Join-Path $dist "1. Start DNS Filter.bat") -Value $start -Encoding ASCII

# ── Launcher: Enable Protection ──────────────────────────────────────────────
$enable = @'
@echo off
title Enable DNS Protection
net session >nul 2>&1
if %errorlevel% neq 0 (
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)
cd /d "%~dp0"
echo Pointing system DNS at the filter...
dns-filter.exe setup-dns
echo Disabling IPv6 DNS on the active adapter (prevents bypass)...
for /f "usebackq delims=" %%A in (`powershell -NoProfile -Command "(Get-NetRoute -DestinationPrefix '0.0.0.0/0' ^| Sort-Object RouteMetric ^| Select-Object -First 1).InterfaceAlias"`) do set IFACE=%%A
powershell -NoProfile -Command "Disable-NetAdapterBinding -Name '%IFACE%' -ComponentID ms_tcpip6 -ErrorAction SilentlyContinue"
if exist "data\ca\ca.crt" (
    echo Installing block-page CA certificate...
    certutil -addstore -f ROOT "data\ca\ca.crt" >nul
)
ipconfig /flushdns >nul
echo.
echo Protection ENABLED. Turn OFF "Secure DNS" in your browser, then restart it.
pause
'@
Set-Content -Path (Join-Path $dist "2. Enable Protection.bat") -Value $enable -Encoding ASCII

# ── Launcher: Disable Protection ─────────────────────────────────────────────
$disable = @'
@echo off
title Disable DNS Protection
net session >nul 2>&1
if %errorlevel% neq 0 (
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)
cd /d "%~dp0"
echo Restoring automatic DNS...
dns-filter.exe reset-dns
for /f "usebackq delims=" %%A in (`powershell -NoProfile -Command "(Get-NetRoute -DestinationPrefix '0.0.0.0/0' ^| Sort-Object RouteMetric ^| Select-Object -First 1).InterfaceAlias"`) do set IFACE=%%A
powershell -NoProfile -Command "Enable-NetAdapterBinding -Name '%IFACE%' -ComponentID ms_tcpip6 -ErrorAction SilentlyContinue"
ipconfig /flushdns >nul
echo.
echo Protection DISABLED. Your internet uses normal DNS again.
pause
'@
Set-Content -Path (Join-Path $dist "3. Disable Protection.bat") -Value $disable -Encoding ASCII

# ── README ───────────────────────────────────────────────────────────────────
$readme = @"
DNS FILTER $version — Portable Edition
======================================

QUICK START (Windows)
  1) Double-click "1. Start DNS Filter.bat"      (approve the admin prompt)
  2) Open http://127.0.0.1:8080  ->  login: admin / changeme  ->  change the password
  3) Double-click "2. Enable Protection.bat"     to route this PC's DNS through the filter
  4) In your browser settings, turn OFF "Secure DNS / DNS-over-HTTPS", then restart it

TO STOP
  - Close the DNS Filter window, then run "3. Disable Protection.bat" to restore normal DNS.
  - IMPORTANT: if you stop the filter without disabling protection, DNS will not work.

NOTES
  - Runs as Administrator (needs ports 53 / 80 / 443).
  - "2. Enable Protection.bat" also disables IPv6 DNS and installs the block-page CA so the
    block page works on HTTPS sites on THIS machine.
  - Everything is self-contained in this folder. Delete the folder to uninstall.
  - See RUNNING.md for Linux/macOS and full details.

Keep data\ca\ca.key private — it is your machine's local certificate authority key.
"@
Set-Content -Path (Join-Path $dist "README.txt") -Value $readme -Encoding UTF8

# ── Zip ──────────────────────────────────────────────────────────────────────
$zip = Join-Path $root "dist\$outName-$version.zip"
if (Test-Path $zip) { Remove-Item $zip -Force }
Write-Host "==> Zipping -> $zip" -ForegroundColor Cyan
Compress-Archive -Path "$dist\*" -DestinationPath $zip

Write-Host ""
Write-Host "Done." -ForegroundColor Green
Write-Host "  Folder: $dist"
Write-Host "  Zip:    $zip"
