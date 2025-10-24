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
  - `job_service.go`: Background job processing
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
- `/videos`: Video metadata and editing
- `/jobs`: Background job management
- `/recorder`: Recording status and control

## Concurrency & Cleanup

The application uses goroutines for:
- HTTP server operation
- Background job processing
- Stream recording
- Graceful shutdown with service cleanup

Signal handlers (SIGTERM, SIGINT) trigger orderly shutdown of services before exit.
