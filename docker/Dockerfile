# syntax=docker/dockerfile:1
# ─────────────────────────────────────────────────────────────────────────────
# BestDefense Device Monitor — portable build/test image
#
# Cross-compiles binaries for Windows, macOS, and Linux from a single Linux
# container.  No local Go installation required — just Docker with BuildKit.
#
# Key design: vet/test stages run at `docker build` time (catching errors early).
# Compilation stages use `build-env` so binaries are produced at `docker run`
# time, writing to the volume-mounted dist/ directory on the host.
# ─────────────────────────────────────────────────────────────────────────────

ARG GO_VERSION=1.22

# ── stage: deps ─────────────────────────────────────────────────────────────
# Download modules into a layer that only rebuilds when go.mod/go.sum change.
FROM golang:${GO_VERSION}-alpine AS deps

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# ── stage: build-env ─────────────────────────────────────────────────────────
# Source + deps, ready to compile.  Actual go build is run at container start
# so the volume-mounted /out directory receives the produced binaries.
FROM deps AS build-env

ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

COPY . .

# Expose version args as env so docker-compose command strings can reference them.
ENV VERSION=${VERSION} GIT_COMMIT=${GIT_COMMIT} BUILD_DATE=${BUILD_DATE}
ENV CGO_ENABLED=0

# ── stage: vet ──────────────────────────────────────────────────────────────
# Run go vet with GOOS=windows (backward-compatible default).
# Runs at docker build time so the compose `run --rm vet` image is cached.
FROM build-env AS vet

ENV GOOS=windows GOARCH=amd64
RUN go vet ./...

# ── stage: vet-all ──────────────────────────────────────────────────────────
# Run go vet across all three platforms.
FROM build-env AS vet-all

RUN GOOS=windows GOARCH=amd64 go vet ./... && \
    GOOS=darwin  GOARCH=amd64 go vet ./... && \
    GOOS=linux   GOARCH=amd64 go vet ./... && \
    echo "vet: all platforms passed"

# ── stage: test ─────────────────────────────────────────────────────────────
# Run tests for platform-agnostic packages natively; cross-compile all others.
FROM build-env AS test

# Test platform-agnostic packages natively on Linux
RUN go test -v -count=1 ./internal/config/... ./internal/identity/... ./internal/reporter/... ./internal/commander/... ./internal/executor/... ./internal/taskresult/...

# Cross-compile test binaries for all platforms (catches build-tag/type errors)
RUN GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/collector/...   && \
    GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/identity/...    && \
    GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/service/...     && \
    GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/logging/...     && \
    GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/commander/...   && \
    GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/executor/...    && \
    GOOS=windows GOARCH=amd64 go test -c -o /dev/null ./internal/taskresult/...  && \
    GOOS=darwin  GOARCH=amd64 go test -c -o /dev/null ./internal/collector/...   && \
    GOOS=darwin  GOARCH=amd64 go test -c -o /dev/null ./internal/identity/...    && \
    GOOS=darwin  GOARCH=amd64 go test -c -o /dev/null ./internal/service/...     && \
    GOOS=darwin  GOARCH=amd64 go test -c -o /dev/null ./internal/logging/...     && \
    GOOS=darwin  GOARCH=amd64 go test -c -o /dev/null ./internal/commander/...   && \
    GOOS=darwin  GOARCH=amd64 go test -c -o /dev/null ./internal/executor/...    && \
    GOOS=darwin  GOARCH=amd64 go test -c -o /dev/null ./internal/taskresult/...  && \
    GOOS=linux   GOARCH=amd64 go test -c -o /dev/null ./internal/collector/...   && \
    GOOS=linux   GOARCH=amd64 go test -c -o /dev/null ./internal/identity/...    && \
    GOOS=linux   GOARCH=amd64 go test -c -o /dev/null ./internal/service/...     && \
    GOOS=linux   GOARCH=amd64 go test -c -o /dev/null ./internal/logging/...     && \
    GOOS=linux   GOARCH=amd64 go test -c -o /dev/null ./internal/commander/...   && \
    GOOS=linux   GOARCH=amd64 go test -c -o /dev/null ./internal/executor/...    && \
    GOOS=linux   GOARCH=amd64 go test -c -o /dev/null ./internal/taskresult/...  && \
    echo "All packages compiled successfully for all platforms"

# ── stage: package-base ──────────────────────────────────────────────────────
# build-env + fpm for .deb/.rpm packaging.
# The actual fpm invocations run at container start time (see docker-compose.yml).
FROM build-env AS package-base

RUN apk add --no-cache ruby ruby-dev build-base rpm tar && \
    gem install fpm --no-document
