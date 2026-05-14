# StrataFS Nautilus Extension

Adds StrataFS context menu actions to GNOME Files (Nautilus).

## Features

### File Actions (right-click on files)
- **View StrataFS Metadata** - Show file info, checksum, size, timestamps
- **View StrataFS Chunks** - Display extracted text chunks
- **Find Similar Files** - Semantic search for similar content
- **Reindex in StrataFS** - Queue file(s) for reprocessing

### Folder Actions (right-click on folder background)
- **Add Folder to StrataFS** - Add folder as a new source
- **Export StrataFS Metadata Here** - Export metadata view to folder

## Installation

```bash
# Create extension directory
mkdir -p ~/.local/share/nautilus-python/extensions

# Copy extension
cp stratafs-nautilus.py ~/.local/share/nautilus-python/extensions/

# Restart Nautilus
nautilus -q
```

## Requirements

- Python 3
- python3-nautilus package
- zenity (for dialogs)
- notify-send (for notifications)
- StrataFS CLI installed and in PATH

Install dependencies on Ubuntu/Debian:
```bash
sudo apt install python3-nautilus zenity libnotify-bin
```

## Configuration

Set `STRATAFS_API_URL` environment variable if using non-default API URL:
```bash
export STRATAFS_API_URL=http://localhost:9000
```
