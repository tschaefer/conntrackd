#!/bin/sh
set -eu

# Configuration
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/conntrackd"
DATA_DIR="/var/lib/conntrackd"
CONFIG_FILE="${CONFIG_DIR}/conntrackd.yaml"
SYSTEMD_SERVICE="/etc/systemd/system/conntrackd.service"
GITHUB_REPO="tschaefer/conntrackd"
GEOIP_URL="https://git.io/GeoLite2-City.mmdb"
GEOIP_FILE="${DATA_DIR}/GeoLite2-City.mmdb"

# Colors for output (disabled if NO_COLOR is set)
if [ "${NO_COLOR:-}" = "1" ] || [ "${NO_COLOR:-}" = "true" ]; then
    RED=''
    GREEN=''
    YELLOW=''
    NC=''
else
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    NC='\033[0m' # No Color
fi

# Print usage
usage() {
    cat << EOF
Usage: $0 [--with-geoip | --help]

Options:
  --with-geoip  Install GeoIP database without prompting (for CI/automation)
  --help, -h    Show this help message

Environment Variables:
  NO_COLOR=1    Disable colored output
  WITH_GEOIP=1  Install GeoIP database without prompting (for CI/automation)

This script will:
  1. Download and install the latest binary for your architecture
     /usr/local/bin/conntrackd
  2. Create configuration directory and basic config file
     /etc/conntrackd/conntrackd.yml
  3. Optionally download and configure GeoIP database
     /var/lib/conntrackd/GeoLite2-City.mmdb
  4. Create and enable systemd service, if applicable
     /etc/systemd/system/conntrackd.service
EOF
    exit 0
}

# Print message
info() {
    printf "%b[INFO]%b %s\n" "${GREEN}" "${NC}" "$1"
}

warn() {
    printf "%b[WARN]%b %s\n" "${YELLOW}" "${NC}" "$1"
}

error() {
    printf "%b[ERROR]%b %s\n" "${RED}" "${NC}" "$1" >&2
}

# Check if running as root
check_root() {
    if [ "$(id -u)" -ne 0 ]; then
        error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect system architecture
detect_arch() {
    arch=$(uname -m)
    case "$arch" in
        x86_64)
            echo "amd64"
            ;;
        aarch64)
            echo "arm64"
            ;;
        *)
            error "Unsupported architecture: $arch"
            error "Supported architectures: x86_64 (amd64), aarch64 (arm64)"
            exit 1
            ;;
    esac
}

# Check required commands
check_dependencies() {
    if ! command -v curl >/dev/null 2>&1; then
        error "Required command 'curl' not found. Please install it first."
        exit 1
    fi
}

# Get latest release download URL
get_latest_release_url() {
    arch="$1"
    binary_name="conntrackd-linux-${arch}"
    download_url="https://github.com/${GITHUB_REPO}/releases/latest/download/${binary_name}"

    echo "$download_url"
}

# Download and install binary
install_binary() {
    arch="$1"
    download_url=$(get_latest_release_url "$arch")

    info "Downloading conntrackd binary for ${arch}..."
    if ! curl -sfL -o /tmp/conntrackd "$download_url"; then
        error "Failed to download conntrackd binary"
        exit 1
    fi

    # Verify downloaded file is not empty
    if [ ! -s /tmp/conntrackd ]; then
        error "Downloaded binary file is empty or does not exist"
        rm -f /tmp/conntrackd
        exit 1
    fi

    info "Installing binary to ${INSTALL_DIR}/conntrackd..."
    install -m 0755 /tmp/conntrackd "${INSTALL_DIR}/conntrackd"
    rm -f /tmp/conntrackd

    info "Binary installed successfully"
}

# Create configuration directory and basic config
create_config() {
    info "Creating configuration directory ${CONFIG_DIR}..."
    mkdir -p "$CONFIG_DIR"

    if [ -f "$CONFIG_FILE" ]; then
        warn "Configuration file already exists: ${CONFIG_FILE}"
        warn "Skipping configuration file creation"
        return 0
    fi

    info "Creating basic configuration file ${CONFIG_FILE}..."
    cat > "$CONFIG_FILE" << 'EOF'
log:
  level: "info"

filter:
  - "log protocol tcp and destination network public"
  - "drop any"

sink:
  stream:
    enable: true
EOF

    info "Configuration file created successfully"
}

# Install GeoIP database
install_geoip() {
    info "Creating data directory ${DATA_DIR}..."
    mkdir -p "$DATA_DIR"

    info "Downloading GeoLite2-City.mmdb..."
    if ! curl -sfL -o "$GEOIP_FILE" "$GEOIP_URL"; then
        error "Failed to download GeoIP database"
        return 1
    fi

    # Verify downloaded file is not empty and has reasonable size
    if [ ! -s "$GEOIP_FILE" ]; then
        error "Downloaded GeoIP database file is empty"
        rm -f "$GEOIP_FILE"
        return 1
    fi

    info "GeoIP database installed to ${GEOIP_FILE}"

    # Add GeoIP configuration if not already present
    if ! grep -q "^geoip:" "$CONFIG_FILE" 2>/dev/null; then
        info "Adding GeoIP configuration to ${CONFIG_FILE}..."
        cat >> "$CONFIG_FILE" << EOF

geoip:
  database: "${GEOIP_FILE}"
EOF
        info "GeoIP configuration added"
    else
        warn "GeoIP configuration already exists in config file"
    fi
}

# Check if systemd is available
has_systemd() {
    [ -d /run/systemd/system ]
}

# Create and enable systemd service
create_systemd_service() {
    if ! has_systemd; then
        warn "systemd not available, skipping service setup"
        warn "You will need to start conntrackd manually or configure your init system"
        return 0
    fi

    info "Creating systemd service ${SYSTEMD_SERVICE}..."
    cat > "$SYSTEMD_SERVICE" << 'EOF'
[Unit]
Description=Conntrack event logger with GEO location
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/conntrackd run
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

    info "Reloading systemd daemon..."
    systemctl daemon-reload

    info "Enabling conntrackd service..."
    systemctl enable conntrackd.service

    info "Starting conntrackd service..."
    systemctl start conntrackd.service

    info "Service created and started successfully"
}

# Parse command line arguments
WITH_GEOIP=${WITH_GEOIP:-false}
for arg in "$@"; do
    case "$arg" in
        --with-geoip)
            WITH_GEOIP=true
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            error "Unknown option: $arg"
            echo "Usage: $0 [--with-geoip | --help]"
            exit 1
            ;;
    esac
done

# Main installation function
main() {
    echo "======================================"
    echo "  conntrackd Installation Script"
    echo "======================================"
    echo

    check_root
    check_dependencies

    arch=$(detect_arch)
    info "Detected architecture: ${arch}"

    install_binary "$arch"
    create_config

    if [ "$WITH_GEOIP" = "true" ] || [ "$WITH_GEOIP" = "1" ]; then
        install_geoip
    fi

    create_systemd_service

    echo
    echo "======================================"
    echo "  Installation Complete!"
    echo "======================================"
    echo
    info "conntrackd has been installed"
    info "Configuration file: ${CONFIG_FILE}"
    if has_systemd; then
        info "Service status: systemctl status conntrackd"
        info "View logs: journalctl -u conntrackd -f"
    else
        info "Start manually: sudo conntrackd run"
    fi
}

main "$@"
