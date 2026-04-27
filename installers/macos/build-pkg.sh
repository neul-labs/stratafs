#!/bin/bash
# AgentFS macOS PKG Installer Build Script

set -e

# Configuration
VERSION="${VERSION:-0.2.0}"
BINARY_PATH="${BINARY_PATH:-../../build/darwin-amd64/agentfs}"
OUTPUT_DIR="${OUTPUT_DIR:-dist}"
SIGN_IDENTITY="${SIGN_IDENTITY:-}"
NOTARIZE="${NOTARIZE:-false}"
APPLE_ID="${APPLE_ID:-}"
APPLE_PASSWORD="${APPLE_PASSWORD:-}"
TEAM_ID="${TEAM_ID:-}"

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
BUILD_DIR="$SCRIPT_DIR/build"
PAYLOAD_DIR="$BUILD_DIR/payload"
SCRIPTS_DIR="$BUILD_DIR/scripts"
RESOURCES_DIR="$BUILD_DIR/resources"

print_info "Building AgentFS macOS Installer v$VERSION"

# Check dependencies
check_dependencies() {
    print_info "Checking dependencies..."

    if ! command -v pkgbuild >/dev/null 2>&1; then
        print_error "pkgbuild not found. Are you running on macOS?"
        exit 1
    fi

    if ! command -v productbuild >/dev/null 2>&1; then
        print_error "productbuild not found. Are you running on macOS?"
        exit 1
    fi

    if [ ! -f "$BINARY_PATH" ]; then
        print_error "Binary not found at: $BINARY_PATH"
        print_info "Please build the macOS binary first:"
        print_info "  GOOS=darwin GOARCH=amd64 go build -tags 'fts5' -o build/darwin-amd64/agentfs ./cmd/agentfs"
        exit 1
    fi

    print_success "Dependencies check passed"
}

# Clean and create build directories
setup_build_dirs() {
    print_info "Setting up build directories..."

    rm -rf "$BUILD_DIR"
    mkdir -p "$PAYLOAD_DIR/usr/local/bin"
    mkdir -p "$PAYLOAD_DIR/usr/local/share/agentfs"
    mkdir -p "$PAYLOAD_DIR/Applications/AgentFS.app/Contents/MacOS"
    mkdir -p "$PAYLOAD_DIR/Applications/AgentFS.app/Contents/Resources"
    mkdir -p "$SCRIPTS_DIR"
    mkdir -p "$RESOURCES_DIR"
    mkdir -p "$OUTPUT_DIR"
}

