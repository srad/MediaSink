#!/usr/bin/env bash
#
# Runs the Go test suite AND a real boot smoke test inside a container.
#
# Why this exists: ./test.sh proves the packages pass, but it can never prove the
# server boots. app.validateEnvironment requires ffmpeg, ffprobe, yt-dlp and a
# loadable ONNX runtime, so on a machine without them the binary exits before it
# ever listens. This image supplies all four.
#
#   ./docker-test.sh              build the image if needed, then run everything
#   ./docker-test.sh --rebuild    force a fresh image build first
#   ./docker-test.sh --in-container   what runs INSIDE the container (not for hosts)
#
set -uo pipefail

IMAGE="mediasink-test"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---------------------------------------------------------------------------
# Host side: build the image, then run this same script inside it.
# ---------------------------------------------------------------------------
run_on_host() {
  local rebuild="$1"

  if [[ "${rebuild}" == "1" ]] || ! docker image inspect "${IMAGE}" >/dev/null 2>&1; then
    echo "[docker-test] Building ${IMAGE} (first build pulls ~500 MB and takes a few minutes)..."
    docker build -f "${ROOT_DIR}/test.Dockerfile" -t "${IMAGE}" "${ROOT_DIR}" || {
      echo "[docker-test] Image build failed." >&2
      exit 1
    }
  else
    echo "[docker-test] Reusing existing ${IMAGE} image (--rebuild to force)."
  fi

  echo "[docker-test] Running suite and smoke test..."
  # No pipe to tail/grep here: that buffers everything until exit and hides progress.
  docker run --rm "${IMAGE}"
}

# ---------------------------------------------------------------------------
# Container side.
# ---------------------------------------------------------------------------
API="http://127.0.0.1:3000/api/v2"
HDR_VERSION="X-API-Version: 1.0"
CREDS='{"username":"smoke@example.com","password":"smoke-pass-123"}'

failures=0
pass() { echo "  PASS  $1"; }
fail() { echo "  FAIL  $1"; failures=$((failures + 1)); }
check() { # check <description> <actual> <expected>
  if [[ "$2" == "$3" ]]; then pass "$1 ($2)"; else fail "$1: got '$2', want '$3'"; fi
}
section() {
  echo
  echo "=============================================================="
  echo "$1"
  echo "=============================================================="
}

SERVER_PID=""

# Backgrounds the server and sets SERVER_PID.
#
# Deliberately NOT `SERVER_PID=$(start_server ...)`. Backgrounding a long-lived
# process inside a command substitution hangs the assignment: bash reads the
# substitution's pipe until EOF, and the server holds it open for its whole life.
start_server() { # start_server <secret> <logfile>
  env DB_FILENAME=/smoke/smoke.db \
      REC_PATH=/smoke/rec \
      DATA_DIR=.previews \
      DATA_DISK=/ \
      NET_ADAPTER=eth0 \
      SECRET="$1" \
      LOG_LEVEL=info \
      ONNXRUNTIME_LIB=/usr/local/lib/libonnxruntime.so \
      mediasink >"$2" 2>&1 &
  SERVER_PID=$!
}

stop_server() {
  [[ -n "${SERVER_PID}" ]] || return 0
  kill "${SERVER_PID}" 2>/dev/null
  # The server drains its worker pools on SIGTERM; give it a moment, then insist.
  for _ in $(seq 1 10); do
    kill -0 "${SERVER_PID}" 2>/dev/null || { SERVER_PID=""; return 0; }
    sleep 1
  done
  kill -9 "${SERVER_PID}" 2>/dev/null
  SERVER_PID=""
}

# Any HTTP status at all means it is listening; the route itself returns 4xx here.
wait_for_server() {
  for _ in $(seq 1 60); do
    if curl -sS -o /dev/null --max-time 3 "${API}/auth/login" 2>/dev/null; then return 0; fi
    sleep 1
  done
  return 1
}

http_code() { # http_code <curl args...>
  curl -sS -o /dev/null -w '%{http_code}' --max-time 10 "$@" 2>/dev/null
}

