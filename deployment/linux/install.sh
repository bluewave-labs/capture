#!/usr/bin/env bash
# Installs the latest Capture release from GitHub and registers it as a
# systemd service.
#
# Usage:
#   sudo bash install.sh
#   sudo bash install.sh --api-secret "your-secret-key"
#   sudo bash install.sh --api-secret "your-secret-key" --port 8080
#   sudo bash install.sh --api-secret "your-secret-key" \
#                        --install-dir /opt/capture     \
#                        --service-name capture
#
# Options:
#   --api-secret   <string>   Authentication key for the Capture API  (required, prompted if omitted)
#   --port         <int>      Port the server listens on              (default: 59232)
#   --install-dir  <path>     Directory to install the binary         (default: /usr/local/bin)
#   --service-name <name>     systemd service name                    (default: capture)

set -euo pipefail

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

step()    { echo -e "\n\033[0;36m==> $*\033[0m"; }
success() { echo -e "    \033[0;32m$*\033[0m"; }
warn()    { echo -e "    \033[0;33mWARNING: $*\033[0m"; }
die()     { echo -e "\n\033[0;31mERROR: $*\033[0m" >&2; exit 1; }

# ---------------------------------------------------------------------------
# Require root
# ---------------------------------------------------------------------------

if [[ $EUID -ne 0 ]]; then
    die "This script must be run as root (sudo)."
fi

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------

API_SECRET=""
PORT=59232
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="capture"

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------

while [[ $# -gt 0 ]]; do
    case "$1" in
        --api-secret)   API_SECRET="$2";   shift 2 ;;
        --port)         PORT="$2";         shift 2 ;;
        --install-dir)  INSTALL_DIR="$2";  shift 2 ;;
        --service-name) SERVICE_NAME="$2"; shift 2 ;;
        *) die "Unknown option: $1" ;;
    esac
done

# ---------------------------------------------------------------------------
# Step 1 – Detect architecture
# ---------------------------------------------------------------------------

step "Detecting system architecture"

MACHINE=$(uname -m)
case "$MACHINE" in
    x86_64)         ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    *) die "Unsupported architecture: $MACHINE" ;;
esac

success "Architecture: $ARCH"

# ---------------------------------------------------------------------------
# Step 2 – Fetch latest release from GitHub
# ---------------------------------------------------------------------------

step "Fetching latest release information from GitHub"

RELEASE_JSON=$(curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    https://api.github.com/repos/bluewave-labs/capture/releases/latest)

VERSION=$(echo "$RELEASE_JSON" | grep -m1 '"tag_name"' | cut -d'"' -f4)          # e.g. v1.3.2
VERSION_NUM=${VERSION#v}                                                            # e.g. 1.3.2

success "Latest version: $VERSION"

# ---------------------------------------------------------------------------
# Step 3 – Resolve download URL
# ---------------------------------------------------------------------------

step "Resolving download URL"

ASSET_NAME="capture_${VERSION_NUM}_linux_${ARCH}.tar.gz"
DOWNLOAD_URL=$(echo "$RELEASE_JSON" \
    | grep "browser_download_url" \
    | grep "$ASSET_NAME" \
    | cut -d'"' -f4)

if [[ -z "$DOWNLOAD_URL" ]]; then
    die "Could not find asset '$ASSET_NAME' in the latest release. \
Check https://github.com/bluewave-labs/capture/releases/latest for available assets."
fi

success "Download URL: $DOWNLOAD_URL"

# ---------------------------------------------------------------------------
# Step 4 – Download and extract
# ---------------------------------------------------------------------------

step "Downloading archive"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

ARCHIVE_PATH="$TMP_DIR/$ASSET_NAME"
curl -fsSL --output "$ARCHIVE_PATH" "$DOWNLOAD_URL"
success "Saved to: $ARCHIVE_PATH"

step "Extracting archive"

EXTRACT_DIR="$TMP_DIR/extracted"
mkdir -p "$EXTRACT_DIR"
tar -xzf "$ARCHIVE_PATH" -C "$EXTRACT_DIR"

BINARY_PATH=$(find "$EXTRACT_DIR" -type f -name "capture" | head -n1)
if [[ -z "$BINARY_PATH" ]]; then
    die "capture binary not found in the extracted archive."
fi

success "Binary found: $BINARY_PATH"

# ---------------------------------------------------------------------------
# Step 5 – Install binary
# ---------------------------------------------------------------------------

step "Installing to $INSTALL_DIR"

mkdir -p "$INSTALL_DIR"

# Stop the existing service before replacing the binary
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    warn "Running service '$SERVICE_NAME' found – stopping it before upgrade."
    systemctl stop "$SERVICE_NAME"
fi

INSTALL_PATH="$INSTALL_DIR/capture"
install -m 0755 "$BINARY_PATH" "$INSTALL_PATH"
success "Binary installed: $INSTALL_PATH"

# ---------------------------------------------------------------------------
# Step 6 – Collect configuration
# ---------------------------------------------------------------------------

if [[ -z "$API_SECRET" ]]; then
    step "Configuration"
    read -rsp "Enter API_SECRET (required): " API_SECRET
    echo
fi

if [[ -z "$API_SECRET" ]]; then
    die "API_SECRET must not be empty."
fi

# ---------------------------------------------------------------------------
# Step 7 – Create / update systemd service
# ---------------------------------------------------------------------------

step "Writing systemd service unit"

SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Capture hardware monitoring agent
Documentation=https://github.com/bluewave-labs/capture
After=network.target

[Service]
ExecStart=${INSTALL_PATH}
WorkingDirectory=${INSTALL_DIR}
Restart=always
RestartSec=3
Type=simple
ProtectSystem=strict
ProtectHome=true
NoNewPrivileges=true
RestrictNamespaces=yes
ProtectKernelTunables=yes
ProtectKernelLogs=yes
ProtectControlGroups=yes
Environment="API_SECRET=${API_SECRET}"
Environment="PORT=${PORT}"
Environment="GIN_MODE=release"

[Install]
WantedBy=multi-user.target
EOF

success "Service unit written: $SERVICE_FILE"

# ---------------------------------------------------------------------------
# Step 8 – Enable and start
# ---------------------------------------------------------------------------

step "Enabling and starting service '$SERVICE_NAME'"

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl start  "$SERVICE_NAME"

if systemctl is-active --quiet "$SERVICE_NAME"; then
    success "Service is running."
else
    warn "Service did not start cleanly. Check logs with: journalctl -u $SERVICE_NAME -n 50"
fi

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------

echo ""
echo "================================================="
echo "  Capture $VERSION installed successfully!"
echo "================================================="
echo ""
echo "  Install path  : $INSTALL_PATH"
echo "  Service name  : $SERVICE_NAME"
echo "  Port          : $PORT"
echo ""
echo "  Useful commands:"
echo "    systemctl status  $SERVICE_NAME"
echo "    systemctl stop    $SERVICE_NAME"
echo "    systemctl start   $SERVICE_NAME"
echo "    systemctl restart $SERVICE_NAME"
echo "    journalctl -u     $SERVICE_NAME -f"
echo ""
echo "  Health check:"
echo "    curl http://localhost:$PORT/health"
echo ""
