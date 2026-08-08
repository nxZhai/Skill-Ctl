---
name: skillctl-local-backend
description: Use when implementing, changing, or reviewing Skillctl's Go backend for local macOS skill management. Covers internal package boundaries, SQLite persistence, token-protected localhost HTTP APIs, Git source sync including multi-remote status, SKILL.md scanning, usage telemetry, symlink-only deployment, project manifests, doctor checks, and safe filesystem behavior.
---

# Skillctl Local Backend

## Overview

Use this skill to preserve Skillctl's backend contract: a local-only web tool that manages skills from Git repositories through SQLite metadata and symlinks, without overwriting user files or performing destructive Git operations.

## Core Domain Rules

- Git repositories are the sync/update unit.
- Skills are the browse/filter/enable unit.
- Symlinks are the deployment mechanism; real skill directories remain in source checkouts.
- SQLite records source metadata, discovered skills, tags, and activations. Filesystem operations must agree with DB state.
- The local UI listens on `127.0.0.1` with a random token. API requests accept the token from query string, `X-Skillctl-Token`, or bearer authorization.

## Package Responsibilities

- `cmd/skillctl`: command dispatch, config/db initialization, random token, browser launch, CLI output.
- `internal/config`: config initialization, path expansion, storage directory application.
- `internal/database`: schema, migrations, CRUD, transactions, UTC timestamps.
- `internal/sources`: Git clone/check/sync, source view status, skill content/tree/open actions.
- `internal/scanner`: recursive `SKILL.md` discovery, metadata extraction, unified skill symlink rebuilding.
- `internal/activation`: enable/disable user or project activations through managed symlinks.
- `internal/project`: `.skillctl.toml` read/write/apply/clean.
- `internal/doctor`: environment, checkout, symlink, and manifest integrity checks.
- `internal/server`: thin HTTP routing, JSON decode/encode, static frontend embedding.

## Safety Patterns

- Never use `git reset --hard`, force checkout, or destructive cleanup for source sync. Current sync uses `fetch --prune` and `merge --ff-only`.
- Refuse sync when a source checkout has local changes. Surface the condition in `SourceView`.
- When the UI needs skill-level Git change indicators, annotate skills by matching changed Git paths to each skill's `relative_path`. Do not mark every skill in a dirty or behind source as changed.
- Refuse to overwrite ordinary files or foreign symlinks. `ensureManagedSymlink` should only create missing links or accept links already pointing at the expected target.
- When disabling activation, delete only the DB-recorded symlink and only when it is a symlink to the expected target.
- When removing a source, keep its checkout by default. Preflight every recorded activation and unified skill entry, remove only registered symlinks with expected targets, then delete the source record so SQLite cascades to skills, tags, and activations.
- Refuse source removal when the unified source directory contains ordinary files, unregistered links, unexpected targets, or resolves outside `SkillsDir`.
- Clean user-provided relative paths before opening files. Ensure resolved paths stay under the skill root.
- Keep operations idempotent where the UI may retry, especially activation enable and tag updates.

## Source And Scanner Patterns

- Derive a source ID from Git URL only when the user did not provide one; validate IDs with letters, numbers, dot, underscore, or dash.
- For source status, treat `ahead` and `behind` separately. `behind > 0` means an update is available; `ahead > 0` means local or personal commits and should be displayed on the relevant remote branch, not promoted to a warning message.
- For checkouts with multiple remotes, expose structured remote/branch summaries in `SourceView`: include the configured source remote/branch and the current branch upstream when they differ. Keep legacy `remote_sha`, `ahead`, and `behind` fields as compatibility mirrors of the configured source branch.
- When status depends on more than one remote, fetch all remotes during Check so personal forks and upstream origins are both current. Keep Sync non-destructive and fast-forward-only.
- Scan dedicated roots first: repository `skills/`, nested `*/skills/`, then `plugins/*/skills/`, and finally the repo root.
- Skip heavy/generated directories: `.git`, `node_modules`, `vendor`, `dist`, `build`, and similar local outputs.
- Limit recursive skill search depth to avoid accidental full-repo traversal.
- Read only the `SKILL.md` frontmatter and early heading needed for name/description/content SHA. Do not require a full YAML parser in scanner hot paths.
- Rebuild unified symlinks after scanning and prune stale symlink files for the source.
- For uncommitted local skill changes, parse raw `git status --porcelain` output without trimming leading status columns; for remote skill changes, compare remote-side paths from `HEAD...origin/<branch>`.

## API Patterns

- Keep `server.handleAPI` as thin routing glue. Put behavior in the relevant manager package.
- Decode request structs close to route handling, trim user text at boundaries, and return JSON for both success and errors.
- Use `writeJSONOrError` for simple internal errors; use explicit HTTP status for bad requests, unauthorized access, and not found.
- Preserve frontend type shapes when adding model fields; update Go structs, JSON tags, and TypeScript types together.
- If changing embedded frontend behavior, build `web/dist` before building or shipping the Go binary.

## Database Patterns

- Use `CREATE TABLE IF NOT EXISTS` for current schema and small `ALTER TABLE` migrations for existing local DBs.
- Enable foreign keys with `PRAGMA foreign_keys = ON`.
- Use transactions for multi-row replacement or tag updates.
- Prefer `INSERT OR IGNORE` for idempotent activation/tag inserts where uniqueness is the intended guard.
- Store timestamps as UTC RFC3339 strings through `database.Now()`.

## Usage Telemetry

- Claude usage comes from explicit `/skill`, `$skill`, or `[$skill]` mentions in `~/.claude/history.jsonl`.
- Codex Desktop usage must read active and archived rollout JSONL files, not only `~/.codex/history.jsonl`.
- Treat a Codex `SKILL.md` read as evidence that Codex selected the skill; this covers implicit invocation.
- Group evidence by turn and count each skill at most once when an explicit mention and `SKILL.md` read occur together.
- Do not multiply one invocation across duplicate skill names. Prefer an exact resolved path; otherwise choose one deterministic candidate.
- JSONL history and rollout records can contain very long single lines. Avoid `bufio.Scanner` for these files unless the max token size is proven sufficient; prefer a line reader without a fixed token ceiling and cover multi-megabyte lines in tests.
- Keep scanning user-triggered. Cache extracted turns by rollout filename, size, and modification time so unchanged conversations are not parsed again.
- Persist the latest ranking snapshot per time range so the Usage UI can restore previous results and show the last update time without scanning history.
- Store derived cache data under `~/.cache/skillctl/`, never inside source repositories or Codex session folders.

## Verification

- Run `gofmt` on changed Go files.
- Run focused Go tests for touched packages when present, then `go test ./...` for backend changes.
- For frontend-visible API changes, also run `npm --prefix web run build`.
- Use `./start.sh doctor` when changes affect filesystem, Git checkout, symlink, or manifest behavior.
