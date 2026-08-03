# Test image for the Go server.
#
# Exists because the host cannot run the full check: the server refuses to start
# without ffmpeg, ffprobe, yt-dlp and the ONNX runtime, so `go test` alone can never
# prove that the binary actually boots and serves authenticated requests.
#
# Deliberately NOT the production Dockerfile. That one compiles ffmpeg 8.1 from
# source and builds the frontend, which takes far longer than a test run is worth.
# Debian's ffmpeg package supplies both ffmpeg and ffprobe, which is all
# app.validateEnvironment looks for — it calls exec.LookPath and never runs them.
#
# Build and run through ./docker-test.sh; see that script for usage.
#
# Named test.Dockerfile, NOT Dockerfile.test: .gitignore carries `*.test` to drop
# compiled Go test binaries, and it silently swallows the latter — the file would
# never be committed and would be missing from a fresh clone. Do not rename it back.
FROM golang:1-bookworm

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y \
      ffmpeg \
      curl \
      ca-certificates \
      build-essential \
      libsqlite3-dev \
      sqlite3 \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# yt-dlp: single static binary, same source the production image uses.
RUN curl -SL "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux" \
      -o /usr/local/bin/yt-dlp && chmod a+rx /usr/local/bin/yt-dlp

# Keep in step with server/install-onnxruntime.sh and the onnx_builder stage of the
# production Dockerfile: a runtime older than the binding requests aborts at startup.
ARG ONNX_VERSION=1.28.0
RUN TARBALL="onnxruntime-linux-x64-${ONNX_VERSION}.tgz" && \
    curl -fsSL "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/${TARBALL}" -o /tmp/${TARBALL} && \
    tar -xzf /tmp/${TARBALL} -C /tmp && \
    find /tmp -name "libonnxruntime.so*" -exec cp {} /usr/local/lib/ \; && \
    ldconfig && rm -rf /tmp/*.tgz /tmp/onnxruntime-*

WORKDIR /app
COPY server /app

# Set the same way the production image sets it, so the run exercises the
# Cfg.ONNXRuntimeLib -> onnx.SetLibraryPath path rather than the system default.
ENV ONNXRUNTIME_LIB="/usr/local/lib/libonnxruntime.so"
ENV GOWORK=off

# Compile once at build time so a re-run does not pay for it again. The second
# build warms the cache for the packages `go test` will compile.
RUN go build -o /usr/local/bin/mediasink \
      -ldflags="-X 'main.Version=test' -X 'main.Commit=test' -X 'main.ApiVersion=1.0'" \
      -mod=mod . \
 && go build ./... 2>/dev/null || true

# Last, so editing the test script does not invalidate the compile layer above.
# Iterate without rebuilding at all:
#   docker run --rm -v "$PWD/docker-test.sh:/usr/local/bin/docker-test.sh:ro" mediasink-test
COPY docker-test.sh /usr/local/bin/docker-test.sh

CMD ["/usr/local/bin/docker-test.sh", "--in-container"]
