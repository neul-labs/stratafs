#!/bin/bash
# StrataFS Linux AppImage Builder

set -e

# Configuration
VERSION="${VERSION:-0.2.0}"
BINARY_PATH="${BINARY_PATH:-../../build/linux-amd64/stratafs}"
OUTPUT_DIR="${OUTPUT_DIR:-dist}"
ARCH="${ARCH:-x86_64}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR/build-appimage"
APPDIR="$BUILD_DIR/StrataFS.AppDir"

print_info "Building StrataFS AppImage v$VERSION for $ARCH"

# Check dependencies
check_dependencies() {
    print_info "Checking dependencies..."

    if [ ! -f "$BINARY_PATH" ]; then
        print_error "Binary not found at: $BINARY_PATH"
        print_info "Please build the Linux binary first:"
        print_info "  GOOS=linux GOARCH=amd64 go build -tags 'fts5' -o build/linux-amd64/stratafs ./cmd/stratafs"
        exit 1
    fi

    # Download appimagetool if not present
    if [ ! -f "$SCRIPT_DIR/appimagetool-$ARCH.AppImage" ]; then
        print_info "Downloading appimagetool..."
        wget -O "$SCRIPT_DIR/appimagetool-$ARCH.AppImage" \
            "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-$ARCH.AppImage"
        chmod +x "$SCRIPT_DIR/appimagetool-$ARCH.AppImage"
    fi

    print_success "Dependencies check passed"
}

# Setup AppDir structure
setup_appdir() {
    print_info "Setting up AppDir structure..."

    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"
    mkdir -p "$OUTPUT_DIR"

    # Create AppDir structure
    mkdir -p "$APPDIR/usr/bin"
    mkdir -p "$APPDIR/usr/lib"
    mkdir -p "$APPDIR/usr/share/applications"
    mkdir -p "$APPDIR/usr/share/icons/hicolor/256x256/apps"
    mkdir -p "$APPDIR/usr/share/stratafs"
    mkdir -p "$APPDIR/usr/share/doc/stratafs"
}

