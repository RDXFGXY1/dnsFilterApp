<#
  build-portable.ps1 — package DNS Filter as a portable Windows folder + zip.

  Produces dist\dns-filter-portable\ (extract-and-run) and a matching .zip.
  Run from the project root:  powershell -ExecutionPolicy Bypass -File scripts\build-portable.ps1
#>

[CmdletBinding()]
param(
    [string]$SigningThumbprint = $env:DNS_FILTER_SIGNING_THUMBPRINT,
    [string]$TimestampServer = "http://timestamp.digicert.com",
    [switch]$RequireSigning
)

$ErrorActionPreference = "Stop"

# ── Locate Go ────────────────────────────────────────────────────────────────
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    $env:Path += ";C:\Program Files\Go\bin"
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go not found. Install Go or add it to PATH."
}

$root    = Split-Path -Parent $PSScriptRoot
$versionFile = Join-Path $root "internal\version\VERSION"
$version = (Get-Content -Raw $versionFile).Trim()
if ([string]::IsNullOrWhiteSpace($version)) {
    throw "Product version is missing from $versionFile"
}
$outName = "dns-filter-portable"
$dist    = Join-Path $root "dist\$outName"
Set-Location $root
$env:GOCACHE = Join-Path $root ".tmp\gocache"
New-Item -ItemType Directory -Force -Path $env:GOCACHE | Out-Null

Write-Host "==> Building dns-filter.exe (CGO disabled, static)..." -ForegroundColor Cyan
$env:CGO_ENABLED = "0"; $env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w" -o "build\dns-filter.exe" ./cmd/server
if ($LASTEXITCODE -ne 0) { throw "build failed" }

Write-Host "==> Assembling portable folder..." -ForegroundColor Cyan
if (Test-Path $dist) { Remove-Item $dist -Recurse -Force }
New-Item -ItemType Directory -Force -Path $dist | Out-Null

Copy-Item "build\dns-filter.exe" $dist
New-Item -ItemType Directory -Force -Path (Join-Path $dist "configs") | Out-Null
Copy-Item "configs\config.example.yaml" (Join-Path $dist "configs\config.yaml")
Copy-Item "configs\custom-blocklist.yaml" (Join-Path $dist "configs\custom-blocklist.yaml")
Copy-Item "web" $dist -Recurse
Copy-Item "scripts\update-windows.ps1" (Join-Path $dist "update-windows.ps1")
Copy-Item "scripts\uninstall-windows.ps1" (Join-Path $dist "uninstall-windows.ps1")
Copy-Item "internal\version\VERSION" (Join-Path $dist "VERSION")
if (Get-Command dotnet -ErrorAction SilentlyContinue) {
    Write-Host "==> Publishing Windows desktop..." -ForegroundColor Cyan
    $desktop = Join-Path $dist "desktop"
    dotnet publish "application\DnsFilter.Avalonia\DnsFilter.Avalonia.csproj" -c Release -r win-x64 --self-contained false -o $desktop -p:DebugType=None -p:DebugSymbols=false -p:PathMap="$root=/_/dnsFilterApp"
    if ($LASTEXITCODE -ne 0) { throw "desktop build failed" }
    Get-ChildItem -LiteralPath $desktop -Filter *.pdb -Recurse -File | Remove-Item -Force
}
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
echo Starting DNS Filter in the background...
powershell -NoProfile -Command "Start-Process -FilePath (Join-Path (Get-Location) 'dns-filter.exe') -WorkingDirectory (Get-Location) -WindowStyle Hidden"
for /L %%I in (1,1,30) do (
    powershell -NoProfile -Command "try { $h=Invoke-RestMethod -Uri 'http://127.0.0.1:8080/healthz' -TimeoutSec 2; $v=(Get-Content '.\VERSION' -Raw).Trim(); if ($h.ready -eq $true -and $h.version -eq $v) { exit 0 } } catch {}; exit 1" >nul 2>&1
    if not errorlevel 1 goto ready
    timeout /t 1 /nobreak >nul
)
echo ERROR: DNS Filter did not become ready. Protection was not enabled.
pause
exit /b 1
:ready
echo DNS Filter is ready. Dashboard: http://dnsfilter.localhost:8080
echo You can close this window; the background server will keep running.
timeout /t 4 /nobreak >nul
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
if errorlevel 1 (
    echo.
    echo ERROR: Protection was NOT enabled because the DNS server is not ready.
    echo Run "1. Start DNS Filter.bat" first and wait for the ready message.
    pause
    exit /b 1
)
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
set RESET_RESULT=%errorlevel%
ipconfig /flushdns >nul
echo.
if not "%RESET_RESULT%"=="0" (
    echo WARNING: Automatic IPv4/IPv6 DNS could not be confirmed.
    pause
    exit /b 1
)
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
  2) Open http://dnsfilter.localhost:8080 and complete the secure first-run setup
  3) Double-click "2. Enable Protection.bat"     to route this PC's DNS through the filter
  4) In your browser settings, turn OFF "Secure DNS / DNS-over-HTTPS", then restart it

