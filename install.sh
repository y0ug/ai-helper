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

# Get version
VERSION=${1:-latest}
if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -s https://api.github.com/repos/y0ug/ai-helper/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "Failed to get latest version"
        exit 1
    fi
fi

# Construct download URL
BINARY_NAME="ai-helper_${OS}_${ARCH}"
DOWNLOAD_URL="https://github.com/y0ug/ai-helper/releases/download/${VERSION}/${BINARY_NAME}"

# Create ~/.local/bin if it doesn't exist
mkdir -p ~/.local/bin

# Download and install
echo "Downloading ai-helper ${LATEST_VERSION} for ${OS}/${ARCH}..."
curl -L "${DOWNLOAD_URL}" -o ~/.local/bin/ai-helper
chmod +x ~/.local/bin/ai-helper

echo "Installation complete! ai-helper has been installed to ~/.local/bin/ai-helper"
echo
echo "Make sure ~/.local/bin is in your PATH. You can add this to your shell's rc file:"
echo 'export PATH="$HOME/.local/bin:$PATH"'
