#!/bin/bash
# Post-installation script for AgentFS macOS
# Sets up LaunchAgent and symlinks

set -e

APP_PATH="/Applications/AgentFS.app"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"
PLIST_NAME="org.agentfs.daemon.plist"

# Create LaunchAgents directory if needed
mkdir -p "$LAUNCH_AGENTS_DIR"

# Copy LaunchAgent plist
if [ -f "$APP_PATH/Contents/Library/LaunchAgents/$PLIST_NAME" ]; then
    cp "$APP_PATH/Contents/Library/LaunchAgents/$PLIST_NAME" "$LAUNCH_AGENTS_DIR/"
fi

# Create symlink in /usr/local/bin
sudo mkdir -p /usr/local/bin
sudo ln -sf "$APP_PATH/Contents/MacOS/agentfs" /usr/local/bin/agentfs

# Install Spotlight importer
SPOTLIGHT_DIR="$HOME/Library/Spotlight"
if [ -d "$APP_PATH/Contents/Library/Spotlight/AgentFSImporter.mdimporter" ]; then
    mkdir -p "$SPOTLIGHT_DIR"
    cp -R "$APP_PATH/Contents/Library/Spotlight/AgentFSImporter.mdimporter" "$SPOTLIGHT_DIR/"
    # Reload Spotlight
    mdimport -r "$SPOTLIGHT_DIR/AgentFSImporter.mdimporter"
fi

# Install Finder Sync extension
# Note: This requires the user to enable it in System Preferences

# Load LaunchAgent
launchctl unload "$LAUNCH_AGENTS_DIR/$PLIST_NAME" 2>/dev/null || true
launchctl load "$LAUNCH_AGENTS_DIR/$PLIST_NAME"

echo "✅ AgentFS installed successfully"
echo ""
echo "The AgentFS daemon is now running in the background."
echo "Use 'agentfs' command or open AgentFS.app to manage your sources."
