#!/bin/bash
# Build script for StrataFS Desktop UI
# This bundles both the UI and the stratafs daemon together

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR/build/bin"

echo "=== Building StrataFS Desktop ==="
echo "Root: $ROOT_DIR"
echo "Output: $BUILD_DIR"

# Create build directory
mkdir -p "$BUILD_DIR"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

echo ""
echo "=== Building stratafs daemon ==="
cd "$ROOT_DIR"
go build -tags "fts5" -o "$BUILD_DIR/stratafs" ./cmd/stratafs/main.go
echo "Built: $BUILD_DIR/stratafs"

echo ""
echo "=== Building StrataFS UI ==="
cd "$SCRIPT_DIR"

# Build with wails
if [ "$OS" = "linux" ]; then
    wails build -tags "webkit2_41" -o stratafs-ui
elif [ "$OS" = "darwin" ]; then
    wails build -o stratafs-ui
else
    wails build -o stratafs-ui.exe
fi

# Copy the wails output to our build directory
if [ "$OS" = "darwin" ]; then
    # macOS creates an .app bundle
    cp -r "$SCRIPT_DIR/build/bin/stratafs-ui.app" "$BUILD_DIR/" 2>/dev/null || true
    # Copy stratafs into the app bundle
    mkdir -p "$BUILD_DIR/stratafs-ui.app/Contents/Resources"
    cp "$BUILD_DIR/stratafs" "$BUILD_DIR/stratafs-ui.app/Contents/Resources/"
elif [ "$OS" = "linux" ]; then
    # Linux binary is already in build/bin
    echo "Linux build complete"
else
    # Windows
    cp "$SCRIPT_DIR/build/bin/stratafs-ui.exe" "$BUILD_DIR/" 2>/dev/null || true
    cp "$BUILD_DIR/stratafs" "$BUILD_DIR/stratafs.exe" 2>/dev/null || true
fi

echo ""
echo "=== Build Complete ==="
echo "Output directory: $BUILD_DIR"
ls -la "$BUILD_DIR"

echo ""
echo "To run: $BUILD_DIR/stratafs-ui"
