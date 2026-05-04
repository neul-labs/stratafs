#!/bin/bash
# Build script for creating cross-platform release binaries

set -e

# Configuration
VERSION="${VERSION:-$(git describe --tags --always)}"
BUILD_DIR="build/release"
BINARY_NAME="agentfs"
ONNX_VERSION="${ONNX_VERSION:-1.16.3}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Platform configurations: OS/ARCH
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
    "windows/386"
)

check_dependencies() {
    print_info "Checking build dependencies..."

    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed"
        exit 1
    fi

    if ! command -v git >/dev/null 2>&1; then
        print_error "Git is not installed"
        exit 1
    fi

    # Check Go version
    local go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    local required_version="1.21"

    if [ "$(printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1)" != "$required_version" ]; then
        print_error "Go version $required_version or higher is required (found: $go_version)"
        exit 1
    fi

    print_success "Dependencies check passed"
}

download_onnx_runtime() {
    local platform=$1
    local arch=$2
    local os_name=""
    local onnx_arch=""

    case "$platform" in
        "linux")
            os_name="linux"
            ;;
        "darwin")
            os_name="osx"
            ;;
        "windows")
            os_name="win"
            ;;
    esac

    case "$arch" in
        "amd64")
            onnx_arch="x64"
            ;;
        "arm64")
            onnx_arch="arm64"
            ;;
        "arm")
            onnx_arch="arm"
            ;;
        "386")
            onnx_arch="x86"
            ;;
    esac

    local onnx_name="onnxruntime-${os_name}-${onnx_arch}-${ONNX_VERSION}"
    local onnx_url="https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/${onnx_name}.tgz"
    local onnx_dir="build/onnx/${platform}-${arch}"

    if [ "$platform" = "windows" ]; then
        onnx_url="https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/${onnx_name}.zip"
    fi

    if [ -d "$onnx_dir" ]; then
        print_info "ONNX Runtime already downloaded for $platform-$arch"
        return 0
    fi

    print_info "Downloading ONNX Runtime $ONNX_VERSION for $platform-$arch..."
    mkdir -p "$onnx_dir"

    if [ "$platform" = "windows" ]; then
        if ! curl -fsSL "$onnx_url" -o "$onnx_dir/onnx.zip"; then
            print_warning "Failed to download ONNX Runtime for $platform-$arch"
            return 1
        fi
        if command -v unzip >/dev/null 2>&1; then
            unzip -q "$onnx_dir/onnx.zip" -d "$onnx_dir/"
            rm "$onnx_dir/onnx.zip"
        else
            print_warning "unzip not available, skipping ONNX Runtime for Windows"
            return 1
        fi
    else
        if ! curl -fsSL "$onnx_url" | tar -xz -C "$onnx_dir" --strip-components=1; then
            print_warning "Failed to download ONNX Runtime for $platform-$arch"
            return 1
        fi
    fi

    print_success "ONNX Runtime downloaded for $platform-$arch"
}

