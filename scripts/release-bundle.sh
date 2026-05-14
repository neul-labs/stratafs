#!/bin/bash
# Release Bundle Script
# Creates complete release bundles with stratafs, stratafs-ui, and ONNX runtime

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")}"
BUILD_DIR="$PROJECT_ROOT/build/release"
ONNX_VERSION="${ONNX_VERSION:-1.16.3}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Detect current platform
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$arch" in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l) arch="arm" ;;
    esac

    echo "${os}/${arch}"
}

# Download ONNX Runtime for current platform
download_onnx() {
    local platform=$1
    local arch=$2
    local onnx_dir="$BUILD_DIR/onnx"

    mkdir -p "$onnx_dir"

    local os_name=""
    local onnx_arch=""

    case "$platform" in
        linux) os_name="linux" ;;
        darwin) os_name="osx" ;;
        *) warn "ONNX Runtime not supported for $platform"; return 1 ;;
    esac

    case "$arch" in
        amd64) onnx_arch="x64" ;;
        arm64) onnx_arch="arm64" ;;
        *) warn "ONNX Runtime arch not supported: $arch"; return 1 ;;
    esac

    local onnx_name="onnxruntime-${os_name}-${onnx_arch}-${ONNX_VERSION}"
    local onnx_url="https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/${onnx_name}.tgz"

    if [ -f "$onnx_dir/lib/libonnxruntime.so" ] || [ -f "$onnx_dir/lib/libonnxruntime.dylib" ]; then
        info "ONNX Runtime already available"
        return 0
    fi

    info "Downloading ONNX Runtime $ONNX_VERSION..."
    curl -fsSL "$onnx_url" | tar -xz -C "$onnx_dir" --strip-components=1
    success "ONNX Runtime downloaded"
}

# Build stratafs daemon
build_daemon() {
    info "Building stratafs daemon..."
    cd "$PROJECT_ROOT"

    local ldflags="-s -w -X main.version=$VERSION"

    go build \
        -tags "fts5" \
        -ldflags "$ldflags" \
        -o "$BUILD_DIR/bin/stratafs" \
        ./cmd/stratafs

    success "stratafs daemon built"
}

# Build desktop UI
build_desktop_ui() {
    local desktop_dir="$PROJECT_ROOT/desktop/stratafs-ui"

    if [ ! -d "$desktop_dir" ]; then
        warn "Desktop UI directory not found, skipping"
        return 1
    fi

    if ! command -v wails >/dev/null 2>&1; then
        warn "Wails not installed, skipping desktop UI"
        return 1
    fi

    info "Building stratafs-ui desktop app..."
    cd "$desktop_dir"

    wails build -platform "$(detect_platform)"

    cp "$desktop_dir/build/bin/stratafs-ui" "$BUILD_DIR/bin/"
    success "stratafs-ui built"
}