# Copy application files
copy_files() {
    print_info "Copying application files..."

    # Copy main binary
    cp "$BINARY_PATH" "$APPDIR/usr/bin/stratafs"
    chmod +x "$APPDIR/usr/bin/stratafs"

    # Copy ONNX Runtime libraries if available
    ONNX_LIB_DIR="$(dirname "$BINARY_PATH")/lib"
    if [ -d "$ONNX_LIB_DIR" ]; then
        cp -r "$ONNX_LIB_DIR"/* "$APPDIR/usr/lib/"
        print_info "Copied ONNX Runtime libraries"
    fi

    # Copy documentation
    if [ -f "$PROJECT_ROOT/README.md" ]; then
        cp "$PROJECT_ROOT/README.md" "$APPDIR/usr/share/doc/stratafs/"
    fi

    if [ -f "$PROJECT_ROOT/LICENSE" ]; then
        cp "$PROJECT_ROOT/LICENSE" "$APPDIR/usr/share/doc/stratafs/"
    fi

    # Create default configuration
    cat > "$APPDIR/usr/share/stratafs/default-config.json" << 'EOF'
{
  "version": "0.2.0",
  "agent_dir": ".stratafs",
  "global_dir": "~/.stratafs",
  "sources": [],
  "server": {
    "api_port": 8080,
    "mcp_port": 8081
  },
  "worker": {
    "count": 4,
    "scan_interval": "30s",
    "batch_size": 10
  },
  "embedding": {
    "model": "bge-base-en-v1.5",
    "cache_dir": "~/.stratafs/fastembed_cache",
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
EOF
}

# Create desktop entry
create_desktop_entry() {
    print_info "Creating desktop entry..."

    cat > "$APPDIR/stratafs.desktop" << EOF
[Desktop Entry]
Type=Application
Name=StrataFS
Comment=The Agentic Filesystem for AI agents
Exec=stratafs
Icon=stratafs
Terminal=false
Categories=System;FileManager;Network;
Keywords=filesystem;search;ai;agent;embedding;
StartupNotify=true
X-AppImage-Version=$VERSION
EOF

    # Copy to applications directory as well
    cp "$APPDIR/stratafs.desktop" "$APPDIR/usr/share/applications/"
}

# Create AppRun script
create_apprun() {
    print_info "Creating AppRun script..."

    cat > "$APPDIR/AppRun" << 'EOF'
#!/bin/bash

# Get the directory where this AppImage is mounted
HERE="$(dirname "$(readlink -f "${0}")")"

# Set library path for ONNX Runtime
export LD_LIBRARY_PATH="$HERE/usr/lib:${LD_LIBRARY_PATH}"

# Add our bin directory to PATH
export PATH="$HERE/usr/bin:$PATH"

# Initialize StrataFS config if it doesn't exist
if [ ! -d "$HOME/.stratafs" ]; then
    echo "Initializing StrataFS configuration..."
    mkdir -p "$HOME/.stratafs"
    if [ -f "$HERE/usr/share/stratafs/default-config.json" ]; then
        cp "$HERE/usr/share/stratafs/default-config.json" "$HOME/.stratafs/config.json"
    fi
fi

# Handle different run modes
case "$1" in
    --config)
        shift
        exec "$HERE/usr/bin/stratafs" config "$@"
        ;;
    --source)
        shift
        exec "$HERE/usr/bin/stratafs" source "$@"
        ;;
    --help|help)
        exec "$HERE/usr/bin/stratafs" --help
        ;;
    --version|version)
        exec "$HERE/usr/bin/stratafs" --version
        ;;
    --desktop)
        # Desktop mode: run in background with GUI integration
        if ! pgrep -f "stratafs" > /dev/null; then
            nohup "$HERE/usr/bin/stratafs" > "$HOME/.stratafs/desktop.log" 2>&1 &
            echo "StrataFS started in background. Check status at http://localhost:8080/health"
        else
            echo "StrataFS is already running"
        fi
        ;;
    --stop)
        # Stop StrataFS
        pkill -f "stratafs" && echo "StrataFS stopped" || echo "StrataFS not running"
        ;;
    *)
        # Default: run StrataFS with all arguments
        exec "$HERE/usr/bin/stratafs" "$@"
        ;;
esac
EOF

    chmod +x "$APPDIR/AppRun"
}

# Create icon
create_icon() {
    print_info "Creating application icon..."

    # Check if icon exists
    if [ -f "$SCRIPT_DIR/assets/stratafs.png" ]; then
        cp "$SCRIPT_DIR/assets/stratafs.png" "$APPDIR/stratafs.png"
        cp "$SCRIPT_DIR/assets/stratafs.png" "$APPDIR/usr/share/icons/hicolor/256x256/apps/stratafs.png"
    else
        print_warning "Icon not found, creating placeholder"
        # Create a simple colored square as placeholder
        if command -v convert >/dev/null 2>&1; then
            convert -size 256x256 xc:'#007AFF' "$APPDIR/stratafs.png"
            cp "$APPDIR/stratafs.png" "$APPDIR/usr/share/icons/hicolor/256x256/apps/stratafs.png"
        else
            print_warning "ImageMagick not found, using text icon"
            echo "StrataFS" > "$APPDIR/stratafs.png"
            cp "$APPDIR/stratafs.png" "$APPDIR/usr/share/icons/hicolor/256x256/apps/stratafs.png"
        fi
    fi
}

# Create launcher script
create_launcher() {
    print_info "Creating launcher script..."

    cat > "$APPDIR/usr/bin/stratafs-gui" << 'EOF'
#!/bin/bash
# StrataFS GUI Launcher

# Check if zenity is available for GUI dialogs
if command -v zenity >/dev/null 2>&1; then
    HAS_ZENITY=true
else
    HAS_ZENITY=false
fi

show_dialog() {
    if [ "$HAS_ZENITY" = "true" ]; then
        zenity --info --text="$1" --title="StrataFS"
    else
        echo "$1"
    fi
}

show_error() {
    if [ "$HAS_ZENITY" = "true" ]; then
        zenity --error --text="$1" --title="StrataFS Error"
    else
        echo "ERROR: $1" >&2
    fi
}

# Check if StrataFS is running
if pgrep -f "stratafs" > /dev/null; then
    show_dialog "StrataFS is already running!\n\nAccess the web interface at:\nhttp://localhost:8080"
    exit 0
fi

# Initialize if needed
if [ ! -d "$HOME/.stratafs" ]; then
    show_dialog "Initializing StrataFS for first use..."
    if ! stratafs config init; then
        show_error "Failed to initialize StrataFS configuration"
        exit 1
    fi
fi

# Start StrataFS
show_dialog "Starting StrataFS...\n\nThe service will be available at:\nhttp://localhost:8080"

# Run in background
nohup stratafs > "$HOME/.stratafs/desktop.log" 2>&1 &

# Wait a moment and check if it started
sleep 2
if pgrep -f "stratafs" > /dev/null; then
    show_dialog "StrataFS started successfully!\n\nWeb interface: http://localhost:8080\nMCP server: http://localhost:8081"
else
    show_error "Failed to start StrataFS. Check the log at ~/.stratafs/desktop.log"
    exit 1
fi
EOF

    chmod +x "$APPDIR/usr/bin/stratafs-gui"
}

# Build AppImage
build_appimage() {
    print_info "Building AppImage..."

    cd "$BUILD_DIR"

    # Set version for AppImage metadata
    export VERSION="$VERSION"

    # Build the AppImage
    "$SCRIPT_DIR/appimagetool-$ARCH.AppImage" \
        --appimage-extract-and-run \
        StrataFS.AppDir \
        "$OUTPUT_DIR/StrataFS-$VERSION-$ARCH.AppImage"

    if [ $? -ne 0 ]; then
        print_error "AppImage build failed"
        exit 1
    fi

    chmod +x "$OUTPUT_DIR/StrataFS-$VERSION-$ARCH.AppImage"
    print_success "AppImage built: $OUTPUT_DIR/StrataFS-$VERSION-$ARCH.AppImage"
}

# Generate checksum
generate_checksum() {
    print_info "Generating checksum..."

    local appimage_path="$OUTPUT_DIR/StrataFS-$VERSION-$ARCH.AppImage"
    local checksum=$(sha256sum "$appimage_path" | cut -d' ' -f1)
    echo "$checksum  $(basename "$appimage_path")" > "$appimage_path.sha256"

    print_success "Checksum generated: $appimage_path.sha256"
    print_info "SHA256: $checksum"
}

# Create desktop integration script
create_integration_script() {
    print_info "Creating desktop integration script..."

    cat > "$OUTPUT_DIR/install-desktop-integration.sh" << EOF
#!/bin/bash
# StrataFS Desktop Integration Installer

APPIMAGE_PATH="\$1"
if [ -z "\$APPIMAGE_PATH" ]; then
    APPIMAGE_PATH="./StrataFS-$VERSION-$ARCH.AppImage"
fi

if [ ! -f "\$APPIMAGE_PATH" ]; then
    echo "Error: AppImage not found at \$APPIMAGE_PATH"
    exit 1
fi

APPIMAGE_PATH="\$(readlink -f "\$APPIMAGE_PATH")"

echo "Installing StrataFS desktop integration..."

# Create desktop entry
mkdir -p "\$HOME/.local/share/applications"
cat > "\$HOME/.local/share/applications/stratafs.desktop" << DESKTOP_EOF
[Desktop Entry]
Type=Application
Name=StrataFS
Comment=The Agentic Filesystem for AI agents
Exec=\$APPIMAGE_PATH --desktop
Icon=stratafs
Terminal=false
Categories=System;FileManager;Network;
Keywords=filesystem;search;ai;agent;embedding;
StartupNotify=true
X-AppImage-Version=$VERSION
DESKTOP_EOF

# Extract and install icon
"\$APPIMAGE_PATH" --appimage-extract usr/share/icons/hicolor/256x256/apps/stratafs.png >/dev/null 2>&1
if [ -f "squashfs-root/usr/share/icons/hicolor/256x256/apps/stratafs.png" ]; then
    mkdir -p "\$HOME/.local/share/icons/hicolor/256x256/apps"
    cp "squashfs-root/usr/share/icons/hicolor/256x256/apps/stratafs.png" "\$HOME/.local/share/icons/hicolor/256x256/apps/"
    rm -rf squashfs-root
fi

# Update desktop database
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "\$HOME/.local/share/applications"
fi

echo "Desktop integration installed successfully!"
echo "You can now find StrataFS in your application menu."
EOF

    chmod +x "$OUTPUT_DIR/install-desktop-integration.sh"
}

# Main execution
main() {
    check_dependencies
    setup_appdir
    copy_files
    create_desktop_entry
    create_apprun
    create_icon
    create_launcher
    build_appimage
    generate_checksum
    create_integration_script

    print_success "AppImage build completed!"
    print_info "AppImage: $OUTPUT_DIR/StrataFS-$VERSION-$ARCH.AppImage"
    print_info "Size: $(du -h "$OUTPUT_DIR/StrataFS-$VERSION-$ARCH.AppImage" | cut -f1)"

    echo ""
    print_info "Usage instructions:"
    print_info "1. Make executable: chmod +x StrataFS-$VERSION-$ARCH.AppImage"
    print_info "2. Run directly: ./StrataFS-$VERSION-$ARCH.AppImage"
    print_info "3. Desktop mode: ./StrataFS-$VERSION-$ARCH.AppImage --desktop"
    print_info "4. Install integration: ./install-desktop-integration.sh"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --binary-path)
            BINARY_PATH="$2"
            shift 2
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --arch)
            ARCH="$2"
            shift 2
            ;;
        --help)
            echo "StrataFS AppImage Build Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --version VERSION      Version string (default: 0.2.0)"
            echo "  --binary-path PATH     Path to stratafs binary"
            echo "  --output-dir DIR       Output directory (default: dist)"
            echo "  --arch ARCH           Architecture (default: x86_64)"
            echo "  --help                Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main "$@"