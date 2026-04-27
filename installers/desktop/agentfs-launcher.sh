#!/bin/bash
# AgentFS Cross-Platform Desktop Launcher

set -e

# Configuration
AGENTFS_NAME="AgentFS"
AGENTFS_DESCRIPTION="The Agentic Filesystem for AI agents"
AGENTFS_VERSION="${AGENTFS_VERSION:-0.2.0}"
AGENTFS_BINARY="${AGENTFS_BINARY:-agentfs}"
AGENTFS_CONFIG_DIR="${AGENTFS_CONFIG_DIR:-$HOME/.agentfs}"
AGENTFS_API_PORT="${AGENTFS_API_PORT:-8080}"
AGENTFS_MCP_PORT="${AGENTFS_MCP_PORT:-8081}"

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
                    error) zenity --error --text="$message" --title="$AGENTFS_NAME" ;;
                    warning) zenity --warning --text="$message" --title="$AGENTFS_NAME" ;;
                    question) zenity --question --text="$message" --title="$AGENTFS_NAME" ;;
                    *) zenity --info --text="$message" --title="$AGENTFS_NAME" ;;
                esac
            else
                echo "$message"
                [ "$type" = "question" ] && read -p "Continue? [y/N]: " -n 1 -r && echo
            fi
            ;;
        macos)
            case "$type" in
                error)
                    osascript -e "display dialog \"$message\" with title \"$AGENTFS_NAME\" buttons {\"OK\"} default button 1 with icon stop"
                    ;;
                warning)
                    osascript -e "display dialog \"$message\" with title \"$AGENTFS_NAME\" buttons {\"OK\"} default button 1 with icon caution"
                    ;;
                question)
                    osascript -e "display dialog \"$message\" with title \"$AGENTFS_NAME\" buttons {\"Cancel\", \"OK\"} default button 2"
                    ;;
                *)
                    osascript -e "display dialog \"$message\" with title \"$AGENTFS_NAME\" buttons {\"OK\"} default button 1"
                    ;;
            esac
            ;;
        windows)
            powershell -Command "
                Add-Type -AssemblyName System.Windows.Forms
                [System.Windows.Forms.MessageBox]::Show('$message', '$AGENTFS_NAME')
            " 2>/dev/null || echo "$message"
            ;;
        *)
            echo "$message"
            ;;
    esac
}

# Check if AgentFS binary exists
check_binary() {
    if ! command -v "$AGENTFS_BINARY" >/dev/null 2>&1; then
        show_dialog "AgentFS binary not found in PATH.\n\nPlease ensure AgentFS is properly installed." "error"
        exit 1
    fi
}

# Initialize AgentFS configuration
initialize_config() {
    if [ ! -d "$AGENTFS_CONFIG_DIR" ]; then
        log_info "Initializing AgentFS configuration..."
        show_notification "$AGENTFS_NAME" "Initializing configuration for first use..."

        mkdir -p "$AGENTFS_CONFIG_DIR"

        if ! "$AGENTFS_BINARY" config init --config-dir="$AGENTFS_CONFIG_DIR"; then
            show_dialog "Failed to initialize AgentFS configuration.\n\nPlease check the installation and try again." "error"
            exit 1
        fi

        show_notification "$AGENTFS_NAME" "Configuration initialized successfully!"
    fi
}

# Check if AgentFS is running
is_running() {
    case "$OS" in
        windows)
            tasklist /FI "IMAGENAME eq agentfs.exe" 2>/dev/null | grep -q "agentfs.exe"
            ;;
        *)
            pgrep -f "$AGENTFS_BINARY" >/dev/null 2>&1
            ;;
    esac
}

# Start AgentFS
start_agentfs() {
    if is_running; then
        show_dialog "$AGENTFS_NAME is already running!\n\nWeb interface: http://localhost:$AGENTFS_API_PORT\nMCP server: http://localhost:$AGENTFS_MCP_PORT" "info"
        open_web_interface
        return 0
    fi

    log_info "Starting $AGENTFS_NAME..."
    show_notification "$AGENTFS_NAME" "Starting AgentFS service..."

    # Start AgentFS in background
    case "$OS" in
        windows)
            start "" "$AGENTFS_BINARY" --config-dir="$AGENTFS_CONFIG_DIR" > "$AGENTFS_CONFIG_DIR/desktop.log" 2>&1
            ;;
        *)
            nohup "$AGENTFS_BINARY" --config-dir="$AGENTFS_CONFIG_DIR" > "$AGENTFS_CONFIG_DIR/desktop.log" 2>&1 &
            ;;
    esac

    # Wait for startup
    local attempts=0
    local max_attempts=10

    while [ $attempts -lt $max_attempts ]; do
        sleep 1
        if is_running; then
            log_success "$AGENTFS_NAME started successfully!"
            show_notification "$AGENTFS_NAME" "Service started successfully!\nWeb interface: http://localhost:$AGENTFS_API_PORT"
            open_web_interface
            return 0
        fi
        attempts=$((attempts + 1))
    done

    show_dialog "Failed to start $AGENTFS_NAME.\n\nPlease check the log file at:\n$AGENTFS_CONFIG_DIR/desktop.log" "error"
    exit 1
}

