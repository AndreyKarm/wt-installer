@echo off

set EXE_NAME=WTInstaller.exe

echo Generating resources...
rsrc -ico res/app_win.ico -o rsrc.syso

echo Starting build process...
go build -ldflags "-s -w -H=windowsgui -extldflags=-static" -p 4 -v -o %EXE_NAME%

if %errorlevel% equ 0 (
  echo Build successful!
) else (
  echo Build failed.
)
pause