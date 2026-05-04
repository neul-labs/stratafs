# AgentFS Windows Build Script
# Run in PowerShell as Administrator

param(
    [switch]$Sign,
    [string]$CertThumbprint
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Resolve-Path "$ScriptDir\..\..\..\"
$BuildDir = "$ProjectRoot\build\windows"

# Get version from main.go
$VersionLine = Get-Content "$ProjectRoot\cmd\agentfs\main.go" | Select-String 'version = "'
$Version = ($VersionLine -split '"')[1]

Write-Host "Building AgentFS v$Version for Windows..." -ForegroundColor Green

# Create build directory
New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null

# Build main agentfs executable
Write-Host "Building agentfs.exe..."
Push-Location $ProjectRoot
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -tags "fts5" -o "$BuildDir\agentfs.exe" .\cmd\agentfs
Pop-Location

# Build service
Write-Host "Building agentfs-service.exe..."
Push-Location "$ProjectRoot\installers\windows\service"
go build -o "$BuildDir\agentfs-service.exe" .
Pop-Location

# Build tray app
Write-Host "Building agentfs-tray.exe..."
Push-Location "$ProjectRoot\installers\windows\tray"
go build -ldflags="-H windowsgui" -o "$BuildDir\agentfs-tray.exe" .
Pop-Location

# Build Wails UI
if (Test-Path "$ProjectRoot\desktop\agentfs-ui") {
    Write-Host "Building agentfs-ui.exe..."
    Push-Location "$ProjectRoot\desktop\agentfs-ui"
    wails build -platform windows/amd64 -o agentfs-ui.exe
    Copy-Item "build\bin\agentfs-ui.exe" "$BuildDir\"
    Pop-Location
}

# Download ONNX Runtime if not present
$OnnxUrl = "https://github.com/microsoft/onnxruntime/releases/download/v1.16.3/onnxruntime-win-x64-1.16.3.zip"
$OnnxZip = "$BuildDir\onnxruntime.zip"
$OnnxDll = "$BuildDir\onnxruntime.dll"

if (-not (Test-Path $OnnxDll)) {
    Write-Host "Downloading ONNX Runtime..."
    Invoke-WebRequest -Uri $OnnxUrl -OutFile $OnnxZip
    Expand-Archive -Path $OnnxZip -DestinationPath "$BuildDir\onnx-temp" -Force
    Copy-Item "$BuildDir\onnx-temp\onnxruntime-win-x64-1.16.3\lib\onnxruntime.dll" $OnnxDll
    Remove-Item -Recurse -Force "$BuildDir\onnx-temp"
    Remove-Item $OnnxZip
}

# Build shell extensions (requires Visual Studio)
$VsWhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path $VsWhere) {
    $VsPath = & $VsWhere -latest -property installationPath
    $MsBuild = "$VsPath\MSBuild\Current\Bin\MSBuild.exe"

    if (Test-Path $MsBuild) {
        Write-Host "Building shell extensions..."

        # Build context menu extension
        if (Test-Path "$ProjectRoot\installers\explorer\AgentFSContextMenu.vcxproj") {
            & $MsBuild "$ProjectRoot\installers\explorer\AgentFSContextMenu.vcxproj" `
                /p:Configuration=Release /p:Platform=x64
            Copy-Item "$ProjectRoot\installers\explorer\x64\Release\AgentFSContextMenu.dll" $BuildDir
        }

        # Build IFilter
        if (Test-Path "$ProjectRoot\installers\ifilter\AgentFSFilter.vcxproj") {
            & $MsBuild "$ProjectRoot\installers\ifilter\AgentFSFilter.vcxproj" `
                /p:Configuration=Release /p:Platform=x64
            Copy-Item "$ProjectRoot\installers\ifilter\x64\Release\AgentFSFilter.dll" $BuildDir
        }
    }
}

# Copy registry files
Copy-Item "$ProjectRoot\installers\explorer\AgentFSContextMenu.reg" $BuildDir
Copy-Item "$ProjectRoot\installers\ifilter\AgentFSFilter.reg" $BuildDir

# Sign executables if requested
if ($Sign -and $CertThumbprint) {
    Write-Host "Signing executables..."
    $FilesToSign = Get-ChildItem "$BuildDir\*.exe", "$BuildDir\*.dll"

    foreach ($File in $FilesToSign) {
        signtool sign /sha1 $CertThumbprint /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 $File.FullName
    }
}

# Build installer
$NsisPath = "${env:ProgramFiles(x86)}\NSIS\makensis.exe"
if (Test-Path $NsisPath) {
    Write-Host "Building installer..."
    & $NsisPath "$ScriptDir\installer.nsi"

    if ($Sign -and $CertThumbprint) {
        signtool sign /sha1 $CertThumbprint /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 "$BuildDir\AgentFS-Setup.exe"
    }
} else {
    Write-Host "NSIS not found. Skipping installer creation." -ForegroundColor Yellow
}

Write-Host "`nBuild complete!" -ForegroundColor Green
Write-Host "Output: $BuildDir"
