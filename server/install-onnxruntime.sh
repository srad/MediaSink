#!/usr/bin/env bash

set -euo pipefail

# Must be >= the ORT API version requested by github.com/yalue/onnxruntime_go
# (see the comment on that require line in go.mod). An older runtime aborts at startup.
ONNX_VERSION="1.28.0"
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
# The library filename carries no version, so track it separately — otherwise a stale
# copy from an older ONNX_VERSION is silently kept and the server aborts at startup.
STAMP="${ROOT_DIR}/lib/.onnx-version"

if [[ -f "${DEST}" ]] && [[ "$(cat "${STAMP}" 2>/dev/null || true)" == "${ONNX_VERSION}" ]]; then
  echo "[install-onnxruntime] Already installed: ${DEST} (${ONNX_VERSION})"
  exit 0
fi

if [[ -f "${DEST}" ]]; then
  echo "[install-onnxruntime] Replacing $(cat "${STAMP}" 2>/dev/null || echo "unknown version") with ${ONNX_VERSION}"
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
echo "${ONNX_VERSION}" > "${STAMP}"

echo "[install-onnxruntime] Installed: ${DEST} (${ONNX_VERSION})"
