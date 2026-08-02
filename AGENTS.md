# AGENTS.md

This file provides guidance to coding agents working with code in this repository.

JavaScript package-manager rule: use `npm` only in this repository. Do not introduce or recommend `pnpm`.

# Agent Instructions

- Never use subagents (Agent tool) or worktrees for any task in this project. Do the work directly using Read, Edit, Write, Grep, Glob, and Bash tools.
- Never run `git add`, `git commit`, `git push`, or any other git command that changes repository state on your own initiative — only when I explicitly ask for it in that moment. An approved plan containing a "commit" step is not a request to commit: stop at the working tree and report what is ready. Read-only git commands (`status`, `diff`, `log`, `show`) are fine at any time.
- No branching without permission by the user 

## Overview

MediaSink.Go is a Go-based web server for video management, stream recording, and editing. It provides:
- Automated stream recording from various sources
- Versioned REST API for video editing and media management, served publicly at `/api/v2`
- SQLite/sqlite-vec-backed persistence in the current runtime
- WebSocket support for real-time updates
- Swagger/OpenAPI documentation
- Integrated Vue 3 frontend served directly from the Go binary via `go:embed`
- Standalone Rust terminal client under `cli/` for terminal-first access to the same server

## Project Structure

- **server/**: Go backend module root
  - **main.go**: Thin bootstrap that delegates process setup and shutdown to `app.InitializeApp(...)` and `App.Run()`
  - **frontend_embed.go**: Embeds `server/frontend/dist` into the Go binary at compile time via `go:embed`
  - **app/**: Composition root and lifecycle management for startup validation, DB/vector-store init, and graceful shutdown
  - **config/**: Configuration — reads exclusively from environment variables (`config.Read()` is cached via `sync.Once`)
  - **docs/**: Generated Swagger/OpenAPI output; `server/docs/swagger.json` is the source of truth for generated clients
  - **docker-entrypoint.sh**: Server-container entrypoint used by the root Dockerfile
  - **internal/api/**: Active public HTTP layer
  - **internal/api/v1/**: Legacy handler implementations still mounted under the public `/api/v2` routes
  - **internal/api/router.go**: Route setup and middleware configuration for the shipped server
  - **internal/api/frontend.go**: Serves embedded frontend — `/env.js`, `/build.js`, and SPA catch-all
- **server/internal/http/v2/**: Refactored v2 handler slice and dependencies; currently present in the repo but not wired as the active public router. It is one part of an **unfinished migration** that also covers `internal/http/middleware/`, `internal/service/*` and `internal/store/relational/`. The whole stack hangs off `initializeV2Dependencies()` in `app/wire_gen.go`, which nothing calls — no request reaches any of it. Treat it as work in progress, not as live code, and do not delete it during cleanup passes.
- **server/internal/services/**: Business logic for core features
  - `recording_service.go`: Video information updates and metadata
  - `recorder_service.go`: Recording orchestration and lifecycle
  - `channel_service.go`: Channel management
  - `job_service.go`: Background job processing (video enhancement, previews, merges)
  - `streaming_service.go`: Stream capture logic
  - `startup_service.go`: Application startup/recovery procedures
- **server/internal/db/**: Data access layer using GORM ORM
  - Models for: channels, recordings, users, jobs, tags, settings
  - Database initialization and connection handling
- **server/internal/store/**: Newer relational/vector store abstractions used by the refactored app slice
- **server/internal/models/**: Data structures
  - **internal/models/requests/**: Request DTOs with validation tags
  - **internal/models/responses/**: Response DTOs
- **server/internal/analysis/**: Video analysis pipeline components
  - **internal/analysis/detectors/**: ONNX-based scene/highlight detectors
  - **internal/analysis/threshold/**: Adaptive threshold strategies
  - **internal/analysis/smoothing/**: Similarity smoothing methods
- **server/internal/middleware/**: HTTP middleware (authentication, authorization)
- **server/internal/jobs/**: Background job executor and worker pool
  - **internal/jobs/handlers/**: Per-job-type handler implementations
- **server/internal/util/**: Utility functions (FFmpeg cmd, video probing, string/sys helpers)
- **server/internal/ws/**: WebSocket event types and broadcasting
- **server/internal/app/**: HTTP-layer helpers (request validation, response formatting, error shapes)
- **frontend/**: Vue 3 TypeScript frontend (source lives here)
  - Built with Vite + npm; output goes to `frontend/dist/`
  - Root `frontend/dist/` is a build artifact; root build scripts mirror it into `server/frontend/dist/` before `go build`
  - Layout/styling source of truth is `frontend/src/assets/custom-bootstrap.scss`
- **cli/**: standalone Rust terminal client
  - `Cargo.toml` + `src/`: native `ratatui` CLI application
  - `src/app`, `src/ui`, `src/overlays`: current high-level Rust module split
  - `bin/mediasink.mjs` + `package.json`: minimal npm wrapper for packaging and `npm start`
  - Talks to the same `/api/v2` backend and WebSocket server as the Vue frontend
  - Reads `/env.js` and `/build.js` from the server at runtime and rejects incompatible `APP_API_VERSION` values

## Building and Running

### Build
```sh
./build.sh
```
Builds frontend first (`npm install && npm run build` in `frontend/`), then generates Swagger docs and builds the Go binary as `./main`. The frontend dist is embedded into the binary via `go:embed`, so the binary is fully self-contained.
Builds frontend first (`npm install && npm run build` in `frontend/`), mirrors the build output into `server/frontend/dist`, then generates Swagger docs in `server/docs/` and builds the Go binary as `./main`.

Key build flags:
- Sets version and commit hash via `-ldflags`
- Requires `swag` tool for Swagger generation
- Sets `CGO_CFLAGS` for SQLite3 compilation compatibility
- **Frontend must be built before `go build`** — `build.sh` handles this automatically

### Run
```sh
./run.sh
```
Performs a full rebuild and starts the server on `0.0.0.0:3000` in this order:
1. `swag init` — regenerates `server/docs/swagger.json` from Go annotations (installs `swag` automatically if missing)
2. `SWAGGER_INPUT=server/docs/swagger.json node swagger.js` — generates `src/services/api/v2/MediaSinkClient.ts` from the local spec (no running server required)
3. `npm run build` — builds the Vue frontend
4. the built frontend is mirrored into `server/frontend/dist`
5. `go build` — compiles the Go binary with the embedded frontend
6. `./main` — starts the server

The `SWAGGER_INPUT` env var in `frontend/swagger.js` lets it consume a local swagger.json file instead of fetching from a live server. When omitted (e.g. running `npm run client` manually) it falls back to `http://localhost:3000/swagger/doc.json`.

### Frontend development (hot-reload)
```sh
cd frontend && npm run dev
```
Runs the Vite dev server (typically `localhost:5173`) with hot module replacement. It reads API/WebSocket URLs from `frontend/public/env.js` — copy `frontend/public/env.js.default` to `frontend/public/env.js` if it doesn't exist. The Go server must be running separately on `:3000` for API calls to work.

### CLI development
```sh
cd cli && cargo build --locked
cd cli && cargo test --locked
cd cli && npm start
```
The CLI is a separate Rust project. Do not treat it as part of the Vue frontend toolchain. It has its own `Cargo.toml`, source tree, and npm wrapper.

CLI notes:
- The npm layer is only a distribution/launcher wrapper; `/cli` should be treated as a Rust crate first.
- Use Cargo for normal development tasks unless the user specifically wants npm-package behavior.
- The CLI depends on the server exposing `/env.js` and `/build.js`.
- The CLI hard-rejects servers whose `APP_API_VERSION` is missing or does not match the client.

### Tests
```sh
./test.sh
```
Runs all Go tests in the `server/` module. Sets test environment variables for database and file paths.

CLI-specific tests:
```sh
cd cli && cargo test --locked
```

### Linting
```sh
./lint.sh
```

## Key Technologies & Dependencies

- **Gin**: HTTP web framework
- **GORM**: ORM for database operations (supports SQLite, MySQL, PostgreSQL)
- **JWT**: Authentication via `golang-jwt/jwt/v4`
- **WebSocket**: Real-time communication via `gorilla/websocket`
- **Swagger/OpenAPI**: API documentation via `swaggo`
- **Logrus**: Structured logging

## Core Architecture

### Request Flow
1. Routes defined in `internal/api/router.go` with middleware stack
2. `/env.js` and `/build.js` served dynamically by `internal/api/frontend.go` (no auth required)
3. Public `/api/v2/*` routes are currently handled by `internal/api/v1/*`, mounted under the v2 base path with JWT auth middleware
4. `/videos/*` served as static files from the recordings directory
5. `/swagger/*` serves Swagger UI
6. All other paths fall through to the SPA handler which serves `frontend/dist/index.html`
7. Services (`internal/services/*`) contain business logic
8. Database layer (`internal/db/*`) handles persistence via GORM

### Frontend Integration
The Vue 3 frontend is embedded into the Go binary at compile time using `go:embed all:frontend/dist` in `server/frontend_embed.go`. Root build scripts mirror the built root `frontend/dist` into `server/frontend/dist` before `go build`. The embedded FS is passed to `api.Setup()` which registers three frontend-related handlers:

- **`GET /env.js`**: Dynamically generated JavaScript that sets `window.APP_*` globals. Derives API and WebSocket URLs from the incoming request's `Host` header, so the binary works on any hostname without reconfiguration. Uses `wss://` automatically when the request arrived over TLS or behind a proxy that sets `X-Forwarded-Proto: https`.
- **`GET /build.js`**: Dynamically generated JavaScript that sets `window.APP_VERSION`, `window.APP_BUILD`, and `window.APP_API_VERSION` from the Go binary's ldflags values.
- **`NoRoute` catch-all**: Tries to serve the requested path from the embedded `frontend/dist`. If the file doesn't exist (i.e., it's a client-side route), serves `index.html` instead. Returns a plain-text error if the frontend was never built.

### Docker
The `Dockerfile` uses a multi-stage build:
1. `frontend_builder` (Node 22): runs `npm ci && npm run build` in `frontend/`, outputs `dist/`
2. `ffmpeg_builder`: compiles FFmpeg from source
3. `yt_dlp_builder`: downloads yt-dlp binary
4. `app_builder` (Go): copies all source, overlays the built frontend into `server/frontend/dist`, runs `swag init` in `server/`, and builds the Go binary from `server/`
5. `final` (Debian slim): copies compiled binaries and assets — no nginx needed

### Authentication
- JWT-based authentication
- `SECRET` environment variable required (checked in `server/main.go` init)
- Middleware: `middleware.CheckAuthorizationHeader`

### Services & Background Processing
- Lifecycle is coordinated by `app.App`; `server/main.go` no longer assembles the server directly.
- **Startup**: `services.StartUpJobs()` - recovery from crashes, integrity checks
- **Recording**: `services.StartRecorder()` - manages active recordings
- **Job Processing**: `services.StartJobProcessing()` - async background tasks
- All services gracefully shut down on SIGTERM/SIGINT

### Database
- Initialized inside `app.InitializeApp()` via `db.Init()`
- The shipped runtime currently initializes `internal/store/vector.SQLiteVecStore` and should be treated as SQLite/sqlite-vec-first
- The lower `internal/db` layer still contains MySQL/PostgreSQL adapter support, but that is not the primary v2 runtime path
- SQLite mode auto-registers `sqlite-vec` for vec0 virtual table support
- Uses GORM models for type safety
- Foreign key constraints enabled during migrations
- **SQLite concurrency**: WAL journal mode + `busy_timeout=10000` + pool size 8. SQLite is a single-writer database. The lower relational layer can target PostgreSQL, but the shipped runtime still assumes the SQLite/sqlite-vec analysis path.
- **frame_vectors**: SQLite-only vec0 virtual table created lazily on first analysis run. All frame vector writes happen in a single short transaction *after* ONNX inference completes (never while holding the lock during CPU-intensive work)

## Configuration

All configuration is read from environment variables — there is no config file. `config.Read()` in `server/config/config.go` reads the variables once and caches the result.

Required environment variables:
- `SECRET`: JWT secret (required, checked at startup)
- `DB_FILENAME`: SQLite database file path (required when using SQLite)
- `REC_PATH`: Absolute path to the recordings directory (must exist)
- `DATA_DIR`: Directory for preview/thumbnail data (e.g. `.previews`)
- `DATA_DISK`: Disk mount path used for storage status queries (must exist)
- `NET_ADAPTER`: Network interface name for bandwidth monitoring (e.g. `eth0`)

Optional / database-specific:
- `DB_ADAPTER`: Current runtime should be considered `sqlite`; alternative relational adapters are not the primary supported v2 path
- `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`: Credentials for non-SQLite relational adapters if you are working on that lower-layer support

ONNX runtime:
- `ONNXRUNTIME_LIB`: Path to `libonnxruntime.so`. Auto-detected by `run.sh` from common local paths. Minimum version **1.26** — `yalue/onnxruntime_go v1.31.0` requests ORT API 26 and aborts startup against anything older; the Dockerfile and `./server/install-onnxruntime.sh` both ship **1.28.0**. Install via `./server/install-onnxruntime.sh`, which replaces a stale copy when `ONNX_VERSION` changes. The current startup path hard-requires ONNX initialization and the Mobilenet model.

Video analysis model:
- `assets/models/mobilenetv4_conv_large.onnx` — MobileNetV4-Large feature extractor (`timm/mobilenetv4_conv_large.e500_r256_in1k`, classifier head removed), **256×256** input, **1280**-d embedding. Committed via **Git LFS** (~120 MB) — a clone without LFS leaves a pointer file and the Docker build fails fast on it; regenerate with **`./server/install-model.sh`** (builds a throwaway virtualenv, exports via `scripts/export_mobilenetv4_onnx.py`, tears it down). `assets/models/mobilenetv4_conv_large.json` records provenance and the sha256.
- The active model name lives in one place: `onnx.DefaultModelName`.

Default detector configuration:
- Scene detector: `onnx_mobilenetv4_conv_large`
- Highlight detector: `onnx_mobilenetv4_conv_large`

ONNX tensor format: **NCHW** `(1, 3, H, W)` — `ImageToTensorNCHW()` in `internal/analysis/preprocessing/conversion.go`. It emits pixels in `[0,1]`; ImageNet mean/std normalization is baked into the exported graph, not applied in Go.

Changing the embedding model invalidates every stored vector — the dimension is unchanged, so nothing would fail loudly. `dropStaleFrameVectors()` (`internal/db/db.go`) compares the active model against the `embedding_model` setting and drops `frame_vectors` on a mismatch; the startup backfill in `enqueueUnanalyzedRecordings()` then re-queues analysis for every recording, and only afterwards is the new model name persisted.

## File System Requirements

The application expects these directories to exist:
- `/disk`: For disk usage/status queries
- `/recordings`: Where video recordings are stored

## API Documentation

Swagger documentation available at `/swagger/index.html` after starting the server. API base path is `/api/v2`.

Main endpoint groups:
- `/auth`: User signup, login, logout
- `/user`: User profile
- `/admin`: Version info, import triggers
- `/channels`: Channel CRUD and streaming control
- `/videos`: Video metadata, editing, and enhancement
- `/jobs`: Background job management
- `/recorder`: Recording status and control
- `/analysis`: analysis jobs/results plus visual similarity endpoints
  - `POST /analysis/all`: enqueue analysis jobs for every recording not yet analyzed
  - `POST /analysis/{id}`: enqueue analysis job for a single recording
  - `GET /analysis/{id}`: get analysis result (scenes + highlights)
  - `POST /analysis/search/image`: image-to-video similarity search (multipart upload)
  - `POST /analysis/group`: threshold-based grouping of visually similar recordings

## Video Enhancement Feature

The application supports high-quality video enhancement with the following capabilities:

### Enhancement Parameters
- **Denoising**: hqdn3d filter with configurable strength (1.0-10.0, recommended 3.0-5.0)
- **Sharpening**: unsharp filter with configurable amount (0.0-2.0, recommended 1.0-1.5)
- **Upscaling**: Lanczos scaling to target resolution (720p, 1080p, 1440p, 4K)
- **Normalization**: Optional color/brightness correction
- **Quality Control**: CRF values (15-28, lower = higher quality)
- **Encoding Presets**: 7 presets from veryfast to veryslow

### Encoding
- **Codec**: H.265/HEVC (libx265)
- **Format**: yuv420p (4:2:0 chroma subsampling for browser compatibility)
- **Audio**: Passthrough without re-encoding
- **Container**: MP4 with faststart flag

### Endpoints
- `POST /videos/{id}/enhance`: Enhance a video with specified parameters
- `POST /videos/{id}/estimate-enhancement`: Real-time file size estimation
- `GET /videos/enhance/descriptions`: Get parameter descriptions and recommended values

### File Size Estimation
Calculates estimated output size based on:
- Resolution scaling factor
- CRF-based empirical compression ratios (0.38-0.80)
- Input file size

### Request Validation
All enhancement requests are validated using struct tags:
- `EnhanceRequest`: Recording ID, resolution, denoise/sharpen strength, preset, optional CRF
- `EstimateEnhancementRequest`: Same parameters minus recording ID
- Validation enforces allowed resolutions, presets, and value ranges

## Video Merging

The application supports merging multiple videos with optional re-encoding:

### Standard Merge
- Concatenates videos without re-encoding (using codec copy)
- Fast operation, preserves quality

### Re-encoded Merge
- Re-encodes all videos to highest quality spec across all inputs
- Calculates maximum resolution, FPS, and bitrate
- Uses H.265 encoding with CRF 18 (high quality)
- Pads videos to uniform aspect ratio before merging
- Endpoint: `POST /channels/{id}/merge` with `MergeRequest`

### Validation
- Requires minimum 2 recordings
- Validates all recording IDs are positive

## Request Validation

The application uses `go-playground/validator` for declarative request validation:

### Validation Method
- Struct tags define validation rules: `validate:"required,min=1.0,max=10.0"`
- `ValidateRequest()` method in `app/request.go` validates requests
- Returns formatted error messages with field names and validation failures
- HTTP 400 response on validation failure

### Request Models
- `EnhanceRequest`: Video enhancement parameters
- `EstimateEnhancementRequest`: File size estimation parameters
- `MergeRequest`: Video merging parameters

## WebSocket Improvements

### Concurrent Write Fix
- Added mutex protection in ping goroutine to prevent concurrent websocket writes
- Ensures broadcast messages and ping messages don't clash
- Prevents "concurrent write to websocket connection" panic

### Event Broadcasting
- `BroadCastClients()` sends real-time updates to connected clients
- Events: job creation, progress, completion, errors, channel online/offline

## Concurrency & Cleanup

The application uses goroutines for:
- HTTP server operation
- Background job processing (enhancement, merging, previews)
- Stream recording and checking
- Graceful shutdown with service cleanup
- WebSocket ping heartbeat (with mutex protection)

Signal handlers (SIGTERM, SIGINT) trigger orderly shutdown of services before exit.

## FFmpeg Integration

The application uses FFmpeg for video processing:

### Video Enhancement
- **Filter Chain**: hqdn3d (denoise) → scale (upscale) → unsharp (sharpen) → format (convert to yuv420p)
- **Parameters**: Dynamic based on user input
- **Audio**: Copied without re-encoding

### Video Merging (Re-encoded)
- **Filter Chain**: scale + pad (normalize dimensions) → format (yuv420p)
- **FPS**: Adjusted to match highest input FPS
- **Bitrate**: CRF 18 (fixed quality)

### Error Handling
- Non-critical FFmpeg info/warnings filtered from logs
- Actual errors still logged and broadcasted to client
- Failed output files cleaned up on error

### Process Cancellation (`util.ErrInterrupted`)

`util.ExecSync` registers every process it starts so `util.Interrupt(pid)` can signal it by PID
(this is what `POST /jobs/stop/{pid}` and the `job.Pid` paths use). When a process is stopped that
way, `ExecSync` returns an error wrapping **`util.ErrInterrupted`**.

Because the point above — *failed output files cleaned up on error* — is applied by most callers,
**any caller that destroys data on error must first check `errors.Is(err, util.ErrInterrupted)`**.
A deliberate stop says nothing about whether the input was good. `services.fixOrphanedFiles` is
the cautionary case: it deletes a recording (file *and* database row) when `util.CheckVideo`
fails, so it routes the decision through `isCorruptionEvidence()`, which excludes interruption.

Two rules follow for anyone touching this path:

- Propagate the error identity. Wrapping with `%w` is fine; `%v` silently breaks `errors.Is` and
  turns a stopped check into a deletion.
- A process that was signalled but still exited **0** completed its work and is reported as
  success, not as an interruption — otherwise callers would discard a finished result.

## Performance Considerations

- **Resolution**: Limiting to 4 standard resolutions (720p, 1080p, 1440p, 4K)
- **Encoding**: Presets allow speed/quality tradeoff
- **CRF Values**: 15-28 range provides fine control
- **Audio Passthrough**: Avoids audio re-encoding overhead
- **Async Processing**: Long-running jobs run in background with progress updates

---

## Frontend

Source lives in `frontend/`. Built with Vite + npm; output goes to `frontend/dist/` which is embedded into the Go binary at compile time.

### Core Stack

- **Framework**: Vue 3 (Composition API with `<script setup>`)
- **State Management**: Pinia with persistedstate plugin
- **Routing**: Vue Router with auth guards
- **API Client**: Auto-generated from Swagger (`src/services/api/v2/MediaSinkClient.ts`)
- **Real-time**: Custom WebSocket manager (`src/utils/socket.ts`)
- **Styling**: Bootstrap 5 + SCSS variables
- **Internationalization**: Vue i18n
- **Testing**: Vitest (unit)
- **PWA**: Vite PWA plugin with Workbox

### Directory Structure

```
src/
├── components/       # Reusable Vue components
│   ├── modals/       # Modal dialogs
│   ├── channels/     # Channel-specific components
│   ├── charts/       # CPU/Traffic chart components
│   ├── controls/     # Control buttons and menus
│   └── navs/         # Navigation components
├── views/            # Page-level components (routed)
├── router/           # Vue Router configuration and auth guards
├── stores/           # Pinia stores (auth, channel, job, settings, toast)
├── services/
│   └── api/v2/       # Auto-generated Swagger API client
├── composables/      # Vue 3 composables (socket, app config, download)
├── layouts/          # Layout wrappers (AuthLayout, DefaultLayout, FullscreenLayout)
├── utils/            # Utility functions (socket, error handling, datetime, etc.)
├── types/            # TypeScript type definitions
├── locales/          # Translation files (en/*)
├── assets/           # Static assets and global styles
├── main.ts           # Vue app entry point
└── App.vue           # Root component
```

### Key Files

- `src/services/api/v2/ClientFactory.ts` — API client factory with auth and server-error handling
- `src/utils/serverError.ts` — detects network errors, logs out user, shows toast
- `src/utils/validator.ts` — custom form validation framework
- `src/stores/auth.ts` — authentication store
- `src/composables/useSocket.ts` — WebSocket singleton composable
- `src/layouts/DefaultLayout.vue` — main layout; registers/unregisters socket event handlers
- `src/components/DataTable.vue` — sortable/searchable table with localStorage persistence
- `src/components/VideoStripe.vue` — video timeline with frames, analysis overlays, selection markers
- `src/components/VideoEnhancementModal.vue` — self-contained enhancement modal
- `src/views/VideoView.vue` — video player and editor with analysis controls
- `src/router/index.ts` — route definitions and auth guards
- `vite.config.ts` — build and plugin configuration

### Key Architectural Patterns

**API client** — auto-generated from Swagger, accessed via factory:
```typescript
import { createClient } from "@/services/api/v2/ClientFactory";
const client = createClient(); // auto-authenticated from auth store
const data = await client.channels.list();
```
401 responses trigger automatic logout and redirect to `/login`. Server unreachability is detected in `ClientFactory.ts` and handled via `handleServerUnreachable()`.

**State management** — two Pinia patterns used interchangeably:
- Setup Store (function-based): `auth.ts`, `settings.ts`
- Options Store (object-based): `channel.ts`, `job.ts`, `toast.ts`

State persists to localStorage via `pinia-plugin-persistedstate`. All mutations must go through actions, never directly from components.

**Real-time** — singleton `SocketManager` via `useSocket()` composable:
```typescript
const { on, off } = useSocket();
on<DbJob>("JOB_UPDATE", (data) => { /* handle */ });
```
Listeners are shared across the app and not auto-cleaned on component unmount.

**Routing** — `router.beforeEach` protects all routes; unauthenticated users are redirected to `/login`. Route meta carries `layout` (`auth`, `default`, `fullscreen`) and `title`.

**Styling** — scoped SCSS with Bootstrap 5 variables. Import pattern:
```scss
@use "@/assets/custom-bootstrap.scss" as bootstrap;
```
Light/dark theme via `[data-bs-theme="light/dark"]` selectors.

Important frontend layout constraints:
- Bootstrap is customized in `frontend/src/assets/custom-bootstrap.scss`; do not assume stock Bootstrap defaults.
- The grid uses **24 columns** (`$grid-columns: 24`), so widths must use 24-column math such as `col-24`, `col-md-12`, `col-xl-8`, etc. A `col-12` spans only half the row in this repo.
- `.card-body` padding is globally overridden to `0 !important`; whenever a card needs inner spacing, add explicit padding classes such as `p-3` or `p-lg-4` on the specific `card-body`.
- When debugging layout problems, inspect `DefaultLayout.vue` and `custom-bootstrap.scss` before assuming the issue is local to the view.

**TypeScript** — ESLint enforces double quotes. Always use `<script setup lang="ts">`. Define props with `defineProps<T>()` and emits with `defineEmits<T>()`.

**Translations** — files in `src/locales/en/`. Use `{{ t("key.path") }}` in templates.

### Common Tasks

**Add a page**: create `.vue` in `src/views/`, add route in `src/router/index.ts` with `meta: { layout, title }`.

**Add a store**: create file in `src/stores/`, use Setup or Options Store syntax, add `{ persist: true }` if localStorage persistence is needed.

**Regenerate API client** (after Go swagger annotations change):
```sh
cd frontend && npm run client
# or, from the repo root, run.sh does this automatically
```

### CLI notes

- The CLI source of truth is Rust under `cli/src`, not TypeScript.
- The npm layer in `/cli` is intentionally thin and should stay secondary to the Rust project.
- The CLI validates the server `APP_API_VERSION` during login/startup and rejects incompatible servers instead of trying to run against mismatched payloads.

**Run tests**:
```sh
cd frontend
npm run test:unit   # Vitest unit tests
```

### Video Analysis UI

- `VideoStripe.vue` renders a scrollable frame timeline with overlaid scene boundaries (colored vertical lines) and motion highlights (colored bars), both positioned by timestamp and scaled with zoom.
- `VideoView.vue` provides mode toggle (Highlights / Scenes), prev/next navigation, and a dropdown index selector.
- Analysis data comes from `GET /api/v2/analysis/{recordingId}`; response includes `scenes[]` and `highlights[]` with timing and intensity. Status: `null` (not analyzed) → `pending` → `processing` → `completed`.
