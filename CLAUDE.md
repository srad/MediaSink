# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

MediaSink.Go is a Go-based web server for video management, stream recording, and editing. It provides:
- Automated stream recording from various sources
- REST API for video editing and media management
- SQLite/MySQL/PostgreSQL support for persistence
- WebSocket support for real-time updates
- Swagger API documentation

## Project Structure

- **main.go**: Entry point that initializes database, services, and HTTP server
- **controllers/**: HTTP request handlers organized by domain (auth, channels, videos, jobs, etc.)
  - **controllers/api/v1/**: API v1 handlers
  - **controllers/router.go**: Route setup and middleware configuration
- **services/**: Business logic for core features
  - `recording_service.go`: Video information updates and metadata
  - `recorder_service.go`: Recording orchestration and lifecycle
  - `channel_service.go`: Channel management
  - `job_service.go`: Background job processing (video enhancement, previews, merges)
  - `streaming_service.go`: Stream capture logic
  - `startup_service.go`: Application startup/recovery procedures
- **database/**: Data access layer using GORM ORM
  - Models for: channels, recordings, users, jobs, tags, settings
  - Database initialization and connection handling
- **models/**: Data structures
  - **models/requests/**: Request DTOs with validation tags
  - **models/responses/**: Response DTOs
- **middlewares/**: HTTP middleware (authentication, authorization)
- **workers/**: Background tasks and utilities (metrics collection)
- **helpers/**: Utility functions
- **patterns/**: Pattern matching and stream detection logic
- **conf/**: Configuration file (app.yml) and config loading
- **internal/config/**: Configuration structures

## Building and Running

### Build
```sh
./build.sh
```
This generates API documentation via Swagger, sets up Go modules, and builds the binary as `./main`.

Key build flags:
- Sets version and commit hash via `-ldflags`
- Requires `swag` tool for Swagger generation
- Sets `CGO_CFLAGS` for SQLite3 compilation compatibility

### Run
```sh
./run.sh
```
Builds and starts the server on `0.0.0.0:3000`.

### Tests
```sh
./test.sh
```
Runs all tests in the project. Sets test environment variables for database and file paths.

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
- **Viper**: Configuration management

## Core Architecture

### Request Flow
1. Routes defined in `controllers/router.go` with middleware stack
2. Handlers in `controllers/api/v1/*` process requests
3. Services (`services/*`) contain business logic
4. Database layer (`database/*`) handles persistence via GORM

### Authentication
- JWT-based authentication
- `SECRET` environment variable required (checked in `main.go` init)
- Middleware: `middlewares.CheckAuthorizationHeader`

### Services & Background Processing
- **Startup**: `services.StartUpJobs()` - recovery from crashes, integrity checks
- **Recording**: `services.StartRecorder()` - manages active recordings
- **Job Processing**: `services.StartJobProcessing()` - async background tasks
- All services gracefully shut down on SIGTERM/SIGINT

### Database
- Initialized via `database.Init()` in main
- Supports SQLite (default), MySQL, or PostgreSQL via `DB_ADAPTER` env var
- Uses GORM models for type safety
- Foreign key constraints disabled during migrations

## Configuration

Located in `conf/app.yml`. Key environment variables:
- `SECRET`: JWT secret (required)
- `DB_ADAPTER`: Database type (mysql/postgres, default: sqlite)
- `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`: Database credentials
- `DB_FILENAME`: SQLite database file path (if using SQLite)
- `REC_PATH`: Recordings directory (must exist, defaults to `/recordings`)
- `DATA_DISK`: Disk root for storage status queries (must exist, defaults to `/disk`)

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
