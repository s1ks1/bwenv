@echo off
setlocal enabledelayedexpansion

REM Global Bitwarden + direnv CLI for Windows
REM Usage: bwenv init | bwenv interactive | bwenv remove

set "LIB_DIR=%USERPROFILE%\.config\direnv\lib"
set "HELPER_SCRIPT=%LIB_DIR%\bitwarden_folders.sh"

REM Check if helper exists
if not exist "%HELPER_SCRIPT%" (
    echo ⚠️ Helper script not found in %HELPER_SCRIPT%
    echo Run 'make install' first.
    exit /b 1
)

REM Function to generate .envrc
if "%1"=="generate_envrc" (
    set "folder_name=%2"
    (
        echo export BW_SESSION=%BW_SESSION% # Optional
        echo export DEBUG_BW=true
        echo use bitwarden_folders
        echo load_bitwarden_folder_vars "%folder_name%"
    ) > .envrc
    echo ✅ .envrc created in %CD% using folder: %folder_name%
    echo Run 'direnv allow' to load variables
    exit /b 0
)

REM Check if Bitwarden CLI is installed
where bw >nul 2>&1
if errorlevel 1 (
    echo ❌ Bitwarden CLI is not installed.
    exit /b 1
)

if "%1"=="init" (
    REM Sync Bitwarden
    bw sync >nul 2>&1

    REM Manual folder input
    set /p "folder_name=📦 Enter Bitwarden folder name to load secrets from: "
    if "!folder_name!"=="" (
        echo ⚠️ Folder name cannot be empty.
        exit /b 1
    )

    REM Unlock session if needed
    if "%BW_SESSION%"=="" (
        echo 🔑 Unlocking Bitwarden vault...
        for /f "delims=" %%i in ('bw unlock --raw') do set "BW_SESSION=%%i"
        echo 🔑 BW_SESSION obtained
    )

    call "%~f0" generate_envrc "!folder_name!"

) else if "%1"=="interactive" (
    REM Sync Bitwarden
    bw sync >nul 2>&1

    REM Unlock session if needed
    if "%BW_SESSION%"=="" (
        echo 🔑 Unlocking Bitwarden vault...
        for /f "delims=" %%i in ('bw unlock --raw') do set "BW_SESSION=%%i"
        echo 🔑 BW_SESSION obtained
    )

    REM Get all folders using PowerShell for JSON parsing
    powershell -Command "& {$folders = bw list folders --session '%BW_SESSION%' | ConvertFrom-Json; if ($folders.Count -eq 0) { Write-Host '⚠️ No folders found in Bitwarden.'; exit 1 }; Write-Host '📂 Available folders:'; for ($i = 0; $i -lt $folders.Count; $i++) { Write-Host (' {0}) {1}' -f ($i + 1), $folders[$i].name) }; $choice = Read-Host 'Select folder number'; $choiceIndex = [int]$choice - 1; if ($choiceIndex -lt 0 -or $choiceIndex -ge $folders.Count) { Write-Host '⚠️ Invalid choice.'; exit 1 }; $folders[$choiceIndex].name }" > %TEMP%\bwenv_folder.txt

    if errorlevel 1 (
        del /q %TEMP%\bwenv_folder.txt 2>nul
        exit /b 1
    )

    set /p folder_name=<%TEMP%\bwenv_folder.txt
    del /q %TEMP%\bwenv_folder.txt

    call "%~f0" generate_envrc "!folder_name!"

) else if "%1"=="remove" (
    echo 🧹 Removing .envrc in %CD%...
    del /q .envrc 2>nul
    echo ✅ .envrc removed

) else (
    echo Usage: bwenv init ^| bwenv interactive ^| bwenv remove
    exit /b 0
)