#!/bin/bash
# StrataFS Cross-Platform Desktop Launcher

set -e

# Configuration
STRATAFS_NAME="StrataFS"
STRATAFS_DESCRIPTION="The Agentic Filesystem for AI agents"
STRATAFS_VERSION="${STRATAFS_VERSION:-0.2.0}"
STRATAFS_BINARY="${STRATAFS_BINARY:-stratafs}"
STRATAFS_CONFIG_DIR="${STRATAFS_CONFIG_DIR:-$HOME/.stratafs}"
STRATAFS_API_PORT="${STRATAFS_API_PORT:-8080}"
STRATAFS_MCP_PORT="${STRATAFS_MCP_PORT:-8081}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Platform detection
PLATFORM="$(uname -s)"
case "$PLATFORM" in
    Linux*)
        OS="linux"
        ;;
    Darwin*)
        OS="macos"
        ;;
    CYGWIN*|MINGW*|MSYS*)
        OS="windows"
        ;;
    *)
        OS="unknown"
        ;;
esac

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# GUI notification functions
show_notification() {
    local title="$1"
    local message="$2"
    local type="${3:-info}"

    case "$OS" in
        linux)
            if command -v notify-send >/dev/null 2>&1; then
                notify-send "$title" "$message"
            elif command -v zenity >/dev/null 2>&1; then
                case "$type" in
                    error) zenity --error --text="$message" --title="$title" ;;
                    warning) zenity --warning --text="$message" --title="$title" ;;
                    *) zenity --info --text="$message" --title="$title" ;;
                esac
            else
                echo "$title: $message"
            fi
            ;;
        macos)
            osascript -e "display notification \"$message\" with title \"$title\""
            ;;
        windows)
            # Use PowerShell for Windows notifications
            powershell -Command "
                [System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms') | Out-Null
                [System.Windows.Forms.MessageBox]::Show('$message', '$title')
            " 2>/dev/null || echo "$title: $message"
            ;;
        *)
            echo "$title: $message"
            ;;
    esac
}

show_dialog() {
    local message="$1"
    local type="${2:-info}"

    case "$OS" in
        linux)
            if command -v zenity >/dev/null 2>&1; then
                case "$type" in
                    error) zenity --error --text="$message" --title="$STRATAFS_NAME" ;;
                    warning) zenity --warning --text="$message" --title="$STRATAFS_NAME" ;;
                    question) zenity --question --text="$message" --title="$STRATAFS_NAME" ;;
                    *) zenity --info --text="$message" --title="$STRATAFS_NAME" ;;
                esac
            else
                echo "$message"
                [ "$type" = "question" ] && read -p "Continue? [y/N]: " -n 1 -r && echo
            fi
            ;;
        macos)
            case "$type" in
                error)
                    osascript -e "display dialog \"$message\" with title \"$STRATAFS_NAME\" buttons {\"OK\"} default button 1 with icon stop"
                    ;;
                warning)
                    osascript -e "display dialog \"$message\" with title \"$STRATAFS_NAME\" buttons {\"OK\"} default button 1 with icon caution"
                    ;;
                question)
                    osascript -e "display dialog \"$message\" with title \"$STRATAFS_NAME\" buttons {\"Cancel\", \"OK\"} default button 2"
                    ;;
                *)
                    osascript -e "display dialog \"$message\" with title \"$STRATAFS_NAME\" buttons {\"OK\"} default button 1"
                    ;;
            esac
            ;;
        windows)
            powershell -Command "
                Add-Type -AssemblyName System.Windows.Forms
                [System.Windows.Forms.MessageBox]::Show('$message', '$STRATAFS_NAME')
            " 2>/dev/null || echo "$message"
            ;;
        *)
            echo "$message"
            ;;
    esac
}

# Check if StrataFS binary exists
check_binary() {
    if ! command -v "$STRATAFS_BINARY" >/dev/null 2>&1; then
        show_dialog "StrataFS binary not found in PATH.\n\nPlease ensure StrataFS is properly installed." "error"
        exit 1
    fi
}

