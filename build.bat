@echo off

set EXE_NAME=WT Installer.exe

echo Cleaning old resources...
if exist rsrc.syso del rsrc.syso

echo Cleaning old executable...
if exist "%EXE_NAME%" del "%EXE_NAME%"

echo Checking for icon...
if not exist media\icon.ico (
  echo ERROR: media\icon.ico not found!
  pause
  exit /b 1
)

echo Generating resources from icon...
rsrc -ico media\icon.ico -o rsrc.syso

if %errorlevel% neq 0 (
  echo ERROR: rsrc generation failed.
  pause
  exit /b 1
)

echo Starting build process...
go build -ldflags "-s -w -H=windowsgui" -v -o "%EXE_NAME%"

if %errorlevel% equ 0 (
  echo Build successful!
) else (
  echo Build failed.
)
pause