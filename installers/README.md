# StrataFS Installers

This directory contains all the installation scripts and packages for deploying StrataFS across different platforms and environments.

## Directory Structure

```
installers/
├── windows/                    # Windows installers
│   ├── stratafs.nsi            # NSIS installer script
│   ├── stratafs-service.xml    # Windows service configuration
│   └── README.md              # Windows-specific documentation
├── macos/                     # macOS installers
│   ├── build-pkg.sh           # PKG installer builder
│   ├── Info.plist             # App bundle info
│   ├── LaunchAgent.plist      # Launch agent configuration
│   └── README.md              # macOS-specific documentation
├── linux/                    # Linux installers
│   ├── build-appimage.sh      # AppImage builder
│   ├── build-deb.sh           # Debian package builder
│   ├── control               # DEB package control file
│   ├── stratafs.service        # Systemd service file
│   └── README.md              # Linux-specific documentation
├── desktop/                   # Cross-platform desktop integration
│   ├── stratafs-launcher.sh    # Universal launcher script
│   ├── stratafs-tray.py        # System tray application
│   ├── stratafs.desktop        # Linux desktop entry
│   ├── auto-updater.sh        # Auto-update mechanism
│   └── README.md              # Desktop integration documentation
└── README.md                  # This file
```

## Quick Start

### Server Deployment

```bash
# Docker
docker build -t stratafs .
docker run -d -p 8080:8080 -p 8081:8081 stratafs

# Docker Compose
docker-compose up -d

# Kubernetes
helm install stratafs ./helm/stratafs
```

### Desktop Installation

#### Windows
```cmd
# Download and run the installer
StrataFS-Setup.exe

# Or build from source
makensis stratafs.nsi
```

#### macOS
```bash
# Build PKG installer
cd installers/macos
./build-pkg.sh

# Install
sudo installer -pkg dist/StrataFS-0.2.0.pkg -target /
```

#### Linux
```bash
# Build AppImage
cd installers/linux
./build-appimage.sh

# Or build DEB package
./build-deb.sh
sudo apt install ./dist/stratafs_0.2.0_amd64.deb
```

## Installation Types

### 1. Server Deployments

**Target**: Production servers, cloud environments, containers
**Features**:
- High availability
- Monitoring integration
- Kubernetes support
- Load balancing
- Service discovery

**Components**:
- Docker containers
- Kubernetes Helm charts
- Docker Compose configurations
- CI/CD workflows

### 2. Desktop Applications

**Target**: End-user workstations, development environments
**Features**:
- GUI system tray
- Desktop integration
- Auto-updates
- Service management
- Native notifications

**Components**:
- Platform-specific installers
- System service configuration
- Desktop integration files
- Cross-platform launcher scripts

## Platform-Specific Features

### Windows
- **NSIS Installer**: Professional Windows installer with GUI
- **Windows Service**: Background service with automatic startup
- **System Tray**: Native Windows system tray integration
- **Registry Integration**: Proper Windows registry configuration
- **Auto-updates**: Windows-compatible update mechanism

### macOS
- **PKG Installer**: Native macOS installer package
- **App Bundle**: Proper macOS application structure
- **LaunchAgent**: Background service management
- **Menu Bar**: Native macOS menu bar integration
- **Code Signing**: Support for Apple code signing and notarization

### Linux
- **Multiple Formats**: AppImage (portable), DEB (Debian/Ubuntu), RPM (planned)
- **Systemd Integration**: Native Linux service management
- **Desktop Environment**: FreeDesktop.org compliant desktop integration
- **Package Managers**: Integration with system package managers
- **Universal Launcher**: Works across different Linux distributions

## Cross-Platform Components

### System Tray Application (`stratafs-tray.py`)

A Python-based system tray application that provides:
- Start/stop/restart controls
- Status monitoring
- Web interface access
- Configuration management
- Cross-platform compatibility (Windows, macOS, Linux)

**Dependencies**:
- `pystray` (preferred)
- `tkinter` (fallback)
- `PIL` (for icons)

