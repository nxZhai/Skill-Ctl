---
name: skillctl-local-packaging
description: Use when installing, packaging, releasing, or reviewing Skillctl as a local macOS CLI. Covers the project-specific rule that web/src changes must build web/dist before Go compilation because the frontend is embedded into the skillctl binary, plus verification of installed binaries and release archives.
---

# Skillctl Local Packaging

## Overview

Use this skill when turning the Skillctl repository into a runnable local command or distributable macOS archive. The key invariant is that the Vite frontend is embedded into the Go binary from `web/dist`, so frontend build freshness matters before every install, update, or package build.

## Workflow

1. Check the current tree before editing or packaging:

```bash
git status --short
```

2. For local installation, prefer the repository script:

```bash
./scripts/install-local.sh
```

This installs to `~/.local/bin/skillctl` by default. Use `PREFIX=/usr/local ./scripts/install-local.sh` when the user explicitly wants another prefix.

3. For updating an existing local install from the current branch's upstream, use:

```bash
./scripts/update-local.sh
```

This requires a clean working tree, fetches updates, fast-forwards the current branch, and then reuses `scripts/install-local.sh` to overwrite the installed binary. If the tree is dirty, do not stash automatically; ask the user to commit/stash or install the current checkout.

4. For release archives, use:

```bash
./scripts/package-macos.sh
```

This generates macOS `arm64` and `amd64` archives under `dist/`. Pass `VERSION=x.y.z` only when the archive name should differ from the version constant in `cmd/skillctl/main.go`.

## Rules

- Do not use plain `go install ./cmd/skillctl` as the recommended path unless `web/dist` was just rebuilt.
- Keep `web/dist` generated output out of source edits except for the tracked placeholder.
- Keep release archives in ignored root `dist/`.
- Preserve the single-binary property: runtime startup should be `skillctl ui`, not `npm`, Vite, or `./start.sh`.
- Avoid adding Homebrew, launch agents, notarization, or auto-update logic unless the user explicitly asks for that distribution layer.

## Verification

Run the smallest checks that match the change:

```bash
bash -n scripts/install-local.sh
bash -n scripts/update-local.sh
bash -n scripts/package-macos.sh
./scripts/install-local.sh
skillctl version
./scripts/package-macos.sh
go test ./...
```

For package verification, inspect archive contents with `tar -tzf dist/<archive>.tar.gz` and run the current-architecture binary from an extracted archive when possible.
