# Docker 镜像下载服务 / Docker Downloader

## 部署地址

服务已部署在 [docker-mirror.thinkmeta.site](http://docker-mirror.thinkmeta.site)。您可以访问该地址高效下载 Docker 镜像，即使在受限的网络环境中也能顺畅使用。

---

## 功能 Features

- 指定仓库地址：默认使用 `docker.io`，支持自定义私有/镜像仓库。
- 可选认证：支持用户名/密码登录后再拉取镜像。
- 平台选择：默认 `linux/amd64`，可选 `linux/arm64`、`linux/arm/v7` 等。
- 下载队列与并发：全局最大并发数为 3（所有镜像总计）。
- 去重下载：同一镜像+平台的并发请求只下载一次，完成后所有请求获得同一结果。
- 本地存储：镜像保存为 `.tar` 文件，前端可列表展示和下载。
- 清理策略：支持磁盘空间阈值与文件过期时间的自动清理。
- 友好前端：Next.js + shadcn UI，支持中英文，搜索与下载。

- Custom registry: defaults to `docker.io`, can use private/mirror registries.
- Optional auth: username/password login before pulling.
- Platform selection: default `linux/amd64`, supports `linux/arm64`, `linux/arm/v7`.
- Queue & concurrency: up to 3 concurrent downloads.
- Deduplication: identical image requests are coalesced into a single download.
- Local storage: images saved as `.tar` files, listed and downloadable in the UI.
- Cleanup policy: free-space threshold and max-age pruning.
- Clean UI: Next.js + shadcn UI with bilingual support, search and download.

---

## 适用场景 Scenarios

- 中国大陆网络环境拉取官方镜像不稳定或失败。
- 企业内网、离线/隔离环境需要提前下载并分发镜像。
- 需要指定自建私有仓库或镜像加速源，且偶尔需要认证。
- 批量导出镜像为 `.tar` 以便在其他环境导入。

- Pulling images from official registries is slow/unreliable in restricted networks.
- Enterprise, offline, or air-gapped environments require prefetch and distribution.
- Use custom/private registries or mirror endpoints with optional authentication.
- Export images as `.tar` for import in other environments.

---

## 架构 Architecture

- 后端：Go + Gin，用宿主机 Docker 套接字执行 `docker pull/save`，并做队列与并发控制，去重下载与清理策略。
- 前端：Next.js 静态导出并由后端静态文件服务，提供下载表单与镜像列表。

- Backend: Go + Gin, talks to host Docker via mounted socket to `pull/save`, provides queue, concurrency control, deduplication, and housekeeping.
- Frontend: Next.js static export served by backend; forms for download and a list view.

---

## 快速开始 Quick Start

依赖：
- 需要宿主机安装 Docker，并允许挂载 Docker 套接字。
- 默认监听端口为 `8080`。

Requirements:
- Docker installed on host, with the Docker socket mount allowed.
- Service listens on `8080` by default.

推荐直接使用公共镜像（持久化到当前目录的 `images/`）：

```bash
mkdir -p images
docker run --rm -t --name docker-downloader \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(pwd)/images:/app/images \
  bzyy2020/docker-downloader:latest
```

从源码构建镜像并运行（同样建议挂载本地 `images` 目录保存下载结果）：

```bash
# 在仓库根目录
docker build -t docker-downloader:latest .

# 运行（需要将宿主 Docker 套接字挂载，供镜像下载；并挂载本地 images 目录）
docker run -t --name docker-downloader \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v ./images:/app/images \
  docker-downloader:latest
```

最简运行（不挂载本地 `images` 目录，镜像 tar 保存在容器内，容器删掉即丢失）：

```bash
docker run --rm -t --name docker-downloader \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  docker-downloader:latest
```

打开浏览器访问 `http://localhost:8080`。

Open `http://localhost:8080` in your browser.

快速验证下载（示例拉取 nginx，默认 `docker.io`，平台 `linux/amd64`）：

```bash
curl -X POST http://localhost:8080/api/download \
  -H "Content-Type: application/json" \
  -d '{"image":"nginx:latest"}'
```

---

## API

- POST `/api/download`
  - 请求体 JSON：
    ```json
    {
      "image": "nginx:latest",
      "registry": "docker.io",      // 可选
      "username": "user",           // 可选
      "password": "pass",           // 可选
      "platform": "linux/amd64"     // 可选，默认 linux/amd64
    }
    ```
  - 响应：
    ```json
    {
      "message": "Download successful",
      "filename": "nginx_latest__linux_amd64.tar",
      "url": "/api/files/nginx_latest__linux_amd64.tar"
    }
    ```

- GET `/api/list?q=keyword`
  - 返回已下载的 tar 文件名列表（支持关键词模糊搜索）。

- GET `/api/files/:name`
  - 下载已保存的 tar 文件。

---

## 配置 Configuration

后端支持以下环境变量（容器内默认已设定部分值）：

- `FRONTEND_DIST`: 前端静态文件路径（默认 `/app/frontend-out`）。
- `DOWNLOAD_MIN_FREE_BYTES`: 最低可用空间阈值字节数（默认 `2147483648`，约 2GB），不足时按最旧优先清理。
- `DOWNLOAD_MAX_AGE_HOURS`: 超过该小时数未访问的 tar 会被清理（默认 `168` 小时，即 7 天）。
- `DOWNLOAD_CLEAN_INTERVAL_MINUTES`: 清理任务的周期（默认 `60` 分钟）。
- 镜像保存目录：容器内 `/app/images`，建议运行前在宿主创建并挂载，确保有写权限。

Backend environment variables (some defaults baked in the image):
- `FRONTEND_DIST`: path to frontend static files (default `/app/frontend-out`).
- `DOWNLOAD_MIN_FREE_BYTES`: minimum free space in bytes to keep (`2147483648` by default, ~2GB).
- `DOWNLOAD_MAX_AGE_HOURS`: max age in hours before stale tar files are removed (`168` hours by default).
- `DOWNLOAD_CLEAN_INTERVAL_MINUTES`: periodic cleanup interval (`60` minutes by default).
- Image output dir: `/app/images` inside the container; mount a writable host dir for persistence.

---

## 本地开发 Development

前端：
```bash
cd frontend
pnpm install
pnpm dev
# 访问 http://localhost:3000
```

后端：
```bash
# 在仓库根目录
GOOS=linux GOARCH=amd64 go run ./backend/main.go
# 监听 8080
```

联通 Docker：本地开发若需下载镜像，请确保本机 Docker 可用，并按需将 Docker 套接字挂载到运行的后端容器或直接使用宿主 Docker。

Frontend & Backend together: you can run the frontend in dev mode on port 3000 and the backend on 8080. For image downloads, ensure the backend has access to Docker (socket mount if running in a container).

---

## 安全与限制 Security & Limitations

- 需要挂载宿主机的 Docker 套接字（`/var/run/docker.sock`），请谨慎使用并在可信环境部署。
- 最大并发下载为 3（全局计数），可根据需要改造。
- 同镜像请求去重：避免重复下载，提高资源利用率。
- 镜像以 `.tar` 保存，适合离线分发与导入（`docker load -i`）。
- 认证参数会通过 `docker login -p` 传给 Docker CLI，仅用于当次拉取，不会持久化；请避免在不可信环境暴露密码，建议使用私有网络或专用凭证。

- Mounting host Docker socket grants high privileges; deploy in trusted environments only.
- Concurrency is capped at 3 globally; adjust as needed.
- Identical image requests are deduplicated.
- Images exported as `.tar` for offline distribution and import (`docker load -i`).
- Credentials are passed to `docker login -p` for the current pull only; avoid exposing secrets in untrusted environments.

---

## 贡献 Contributing

欢迎 Issue 与 PR！可补充镜像源选择、下载进度展示、更多平台与清理策略等功能。

Issues and PRs are welcome. Ideas include mirror selection, progress UI, more platforms, and housekeeping improvements.

---

## 许可 License

当前仓库未指定许可证。若要以开源方式发布，建议添加常见开源许可证（如 MIT、Apache-2.0 等）。

No license specified yet. For open-source distribution, consider adding a license (MIT, Apache-2.0, etc.).
