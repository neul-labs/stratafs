# AgentFS Installation Guide

Complete guide for installing AgentFS across all supported platforms, including both server deployments and desktop installations.

## Overview

AgentFS provides multiple installation methods to suit different use cases:

- **Server Deployments**: Docker, Kubernetes, cloud environments
- **Desktop Applications**: Native installers for Windows, macOS, and Linux
- **Development**: Source builds and development environments

## Quick Start

### Docker (Recommended for Servers)

```bash
# Pull and run the latest image
docker run -d \
  --name agentfs \
  -p 8080:8080 \
  -p 8081:8081 \
  -v agentfs-data:/data \
  agentfs:latest

# Access the web interface
open http://localhost:8080
```

### Desktop Installation

#### Windows
1. Download `AgentFS-Setup.exe` from releases
2. Run the installer as Administrator
3. Choose installation options (Service or Application)
4. Follow the setup wizard

#### macOS
1. Download `AgentFS-{version}.pkg` from releases
2. Double-click to run the installer
3. Follow the installation prompts
4. AgentFS will start automatically

#### Linux
```bash
# Ubuntu/Debian (DEB package)
wget https://github.com/your-repo/agentfs/releases/download/v0.2.0/agentfs_0.2.0_amd64.deb
sudo apt install ./agentfs_0.2.0_amd64.deb

# Universal (AppImage)
wget https://github.com/your-repo/agentfs/releases/download/v0.2.0/AgentFS-0.2.0-x86_64.AppImage
chmod +x AgentFS-0.2.0-x86_64.AppImage
./AgentFS-0.2.0-x86_64.AppImage
```

## Detailed Installation Instructions

### Server Deployments

#### Docker

##### Basic Setup
```bash
# Create a data volume
docker volume create agentfs-data

# Run AgentFS
docker run -d \
  --name agentfs \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 8081:8081 \
  -v agentfs-data:/data \
  -v /path/to/config:/app/config \
  -e AGENTFS_CONFIG_DIR=/app/config \
  agentfs:latest
```

##### Production Setup
```bash
# Create network
docker network create agentfs-network

# Run with custom configuration
docker run -d \
  --name agentfs \
  --network agentfs-network \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 8081:8081 \
  -v agentfs-data:/data \
  -v agentfs-config:/app/config \
  -v /var/log/agentfs:/app/logs \
  -e AGENTFS_LOG_LEVEL=info \
  -e AGENTFS_CONFIG_DIR=/app/config \
  --memory=1g \
  --cpus=1.0 \
  agentfs:latest
```

#### Docker Compose

Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  agentfs:
    image: agentfs:latest
    container_name: agentfs
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - agentfs-data:/data
      - agentfs-config:/app/config
    environment:
      - AGENTFS_CONFIG_DIR=/app/config
      - AGENTFS_LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  agentfs-data:
  agentfs-config:
```

Run with:
```bash
docker-compose up -d
```

#### Kubernetes

##### Using Helm
```bash
# Add the helm repository (replace with actual repo)
helm repo add agentfs https://your-repo.com/helm-charts
helm repo update

# Install AgentFS
helm install agentfs agentfs/agentfs \
  --namespace agentfs \
  --create-namespace \
  --set image.tag=0.2.0 \
  --set persistence.enabled=true \
  --set persistence.size=10Gi
```

##### Manual Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentfs
  namespace: agentfs
spec:
  replicas: 1
  selector:
    matchLabels:
      app: agentfs
  template:
    metadata:
      labels:
        app: agentfs
    spec:
      containers:
      - name: agentfs
        image: agentfs:0.2.0
        ports:
        - containerPort: 8080
        - containerPort: 8081
        env:
        - name: AGENTFS_CONFIG_DIR
          value: "/app/config"
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /app/config
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: agentfs-data
      - name: config
        configMap:
          name: agentfs-config
```

### Desktop Installations

#### Windows Installation

##### System Requirements
- Windows 10 or later
- .NET Framework 4.7.2 or later
- 100 MB free disk space
- Administrator privileges (for service installation)

