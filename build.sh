#!/bin/bash

set -euo pipefail

# Build frontend
echo "Building frontend..."
(cd frontend && npm install && npm run build)

go install github.com/swaggo/swag/cmd/swag@latest
swag init

# https://github.com/mattn/go-sqlite3/issues/803
export CGO_CFLAGS="-g -O2 -Wno-return-local-addr"
VERSION=dev
COMMIT="$(git rev-parse --short HEAD)"
API_VERSION="${API_VERSION:-0.1.0}"
go mod vendor
go build -o ./main -ldflags="-X 'main.Version=$VERSION' -X 'main.Commit=$COMMIT' -X 'main.ApiVersion=$API_VERSION'" -mod=mod
