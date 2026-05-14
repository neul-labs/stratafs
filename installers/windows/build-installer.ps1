# StrataFS Windows Installer Build Script
param(
    [string]$Version = "0.2.0",
    [string]$BinaryPath = "../../build/windows-amd64/stratafs.exe",
    [string]$OutputDir = "dist",
    [switch]$Sign = $false,
    [string]$SignCert = "",
    [string]$SignPassword = ""
)

# Configuration
$ErrorActionPreference = "Stop"
$InformationPreference = "Continue"

# Paths
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Resolve-Path "$ScriptDir/../.."
$InstallerDir = $ScriptDir
$BuildDir = Join-Path $InstallerDir "build"
$AssetsDir = Join-Path $InstallerDir "assets"

Write-Information "Building StrataFS Windows Installer v$Version"

# Check NSIS installation
$NSISPath = "${env:ProgramFiles(x86)}\NSIS\makensis.exe"
if (-not (Test-Path $NSISPath)) {
    $NSISPath = "${env:ProgramFiles}\NSIS\makensis.exe"
    if (-not (Test-Path $NSISPath)) {
        Write-Error "NSIS not found. Please install NSIS from https://nsis.sourceforge.io/"
        exit 1
    }
}

Write-Information "Found NSIS at: $NSISPath"

# Clean and create build directory
if (Test-Path $BuildDir) {
    Remove-Item $BuildDir -Recurse -Force
}
New-Item -ItemType Directory -Path $BuildDir -Force | Out-Null
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null

# Copy binary
if (-not (Test-Path $BinaryPath)) {
    Write-Error "Binary not found at: $BinaryPath"
    Write-Information "Please build the Windows binary first:"
    Write-Information "  go build -tags 'fts5' -o build/windows-amd64/stratafs.exe ./cmd/stratafs"
    exit 1
}

Copy-Item $BinaryPath $BuildDir
Write-Information "Copied binary: $BinaryPath"

# Copy ONNX Runtime DLL (if available)
$OnnxDllPath = Join-Path (Split-Path $BinaryPath) "onnxruntime.dll"
if (Test-Path $OnnxDllPath) {
    Copy-Item $OnnxDllPath $BuildDir
    Write-Information "Copied ONNX Runtime DLL"
} else {
    Write-Warning "ONNX Runtime DLL not found, installer will download separately"
}

# Copy documentation
$DocsFiles = @("README.md", "LICENSE")
foreach ($doc in $DocsFiles) {
    $srcPath = Join-Path $ProjectRoot $doc
    if (Test-Path $srcPath) {
        $destName = if ($doc -eq "README.md") { "README.txt" } else { "$doc.txt" }
        Copy-Item $srcPath (Join-Path $BuildDir $destName)
    }
}

# Create default configuration
$ConfigDir = Join-Path $BuildDir "config"
New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null

$DefaultConfig = @{
    version = "0.2.0"
    agent_dir = ".stratafs"
    global_dir = "%APPDATA%\StrataFS"
    sources = @()
    server = @{
        api_port = 8080
        mcp_port = 8081
    }
    worker = @{
        count = 4
        scan_interval = "30s"
        batch_size = 10
    }
    embedding = @{
        model = "bge-base-en-v1.5"
        cache_dir = "%APPDATA%\StrataFS\fastembed_cache"
        dimension = 768
    }
    database = @{
        compression_enabled = $true
        compression_threshold = 512
        maintenance_interval = "24h"
        deleted_threshold = "168h"
    }
    chunking = @{
        default_strategy = "simple"
        chunk_size = 1000
        overlap_size = 100
        min_chunk_size = 50
    }
}

$DefaultConfig | ConvertTo-Json -Depth 10 | Out-File (Join-Path $ConfigDir "default.json") -Encoding UTF8

# Create/copy assets
if (-not (Test-Path $AssetsDir)) {
    New-Item -ItemType Directory -Path $AssetsDir -Force | Out-Null
}

# Create basic icon (placeholder - should be replaced with actual icon)
$IconPath = Join-Path $AssetsDir "stratafs.ico"
if (-not (Test-Path $IconPath)) {
    Write-Warning "Icon not found at $IconPath, using default"
    # Create a simple ICO file (this would normally be a proper icon)
    $EmptyIcon = @()
    [System.IO.File]::WriteAllBytes($IconPath, $EmptyIcon)
}

