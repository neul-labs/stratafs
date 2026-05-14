# StrataFS Installation Script for Windows PowerShell
# Usage: Invoke-WebRequest -Uri "https://raw.githubusercontent.com/neul-labs/stratafs/main/scripts/install.ps1" -OutFile "install.ps1"; .\install.ps1

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\StrataFS",
    [string]$Version = "latest",
    [switch]$Force = $false,
    [switch]$AddToPath = $true,
    [switch]$Help = $false
)

# Color functions
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    } else {
        $input | Write-Output
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

function Write-Info($message) {
    Write-ColorOutput Blue "[INFO] $message"
}

function Write-Success($message) {
    Write-ColorOutput Green "[SUCCESS] $message"
}

function Write-Warning($message) {
    Write-ColorOutput Yellow "[WARNING] $message"
}

function Write-Error($message) {
    Write-ColorOutput Red "[ERROR] $message"
}

function Show-Help {
    Write-Output "StrataFS Installation Script for Windows"
    Write-Output ""
    Write-Output "Usage: .\install.ps1 [OPTIONS]"
    Write-Output ""
    Write-Output "Options:"
    Write-Output "  -InstallDir DIR    Installation directory (default: %LOCALAPPDATA%\StrataFS)"
    Write-Output "  -Version VERSION   StrataFS version to install (default: latest)"
    Write-Output "  -Force            Force installation even if already installed"
    Write-Output "  -AddToPath        Add installation directory to PATH (default: true)"
    Write-Output "  -Help             Show this help message"
    Write-Output ""
    Write-Output "Examples:"
    Write-Output "  .\install.ps1"
    Write-Output "  .\install.ps1 -InstallDir 'C:\Program Files\StrataFS'"
    Write-Output "  .\install.ps1 -Version 'v0.2.0' -Force"
}

function Test-AdminRights {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86" { return "386" }
        default {
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

function Get-LatestVersion {
    if ($Version -eq "latest") {
        Write-Info "Fetching latest version..."
        try {
            $response = Invoke-RestMethod -Uri "https://api.github.com/repos/neul-labs/stratafs/releases/latest"
            $script:Version = $response.tag_name
        }
        catch {
            Write-Error "Failed to fetch latest version: $_"
            exit 1
        }
    }
    Write-Info "Installing StrataFS version: $Version"
}

function Download-StrataFS {
    $architecture = Get-Architecture
    $downloadUrl = "https://github.com/neul-labs/stratafs/releases/download/$Version/stratafs-$Version-windows-$architecture.zip"
    $tempDir = [System.IO.Path]::GetTempPath()
    $tempFile = Join-Path $tempDir "stratafs.zip"
    $extractDir = Join-Path $tempDir "stratafs-extract"

    Write-Info "Downloading StrataFS from: $downloadUrl"

    try {
        # Download the file
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($downloadUrl, $tempFile)

        # Extract the archive
        Write-Info "Extracting StrataFS..."
        if (Test-Path $extractDir) {
            Remove-Item $extractDir -Recurse -Force
        }
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($tempFile, $extractDir)

        # Find the binary
        $binaryPath = Get-ChildItem -Path $extractDir -Name "stratafs.exe" -Recurse | Select-Object -First 1
        if (!$binaryPath) {
            throw "StrataFS binary not found in archive"
        }

        return Join-Path $extractDir $binaryPath.FullName
    }
    catch {
        Write-Error "Failed to download or extract StrataFS: $_"
        exit 1
    }
    finally {
        if (Test-Path $tempFile) {
            Remove-Item $tempFile -Force
        }
    }
}

function Install-StrataFS($binaryPath) {
    $installPath = Join-Path $InstallDir "stratafs.exe"

    # Check if already installed
    if ((Test-Path $installPath) -and !$Force) {
        Write-Warning "StrataFS is already installed at $installPath"
        $response = Read-Host "Do you want to overwrite it? (y/N)"
        if ($response -notmatch "^[Yy]$") {
            Write-Info "Installation cancelled"
            exit 0
        }
    }

    # Create install directory if it doesn't exist
    if (!(Test-Path $InstallDir)) {
        Write-Info "Creating install directory: $InstallDir"
        try {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        catch {
            Write-Error "Failed to create install directory: $_"
            exit 1
        }
    }

    Write-Info "Installing StrataFS to $installPath..."
    try {
        Copy-Item $binaryPath $installPath -Force
        Write-Success "StrataFS installed successfully!"
    }
    catch {
        Write-Error "Failed to install StrataFS: $_"
        exit 1
    }
}

function Add-ToPath($directory) {
    if (!$AddToPath) {
        return
    }

    Write-Info "Adding $directory to PATH..."

    # Get current PATH
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")

    # Check if directory is already in PATH
    if ($currentPath -split ";" | Where-Object { $_ -eq $directory }) {
        Write-Info "Directory already in PATH"
        return
    }

    # Add to PATH
    try {
        $newPath = if ($currentPath) { "$currentPath;$directory" } else { $directory }
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-Success "Added to PATH. You may need to restart your terminal."
    }
    catch {
        Write-Warning "Failed to add to PATH: $_"
        Write-Info "You can manually add $directory to your PATH environment variable"
    }
}

function Setup-Config {
    $configDir = Join-Path $env:USERPROFILE ".stratafs"
    $configFile = Join-Path $configDir "config.json"

    if (!(Test-Path $configDir)) {
        Write-Info "Creating configuration directory: $configDir"
        New-Item -ItemType Directory -Path $configDir -Force | Out-Null
    }

    if (!(Test-Path $configFile)) {
        Write-Info "Creating default configuration..."
        $stratafsPath = Join-Path $InstallDir "stratafs.exe"
        try {
            & $stratafsPath config init
            Write-Success "Default configuration created at $configFile"
        }
        catch {
            Write-Warning "Failed to create default configuration: $_"
        }
    } else {
        Write-Info "Configuration already exists at $configFile"
    }
}

function Test-Installation {
    Write-Info "Verifying installation..."

    $stratafsPath = Join-Path $InstallDir "stratafs.exe"

    if (!(Test-Path $stratafsPath)) {
        Write-Error "StrataFS binary not found at $stratafsPath"
        return
    }

    try {
        $version = & $stratafsPath --version 2>$null
        if ($version) {
            Write-Success "StrataFS $version is ready to use!"
        } else {
            Write-Success "StrataFS is installed and ready to use!"
        }
    }
    catch {
        Write-Warning "StrataFS installed but version check failed: $_"
    }

    Write-Info "Next steps:"
    Write-Output "  1. Open a new terminal (if PATH was updated)"
    Write-Output "  2. Initialize configuration: stratafs config init"
    Write-Output "  3. Add storage sources: stratafs source add"
    Write-Output "  4. Start StrataFS: stratafs"
    Write-Output ""
    Write-Output "For more help, run: stratafs --help"
}

function Main {
    if ($Help) {
        Show-Help
        return
    }

    Write-Info "StrataFS Installation Script for Windows"
    Write-Info "======================================"

    # Check if running with sufficient privileges for system-wide install
    if ($InstallDir -like "$env:ProgramFiles*" -and !(Test-AdminRights)) {
        Write-Warning "Installing to Program Files requires administrator privileges"
        Write-Info "Either run as administrator or choose a user directory"
        Write-Info "Using user directory: $env:LOCALAPPDATA\StrataFS"
        $script:InstallDir = "$env:LOCALAPPDATA\StrataFS"
    }

    Get-LatestVersion
    $binaryPath = Download-StrataFS
    Install-StrataFS $binaryPath
    Add-ToPath $InstallDir
    Setup-Config
    Test-Installation

    Write-Success "Installation completed!"
}

# Execute main function
try {
    Main
}
catch {
    Write-Error "Installation failed: $_"
    exit 1
}