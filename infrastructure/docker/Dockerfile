# syntax=docker/dockerfile:1

# ---- Stage 1: frontend build -------------------------------------------------
# The frontend is arch-independent, so build it on the BUILD platform instead
# of the target platform: running node/pnpm under QEMU emulation (arm64 tag
# builds on amd64 runners) is flaky and has crashed with SIGILL during
# pnpm install. Only the Go stage and the runtime image target the platform.
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src

COPY . .
RUN corepack enable && \
    pnpm install --frozen-lockfile && \
    pnpm --filter @my/web build

# ---- Stage 2: Go binary with embedded assets ---------------------------------
FROM golang:1.26-alpine AS api
WORKDIR /src

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

# Vite output from the web stage; the go:embed of static/* requires these files
# to exist at compile time.
COPY --from=web /src/apps/web/dist apps/web/dist
COPY . .

# -p 2 bounds parallel compiles so the image builds on hosts with ~2GB of
# Docker memory (default Go parallelism OOM-killed here). CI runners have
# 16GB and are unaffected by the bound.
RUN sh scripts/copy-web-assets.sh && \
    cd apps/api && \
    CGO_ENABLED=0 go build \
      -p 2 \
      -trimpath \
      -ldflags "-s -w \
        -X github.com/jjspscl/my/internal/platform/version.Version=${VERSION} \
        -X github.com/jjspscl/my/internal/platform/version.Commit=${COMMIT} \
        -X github.com/jjspscl/my/internal/platform/version.Date=${DATE}" \
      -o /out/my ./cmd/api

# Empty dirs to seed writable mount points owned by the nonroot user.
RUN mkdir -p /tmp/data

# ---- Stage 3: minimal runtime -------------------------------------------------
# distroless/static: CA certs included, no shell/curl, runs as nonroot 65532.
# No HEALTHCHECK in-image: there is no shell to run it; probe externally.
FROM gcr.io/distroless/static:nonroot

COPY --from=api /out/my /my
# Ownership carried over so a named-volume mount is writable by the nonroot
# user (Docker initializes named volumes from the image directory ownership).
COPY --from=api --chown=65532:65532 /tmp/data /data
COPY --from=api --chown=65532:65532 /tmp/data /tmp

ENV MY_DATABASE_URL=file:/data/my.db
ENV TMPDIR=/tmp

EXPOSE 8080
VOLUME ["/data"]

# Direct ENTRYPOINT, no shell wrapper: the binary is PID 1 and receives
# SIGTERM, which drives its graceful 15s shutdown drain.
ENTRYPOINT ["/my"]
