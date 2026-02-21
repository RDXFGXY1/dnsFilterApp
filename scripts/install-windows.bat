@echo off
REM DNS Filter - Windows Installation Script

echo ========================================
echo   DNS Content Filter - Windows Setup
echo ========================================
echo.

REM Check for admin privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Error: This script requires administrator privileges
    echo Please run as Administrator
    pause
    exit /b 1
)

REM Set installation directory
set INSTALL_DIR=C:\Program Files\DNS-Filter

REM Create directories
echo Creating installation directories...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%INSTALL_DIR%\configs" mkdir "%INSTALL_DIR%\configs"
if not exist "%INSTALL_DIR%\data" mkdir "%INSTALL_DIR%\data"
if not exist "%INSTALL_DIR%\data\logs" mkdir "%INSTALL_DIR%\data\logs"
if not exist "%INSTALL_DIR%\web" mkdir "%INSTALL_DIR%\web"

REM Copy files
echo Copying files...
copy /Y build\dns-filter-windows-amd64.exe "%INSTALL_DIR%\dns-filter.exe"
xcopy /E /I /Y configs "%INSTALL_DIR%\configs"
xcopy /E /I /Y web "%INSTALL_DIR%\web"

REM Create Windows Service using NSSM (if available)
echo.
echo To install as a Windows Service, you can use NSSM:
echo 1. Download NSSM from https://nssm.cc/download
echo 2. Run: nssm install DNS-Filter "%INSTALL_DIR%\dns-filter.exe"
echo 3. Configure the service using: nssm edit DNS-Filter
echo.

REM Configure Windows Firewall
echo Configuring Windows Firewall...
netsh advfirewall firewall add rule name="DNS Filter - DNS" dir=in action=allow protocol=UDP localport=53
netsh advfirewall firewall add rule name="DNS Filter - Web" dir=in action=allow protocol=TCP localport=8080

echo.
echo ========================================
echo   Installation Complete!
echo ========================================
echo.
echo Installation directory: %INSTALL_DIR%
echo.
echo Next steps:
echo 1. Edit configuration: %INSTALL_DIR%\configs\config.yaml
echo 2. Run the service or execute: %INSTALL_DIR%\dns-filter.exe
echo 3. Set your DNS to 127.0.0.1 in Network Settings
echo 4. Access dashboard: http://localhost:8080
echo.
echo Default login:
echo   Username: admin
echo   Password: changeme
echo.
echo WARNING: Remember to change the default password!
echo.
pause
