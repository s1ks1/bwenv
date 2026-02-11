# bwenv installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/s1ks1/bwenv/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "s1ks1/bwenv"
$Branch = "main"
$InstallLib = "$env:USERPROFILE\.config\direnv\lib"
$InstallBin = "$env:USERPROFILE\.local\bin"
$BaseURL = "https://raw.githubusercontent.com/$Repo/$Branch"

function Write-Info  { param($msg) Write-Host "  [INFO] $msg" -ForegroundColor Cyan }
function Write-Ok    { param($msg) Write-Host "  [OK]   $msg" -ForegroundColor Green }
function Write-Warn  { param($msg) Write-Host "  [WARN] $msg" -ForegroundColor Yellow }

Write-Host "`nInstalling bwenv...`n" -ForegroundColor Blue

# Check dependencies (only bw and direnv - jq is not needed on Windows)
Write-Host "Checking dependencies..." -ForegroundColor White
foreach ($dep in @("bw", "direnv")) {
    if (Get-Command $dep -ErrorAction SilentlyContinue) {
        Write-Ok "$dep found"
    } else {
        Write-Warn "$dep not found - please install it"
    }
}

# Create directories
New-Item -ItemType Directory -Force -Path $InstallLib | Out-Null
New-Item -ItemType Directory -Force -Path $InstallBin | Out-Null

# Download files
Write-Host "`nDownloading bwenv..." -ForegroundColor White

try {
    Invoke-WebRequest -Uri "$BaseURL/setup/bitwarden_folders.sh" -OutFile "$InstallLib\bitwarden_folders.sh" -UseBasicParsing
    Write-Ok "Helper script -> $InstallLib\bitwarden_folders.sh"
} catch {
    Write-Host "  [ERROR] Failed to download helper script: $_" -ForegroundColor Red
    exit 1
}

try {
    Invoke-WebRequest -Uri "$BaseURL/setup/bwenv.bat" -OutFile "$InstallBin\bwenv.bat" -UseBasicParsing
    Write-Ok "CLI -> $InstallBin\bwenv.bat"
} catch {
    Write-Host "  [ERROR] Failed to download bwenv.bat: $_" -ForegroundColor Red
    exit 1
}

# Also install the bash version for Git Bash / WSL users
try {
    Invoke-WebRequest -Uri "$BaseURL/setup/bwenv" -OutFile "$InstallBin\bwenv" -UseBasicParsing
    Write-Ok "Bash CLI -> $InstallBin\bwenv (for Git Bash/WSL)"
} catch {
    Write-Warn "Could not download bash version (optional)"
}

# Check/update PATH
Write-Host ""
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -and ($UserPath -split ";" | Where-Object { $_ -eq $InstallBin })) {
    Write-Ok "$InstallBin is in your PATH"
} else {
    if ($UserPath) {
        $NewPath = "$InstallBin;$UserPath"
    } else {
        $NewPath = $InstallBin
    }
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    $env:PATH = "$InstallBin;$env:PATH"
    Write-Ok "Added $InstallBin to your user PATH"
    Write-Info "Restart your terminal for PATH changes to take effect"
}

Write-Host "`nbwenv installed successfully!" -ForegroundColor Green
Write-Host "   Run 'bwenv test' to verify your setup.`n"
