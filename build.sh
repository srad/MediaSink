#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="${ROOT_DIR}/server"
EMBED_DIST_DIR="${SERVER_DIR}/frontend/dist"
SWAG_BIN="$(go env GOPATH)/bin/swag"

# Build frontend
echo "Building frontend..."
(cd "${ROOT_DIR}/frontend" && npm install && npm run build)

mkdir -p "${SERVER_DIR}/frontend" "${EMBED_DIST_DIR}"
rm -rf "${EMBED_DIST_DIR}"
mkdir -p "${EMBED_DIST_DIR}"
cp -R "${ROOT_DIR}/frontend/dist/." "${EMBED_DIST_DIR}/"

go install github.com/swaggo/swag/cmd/swag@latest
(cd "${SERVER_DIR}" && GOWORK=off "${SWAG_BIN}" init --parseDependency --parseInternal -g main.go -o docs)

# https://github.com/mattn/go-sqlite3/issues/803
export CGO_CFLAGS="-g -O2 -Wno-return-local-addr"
VERSION=dev
COMMIT="$(git rev-parse --short HEAD)"
API_VERSION="${API_VERSION:-0.1.0}"
(cd "${SERVER_DIR}" && GOWORK=off go mod vendor && GOWORK=off go build -o "${ROOT_DIR}/main" -ldflags="-X 'main.Version=$VERSION' -X 'main.Commit=$COMMIT' -X 'main.ApiVersion=$API_VERSION'" -mod=mod)
