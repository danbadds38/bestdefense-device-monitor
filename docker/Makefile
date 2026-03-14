# BestDefense Device Monitor — Docker-based build tooling
#
# All targets run inside Docker so no local Go installation is required.
# Prerequisites: Docker with BuildKit enabled (default in Docker 23+).
#
# Usage:
#   make build              Build Windows .exe -> dist/
#   make build-all          Build all 5 platform binaries -> dist/
#   make build-windows      Build Windows amd64 binary -> dist/
#   make build-darwin-amd64 Build macOS Intel binary -> dist/
#   make build-darwin-arm64 Build macOS Apple Silicon binary -> dist/
#   make build-linux-amd64  Build Linux amd64 binary -> dist/
#   make build-linux-arm64  Build Linux arm64 binary -> dist/
#   make package-linux      Build .deb and .rpm packages -> dist/
#   make test               Run tests (native + cross-compile checks for all platforms)
#   make vet                Run go vet for Windows (default)
#   make vet-all            Run go vet for all three platforms
#   make tidy               Run go mod tidy, updates go.mod / go.sum on host
#   make all                vet-all + test + build-all
#   make clean              Remove dist/ directory
#   make shell              Interactive shell inside the build container
#   make help               Show this help message

COMPOSE := docker compose -f docker/docker-compose.yml
DIST_DIR := dist

# Inject version metadata from git when available
VERSION     ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
GIT_COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE  ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

export VERSION GIT_COMMIT BUILD_DATE

.PHONY: all \
        build build-windows build-darwin-amd64 build-darwin-arm64 \
        build-linux-amd64 build-linux-arm64 build-all \
        package-linux \
        test vet vet-all \
        tidy clean shell help

## all: Run vet-all, test, then build-all
all: vet-all test build-all

# ── Build targets ──────────────────────────────────────────────────────────────

## build: Cross-compile Windows .exe into dist/ (legacy default)
build: $(DIST_DIR)
	$(COMPOSE) run --rm build
	@echo ""
	@echo "  Output: $(DIST_DIR)/bestdefense-device-monitor-windows-amd64.exe"
	@ls -lh $(DIST_DIR)/bestdefense-device-monitor-windows-amd64.exe 2>/dev/null || true

## build-windows: Build Windows amd64 binary into dist/
build-windows: $(DIST_DIR)
	$(COMPOSE) run --rm build-windows
	@echo ""
	@echo "  Output: $(DIST_DIR)/bestdefense-device-monitor-windows-amd64.exe"
	@ls -lh $(DIST_DIR)/bestdefense-device-monitor-windows-amd64.exe 2>/dev/null || true

## build-darwin-amd64: Build macOS Intel binary into dist/
build-darwin-amd64: $(DIST_DIR)
	$(COMPOSE) run --rm build-darwin-amd64
	@echo ""
	@echo "  Output: $(DIST_DIR)/bestdefense-device-monitor-darwin-amd64"
	@ls -lh $(DIST_DIR)/bestdefense-device-monitor-darwin-amd64 2>/dev/null || true

## build-darwin-arm64: Build macOS Apple Silicon binary into dist/
build-darwin-arm64: $(DIST_DIR)
	$(COMPOSE) run --rm build-darwin-arm64
	@echo ""
	@echo "  Output: $(DIST_DIR)/bestdefense-device-monitor-darwin-arm64"
	@ls -lh $(DIST_DIR)/bestdefense-device-monitor-darwin-arm64 2>/dev/null || true

## build-linux-amd64: Build Linux amd64 binary into dist/
build-linux-amd64: $(DIST_DIR)
	$(COMPOSE) run --rm build-linux-amd64
	@echo ""
	@echo "  Output: $(DIST_DIR)/bestdefense-device-monitor-linux-amd64"
	@ls -lh $(DIST_DIR)/bestdefense-device-monitor-linux-amd64 2>/dev/null || true

## build-linux-arm64: Build Linux arm64 binary into dist/
build-linux-arm64: $(DIST_DIR)
	$(COMPOSE) run --rm build-linux-arm64
	@echo ""
	@echo "  Output: $(DIST_DIR)/bestdefense-device-monitor-linux-arm64"
	@ls -lh $(DIST_DIR)/bestdefense-device-monitor-linux-arm64 2>/dev/null || true

## build-all: Build all 5 platform binaries into dist/
build-all: $(DIST_DIR)
	$(COMPOSE) run --rm build-all
	@echo ""
	@echo "  Outputs in $(DIST_DIR)/:"
	@ls -lh $(DIST_DIR)/ 2>/dev/null || true

# ── Package targets ────────────────────────────────────────────────────────────

## package-linux: Build .deb and .rpm packages from the Linux amd64 binary
package-linux: $(DIST_DIR)
	$(COMPOSE) run --rm package-linux
	@echo ""
	@echo "  Packages in $(DIST_DIR)/:"
	@ls -lh $(DIST_DIR)/*.deb $(DIST_DIR)/*.rpm 2>/dev/null || true

# ── Test / vet targets ─────────────────────────────────────────────────────────

## test: Run tests for all packages (native + cross-compile for all platforms)
test:
	$(COMPOSE) run --rm test

## vet: Run go vet with GOOS=windows across all packages
vet:
	$(COMPOSE) run --rm vet

## vet-all: Run go vet for Windows, macOS, and Linux
vet-all:
	$(COMPOSE) run --rm vet-all

# ── Utility targets ────────────────────────────────────────────────────────────

## tidy: Run go mod tidy (updates go.mod and go.sum)
tidy:
	$(COMPOSE) run --rm tidy

## clean: Remove the dist/ directory
clean:
	rm -rf $(DIST_DIR)
	@echo "Cleaned dist/"

## shell: Open an interactive shell inside the build container
shell:
	$(COMPOSE) run --rm shell

$(DIST_DIR):
	mkdir -p $(DIST_DIR)

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