TO STOP
  - Close the DNS Filter window, then run "3. Disable Protection.bat" to restore normal DNS.
  - IMPORTANT: if you stop the filter without disabling protection, DNS will not work.

NOTES
  - Runs as Administrator (needs ports 53 / 80 / 443).
  - "2. Enable Protection.bat" routes both IPv4 and IPv6 DNS locally and installs the
    block-page CA so the block page works on HTTPS sites on THIS machine.
  - Everything is self-contained in this folder. Delete the folder to uninstall.
  - See RUNNING.md for Linux/macOS and full details.

Keep data\ca\ca.key private — it is your machine's local certificate authority key.

SAFE INSTALL OR UPDATE
  Run PowerShell as Administrator, then:
  .\update-windows.ps1 -PackageDir .

  Existing configs, database, Vault, logs, certificates, and recovery state are preserved.
  If the new service fails its health check, the previous executable and web assets return automatically.
"@

$installUpdate = @'
@echo off
title DNS Filter - Safe Install or Update
net session >nul 2>&1
if %errorlevel% neq 0 (
    powershell -NoProfile -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File ".\update-windows.ps1" -PackageDir "."
if %errorlevel% neq 0 pause
'@
Set-Content -Path (Join-Path $dist "Install or Update DNS Filter.bat") -Value $installUpdate -Encoding ASCII

$uninstall = @'
@echo off
title DNS Filter - Safe Uninstall
net session >nul 2>&1
if %errorlevel% neq 0 (
    powershell -NoProfile -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File ".\uninstall-windows.ps1" -InstallDir "."
pause
'@
Set-Content -Path (Join-Path $dist "Uninstall DNS Filter.bat") -Value $uninstall -Encoding ASCII
Set-Content -Path (Join-Path $dist "README.txt") -Value $readme -Encoding UTF8

# ── Authenticode signing ────────────────────────────────────────────────────
if (-not [string]::IsNullOrWhiteSpace($SigningThumbprint)) {
    Write-Host "==> Authenticode-signing DNS Filter files..." -ForegroundColor Cyan
    & (Join-Path $PSScriptRoot "sign-release.ps1") -Path $dist -Thumbprint $SigningThumbprint -TimestampServer $TimestampServer
    if ($LASTEXITCODE -ne 0) { throw "release signing failed" }
}
elseif ($RequireSigning) {
    throw "Official release signing is required. Pass -SigningThumbprint or set DNS_FILTER_SIGNING_THUMBPRINT."
}
else {
    Write-Warning "Development package is unsigned. Official releases must use -RequireSigning and a code-signing certificate."
}

# ── Zip ──────────────────────────────────────────────────────────────────────
$zip = Join-Path $root "dist\$outName-$version.zip"
if (Test-Path $zip) { Remove-Item $zip -Force }
foreach ($oldMetadata in @("$zip.sha256", "$zip.manifest.json")) {
    if (Test-Path -LiteralPath $oldMetadata) { Remove-Item -LiteralPath $oldMetadata -Force }
}
Write-Host "==> Zipping -> $zip" -ForegroundColor Cyan
Compress-Archive -Path "$dist\*" -DestinationPath $zip

Write-Host "==> Auditing release folder and archive..." -ForegroundColor Cyan
& (Join-Path $PSScriptRoot "audit-release.ps1") -Path $dist
if ($LASTEXITCODE -ne 0) { throw "release folder audit failed" }
& (Join-Path $PSScriptRoot "audit-release.ps1") -Path $zip
if ($LASTEXITCODE -ne 0) { throw "release archive audit failed" }

Write-Host "==> Creating SHA-256 checksum and release manifest..." -ForegroundColor Cyan
$signingApplied = -not [string]::IsNullOrWhiteSpace($SigningThumbprint)
& (Join-Path $PSScriptRoot "write-release-metadata.ps1") -Artifact $zip -Version $version -AuthenticodeSigned:$signingApplied
if ($LASTEXITCODE -ne 0) { throw "release metadata generation failed" }
& (Join-Path $PSScriptRoot "verify-release.ps1") -Artifact $zip
if ($LASTEXITCODE -ne 0) { throw "release metadata verification failed" }

Write-Host ""
Write-Host "Done." -ForegroundColor Green
Write-Host "  Folder: $dist"
Write-Host "  Zip:    $zip"
Write-Host "  SHA256: $zip.sha256"
Write-Host "  Manifest: $zip.manifest.json"