# Copy binary and resources
copy_files() {
    print_info "Copying files..."

    # Copy main binary
    cp "$BINARY_PATH" "$PAYLOAD_DIR/usr/local/bin/agentfs"
    chmod +x "$PAYLOAD_DIR/usr/local/bin/agentfs"

    # Copy ONNX Runtime libraries if available
    ONNX_LIB_DIR="$(dirname "$BINARY_PATH")/lib"
    if [ -d "$ONNX_LIB_DIR" ]; then
        cp -r "$ONNX_LIB_DIR" "$PAYLOAD_DIR/usr/local/share/agentfs/"
        print_info "Copied ONNX Runtime libraries"
    fi

    # Copy documentation
    if [ -f "$PROJECT_ROOT/README.md" ]; then
        cp "$PROJECT_ROOT/README.md" "$PAYLOAD_DIR/usr/local/share/agentfs/"
    fi

    if [ -f "$PROJECT_ROOT/LICENSE" ]; then
        cp "$PROJECT_ROOT/LICENSE" "$PAYLOAD_DIR/usr/local/share/agentfs/"
    fi

    # Create default configuration
    cat > "$PAYLOAD_DIR/usr/local/share/agentfs/default-config.json" << 'EOF'
{
  "version": "0.2.0",
  "agent_dir": ".agentfs",
  "global_dir": "~/.agentfs",
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
EOF
}

# Create macOS app bundle
create_app_bundle() {
    print_info "Creating macOS app bundle..."

    local app_dir="$PAYLOAD_DIR/Applications/AgentFS.app"

    # Copy binary to app bundle
    cp "$BINARY_PATH" "$app_dir/Contents/MacOS/agentfs"
    chmod +x "$app_dir/Contents/MacOS/agentfs"

    # Create Info.plist
    cat > "$app_dir/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>agentfs</string>
    <key>CFBundleIdentifier</key>
    <string>com.agentfs.agentfs</string>
    <key>CFBundleName</key>
    <string>AgentFS</string>
    <key>CFBundleDisplayName</key>
    <string>AgentFS</string>
    <key>CFBundleVersion</key>
    <string>$VERSION</string>
    <key>CFBundleShortVersionString</key>
    <string>$VERSION</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleSignature</key>
    <string>????</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
</dict>
</plist>
EOF

    # Create app icon (placeholder)
    if [ ! -f "$SCRIPT_DIR/assets/agentfs.icns" ]; then
        print_warning "App icon not found, creating placeholder"
        mkdir -p "$SCRIPT_DIR/assets"
        # This would normally be a proper ICNS file
        touch "$app_dir/Contents/Resources/agentfs.icns"
    else
        cp "$SCRIPT_DIR/assets/agentfs.icns" "$app_dir/Contents/Resources/"
    fi
}

# Create installer scripts
create_installer_scripts() {
    print_info "Creating installer scripts..."

    # Pre-install script
    cat > "$SCRIPTS_DIR/preinstall" << 'EOF'
#!/bin/bash
# Stop AgentFS if running
pkill -f agentfs || true

# Stop LaunchAgent if running
launchctl unload ~/Library/LaunchAgents/com.agentfs.agentfs.plist 2>/dev/null || true

exit 0
EOF

    # Post-install script
    cat > "$SCRIPTS_DIR/postinstall" << 'EOF'
#!/bin/bash

# Ensure binary is executable
chmod +x /usr/local/bin/agentfs

# Create user's AgentFS directory
USER_HOME="${3%/*}"
USER_AGENTFS_DIR="$USER_HOME/.agentfs"

if [ ! -d "$USER_AGENTFS_DIR" ]; then
    mkdir -p "$USER_AGENTFS_DIR"
    chown -R "${3##*/}:staff" "$USER_AGENTFS_DIR"
fi

# Initialize configuration if it doesn't exist
if [ ! -f "$USER_AGENTFS_DIR/config.json" ]; then
    sudo -u "${3##*/}" /usr/local/bin/agentfs config init --config-dir="$USER_AGENTFS_DIR"
fi

# Create LaunchAgent plist for auto-start
LAUNCH_AGENT_DIR="$USER_HOME/Library/LaunchAgents"
LAUNCH_AGENT_PLIST="$LAUNCH_AGENT_DIR/com.agentfs.agentfs.plist"

mkdir -p "$LAUNCH_AGENT_DIR"

cat > "$LAUNCH_AGENT_PLIST" << PLIST_EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.agentfs.agentfs</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/agentfs</string>
        <string>--config-dir=$USER_AGENTFS_DIR</string>
    </array>
    <key>RunAtLoad</key>
    <false/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardOutPath</key>
    <string>$USER_AGENTFS_DIR/agentfs.log</string>
    <key>StandardErrorPath</key>
    <string>$USER_AGENTFS_DIR/agentfs.error.log</string>
    <key>WorkingDirectory</key>
    <string>$USER_HOME</string>
</dict>
</plist>
PLIST_EOF

chown "${3##*/}:staff" "$LAUNCH_AGENT_PLIST"

# Ask user if they want to start AgentFS automatically
echo "AgentFS has been installed successfully!"
echo "To start AgentFS automatically at login, run:"
echo "  launchctl load ~/Library/LaunchAgents/com.agentfs.agentfs.plist"
echo ""
echo "To start AgentFS now, run:"
echo "  agentfs"

exit 0
EOF

    chmod +x "$SCRIPTS_DIR/preinstall"
    chmod +x "$SCRIPTS_DIR/postinstall"
}

# Create distribution XML
create_distribution() {
    print_info "Creating distribution file..."

    cat > "$BUILD_DIR/distribution.xml" << EOF
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
    <title>AgentFS $VERSION</title>
    <organization>com.agentfs</organization>
    <domains enable_anywhere="true"/>
    <options customize="never" require-scripts="false" rootVolumeOnly="true"/>

    <!-- Welcome -->
    <welcome file="welcome.html" mime-type="text/html"/>

    <!-- License -->
    <license file="license.txt"/>

    <!-- Conclusion -->
    <conclusion file="conclusion.html" mime-type="text/html"/>

    <!-- Background -->
    <background file="background.png" mime-type="image/png" alignment="topleft" scaling="tofit"/>

    <pkg-ref id="com.agentfs.agentfs"/>

    <options customize="never" require-scripts="false"/>

    <choices-outline>
        <line choice="default">
            <line choice="com.agentfs.agentfs"/>
        </line>
    </choices-outline>

    <choice id="default"/>
    <choice id="com.agentfs.agentfs" visible="false">
        <pkg-ref id="com.agentfs.agentfs"/>
    </choice>

    <pkg-ref id="com.agentfs.agentfs" version="$VERSION" onConclusion="none">agentfs-core.pkg</pkg-ref>
</installer-gui-script>
EOF
}

# Create installer resources
create_resources() {
    print_info "Creating installer resources..."

    # Welcome HTML
    cat > "$RESOURCES_DIR/welcome.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; margin: 20px; }
        h1 { color: #007AFF; }
        .feature { margin: 10px 0; }
        .icon { color: #34C759; margin-right: 8px; }
    </style>
</head>
<body>
    <h1>Welcome to AgentFS</h1>
    <p>AgentFS transforms your filesystem into an intelligent, searchable knowledge base for AI agents.</p>

    <div class="feature"><span class="icon">✓</span>Semantic search across all your files</div>
    <div class="feature"><span class="icon">✓</span>AI agent integration via Model Context Protocol</div>
    <div class="feature"><span class="icon">✓</span>Multi-storage support (local, cloud)</div>
    <div class="feature"><span class="icon">✓</span>Real-time file monitoring and indexing</div>

    <p>This installer will guide you through the setup process.</p>
</body>
</html>
EOF

    # License
    if [ -f "$PROJECT_ROOT/LICENSE" ]; then
        cp "$PROJECT_ROOT/LICENSE" "$RESOURCES_DIR/license.txt"
    else
        echo "MIT License - See project repository for details" > "$RESOURCES_DIR/license.txt"
    fi

    # Conclusion HTML
    cat > "$RESOURCES_DIR/conclusion.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; margin: 20px; }
        h1 { color: #34C759; }
        .command { background: #f5f5f5; padding: 10px; border-radius: 5px; font-family: monospace; }
        .step { margin: 15px 0; }
    </style>
</head>
<body>
    <h1>Installation Complete!</h1>
    <p>AgentFS has been successfully installed on your Mac.</p>

    <div class="step">
        <strong>Next Steps:</strong>
        <ol>
            <li>Open Terminal</li>
            <li>Initialize AgentFS: <div class="command">agentfs config init</div></li>
            <li>Add storage sources: <div class="command">agentfs source add</div></li>
            <li>Start AgentFS: <div class="command">agentfs</div></li>
        </ol>
    </div>

    <div class="step">
        <strong>Auto-start (optional):</strong>
        <div class="command">launchctl load ~/Library/LaunchAgents/com.agentfs.agentfs.plist</div>
    </div>

    <p>For help and documentation, visit: <a href="https://github.com/yourusername/agentfs">github.com/yourusername/agentfs</a></p>
</body>
</html>
EOF

    # Create placeholder background image
    if [ ! -f "$SCRIPT_DIR/assets/background.png" ]; then
        print_warning "Background image not found, installer will use default"
        # Create a 1x1 transparent PNG as placeholder
        echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==" | base64 -d > "$RESOURCES_DIR/background.png"
    else
        cp "$SCRIPT_DIR/assets/background.png" "$RESOURCES_DIR/"
    fi
}

# Build the package
build_package() {
    print_info "Building package..."

    # Build core package
    pkgbuild \
        --root "$PAYLOAD_DIR" \
        --scripts "$SCRIPTS_DIR" \
        --identifier "com.agentfs.agentfs" \
        --version "$VERSION" \
        --install-location "/" \
        "$BUILD_DIR/agentfs-core.pkg"

    # Build product (installer)
    productbuild \
        --distribution "$BUILD_DIR/distribution.xml" \
        --resources "$RESOURCES_DIR" \
        --package-path "$BUILD_DIR" \
        "$OUTPUT_DIR/AgentFS-$VERSION.pkg"

    print_success "Package built: $OUTPUT_DIR/AgentFS-$VERSION.pkg"
}

# Sign the package (if requested)
sign_package() {
    if [ -n "$SIGN_IDENTITY" ]; then
        print_info "Signing package with identity: $SIGN_IDENTITY"

        # Sign the core package first
        productsign \
            --sign "$SIGN_IDENTITY" \
            "$BUILD_DIR/agentfs-core.pkg" \
            "$BUILD_DIR/agentfs-core-signed.pkg"

        mv "$BUILD_DIR/agentfs-core-signed.pkg" "$BUILD_DIR/agentfs-core.pkg"

        # Sign the final installer
        productsign \
            --sign "$SIGN_IDENTITY" \
            "$OUTPUT_DIR/AgentFS-$VERSION.pkg" \
            "$OUTPUT_DIR/AgentFS-$VERSION-signed.pkg"

        mv "$OUTPUT_DIR/AgentFS-$VERSION-signed.pkg" "$OUTPUT_DIR/AgentFS-$VERSION.pkg"

        print_success "Package signed successfully"
    fi
}

# Notarize the package (if requested)
notarize_package() {
    if [ "$NOTARIZE" = "true" ]; then
        if [ -z "$APPLE_ID" ] || [ -z "$APPLE_PASSWORD" ] || [ -z "$TEAM_ID" ]; then
            print_error "Notarization requested but Apple ID, password, or team ID not provided"
            exit 1
        fi

        print_info "Submitting package for notarization..."

        # Upload for notarization
        xcrun notarytool submit \
            "$OUTPUT_DIR/AgentFS-$VERSION.pkg" \
            --apple-id "$APPLE_ID" \
            --password "$APPLE_PASSWORD" \
            --team-id "$TEAM_ID" \
            --wait

        # Staple the notarization
        xcrun stapler staple "$OUTPUT_DIR/AgentFS-$VERSION.pkg"

        print_success "Package notarized and stapled"
    fi
}

# Generate checksum
generate_checksum() {
    print_info "Generating checksum..."

    local pkg_path="$OUTPUT_DIR/AgentFS-$VERSION.pkg"
    local checksum=$(shasum -a 256 "$pkg_path" | cut -d' ' -f1)
    echo "$checksum  $(basename "$pkg_path")" > "$pkg_path.sha256"

    print_success "Checksum generated: $pkg_path.sha256"
    print_info "SHA256: $checksum"
}

# Main execution
main() {
    check_dependencies
    setup_build_dirs
    copy_files
    create_app_bundle
    create_installer_scripts
    create_distribution
    create_resources
    build_package
    sign_package
    notarize_package
    generate_checksum

    print_success "macOS installer build completed!"
    print_info "Installer: $OUTPUT_DIR/AgentFS-$VERSION.pkg"
    print_info "Size: $(du -h "$OUTPUT_DIR/AgentFS-$VERSION.pkg" | cut -f1)"

    if [ -n "$SIGN_IDENTITY" ]; then
        print_info "Signed: Yes"
    fi

    if [ "$NOTARIZE" = "true" ]; then
        print_info "Notarized: Yes"
    fi

    echo ""
    print_info "Installation instructions:"
    print_info "1. Double-click AgentFS-$VERSION.pkg"
    print_info "2. Follow the installation wizard"
    print_info "3. Open Terminal and run 'agentfs config init'"
    print_info "4. Start using AgentFS!"
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
        --sign)
            SIGN_IDENTITY="$2"
            shift 2
            ;;
        --notarize)
            NOTARIZE="true"
            shift
            ;;
        --apple-id)
            APPLE_ID="$2"
            shift 2
            ;;
        --apple-password)
            APPLE_PASSWORD="$2"
            shift 2
            ;;
        --team-id)
            TEAM_ID="$2"
            shift 2
            ;;
        --help)
            echo "AgentFS macOS Installer Build Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --version VERSION          Version string (default: 0.2.0)"
            echo "  --binary-path PATH         Path to agentfs binary"
            echo "  --output-dir DIR           Output directory (default: dist)"
            echo "  --sign IDENTITY            Code signing identity"
            echo "  --notarize                 Enable notarization"
            echo "  --apple-id ID              Apple ID for notarization"
            echo "  --apple-password PASS      App-specific password"
            echo "  --team-id ID               Team ID for notarization"
            echo "  --help                     Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main "$@"