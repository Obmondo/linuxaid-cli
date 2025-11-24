#!/usr/bin/env bash
set -euo pipefail

PROJECT="linuxaid-cli"
REPO="Obmondo/$PROJECT"
API_URL="https://api.github.com/repos/$REPO/releases/latest"
BIN_DIR="/usr/local/bin"
BIN_NAME="linuxaid-install"

function cleanup() {
    rm -rf "$TMP_DIR"
    exit 130
}

# -------------------------------------------------
# 1. Detect architecture
# -------------------------------------------------
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)   TARGET="amd64"   ;;
    aarch64)  TARGET="arm64"   ;;
    armv7l)   TARGET="armv7"   ;;
    *) 
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

# -------------------------------------------------
# 2. Find latest release tag on GitHub
# -------------------------------------------------
TAG=$(curl -sSf "$API_URL" | jq -r '.tag_name')
if [[ -z "$TAG" ]]; then
    echo "Failed to obtain latest release tag" >&2
    exit 1
fi

# -------------------------------------------------
# 3. Build download URL for the appropriate asset
# -------------------------------------------------
ASSET="${PROJECT}_${TAG}_linux_${TARGET}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG/$ASSET"
SOURCE_CHECKSUM=$(curl -sSf "https://api.github.com/repos/$REPO/releases/latest" | jq -r --arg url "$DOWNLOAD_URL" '.assets[] | select(.browser_download_url == $url) | .digest | split(":")[1]')

# -------------------------------------------------
# 4. Download the package from Github
# -------------------------------------------------
# Trap SIGTERM (15), SIGINT (2 - Ctrl+C), and SIGHUP (1)
trap cleanup SIGTERM SIGINT SIGHUP EXIT

TMP_DIR=$(mktemp -d /tmp/${PROJECT}-XXX)
curl -sL -o "$TMP_DIR/$ASSET" "$DOWNLOAD_URL"

# -------------------------------------------------
# 5. Checksum verification
# -------------------------------------------------
ACTUAL_CHECKSUM=$(sha256sum "$TMP_DIR/$ASSET" | cut -d' ' -f1)
if [[ "$SOURCE_CHECKSUM" != "$ACTUAL_CHECKSUM" ]]; then
    echo "Failed to verify the checksum" >&2
    exit 1
fi

# -------------------------------------------------
# 6. Extract and execute the linuxaid-install binary
# -------------------------------------------------
tar -xzf "$TMP_DIR/$ASSET" -C "$TMP_DIR"
mv "$TMP_DIR/$BIN_NAME" "$BIN_DIR"

"$BIN_DIR/$BIN_NAME"
