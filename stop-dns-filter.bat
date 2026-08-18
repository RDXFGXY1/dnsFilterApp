@echo off
REM ════════════════════════════════════════════════════════════════
REM  DNS Filter - Stop Service Launcher
REM  Double-click this file to stop DNS Filter
REM ════════════════════════════════════════════════════════════════

echo.
echo Checking for Administrator privileges...
echo.

REM Check for admin privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo.
    echo ERROR: This script requires Administrator privileges!
    echo.
    echo Please:
    echo  1. Right-click this file
    echo  2. Select "Run as Administrator"
    echo.
    pause
    exit /b 1
)

REM Get the directory where this batch file is located
setlocal enabledelayedexpansion
set SCRIPT_DIR=%~dp0
set PS_SCRIPT=%SCRIPT_DIR%stop-dns-filter.ps1

echo Stopping DNS Filter...
echo.

REM Check if PowerShell script exists
if not exist "!PS_SCRIPT!" (
    echo ERROR: PowerShell script not found!
    echo Expected: !PS_SCRIPT!
    echo.
    pause
    exit /b 1
)

REM Run the PowerShell script
powershell -NoProfile -ExecutionPolicy Bypass -File "!PS_SCRIPT!"

exit /b %errorLevel%
