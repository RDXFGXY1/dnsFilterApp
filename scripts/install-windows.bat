@echo off
setlocal
title DNS Filter - Safe Windows Setup
cd /d "%~dp0\.."

net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Requesting Administrator permission...
    powershell -NoProfile -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

echo Building the verified portable package...
powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\build-portable.ps1"
if errorlevel 1 goto :failed

echo Installing with state preservation and automatic rollback...
powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\update-windows.ps1" -PackageDir ".\dist\dns-filter-portable" -InstallDir "%ProgramFiles%\DNS Filter"
if errorlevel 1 goto :failed

echo.
echo DNS Filter installation completed securely.
echo Dashboard: http://dnsfilter.localhost:8080
exit /b 0

:failed
echo.
echo DNS Filter installation failed. Review the message above.
pause
exit /b 1