# Copy assets to build directory
Copy-Item $IconPath $BuildDir

# Prepare NSIS script variables
$NSISVars = @{
    "VERSION" = $Version
    "BUILD_DIR" = $BuildDir
    "OUTPUT_DIR" = $OutputDir
}

# Build installer with NSIS
Write-Information "Building installer with NSIS..."

$NSISArgs = @(
    "/DVERSION=$Version"
    "/DBUILD_DIR=$BuildDir"
    "/DOUTPUT_DIR=$OutputDir"
    (Join-Path $InstallerDir "stratafs.nsi")
)

$Process = Start-Process -FilePath $NSISPath -ArgumentList $NSISArgs -Wait -PassThru -NoNewWindow
if ($Process.ExitCode -ne 0) {
    Write-Error "NSIS build failed with exit code: $($Process.ExitCode)"
    exit 1
}

$InstallerPath = Join-Path $OutputDir "StrataFS-$Version-Setup.exe"
if (-not (Test-Path $InstallerPath)) {
    Write-Error "Installer was not created at expected path: $InstallerPath"
    exit 1
}

Write-Information "Installer created: $InstallerPath"

# Code signing (if requested)
if ($Sign) {
    if (-not $SignCert) {
        Write-Error "Code signing requested but no certificate specified"
        exit 1
    }

    Write-Information "Signing installer..."

    $SignToolPath = "${env:ProgramFiles(x86)}\Windows Kits\10\bin\*\x64\signtool.exe"
    $SignTool = Get-ChildItem $SignToolPath | Sort-Object Name -Descending | Select-Object -First 1

    if (-not $SignTool) {
        Write-Error "SignTool not found. Please install Windows SDK."
        exit 1
    }

    $SignArgs = @(
        "sign"
        "/f", $SignCert
        "/fd", "SHA256"
        "/tr", "http://timestamp.digicert.com"
        "/td", "SHA256"
        $InstallerPath
    )

    if ($SignPassword) {
        $SignArgs += @("/p", $SignPassword)
    }

    $SignProcess = Start-Process -FilePath $SignTool.FullName -ArgumentList $SignArgs -Wait -PassThru -NoNewWindow
    if ($SignProcess.ExitCode -ne 0) {
        Write-Error "Code signing failed with exit code: $($SignProcess.ExitCode)"
        exit 1
    }

    Write-Information "Installer signed successfully"
}

# Generate checksum
$Hash = Get-FileHash $InstallerPath -Algorithm SHA256
$ChecksumFile = "$InstallerPath.sha256"
"$($Hash.Hash)  $(Split-Path $InstallerPath -Leaf)" | Out-File $ChecksumFile -Encoding ASCII

Write-Information "Generated checksum: $ChecksumFile"

# Display results
Write-Information ""
Write-Information "Build completed successfully!"
Write-Information "Installer: $InstallerPath"
Write-Information "Size: $([math]::Round((Get-Item $InstallerPath).Length / 1MB, 2)) MB"
Write-Information "SHA256: $($Hash.Hash)"

# Test installer (basic validation)
Write-Information ""
Write-Information "Running installer validation..."

$TestArgs = @("/S", "/D=C:\Temp\StrataFSTest")
$TestProcess = Start-Process -FilePath $InstallerPath -ArgumentList $TestArgs -Wait -PassThru -NoNewWindow

if ($TestProcess.ExitCode -eq 0) {
    Write-Information "Installer validation passed"
    # Cleanup test installation
    if (Test-Path "C:\Temp\StrataFSTest") {
        Remove-Item "C:\Temp\StrataFSTest" -Recurse -Force -ErrorAction SilentlyContinue
    }
} else {
    Write-Warning "Installer validation failed with exit code: $($TestProcess.ExitCode)"
}

Write-Information ""
Write-Information "Installation instructions:"
Write-Information "1. Run StrataFS-$Version-Setup.exe as Administrator"
Write-Information "2. Follow the installation wizard"
Write-Information "3. Choose desktop application or Windows service mode"
Write-Information "4. StrataFS will be available in Start Menu and Desktop (if selected)"