#!/usr/bin/env bash

set -euo pipefail

ONNX_VERSION="1.24.1"
ARCH="$(uname -m)"

case "${ARCH}" in
  x86_64)  ONNX_ARCH="x64" ;;
  aarch64) ONNX_ARCH="aarch64" ;;
  *)
    echo "[install-onnxruntime] Unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

TARBALL="onnxruntime-linux-${ONNX_ARCH}-${ONNX_VERSION}.tgz"
URL="https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/${TARBALL}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEST="${ROOT_DIR}/lib/libonnxruntime.so"

if [[ -f "${DEST}" ]]; then
  echo "[install-onnxruntime] Already installed: ${DEST}"
  exit 0
fi

echo "[install-onnxruntime] Downloading onnxruntime ${ONNX_VERSION} (${ONNX_ARCH})..."
TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT

curl -fsSL --progress-bar "${URL}" -o "${TMP}/${TARBALL}"

echo "[install-onnxruntime] Extracting..."
tar -xzf "${TMP}/${TARBALL}" -C "${TMP}"

EXTRACTED_LIB="$(find "${TMP}" -name "libonnxruntime.so" | head -1)"
if [[ -z "${EXTRACTED_LIB}" ]]; then
  echo "[install-onnxruntime] libonnxruntime.so not found in archive" >&2
  exit 1
fi

mkdir -p "${ROOT_DIR}/lib"
cp "${EXTRACTED_LIB}" "${DEST}"
chmod 755 "${DEST}"

echo "[install-onnxruntime] Installed: ${DEST}"
