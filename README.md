<p align="center">
  <img src="assets/skillctl-logo.svg" width="420" alt="Skillctl logo">
</p>

<h1 align="center">⚡ Skillctl</h1>

<p align="center">
  <strong>A local macOS Web UI for managing Agent Skills from Git repositories.</strong>
</p>

<p align="center">
  <a href="README.zh-CN.md">中文</a>
  ·
  <a href="https://github.com/nxZhai/Skill-Ctl/releases">Releases</a>
</p>

<p align="center">
  <a href="https://github.com/nxZhai/Skill-Ctl/releases"><img alt="Release" src="https://img.shields.io/github/v/release/nxZhai/Skill-Ctl?style=flat-square&color=2F6F68"></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-macOS-5B6575?style=flat-square">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React%20%2B%20Vite-19%20%2F%208-646CFF?style=flat-square&logo=react&logoColor=white">
  <img alt="Local first" src="https://img.shields.io/badge/local--first-SQLite-41B3A3?style=flat-square">
</p>

## ✨ Features

- **Git source management**: add GitHub or Git-compatible repositories as skill sources, then sync them locally.
- **Recursive skill discovery**: scan repositories for `SKILL.md` files and track each skill's source, path, description, tags, and activation state.
- **Local skill inventory**: inspect existing Claude Code or Codex skill folders so Skillctl reflects what agents can already see.
- **Search and filters**: filter skills by keyword, source, tag, agent, and enabled state.
- **Tags and notes**: add repository/skill tags and personal notes to keep large skill libraries navigable.
- **Batch operations**: select multiple skills to add/remove tags or enable them in bulk.
- **Global and project activation**: enable skills globally or into a project-local agent directory.
- **Symlink deployment**: deploy skills through managed symlinks instead of copying source files.
- **Project manifests**: generate and apply project-level `.skillctl.toml` manifests.
- **Usage ranking**: summarize observed skill usage from local Codex/agent history.
- **Doctor checks**: inspect Git availability, source checkouts, local changes, broken symlinks, and manifest references.
- **Single-binary local app**: run a local token-protected HTTP server with the Vite frontend embedded in the Go binary.

## 🧭 Core Concepts

```text
Git repo is the sync/update unit
Skill is the browse/filter/enable unit
Symlink is the deployment mechanism
```

Skillctl keeps source repositories separate from activation targets. Repositories are cloned under local data storage, discovered skills are indexed in SQLite, and enabled skills are exposed to agents through controlled symlinks.

## 🚀 Commands

```bash
skillctl ui
skillctl doctor
skillctl rescan [source-id]
skillctl update [--check]
skillctl uninstall
skillctl version
```

`skillctl ui` initializes local config/data directories, starts an HTTP server on `127.0.0.1` with a random token, and opens the browser.

## 📦 Local Install

Install the embedded `skillctl` binary to `~/.local/bin`:

```bash
./scripts/install-local.sh
```

If `~/.local/bin` is not already in your shell PATH, add it to `~/.zshrc`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then start Skillctl from any directory:

```bash
skillctl ui
```

To install somewhere else, pass `PREFIX`:

```bash
PREFIX=/usr/local ./scripts/install-local.sh
```

## 🔄 Updating

Check for a newer GitHub Release:

```bash
skillctl update --check
```

Install the latest compatible release:

```bash
skillctl update
```

Before replacing the binary, Skillctl creates a compressed backup under `~/.cache/skillctl/backups/` containing its config, state, managed repositories, and unified skill links. It hashes that data before and after the update and reports an error if anything changed. Starting `skillctl ui` also prints a terminal notice when a newer release is available.

For a source checkout used in development, update the checkout and reinstall it with:

```bash
./scripts/update-local.sh
```

## 🗑️ Uninstall

```bash
skillctl uninstall
```

Uninstall first removes every Skillctl-recorded agent skill symlink, with the same target validation used by the UI. It then asks whether to delete Skillctl-managed local skill repositories. Choosing no preserves those repositories and the local state for a future reinstall.

## 🗜️ Packaging

Create macOS `arm64` and `amd64` release archives under `dist/`:

```bash
./scripts/package-macos.sh
```

Set `VERSION` to override the version used in archive names:

```bash
VERSION=0.5.0 ./scripts/package-macos.sh
```

## 🛠️ Development

Build the frontend first so Go can embed `web/dist`:

```bash
cd web
npm install
npm run build
cd ..
go build -o skillctl ./cmd/skillctl
```

Run checks:

```bash
go test ./...
npm --prefix web run build
```

## 🗂️ Local Data

Skillctl uses:

```text
~/.config/skillctl/
~/.local/share/skillctl/
~/.cache/skillctl/
```

Git repositories are cloned under `~/.local/share/skillctl/repos/`. Unified skill symlink entries are created under `~/.local/share/skillctl/skills/`.

## 🔒 Safety

Skillctl only creates symlinks, refuses to overwrite ordinary files or foreign symlinks, and deletes only activation links recorded in SQLite.

Git synchronization uses `fetch --prune` and `merge --ff-only`; it does not run `git reset --hard`. The local UI listens only on `127.0.0.1`, and all API requests must include the random token generated at startup.
