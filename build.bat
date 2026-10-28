@echo off

set EXE_NAME=WT Installer.exe
rem Go only links .syso files from the main package's folder.
set RSRC=cmd\wt-installer\rsrc.syso

echo Cleaning old resources...
if exist "%RSRC%" del "%RSRC%"
if exist rsrc.syso del rsrc.syso

echo Cleaning old executable...
if exist "%EXE_NAME%" del "%EXE_NAME%"

echo Checking for source PNG icon...
if not exist media\icon.png (
  echo ERROR: media\icon.png not found!
  pause
  exit /b 1
)

echo Checking for ImageMagick...
where magick >nul 2>nul
if %errorlevel% neq 0 (
  echo ERROR: ImageMagick is not installed or not in PATH!
  echo Run: winget install ImageMagick.ImageMagick
  pause
  exit /b 1
)

echo Converting PNG to multi-size ICO...
magick media\icon.png -define icon:auto-resize=256,48,32,16 media\icon.ico

if %errorlevel% neq 0 (
  echo ERROR: ImageMagick conversion failed.
  pause
  exit /b 1
)

echo Generating resources from icon...
rsrc -ico media\icon.ico -o "%RSRC%"

if %errorlevel% neq 0 (
  echo ERROR: rsrc generation failed.
  pause
  exit /b 1
)

echo Starting build process...
go build -ldflags "-s -w -H=windowsgui" -v -o "%EXE_NAME%" ./cmd/wt-installer

if %errorlevel% neq 0 (
  echo Build failed.
  pause
  exit /b 1
)
echo Build successful!

echo Compressing using UPX...
upx --best --lzma "%EXE_NAME%"

pause