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

REM Parse debug level from arguments
set debug_level=1
if "%1"=="--debug=0" set debug_level=0 && shift
if "%1"=="--debug=1" set debug_level=1 && shift  
if "%1"=="--debug=2" set debug_level=2 && shift
if "%1"=="--debug" set debug_level=2 && shift
if "%1"=="--quiet" set debug_level=0 && shift
if "%1"=="-q" set debug_level=0 && shift

REM Function to generate .envrc
if "%1"=="generate_envrc" (
    set "folder_name=%2"
    (
        echo # Bitwarden environment variables
        echo export BW_SESSION=%BW_SESSION% # Required for Bitwarden access
        echo.
        echo # Debug levels:
        echo #   BWENV_DEBUG=0: No debug output
        echo #   BWENV_DEBUG=1: Show steps only ^(default^)
        echo #   BWENV_DEBUG=2: Show steps and secrets ^(full debug^)
        echo export BWENV_DEBUG=%debug_level%
        echo.
        echo # Load Bitwarden integration
        echo use bitwarden_folders
        echo load_bitwarden_folder_vars "%folder_name%"
    ) > .envrc
    echo.
    echo ┌─────────────────────────────────────────────────┐
    echo │              ✅ SUCCESS                        │
    echo ├─────────────────────────────────────────────────┤
    echo │ 📁 .envrc created in: %CD%
    echo │ 📦 Using folder: %folder_name%
    if %debug_level%==0 (
        echo │ 📝 Debug level: %debug_level% ^(silent^)
    ) else if %debug_level%==1 (
        echo │ 📝 Debug level: %debug_level% ^(steps only^)
    ) else (
        echo │ 📝 Debug level: %debug_level% ^(full debug^)
    )
    echo │
    echo │ 🔄 Next step: Run 'direnv allow' to load variables
    echo └─────────────────────────────────────────────────┘
    echo.
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

) else if "%1"=="test" (
    echo 🧪 Testing bwenv installation...
    echo.
    echo 📋 Checking dependencies:

    where bw >nul 2>&1
    if errorlevel 1 (
        echo   ❌ Bitwarden CLI: not installed
    ) else (
        for /f "delims=" %%v in ('bw --version 2^>nul') do echo   ✅ Bitwarden CLI: %%v
    )

    where jq >nul 2>&1
    if errorlevel 1 (
        echo   ❌ jq: not installed
    ) else (
        for /f "delims=" %%v in ('jq --version 2^>nul') do echo   ✅ jq: %%v
    )

    where direnv >nul 2>&1
    if errorlevel 1 (
        echo   ❌ direnv: not installed
    ) else (
        for /f "delims=" %%v in ('direnv version 2^>nul') do echo   ✅ direnv: %%v
    )

    echo.
    echo 📋 Checking configuration:

    if exist "%HELPER_SCRIPT%" (
        echo   ✅ Helper script: %HELPER_SCRIPT%
    ) else (
        echo   ❌ Helper script: not found at %HELPER_SCRIPT%
    )

    if defined BW_SESSION (
        echo   ✅ BW_SESSION: set
    ) else (
        echo   ⚠️  BW_SESSION: not set
    )

) else (
    echo Usage:
    echo   bwenv [--debug[=LEVEL]^|--quiet] init          - Manual folder input
    echo   bwenv [--debug[=LEVEL]^|--quiet] interactive   - Interactive folder selection
    echo   bwenv remove                                  - Remove .envrc
    echo   bwenv test                                    - Test installation
    echo.
    echo Debug options:
    echo   --quiet, -q     No debug output ^(BWENV_DEBUG=0^)
    echo   --debug         Full debug with secrets ^(BWENV_DEBUG=2^)
    echo   --debug=1       Show steps only, hide secrets ^(default^)
    echo   --debug=2       Show steps and secrets ^(full debug^)
    exit /b 0
)