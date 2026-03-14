# BestDefense Device Monitor — Docker-based build tooling
#
# All targets run inside Docker so no local Go installation is required.
# Prerequisites: Docker with BuildKit enabled (default in Docker 23+).
#
# Usage:
#   make build          Build bestdefense-device-monitor.exe → dist/
#   make test           Run tests (native + cross-compile checks)
#   make vet            Run go vet across all packages (GOOS=windows)
#   make tidy           Run go mod tidy, updates go.mod / go.sum on host
#   make all            vet + test + build
#   make clean          Remove dist/ directory
#   make shell          Interactive shell inside the build container

COMPOSE := docker compose -f docker/docker-compose.yml
DIST_DIR := dist

# Inject version metadata from git when available
VERSION     ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
GIT_COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE  ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

export VERSION GIT_COMMIT BUILD_DATE

.PHONY: all build test vet tidy clean shell help

## all: Run vet, test, then build
all: vet test build

## build: Cross-compile bestdefense-device-monitor.exe into dist/
build: $(DIST_DIR)
	$(COMPOSE) run --rm build
	@echo ""
	@echo "  Output: $(DIST_DIR)/bestdefense-device-monitor.exe"
	@ls -lh $(DIST_DIR)/bestdefense-device-monitor.exe 2>/dev/null || true

$(DIST_DIR):
	mkdir -p $(DIST_DIR)

## test: Run tests for all packages
test:
	$(COMPOSE) run --rm test

## vet: Run go vet with GOOS=windows across all packages
vet:
	$(COMPOSE) run --rm vet

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

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