# Initialize StrataFS configuration
initialize_config() {
    if [ ! -d "$STRATAFS_CONFIG_DIR" ]; then
        log_info "Initializing StrataFS configuration..."
        show_notification "$STRATAFS_NAME" "Initializing configuration for first use..."

        mkdir -p "$STRATAFS_CONFIG_DIR"

        if ! "$STRATAFS_BINARY" config init --config-dir="$STRATAFS_CONFIG_DIR"; then
            show_dialog "Failed to initialize StrataFS configuration.\n\nPlease check the installation and try again." "error"
            exit 1
        fi

        show_notification "$STRATAFS_NAME" "Configuration initialized successfully!"
    fi
}

# Check if StrataFS is running
is_running() {
    case "$OS" in
        windows)
            tasklist /FI "IMAGENAME eq stratafs.exe" 2>/dev/null | grep -q "stratafs.exe"
            ;;
        *)
            pgrep -f "$STRATAFS_BINARY" >/dev/null 2>&1
            ;;
    esac
}

# Start StrataFS
start_stratafs() {
    if is_running; then
        show_dialog "$STRATAFS_NAME is already running!\n\nWeb interface: http://localhost:$STRATAFS_API_PORT\nMCP server: http://localhost:$STRATAFS_MCP_PORT" "info"
        open_web_interface
        return 0
    fi

    log_info "Starting $STRATAFS_NAME..."
    show_notification "$STRATAFS_NAME" "Starting StrataFS service..."

    # Start StrataFS in background
    case "$OS" in
        windows)
            start "" "$STRATAFS_BINARY" --config-dir="$STRATAFS_CONFIG_DIR" > "$STRATAFS_CONFIG_DIR/desktop.log" 2>&1
            ;;
        *)
            nohup "$STRATAFS_BINARY" --config-dir="$STRATAFS_CONFIG_DIR" > "$STRATAFS_CONFIG_DIR/desktop.log" 2>&1 &
            ;;
    esac

    # Wait for startup
    local attempts=0
    local max_attempts=10

    while [ $attempts -lt $max_attempts ]; do
        sleep 1
        if is_running; then
            log_success "$STRATAFS_NAME started successfully!"
            show_notification "$STRATAFS_NAME" "Service started successfully!\nWeb interface: http://localhost:$STRATAFS_API_PORT"
            open_web_interface
            return 0
        fi
        attempts=$((attempts + 1))
    done

    show_dialog "Failed to start $STRATAFS_NAME.\n\nPlease check the log file at:\n$STRATAFS_CONFIG_DIR/desktop.log" "error"
    exit 1
}

# Stop StrataFS
stop_stratafs() {
    if ! is_running; then
        show_notification "$STRATAFS_NAME" "StrataFS is not running"
        return 0
    fi

    log_info "Stopping $STRATAFS_NAME..."

    case "$OS" in
        windows)
            taskkill /F /IM stratafs.exe 2>/dev/null || true
            ;;
        *)
            pkill -f "$STRATAFS_BINARY" || true
            ;;
    esac

    # Wait for shutdown
    local attempts=0
    local max_attempts=5

    while [ $attempts -lt $max_attempts ]; do
        sleep 1
        if ! is_running; then
            log_success "$STRATAFS_NAME stopped successfully!"
            show_notification "$STRATAFS_NAME" "Service stopped"
            return 0
        fi
        attempts=$((attempts + 1))
    done

    log_warning "$STRATAFS_NAME may still be running"
}

# Restart StrataFS
restart_stratafs() {
    log_info "Restarting $STRATAFS_NAME..."
    stop_stratafs
    sleep 2
    start_stratafs
}

# Open web interface in browser
open_web_interface() {
    local url="http://localhost:$STRATAFS_API_PORT"

    case "$OS" in
        linux)
            if command -v xdg-open >/dev/null 2>&1; then
                xdg-open "$url" &
            fi
            ;;
        macos)
            open "$url" &
            ;;
        windows)
            start "$url"
            ;;
    esac
}

