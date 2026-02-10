# bwenv uninstaller for Windows (PowerShell)

$ErrorActionPreference = "Stop"

$InstallLib = "$env:USERPROFILE\.config\direnv\lib"
$InstallBin = "$env:USERPROFILE\.local\bin"

Write-Host "`nUninstalling bwenv..." -ForegroundColor Blue

$files = @(
    "$InstallLib\bitwarden_folders.sh",
    "$InstallBin\bwenv.bat",
    "$InstallBin\bwenv"
)

foreach ($f in $files) {
    if (Test-Path $f) {
        Remove-Item -Force $f
        Write-Host "  +  Removed $f" -ForegroundColor Green
    }
}

# Remove from PATH
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
$NewPath = ($UserPath -split ";" | Where-Object { $_ -ne $InstallBin }) -join ";"
if ($NewPath -ne $UserPath) {
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    Write-Host "  +  Removed $InstallBin from PATH" -ForegroundColor Green
}

Write-Host "`nbwenv uninstalled.`n" -ForegroundColor Green
