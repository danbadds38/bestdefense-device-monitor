# syntax=docker/dockerfile:1
# ─────────────────────────────────────────────────────────────────────────────
# BestDefense Device Monitor — portable build/test image
#
# Cross-compiles a Windows .exe from Linux so CI can run on fast ubuntu runners
# instead of expensive windows runners.
# ─────────────────────────────────────────────────────────────────────────────

ARG GO_VERSION=1.22

# ── stage: deps ─────────────────────────────────────────────────────────────
# Download modules into a layer that only rebuilds when go.mod/go.sum change.
FROM golang:${GO_VERSION}-alpine AS deps

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# ── stage: vet ──────────────────────────────────────────────────────────────
# Run go vet against all packages, including Windows-tagged files.
# We don't need to run on a Windows host — go vet with GOOS=windows checks
# type-correctness and build constraints without executing any code.
FROM deps AS vet

COPY . .
ENV GOOS=windows GOARCH=amd64 CGO_ENABLED=0
RUN go vet ./...

# ── stage: test ─────────────────────────────────────────────────────────────
# Run tests for platform-agnostic packages (config, reporter).
# Windows-specific collector/service/logging packages are tested via
# cross-compilation (go test -c) which catches type errors without needing
# a Windows host to execute them.
FROM deps AS test

COPY . .

# Test platform-agnostic packages natively on Linux
RUN go test -v -count=1 ./internal/config/... ./internal/reporter/...

# Cross-compile tests for Windows-specific packages to catch compile errors.
# -c compiles a test binary but does not run it (can't run .exe on Linux).
ENV GOOS=windows GOARCH=amd64 CGO_ENABLED=0
RUN go test -c -o /dev/null ./internal/collector/... && \
    go test -c -o /dev/null ./internal/service/...   && \
    go test -c -o /dev/null ./internal/logging/...   && \
    echo "All packages compiled successfully"

# ── stage: build ─────────────────────────────────────────────────────────────
# Produce the final bestdefense-device-monitor.exe
FROM deps AS build

ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

COPY . .

ENV GOOS=windows GOARCH=amd64 CGO_ENABLED=0

RUN go build \
      -ldflags="-s -w -H windowsgui \
        -X main.Version=${VERSION} \
        -X main.BuildCommit=${GIT_COMMIT} \
        -X main.BuildDate=${BUILD_DATE}" \
      -o /out/bestdefense-device-monitor.exe \
      ./cmd/bestdefense-device-monitor

# ── stage: artifacts ─────────────────────────────────────────────────────────
# Minimal final image that just holds the compiled artifact.
# Use `docker cp` or a volume mount to extract the .exe.
FROM scratch AS artifacts
COPY --from=build /out/bestdefense-device-monitor.exe /bestdefense-device-monitor.exe