# Show status
show_status() {
    if is_running; then
        local pid=""
        case "$OS" in
            windows)
                pid=$(tasklist /FI "IMAGENAME eq stratafs.exe" /FO CSV | grep stratafs.exe | cut -d',' -f2 | tr -d '"' | head -1)
                ;;
            *)
                pid=$(pgrep -f "$STRATAFS_BINARY" | head -1)
                ;;
        esac

        show_dialog "$STRATAFS_NAME Status: RUNNING\nPID: $pid\n\nWeb interface: http://localhost:$STRATAFS_API_PORT\nMCP server: http://localhost:$STRATAFS_MCP_PORT\n\nLog file: $STRATAFS_CONFIG_DIR/desktop.log" "info"
    else
        show_dialog "$STRATAFS_NAME Status: STOPPED\n\nWeb interface: http://localhost:$STRATAFS_API_PORT (not available)\nMCP server: http://localhost:$STRATAFS_MCP_PORT (not available)" "warning"
    fi
}

# Show configuration
show_config() {
    if [ -f "$STRATAFS_CONFIG_DIR/config.json" ]; then
        local config_content
        case "$OS" in
            linux|macos)
                if command -v zenity >/dev/null 2>&1; then
                    zenity --text-info --title="$STRATAFS_NAME Configuration" --filename="$STRATAFS_CONFIG_DIR/config.json" --width=800 --height=600
                else
                    cat "$STRATAFS_CONFIG_DIR/config.json"
                fi
                ;;
            windows)
                notepad "$STRATAFS_CONFIG_DIR/config.json"
                ;;
            *)
                cat "$STRATAFS_CONFIG_DIR/config.json"
                ;;
        esac
    else
        show_dialog "Configuration file not found.\n\nPlease initialize StrataFS first." "error"
    fi
}

# Show help
show_help() {
    cat << EOF
$STRATAFS_NAME Desktop Launcher

Usage: $0 [COMMAND]

Commands:
  start         Start StrataFS service
  stop          Stop StrataFS service
  restart       Restart StrataFS service
  status        Show service status
  config        Show configuration
  init          Initialize configuration
  web           Open web interface
  help          Show this help message

No command runs the GUI launcher (start if not running, show status if running).

Environment Variables:
  STRATAFS_BINARY      Path to StrataFS binary (default: stratafs)
  STRATAFS_CONFIG_DIR  Configuration directory (default: ~/.stratafs)
  STRATAFS_API_PORT    API server port (default: 8080)
  STRATAFS_MCP_PORT    MCP server port (default: 8081)

Examples:
  $0                  # GUI launcher
  $0 start            # Start service
  $0 status           # Show status
  $0 web              # Open web interface
EOF
}

# Main function
main() {
    case "${1:-gui}" in
        start)
            check_binary
            initialize_config
            start_stratafs
            ;;
        stop)
            stop_stratafs
            ;;
        restart)
            check_binary
            restart_stratafs
            ;;
        status)
            show_status
            ;;
        config)
            show_config
            ;;
        init)
            check_binary
            initialize_config
            show_notification "$STRATAFS_NAME" "Configuration initialized!"
            ;;
        web)
            if is_running; then
                open_web_interface
            else
                show_dialog "$STRATAFS_NAME is not running.\n\nWould you like to start it?" "question"
                if [ $? -eq 0 ]; then
                    check_binary
                    initialize_config
                    start_stratafs
                fi
            fi
            ;;
        help|--help|-h)
            show_help
            ;;
        gui|*)
            # Default GUI mode
            check_binary
            initialize_config

            if is_running; then
                show_status
            else
                if show_dialog "$STRATAFS_NAME is not running.\n\nWould you like to start it?" "question"; then
                    start_stratafs
                fi
            fi
            ;;
    esac
}

# Execute main function
main "$@"