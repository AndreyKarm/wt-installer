@echo off

set "EXE=WT Installer.exe"
set "EXT_FOLDER=extension"

set "EXE_ZIP=WT_Installer.zip"
set "EXT_ZIP=extension.zip"

if not exist "%EXE%" (
  echo [ERROR] "%EXE%" not found. Skipping.
) else (
  echo Archiving "%EXE%" -^> "%EXE_ZIP%" ...
  powershell -NoProfile -Command "Compress-Archive -Path '%EXE%' -DestinationPath '%EXE_ZIP%' -Force"
  if errorlevel 1 (
    echo [ERROR] Failed to create "%EXE_ZIP%".
  ) else (
    echo [OK] Created "%EXE_ZIP%".
  )
)

if not exist "%EXT_FOLDER%\" (
  echo [ERROR] Folder "%EXT_FOLDER%" not found. Skipping.
) else (
  echo Archiving "%EXT_FOLDER%\" -^> "%EXT_ZIP%" ...
  powershell -NoProfile -Command "Compress-Archive -Path '%EXT_FOLDER%\*' -DestinationPath '%EXT_ZIP%' -Force"
  if errorlevel 1 (
    echo [ERROR] Failed to create "%EXT_ZIP%".
  ) else (
    echo [OK] Created "%EXT_ZIP%".
  )
)

echo.
echo Done.
pause