### Universal Launcher (`stratafs-launcher.sh`)

A bash script that provides:
- Platform detection
- Service management
- GUI notifications
- Configuration initialization
- Web interface launching

**Features**:
- Works on all UNIX-like systems
- Native GUI dialogs where available
- Fallback to command-line interface
- Environment variable configuration

### Auto-Updater (`auto-updater.sh`)

Automatic update mechanism featuring:
- GitHub release monitoring
- Platform-specific download and installation
- User consent for updates
- Configuration-based enable/disable
- Background update checking

## Build Requirements

### Windows
- **NSIS**: Nullsoft Scriptable Install System
- **Go**: For building the binary
- **Optional**: Code signing certificate

### macOS
- **Xcode Command Line Tools**: For building
- **Go**: For building the binary
- **Optional**: Apple Developer certificate for code signing

### Linux
- **Go**: For building the binary
- **AppImageTool**: For AppImage creation
- **dpkg-deb**: For DEB package creation
- **Standard build tools**: make, gcc, etc.

### Cross-Platform
- **Docker**: For container builds
- **Git**: For version control and CI/CD
- **Python 3**: For desktop integration scripts

## Configuration Management

All installers support configuration through:

1. **Environment Variables**:
   ```bash
   STRATAFS_CONFIG_DIR=/path/to/config
   STRATAFS_API_PORT=8080
   STRATAFS_MCP_PORT=8081
   ```

2. **Configuration File** (`config.json`):
   ```json
   {
     "server": {
       "api_port": 8080,
       "mcp_port": 8081
     },
     "auto_update": true
   }
   ```

3. **Command Line Arguments**:
   ```bash
   stratafs --config-dir=/path/to/config --api-port=8080
   ```

## Security Considerations

### Code Signing
- **Windows**: Authenticode signing for executables and installers
- **macOS**: Apple code signing and notarization
- **Linux**: GPG signing for packages

### Permissions
- **Minimal Privileges**: Run with least required permissions
- **User Installation**: Support for user-level installation
- **Service Isolation**: Proper service account configuration

### Network Security
- **TLS Support**: HTTPS/TLS for web interfaces
- **Firewall Configuration**: Documented port requirements
- **Access Control**: Authentication and authorization options

## Testing

### Automated Testing
- **Unit Tests**: Go test suite
- **Integration Tests**: Docker-based testing
- **Platform Tests**: CI/CD pipeline testing on multiple platforms

### Manual Testing
- **Installation Testing**: Verify installers work correctly
- **Upgrade Testing**: Test update mechanisms
- **Uninstall Testing**: Ensure clean removal

## Distribution

### Release Process
1. **Version Tagging**: Semantic versioning (v0.2.0)
2. **Asset Building**: Automated builds via GitHub Actions
3. **Testing**: Automated and manual testing
4. **Publishing**: Release creation with assets
5. **Distribution**: Package repository updates

### Release Assets
- Windows: `StrataFS-{version}-Setup.exe`
- macOS: `StrataFS-{version}-{arch}.pkg`
- Linux AppImage: `StrataFS-{version}-x86_64.AppImage`
- Linux DEB: `stratafs_{version}_{arch}.deb`
- Docker Images: `stratafs:{version}`

## Support and Maintenance

### Documentation
- Platform-specific installation guides
- Troubleshooting documentation
- Configuration reference
- API documentation

### Issue Tracking
- GitHub Issues for bug reports
- Feature requests and enhancements
- Security vulnerability reporting
- Community support

### Maintenance
- Regular security updates
- Dependency updates
- Platform compatibility updates
- Performance improvements

## Contributing

### Adding New Platforms
1. Create platform-specific directory
2. Implement installer script
3. Add platform detection to universal launcher
4. Update CI/CD pipeline
5. Add documentation and tests

### Improving Existing Installers
1. Follow existing patterns and conventions
2. Maintain backward compatibility
3. Add comprehensive testing
4. Update documentation
5. Consider security implications

For detailed platform-specific instructions, see the README.md files in each subdirectory.