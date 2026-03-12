# MediaSink

![License](https://img.shields.io/badge/license-AGPL--v3-blue)
![Go Version](https://img.shields.io/badge/Go-1.x-blue)
[![Build Status](https://teamcity.sedrad.com/app/rest/builds/buildType:(id:MediaSinkGo_Build)/statusIcon)](https://teamcity.sedrad.com/viewType.html?buildTypeId=MediaSinkGo_Build&guest=1)
![Build](https://img.shields.io/github/actions/workflow/status/srad/MediaSink/build.yml)

MediaSink is a powerful web-based video management, editing and streaming server written in Go. It provides automated stream recording capabilities and a REST API for video editing, making it an ideal solution for media-heavy applications. The Vue 3 frontend is bundled directly into the Go binary, and the repository also includes a standalone Rust terminal client under [`cli/`](./cli) for terminal-first access to the same MediaSink server.

## Features
- **Media Management**: Scans all media and generate previews and organizes them. Allows bookmarking folders, channel, media items, and tagging the media.
- **Automated Stream Recording**: Capture and store video streams automatically.
- **REST API for Video Editing**: Perform video editing tasks programmatically.
- **Video Analysis (ONNX + sqlite-vec)**: Detect scenes and highlights from preview frames using ONNX feature extraction and sqlite-vec similarity queries.
- **Integrated Web UI**: Vue 3 frontend embedded directly in the binary — served from the same port as the API, no nginx or separate deployment needed.
- **Integrated Terminal UI**: Separate Rust CLI under `cli/` with login, live workspace views, WebSocket updates, themes, forms, and popup video playback.
- **Scalable & Lightweight**: Optimized for performance with a minimal resource footprint.
- **Easy Integration**: RESTful API for seamless integration with other applications.
- **Disaster Recovery**: If the system crashes during recordings or while processing background jobs, it will recover on the next restart and check the media files for integrity.

## Installation

This is mainly for development purposes. In production you'd use the Docker image.

### Prerequisites
- Go 1.x or later
- Node.js 22+ and npm (for building the frontend)
- Rust + Cargo (only if you want to build or run `cli/`)
- FFmpeg (for video processing)
- yt-dlp
- FFprobe
- SQLite 3
- ONNX Runtime shared library (for ONNX-based video analysis)

If you run the application outside of Docker, you must manually install the above dependencies.

Debian setup:

```sh
sudo apt update && sudo apt install -y wget ffmpeg sqlite3
# Install yt-dlp
curl -SL https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux -o /usr/local/bin/yt-dlp && chmod +x /usr/local/bin/yt-dlp
```

Go setup (replace with latest version):

```sh
sudo apt update && sudo apt install -y wget
wget https://golang.org/dl/go1.23.linux-amd64.tar.gz
sudo tar -C /usr/local -xvzf go1.23.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bashrc
source ~/.bashrc
```

### Clone the Repository
```sh
git clone https://github.com/srad/MediaSink.git
cd MediaSink
```

### Build

Builds the frontend and then the Go binary (self-contained, no separate web server needed):

```sh
./build.sh
```

All configuration is provided via environment variables (see the **Run** section below). There is no config file.

For ONNX-based analysis outside Docker, set `ONNXRUNTIME_LIB` if the runtime library is not on your default linker path.

### Run

```sh
./run.sh
```

Builds the frontend if needed, then builds and starts the server on `http://0.0.0.0:3000`. The web UI is available at the same address.

`run.sh` exports sensible local-development defaults for all required variables. The only hard requirement is `SECRET` (a JWT secret); everything else has a working default.

### Frontend development (hot-reload)

```sh
cd frontend && npm run dev
```

Runs the Vite dev server (typically `http://localhost:5173`) with hot module replacement. Copy `frontend/public/env.js.default` to `frontend/public/env.js` if you haven't already, and make sure the Go server is running on `:3000`.

### CLI

The terminal client lives in `cli/` and is built separately from the Go server and Vue frontend.

Run it with npm:

```sh
cd cli && npm start
```

or directly with Cargo:

```sh
cd cli && cargo run
```

CLI build/test:

```sh
cd cli && cargo build --locked
cd cli && cargo test --locked
```

CLI notes:

- Rust-first project layout under `cli/src`
- Minimal npm wrapper only for packaging/distribution
- Full-screen `ratatui` TUI with login/registration, themes, live views, confirm dialogs, mouse support, and popup video playback
- Reads runtime settings from the target server's `/env.js` and `/build.js`
- Rejects incompatible MediaSink servers when the exposed `APP_API_VERSION` is missing or does not match

### Run Tests

```sh
go test ./...
```

CLI tests:

```sh
cd cli && cargo test --locked
```

## Usage

### Storage device file system

You might want to spend some time looking at a reasonable choice for your file system because it might have
a significant effect on the lifespan of your storage device, especially with write-heavy large files workloads.

These are the most common file systems and their characteristics in this context:

| File System | Performance  | Data Integrity | Tuning Complexity | Best Use Case                           |
|-------------|--------------|----------------|-------------------|-----------------------------------------|
| XFS         | 🚀 Very High | ❌ Basic only | 🔧 Minimal        | Streaming large files                  |
| EXT4        | ⚡ Good      | ❌ Basic only | 🔧 Minimal        | General-purpose, legacy support        |
| ZFS         | ⚖️ Medium    | ✅ Excellent  | 🔧🔧🔧 High      | When data integrity > raw speed         |
| Btrfs       | ⚡ Okay      | ✅ Good       | 🔧 Medium         | Light snapshots, lower overhead than ZFS|

If you do not require the highest amout of data integritity checking and snapshots, at the cost of your device's lifespan, then
it is highly recommended to format your storage device with the XFS filesystem, since it is optimized large write file write heavy workloads.

You can do that from the shell:

```sh
mkfs.xfs -f /dev/sdX
mount -o noatime /dev/sdX /mnt/video
```

### API Endpoints
MediaSink provides a REST API to manage video recording and editing. Below are some key endpoints:
For a complete API reference, check the [API Documentation](https://github.com/srad/MediaSink/wiki/API-Docs).

Visual similarity endpoints:
- `POST /api/v1/analysis/search/image` (multipart): upload an image (`file`) and search similar videos. Supports `similarity` slider values in `0..1` or `0..100`.
- `POST /api/v1/analysis/group` (json): group similar videos by similarity threshold. Supports optional `recordingIds`, `pairLimit`, and `includeSingletons`.

## Docker

The Docker image bundles the Go server, Vue frontend, FFmpeg, yt-dlp, and SQLite into a single container. No separate nginx, client container, or external web server is needed.

#### Docker Compose

Minimal setup — only `SECRET` and a volume for persistent storage are required:

```yaml
services:
  mediasink:
    image: sedrad/mediasink
    environment:
      - SECRET=change-me
    volumes:
      - /path/to/recordings:/recordings
    ports:
      - "3000:3000"
```

Full setup with persistent storage and timezone:

```yaml
services:
  mediasink:
    image: sedrad/mediasink
    environment:
      - TZ=${TIMEZONE}
      - SECRET=${SECRET}
    volumes:
      - ${DATA_PATH}:/recordings
      - ${DISK}:/disk
      - /etc/localtime:/etc/localtime:ro
      - /etc/timezone:/etc/timezone:ro
    ports:
      - "3000:3000"
```

compose variables (host-side only, not passed into the container):

| Variable | Example | Description |
|---|---|---|
| `TIMEZONE` | `Europe/Berlin` | Sets `TZ` inside the container |
| `SECRET` | `change-me` | JWT signing secret — **required** |
| `DATA_PATH` | `/path/to/your/recordings` | Host path mounted as `/recordings` |
| `DISK` | `/mnt/disk1` | Host path mounted as `/disk` |

Application environment variables (passed into the container, all optional):

| Variable | Required | Default | Description |
|---|---|---|---|
| `SECRET` | **yes** | — | JWT signing secret — use a long random string |
| `TZ` | no | `Europe/Berlin` | Container timezone |
| `DB_FILENAME` | no | `/recordings/mediasink.sqlite3` | SQLite database file path |
| `REC_PATH` | no | `/recordings` | Recordings directory inside the container |
| `DATA_DIR` | no | `.previews` | Preview/thumbnail cache directory |
| `DATA_DISK` | no | `/disk` | Disk mount path used for storage status queries |
| `NET_ADAPTER` | no | `eth0` | Network interface for bandwidth monitoring |
| `DB_ADAPTER` | no | `sqlite` | Database backend (`sqlite`, `mysql`, `postgres`) |
| `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` | no | — | Required only when `DB_ADAPTER` is `mysql` or `postgres` |

The web UI is available at `http://<host>:3000` and the API at `http://<host>:3000/api/v1`.

#### Deploy

```sh
docker-compose --env-file .env up -d
```

## Contributing
We welcome contributions! To get started:
1. Fork the repository.
2. Create a new branch.
3. Make your changes and commit them.
4. Submit a pull request.

## License
MediaSink is dual-licensed under the GNU Affero General Public License (AGPL) and a commercial license.

- **Open-Source Use (AGPL License)**: MediaSink is free to use, modify, and distribute under the terms of the [GNU AGPL v3](https://www.gnu.org/licenses/agpl-3.0.html). Any modifications and derivative works must also be open-sourced under the same license.
- **Commercial Use**: Companies that wish to use MediaSink without AGPL restrictions must obtain a commercial license. For more details, please refer to the [LICENSE](LICENSE) file or contact us for licensing inquiries.
MediaSink is available for free for non-profit and educational institutions. However, a commercial license is required for companies. For more details, please refer to the [LICENSE](LICENSE) file or contact us for licensing inquiries.

## Contact
For issues and feature requests, please use the [GitHub Issues](https://github.com/srad/MediaSink/issues) section.

## Notes & Limitations

1. All streaming services allow only a limited number of request made by each client.
If this limit is exceeded the client will be temporarily or permanently blocked.
In order to circumvent this issue, the application does strictly control the 
timing between each request. However, this might cause that the recording will only start
recording after a few minutes and not instantly.
2. The system has disaster recovery which means that if the system crashes during recordings,
it will try to recover all recordings on the next launch. However, due to the nature of
streaming videos and the crashing behavior, the video files might get corrupted.
In this case they will be automatically delete from the system, after they have been
checked for integrity. Otherwise, they are added to the library.


---
Star the repo if you find it useful! ⭐
