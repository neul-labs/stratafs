#!/bin/bash
# StrataFS Debian Package Builder

set -e

# Configuration
VERSION="${VERSION:-0.2.0}"
BINARY_PATH="${BINARY_PATH:-../../build/linux-amd64/stratafs}"
OUTPUT_DIR="${OUTPUT_DIR:-dist}"
ARCH="${ARCH:-amd64}"
MAINTAINER="${MAINTAINER:-StrataFS Team <team@stratafs.dev>}"

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
BUILD_DIR="$SCRIPT_DIR/build-deb"
PACKAGE_DIR="$BUILD_DIR/stratafs_$VERSION-1_$ARCH"

print_info "Building StrataFS Debian package v$VERSION for $ARCH"

# Check dependencies
check_dependencies() {
    print_info "Checking dependencies..."

    if [ ! -f "$BINARY_PATH" ]; then
        print_error "Binary not found at: $BINARY_PATH"
        print_info "Please build the Linux binary first:"
        print_info "  GOOS=linux GOARCH=amd64 go build -tags 'fts5' -o build/linux-amd64/stratafs ./cmd/stratafs"
        exit 1
    fi

    if ! command -v dpkg-deb >/dev/null 2>&1; then
        print_error "dpkg-deb not found. Please install dpkg-dev:"
        print_info "  sudo apt-get install dpkg-dev"
        exit 1
    fi

    print_success "Dependencies check passed"
}

# Setup package structure
setup_package_structure() {
    print_info "Setting up package structure..."

    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"
    mkdir -p "$OUTPUT_DIR"

    # Create package directory structure
    mkdir -p "$PACKAGE_DIR/DEBIAN"
    mkdir -p "$PACKAGE_DIR/usr/bin"
    mkdir -p "$PACKAGE_DIR/usr/lib/stratafs"
    mkdir -p "$PACKAGE_DIR/usr/share/applications"
    mkdir -p "$PACKAGE_DIR/usr/share/icons/hicolor/256x256/apps"
    mkdir -p "$PACKAGE_DIR/usr/share/doc/stratafs"
    mkdir -p "$PACKAGE_DIR/usr/share/stratafs"
    mkdir -p "$PACKAGE_DIR/etc/stratafs"
    mkdir -p "$PACKAGE_DIR/lib/systemd/system"
    mkdir -p "$PACKAGE_DIR/usr/share/man/man1"
}

