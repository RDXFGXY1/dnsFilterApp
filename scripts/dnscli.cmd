@echo off
setlocal EnableExtensions EnableDelayedExpansion
cd /d "%~dp0"
set "DNSCLI_SELF=%~f0"
set "DNSCLI_ARGS=%*"
set "NEEDS_ADMIN=0"
set "STOP_PARSING=0"
if "%~1"=="" goto help_only

rem Elevate the complete chain once, so -S -E stays ordered in one process.
for %%A in (%*) do call :classify_admin "%%~A"
if "%NEEDS_ADMIN%"=="1" (
  fltmc >nul 2>&1
  if errorlevel 1 (
    echo Requesting Administrator permission...
    powershell -NoProfile -Command "Start-Process -FilePath $env:DNSCLI_SELF -Verb RunAs -ArgumentList $env:DNSCLI_ARGS"
    exit /b !errorlevel!
  )
)

:parse_next
if "%~1"=="" exit /b 0
call :run_argument "%~1"
set "ACTION_EXIT=!errorlevel!"
if not "!ACTION_EXIT!"=="0" exit /b !ACTION_EXIT!
if "!STOP_PARSING!"=="1" exit /b 0
shift
goto parse_next

:classify_admin
set "CHECK_ARG=%~1"
for %%M in (-S -T -R -E -D -N -I -U -X -F start stop restart enable-dns disable-dns reset-network install update uninstall recover-dns) do (
  if /I "%CHECK_ARG%"=="%%M" set "NEEDS_ADMIN=1"
)
exit /b 0

:run_argument
set "COMMAND=%~1"
if /I "%COMMAND%"=="help" call :help& exit /b !errorlevel!
if /I "%COMMAND%"=="--help" call :help& exit /b !errorlevel!
if /I "%COMMAND%"=="-H" call :help& exit /b !errorlevel!
if /I "%COMMAND%"=="status" call :status& exit /b !errorlevel!
if /I "%COMMAND%"=="version" call :version& exit /b !errorlevel!
if /I "%COMMAND%"=="-V" call :version& exit /b !errorlevel!
if /I "%COMMAND%"=="dashboard" call :dashboard& exit /b !errorlevel!
if /I "%COMMAND%"=="start" call :start_service& exit /b !errorlevel!
if /I "%COMMAND%"=="-S" call :start_service& exit /b !errorlevel!
if /I "%COMMAND%"=="stop" call :stop_service& exit /b !errorlevel!
if /I "%COMMAND%"=="-T" call :stop_service& exit /b !errorlevel!
if /I "%COMMAND%"=="restart" call :restart_service& exit /b !errorlevel!
if /I "%COMMAND%"=="-R" call :restart_service& exit /b !errorlevel!
if /I "%COMMAND%"=="enable-dns" call :enable_dns& exit /b !errorlevel!
if /I "%COMMAND%"=="-E" call :enable_dns& exit /b !errorlevel!
if /I "%COMMAND%"=="disable-dns" call :disable_dns& exit /b !errorlevel!
if /I "%COMMAND%"=="-D" call :disable_dns& exit /b !errorlevel!
if /I "%COMMAND%"=="reset-network" call :reset_network& exit /b !errorlevel!
if /I "%COMMAND%"=="-N" call :reset_network& exit /b !errorlevel!
if /I "%COMMAND%"=="install" call :install_app& exit /b !errorlevel!
if /I "%COMMAND%"=="-I" call :install_app& exit /b !errorlevel!
if /I "%COMMAND%"=="update" call :update_app& exit /b !errorlevel!
if /I "%COMMAND%"=="-U" call :update_app& exit /b !errorlevel!
if /I "%COMMAND%"=="uninstall" call :uninstall_app& exit /b !errorlevel!
if /I "%COMMAND%"=="-X" call :uninstall_app& exit /b !errorlevel!
if /I "%COMMAND%"=="recover-dns" call :recover_dns& exit /b !errorlevel!
if /I "%COMMAND%"=="-F" call :recover_dns& exit /b !errorlevel!
echo Unknown command or flag: %COMMAND%
echo Run: dnscli -H
exit /b 2

:status
powershell -NoProfile -Command "try{$h=Invoke-RestMethod 'http://127.0.0.1:8080/healthz' -TimeoutSec 3; Write-Host ('DNS Filter: READY  version ' + $h.version) -ForegroundColor Green; exit 0}catch{Write-Host 'DNS Filter: NOT RUNNING' -ForegroundColor Red; exit 1}"
exit /b %errorlevel%

:version
if exist ".\VERSION" (type ".\VERSION"& exit /b 0)
if exist "..\internal\version\VERSION" (type "..\internal\version\VERSION"& exit /b 0)
echo VERSION file not found.
exit /b 1

