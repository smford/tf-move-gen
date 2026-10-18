#!/usr/bin/env bash
set -euo pipefail

# Installer script for tf-move-gen
REPO="smford/tf-move-gen"
BINARY="tf-move-gen"

echo "Installing ${BINARY}..."

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  *)
    echo "Error: Unsupported architecture $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  darwin|linux)
    ;;
  *)
    echo "Error: Unsupported operating system $OS. Please download binaries manually from GitHub Releases." >&2
    exit 1
    ;;
esac

LATEST_TAG=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
if [ -z "$LATEST_TAG" ]; then
  LATEST_TAG="v0.1.0"
fi
CLEAN_VERSION="${LATEST_TAG#v}"

ARCHIVE="${BINARY}_${CLEAN_VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ARCHIVE}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${BINARY} ${LATEST_TAG} for ${OS}/${ARCH}..."
curl -sSL "$URL" -o "${TMP_DIR}/${ARCHIVE}"

tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"

INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
  if command -v sudo >/dev/null 2>&1; then
    echo "Installing ${BINARY} to ${INSTALL_DIR} using sudo..."
    sudo install -m 0755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
    echo "Installing ${BINARY} to ${INSTALL_DIR}..."
    install -m 0755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    echo "Note: Please ensure ${INSTALL_DIR} is in your PATH."
  fi
else
  echo "Installing ${BINARY} to ${INSTALL_DIR}..."
  install -m 0755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "Successfully installed ${BINARY} $("${INSTALL_DIR}/${BINARY}" --version || echo "${LATEST_TAG}")"