##### Installation Steps
1. Download the installer from the releases page
2. Right-click and "Run as Administrator"
3. Accept the license agreement
4. Choose installation directory (default: `C:\Program Files\AgentFS`)
5. Select installation type:
   - **Service Installation**: Runs as Windows service (recommended)
   - **Application Installation**: Runs as regular application
6. Choose startup options:
   - **Start with Windows**: Add to startup programs
   - **Desktop Shortcut**: Create desktop shortcut
   - **Start Menu**: Add to Start Menu
7. Click "Install" and wait for completion

##### Post-Installation
- **Configuration**: Located at `%USERPROFILE%\.agentfs\config.json`
- **Logs**: Located at `%USERPROFILE%\.agentfs\logs\`
- **Service Management**: Use Windows Services (`services.msc`)
- **System Tray**: AgentFS icon in system tray for quick access

##### Uninstallation
- Use "Add or Remove Programs" in Windows Settings
- Or run the uninstaller from the installation directory

#### macOS Installation

##### System Requirements
- macOS 10.15 (Catalina) or later
- 100 MB free disk space
- Administrator privileges

##### Installation Steps
1. Download the `.pkg` file from releases
2. Double-click the downloaded file
3. Follow the installation wizard:
   - Introduction and license
   - Installation type selection
   - Administrator password entry
4. AgentFS will be installed and started automatically

##### Post-Installation
- **Configuration**: Located at `~/.agentfs/config.json`
- **Logs**: Located at `~/.agentfs/logs/`
- **LaunchAgent**: Automatically configured for background operation
- **Menu Bar**: AgentFS icon in menu bar for quick access

##### Uninstallation
```bash
# Stop the service
launchctl unload ~/Library/LaunchAgents/com.agentfs.plist

# Remove files
sudo rm -rf /Applications/AgentFS.app
rm -rf ~/.agentfs
rm ~/Library/LaunchAgents/com.agentfs.plist
```

#### Linux Installation

##### System Requirements
- 64-bit Linux distribution
- GLIBC 2.17 or later
- 100 MB free disk space
- Desktop environment (for GUI features)

##### Option 1: DEB Package (Debian/Ubuntu)
```bash
# Download the DEB package
wget https://github.com/your-repo/agentfs/releases/download/v0.2.0/agentfs_0.2.0_amd64.deb

# Install using apt
sudo apt update
sudo apt install ./agentfs_0.2.0_amd64.deb

# Or using dpkg
sudo dpkg -i agentfs_0.2.0_amd64.deb
sudo apt-get install -f  # Fix dependencies if needed
```

##### Option 2: AppImage (Universal)
```bash
# Download AppImage
wget https://github.com/your-repo/agentfs/releases/download/v0.2.0/AgentFS-0.2.0-x86_64.AppImage

# Make executable
chmod +x AgentFS-0.2.0-x86_64.AppImage

# Run directly
./AgentFS-0.2.0-x86_64.AppImage

# Install desktop integration (optional)
./AgentFS-0.2.0-x86_64.AppImage --appimage-extract
./squashfs-root/usr/bin/install-desktop-integration.sh
```

##### Option 3: Source Build
```bash
# Install dependencies
sudo apt update
sudo apt install golang-go git build-essential

# Clone and build
git clone https://github.com/your-repo/agentfs.git
cd agentfs
make build

# Install manually
sudo cp build/agentfs /usr/local/bin/
mkdir -p ~/.agentfs
agentfs config init
```

##### Post-Installation
- **Configuration**: Located at `~/.agentfs/config.json`
- **Service**: Managed via systemd (`systemctl --user`)
- **Desktop Entry**: Added to application menu
- **System Tray**: Available in system tray

##### Service Management
```bash
# Start service
systemctl --user start agentfs

# Enable auto-start
systemctl --user enable agentfs

# Check status
systemctl --user status agentfs

# View logs
journalctl --user -u agentfs -f
```

## Cross-Platform Desktop Integration

### System Tray Application

AgentFS includes a cross-platform system tray application (`agentfs-tray.py`) that provides:

- Start/Stop/Restart controls
- Status monitoring
- Web interface access
- Configuration management
- Update notifications

#### Running the System Tray
```bash
# Install Python dependencies
pip install pystray pillow

