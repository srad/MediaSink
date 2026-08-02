#!/usr/bin/env bash

set -euo pipefail

# Runtime defaults for local development.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="${ROOT_DIR}/server"
EMBED_DIST_DIR="${SERVER_DIR}/frontend/dist"
RUNTIME_DIR="${ROOT_DIR}/.runtime"

export DB_FILENAME="${DB_FILENAME:-${RUNTIME_DIR}/mediasink.sqlite3}"
export REC_PATH="${REC_PATH:-${RUNTIME_DIR}/recordings}"
export DATA_DIR="${DATA_DIR:-.previews}"
export DATA_DISK="${DATA_DISK:-${RUNTIME_DIR}/disk}"
export NET_ADAPTER="${NET_ADAPTER:-lo}"
export DB_ADAPTER="${DB_ADAPTER:-sqlite}"

# JWT secret required by main init.
if [[ -z "${SECRET:-}" ]]; then
  export SECRET="dev-secret-$(date +%s)"
  echo "[run.sh] SECRET was not set; generated ephemeral development secret."
fi

mkdir -p "${RUNTIME_DIR}" "${REC_PATH}" "${DATA_DISK}"

for bin in ffmpeg yt-dlp ffprobe; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "[run.sh] Missing required executable: ${bin}" >&2
    exit 1
  fi
done

export CGO_CFLAGS="${CGO_CFLAGS:--g -O2 -Wno-return-local-addr}"

# ONNX runtime shared library — set ONNXRUNTIME_LIB to the path of
# libonnxruntime.so if it is not on the system library path.
# Easiest route: ./server/install-onnxruntime.sh (installs into server/lib/).
# Otherwise download onnxruntime 1.28.0 from:
#   https://github.com/microsoft/onnxruntime/releases/tag/v1.28.0
# and extract it, then set the variable:
#   export ONNXRUNTIME_LIB=/path/to/onnxruntime-linux-x64-1.28.0/lib/libonnxruntime.so
# The runtime must be at least as new as the ORT API version requested by
# github.com/yalue/onnxruntime_go (see server/go.mod), or the server aborts at startup.
if [[ -z "${ONNXRUNTIME_LIB:-}" ]]; then
  # Auto-detect: look for a local copy next to this script. The extracted-tarball
  # paths are globbed so they stay valid across onnxruntime versions.
  for candidate in \
    "${ROOT_DIR}/server/onnxruntime-linux-"*"/lib/libonnxruntime.so" \
    "${ROOT_DIR}/server/lib/libonnxruntime.so" \
    "${ROOT_DIR}/server/libonnxruntime.so" \
    "${ROOT_DIR}/onnxruntime-linux-"*"/lib/libonnxruntime.so" \
    "${ROOT_DIR}/lib/libonnxruntime.so" \
    "${ROOT_DIR}/libonnxruntime.so"; do
    if [[ -f "${candidate}" ]]; then
      export ONNXRUNTIME_LIB="${candidate}"
      echo "[run.sh] ONNXRUNTIME_LIB=${ONNXRUNTIME_LIB}"
      break
    fi
  done
fi

# Generate latest swagger.json from Go annotations
echo "[run.sh] Generating swagger docs..."
SWAG_BIN="$(go env GOPATH)/bin/swag"
if [[ ! -x "${SWAG_BIN}" ]]; then
  echo "[run.sh] Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest
fi
(cd "${SERVER_DIR}" && GOWORK=off "${SWAG_BIN}" init --parseDependency --parseInternal -g main.go -o docs)

# Generate API client from the freshly generated swagger.json (no server needed)
echo "[run.sh] Generating API client..."
(cd "${ROOT_DIR}/frontend" && npm install && SWAGGER_INPUT="${ROOT_DIR}/server/docs/swagger.json" node swagger.js)

# Build frontend (always rebuild to pick up source changes)
echo "[run.sh] Building frontend..."
(cd "${ROOT_DIR}/frontend" && npm run build)

mkdir -p "${SERVER_DIR}/frontend" "${EMBED_DIST_DIR}"
rm -rf "${EMBED_DIST_DIR}"
mkdir -p "${EMBED_DIST_DIR}"
cp -R "${ROOT_DIR}/frontend/dist/." "${EMBED_DIST_DIR}/"

echo "[run.sh] Building mediasink..."
VERSION="${VERSION:-dev}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
API_VERSION="0.1.0"
(cd "${SERVER_DIR}" && GOWORK=off go build -o "${ROOT_DIR}/main" -ldflags="-X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.ApiVersion=$API_VERSION'" -mod=mod)

echo "[run.sh] Starting mediasink..."
echo "[run.sh] DB_FILENAME=${DB_FILENAME}"
echo "[run.sh] REC_PATH=${REC_PATH}"
echo "[run.sh] DATA_DISK=${DATA_DISK}"
exec "${ROOT_DIR}/main"
