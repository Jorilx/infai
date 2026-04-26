#!/bin/bash
set -e

# infai install script for Linux

REPO="dipankardas011/infai"

# Use provided version or fetch latest
if [ -n "$INFAI_VERSION" ]; then
    VERSION="$INFAI_VERSION"
    TAG="v$INFAI_VERSION"
else
    TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    VERSION=${TAG#v}
fi

# Detection
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi
if [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; fi

FILE="infai_${VERSION}_linux_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$TAG/$FILE"

echo "Downloading infai $TAG for Linux $ARCH..."

TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

curl -LO "$URL"
tar -xzf "$FILE"

echo "Installing to /usr/local/bin/infai..."
sudo install -m 0755 infai /usr/local/bin/infai

# Cleanup
cd -
rm -rf "$TMP_DIR"

echo "Installation complete! Run 'infai' to get started."
