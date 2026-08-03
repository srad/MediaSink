#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="${ROOT_DIR}/server"
GOLANGCI_VERSION="${GOLANGCI_VERSION:-v2.12.2}"

# https://github.com/mattn/go-sqlite3/issues/803
export CGO_CFLAGS="${CGO_CFLAGS:--g -O2 -Wno-return-local-addr}"

GOLANGCI_BIN="$(go env GOPATH)/bin/golangci-lint"
if [[ ! -x "${GOLANGCI_BIN}" ]]; then
  echo "[lint.sh] Installing golangci-lint ${GOLANGCI_VERSION}..."
  GOWORK=off go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_VERSION}"
fi

echo "[lint.sh] gofmt..."
# Only the Go module; vendor/ and the Python venv are not ours to format.
unformatted="$(find "${SERVER_DIR}" -name '*.go' \
  -not -path '*/vendor/*' \
  -not -path '*/.venv-export/*' \
  -not -path '*/docs/*' \
  -print0 | xargs -0 gofmt -l)"

if [[ -n "${unformatted}" ]]; then
  echo "[lint.sh] The following files are not gofmt-clean:"
  echo "${unformatted}" | sed 's/^/[lint.sh]   - /'
  echo "[lint.sh] Fix with: gofmt -w <file>"
  exit 1
fi
echo "[lint.sh]   OK"

echo "[lint.sh] go vet..."
(cd "${SERVER_DIR}" && GOWORK=off go vet ./...)
echo "[lint.sh]   OK"

echo "[lint.sh] golangci-lint (config: server/.golangci.yml)..."
(cd "${SERVER_DIR}" && GOWORK=off "${GOLANGCI_BIN}" run ./...)

echo "[lint.sh] All lint checks passed."
