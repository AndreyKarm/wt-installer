@echo off

set EXE_NAME=WTInstaller.exe

echo Cleaning old resources...
if exist rsrc.syso del rsrc.syso

echo Cleaning old executable...
if exist %EXE_NAME% del %EXE_NAME%

echo Checking for favicon...
if not exist media/favicon.ico (
  echo ERROR: media/favicon.ico not found!
  pause
  exit /b 1
)

echo Generating resources from new favicon...
rsrc -ico media/favicon.ico -o rsrc.syso

if %errorlevel% neq 0 (
  echo ERROR: rsrc generation failed.
  pause
  exit /b 1
)

echo Starting build process...
go build -ldflags "-s -w -H=windowsgui -extldflags=-static" -p 4 -v -o %EXE_NAME%

if %errorlevel% equ 0 (
  echo Build successful!
) else (
  echo Build failed.
)
pause