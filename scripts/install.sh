#!/bin/bash
# AgentFS Installation Script for Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/yourusername/agentfs/main/scripts/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
AGENTFS_VERSION="${AGENTFS_VERSION:-latest}"
FORCE_INSTALL="${FORCE_INSTALL:-false}"

# Platform detection
OS="$(uname -s)"
ARCH="$(uname -m)"

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

detect_platform() {
    case "$OS" in
        Linux*)
            PLATFORM="linux"
            ;;
        Darwin*)
            PLATFORM="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCHITECTURE="amd64"
            ;;
        arm64|aarch64)
            ARCHITECTURE="arm64"
            ;;
        armv7l)
            ARCHITECTURE="arm"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    print_info "Detected platform: $PLATFORM-$ARCHITECTURE"
}

check_dependencies() {
    print_info "Checking dependencies..."

    # Check for curl or wget
    if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
        print_error "Either curl or wget is required for installation"
        exit 1
    fi

    # Check for tar
    if ! command -v tar >/dev/null 2>&1; then
        print_error "tar is required for installation"
        exit 1
    fi

    # Check for SQLite with FTS5
    if command -v sqlite3 >/dev/null 2>&1; then
        if ! echo ".quit" | sqlite3 ":memory:" "CREATE VIRTUAL TABLE test USING fts5(content);" >/dev/null 2>&1; then
            print_warning "SQLite FTS5 support not detected. Some features may not work properly."
        fi
    else
        print_warning "SQLite not found. Please install sqlite3 for full functionality."
    fi

    print_success "Dependencies check completed"
}

get_latest_version() {
    if [ "$AGENTFS_VERSION" = "latest" ]; then
        print_info "Fetching latest version..."
        if command -v curl >/dev/null 2>&1; then
            AGENTFS_VERSION=$(curl -fsSL https://api.github.com/repos/yourusername/agentfs/releases/latest | grep '"tag_name"' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')
        elif command -v wget >/dev/null 2>&1; then
            AGENTFS_VERSION=$(wget -qO- https://api.github.com/repos/yourusername/agentfs/releases/latest | grep '"tag_name"' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')
        fi

        if [ -z "$AGENTFS_VERSION" ]; then
            print_error "Failed to fetch latest version"
            exit 1
        fi
    fi

    print_info "Installing AgentFS version: $AGENTFS_VERSION"
}

download_agentfs() {
    local download_url="https://github.com/yourusername/agentfs/releases/download/${AGENTFS_VERSION}/agentfs-${AGENTFS_VERSION}-${PLATFORM}-${ARCHITECTURE}.tar.gz"
    local tmp_dir=$(mktemp -d)
    local tmp_file="$tmp_dir/agentfs.tar.gz"

    print_info "Downloading AgentFS from: $download_url"

    if command -v curl >/dev/null 2>&1; then
        if ! curl -fsSL "$download_url" -o "$tmp_file"; then
            print_error "Failed to download AgentFS"
            rm -rf "$tmp_dir"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "$download_url" -O "$tmp_file"; then
            print_error "Failed to download AgentFS"
            rm -rf "$tmp_dir"
            exit 1
        fi
    fi

    print_info "Extracting AgentFS..."
    if ! tar -xzf "$tmp_file" -C "$tmp_dir"; then
        print_error "Failed to extract AgentFS"
        rm -rf "$tmp_dir"
        exit 1
    fi

    # Find the binary
    BINARY_PATH=$(find "$tmp_dir" -name "agentfs" -type f | head -1)
    if [ -z "$BINARY_PATH" ]; then
        print_error "AgentFS binary not found in archive"
        rm -rf "$tmp_dir"
        exit 1
    fi

    echo "$BINARY_PATH"
}

install_agentfs() {
    local binary_path="$1"
    local install_path="$INSTALL_DIR/agentfs"

    # Check if already installed
    if [ -f "$install_path" ] && [ "$FORCE_INSTALL" != "true" ]; then
        print_warning "AgentFS is already installed at $install_path"
        read -p "Do you want to overwrite it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Installation cancelled"
            exit 0
        fi
    fi

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        print_info "Creating install directory: $INSTALL_DIR"
        if ! mkdir -p "$INSTALL_DIR"; then
            print_error "Failed to create install directory. Try running with sudo."
            exit 1
        fi
    fi

    print_info "Installing AgentFS to $install_path..."
    if ! cp "$binary_path" "$install_path"; then
        print_error "Failed to install AgentFS. Try running with sudo."
        exit 1
    fi

    if ! chmod +x "$install_path"; then
        print_error "Failed to make AgentFS executable"
        exit 1
    fi

    print_success "AgentFS installed successfully!"
}

setup_config() {
    local config_dir="$HOME/.agentfs"
    local config_file="$config_dir/config.json"

    if [ ! -d "$config_dir" ]; then
        print_info "Creating configuration directory: $config_dir"
        mkdir -p "$config_dir"
    fi

    if [ ! -f "$config_file" ]; then
        print_info "Creating default configuration..."
        agentfs config init
        print_success "Default configuration created at $config_file"
    else
        print_info "Configuration already exists at $config_file"
    fi
}

verify_installation() {
    print_info "Verifying installation..."

    if ! command -v agentfs >/dev/null 2>&1; then
        print_warning "AgentFS not found in PATH. You may need to add $INSTALL_DIR to your PATH."
        echo "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo "export PATH=\"$INSTALL_DIR:\$PATH\""
        return
    fi

    local version=$(agentfs --version 2>/dev/null || echo "unknown")
    print_success "AgentFS $version is ready to use!"

    print_info "Next steps:"
    echo "  1. Initialize configuration: agentfs config init"
    echo "  2. Add storage sources: agentfs source add"
    echo "  3. Start AgentFS: agentfs"
    echo ""
    echo "For more help, run: agentfs --help"
}

cleanup() {
    if [ -n "$tmp_dir" ] && [ -d "$tmp_dir" ]; then
        rm -rf "$tmp_dir"
    fi
}

main() {
    trap cleanup EXIT

    print_info "AgentFS Installation Script"
    print_info "============================"

    detect_platform
    check_dependencies
    get_latest_version

    local binary_path=$(download_agentfs)
    install_agentfs "$binary_path"
    setup_config
    verify_installation

    print_success "Installation completed!"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --version)
            AGENTFS_VERSION="$2"
            shift 2
            ;;
        --force)
            FORCE_INSTALL="true"
            shift
            ;;
        --help)
            echo "AgentFS Installation Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --install-dir DIR    Installation directory (default: /usr/local/bin)"
            echo "  --version VERSION    AgentFS version to install (default: latest)"
            echo "  --force             Force installation even if already installed"
            echo "  --help              Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main "$@"