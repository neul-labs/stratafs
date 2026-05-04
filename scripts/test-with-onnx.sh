#!/usr/bin/env bash
# Run go test with ONNX Runtime enabled (downloads runtime if missing)

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ONNX_VERSION="${ONNX_VERSION:-1.16.3}"
TARGET_OS="${TARGET_OS:-$(go env GOOS)}"
TARGET_ARCH="${TARGET_ARCH:-$(go env GOARCH)}"
PKGS=("${@:-./...}")

BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

log() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Ensure ONNX Runtime is available
log "Ensuring ONNX Runtime $ONNX_VERSION for ${TARGET_OS}/${TARGET_ARCH}"
ONNX_VERSION="$ONNX_VERSION" TARGET_OS="$TARGET_OS" TARGET_ARCH="$TARGET_ARCH" bash "$ROOT_DIR/scripts/get-onnx-runtime.sh"

ONNX_DIR="$ROOT_DIR/build/onnx/${TARGET_OS}-${TARGET_ARCH}"
if [ ! -d "$ONNX_DIR/lib" ]; then
    log_error "ONNX Runtime libraries not found in $ONNX_DIR/lib"
    exit 1
fi

export CGO_ENABLED=1
export CGO_CFLAGS="-I$ONNX_DIR/include"
export CGO_LDFLAGS="-L$ONNX_DIR/lib -lonnxruntime"

case "$TARGET_OS" in
    linux)
        export LD_LIBRARY_PATH="$ONNX_DIR/lib${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
        ;;
    darwin)
        export DYLD_LIBRARY_PATH="$ONNX_DIR/lib${DYLD_LIBRARY_PATH:+:$DYLD_LIBRARY_PATH}"
        ;;
    windows)
        export PATH="$ONNX_DIR/lib${PATH:+;$PATH}"
        ;;
esac

GOCACHE="${GOCACHE:-$ROOT_DIR/.gocache}"
export GOCACHE

log "Running go test with tags fts5 for packages: ${PKGS[*]}"
go test -tags "fts5" "${PKGS[@]}"
log_success "Tests completed successfully"
