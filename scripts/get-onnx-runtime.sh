#!/usr/bin/env bash
# Download ONNX Runtime prebuilt binaries for the host (or specified) platform.

set -euo pipefail

ONNX_VERSION="${ONNX_VERSION:-1.16.3}"
TARGET_OS="${TARGET_OS:-}"
TARGET_ARCH="${TARGET_ARCH:-}"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

detect_platform() {
    if [ -z "$TARGET_OS" ]; then
        local uname_s
        uname_s="$(uname -s | tr '[:upper:]' '[:lower:]')"
        case "$uname_s" in
            linux*) TARGET_OS="linux" ;;
            darwin*) TARGET_OS="darwin" ;;
            msys*|mingw*|cygwin*) TARGET_OS="windows" ;;
            *) log_error "Unsupported OS: $uname_s"; exit 1 ;;
        esac
    fi

    if [ -z "$TARGET_ARCH" ]; then
        local uname_m
        uname_m="$(uname -m)"
        case "$uname_m" in
            x86_64|amd64) TARGET_ARCH="amd64" ;;
            arm64|aarch64) TARGET_ARCH="arm64" ;;
            armv7l|armv6l) TARGET_ARCH="arm" ;;
            i386|i686) TARGET_ARCH="386" ;;
            *) log_error "Unsupported architecture: $uname_m"; exit 1 ;;
        esac
    fi
}

map_onnx_arch() {
    case "$1" in
        amd64) echo "x64" ;;
        arm64) echo "arm64" ;;
        arm) echo "arm" ;;
        386) echo "x86" ;;
        *) log_error "Unsupported architecture for ONNX Runtime: $1"; exit 1 ;;
    esac
}

download_runtime() {
    local platform="$1"
    local arch="$2"
    local onnx_arch
    onnx_arch="$(map_onnx_arch "$arch")"
    local os_name="$platform"

    if [ "$platform" = "darwin" ]; then
        os_name="osx"
    elif [ "$platform" = "windows" ]; then
        os_name="win"
    fi

    local dest_dir="build/onnx/${platform}-${arch}"
    mkdir -p "$dest_dir"
    if [ -d "$dest_dir" ]; then
        log_info "ONNX Runtime already present at $dest_dir"
        return
    fi

    local archive_name="onnxruntime-${os_name}-${onnx_arch}-${ONNX_VERSION}"
    local url="https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/${archive_name}"

    if [ "$platform" = "windows" ]; then
        local zip="${archive_name}.zip"
        log_info "Downloading ${zip}..."
        if ! curl -fsSL "${url}.zip" -o "$dest_dir/runtime.zip"; then
            log_error "Failed to download ${zip}. Check network connectivity or download manually."
            rm -rf "$dest_dir"
            exit 1
        fi
        if command -v unzip >/dev/null 2>&1; then
            unzip -q "$dest_dir/runtime.zip" -d "$dest_dir"
            rm "$dest_dir/runtime.zip"
        else
            log_error "unzip not available; install unzip or extract ${zip} manually."
            exit 1
        fi
    else
        local tgz="${archive_name}.tgz"
        log_info "Downloading ${tgz}..."
        if ! curl -fsSL "${url}.tgz" | tar -xz -C "$dest_dir" --strip-components=1; then
            log_error "Failed to download ${tgz}. Check network connectivity or download manually."
            rm -rf "$dest_dir"
            exit 1
        fi
    fi

    log_success "ONNX Runtime $ONNX_VERSION downloaded to $dest_dir"
}

main() {
    detect_platform
    log_info "Preparing ONNX Runtime $ONNX_VERSION for ${TARGET_OS}/${TARGET_ARCH}"
    download_runtime "$TARGET_OS" "$TARGET_ARCH"
    echo
    echo "Next steps:"
    echo "  • CGO_CFLAGS and CGO_LDFLAGS should point at $(pwd)/build/onnx/${TARGET_OS}-${TARGET_ARCH}/include and lib."
    echo "  • go build example:"
    echo "      CGO_ENABLED=1 \\"
    echo "      CGO_CFLAGS=\"-I$(pwd)/build/onnx/${TARGET_OS}-${TARGET_ARCH}/include\" \\"
    echo "      CGO_LDFLAGS=\"-L$(pwd)/build/onnx/${TARGET_OS}-${TARGET_ARCH}/lib -lonnxruntime\" \\"
    echo "      go build -tags fts5 ./cmd/agentfs"
}

main "$@"
