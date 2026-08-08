#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

command -v git >/dev/null 2>&1 || { echo "error: git not found in PATH" >&2; exit 1; }

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: update-local.sh must run from a Git checkout" >&2
  exit 1
fi

if [ -n "$(git status --porcelain)" ]; then
  echo "error: working tree has uncommitted changes" >&2
  echo "commit or stash them before updating, or run ./scripts/install-local.sh to install the current checkout" >&2
  exit 1
fi

upstream="$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null || true)"
if [ -z "$upstream" ]; then
  echo "error: current branch has no upstream" >&2
  echo "set an upstream branch, or pull updates manually and then run ./scripts/install-local.sh" >&2
  exit 1
fi

before="$(git rev-parse --short HEAD)"

git fetch --prune
git merge --ff-only "$upstream"

after="$(git rev-parse --short HEAD)"
if [ "$before" = "$after" ]; then
  echo "source already up to date at $after"
else
  echo "updated source: $before -> $after"
fi

exec "$ROOT_DIR/scripts/install-local.sh"
