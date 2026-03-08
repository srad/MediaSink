# AGENTS.md

This file provides guidance to coding agents working with code in this repository.

## Overview

MediaSink.Go is a Go-based web server for video management, stream recording, and editing. It provides:
- Automated stream recording from various sources
- REST API for video editing and media management
- SQLite/MySQL/PostgreSQL support for persistence
- WebSocket support for real-time updates
- Swagger API documentation
- Integrated Vue 3 frontend served directly from the Go binary via `go:embed`

## Project Structure

- **main.go**: Entry point that initializes database, services, and HTTP server
- **frontend_embed.go**: Embeds `frontend/dist` into the Go binary at compile time via `go:embed`
- **config/**: Configuration — reads exclusively from environment variables (`config.Read()` is cached via `sync.Once`)
- **internal/api/**: HTTP request handlers organized by domain (auth, channels, videos, jobs, etc.)
  - **internal/api/v1/**: API v1 handlers (11 handler files)
  - **internal/api/router.go**: Route setup and middleware configuration
  - **internal/api/frontend.go**: Serves embedded frontend — `/env.js`, `/build.js`, and SPA catch-all
- **internal/services/**: Business logic for core features
  - `recording_service.go`: Video information updates and metadata
  - `recorder_service.go`: Recording orchestration and lifecycle
  - `channel_service.go`: Channel management
  - `job_service.go`: Background job processing (video enhancement, previews, merges)
  - `streaming_service.go`: Stream capture logic
  - `startup_service.go`: Application startup/recovery procedures
- **internal/db/**: Data access layer using GORM ORM
  - Models for: channels, recordings, users, jobs, tags, settings
  - Database initialization and connection handling
- **internal/models/**: Data structures
  - **internal/models/requests/**: Request DTOs with validation tags
  - **internal/models/responses/**: Response DTOs
- **internal/analysis/**: Video analysis pipeline components
  - **internal/analysis/detectors/**: Scene/highlight detectors (SSIM, frame-diff, ONNX)
  - **internal/analysis/threshold/**: Adaptive threshold strategies
  - **internal/analysis/smoothing/**: Similarity smoothing methods
- **internal/middleware/**: HTTP middleware (authentication, authorization)
- **internal/jobs/**: Background job executor and worker pool; absorbs metrics helpers
  - **internal/jobs/handlers/**: Per-job-type handler implementations
- **internal/util/**: Utility functions (FFmpeg cmd, video probing, string/sys helpers)
- **internal/ws/**: WebSocket event types and broadcasting
- **internal/app/**: HTTP-layer helpers (request validation, response formatting, error shapes)
- **frontend/**: Vue 3 TypeScript frontend (source lives here; see `frontend/CLAUDE.md` for details)
  - Built with Vite + npm; output goes to `frontend/dist/`
  - `frontend/dist/` is gitignored (build artifact) except for `frontend/dist/.gitkeep`

## Building and Running

### Build
```sh
./build.sh
```
Builds frontend first (`npm install && npm run build` in `frontend/`), then generates Swagger docs and builds the Go binary as `./main`. The frontend dist is embedded into the binary via `go:embed`, so the binary is fully self-contained.

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
1. `swag init` — regenerates `docs/swagger.json` from Go annotations (installs `swag` automatically if missing)
2. `SWAGGER_INPUT=docs/swagger.json node swagger.js` — generates `src/services/api/v1/MediaSinkClient.ts` from the local spec (no running server required)
3. `npm run build` — builds the Vue frontend
4. `go build` — compiles the Go binary with the embedded frontend
5. `./main` — starts the server

The `SWAGGER_INPUT` env var in `frontend/swagger.js` lets it consume a local swagger.json file instead of fetching from a live server. When omitted (e.g. running `pnpm client` manually) it falls back to `http://localhost:3000/swagger/doc.json`.

### Frontend development (hot-reload)
```sh
cd frontend && npm run dev
```
Runs the Vite dev server (typically `localhost:5173`) with hot module replacement. It reads API/WebSocket URLs from `frontend/public/env.js` — copy `frontend/public/env.js.default` to `frontend/public/env.js` if it doesn't exist. The Go server must be running separately on `:3000` for API calls to work.

### Tests
```sh
./test.sh
```
Runs all Go tests in the project. Sets test environment variables for database and file paths.

### Linting
```sh
./lint.sh
```

## Key Technologies & Dependencies

- **Gin**: HTTP web framework
- **GORM**: ORM for database operations (supports SQLite, MySQL, PostgreSQL)
- **JWT**: Authentication via `golang-jwt/jwt/v4`
- **WebSocket**: Real-time communication via `gorilla/websocket`
- **Swagger**: API documentation via `swaggo`
- **Logrus**: Structured logging

## Core Architecture

### Request Flow
1. Routes defined in `internal/api/router.go` with middleware stack
2. `/env.js` and `/build.js` served dynamically by `internal/api/frontend.go` (no auth required)
3. `/api/v1/*` routes handled by `internal/api/v1/*` with JWT auth middleware
4. `/videos/*` served as static files from the recordings directory
5. `/swagger/*` serves Swagger UI
6. All other paths fall through to the SPA handler which serves `frontend/dist/index.html`
7. Services (`internal/services/*`) contain business logic
8. Database layer (`internal/db/*`) handles persistence via GORM

### Frontend Integration
The Vue 3 frontend is embedded into the Go binary at compile time using `go:embed all:frontend/dist` in `frontend_embed.go`. The embedded FS is passed to `api.Setup()` which registers three frontend-related handlers:

- **`GET /env.js`**: Dynamically generated JavaScript that sets `window.APP_*` globals. Derives API and WebSocket URLs from the incoming request's `Host` header, so the binary works on any hostname without reconfiguration. Uses `wss://` automatically when the request arrived over TLS or behind a proxy that sets `X-Forwarded-Proto: https`.
- **`GET /build.js`**: Dynamically generated JavaScript that sets `window.APP_VERSION`, `window.APP_BUILD`, and `window.APP_API_VERSION` from the Go binary's ldflags values.
- **`NoRoute` catch-all**: Tries to serve the requested path from the embedded `frontend/dist`. If the file doesn't exist (i.e., it's a client-side route), serves `index.html` instead. Returns a plain-text error if the frontend was never built.

### Docker
The `Dockerfile` uses a multi-stage build:
1. `frontend_builder` (Node 22): runs `npm install && npm run build` in `frontend/`, outputs `dist/`
2. `ffmpeg_builder`: compiles FFmpeg from source
3. `yt_dlp_builder`: downloads yt-dlp binary
4. `app_builder` (Go): copies all source, overlays `frontend/dist` from `frontend_builder`, runs `swag init` and `go build`
5. `final` (Debian slim): copies compiled binaries and assets — no nginx needed

### Authentication
- JWT-based authentication
- `SECRET` environment variable required (checked in `main.go` init)
- Middleware: `middleware.CheckAuthorizationHeader`

### Services & Background Processing
- **Startup**: `services.StartUpJobs()` - recovery from crashes, integrity checks
- **Recording**: `services.StartRecorder()` - manages active recordings
- **Job Processing**: `services.StartJobProcessing()` - async background tasks
- All services gracefully shut down on SIGTERM/SIGINT

### Database
- Initialized via `db.Init()` in main
- Supports SQLite (default), MySQL, or PostgreSQL via `DB_ADAPTER` env var
- SQLite mode auto-registers `sqlite-vec` for vec0 virtual table support
- Uses GORM models for type safety
- Foreign key constraints enabled during migrations
- **SQLite concurrency**: WAL journal mode + `busy_timeout=10000` + pool size 8. SQLite is a single-writer database — for production multi-user workloads use PostgreSQL (`DB_ADAPTER=postgres`)
- **frame_vectors**: SQLite-only vec0 virtual table created lazily on first analysis run. All frame vector writes happen in a single short transaction *after* ONNX inference completes (never while holding the lock during CPU-intensive work)

## Configuration

All configuration is read from environment variables — there is no config file. `config.Read()` in `config/config.go` reads the variables once and caches the result.

Required environment variables:
- `SECRET`: JWT secret (required, checked at startup)
- `DB_FILENAME`: SQLite database file path (required when using SQLite)
- `REC_PATH`: Absolute path to the recordings directory (must exist)
- `DATA_DIR`: Directory for preview/thumbnail data (e.g. `.previews`)
- `DATA_DISK`: Disk mount path used for storage status queries (must exist)
- `NET_ADAPTER`: Network interface name for bandwidth monitoring (e.g. `eth0`)

Optional / database-specific:
- `DB_ADAPTER`: Database type (`mysql`, `postgres`, or omit for SQLite)
- `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`: Credentials for MySQL/PostgreSQL

ONNX runtime:
- `ONNXRUNTIME_LIB`: Path to `libonnxruntime.so`. Auto-detected by `run.sh` from common local paths; required version **1.24.1** (matches `yalue/onnxruntime_go v1.27.0`). Install via `./install-onnxruntime.sh`.

Video analysis model path:
- `assets/models/mobilenet_v3_large.onnx`

Default detector configuration (falls back to SSIM/FrameDiff when ONNX is unavailable):
- Scene detector: `onnx_mobilenet_v3_large`
- Highlight detector: `onnx_mobilenet_v3_large`

ONNX tensor format: **NCHW** `(1, 3, H, W)` — `ImageToTensorNCHW()` in `internal/analysis/preprocessing/conversion.go`.

## File System Requirements

The application expects these directories to exist:
- `/disk`: For disk usage/status queries
- `/recordings`: Where video recordings are stored

## API Documentation

Swagger documentation available at `/swagger/index.html` after starting the server. API base path is `/api/v1`.

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
- **API Client**: Auto-generated from Swagger (`src/services/api/v1/MediaSinkClient.ts`)
- **Real-time**: Custom WebSocket manager (`src/utils/socket.ts`)
- **Styling**: Bootstrap 5 + SCSS variables
- **Internationalization**: Vue i18n
- **Testing**: Vitest (unit), Nightwatch (E2E)
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
│   └── api/v1/       # Auto-generated Swagger API client
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

- `src/services/api/v1/ClientFactory.ts` — API client factory with auth and server-error handling
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
import { createClient } from "@/services/api/v1/ClientFactory";
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

**Run tests**:
```sh
cd frontend
npm run test:unit   # Vitest unit tests
npm run test:e2e    # Nightwatch E2E tests
```

### Video Analysis UI

- `VideoStripe.vue` renders a scrollable frame timeline with overlaid scene boundaries (colored vertical lines) and motion highlights (colored bars), both positioned by timestamp and scaled with zoom.
- `VideoView.vue` provides mode toggle (Highlights / Scenes), prev/next navigation, and a dropdown index selector.
- Analysis data comes from `GET /api/v1/analysis/{recordingId}`; response includes `scenes[]` and `highlights[]` with timing and intensity. Status: `null` (not analyzed) → `pending` → `processing` → `completed`.
