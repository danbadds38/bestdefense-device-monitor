#!/usr/bin/env bash
# Build script for bestdefense-device-monitor (Linux/macOS CI)
# Usage: ./scripts/build.sh [version]
set -euo pipefail

VERSION="${1:-dev}"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
OUTPUT="dist/bestdefense-device-monitor.exe"

mkdir -p dist

export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=0

echo "Building $OUTPUT (version=$VERSION, commit=$COMMIT)..."

go build \
  -ldflags="-s -w -H windowsgui \
    -X main.Version=${VERSION} \
    -X main.BuildCommit=${COMMIT} \
    -X main.BuildDate=${BUILD_DATE}" \
  -o "${OUTPUT}" \
  ./cmd/bestdefense-device-monitor

SIZE=$(du -sh "${OUTPUT}" | cut -f1)
echo "Built: ${OUTPUT} (${SIZE})"
