# AgentFS Deployment Guide

This guide covers all deployment options for AgentFS, including server infrastructure and desktop installations.

## Table of Contents

1. [Server Deployment](#server-deployment)
   - [Docker](#docker)
   - [Docker Compose](#docker-compose)
   - [Kubernetes (Helm)](#kubernetes-helm)
2. [Desktop Installation](#desktop-installation)
   - [Windows](#windows)
   - [macOS](#macos)
   - [Linux](#linux)
3. [Configuration](#configuration)
4. [Monitoring & Maintenance](#monitoring--maintenance)
5. [Troubleshooting](#troubleshooting)

## Server Deployment

### Docker

#### Quick Start

```bash
# Build the image
docker build -t agentfs:latest .

# Run with default configuration
docker run -d \
  --name agentfs \
  -p 8080:8080 \
  -p 8081:8081 \
  -v agentfs-data:/data \
  agentfs:latest

# Run with custom configuration
docker run -d \
  --name agentfs \
  -p 8080:8080 \
  -p 8081:8081 \
  -v /path/to/config:/app/config \
  -v /path/to/data:/data \
  -e AGENTFS_CONFIG_DIR=/app/config \
  agentfs:latest
```

#### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTFS_CONFIG_DIR` | `/app/config` | Configuration directory |
| `AGENTFS_DATA_DIR` | `/data` | Data storage directory |
| `AGENTFS_API_PORT` | `8080` | API server port |
| `AGENTFS_MCP_PORT` | `8081` | MCP server port |
| `AGENTFS_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

#### Health Checks

The container includes built-in health checks:

```bash
# Check container health
docker ps --filter "name=agentfs"

# View health check logs
docker inspect agentfs | jq '.[0].State.Health'
```

### Docker Compose

#### Basic Setup

```bash
# Start AgentFS with monitoring
docker-compose up -d

# Start with development profile
docker-compose --profile dev up -d

# Start with monitoring profile
docker-compose --profile monitoring up -d
```

#### Profiles Available

- **Default**: AgentFS server only
- **dev**: Includes development tools and debug logging
- **monitoring**: Includes Prometheus, Grafana, and Traefik

#### Services Included

| Service | Port | Profile | Description |
|---------|------|---------|-------------|
| agentfs | 8080, 8081 | default | Main AgentFS server |
| traefik | 80, 443, 8080 | monitoring | Reverse proxy and load balancer |
| prometheus | 9090 | monitoring | Metrics collection |
| grafana | 3000 | monitoring | Metrics visualization |

#### Accessing Services

- **AgentFS API**: http://localhost:8080
- **AgentFS MCP**: http://localhost:8081
- **Traefik Dashboard**: http://localhost:8080/dashboard/
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

### Kubernetes (Helm)

#### Installation

```bash
# Add the helm chart (if published)
helm repo add agentfs https://your-repo.com/helm-charts
helm repo update

# Or install from local chart
cd helm/agentfs

# Install with default values
helm install agentfs . -n agentfs --create-namespace

# Install with custom values
helm install agentfs . -n agentfs --create-namespace -f custom-values.yaml

# Upgrade existing installation
helm upgrade agentfs . -n agentfs
```

#### Configuration

Key configuration options in `values.yaml`:

```yaml
# Replica configuration
replicaCount: 3

# Image configuration
image:
  repository: agentfs
  tag: "latest"
  pullPolicy: IfNotPresent

# Service configuration
service:
  type: ClusterIP
  apiPort: 8080
  mcpPort: 8081

# Ingress configuration
ingress:
  enabled: true
  className: "nginx"
  hosts:
    - host: agentfs.example.com
      paths:
        - path: /
          pathType: Prefix

# Storage configuration
persistence:
  enabled: true
  storageClass: "default"
  size: 10Gi

# Resource limits
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi
```

#### Monitoring

```bash
# Check deployment status
kubectl get pods -n agentfs
kubectl get services -n agentfs

# View logs
kubectl logs -f deployment/agentfs -n agentfs

# Check configuration
kubectl get configmap agentfs-config -n agentfs -o yaml
```

## Desktop Installation

### Windows

#### NSIS Installer

The Windows installer (`agentfs.nsi`) provides:

- GUI installation wizard
- Service installation options
- Desktop shortcuts
- Start menu integration
- Automatic startup configuration

#### Building the Installer

```bash
# Prerequisites: NSIS installed
cd installers/windows

# Build the installer
makensis agentfs.nsi

# Output: AgentFS-Setup.exe
```

#### Installation Options

1. **Service Installation**: Installs AgentFS as a Windows service
2. **Desktop Application**: Installs as a regular application
3. **Auto-start**: Configures automatic startup

#### Post-Installation

- **Configuration**: `%USERPROFILE%\.agentfs\config.json`
- **Logs**: `%USERPROFILE%\.agentfs\logs\`
- **Service Control**: Windows Services (services.msc)

### macOS

#### PKG Installer

The macOS installer provides:

- Native PKG installation
- App bundle creation
- LaunchAgent configuration
- Code signing support

#### Building the Installer

```bash
cd installers/macos

# Build the PKG installer
./build-pkg.sh

# With custom version
./build-pkg.sh --version 0.2.0

# Output: dist/AgentFS-0.2.0.pkg
```

#### Installation Features

- **LaunchAgent**: Automatic background service
- **Menu Bar Integration**: System tray icon
- **Notifications**: Native macOS notifications

#### Post-Installation

- **Configuration**: `~/.agentfs/config.json`
- **Logs**: `~/.agentfs/logs/`
- **LaunchAgent**: `~/Library/LaunchAgents/com.agentfs.plist`

### Linux

#### Multiple Installation Options

##### 1. AppImage (Portable)

```bash
cd installers/linux

# Build AppImage
./build-appimage.sh

# Make executable and run
chmod +x dist/AgentFS-0.2.0-x86_64.AppImage
./dist/AgentFS-0.2.0-x86_64.AppImage

# Desktop integration
./dist/install-desktop-integration.sh
```

##### 2. DEB Package (Debian/Ubuntu)

```bash
# Build DEB package
./build-deb.sh

# Install
sudo dpkg -i dist/agentfs_0.2.0_amd64.deb

# Or using apt
sudo apt install ./dist/agentfs_0.2.0_amd64.deb
```

##### 3. Desktop Launcher

```bash
# Cross-platform launcher
./installers/desktop/agentfs-launcher.sh

# System tray application
python3 ./installers/desktop/agentfs-tray.py
```

#### Features

- **Systemd Service**: Background service management
- **Desktop Integration**: Menu entries and shortcuts
- **System Tray**: GUI control interface
- **Auto-updates**: Built-in update mechanism

#### Post-Installation

- **Configuration**: `~/.agentfs/config.json`
- **Service Control**: `systemctl --user start/stop/status agentfs`
- **Desktop Entry**: `~/.local/share/applications/agentfs.desktop`

## Configuration

### Basic Configuration

AgentFS uses a JSON configuration file located at:

- **Windows**: `%USERPROFILE%\.agentfs\config.json`
- **macOS/Linux**: `~/.agentfs/config.json`

#### Example Configuration

```json
{
  "version": "0.2.0",
  "agent_dir": ".agentfs",
  "global_dir": "~/.agentfs",
  "sources": [
    {
      "path": "/path/to/source",
      "type": "filesystem",
      "enabled": true
    }
  ],
  "server": {
    "api_port": 8080,
    "mcp_port": 8081,
    "host": "0.0.0.0"
  },
  "worker": {
    "count": 4,
    "scan_interval": "30s",
    "batch_size": 10
  },
  "embedding": {
    "model": "bge-base-en-v1.5",
    "cache_dir": "~/.agentfs/fastembed_cache",
    "dimension": 768
  },
  "database": {
    "compression_enabled": true,
    "compression_threshold": 512,
    "maintenance_interval": "24h",
    "deleted_threshold": "168h"
  },
  "chunking": {
    "default_strategy": "simple",
    "chunk_size": 1000,
    "overlap_size": 100,
    "min_chunk_size": 50
  }
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `AGENTFS_CONFIG_DIR` | Configuration directory | Platform-specific |
| `AGENTFS_DATA_DIR` | Data storage directory | `$CONFIG_DIR/data` |
| `AGENTFS_LOG_LEVEL` | Logging level | `info` |
| `AGENTFS_API_PORT` | API server port | `8080` |
| `AGENTFS_MCP_PORT` | MCP server port | `8081` |

## Monitoring & Maintenance

### Health Checks

AgentFS provides health check endpoints:

```bash
# Basic health check
curl http://localhost:8080/health

# Detailed status
curl http://localhost:8080/status

# Metrics (Prometheus format)
curl http://localhost:8080/metrics
```

### Logging

#### Log Levels

- **debug**: Detailed debugging information
- **info**: General operational messages
- **warn**: Warning conditions
- **error**: Error conditions

#### Log Locations

- **Docker**: Container logs via `docker logs`
- **Kubernetes**: Pod logs via `kubectl logs`
- **Windows**: `%USERPROFILE%\.agentfs\logs\`
- **macOS/Linux**: `~/.agentfs/logs/`

### Auto-Updates

Desktop installations include an auto-updater:

```bash
# Check for updates
./installers/desktop/auto-updater.sh check

# Force update check
./installers/desktop/auto-updater.sh force-check

# Enable/disable auto-updates
./installers/desktop/auto-updater.sh enable
./installers/desktop/auto-updater.sh disable

# Check status
./installers/desktop/auto-updater.sh status
```

### Backup and Recovery

#### Configuration Backup

```bash
# Backup configuration
cp ~/.agentfs/config.json ~/.agentfs/config.json.backup

# Backup entire config directory
tar -czf agentfs-config-backup.tar.gz ~/.agentfs/
```

#### Data Backup

```bash
# For server deployments
docker run --rm -v agentfs-data:/data -v $(pwd):/backup alpine \
  tar -czf /backup/agentfs-data-backup.tar.gz -C /data .

# For desktop installations
tar -czf agentfs-data-backup.tar.gz ~/.agentfs/data/
```

## Troubleshooting

### Common Issues

#### 1. Port Already in Use

```bash
# Check what's using the port
netstat -tulpn | grep :8080
lsof -i :8080

# Change port in configuration
# Edit config.json and set different api_port/mcp_port
```

#### 2. Permission Issues

```bash
# Fix ownership (Linux/macOS)
sudo chown -R $USER:$USER ~/.agentfs/

# Fix permissions
chmod -R 755 ~/.agentfs/
chmod 600 ~/.agentfs/config.json
```

#### 3. Service Not Starting

```bash
# Check service status (Linux)
systemctl --user status agentfs

# Check service logs
journalctl --user -u agentfs -f

# Check Windows service
sc query AgentFS
```

#### 4. Database Issues

```bash
# Reset database (will lose data)
rm -rf ~/.agentfs/data/
agentfs config init

# Check database integrity
agentfs database check
```

### Debug Mode

Enable debug logging:

```bash
# Environment variable
export AGENTFS_LOG_LEVEL=debug

# Configuration file
{
  "log_level": "debug"
}
```

### Support

For additional support:

1. Check the logs for error messages
2. Verify configuration syntax
3. Ensure all dependencies are installed
4. Check firewall and network settings
5. Review GitHub issues and documentation

## CI/CD Integration

### GitHub Actions

The repository includes GitHub Actions workflows for:

- **Build and Test**: Automated building and testing
- **Release**: Automated release creation and asset publishing
- **Docker**: Container image building and publishing

### Custom CI/CD

To integrate AgentFS into your CI/CD pipeline:

```yaml
# Example GitHub Actions workflow
name: Deploy AgentFS
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build Docker image
        run: docker build -t agentfs:${{ github.sha }} .

      - name: Deploy to Kubernetes
        run: |
          helm upgrade agentfs ./helm/agentfs \
            --set image.tag=${{ github.sha }} \
            --namespace production
```

This completes the comprehensive deployment guide for AgentFS covering all server and desktop deployment scenarios.