:dashboard
start "" "http://127.0.0.1:8080/open"
exit /b 0

:start_service
echo Starting DNS Filter...
schtasks /Query /TN "DNSFilter" >nul 2>&1
if not errorlevel 1 (
  schtasks /Change /TN "DNSFilter" /ENABLE >nul 2>&1
  schtasks /Run /TN "DNSFilter" >nul
) else (
  if not exist "%~dp0dns-filter.exe" (echo dns-filter.exe was not found beside dnscli.cmd.& exit /b 1)
  powershell -NoProfile -Command "Start-Process -FilePath '%~dp0dns-filter.exe' -WorkingDirectory '%~dp0' -WindowStyle Hidden"
  if errorlevel 1 exit /b 1
)
call :wait_ready
exit /b %errorlevel%

:wait_ready
powershell -NoProfile -Command "$end=(Get-Date).AddSeconds(30); do { try { $h=Invoke-RestMethod 'http://127.0.0.1:8080/healthz' -TimeoutSec 2; if($h.ready){Write-Host 'DNS Filter is ready.' -ForegroundColor Green; exit 0} } catch {}; Start-Sleep 1 } while((Get-Date) -lt $end); Write-Host 'DNS Filter did not become ready within 30 seconds.' -ForegroundColor Red; exit 1"
exit /b %errorlevel%

:stop_service
echo Stopping DNS Filter...
schtasks /End /TN "DNSFilter" >nul 2>&1
powershell -NoProfile -Command "$target=[IO.Path]::GetFullPath('%~dp0dns-filter.exe'); Get-CimInstance Win32_Process -Filter \"Name='dns-filter.exe'\" -ErrorAction SilentlyContinue | Where-Object { $_.ExecutablePath -and [IO.Path]::GetFullPath($_.ExecutablePath) -eq $target } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }"
exit /b %errorlevel%

:restart_service
call :stop_service
timeout /t 1 /nobreak >nul
call :start_service
exit /b %errorlevel%

:enable_dns
if not exist "%~dp0dns-filter.exe" (echo dns-filter.exe was not found beside dnscli.cmd.& exit /b 1)
"%~dp0dns-filter.exe" setup-dns
exit /b %errorlevel%

:disable_dns
if not exist "%~dp0dns-filter.exe" (echo dns-filter.exe was not found beside dnscli.cmd.& exit /b 1)
"%~dp0dns-filter.exe" reset-dns
exit /b %errorlevel%

:reset_network
if not exist "%~dp0dns-filter.exe" (echo dns-filter.exe was not found beside dnscli.cmd.& exit /b 1)
"%~dp0dns-filter.exe" reset-app-network
exit /b %errorlevel%

:install_app
echo Installing DNS Filter from this portable package...
if not exist "%~dp0update-windows.ps1" (echo update-windows.ps1 was not found.& exit /b 1)
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0update-windows.ps1" -PackageDir "%~dp0"
exit /b %errorlevel%

:update_app
echo Updating DNS Filter from this portable package...
if not exist "%~dp0update-windows.ps1" (echo update-windows.ps1 was not found.& exit /b 1)
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0update-windows.ps1" -PackageDir "%~dp0"
exit /b %errorlevel%

:uninstall_app
echo Uninstalling DNS Filter while preserving user data...
if not exist "%~dp0uninstall-windows.ps1" (echo uninstall-windows.ps1 was not found.& exit /b 1)
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall-windows.ps1"
set "STOP_PARSING=1"
exit /b %errorlevel%

:recover_dns
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0emergency-restore-dns.ps1"
set "STOP_PARSING=1"
exit /b %errorlevel%

:help_only
call :help
exit /b %errorlevel%

:help
echo.
echo DNS Filter CLI - Windows portable control
echo.
echo Short flags can be chained and run from left to right:
echo   dnscli -S -E        Start, wait until ready, then enable DNS
echo   dnscli -D -T        Restore automatic DNS, then stop
echo.
echo   -S  start           Start the task or local portable service
echo   -T  stop            Stop the task or local portable service
echo   -R  restart         Restart and wait until ready
echo   -E  enable-dns      Route Windows DNS through DNS Filter
echo   -D  disable-dns     Restore automatic Windows DNS
echo   -N  reset-network   Release application network rules
echo   -I  install         Install from this portable package
echo   -U  update          Update from this portable package
echo   -X  uninstall       Uninstall and preserve user data
echo   -F  recover-dns     Emergency fallback; stops further actions
echo   -V  version         Show version
echo   -H  help            Show help
echo.
echo Other commands: status, dashboard
echo Full command names remain supported. System changes request Administrator
echo permission automatically. Uninstall and recovery must be the final action.
exit /b 0
