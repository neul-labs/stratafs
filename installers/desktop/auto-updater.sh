#!/bin/bash
# StrataFS Auto-Updater Script
# Cross-platform automatic update checker and installer

set -e

# Configuration
GITHUB_REPO="dipankar/stratafs"
CURRENT_VERSION="${STRATAFS_VERSION:-0.2.0}"
CONFIG_DIR="${STRATAFS_CONFIG_DIR:-$HOME/.stratafs}"
UPDATE_CHECK_FILE="$CONFIG_DIR/last_update_check"
BINARY_NAME="stratafs"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Platform detection
PLATFORM="$(uname -s)"
ARCH="$(uname -m)"

case "$PLATFORM" in
    Linux*)
        OS="linux"
        case "$ARCH" in
            x86_64) ARCH="amd64" ;;
            aarch64) ARCH="arm64" ;;
            armv7l) ARCH="arm" ;;
        esac
        ;;
    Darwin*)
        OS="darwin"
        case "$ARCH" in
            x86_64) ARCH="amd64" ;;
            arm64) ARCH="arm64" ;;
        esac
        ;;
    CYGWIN*|MINGW*|MSYS*)
        OS="windows"
        ARCH="amd64"
        BINARY_NAME="stratafs.exe"
        ;;
esac

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if we should perform update check
should_check_update() {
    local check_interval="${UPDATE_CHECK_INTERVAL:-86400}" # 24 hours default

    if [ ! -f "$UPDATE_CHECK_FILE" ]; then
        return 0
    fi

    local last_check=$(cat "$UPDATE_CHECK_FILE" 2>/dev/null || echo "0")
    local current_time=$(date +%s)
    local time_diff=$((current_time - last_check))

    [ $time_diff -gt $check_interval ]
}

# Update last check time
update_check_time() {
    mkdir -p "$CONFIG_DIR"
    date +%s > "$UPDATE_CHECK_FILE"
}

# Get latest version from GitHub
get_latest_version() {
    log_info "Checking for updates..."

    # Try to get latest release from GitHub API
    local api_url="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    local latest_version=""

    if command -v curl >/dev/null 2>&1; then
        latest_version=$(curl -s "$api_url" | grep '"tag_name"' | cut -d'"' -f4 | sed 's/^v//')
    elif command -v wget >/dev/null 2>&1; then
        latest_version=$(wget -qO- "$api_url" | grep '"tag_name"' | cut -d'"' -f4 | sed 's/^v//')
    else
        log_error "Neither curl nor wget available for checking updates"
        return 1
    fi

    if [ -z "$latest_version" ]; then
        log_error "Failed to fetch latest version"
        return 1
    fi

    echo "$latest_version"
}

# Compare versions
version_greater() {
    local ver1="$1"
    local ver2="$2"

    # Simple version comparison (works for semantic versioning)
    printf '%s\n%s\n' "$ver1" "$ver2" | sort -V | head -n1 | grep -q "^$ver2$"
}

# Download and install update
download_and_install() {
    local version="$1"
    local temp_dir=$(mktemp -d)
    local download_url=""
    local filename=""

    log_info "Downloading StrataFS v$version..."

    # Determine download URL based on platform
    case "$OS" in
        linux)
            if command -v apt-get >/dev/null 2>&1; then
                filename="stratafs_${version}_${ARCH}.deb"
            else
                filename="StrataFS-${version}-x86_64.AppImage"
            fi
            ;;
        darwin)
            filename="StrataFS-${version}-${ARCH}.pkg"
            ;;
        windows)
            filename="StrataFS-${version}-Setup.exe"
            ;;
    esac

    download_url="https://github.com/$GITHUB_REPO/releases/download/v$version/$filename"

    # Download the file
    cd "$temp_dir"
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$filename" "$download_url"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$filename" "$download_url"
    else
        log_error "No download tool available"
        rm -rf "$temp_dir"
        return 1
    fi

    if [ ! -f "$filename" ]; then
        log_error "Download failed"
        rm -rf "$temp_dir"
        return 1
    fi

    log_success "Downloaded $filename"

    # Install based on file type
    case "$filename" in
        *.deb)
            log_info "Installing DEB package..."
            sudo dpkg -i "$filename" || {
                log_warning "DEB install failed, trying to fix dependencies..."
                sudo apt-get install -f -y
            }
            ;;
        *.AppImage)
            log_info "Installing AppImage..."
            chmod +x "$filename"
            # Move to applications directory
            mkdir -p "$HOME/.local/bin"
            mv "$filename" "$HOME/.local/bin/stratafs"
            # Update desktop integration if available
            if [ -f "$HOME/.local/share/applications/stratafs.desktop" ]; then
                sed -i "s|Exec=.*|Exec=$HOME/.local/bin/stratafs|" "$HOME/.local/share/applications/stratafs.desktop"
            fi
            ;;
        *.pkg)
            log_info "Installing PKG package..."
            sudo installer -pkg "$filename" -target /
            ;;
        *.exe)
            log_info "Running Windows installer..."
            if command -v powershell >/dev/null 2>&1; then
                powershell -Command "Start-Process -FilePath '$filename' -ArgumentList '/S' -Wait"
            else
                "$filename" /S
            fi
            ;;
    esac

    # Cleanup
    rm -rf "$temp_dir"

    log_success "Update completed successfully!"
    return 0
}