# Run the system tray application
python3 installers/desktop/agentfs-tray.py
```

### Universal Launcher

The universal launcher script (`agentfs-launcher.sh`) provides:

- Cross-platform service management
- GUI notifications
- Web interface launching
- Configuration initialization

#### Using the Launcher
```bash
# Make executable
chmod +x installers/desktop/agentfs-launcher.sh

# Start AgentFS
./installers/desktop/agentfs-launcher.sh start

# Show status
./installers/desktop/agentfs-launcher.sh status

# Open web interface
./installers/desktop/agentfs-launcher.sh web
```

## Configuration

### Initial Setup

After installation, AgentFS needs to be configured:

1. **Initialize Configuration**:
   ```bash
   agentfs config init
   ```

2. **Add Data Sources**:
   ```bash
   agentfs source add /path/to/your/documents
   agentfs source add /path/to/your/projects
   ```

3. **Start the Service**:
   ```bash
   agentfs start
   # Or use the system service manager
   ```

4. **Access Web Interface**:
   Open http://localhost:8080 in your browser

### Configuration File

The main configuration file is located at:
- Windows: `%USERPROFILE%\.agentfs\config.json`
- macOS/Linux: `~/.agentfs/config.json`

Example configuration:
```json
{
  "version": "0.2.0",
  "sources": [
    {
      "path": "/Users/username/Documents",
      "type": "filesystem",
      "enabled": true
    }
  ],
  "server": {
    "api_port": 8080,
    "mcp_port": 8081,
    "host": "127.0.0.1"
  },
  "embedding": {
    "model": "bge-base-en-v1.5",
    "cache_dir": "~/.agentfs/fastembed_cache"
  }
}
```

## Troubleshooting

### Common Issues

#### Port Already in Use
```bash
# Check what's using the port
netstat -tulpn | grep :8080
lsof -i :8080

# Change port in configuration
# Edit ~/.agentfs/config.json
```

#### Service Won't Start
```bash
# Check logs
# Linux: journalctl --user -u agentfs
# macOS: cat ~/.agentfs/logs/agentfs.log
# Windows: Check Event Viewer

# Check configuration
agentfs config validate

# Reset configuration
agentfs config reset
```

#### Permission Issues
```bash
# Fix file permissions (Linux/macOS)
chmod -R 755 ~/.agentfs/
chmod 600 ~/.agentfs/config.json

# Windows: Run as Administrator or check folder permissions
```

### Getting Help

1. **Check Documentation**: Review the configuration guide
2. **View Logs**: Check application logs for error messages
3. **Validate Config**: Use `agentfs config validate`
4. **Reset Config**: Use `agentfs config reset` to start fresh
5. **GitHub Issues**: Report bugs or ask questions on GitHub

### Debug Mode

Enable debug logging for troubleshooting:

```bash
# Environment variable
export AGENTFS_LOG_LEVEL=debug

# Configuration file
{
  "log_level": "debug"
}

# Command line
agentfs --log-level=debug
```

## Updates and Maintenance

### Automatic Updates

Desktop installations include automatic update checking:

```bash
# Check for updates
./installers/desktop/auto-updater.sh check

# Enable/disable auto-updates
./installers/desktop/auto-updater.sh enable
./installers/desktop/auto-updater.sh disable
```

### Manual Updates

#### Docker
```bash
# Pull latest image
docker pull agentfs:latest

# Recreate container
docker-compose down
docker-compose up -d
```

#### Desktop Applications
1. Download new installer from releases
2. Run installer (will upgrade existing installation)
3. Restart AgentFS service

### Backup and Restore

#### Backup Configuration
```bash
# Backup entire config directory
tar -czf agentfs-backup.tar.gz ~/.agentfs/

# Backup just configuration
cp ~/.agentfs/config.json ~/agentfs-config-backup.json
```

#### Restore Configuration
```bash
# Restore from backup
tar -xzf agentfs-backup.tar.gz -C ~/

# Restart service
systemctl --user restart agentfs
```

This comprehensive installation guide covers all supported platforms and deployment scenarios for AgentFS.