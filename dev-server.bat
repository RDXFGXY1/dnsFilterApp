@echo off
setlocal
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File ".\scripts\dev-server.ps1" restart
if errorlevel 1 pause
endlocal
