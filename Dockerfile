# syntax=docker/dockerfile:1
#
# XiaoWork Nav — single-container build.
#   Stage 1 builds the Vue frontend.
#   Stage 2 builds the Go binary with the frontend embedded (go:embed).
#   Stage 3 is a minimal, non-root runtime image.

# ---------- Stage 1: frontend ----------
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci --no-audit --no-fund; else npm install --no-audit --no-fund; fi
COPY frontend/ ./
RUN npm run build

# ---------- Stage 2: backend ----------
# Keep this in sync with the `go` directive in backend/go.mod.
FROM golang:1.26-alpine AS backend

# Module proxy. Override if your network cannot reach the default proxy, e.g.
#   docker build --build-arg GOPROXY=https://goproxy.cn,direct .
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# go:embed requires internal/static/dist to exist; replace the placeholder
# with the real build output.
RUN rm -rf internal/static/dist && mkdir -p internal/static/dist
COPY --from=frontend /app/frontend/dist/ ./internal/static/dist/

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w \
        -X github.com/xiaowork-dev/xiaowork-nav/backend/internal/version.Version=${VERSION} \
        -X github.com/xiaowork-dev/xiaowork-nav/backend/internal/version.Commit=${COMMIT} \
        -X github.com/xiaowork-dev/xiaowork-nav/backend/internal/version.BuildTime=${BUILD_TIME}" \
      -o /out/xiaowork-nav ./cmd/server

# ---------- Stage 3: runtime ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
 && addgroup -S app \
 && adduser -S -G app -h /app app

COPY --from=backend /out/xiaowork-nav /usr/local/bin/xiaowork-nav
COPY deploy/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 0755 /usr/local/bin/docker-entrypoint.sh

WORKDIR /app
RUN mkdir -p /data && chown -R app:app /data /app

USER app

ENV XN_DATA_DIR=/data \
    XN_HOST=0.0.0.0 \
    XN_PORT=8080

EXPOSE 8080
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null 2>&1 || exit 1

# The entrypoint verifies that /data is writable and then runs the server.
# It stays unprivileged: the container never starts as root.
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/usr/local/bin/xiaowork-nav"]
