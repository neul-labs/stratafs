# AgentFS Desktop UI

A Wails-based desktop control panel for managing AgentFS.

## Features

- **Dashboard**: View service status, queue statistics, start/stop/restart AgentFS
- **Search**: Perform hybrid, full-text, or semantic search across indexed files
- **Sources**: Add/remove storage sources to monitor
- **Queue**: View processing queue statistics
- **Export**: Export metadata for sources
- **Settings**: View and manage configuration

## Prerequisites

### Build Dependencies (Ubuntu/Debian)

```bash
sudo apt-get install -y \
    libwebkit2gtk-4.0-dev \
    build-essential \
    pkg-config \
    libgtk-3-dev
```

### Runtime Dependencies

- AgentFS daemon (`agentfs`)
- ONNX Runtime (for embeddings)

## Development

### Run in development mode

```bash
wails dev
```

This will start the app with hot-reload for frontend changes.

### Build for production

```bash
wails build -platform linux/amd64
```

The binary will be created in `build/bin/agentfs-ui`.

## Packaging

### Create AppImage

```bash
./build/linux/package-appimage.sh
```

Creates a portable AppImage at `build/appimage/AgentFS-x86_64.AppImage`.

### Create Debian package

```bash
VERSION=0.2.0 ./build/linux/package-deb.sh
```

Creates a .deb package with:
- `agentfs-ui` - Desktop control panel
- `agentfs` - Core daemon
- systemd user service

### Install .deb package

```bash
sudo dpkg -i build/deb/agentfs_0.2.0_amd64.deb
```

### Enable autostart

```bash
systemctl --user enable agentfs
systemctl --user start agentfs
```

## Architecture

The app uses Wails v2 with:
- **Backend**: Go bindings that communicate with AgentFS REST API
- **Frontend**: Vue 3 with Composition API

### Backend Methods

- `GetStatus()` - Get AgentFS service status
- `StartAgentFS()` / `StopAgentFS()` / `RestartAgentFS()` - Service control
- `GetQueueStats()` - Queue statistics
- `Search()` - Perform searches
- `GetConfig()` - Load configuration
- `AddSource()` / `RemoveSource()` - Manage sources
- `ExportSource()` - Export metadata
- `InitConfig()` - Initialize configuration

## Configuration

The app reads configuration from `~/.agentfs/config.json`.

Default API endpoint: `http://localhost:8080`
