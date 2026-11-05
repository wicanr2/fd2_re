@echo off
setlocal
cd /d "%~dp0"
if not exist "%~dp0fd2.exe" exit /b 1
set "FD2_ASSET_PACK=%~dp0pack"
set "FD2_MODERN_THEME_PACK=%~dp0modern"
set "EBITENGINE_GRAPHICS_LIBRARY=directx"
set "EBITENGINE_DIRECTX=version=11"
"%~dp0fd2.exe" %*
endlocal
