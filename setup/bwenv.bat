@echo off
setlocal enabledelayedexpansion

REM bwenv - Bitwarden + direnv CLI for Windows
REM Usage: bwenv init | bwenv interactive | bwenv remove | bwenv test

set "BWENV_VERSION=1.1.1"
set "LIB_DIR=%USERPROFILE%\.config\direnv\lib"
set "HELPER_SCRIPT=%LIB_DIR%\bitwarden_folders.sh"

REM Parse debug level from arguments
set "debug_level=1"
set "arg1=%~1"
if "!arg1!"=="--debug=0" ( set "debug_level=0" & shift )
if "!arg1!"=="--debug=1" ( set "debug_level=1" & shift )
if "!arg1!"=="--debug=2" ( set "debug_level=2" & shift )
if "!arg1!"=="--debug"   ( set "debug_level=2" & shift )
if "!arg1!"=="--quiet"   ( set "debug_level=0" & shift )
if "!arg1!"=="-q"        ( set "debug_level=0" & shift )

set "cmd=%~1"

if "!cmd!"=="init"        goto :cmd_init
if "!cmd!"=="interactive" goto :cmd_interactive
if "!cmd!"=="remove"      goto :cmd_remove
if "!cmd!"=="test"        goto :cmd_test
if "!cmd!"=="version"     goto :cmd_version
if "!cmd!"=="--version"   goto :cmd_version
if "!cmd!"=="-v"          goto :cmd_version
goto :cmd_help

:cmd_init
where bw >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Bitwarden CLI is not installed.
    echo   Install from: https://bitwarden.com/help/cli/
    exit /b 1
)

echo Syncing Bitwarden vault...
bw sync >nul 2>&1

set /p "folder_name=Enter Bitwarden folder name: "
if "!folder_name!"=="" (
    echo [ERROR] Folder name cannot be empty.
    exit /b 1
)

if "!BW_SESSION!"=="" (
    echo Unlocking Bitwarden vault...
    for /f "delims=" %%i in ('bw unlock --raw') do set "BW_SESSION=%%i"
    if "!BW_SESSION!"=="" (
        echo [ERROR] Failed to unlock vault.
        exit /b 1
    )
    echo Session obtained.
)

call :generate_envrc "!folder_name!"
goto :eof

:cmd_interactive
where bw >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Bitwarden CLI is not installed.
    echo   Install from: https://bitwarden.com/help/cli/
    exit /b 1
)

echo Syncing Bitwarden vault...
bw sync >nul 2>&1

if "!BW_SESSION!"=="" (
    echo Unlocking Bitwarden vault...
    for /f "delims=" %%i in ('bw unlock --raw') do set "BW_SESSION=%%i"
    if "!BW_SESSION!"=="" (
        echo [ERROR] Failed to unlock vault.
        exit /b 1
    )
    echo Session obtained.
)

REM Fetch folders and let user choose - uses PowerShell for JSON parsing
echo Fetching folders...
set "TEMP_FOLDER_FILE=%TEMP%\bwenv_folders_%RANDOM%.txt"
set "TEMP_RESULT_FILE=%TEMP%\bwenv_result_%RANDOM%.txt"

powershell -NoProfile -Command ^
  "$s = '!BW_SESSION!'; " ^
  "$raw = bw list folders --session $s 2>$null; " ^
  "if (-not $raw) { Write-Error 'No data from Bitwarden'; exit 1 }; " ^
  "$folders = $raw | ConvertFrom-Json; " ^
  "if ($folders.Count -eq 0) { Write-Error 'No folders found'; exit 1 }; " ^
  "Write-Host ''; " ^
  "Write-Host 'Available folders:'; " ^
  "for ($i = 0; $i -lt $folders.Count; $i++) { " ^
  "  Write-Host ('  ' + ($i + 1).ToString() + ') ' + $folders[$i].name) " ^
  "}; " ^
  "Write-Host ''; " ^
  "$choice = Read-Host 'Select folder number'; " ^
  "$idx = [int]$choice - 1; " ^
  "if ($idx -lt 0 -or $idx -ge $folders.Count) { Write-Error 'Invalid choice'; exit 1 }; " ^
  "$folders[$idx].name | Out-File -Encoding ascii -NoNewline '%TEMP_RESULT_FILE%'"

if errorlevel 1 (
    del /q "%TEMP_RESULT_FILE%" 2>nul
    echo [ERROR] Folder selection failed.
    exit /b 1
)

set /p folder_name=<"%TEMP_RESULT_FILE%"
del /q "%TEMP_RESULT_FILE%" 2>nul

if "!folder_name!"=="" (
    echo [ERROR] No folder selected.
    exit /b 1
)

call :generate_envrc "!folder_name!"
goto :eof

:cmd_remove
if exist .envrc (
    del /q .envrc
    echo .envrc removed from %CD%
) else (
    echo No .envrc found in %CD%
)
goto :eof

:cmd_test
echo bwenv v%BWENV_VERSION% -- Installation test
echo.
echo Dependencies:

where bw >nul 2>&1
if errorlevel 1 (
    echo   [MISSING] Bitwarden CLI
) else (
    for /f "delims=" %%v in ('bw --version 2^>nul') do echo   [OK] Bitwarden CLI: %%v
)

where direnv >nul 2>&1
if errorlevel 1 (
    echo   [MISSING] direnv
) else (
    for /f "delims=" %%v in ('direnv version 2^>nul') do echo   [OK] direnv: %%v
)

echo.
echo Configuration:

if exist "%HELPER_SCRIPT%" (
    echo   [OK] Helper script: %HELPER_SCRIPT%
) else (
    echo   [MISSING] Helper script: %HELPER_SCRIPT%
    echo           Run install.ps1 to fix this.
)

if defined BW_SESSION (
    echo   [OK] BW_SESSION: set
) else (
    echo   [INFO] BW_SESSION: not set ^(will be requested on first use^)
)
goto :eof

:cmd_version
echo bwenv %BWENV_VERSION%
goto :eof

:cmd_help
echo Usage: bwenv [OPTIONS] COMMAND
echo.
echo Commands:
echo   init          Initialize secrets (manual folder input)
echo   interactive   Initialize secrets (choose from list)
echo   remove        Remove .envrc from current directory
echo   test          Verify installation and dependencies
echo   version       Show version
echo.
echo Options:
echo   --quiet, -q   No debug output (BWENV_DEBUG=0)
echo   --debug       Full debug with secrets (BWENV_DEBUG=2)
echo   --debug=1     Show steps only, hide secrets (default)
echo   --debug=2     Show steps and secrets (full debug)
exit /b 0

:generate_envrc
set "gfolder=%~1"
(
    echo # Generated by bwenv -- do not edit manually
    echo export BW_SESSION="!BW_SESSION!"
    echo export BWENV_DEBUG=!debug_level!
    echo.
    echo # Suppress direnv slow-execution warning
    echo export DIRENV_WARN_TIMEOUT=600
    echo.
    echo # Load Bitwarden integration
    echo use bitwarden_folders
    echo load_bitwarden_folder_vars "!gfolder!"
) > .envrc
echo.
echo   SUCCESS
echo   .envrc created in: %CD%
echo   Using folder: !gfolder!
echo   Debug level: !debug_level!
echo.
echo   Next step: Run 'direnv allow' to load variables
echo.
goto :eof