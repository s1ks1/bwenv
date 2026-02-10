# bwenv installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/s1ks1/bwenv/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "s1ks1/bwenv"
$Branch = "main"
$InstallLib = "$env:USERPROFILE\.config\direnv\lib"
$InstallBin = "$env:USERPROFILE\.local\bin"
$BaseURL = "https://raw.githubusercontent.com/$Repo/$Branch"

function Write-Info  { param($msg) Write-Host "  i  $msg" -ForegroundColor Cyan }
function Write-Ok    { param($msg) Write-Host "  +  $msg" -ForegroundColor Green }
function Write-Warn  { param($msg) Write-Host "  !  $msg" -ForegroundColor Yellow }
function Write-Err   { param($msg) Write-Host "  X  $msg" -ForegroundColor Red }

Write-Host "`nInstalling bwenv...`n" -ForegroundColor Blue

# Check dependencies
Write-Host "Checking dependencies..." -ForegroundColor White
foreach ($dep in @("bw", "direnv", "jq")) {
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
Invoke-WebRequest -Uri "$BaseURL/setup/bitwarden_folders.sh" -OutFile "$InstallLib\bitwarden_folders.sh" -UseBasicParsing
Write-Ok "Helper script installed to $InstallLib\bitwarden_folders.sh"

Invoke-WebRequest -Uri "$BaseURL/setup/bwenv.bat" -OutFile "$InstallBin\bwenv.bat" -UseBasicParsing
Write-Ok "CLI installed to $InstallBin\bwenv.bat"

# Also install the bash version for Git Bash / WSL users
Invoke-WebRequest -Uri "$BaseURL/setup/bwenv" -OutFile "$InstallBin\bwenv" -UseBasicParsing
Write-Ok "Bash CLI installed to $InstallBin\bwenv (for Git Bash/WSL)"

# Check/update PATH
Write-Host ""
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -split ";" | Where-Object { $_ -eq $InstallBin }) {
    Write-Ok "$InstallBin is in your PATH"
} else {
    $NewPath = "$InstallBin;$UserPath"
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    $env:PATH = "$InstallBin;$env:PATH"
    Write-Ok "Added $InstallBin to your user PATH"
    Write-Info "Restart your terminal for PATH changes to take effect"
}

Write-Host "`nbwenv installed successfully!" -ForegroundColor Green
Write-Host "   Run 'bwenv test' to verify your setup.`n"
