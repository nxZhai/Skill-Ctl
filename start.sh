#!/usr/bin/env bash
set -euo pipefail

# One-key start for Skillctl:
#   1. build the web frontend (so Go can embed web/dist)
#   2. build the Go binary
#   3. launch the local Web UI
#
# Usage:
#   ./start.sh            build (incremental) and run the UI
#   ./start.sh run        skip building, just run the existing binary (fast)
#   ./start.sh --clean    force reinstall web deps and rebuild everything
#   ./start.sh doctor     build then run `skillctl doctor`

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

CLEAN=0
SKIP_BUILD=0
RUN_CMD="ui"
for arg in "$@"; do
  case "$arg" in
    --clean) CLEAN=1 ;;
    run) SKIP_BUILD=1 ;;
    ui|doctor|version) RUN_CMD="$arg" ;;
    *) echo "unknown argument: $arg" >&2; exit 2 ;;
  esac
done

# Fast path: run the already-built binary without rebuilding.
if [ "$SKIP_BUILD" -eq 1 ]; then
  if [ ! -x ./skillctl ]; then
    echo "error: ./skillctl not found; run ./start.sh first to build it" >&2
    exit 1
  fi
  echo "==> Starting skillctl $RUN_CMD (no build)"
  exec ./skillctl "$RUN_CMD"
fi

command -v go >/dev/null 2>&1 || { echo "error: 'go' not found in PATH" >&2; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "error: 'npm' not found in PATH" >&2; exit 1; }

echo "==> Building web frontend"
if [ "$CLEAN" -eq 1 ] || [ ! -d web/node_modules ]; then
  npm --prefix web install
fi
npm --prefix web run build

echo "==> Building skillctl binary"
go build -o skillctl ./cmd/skillctl

echo "==> Starting skillctl $RUN_CMD"
exec ./skillctl "$RUN_CMD"
