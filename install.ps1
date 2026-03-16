# =============================================================================
# bwenv — Windows Installer (PowerShell)
#
# Usage:
#   irm https://raw.githubusercontent.com/s1ks1/bwenv/main/install.ps1 | iex
#
# Or download and run manually:
#   Invoke-WebRequest -Uri "https://raw.githubusercontent.com/s1ks1/bwenv/main/install.ps1" -OutFile install.ps1
#   .\install.ps1
#
# Options:
#   -Version     Specific version to install (default: latest)
#   -InstallDir  Installation directory (default: ~\.local\bin)
#
# Supports: Windows amd64 and arm64
# =============================================================================

param(
    [string]$Version = "",
    [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"

# -- Configuration --
$GitHubRepo = "s1ks1/bwenv"
if (-not $InstallDir) {
    $InstallDir = Join-Path $env:USERPROFILE ".local\bin"
}

# -- Helpers --
function Write-Info    { param($msg) Write-Host "  [info]  $msg" -ForegroundColor Blue }
function Write-OK      { param($msg) Write-Host "  [ok]    $msg" -ForegroundColor Green }
function Write-Warn    { param($msg) Write-Host "  [warn]  $msg" -ForegroundColor Yellow }
function Write-Err     { param($msg) Write-Host "  [error] $msg" -ForegroundColor Red; exit 1 }

# -- Detect Architecture --
function Get-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default {
            # Fallback: check PROCESSOR_ARCHITECTURE
            $procArch = $env:PROCESSOR_ARCHITECTURE
            switch ($procArch) {
                "AMD64" { return "amd64" }
                "ARM64" { return "arm64" }
                default { Write-Err "Unsupported architecture: $procArch" }
            }
        }
    }
}

# -- Get Latest Version --
function Get-LatestVersion {
    $url = "https://api.github.com/repos/$GitHubRepo/releases/latest"
    try {
        $response = Invoke-RestMethod -Uri $url -Headers @{ "User-Agent" = "bwenv-installer" }
        return $response.tag_name
    }
    catch {
        Write-Err "Could not determine latest version. Set -Version manually."
    }
}

# -- Verify Checksum --
function Test-Checksum {
    param($FilePath, $ArchiveName, $Ver)

    $checksumsUrl = "https://github.com/$GitHubRepo/releases/download/$Ver/checksums.txt"
    try {
        $checksums = Invoke-RestMethod -Uri $checksumsUrl -Headers @{ "User-Agent" = "bwenv-installer" }
    }
    catch {
        Write-Warn "Checksums not available - skipping verification"
        return
    }

    $line = ($checksums -split "`n") | Where-Object { $_ -match [regex]::Escape($ArchiveName) } | Select-Object -First 1
    if (-not $line) {
        Write-Warn "No checksum found for $ArchiveName - skipping verification"
        return
    }

    $expected = ($line -split '\s+')[0]
    $actual = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLower()

    if ($expected -eq $actual) {
        Write-OK "Checksum verified"
    }
    else {
        Write-Err "Checksum mismatch!`n  Expected: $expected`n  Actual:   $actual"
    }
}

# -- Add to PATH --
function Add-ToUserPath {
    param($Dir)

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -split ";" | Where-Object { $_ -eq $Dir }) {
        Write-OK "$Dir is already in your PATH"
        return $true
    }

    Write-Info "Adding $Dir to your user PATH..."
    $newPath = "$Dir;$currentPath"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")

    # Also update the current session.
    $env:Path = "$Dir;$env:Path"

    Write-OK "Added $Dir to PATH"
    Write-Info "The PATH change takes effect in new terminal windows."
    return $true
}

# -- Main --
function Install-Bwenv {
    Write-Host ""
    Write-Host "  bwenv installer" -ForegroundColor Blue -NoNewline
    Write-Host " (Windows)" -ForegroundColor DarkGray
    Write-Host "  ----------------------------------------"
    Write-Host ""

    $arch = Get-Arch
    Write-Info "Detected architecture: windows/$arch"

    # Determine version.
    if ($Version) {
        Write-Info "Using specified version: $Version"
    }
    else {
        Write-Info "Fetching latest release..."
        $Version = Get-LatestVersion
        Write-Info "Latest version: $Version"
    }

    # Construct download URL.
    $archiveName = "bwenv-$Version-windows-$arch.zip"
    $downloadUrl = "https://github.com/$GitHubRepo/releases/download/$Version/$archiveName"

    # Create temp directory.
    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "bwenv-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    try {
        # Download the archive.
        Write-Info "Downloading $archiveName..."
        $archivePath = Join-Path $tmpDir $archiveName

        try {
            Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
        }
        catch {
            Write-Err "Download failed. Check that version $Version exists for windows/$arch.`n  URL: $downloadUrl"
        }

        Write-OK "Downloaded successfully"

        # Verify checksum.
        Test-Checksum -FilePath $archivePath -ArchiveName $archiveName -Ver $Version

        # Extract the archive.
        Write-Info "Extracting..."
        $extractDir = Join-Path $tmpDir "extracted"
        Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force

        # Find the binary.
        $binary = Get-ChildItem -Path $extractDir -Recurse -Filter "bwenv.exe" | Select-Object -First 1
        if (-not $binary) {
            Write-Err "Could not find bwenv.exe in the archive"
        }

        # Install to the target directory.
        Write-Info "Installing to $InstallDir..."
        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }

        $destPath = Join-Path $InstallDir "bwenv.exe"
        Copy-Item -Path $binary.FullName -Destination $destPath -Force

        Write-OK "Installed bwenv $Version to $destPath"

        # Add to PATH if needed.
        Write-Host ""
        Add-ToUserPath -Dir $InstallDir

        # Also copy the .cmd wrapper for convenience in cmd.exe.
        $cmdWrapper = Join-Path $InstallDir "bwenv.cmd"
        if (-not (Test-Path $cmdWrapper)) {
            $wrapperContent = @"
@echo off
"%~dp0bwenv.exe" %*
"@
            Set-Content -Path $cmdWrapper -Value $wrapperContent -Encoding ASCII
        }

        # Verify installation.
        Write-Host ""
        try {
            $null = & $destPath version 2>&1
            Write-OK "bwenv is ready! Run 'bwenv status' to verify your setup."
        }
        catch {
            Write-Info "Installation complete. Open a new terminal and run: bwenv status"
        }
    }
    finally {
        # Clean up temp directory.
        if (Test-Path $tmpDir) {
            Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    Write-Host ""
}

Install-Bwenv
