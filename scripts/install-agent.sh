#!/bin/bash
#
# Nexus RMM Agent Installer for Linux/macOS
#
# Usage: curl -sSL https://rmm.mezfashion.pro/install-agent.sh | sudo bash -s -- -u https://rmm.mezfashion.pro -t ENROLLMENT_TOKEN
# Or:    sudo ./install-agent.sh -u https://rmm.mezfashion.pro -t ENROLLMENT_TOKEN
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default values
SERVER_URL=""
ENROLLMENT_TOKEN=""
INSTALL_PATH="/opt/nexus-rmm"
SERVICE_NAME="nexus-rmm-agent"
START_SERVICE=true
ARCH=""

# Print functions
print_header() {
    echo -e "${CYAN}=========================================="
    echo -e "  Nexus RMM Agent Installer for Linux/macOS"
    echo -e "==========================================${NC}"
    echo
}

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

# Check root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case $os in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac

    case $arch in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armhf)
            ARCH="arm"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    print_info "Detected platform: $OS/$ARCH"
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -u|--url)
                SERVER_URL="$2"
                shift 2
                ;;
            -t|--token)
                ENROLLMENT_TOKEN="$2"
                shift 2
                ;;
            -p|--path)
                INSTALL_PATH="$2"
                shift 2
                ;;
            -s|--service)
                SERVICE_NAME="$2"
                shift 2
                ;;
            --no-start)
                START_SERVICE=false
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Validate required arguments
    if [[ -z "$SERVER_URL" ]]; then
        print_error "Server URL is required (-u/--url)"
        show_help
        exit 1
    fi

    if [[ -z "$ENROLLMENT_TOKEN" ]]; then
        print_error "Enrollment token is required (-t/--token)"
        show_help
        exit 1
    fi

    if [[ ! "$SERVER_URL" =~ ^https?:// ]]; then
        print_error "Server URL must start with http:// or https://"
        exit 1
    fi
}

show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Options:
    -u, --url URL           Nexus RMM server URL (e.g., https://rmm.mezfashion.pro)
    -t, --token TOKEN       Enrollment token from web console
    -p, --path PATH         Installation directory (default: /opt/nexus-rmm)
    -s, --service NAME      Service name (default: nexus-rmm-agent)
    --no-start              Don't start service after installation
    -h, --help              Show this help message

Examples:
    $0 -u https://rmm.mezfashion.pro -t abc123...
    $0 -u https://rmm.mezfashion.pro -t abc123... -p /usr/local/nexus-rmm
EOF
}

# Download agent binary
download_agent() {
    local agent_url="${SERVER_URL}/api/v1/agent/download/${OS}/${ARCH}"
    local agent_path="${INSTALL_PATH}/nexus-agent"

    print_info "Downloading agent from: $agent_url"

    if command -v curl &> /dev/null; then
        curl -fsSL -H "User-Agent: NexusRMMAgentInstaller/1.0" -o "$agent_path" "$agent_url"
    elif command -v wget &> /dev/null; then
        wget -q -U "NexusRMMAgentInstaller/1.0" -O "$agent_path" "$agent_url"
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    chmod +x "$agent_path"
    print_success "Agent downloaded to $agent_path"
}

# Create configuration
create_config() {
    local config_path="${INSTALL_PATH}/agent.yaml"

    print_info "Creating configuration: $config_path"

    mkdir -p "$(dirname "$config_path")"

    cat > "$config_path" << EOC
server_url: "$SERVER_URL"
enrollment_token: "$ENROLLMENT_TOKEN"
install_path: "$INSTALL_PATH"
log_level: "info"
log_file: "/var/log/nexus-rmm/agent.log"
heartbeat_interval: 30s
command_timeout: 5m
max_concurrent_tasks: 5
auto_update: true
update_check_interval: 1h
EOC

    print_success "Configuration created"
}

# Create log directory
create_log_dir() {
    mkdir -p /var/log/nexus-rmm
    touch /var/log/nexus-rmm/agent.log
    chmod 644 /var/log/nexus-rmm/agent.log
}

# Install systemd service (Linux)
install_systemd_service() {
    if [[ "$OS" != "linux" ]]; then
        return 0
    fi

    print_info "Installing systemd service: $SERVICE_NAME"

    cat > "/etc/systemd/system/${SERVICE_NAME}.service" << EOS
[Unit]
Description=Nexus RMM Endpoint Agent
Documentation=https://github.com/nexus-rmm/nexus-rmm
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=${INSTALL_PATH}
ExecStart=${INSTALL_PATH}/nexus-agent --config ${INSTALL_PATH}/agent.yaml run-service
Restart=always
RestartSec=10
StartLimitInterval=60
StartLimitBurst=3
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${SERVICE_NAME}

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${INSTALL_PATH} /var/log/nexus-rmm
CapabilityBoundingSet=CAP_DAC_OVERRIDE CAP_DAC_READ_SEARCH

[Install]
WantedBy=multi-user.target
EOS

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"

    print_success "Systemd service installed and enabled"
}

# Install launchd service (macOS)
install_launchd_service() {
    if [[ "$OS" != "darwin" ]]; then
        return 0
    fi

    print_info "Installing launchd service: $SERVICE_NAME"

    cat > "/Library/LaunchDaemons/${SERVICE_NAME}.plist" << EOP
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${SERVICE_NAME}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_PATH}/nexus-agent</string>
        <string>--config</string>
        <string>${INSTALL_PATH}/agent.yaml</string>
        <string>run-service</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/nexus-rmm/agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/nexus-rmm/agent.log</string>
    <key>WorkingDirectory</key>
    <string>${INSTALL_PATH}</string>
    <key>ProcessType</key>
    <string>Background</string>
</dict>
</plist>
EOP

    launchctl load "/Library/LaunchDaemons/${SERVICE_NAME}.plist"
    launchctl enable "system/${SERVICE_NAME}"

    print_success "Launchd service installed and enabled"
}

