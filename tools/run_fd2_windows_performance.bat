@echo off
setlocal
cd /d "%~dp0"
if not exist "%~dp0fd2.exe" exit /b 1
if not defined LOCALAPPDATA exit /b 1
set "FD2_PERF_DIR=%LOCALAPPDATA%\fd2_re\performance"
if not exist "%FD2_PERF_DIR%" mkdir "%FD2_PERF_DIR%"
if not exist "%FD2_PERF_DIR%" exit /b 1
set "FD2_PERF_ID=%RANDOM%-%RANDOM%"
set "FD2_PERFORMANCE_REPORT=%FD2_PERF_DIR%\performance-%FD2_PERF_ID%.jsonl"
set "FD2_PERF_ADAPTER_REPORT=%FD2_PERF_DIR%\adapters-%FD2_PERF_ID%.json"
if exist "%FD2_PERFORMANCE_REPORT%" exit /b 1
set "FD2_ASSET_PACK=%~dp0pack"
set "FD2_MODERN_THEME_PACK=%~dp0modern"
rem Read-only Windows adapter inventory; not proof of the game's selected adapter.
rem https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-videocontroller
powershell.exe -NoProfile -Command "Get-CimInstance -ClassName Win32_VideoController | Select-Object Name,DriverVersion,VideoProcessor,Status | ConvertTo-Json | Set-Content -LiteralPath $env:FD2_PERF_ADAPTER_REPORT -Encoding UTF8" 2> "%FD2_PERF_DIR%\adapters-%FD2_PERF_ID%.log"
"%~dp0fd2.exe" %* > "%FD2_PERF_DIR%\game-%FD2_PERF_ID%.log" 2>&1
echo Performance logs: %FD2_PERF_DIR%
pause
endlocal
