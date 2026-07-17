@echo off

set "EXT_FOLDER=extension"
set "EXT_ZIP=extension.zip"

if not exist "%EXT_FOLDER%\" (
  echo [ERROR] Folder "%EXT_FOLDER%" not found. Skipping.
) else (
  echo Archiving "%EXT_FOLDER%\" -^> "%EXT_ZIP%" ...
  powershell -NoProfile -Command "Get-ChildItem -Path \".\%EXT_FOLDER%\" | Compress-Archive -DestinationPath \".\%EXT_ZIP%\" -Force"
  if errorlevel 1 (
    echo [ERROR] Failed to create "%EXT_ZIP%".
  ) else (
    echo [OK] Created "%EXT_ZIP%".
  )
)

echo.
echo Done.
pause