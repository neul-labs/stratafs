# AgentFS Installation Script for Windows PowerShell
# Usage: Invoke-WebRequest -Uri "https://raw.githubusercontent.com/yourusername/agentfs/main/scripts/install.ps1" -OutFile "install.ps1"; .\install.ps1

param(
    [string]$InstallDir = "$env:LOCALAPPDATA\AgentFS",
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
    Write-Output "AgentFS Installation Script for Windows"
    Write-Output ""
    Write-Output "Usage: .\install.ps1 [OPTIONS]"
    Write-Output ""
    Write-Output "Options:"
    Write-Output "  -InstallDir DIR    Installation directory (default: %LOCALAPPDATA%\AgentFS)"
    Write-Output "  -Version VERSION   AgentFS version to install (default: latest)"
    Write-Output "  -Force            Force installation even if already installed"
    Write-Output "  -AddToPath        Add installation directory to PATH (default: true)"
    Write-Output "  -Help             Show this help message"
    Write-Output ""
    Write-Output "Examples:"
    Write-Output "  .\install.ps1"
    Write-Output "  .\install.ps1 -InstallDir 'C:\Program Files\AgentFS'"
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
            $response = Invoke-RestMethod -Uri "https://api.github.com/repos/yourusername/agentfs/releases/latest"
            $script:Version = $response.tag_name
        }
        catch {
            Write-Error "Failed to fetch latest version: $_"
            exit 1
        }
    }
    Write-Info "Installing AgentFS version: $Version"
}

function Download-AgentFS {
    $architecture = Get-Architecture
    $downloadUrl = "https://github.com/yourusername/agentfs/releases/download/$Version/agentfs-$Version-windows-$architecture.zip"
    $tempDir = [System.IO.Path]::GetTempPath()
    $tempFile = Join-Path $tempDir "agentfs.zip"
    $extractDir = Join-Path $tempDir "agentfs-extract"

    Write-Info "Downloading AgentFS from: $downloadUrl"

    try {
        # Download the file
        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($downloadUrl, $tempFile)

        # Extract the archive
        Write-Info "Extracting AgentFS..."
        if (Test-Path $extractDir) {
            Remove-Item $extractDir -Recurse -Force
        }
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($tempFile, $extractDir)

        # Find the binary
        $binaryPath = Get-ChildItem -Path $extractDir -Name "agentfs.exe" -Recurse | Select-Object -First 1
        if (!$binaryPath) {
            throw "AgentFS binary not found in archive"
        }

        return Join-Path $extractDir $binaryPath.FullName
    }
    catch {
        Write-Error "Failed to download or extract AgentFS: $_"
        exit 1
    }
    finally {
        if (Test-Path $tempFile) {
            Remove-Item $tempFile -Force
        }
    }
}

function Install-AgentFS($binaryPath) {
    $installPath = Join-Path $InstallDir "agentfs.exe"

    # Check if already installed
    if ((Test-Path $installPath) -and !$Force) {
        Write-Warning "AgentFS is already installed at $installPath"
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

    Write-Info "Installing AgentFS to $installPath..."
    try {
        Copy-Item $binaryPath $installPath -Force
        Write-Success "AgentFS installed successfully!"
    }
    catch {
        Write-Error "Failed to install AgentFS: $_"
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
    $configDir = Join-Path $env:USERPROFILE ".agentfs"
    $configFile = Join-Path $configDir "config.json"

    if (!(Test-Path $configDir)) {
        Write-Info "Creating configuration directory: $configDir"
        New-Item -ItemType Directory -Path $configDir -Force | Out-Null
    }

    if (!(Test-Path $configFile)) {
        Write-Info "Creating default configuration..."
        $agentfsPath = Join-Path $InstallDir "agentfs.exe"
        try {
            & $agentfsPath config init
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

    $agentfsPath = Join-Path $InstallDir "agentfs.exe"

    if (!(Test-Path $agentfsPath)) {
        Write-Error "AgentFS binary not found at $agentfsPath"
        return
    }

    try {
        $version = & $agentfsPath --version 2>$null
        if ($version) {
            Write-Success "AgentFS $version is ready to use!"
        } else {
            Write-Success "AgentFS is installed and ready to use!"
        }
    }
    catch {
        Write-Warning "AgentFS installed but version check failed: $_"
    }

    Write-Info "Next steps:"
    Write-Output "  1. Open a new terminal (if PATH was updated)"
    Write-Output "  2. Initialize configuration: agentfs config init"
    Write-Output "  3. Add storage sources: agentfs source add"
    Write-Output "  4. Start AgentFS: agentfs"
    Write-Output ""
    Write-Output "For more help, run: agentfs --help"
}

function Main {
    if ($Help) {
        Show-Help
        return
    }

    Write-Info "AgentFS Installation Script for Windows"
    Write-Info "======================================"

    # Check if running with sufficient privileges for system-wide install
    if ($InstallDir -like "$env:ProgramFiles*" -and !(Test-AdminRights)) {
        Write-Warning "Installing to Program Files requires administrator privileges"
        Write-Info "Either run as administrator or choose a user directory"
        Write-Info "Using user directory: $env:LOCALAPPDATA\AgentFS"
        $script:InstallDir = "$env:LOCALAPPDATA\AgentFS"
    }

    Get-LatestVersion
    $binaryPath = Download-AgentFS
    Install-AgentFS $binaryPath
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