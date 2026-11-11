#!/usr/bin/env bash
# Installs the latest Capture release from GitHub and registers it as a
# launchd daemon.
#
# Usage:
#   sudo bash install.sh
#   sudo bash install.sh --api-secret "your-secret-key"
#   sudo bash install.sh --api-secret "your-secret-key" --port 8080 --listen 127.0.0.1
#   sudo bash install.sh --api-secret "your-secret-key" \
#                        --install-dir /usr/local/bin       \
#                        --service-name com.bluewavelabs.capture
#
# Options:
#   --api-secret   <string>   Authentication key for the Capture API  (required, prompted if omitted)
#   --port         <int>      Port the server listens on              (default: 59232)
#   --listen       <address>  IP address the server binds to          (default: 0.0.0.0)
#   --install-dir  <path>     Directory to install the binary         (default: /usr/local/bin)
#   --service-name <name>     launchd label and plist name            (default: com.bluewavelabs.capture)

set -euo pipefail

step()    { echo -e "\n\033[0;36m==> $*\033[0m"; }
success() { echo -e "    \033[0;32m$*\033[0m"; }
warn()    { echo -e "    \033[0;33mWARNING: $*\033[0m"; }
die()     { echo -e "\n\033[0;31mERROR: $*\033[0m" >&2; exit 1; }

if [[ $EUID -ne 0 ]]; then
    die "This script must be run as root (sudo)."
fi

API_SECRET=""
PORT=59232
LISTEN="0.0.0.0"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="com.bluewavelabs.capture"
PLIST_DIR="/Library/LaunchDaemons"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --api-secret)   API_SECRET="$2";   shift 2 ;;
        --port)         PORT="$2";         shift 2 ;;
        --listen)       LISTEN="$2";       shift 2 ;;
        --install-dir)  INSTALL_DIR="$2";  shift 2 ;;
        --service-name) SERVICE_NAME="$2"; shift 2 ;;
        *) die "Unknown option: $1" ;;
    esac
done

step "Detecting system architecture"

MACHINE=$(uname -m)
case "$MACHINE" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) die "Unsupported architecture: $MACHINE" ;;
esac

success "Architecture: $ARCH"

step "Fetching latest release information from GitHub"

RELEASE_JSON=$(curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    https://api.github.com/repos/bluewave-labs/capture/releases/latest)

VERSION=$(echo "$RELEASE_JSON" | grep -m1 '"tag_name"' | cut -d'"' -f4)
VERSION_NUM=${VERSION#v}

success "Latest version: $VERSION"

step "Resolving download URL"

ASSET_NAME="capture_${VERSION_NUM}_darwin_${ARCH}.tar.gz"
DOWNLOAD_URL=$(echo "$RELEASE_JSON" \
    | grep 'browser_download_url' \
    | grep "$ASSET_NAME" \
    | cut -d'"' -f4)

if [[ -z "$DOWNLOAD_URL" ]]; then
    die "Could not find asset '$ASSET_NAME' in the latest release. Check https://github.com/bluewave-labs/capture/releases/latest for available assets."
fi

success "Download URL: $DOWNLOAD_URL"

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

BINARY_PATH=$(find "$EXTRACT_DIR" -type f -name capture | head -n1)
if [[ -z "$BINARY_PATH" ]]; then
    die "capture binary not found in the extracted archive."
fi

success "Binary found: $BINARY_PATH"

step "Installing to $INSTALL_DIR"

mkdir -p "$INSTALL_DIR"

INSTALL_PATH="$INSTALL_DIR/capture"
install -m 0755 "$BINARY_PATH" "$INSTALL_PATH"
success "Binary installed: $INSTALL_PATH"

if [[ -z "$API_SECRET" ]]; then
    step "Configuration"
    read -rsp "Enter API_SECRET (required): " API_SECRET
    echo
fi

if [[ -z "$API_SECRET" ]]; then
    die "API_SECRET must not be empty."
fi

PLIST_FILE="$PLIST_DIR/${SERVICE_NAME}.plist"
STDOUT_LOG="/var/log/${SERVICE_NAME}.log"
STDERR_LOG="/var/log/${SERVICE_NAME}.error.log"

step "Writing launchd plist"

mkdir -p "$PLIST_DIR"
touch "$STDOUT_LOG" "$STDERR_LOG"
chmod 0644 "$STDOUT_LOG" "$STDERR_LOG"

cat > "$PLIST_FILE" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${SERVICE_NAME}</string>

    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_PATH}</string>
    </array>

    <key>WorkingDirectory</key>
    <string>${INSTALL_DIR}</string>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>EnvironmentVariables</key>
    <dict>
        <key>API_SECRET</key>
        <string>${API_SECRET}</string>
        <key>PORT</key>
        <string>${PORT}</string>
        <key>LISTEN</key>
        <string>${LISTEN}</string>
        <key>GIN_MODE</key>
        <string>release</string>
    </dict>

    <key>StandardOutPath</key>
    <string>${STDOUT_LOG}</string>

    <key>StandardErrorPath</key>
    <string>${STDERR_LOG}</string>
</dict>
</plist>
EOF

chown root:wheel "$PLIST_FILE"
chmod 0644 "$PLIST_FILE"
success "Plist written: $PLIST_FILE"

step "Loading service '$SERVICE_NAME'"

launchctl unload "$PLIST_FILE" 2>/dev/null || true
launchctl load "$PLIST_FILE"
launchctl start "$SERVICE_NAME"

if launchctl list | grep -q "$SERVICE_NAME"; then
    success "Service is loaded."
else
    warn "Service did not load cleanly. Check logs at: $STDERR_LOG"
fi

echo ""
echo "================================================="
echo "  Capture $VERSION installed successfully!"
echo "================================================="
echo ""
echo "  Install path  : $INSTALL_PATH"
echo "  Service name  : $SERVICE_NAME"
echo "  Listen address: $LISTEN"
echo "  Port          : $PORT"
echo "  Plist path    : $PLIST_FILE"
echo ""
echo "  Useful commands:"
echo "    sudo launchctl load $PLIST_FILE"
echo "    sudo launchctl start $SERVICE_NAME"
echo "    sudo launchctl list | grep $SERVICE_NAME"
echo "    tail -f $STDOUT_LOG"
echo "    tail -f $STDERR_LOG"
echo ""
HEALTH_HOST="$LISTEN"
case "$HEALTH_HOST" in
    ""|0.0.0.0) HEALTH_HOST="127.0.0.1" ;;
    ::) HEALTH_HOST="[::1]" ;;
    *:*) HEALTH_HOST="[$HEALTH_HOST]" ;;
esac
echo "  Health check:"
echo "    curl http://$HEALTH_HOST:$PORT/health"
echo ""