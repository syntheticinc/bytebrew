# ByteBrew installer for Windows (CLI + Server).
# Usage: irm https://bytebrew.ai/releases/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$BaseUrl = 'https://bytebrew.ai/releases'
$InstallDir = Join-Path $env:USERPROFILE '.bytebrew\bin'

# Detect architecture
$Arch = $env:PROCESSOR_ARCHITECTURE
switch ($Arch) {
    'AMD64'  { $PlatformArch = 'amd64' }
    'x86'    { $PlatformArch = 'amd64' }  # 32-bit PS on 64-bit Windows
    'ARM64'  { $PlatformArch = 'arm64' }
    default  {
        Write-Error "Unsupported architecture: $Arch"
        exit 1
    }
}

$Platform = "windows_$PlatformArch"

# Get latest version
Write-Host 'Detecting latest version...'
$Version = (Invoke-RestMethod -Uri "$BaseUrl/LATEST" -UseBasicParsing).Trim()

if (-not $Version) {
    Write-Error "Could not detect latest version. Check $BaseUrl/LATEST"
    exit 1
}

Write-Host "Installing ByteBrew v$Version ($Platform)..."
Write-Host ''

# Create install directory
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# Download temp directory
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "bytebrew-install-$(Get-Random)"
New-Item -ItemType Directory -Force -Path $TmpDir | Out-Null

try {
    # --- CLI ---
    $CliArchive = "bytebrew_${Version}_${Platform}.zip"
    $CliUrl = "$BaseUrl/v$Version/$CliArchive"
    Write-Host "Downloading CLI...  $CliArchive"
    $CliArchivePath = Join-Path $TmpDir $CliArchive
    Invoke-WebRequest -Uri $CliUrl -OutFile $CliArchivePath -UseBasicParsing

    $CliExtractDir = Join-Path $TmpDir 'cli'
    Expand-Archive -Path $CliArchivePath -DestinationPath $CliExtractDir -Force
    Copy-Item -Path (Join-Path $CliExtractDir 'bytebrew.exe') -Destination (Join-Path $InstallDir 'bytebrew.exe') -Force

    # --- Server ---
    $SrvArchive = "bytebrew-srv_${Version}_${Platform}.zip"
    $SrvUrl = "$BaseUrl/v$Version/$SrvArchive"
    Write-Host "Downloading Server... $SrvArchive"
    $SrvArchivePath = Join-Path $TmpDir $SrvArchive
    Invoke-WebRequest -Uri $SrvUrl -OutFile $SrvArchivePath -UseBasicParsing

    $SrvExtractDir = Join-Path $TmpDir 'srv'
    Expand-Archive -Path $SrvArchivePath -DestinationPath $SrvExtractDir -Force
    Copy-Item -Path (Join-Path $SrvExtractDir 'bytebrew-srv.exe') -Destination (Join-Path $InstallDir 'bytebrew-srv.exe') -Force

    Write-Host ''
    Write-Host "Installed to $InstallDir"
    Write-Host "  bytebrew.exe     (CLI)"
    Write-Host "  bytebrew-srv.exe (Server)"
}
catch {
    Write-Error "Installation failed: $_"
    exit 1
}
finally {
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
}

# Check PATH
$UserPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
if ($UserPath -split ';' | Where-Object { $_ -eq $InstallDir }) {
    Write-Host ''
    Write-Host 'Ready! Run:'
    Write-Host '  bytebrew login    # authenticate with your account'
    Write-Host '  bytebrew          # start coding'
}
else {
    # Add to PATH automatically
    $NewPath = "$UserPath;$InstallDir"
    [Environment]::SetEnvironmentVariable('PATH', $NewPath, 'User')
    $env:PATH = "$env:PATH;$InstallDir"

    Write-Host ''
    Write-Host "Added $InstallDir to PATH."
    Write-Host 'Restart your terminal, then run:'
    Write-Host '  bytebrew login    # authenticate with your account'
    Write-Host '  bytebrew          # start coding'
}
