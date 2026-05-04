# AgentFS Nautilus Extension

Adds AgentFS context menu actions to GNOME Files (Nautilus).

## Features

### File Actions (right-click on files)
- **View AgentFS Metadata** - Show file info, checksum, size, timestamps
- **View AgentFS Chunks** - Display extracted text chunks
- **Find Similar Files** - Semantic search for similar content
- **Reindex in AgentFS** - Queue file(s) for reprocessing

### Folder Actions (right-click on folder background)
- **Add Folder to AgentFS** - Add folder as a new source
- **Export AgentFS Metadata Here** - Export metadata view to folder

## Installation

```bash
# Create extension directory
mkdir -p ~/.local/share/nautilus-python/extensions

# Copy extension
cp agentfs-nautilus.py ~/.local/share/nautilus-python/extensions/

# Restart Nautilus
nautilus -q
```

## Requirements

- Python 3
- python3-nautilus package
- zenity (for dialogs)
- notify-send (for notifications)
- AgentFS CLI installed and in PATH

Install dependencies on Ubuntu/Debian:
```bash
sudo apt install python3-nautilus zenity libnotify-bin
```

## Configuration

Set `AGENTFS_API_URL` environment variable if using non-default API URL:
```bash
export AGENTFS_API_URL=http://localhost:9000
```
