ARG NODE_IMAGE=node:20-alpine
ARG GO_IMAGE=golang:1.21-alpine
ARG RUNTIME_IMAGE=alpine:3.20

### Frontend build
FROM ${NODE_IMAGE} AS frontend
WORKDIR /app/frontend
RUN corepack enable
COPY frontend/pnpm-lock.yaml frontend/package.json ./
RUN pnpm install --frozen-lockfile
COPY frontend ./
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm build

### Backend build
FROM ${GO_IMAGE} AS backend
WORKDIR /app
COPY backend/go.mod ./
RUN go mod download
COPY backend ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./main.go

### Final runtime
FROM ${RUNTIME_IMAGE}
WORKDIR /app
RUN apk add --no-cache ca-certificates docker-cli
COPY --from=backend /app/server /app/server
COPY --from=frontend /app/frontend/out /app/frontend-out
ENV FRONTEND_DIST=/app/frontend-out
ENV DOWNLOAD_MIN_FREE_BYTES=2147483648
EXPOSE 8080
ENTRYPOINT ["/app/server"]