# Show update notification
show_update_notification() {
    local version="$1"
    local message="StrataFS v$version is available. Current version: v$CURRENT_VERSION"

    case "$OS" in
        linux)
            if command -v notify-send >/dev/null 2>&1; then
                notify-send "StrataFS Update Available" "$message" -i "software-update-available"
            elif command -v zenity >/dev/null 2>&1; then
                zenity --info --text="$message\n\nWould you like to update now?" --title="StrataFS Update"
                return $?
            fi
            ;;
        darwin)
            osascript -e "display notification \"$message\" with title \"StrataFS Update Available\""
            osascript -e "display dialog \"$message\" buttons {\"Later\", \"Update Now\"} default button 2" >/dev/null 2>&1
            return $?
            ;;
        windows)
            powershell -Command "
                Add-Type -AssemblyName System.Windows.Forms
                [System.Windows.Forms.MessageBox]::Show('$message', 'StrataFS Update Available', 'YesNo', 'Information')
            " 2>/dev/null | grep -q "Yes"
            return $?
            ;;
    esac

    return 1
}

# Check and install updates
check_and_update() {
    local force_check="$1"

    # Check if we should perform update check
    if [ "$force_check" != "true" ] && ! should_check_update; then
        return 0
    fi

    # Update check time
    update_check_time

    # Get latest version
    local latest_version
    latest_version=$(get_latest_version)
    if [ $? -ne 0 ]; then
        return 1
    fi

    log_info "Current version: v$CURRENT_VERSION"
    log_info "Latest version: v$latest_version"

    # Compare versions
    if version_greater "$latest_version" "$CURRENT_VERSION"; then
        log_info "Update available: v$CURRENT_VERSION -> v$latest_version"

        # Show notification and ask for permission
        if show_update_notification "$latest_version"; then
            download_and_install "$latest_version"
        else
            log_info "Update postponed by user"
        fi
    else
        log_info "StrataFS is up to date"
    fi
}

# Disable auto-updates
disable_auto_updates() {
    local config_file="$CONFIG_DIR/config.json"

    if [ -f "$config_file" ]; then
        # Add auto_update: false to config
        if command -v jq >/dev/null 2>&1; then
            jq '.auto_update = false' "$config_file" > "$config_file.tmp" && mv "$config_file.tmp" "$config_file"
            log_success "Auto-updates disabled"
        else
            log_warning "jq not available. Please manually add '\"auto_update\": false' to $config_file"
        fi
    else
        mkdir -p "$CONFIG_DIR"
        echo '{"auto_update": false}' > "$config_file"
        log_success "Auto-updates disabled"
    fi
}

# Enable auto-updates
enable_auto_updates() {
    local config_file="$CONFIG_DIR/config.json"

    if [ -f "$config_file" ]; then
        if command -v jq >/dev/null 2>&1; then
            jq '.auto_update = true' "$config_file" > "$config_file.tmp" && mv "$config_file.tmp" "$config_file"
            log_success "Auto-updates enabled"
        else
            log_warning "jq not available. Please manually add '\"auto_update\": true' to $config_file"
        fi
    else
        mkdir -p "$CONFIG_DIR"
        echo '{"auto_update": true}' > "$config_file"
        log_success "Auto-updates enabled"
    fi
}

# Check if auto-updates are enabled
auto_updates_enabled() {
    local config_file="$CONFIG_DIR/config.json"

    if [ -f "$config_file" ]; then
        if command -v jq >/dev/null 2>&1; then
            local enabled=$(jq -r '.auto_update // true' "$config_file")
            [ "$enabled" = "true" ]
        else
            # Default to enabled if jq not available
            return 0
        fi
    else
        # Default to enabled if no config
        return 0
    fi
}

# Show help
show_help() {
    cat << EOF
StrataFS Auto-Updater

Usage: $0 [COMMAND]

Commands:
  check         Check for updates and install if available
  force-check   Force check for updates (ignore check interval)
  enable        Enable automatic updates
  disable       Disable automatic updates
  status        Show current update status
  help          Show this help message

Environment Variables:
  STRATAFS_VERSION           Current StrataFS version
  STRATAFS_CONFIG_DIR        Configuration directory
  UPDATE_CHECK_INTERVAL     Check interval in seconds (default: 86400)
  GITHUB_REPO              GitHub repository (default: dipankar/stratafs)

Examples:
  $0 check              # Check for updates
  $0 force-check        # Force update check
  $0 disable            # Disable auto-updates
EOF
}

# Main function
main() {
    case "${1:-check}" in
        check)
            if auto_updates_enabled; then
                check_and_update false
            else
                log_info "Auto-updates are disabled"
            fi
            ;;
        force-check)
            check_and_update true
            ;;
        enable)
            enable_auto_updates
            ;;
        disable)
            disable_auto_updates
            ;;
        status)
            if auto_updates_enabled; then
                log_info "Auto-updates: ENABLED"
            else
                log_info "Auto-updates: DISABLED"
            fi

            if [ -f "$UPDATE_CHECK_FILE" ]; then
                local last_check=$(cat "$UPDATE_CHECK_FILE")
                local last_check_date=$(date -d "@$last_check" 2>/dev/null || date -r "$last_check" 2>/dev/null || echo "Unknown")
                log_info "Last check: $last_check_date"
            else
                log_info "Last check: Never"
            fi
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"