# Create Linux bundle
create_linux_bundle() {
    local bundle_dir="$BUILD_DIR/stratafs-${VERSION}-linux-amd64"

    info "Creating Linux bundle..."

    mkdir -p "$bundle_dir/bin"
    mkdir -p "$bundle_dir/lib"
    mkdir -p "$bundle_dir/share/applications"
    mkdir -p "$bundle_dir/share/systemd/user"

    # Copy binaries
    cp "$BUILD_DIR/bin/stratafs" "$bundle_dir/bin/"
    [ -f "$BUILD_DIR/bin/stratafs-ui" ] && cp "$BUILD_DIR/bin/stratafs-ui" "$bundle_dir/bin/"

    # Copy ONNX runtime
    if [ -d "$BUILD_DIR/onnx/lib" ]; then
        cp -P "$BUILD_DIR/onnx/lib"/*.so* "$bundle_dir/lib/"
    fi

    # Create desktop entry
    cat > "$bundle_dir/share/applications/stratafs-ui.desktop" << EOF
[Desktop Entry]
Name=StrataFS Control Panel
Comment=Manage your StrataFS semantic filesystem
Exec=stratafs-ui
Icon=stratafs-ui
Terminal=false
Type=Application
Categories=Utility;FileTools;
EOF

    # Create systemd service
    cat > "$bundle_dir/share/systemd/user/stratafs.service" << EOF
[Unit]
Description=StrataFS Semantic Filesystem Daemon
After=network.target

[Service]
Type=simple
ExecStart=%h/.local/bin/stratafs
Environment=LD_LIBRARY_PATH=%h/.local/lib
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF

    # Create install script
    cat > "$bundle_dir/install.sh" << 'EOF'
#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local}"

echo "Installing StrataFS to $INSTALL_DIR..."

mkdir -p "$INSTALL_DIR/bin"
mkdir -p "$INSTALL_DIR/lib"
mkdir -p "$INSTALL_DIR/share/applications"
mkdir -p "$HOME/.config/systemd/user"

cp bin/* "$INSTALL_DIR/bin/"
cp lib/* "$INSTALL_DIR/lib/" 2>/dev/null || true
cp share/applications/* "$INSTALL_DIR/share/applications/" 2>/dev/null || true
cp share/systemd/user/* "$HOME/.config/systemd/user/" 2>/dev/null || true

# Update desktop database
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "$INSTALL_DIR/share/applications" 2>/dev/null || true
fi

echo ""
echo "Installation complete!"
echo ""
echo "Add to your shell profile:"
echo '  export PATH="$HOME/.local/bin:$PATH"'
echo '  export LD_LIBRARY_PATH="$HOME/.local/lib:$LD_LIBRARY_PATH"'
echo ""
echo "To start StrataFS automatically:"
echo "  systemctl --user daemon-reload"
echo "  systemctl --user enable stratafs"
echo "  systemctl --user start stratafs"
echo ""
echo "Initialize configuration:"
echo "  stratafs config init"
EOF
    chmod +x "$bundle_dir/install.sh"

    # Create README
    cat > "$bundle_dir/README.md" << EOF
# StrataFS $VERSION

## Quick Start

1. Run the installer:
   \`\`\`bash
   ./install.sh
   \`\`\`

2. Add to your shell profile (~/.bashrc or ~/.zshrc):
   \`\`\`bash
   export PATH="\$HOME/.local/bin:\$PATH"
   export LD_LIBRARY_PATH="\$HOME/.local/lib:\$LD_LIBRARY_PATH"
   \`\`\`

3. Initialize configuration:
   \`\`\`bash
   stratafs config init
   \`\`\`

4. Start the service:
   \`\`\`bash
   systemctl --user enable stratafs
   systemctl --user start stratafs
   \`\`\`

5. Launch the control panel:
   \`\`\`bash
   stratafs-ui
   \`\`\`

## Contents

- \`bin/stratafs\` - Core daemon with indexing and search
- \`bin/stratafs-ui\` - Desktop control panel
- \`lib/\` - ONNX Runtime libraries for embeddings
- \`share/\` - Desktop entries and systemd service

## Documentation

See https://github.com/neul-labs/stratafs for full documentation.
EOF

    # Create archive
    cd "$BUILD_DIR"
    tar -czf "stratafs-${VERSION}-linux-amd64.tar.gz" "$(basename "$bundle_dir")"

    success "Linux bundle created: stratafs-${VERSION}-linux-amd64.tar.gz"
}

# Generate checksums
generate_checksums() {
    info "Generating checksums..."
    cd "$BUILD_DIR"

    sha256sum *.tar.gz *.zip 2>/dev/null > checksums.txt || \
    shasum -a 256 *.tar.gz *.zip 2>/dev/null > checksums.txt || \
    warn "Could not generate checksums"

    success "Checksums generated"
}

# Main
main() {
    echo ""
    info "StrataFS Release Bundle Builder"
    info "==============================="
    info "Version: $VERSION"
    echo ""

    local platform_arch=$(detect_platform)
    local platform=$(echo "$platform_arch" | cut -d'/' -f1)
    local arch=$(echo "$platform_arch" | cut -d'/' -f2)

    info "Building for: $platform/$arch"

    # Clean and setup
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR/bin"

    # Download ONNX Runtime
    download_onnx "$platform" "$arch" || true

    # Build components
    build_daemon
    build_desktop_ui || true

    # Create platform-specific bundle
    case "$platform" in
        linux)
            create_linux_bundle
            ;;
        darwin)
            info "macOS bundle creation not yet implemented"
            ;;
        *)
            warn "Unknown platform: $platform"
            ;;
    esac

    generate_checksums

    echo ""
    success "Release bundle complete!"
    info "Output: $BUILD_DIR"
    echo ""
    ls -lh "$BUILD_DIR"/*.tar.gz "$BUILD_DIR"/*.zip 2>/dev/null || true
}

# Handle arguments
case "${1:-}" in
    clean)
        info "Cleaning build directory..."
        rm -rf "$BUILD_DIR"
        success "Clean complete"
        ;;
    help|--help|-h)
        echo "StrataFS Release Bundle Script"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  (none)   Build release bundle for current platform"
        echo "  clean    Remove build artifacts"
        echo "  help     Show this help"
        echo ""
        echo "Environment:"
        echo "  VERSION       Version string (default: git tag)"
        echo "  ONNX_VERSION  ONNX Runtime version (default: 1.16.3)"
        ;;
    *)
        main
        ;;
esac
