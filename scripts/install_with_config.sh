#!/bin/bash
# Capture Installation Script with Configuration Setup
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
INSTALL_DIR="/opt/capture"
CONFIG_DIR="/etc/capture"
USER="capture"
GROUP="capture"
SERVICE_NAME="capture"

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   log_error "This script must be run as root"
   exit 1
fi

# Create capture user if it doesn't exist
if ! id "$USER" &>/dev/null; then
    log_info "Creating user $USER..."
    useradd --system --shell /bin/false --home-dir "$INSTALL_DIR" --create-home "$USER"
fi

# Create directories
log_info "Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"
mkdir -p "/var/log/capture"

# Set ownership
chown "$USER:$GROUP" "$INSTALL_DIR"
chown "$USER:$GROUP" "$CONFIG_DIR"
chown "$USER:$GROUP" "/var/log/capture"

# Copy binary (assumes capture binary exists in current directory)
if [[ -f "./capture" ]]; then
    log_info "Installing capture binary..."
    cp "./capture" "$INSTALL_DIR/"
    chown "$USER:$GROUP" "$INSTALL_DIR/capture"
    chmod +x "$INSTALL_DIR/capture"
else
    log_error "capture binary not found in current directory"
    log_info "Please build the binary first: go build ./cmd/capture"
    exit 1
fi

# Generate default configuration if it doesn't exist
if [[ ! -f "$CONFIG_DIR/capture.yaml" ]]; then
    log_info "Generating default configuration..."
    "$INSTALL_DIR/capture" --generate-config "$CONFIG_DIR/capture.yaml"
    chown "$USER:$GROUP" "$CONFIG_DIR/capture.yaml"
    chmod 600 "$CONFIG_DIR/capture.yaml"  # Secure permissions for config with secrets
    
    log_warn "Configuration file created at $CONFIG_DIR/capture.yaml"
    log_warn "Please edit this file and set your API_SECRET before starting the service"
else
    log_info "Configuration file already exists at $CONFIG_DIR/capture.yaml"
fi

# Install systemd service
if [[ -f "./docs/systemd/capture.service" ]]; then
    log_info "Installing systemd service..."
    cp "./docs/systemd/capture.service" "/etc/systemd/system/"
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    log_info "Service installed and enabled"
else
    log_warn "Systemd service file not found, creating basic one..."
    cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOF
[Unit]
Description=Capture Hardware Monitoring Agent
After=network.target

[Service]
Type=simple
User=$USER
Group=$GROUP
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/capture --config $CONFIG_DIR/capture.yaml
Restart=always
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
fi

# Validate configuration before finishing
log_info "Validating configuration..."
if sudo -u "$USER" "$INSTALL_DIR/capture" --validate-config "$CONFIG_DIR/capture.yaml" 2>/dev/null; then
    log_info "Configuration is valid"
else
    log_warn "Configuration validation failed - please check $CONFIG_DIR/capture.yaml"
fi

log_info "Installation completed successfully!"
echo
echo "Next steps:"
echo "1. Edit the configuration file: $CONFIG_DIR/capture.yaml"
echo "2. Set your API_SECRET in the configuration file"
echo "3. Start the service: systemctl start $SERVICE_NAME"
echo "4. Check status: systemctl status $SERVICE_NAME"
echo "5. View logs: journalctl -u $SERVICE_NAME -f"
echo
echo "Configuration commands:"
echo "- Show current config: $INSTALL_DIR/capture --config $CONFIG_DIR/capture.yaml --show-config"
echo "- Validate config: $INSTALL_DIR/capture --validate-config $CONFIG_DIR/capture.yaml"
echo "- Generate new config: $INSTALL_DIR/capture --generate-config /path/to/config.yaml"
