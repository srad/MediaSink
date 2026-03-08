# MediaSink.Go

![License](https://img.shields.io/badge/license-AGPL--v3-blue)
![Go Version](https://img.shields.io/badge/Go-1.x-blue)
[![Build Status](https://teamcity.sedrad.com/app/rest/builds/buildType:(id:MediaSinkGo_Build)/statusIcon)](https://teamcity.sedrad.com/viewType.html?buildTypeId=MediaSinkGo_Build&guest=1)
![Build](https://img.shields.io/github/actions/workflow/status/srad/MediaSink.Go/build.yml)

MediaSink.Go is a powerful web-based video management, editing and streaming server written in Go. It provides automated stream recording capabilities and a REST API for video editing, making it an ideal solution for media-heavy applications. The Vue 3 frontend is bundled directly into the Go binary — no separate web server required.

## Features
- **Media Management**: Scans all media and generate previews and organizes them. Allows bookmarking folders, channel, media items, and tagging the media.
- **Automated Stream Recording**: Capture and store video streams automatically.
- **REST API for Video Editing**: Perform video editing tasks programmatically.
- **Video Analysis (ONNX + sqlite-vec)**: Detect scenes and highlights from preview frames using ONNX feature extraction and sqlite-vec similarity queries.
- **Integrated Web UI**: Vue 3 frontend embedded directly in the binary — served from the same port as the API, no nginx or separate deployment needed.
- **Scalable & Lightweight**: Optimized for performance with a minimal resource footprint.
- **Easy Integration**: RESTful API for seamless integration with other applications.
- **Disaster Recovery**: If the system crashes during recordings or while processing background jobs, it will recover on the next restart and check the media files for integrity.

## Installation

This is mainly for development purposes. In production you'd use the Docker image.

### Prerequisites
- Go 1.x or later
- Node.js 22+ and npm (for building the frontend)
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
git clone https://github.com/srad/MediaSink.Go.git
cd MediaSink.Go
```

### Build

Builds the frontend and then the Go binary (self-contained, no separate web server needed):

```sh
./build.sh
```

Configuration is loaded from `conf/app.*` (e.g. `conf/app.default.yml` and `conf/app.docker.yml`) and can be overridden with environment variables.

For ONNX-based analysis outside Docker, set `ONNXRUNTIME_LIB` if the runtime library is not on your default linker path.

### Run

```sh
./run.sh
```

Builds the frontend if needed, then builds and starts the server on `http://0.0.0.0:3000`. The web UI is available at the same address.

### Frontend development (hot-reload)

```sh
cd frontend && npm run dev
```

Runs the Vite dev server (typically `http://localhost:5173`) with hot module replacement. Copy `frontend/public/env.js.default` to `frontend/public/env.js` if you haven't already, and make sure the Go server is running on `:3000`.

### Run Tests

```sh
go test ./...
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
MediaSink.Go provides a REST API to manage video recording and editing. Below are some key endpoints:
For a complete API reference, check the [API Documentation](https://github.com/srad/MediaSink.Go/wiki/API-Docs).

Visual similarity endpoints:
- `POST /api/v1/analysis/search/image` (multipart): upload an image (`file`) and search similar videos. Supports `similarity` slider values in `0..1` or `0..100`.
- `POST /api/v1/analysis/group` (json): group similar videos by similarity threshold. Supports optional `recordingIds`, `pairLimit`, and `includeSingletons`.

## Docker

The Docker image bundles the Go server, Vue frontend, FFmpeg, yt-dlp, and SQLite into a single container. No separate nginx, client container, or external web server is needed.

#### Docker Compose

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
    ports:
      - "3000:3000"
```

`.env` file:

```
TIMEZONE=Europe/Berlin

# JWT secret — required, set to a long random string
SECRET=change-me

# Path where recorded videos will be stored
DATA_PATH=/path/to/files

# Path to the disk root (used for disk status queries)
DISK=/mnt/disk1
```

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
MediaSink.Go is dual-licensed under the GNU Affero General Public License (AGPL) and a commercial license.

- **Open-Source Use (AGPL License)**: MediaSink.Go is free to use, modify, and distribute under the terms of the [GNU AGPL v3](https://www.gnu.org/licenses/agpl-3.0.html). Any modifications and derivative works must also be open-sourced under the same license.
- **Commercial Use**: Companies that wish to use MediaSink.Go without AGPL restrictions must obtain a commercial license. For more details, please refer to the [LICENSE](LICENSE) file or contact us for licensing inquiries.
MediaSink.Go is available for free for non-profit and educational institutions. However, a commercial license is required for companies. For more details, please refer to the [LICENSE](LICENSE) file or contact us for licensing inquiries.

## Contact
For issues and feature requests, please use the [GitHub Issues](https://github.com/srad/MediaSink.Go/issues) section.

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
