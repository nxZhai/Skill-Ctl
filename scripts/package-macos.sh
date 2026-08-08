#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${VERSION:-$(awk -F'"' '/const version =/ {print $2; exit}' "$ROOT_DIR/cmd/skillctl/main.go")}"
VERSION="${VERSION:-dev}"
DIST_DIR="$ROOT_DIR/dist"

cd "$ROOT_DIR"

command -v go >/dev/null 2>&1 || { echo "error: go not found in PATH" >&2; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "error: npm not found in PATH" >&2; exit 1; }

if [ ! -d web/node_modules ]; then
  npm --prefix web install
fi
npm --prefix web run build

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

for arch in arm64 amd64; do
  package_dir="$DIST_DIR/skillctl_${VERSION}_darwin_${arch}"
  mkdir -p "$package_dir"

  GOOS=darwin GOARCH="$arch" go build -trimpath -ldflags="-s -w" -o "$package_dir/skillctl" ./cmd/skillctl
  cp README.md README.zh-CN.md "$package_dir/"

  tar -C "$DIST_DIR" -czf "$DIST_DIR/skillctl_${VERSION}_darwin_${arch}.tar.gz" "skillctl_${VERSION}_darwin_${arch}"
  rm -rf "$package_dir"
done

echo "packages:"
ls -1 "$DIST_DIR"/*.tar.gz
