#!/bin/bash

set -e

# Determine OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Convert architecture names
case ${ARCH} in
x86_64) ARCH="amd64" ;;
aarch64) ARCH="arm64" ;;
i386) ARCH="386" ;;
i686) ARCH="386" ;;
*)
  echo "Unsupported architecture: ${ARCH}"
  exit 1
  ;;
esac

# Verify OS is supported
case ${OS} in
linux | darwin) ;;
*)
  echo "Unsupported OS: ${OS}"
  exit 1
  ;;
esac

# Get version and download URL
VERSION=${1:-latest}
if [ "$VERSION" = "latest" ]; then
  API_RESPONSE=$(curl -s https://api.github.com/repos/y0ug/ai-helper/releases/$VERSION)
  VERSION=$(echo "$API_RESPONSE" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)
  DOWNLOAD_URLS=$(echo "$API_RESPONSE" | grep -o '"browser_download_url": "[^"]*"' | grep "${OS}-${ARCH}" | cut -d'"' -f4)
  IFS=' ' read -r DOWNLOAD_URL DOWNLOAD_URL_HASH <<<"$DOWNLOAD_URLS"
  if [ -z "$VERSION" ]; then
    echo "Failed to get version $VERSION"
    exit 1
  fi
  if [ -z "$DOWNLOAD_URL" ]; then
    echo "Failed to find download URL for ${OS}/${ARCH}"
    exit 1
  fi
fi

BINARY_NAME=$(echo "$DOWNLOAD_URL" | cut -d'/' -f9)
echo $DOWNLOAD_URL
echo $BINARY_NAME
# Create ~/.local/bin if it doesn't exist
mkdir -p "$HOME/.local/bin"

# Download and install
echo "Downloading ai-helper ${VERSION} for ${OS}/${ARCH}..."
TEMP_DIR=$(mktemp -d)
curl -L "${DOWNLOAD_URL}" -o "${TEMP_DIR}/${BINARY_NAME}"
cd "${TEMP_DIR}"
tar xzf "${BINARY_NAME}"
mv ai-helper ~/.local/bin/ai-helper
chmod +x ~/.local/bin/ai-helper
rm -rf "${TEMP_DIR}"

echo "Installation complete! ai-helper has been installed to ~/.local/bin/ai-helper"
echo
echo "Make sure ~/.local/bin is in your PATH. You can add this to your shell's rc file:"
echo 'export PATH="$HOME/.local/bin:$PATH"'