# Start service
start_service() {
    if [[ "$START_SERVICE" != "true" ]]; then
        print_warning "Skipping service start (--no-start specified)"
        return 0
    fi

    print_info "Starting service: $SERVICE_NAME"

    if [[ "$OS" == "linux" ]]; then
        systemctl start "$SERVICE_NAME"
        sleep 2
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            print_success "Service started successfully"
        else
            print_warning "Service may have failed to start. Check logs: journalctl -u $SERVICE_NAME -f"
        fi
    elif [[ "$OS" == "darwin" ]]; then
        launchctl kickstart -k "system/${SERVICE_NAME}"
        sleep 2
        if launchctl list | grep -q "$SERVICE_NAME"; then
            print_success "Service started successfully"
        else
            print_warning "Service may have failed to start. Check logs: tail -f /var/log/nexus-rmm/agent.log"
        fi
    fi
}

# Verify installation
verify_installation() {
    print_info "Verifying installation..."

    # Check binary
    if [[ ! -f "${INSTALL_PATH}/nexus-agent" ]]; then
        print_error "Agent binary not found"
        return 1
    fi

    # Check config
    if [[ ! -f "${INSTALL_PATH}/agent.yaml" ]]; then
        print_error "Configuration file not found"
        return 1
    fi

    # Check service
    if [[ "$OS" == "linux" ]]; then
        if systemctl is-enabled --quiet "$SERVICE_NAME"; then
            print_success "Service is enabled"
        else
            print_warning "Service is not enabled"
        fi
    elif [[ "$OS" == "darwin" ]]; then
        if launchctl list | grep -q "$SERVICE_NAME"; then
            print_success "Service is loaded"
        else
            print_warning "Service is not loaded"
        fi
    fi

    print_success "Installation verified"
}

# Main
main() {
    print_header
    check_root
    detect_platform
    parse_args "$@"

    print_info "Installing Nexus RMM Agent..."
    print_info "Server: $SERVER_URL"
    print_info "Install path: $INSTALL_PATH"
    print_info "Service name: $SERVICE_NAME"
    echo

    # Create install directory
    mkdir -p "$INSTALL_PATH"

    # Download agent
    download_agent

    # Create configuration
    create_config

    # Create log directory
    create_log_dir

    # Install service
    if [[ "$OS" == "linux" ]]; then
        install_systemd_service
    elif [[ "$OS" == "darwin" ]]; then
        install_launchd_service
    fi

    # Start service
    start_service

    # Verify
    verify_installation

    echo
    echo -e "${CYAN}=========================================="
    echo -e "  Installation Complete!"
    echo -e "==========================================${NC}"
    echo
    echo -e "Service Name: ${GREEN}$SERVICE_NAME${NC}"
    echo -e "Install Path: ${GREEN}$INSTALL_PATH${NC}"
    echo -e "Config File:  ${GREEN}${INSTALL_PATH}/agent.yaml${NC}"
    echo -e "Log File:     ${GREEN}/var/log/nexus-rmm/agent.log${NC}"
    echo
    echo -e "${YELLOW}Useful commands:${NC}"
    if [[ "$OS" == "linux" ]]; then
        echo -e "  Start service:   ${GRAY}systemctl start $SERVICE_NAME${NC}"
        echo -e "  Stop service:    ${GRAY}systemctl stop $SERVICE_NAME${NC}"
        echo -e "  Restart service: ${GRAY}systemctl restart $SERVICE_NAME${NC}"
        echo -e "  View logs:       ${GRAY}journalctl -u $SERVICE_NAME -f${NC}"
        echo -e "  Status:          ${GRAY}systemctl status $SERVICE_NAME${NC}"
        echo -e "  Uninstall:       ${GRAY}systemctl stop $SERVICE_NAME && systemctl disable $SERVICE_NAME && rm /etc/systemd/system/${SERVICE_NAME}.service${NC}"
    elif [[ "$OS" == "darwin" ]]; then
        echo -e "  Start service:   ${GRAY}sudo launchctl kickstart -k system/$SERVICE_NAME${NC}"
        echo -e "  Stop service:    ${GRAY}sudo launchctl unload /Library/LaunchDaemons/${SERVICE_NAME}.plist${NC}"
        echo -e "  View logs:       ${GRAY}tail -f /var/log/nexus-rmm/agent.log${NC}"
        echo -e "  Uninstall:       ${GRAY}sudo launchctl unload /Library/LaunchDaemons/${SERVICE_NAME}.plist && sudo rm /Library/LaunchDaemons/${SERVICE_NAME}.plist${NC}"
    fi
    echo
    echo -e "${CYAN}The agent will now enroll with the server and appear in the Nexus RMM console.${NC}"
}

# Run main
main "$@"