build_binary() {
    local platform=$1
    local arch=$2
    local output_dir="$BUILD_DIR/$platform-$arch"
    local binary_name="$BINARY_NAME"

    if [ "$platform" = "windows" ]; then
        binary_name="$BINARY_NAME.exe"
    fi

    print_info "Building for $platform/$arch..."

    # Create output directory
    mkdir -p "$output_dir"

    # Set environment variables for cross-compilation
    export GOOS="$platform"
    export GOARCH="$arch"
    export CGO_ENABLED=1

    # Set up ONNX Runtime paths if available
    local onnx_dir="build/onnx/${platform}-${arch}"
    if [ -d "$onnx_dir" ]; then
        local rpath_flag=""
        if [ "$platform" = "darwin" ]; then
            rpath_flag="-Wl,-rpath,@executable_path/lib"
        elif [ "$platform" = "linux" ]; then
            rpath_flag="-Wl,-rpath,\$ORIGIN/lib"
        fi

        export CGO_CFLAGS="-I$PWD/$onnx_dir/include"
        if [ -n "$rpath_flag" ]; then
            export CGO_LDFLAGS="-L$PWD/$onnx_dir/lib -lonnxruntime $rpath_flag"
        else
            export CGO_LDFLAGS="-L$PWD/$onnx_dir/lib -lonnxruntime"
        fi
    else
        print_warning "ONNX Runtime not available for $platform-$arch, building without ML support"
        export CGO_ENABLED=0
    fi

    # Build flags
    local ldflags="-s -w -X main.version=$VERSION -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    local build_tags="fts5"

    # Build the binary
    if ! go build \
        -tags "$build_tags" \
        -ldflags "$ldflags" \
        -o "$output_dir/$binary_name" \
        ./cmd/agentfs; then
        print_error "Failed to build for $platform/$arch"
        return 1
    fi

    # Copy ONNX Runtime libraries if available
    if [ -d "$onnx_dir/lib" ]; then
        if [ "$platform" = "windows" ]; then
            cp "$onnx_dir/lib"/*.dll "$output_dir/" 2>/dev/null || true
        else
            mkdir -p "$output_dir/lib"
            if [ "$platform" = "darwin" ]; then
                cp "$onnx_dir/lib"/*.dylib "$output_dir/lib/" 2>/dev/null || true
            else
                cp "$onnx_dir/lib"/*.so* "$output_dir/lib/" 2>/dev/null || true
            fi
        fi
    fi

    # Create README for the release
    cat > "$output_dir/README.txt" << EOF
AgentFS $VERSION
================

This is the AgentFS binary for $platform/$arch.

Installation:
1. Extract this archive
2. Copy the agentfs binary to a directory in your PATH
3. Run 'agentfs config init' to create initial configuration
4. Run 'agentfs --help' for usage information

For more information, visit: https://github.com/yourusername/agentfs

Build Information:
- Version: $VERSION
- Platform: $platform/$arch
- Build Time: $(date -u +%Y-%m-%dT%H:%M:%SZ)
- Go Version: $(go version)
EOF

    print_success "Built $platform/$arch successfully"
}

create_archives() {
    print_info "Creating release archives..."

    cd "$BUILD_DIR"

    for dir in */; do
        if [ -d "$dir" ]; then
            local platform_arch=$(basename "$dir")
            local platform=$(echo "$platform_arch" | cut -d'-' -f1)

            print_info "Creating archive for $platform_arch..."

            if [ "$platform" = "windows" ]; then
                # Create ZIP for Windows
                if command -v zip >/dev/null 2>&1; then
                    zip -r "${BINARY_NAME}-${VERSION}-${platform_arch}.zip" "$dir"
                else
                    print_warning "zip not available, skipping Windows archive"
                fi
            else
                # Create tar.gz for Unix-like systems
                tar -czf "${BINARY_NAME}-${VERSION}-${platform_arch}.tar.gz" "$dir"
            fi
        fi
    done

    cd - >/dev/null
    print_success "Archives created in $BUILD_DIR"
}

generate_checksums() {
    print_info "Generating checksums..."

    cd "$BUILD_DIR"

    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum *.tar.gz *.zip 2>/dev/null > "checksums.txt" || true
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 *.tar.gz *.zip 2>/dev/null > "checksums.txt" || true
    else
        print_warning "No checksum utility available"
    fi

    cd - >/dev/null
    print_success "Checksums generated"
}

clean_build() {
    print_info "Cleaning previous builds..."
    rm -rf "$BUILD_DIR"
    rm -rf "build/onnx"
    print_success "Build directory cleaned"
}

main() {
    print_info "AgentFS Release Build Script"
    print_info "============================"
    print_info "Version: $VERSION"

    check_dependencies

    if [ "$1" = "clean" ]; then
        clean_build
        exit 0
    fi

    # Clean and create build directory
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"

    # Download ONNX Runtime for all platforms
    for platform_arch in "${PLATFORMS[@]}"; do
        IFS='/' read -r platform arch <<< "$platform_arch"
        download_onnx_runtime "$platform" "$arch"
    done

    # Build for all platforms
    local success_count=0
    local total_count=${#PLATFORMS[@]}

    for platform_arch in "${PLATFORMS[@]}"; do
        IFS='/' read -r platform arch <<< "$platform_arch"
        if build_binary "$platform" "$arch"; then
            ((success_count++))
        fi
    done

    print_info "Built $success_count/$total_count platforms successfully"

    if [ $success_count -gt 0 ]; then
        create_archives
        generate_checksums

        print_success "Release build completed!"
        print_info "Artifacts available in: $BUILD_DIR"

        echo ""
        echo "Files created:"
        ls -la "$BUILD_DIR"
    else
        print_error "No successful builds"
        exit 1
    fi
}

# Parse command line arguments
case "${1:-}" in
    "clean")
        clean_build
        ;;
    "help"|"--help"|"-h")
        echo "AgentFS Release Build Script"
        echo ""
        echo "Usage: $0 [COMMAND]"
        echo ""
        echo "Commands:"
        echo "  (none)    Build release binaries for all platforms"
        echo "  clean     Clean build artifacts"
        echo "  help      Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  VERSION   Version string (default: git describe --tags --always)"
        ;;
    *)
        main "$@"
        ;;
esac