# Stop AgentFS
stop_agentfs() {
    if ! is_running; then
        show_notification "$AGENTFS_NAME" "AgentFS is not running"
        return 0
    fi

    log_info "Stopping $AGENTFS_NAME..."

    case "$OS" in
        windows)
            taskkill /F /IM agentfs.exe 2>/dev/null || true
            ;;
        *)
            pkill -f "$AGENTFS_BINARY" || true
            ;;
    esac

    # Wait for shutdown
    local attempts=0
    local max_attempts=5

    while [ $attempts -lt $max_attempts ]; do
        sleep 1
        if ! is_running; then
            log_success "$AGENTFS_NAME stopped successfully!"
            show_notification "$AGENTFS_NAME" "Service stopped"
            return 0
        fi
        attempts=$((attempts + 1))
    done

    log_warning "$AGENTFS_NAME may still be running"
}

# Restart AgentFS
restart_agentfs() {
    log_info "Restarting $AGENTFS_NAME..."
    stop_agentfs
    sleep 2
    start_agentfs
}

# Open web interface in browser
open_web_interface() {
    local url="http://localhost:$AGENTFS_API_PORT"

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
                pid=$(tasklist /FI "IMAGENAME eq agentfs.exe" /FO CSV | grep agentfs.exe | cut -d',' -f2 | tr -d '"' | head -1)
                ;;
            *)
                pid=$(pgrep -f "$AGENTFS_BINARY" | head -1)
                ;;
        esac

        show_dialog "$AGENTFS_NAME Status: RUNNING\nPID: $pid\n\nWeb interface: http://localhost:$AGENTFS_API_PORT\nMCP server: http://localhost:$AGENTFS_MCP_PORT\n\nLog file: $AGENTFS_CONFIG_DIR/desktop.log" "info"
    else
        show_dialog "$AGENTFS_NAME Status: STOPPED\n\nWeb interface: http://localhost:$AGENTFS_API_PORT (not available)\nMCP server: http://localhost:$AGENTFS_MCP_PORT (not available)" "warning"
    fi
}

# Show configuration
show_config() {
    if [ -f "$AGENTFS_CONFIG_DIR/config.json" ]; then
        local config_content
        case "$OS" in
            linux|macos)
                if command -v zenity >/dev/null 2>&1; then
                    zenity --text-info --title="$AGENTFS_NAME Configuration" --filename="$AGENTFS_CONFIG_DIR/config.json" --width=800 --height=600
                else
                    cat "$AGENTFS_CONFIG_DIR/config.json"
                fi
                ;;
            windows)
                notepad "$AGENTFS_CONFIG_DIR/config.json"
                ;;
            *)
                cat "$AGENTFS_CONFIG_DIR/config.json"
                ;;
        esac
    else
        show_dialog "Configuration file not found.\n\nPlease initialize AgentFS first." "error"
    fi
}

# Show help
show_help() {
    cat << EOF
$AGENTFS_NAME Desktop Launcher

Usage: $0 [COMMAND]

Commands:
  start         Start AgentFS service
  stop          Stop AgentFS service
  restart       Restart AgentFS service
  status        Show service status
  config        Show configuration
  init          Initialize configuration
  web           Open web interface
  help          Show this help message

No command runs the GUI launcher (start if not running, show status if running).

Environment Variables:
  AGENTFS_BINARY      Path to AgentFS binary (default: agentfs)
  AGENTFS_CONFIG_DIR  Configuration directory (default: ~/.agentfs)
  AGENTFS_API_PORT    API server port (default: 8080)
  AGENTFS_MCP_PORT    MCP server port (default: 8081)

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
            start_agentfs
            ;;
        stop)
            stop_agentfs
            ;;
        restart)
            check_binary
            restart_agentfs
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
            show_notification "$AGENTFS_NAME" "Configuration initialized!"
            ;;
        web)
            if is_running; then
                open_web_interface
            else
                show_dialog "$AGENTFS_NAME is not running.\n\nWould you like to start it?" "question"
                if [ $? -eq 0 ]; then
                    check_binary
                    initialize_config
                    start_agentfs
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
                if show_dialog "$AGENTFS_NAME is not running.\n\nWould you like to start it?" "question"; then
                    start_agentfs
                fi
            fi
            ;;
    esac
}

# Execute main function
main "$@"