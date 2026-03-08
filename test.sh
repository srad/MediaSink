#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_ROOT="${TMPDIR:-/tmp}/mediasink-tests"
mkdir -p "${TMP_ROOT}/recordings" "${TMP_ROOT}/disk"

# Test/runtime defaults.
export CGO_CFLAGS="${CGO_CFLAGS:--g -O2 -Wno-return-local-addr}"
export DB_FILENAME="${DB_FILENAME:-:memory:}"
export DATA_DIR="${DATA_DIR:-.previews}"
export DATA_DISK="${DATA_DISK:-${TMP_ROOT}/disk}"
export NET_ADAPTER="${NET_ADAPTER:-lo}"
export REC_PATH="${REC_PATH:-/tmp}"
export SECRET="${SECRET:-test-secret}"

START_TS="$(date +%s)"
COVERPROFILE="${TMP_ROOT}/cover.out"
RAW_LOG="${TMP_ROOT}/go-test.log"

echo "[test.sh] Running full test suite..."
echo "[test.sh] REC_PATH=${REC_PATH}"
echo "[test.sh] DATA_DISK=${DATA_DISK}"

set +e
go test -count=1 -coverprofile="${COVERPROFILE}" ./... >"${RAW_LOG}" 2>&1
status=$?
set -e

ok_count="$(awk '/^ok[[:space:]]+github.com\/srad\/mediasink/ {c++} END{print c+0}' "${RAW_LOG}")"
no_test_count="$(awk '/^\?[[:space:]]+github.com\/srad\/mediasink/ {c++} END{print c+0}' "${RAW_LOG}")"
fail_count="$(awk '/^FAIL[[:space:]]+github.com\/srad\/mediasink/ {c++} END{print c+0}' "${RAW_LOG}")"

echo "[test.sh] Summary:"
echo "[test.sh]   passed packages: ${ok_count}"
echo "[test.sh]   no-test packages: ${no_test_count}"
echo "[test.sh]   failed packages: ${fail_count}"

if [[ "${status}" -ne 0 ]]; then
  echo "[test.sh] Failed package(s):"
  awk '/^FAIL[[:space:]]+github.com\/srad\/mediasink/ {print "[test.sh]   - " $2}' "${RAW_LOG}" | sort -u
  echo "[test.sh] Re-running failed package(s) with -v for details..."
  while read -r pkg; do
    [[ -z "${pkg}" ]] && continue
    echo "[test.sh] ---- ${pkg} ----"
    go test -count=1 -v "${pkg}" || true
  done < <(awk '/^FAIL[[:space:]]+github.com\/srad\/mediasink/ {print $2}' "${RAW_LOG}" | sort -u)
fi

if [[ -f "${COVERPROFILE}" ]]; then
  total_cov="$(go tool cover -func="${COVERPROFILE}" | tail -n 1)"
  echo "[test.sh] Coverage: ${total_cov}"
fi

END_TS="$(date +%s)"
if [[ "${status}" -eq 0 ]]; then
  echo "[test.sh] All tests passed in $((END_TS - START_TS))s."
else
  echo "[test.sh] Tests failed in $((END_TS - START_TS))s."
  echo "[test.sh] Raw log: ${RAW_LOG}"
fi

exit "${status}"
