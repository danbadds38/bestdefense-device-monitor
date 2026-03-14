# Docker Build Environment

Portable build, test, and lint tooling for `bestdefense-device-monitor`.

No local Go installation required — everything runs inside Docker and cross-compiles a Windows `.exe` from a Linux container.

## Prerequisites

- Docker 23+ (BuildKit enabled by default)
- `make` (GNU Make)

## Quick start

Run all checks and produce the `.exe`:

```sh
# From the repo root
make -f docker/Makefile all
```

Output lands in `dist/bestdefense-device-monitor.exe`.

---

## Targets

| Target | What it does |
|--------|-------------|
| `make -f docker/Makefile all` | `vet` → `test` → `build` |
| `make -f docker/Makefile build` | Cross-compile `bestdefense-device-monitor.exe` into `dist/` |
| `make -f docker/Makefile test` | Run tests — native for platform-agnostic packages; cross-compile check for Windows packages |
| `make -f docker/Makefile vet` | `go vet` with `GOOS=windows` across all packages |
| `make -f docker/Makefile tidy` | `go mod tidy` — writes updated `go.mod`/`go.sum` back to the host |
| `make -f docker/Makefile clean` | Delete `dist/` |
| `make -f docker/Makefile shell` | Interactive shell inside the build container for debugging |

You can also run targets via docker compose directly:

```sh
docker compose -f docker/docker-compose.yml run --rm test
docker compose -f docker/docker-compose.yml run --rm build
```

---

## How testing works

Most of the agent's code uses Windows-specific APIs (`WMI`, `Win32`, registry) guarded by `//go:build windows`. Since we can't execute Windows binaries inside a Linux container, tests run in two modes:

**Native (Linux)** — packages with no Windows dependencies run natively:
- `internal/config`
- `internal/reporter`

**Cross-compile check (Windows target)** — Windows-specific packages are compiled into test binaries with `go test -c` but not executed. This catches type errors, missing symbols, and build constraint issues without needing a Windows host:
- `internal/collector`
- `internal/service`
- `internal/logging`

For full end-to-end execution tests (confirming the `.exe` actually runs and reports correctly) use the GitHub Actions `windows-latest` runner — see [`.github/workflows/release.yml`](../.github/workflows/release.yml).

---

## Using with GitHub Actions (ubuntu runner)

Switching from `windows-latest` to `ubuntu-latest` via Docker is faster and cheaper. Add this job to your workflow:

```yaml
jobs:
  build-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set build metadata
        run: |
          echo "VERSION=${GITHUB_REF_NAME#v}" >> $GITHUB_ENV
          echo "GIT_COMMIT=$(git rev-parse --short HEAD)" >> $GITHUB_ENV
          echo "BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> $GITHUB_ENV

      - name: Run vet + test + build
        run: make -f docker/Makefile all

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: bestdefense-device-monitor-windows-amd64
          path: dist/bestdefense-device-monitor.exe
```

> The existing `windows-latest` workflow in `.github/workflows/release.yml` is the authoritative release pipeline. This Docker-based approach is intended for fast PR checks and local development.

---

## Overriding the Go version

```sh
# Build with a specific Go version
docker compose -f docker/docker-compose.yml build --build-arg GO_VERSION=1.23 build
```

Or set it in the environment before running make targets.

---

## Build cache

Docker BuildKit caches the `go mod download` layer separately from source code, so dependency downloads only happen when `go.mod` or `go.sum` change. Subsequent builds of just source changes are fast.

To prune the build cache:

```sh
docker builder prune
```
