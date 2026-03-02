# ByteBrew CLI installer for Windows.
# Usage: irm https://raw.githubusercontent.com/syntheticinc/bytebrew/main/scripts/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$Repo = 'syntheticinc/bytebrew'
$InstallDir = Join-Path $env:USERPROFILE '.bytebrew\bin'
$BinaryName = 'bytebrew.exe'

# Detect architecture
$Arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
switch ($Arch) {
    'X64'   { $PlatformArch = 'amd64' }
    'Arm64' { $PlatformArch = 'arm64' }
    default {
        Write-Error "Unsupported architecture: $Arch"
        exit 1
    }
}

$Platform = "windows_$PlatformArch"

# Get latest version from GitHub API
Write-Host 'Detecting latest version...'
$Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Version = $Release.tag_name -replace '^v', ''

if (-not $Version) {
    Write-Error "Could not detect latest version. Check https://github.com/$Repo/releases"
    exit 1
}

$Archive = "bytebrew_${Version}_${Platform}.zip"
$Url = "https://github.com/$Repo/releases/download/v$Version/$Archive"

Write-Host "Installing ByteBrew CLI v$Version ($Platform)..."
Write-Host "  From: $Url"
Write-Host ''

# Create install directory
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# Download
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "bytebrew-install-$(Get-Random)"
New-Item -ItemType Directory -Force -Path $TmpDir | Out-Null

try {
    Write-Host 'Downloading...'
    $ArchivePath = Join-Path $TmpDir $Archive
    Invoke-WebRequest -Uri $Url -OutFile $ArchivePath -UseBasicParsing

    Write-Host 'Extracting...'
    Expand-Archive -Path $ArchivePath -DestinationPath $TmpDir -Force

    # Install binary
    $BinaryPath = Join-Path $TmpDir $BinaryName
    Copy-Item -Path $BinaryPath -Destination (Join-Path $InstallDir $BinaryName) -Force

    Write-Host ''
    Write-Host "Installed: $InstallDir\$BinaryName"
}
catch {
    Write-Error "Installation failed: $_"
    Write-Error "Check that release v$Version exists for $Platform at: $Url"
    exit 1
}
finally {
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
}

# Check PATH
$UserPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
if ($UserPath -split ';' | Where-Object { $_ -eq $InstallDir }) {
    Write-Host ''
    Write-Host 'Ready! Run: bytebrew ask "hello"'
}
else {
    # Add to PATH automatically
    $NewPath = "$UserPath;$InstallDir"
    [Environment]::SetEnvironmentVariable('PATH', $NewPath, 'User')
    $env:PATH = "$env:PATH;$InstallDir"

    Write-Host ''
    Write-Host "Added $InstallDir to PATH."
    Write-Host 'Restart your terminal, then run: bytebrew ask "hello"'
}
