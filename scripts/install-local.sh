#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PREFIX="${PREFIX:-$HOME/.local}"
BIN_DIR="$PREFIX/bin"

cd "$ROOT_DIR"

command -v go >/dev/null 2>&1 || { echo "error: go not found in PATH" >&2; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "error: npm not found in PATH" >&2; exit 1; }

if [ ! -d web/node_modules ]; then
  npm --prefix web install
fi
npm --prefix web run build

mkdir -p "$BIN_DIR"
go build -trimpath -ldflags="-s -w" -o "$BIN_DIR/skillctl" ./cmd/skillctl

echo "installed: $BIN_DIR/skillctl"
"$BIN_DIR/skillctl" version
echo "run: skillctl ui (or skillctl headless)"

resolved="$(command -v skillctl || true)"
if [ -z "$resolved" ]; then
  echo "note: add $BIN_DIR to PATH before running skillctl from any directory"
elif [ "$resolved" != "$BIN_DIR/skillctl" ]; then
  echo "note: PATH currently resolves skillctl to $resolved"
  echo "note: put $BIN_DIR earlier in PATH to use this installation by default"
fi
