# Docker Downloader

A simple, reliable offline Docker image downloader/distributor for network-restricted environments. Supports custom registries, optional auth, concurrency control, deduped downloads, and a clean web UI.

Chinese version: [README.md](README.md)

**Public image**: `bzyy2020/docker-downloader`

```bash
docker pull bzyy2020/docker-downloader:latest

docker run --rm -t --name docker-downloader \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/images:/app/images \
  bzyy2020/docker-downloader:latest
```

## Features

- Custom registry (defaults to `docker.io`); optional username/password.
- Platform selection (default `linux/amd64`, supports `linux/arm64`, `linux/arm/v7`).
- Global max concurrency: 3 downloads.
- Deduplication: identical image+platform requests coalesce into one job; all callers share the result.
- Images saved as `.tar` and listed/downloadable from the UI.
- Cleanup policy: free-space threshold, max-age pruning, periodic cleanup.
- Next.js + shadcn UI with language toggle and search.

## Quick Start

Prereqs:
- Docker on the host with socket mount allowed.
- Service listens on `8080` by default.

Recommended (public image, persistent downloads to `./images`):

```bash
mkdir -p images

docker run --rm -t --name docker-downloader \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/images:/app/images \
  bzyy2020/docker-downloader:latest
```

Build from source:

```bash
# repo root
docker build -t docker-downloader:latest .

docker run -t --name docker-downloader \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/images:/app/images \
  docker-downloader:latest
```

Minimal (non-persistent) run:

```bash
docker run --rm -t --name docker-downloader \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  docker-downloader:latest
```

Open http://localhost:8080.

Quick test (defaults to docker.io, linux/amd64):

```bash
curl -X POST http://localhost:8080/api/download \
  -H "Content-Type: application/json" \
  -d '{"image":"nginx:latest"}'
```

## API

- POST `/api/download`
  - Body:
    ```json
    {
      "image": "nginx:latest",
      "registry": "docker.io",
      "username": "user",
      "password": "pass",
      "platform": "linux/amd64"
    }
    ```
  - Response:
    ```json
    {
      "message": "Download successful",
      "filename": "nginx_latest__linux_amd64.tar",
      "url": "/api/files/nginx_latest__linux_amd64.tar"
    }
    ```

- GET `/api/list?q=keyword`
  - Returns downloaded tar filenames (keyword filter, case-insensitive).

- GET `/api/files/:name`
  - Downloads a saved tar.

## Configuration

Backend env vars (defaults baked into the image):
- `FRONTEND_DIST`: path to frontend static files (default `/app/frontend-out`).
- `DOWNLOAD_MIN_FREE_BYTES`: minimum free space in bytes before cleanup (`2147483648`, ~2GB).
- `DOWNLOAD_MAX_AGE_HOURS`: max age in hours before stale tars are removed (`168`, i.e., 7 days).
- `DOWNLOAD_CLEAN_INTERVAL_MINUTES`: periodic cleanup interval (`60` minutes).
- Image output dir: `/app/images` in the container; mount a writable host dir for persistence.

## Security & Limitations

- Host Docker socket (`/var/run/docker.sock`) is mounted—deploy only in trusted environments.
- Global concurrency capped at 3; deduped per image+platform.
- Images are exported as `.tar` for offline distribution/import (`docker load -i <file>.tar`).
- Credentials are passed to `docker login -p` for the current pull only; avoid exposing secrets.

## Development

Frontend:
```bash
cd frontend
pnpm install
pnpm dev
# http://localhost:3000
```

Backend:
```bash
GOOS=linux GOARCH=amd64 go run ./backend/main.go
# listens on 8080
```

Ensure Docker is available to the backend (socket mount if running in a container).

## License

No license specified yet. For open-source distribution, add one (MIT, Apache-2.0, etc.).