# Copy application files
copy_files() {
    print_info "Copying application files..."

    # Copy main binary
    cp "$BINARY_PATH" "$PACKAGE_DIR/usr/bin/stratafs"
    chmod +x "$PACKAGE_DIR/usr/bin/stratafs"

    # Copy ONNX Runtime libraries if available
    ONNX_LIB_DIR="$(dirname "$BINARY_PATH")/lib"
    if [ -d "$ONNX_LIB_DIR" ]; then
        cp -r "$ONNX_LIB_DIR"/* "$PACKAGE_DIR/usr/lib/stratafs/"
        print_info "Copied ONNX Runtime libraries"
    fi

    # Copy documentation
    if [ -f "$PROJECT_ROOT/README.md" ]; then
        cp "$PROJECT_ROOT/README.md" "$PACKAGE_DIR/usr/share/doc/stratafs/"
        gzip -9 -c "$PROJECT_ROOT/README.md" > "$PACKAGE_DIR/usr/share/doc/stratafs/README.gz"
    fi

    if [ -f "$PROJECT_ROOT/LICENSE" ]; then
        cp "$PROJECT_ROOT/LICENSE" "$PACKAGE_DIR/usr/share/doc/stratafs/copyright"
    fi

    # Create changelog
    cat > "$PACKAGE_DIR/usr/share/doc/stratafs/changelog.Debian" << EOF
stratafs ($VERSION-1) unstable; urgency=medium

  * Initial release of StrataFS
  * Semantic filesystem search and AI agent integration
  * Support for multiple storage backends
  * Real-time file monitoring and indexing
  * Streaming text chunking with multiple strategies
  * Model Context Protocol server integration

 -- $MAINTAINER  $(date -R)
EOF
    gzip -9 "$PACKAGE_DIR/usr/share/doc/stratafs/changelog.Debian"

    # Create default configuration
    cat > "$PACKAGE_DIR/etc/stratafs/config.json" << 'EOF'
{
  "version": "0.2.0",
  "agent_dir": ".stratafs",
  "global_dir": "/var/lib/stratafs",
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
    "cache_dir": "/var/lib/stratafs/fastembed_cache",
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

    # User-specific default config
    cat > "$PACKAGE_DIR/usr/share/stratafs/user-config.json" << 'EOF'
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

# Create systemd service
create_systemd_service() {
    print_info "Creating systemd service..."

    cat > "$PACKAGE_DIR/lib/systemd/system/stratafs.service" << 'EOF'
[Unit]
Description=StrataFS - The Agentic Filesystem
Documentation=https://github.com/neul-labs/stratafs
After=network.target

[Service]
Type=simple
User=stratafs
Group=stratafs
ExecStart=/usr/bin/stratafs --config /etc/stratafs/config.json
ExecReload=/bin/kill -HUP $MAINPID
KillMode=process
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=stratafs

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/stratafs /etc/stratafs
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
EOF
}

# Create desktop entry
create_desktop_entry() {
    print_info "Creating desktop entry..."

    cat > "$PACKAGE_DIR/usr/share/applications/stratafs.desktop" << EOF
[Desktop Entry]
Type=Application
Name=StrataFS
Comment=The Agentic Filesystem for AI agents
Exec=stratafs-gui
Icon=stratafs
Terminal=false
Categories=System;FileManager;Network;
Keywords=filesystem;search;ai;agent;embedding;
StartupNotify=true
Version=$VERSION
EOF

    # Create GUI launcher script
    cat > "$PACKAGE_DIR/usr/bin/stratafs-gui" << 'EOF'
#!/bin/bash
# StrataFS GUI Launcher

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo "StrataFS should not be run as root for desktop use."
    echo "Please run as a regular user."
    exit 1
fi

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
        notify-send "StrataFS" "$1" 2>/dev/null || echo "$1"
    fi
}

show_error() {
    if [ "$HAS_ZENITY" = "true" ]; then
        zenity --error --text="$1" --title="StrataFS Error"
    else
        notify-send "StrataFS Error" "$1" 2>/dev/null || echo "ERROR: $1" >&2
    fi
}

# Initialize user configuration if needed
if [ ! -d "$HOME/.stratafs" ]; then
    show_dialog "Initializing StrataFS for first use..."
    mkdir -p "$HOME/.stratafs"
    if [ -f "/usr/share/stratafs/user-config.json" ]; then
        cp "/usr/share/stratafs/user-config.json" "$HOME/.stratafs/config.json"
    fi
    stratafs config init --config-dir="$HOME/.stratafs" || {
        show_error "Failed to initialize StrataFS configuration"
        exit 1
    }
fi

# Check if StrataFS is running
if pgrep -f "stratafs" > /dev/null; then
    show_dialog "StrataFS is already running!\n\nAccess the web interface at:\nhttp://localhost:8080"
    # Open browser if available
    if command -v xdg-open >/dev/null 2>&1; then
        xdg-open "http://localhost:8080" &
    fi
    exit 0
fi

# Start StrataFS
show_dialog "Starting StrataFS...\n\nThe service will be available at:\nhttp://localhost:8080"

# Run in background
nohup stratafs --config-dir="$HOME/.stratafs" > "$HOME/.stratafs/desktop.log" 2>&1 &

# Wait a moment and check if it started
sleep 2
if pgrep -f "stratafs" > /dev/null; then
    show_dialog "StrataFS started successfully!\n\nWeb interface: http://localhost:8080\nMCP server: http://localhost:8081"
    # Open browser if available
    if command -v xdg-open >/dev/null 2>&1; then
        xdg-open "http://localhost:8080" &
    fi
else
    show_error "Failed to start StrataFS. Check the log at ~/.stratafs/desktop.log"
    exit 1
fi
EOF

    chmod +x "$PACKAGE_DIR/usr/bin/stratafs-gui"
}

# Create icon
create_icon() {
    print_info "Creating application icon..."

    # Check if icon exists
    if [ -f "$SCRIPT_DIR/assets/stratafs.png" ]; then
        cp "$SCRIPT_DIR/assets/stratafs.png" "$PACKAGE_DIR/usr/share/icons/hicolor/256x256/apps/stratafs.png"
    else
        print_warning "Icon not found, creating placeholder"
        # Create a simple colored square as placeholder
        if command -v convert >/dev/null 2>&1; then
            convert -size 256x256 xc:'#007AFF' -gravity center -pointsize 48 -fill white -annotate +0+0 'StrataFS' \
                "$PACKAGE_DIR/usr/share/icons/hicolor/256x256/apps/stratafs.png"
        else
            print_warning "ImageMagick not found, using text file as icon"
            echo "StrataFS Icon" > "$PACKAGE_DIR/usr/share/icons/hicolor/256x256/apps/stratafs.png"
        fi
    fi
}

# Create man page
create_man_page() {
    print_info "Creating man page..."

    cat > "$PACKAGE_DIR/usr/share/man/man1/stratafs.1" << 'EOF'
.TH STRATAFS 1 "$(date '+%B %Y')" "stratafs $VERSION" "User Commands"
.SH NAME
stratafs \- The Agentic Filesystem for AI agents
.SH SYNOPSIS
.B stratafs
[\fIOPTION\fR]...
.SH DESCRIPTION
StrataFS transforms passive file storage into an active, searchable, and semantically-aware knowledge base that AI agents can reason about and interact with naturally.

.SH OPTIONS
.TP
\fB\-\-config\fR \fIFILE\fR
Specify configuration file path
.TP
\fB\-\-config-dir\fR \fIDIR\fR
Specify configuration directory
.TP
\fB\-\-help\fR
Show help message
.TP
\fB\-\-version\fR
Show version information

.SH COMMANDS
.TP
\fBconfig init\fR
Initialize StrataFS configuration
.TP
\fBconfig show\fR
Display current configuration
.TP
\fBsource add\fR
Add a new storage source
.TP
\fBsource list\fR
List configured sources

.SH FILES
.TP
\fI~/.stratafs/config.json\fR
User configuration file
.TP
\fI/etc/stratafs/config.json\fR
System-wide configuration file

.SH AUTHOR
Written by the StrataFS Team.

.SH "REPORTING BUGS"
Report bugs to: https://github.com/neul-labs/stratafs/issues

.SH COPYRIGHT
Copyright © 2024 StrataFS Team.
This is free software; see the source for copying conditions.

.SH "SEE ALSO"
Project homepage: https://github.com/neul-labs/stratafs
EOF

    gzip -9 "$PACKAGE_DIR/usr/share/man/man1/stratafs.1"
}

# Create DEBIAN control files
create_debian_control() {
    print_info "Creating Debian control files..."

    # Calculate installed size
    INSTALLED_SIZE=$(du -sk "$PACKAGE_DIR" | cut -f1)

    # Main control file
    cat > "$PACKAGE_DIR/DEBIAN/control" << EOF
Package: stratafs
Version: $VERSION-1
Section: utils
Priority: optional
Architecture: $ARCH
Depends: libc6 (>= 2.17), libsqlite3-0 (>= 3.7.15)
Suggests: zenity, xdg-utils
Installed-Size: $INSTALLED_SIZE
Maintainer: $MAINTAINER
Description: The Agentic Filesystem for AI agents
 StrataFS transforms passive file storage into an active, searchable, and
 semantically-aware knowledge base that AI agents can reason about and
 interact with naturally.
 .
 Features:
  * Semantic search across all your files
  * AI agent integration via Model Context Protocol
  * Multi-storage support (local, cloud)
  * Real-time file monitoring and indexing
  * Streaming text chunking with multiple strategies
  * Hybrid search combining full-text and vector similarity
Homepage: https://github.com/neul-labs/stratafs
EOF

    # Pre-installation script
    cat > "$PACKAGE_DIR/DEBIAN/preinst" << 'EOF'
#!/bin/bash
set -e

# Stop service if running
if systemctl is-active --quiet stratafs 2>/dev/null; then
    systemctl stop stratafs
fi

exit 0
EOF

    # Post-installation script
    cat > "$PACKAGE_DIR/DEBIAN/postinst" << 'EOF'
#!/bin/bash
set -e

case "$1" in
    configure)
        # Create stratafs user and group
        if ! getent group stratafs >/dev/null; then
            groupadd --system stratafs
        fi

        if ! getent passwd stratafs >/dev/null; then
            useradd --system --gid stratafs --home /var/lib/stratafs \
                    --shell /bin/false --comment "StrataFS daemon" stratafs
        fi

        # Create directories
        mkdir -p /var/lib/stratafs
        chown stratafs:stratafs /var/lib/stratafs
        chmod 755 /var/lib/stratafs

        # Set permissions
        chown root:stratafs /etc/stratafs/config.json
        chmod 640 /etc/stratafs/config.json

        # Reload systemd
        systemctl daemon-reload

        # Enable service (but don't start automatically)
        systemctl enable stratafs

        # Update desktop database
        if command -v update-desktop-database >/dev/null 2>&1; then
            update-desktop-database /usr/share/applications
        fi

        # Update icon cache
        if command -v gtk-update-icon-cache >/dev/null 2>&1; then
            gtk-update-icon-cache -q /usr/share/icons/hicolor
        fi

        echo "StrataFS has been installed successfully!"
        echo ""
        echo "To start the service:"
        echo "  sudo systemctl start stratafs"
        echo ""
        echo "To start at boot:"
        echo "  sudo systemctl enable stratafs"
        echo ""
        echo "For desktop use, find StrataFS in your application menu"
        echo "or run: stratafs-gui"
        ;;
esac

exit 0
EOF

    # Pre-removal script
    cat > "$PACKAGE_DIR/DEBIAN/prerm" << 'EOF'
#!/bin/bash
set -e

case "$1" in
    remove|upgrade|deconfigure)
        # Stop service
        if systemctl is-active --quiet stratafs 2>/dev/null; then
            systemctl stop stratafs
        fi

        # Disable service
        if systemctl is-enabled --quiet stratafs 2>/dev/null; then
            systemctl disable stratafs
        fi
        ;;
esac

exit 0
EOF

    # Post-removal script
    cat > "$PACKAGE_DIR/DEBIAN/postrm" << 'EOF'
#!/bin/bash
set -e

case "$1" in
    remove)
        # Update desktop database
        if command -v update-desktop-database >/dev/null 2>&1; then
            update-desktop-database /usr/share/applications
        fi

        # Update icon cache
        if command -v gtk-update-icon-cache >/dev/null 2>&1; then
            gtk-update-icon-cache -q /usr/share/icons/hicolor
        fi
        ;;

    purge)
        # Remove user data directory (ask user first)
        if [ -d /var/lib/stratafs ]; then
            echo "Remove StrataFS data directory /var/lib/stratafs? [y/N]"
            read -r response
            if [ "$response" = "y" ] || [ "$response" = "Y" ]; then
                rm -rf /var/lib/stratafs
            fi
        fi

        # Remove user and group
        if getent passwd stratafs >/dev/null; then
            userdel stratafs
        fi

        if getent group stratafs >/dev/null; then
            groupdel stratafs
        fi

        # Reload systemd
        systemctl daemon-reload
        ;;
esac

exit 0
EOF

    # Make scripts executable
    chmod +x "$PACKAGE_DIR/DEBIAN/preinst"
    chmod +x "$PACKAGE_DIR/DEBIAN/postinst"
    chmod +x "$PACKAGE_DIR/DEBIAN/prerm"
    chmod +x "$PACKAGE_DIR/DEBIAN/postrm"
}

# Build the package
build_package() {
    print_info "Building Debian package..."

    # Fix permissions
    find "$PACKAGE_DIR" -type d -exec chmod 755 {} \;
    find "$PACKAGE_DIR" -type f -exec chmod 644 {} \;
    chmod +x "$PACKAGE_DIR/usr/bin/stratafs"
    chmod +x "$PACKAGE_DIR/usr/bin/stratafs-gui"

    # Build package
    dpkg-deb --build "$PACKAGE_DIR" "$OUTPUT_DIR/stratafs_$VERSION-1_$ARCH.deb"

    if [ $? -ne 0 ]; then
        print_error "Package build failed"
        exit 1
    fi

    print_success "Package built: $OUTPUT_DIR/stratafs_$VERSION-1_$ARCH.deb"
}

# Generate checksum
generate_checksum() {
    print_info "Generating checksum..."

    local deb_path="$OUTPUT_DIR/stratafs_$VERSION-1_$ARCH.deb"
    local checksum=$(sha256sum "$deb_path" | cut -d' ' -f1)
    echo "$checksum  $(basename "$deb_path")" > "$deb_path.sha256"

    print_success "Checksum generated: $deb_path.sha256"
    print_info "SHA256: $checksum"
}

# Test package
test_package() {
    print_info "Testing package..."

    local deb_path="$OUTPUT_DIR/stratafs_$VERSION-1_$ARCH.deb"

    # Check package structure
    if dpkg-deb --info "$deb_path" >/dev/null 2>&1; then
        print_success "Package structure is valid"
    else
        print_error "Package structure validation failed"
        exit 1
    fi

    # Show package info
    print_info "Package information:"
    dpkg-deb --info "$deb_path"
}

# Main execution
main() {
    check_dependencies
    setup_package_structure
    copy_files
    create_systemd_service
    create_desktop_entry
    create_icon
    create_man_page
    create_debian_control
    build_package
    generate_checksum
    test_package

    print_success "Debian package build completed!"
    print_info "Package: $OUTPUT_DIR/stratafs_$VERSION-1_$ARCH.deb"
    print_info "Size: $(du -h "$OUTPUT_DIR/stratafs_$VERSION-1_$ARCH.deb" | cut -f1)"

    echo ""
    print_info "Installation instructions:"
    print_info "1. Install: sudo dpkg -i stratafs_$VERSION-1_$ARCH.deb"
    print_info "2. Fix dependencies (if needed): sudo apt-get install -f"
    print_info "3. Start service: sudo systemctl start stratafs"
    print_info "4. Or use desktop app: stratafs-gui"
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
        --maintainer)
            MAINTAINER="$2"
            shift 2
            ;;
        --help)
            echo "StrataFS Debian Package Build Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --version VERSION        Version string (default: 0.2.0)"
            echo "  --binary-path PATH       Path to stratafs binary"
            echo "  --output-dir DIR         Output directory (default: dist)"
            echo "  --arch ARCH             Architecture (default: amd64)"
            echo "  --maintainer EMAIL       Maintainer info"
            echo "  --help                  Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main "$@"