run_in_container() {
  trap stop_server EXIT

  section "1. Go test suite"
  # Scoped to this one command, never exported: an exported SECRET would leak into
  # the "SECRET is missing" case below, the server would boot instead of exiting,
  # and the command substitution there would hang forever waiting for it.
  # Run once. Streamed, not captured, so progress is visible.
  if DB_FILENAME=/tmp/suite.db REC_PATH=/tmp DATA_DIR=.previews \
     DATA_DISK=/ NET_ADAPTER=eth0 SECRET=suite-secret go test ./...; then
    pass "go test ./..."
  else
    fail "go test ./... reported failures"
  fi

  section "2. The server boots"
  mkdir -p /smoke/rec
  start_server "first-secret" /smoke/server1.log
  if wait_for_server; then
    pass "listening on :3000"
  else
    fail "server never came up; log follows"
    cat /smoke/server1.log
    return 1
  fi

  section "3. Signup, login, authenticated route"
  check "POST /auth/signup" \
    "$(http_code -X POST -H "${HDR_VERSION}" -H 'Content-Type: application/json' -d "${CREDS}" "${API}/auth/signup")" "200"

  curl -sS -o /tmp/login.json -X POST -H "${HDR_VERSION}" \
       -H 'Content-Type: application/json' -d "${CREDS}" "${API}/auth/login" >/dev/null 2>&1
  TOKEN=$(sed -n 's/.*"token":"\([^"]*\)".*/\1/p' /tmp/login.json)
  if [[ -n "${TOKEN}" ]]; then
    pass "login returned a JWT (${#TOKEN} chars)"
  else
    fail "no token in $(cat /tmp/login.json)"
  fi

  check "GET /user/profile WITH token" \
    "$(http_code -H "${HDR_VERSION}" -H "Authorization: Bearer ${TOKEN}" "${API}/user/profile")" "200"
  check "GET /user/profile WITHOUT token" \
    "$(http_code -H "${HDR_VERSION}" "${API}/user/profile")" "401"
  check "GET /user/profile with a malformed token" \
    "$(http_code -H "${HDR_VERSION}" -H "Authorization: Bearer garbage.token.here" "${API}/user/profile")" "401"

  section "4. The JWT is bound to the configured secret"
  # Same database, different SECRET. If the secret really comes from the config
  # value threaded in at construction, the old token has to stop working.
  stop_server
  start_server "second-secret-entirely-different" /smoke/server2.log
  if wait_for_server; then
    pass "restarted under a different SECRET"
  else
    fail "server did not restart"
    cat /smoke/server2.log
    return 1
  fi

  check "old token against the new secret" \
    "$(http_code -H "${HDR_VERSION}" -H "Authorization: Bearer ${TOKEN}" "${API}/user/profile")" "401"

  curl -sS -o /tmp/login2.json -X POST -H "${HDR_VERSION}" \
       -H 'Content-Type: application/json' -d "${CREDS}" "${API}/auth/login" >/dev/null 2>&1
  TOKEN2=$(sed -n 's/.*"token":"\([^"]*\)".*/\1/p' /tmp/login2.json)
  check "freshly issued token against the new secret" \
    "$(http_code -H "${HDR_VERSION}" -H "Authorization: Bearer ${TOKEN2}" "${API}/user/profile")" "200"

  stop_server

  section "5. Missing configuration aborts startup"
  # timeout turns a would-be hang into a loud failure: if the binary ever starts
  # when it should not, we get exit 124 here instead of a wedged test run.
  out=$(timeout 30 env -u DB_FILENAME -u REC_PATH -u DATA_DIR -u DATA_DISK -u NET_ADAPTER -u SECRET mediasink 2>&1)
  check "exit code with no configuration" "$?" "1"
  echo "  message: ${out}"
  if grep -q "DB_FILENAME, REC_PATH, DATA_DIR, DATA_DISK, NET_ADAPTER, SECRET" <<<"${out}"; then
    pass "every missing variable named in one message"
  else
    fail "aggregated message missing or incomplete"
  fi

  # -u SECRET is load-bearing: without it an inherited SECRET would satisfy the
  # config and the server would start and never return.
  out=$(timeout 30 env -u SECRET DB_FILENAME=/smoke/x.db REC_PATH=/smoke/rec DATA_DIR=.previews \
            DATA_DISK=/ NET_ADAPTER=eth0 mediasink 2>&1)
  check "exit code with only SECRET missing" "$?" "1"
  # Not anchored with $: the message is wrapped in logrus' msg="..." field, so the
  # line ends with a quote rather than with SECRET. Assert positively that SECRET is
  # named and negatively that none of the variables that ARE set leaked in.
  if grep -q "missing required environment variables: SECRET" <<<"${out}" \
     && ! grep -qE "DB_FILENAME|REC_PATH|DATA_DIR|DATA_DISK|NET_ADAPTER" <<<"${out}"; then
    pass "only the missing variable is named"
  else
    fail "expected SECRET alone, got: ${out}"
  fi

  section "RESULT"
  if [[ ${failures} -eq 0 ]]; then
    echo "ALL CHECKS PASSED"
    return 0
  fi
  echo "${failures} CHECK(S) FAILED"
  return 1
}

case "${1:-}" in
  --in-container) run_in_container ;;
  --rebuild)      run_on_host 1 ;;
  "")             run_on_host 0 ;;
  *)
    echo "usage: $0 [--rebuild]" >&2
    exit 2
    ;;
esac
