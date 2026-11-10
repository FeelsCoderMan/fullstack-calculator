# syntax=docker/dockerfile:1

# ---- Build: static Go binary ----
FROM golang:1.27.3-alpine AS build
WORKDIR /src

# Dependencies first, so source edits don't invalidate the module cache layer.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- Runtime: minimal image, non-root ----
# Alpine rather than distroless so busybox wget is available for the compose healthcheck.
FROM alpine:3.24

RUN addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app

COPY --from=build /out/server /usr/local/bin/server

USER 10001:10001
# Documentation only: the app listens on PORT (default 8080), set at runtime.
EXPOSE 8080

# Configuration comes from the runtime environment (compose: backend/.env).
# Logs go to stdout only; never set LOG_TO_FILE in containers.
ENTRYPOINT ["/usr/local/bin/server"]
