# Repository Guidelines

## Project Structure & Module Organization

This repository builds `skillctl`, a local macOS Web UI for managing Agent Skills.

- `cmd/skillctl/` contains the CLI entry point.
- `internal/` contains Go packages for config, database, scanning, sources, activation, doctor checks, and the local HTTP server.
- `migrations/` contains SQLite schema migrations.
- `web/` contains the Vite/React frontend. Edit source in `web/src/`; `web/dist/` is the built output embedded by `web/embed.go`.
- `README.md` and `README.zh-CN.md` document user-facing behavior and setup.
- `start.sh` is the main local build/run helper.

## Build, Test, and Development Commands

- `npm --prefix web install`: install frontend dependencies.
- `npm --prefix web run build`: run TypeScript build and Vite production build into `web/dist/`.
- `go build -o skillctl ./cmd/skillctl`: build the CLI binary.
- `go test ./...`: run all Go tests.
- `./start.sh`: build frontend, build Go binary, then launch `skillctl ui`.
- `./start.sh run`: run the existing binary without rebuilding.
- `./start.sh doctor`: build, then run repository and local environment checks.

Build the frontend before building Go when changes affect `web/src/`, because the Go binary embeds `web/dist`.

## Coding Style & Naming Conventions

Use `gofmt` for all Go files and keep package boundaries aligned with `internal/` responsibilities. Prefer small, direct functions and avoid speculative abstractions. Go package names should stay lowercase and concise, such as `scanner` or `localskills`.

Frontend code is TypeScript with React and Vite. Keep component and type names descriptive, use `PascalCase` for React components, and keep static assets under `web/src/` unless they are generated build output.

## Testing Guidelines

No test files are currently present in this checkout. Add Go tests beside the package under test using the standard `*_test.go` convention, then run `go test ./...`. For frontend changes, at minimum run `npm --prefix web run build` to catch TypeScript and bundling failures.

## Commit & Pull Request Guidelines

This checkout does not include Git history, so no repository-specific commit convention can be inferred. Use concise, imperative commit subjects, for example `Add skill scanner tests` or `Fix activation symlink cleanup`.

Pull requests should include a short problem summary, the implementation approach, verification commands run, and screenshots for visible UI changes.

## Agent-Specific Instructions

Keep changes surgical. Do not refactor unrelated code, do not edit generated `web/dist/` unless rebuilding frontend output is part of the task, and document any skipped verification in your handoff.

After completing any requested update, ask the user whether to commit and push the changes, and whether to reinstall the local `skillctl` binary with `./scripts/install-local.sh`. For frontend-visible changes, keep the final verified local UI session and URL available for user inspection before asking those questions. Do not commit, push, reinstall, or stop the review UI before explicit confirmation. After the user confirms the follow-up action, stop any review UI or browser session started for that inspection.

## Reusable Pattern Skills

After each conversation, or after several related conversations, do a brief reusable-pattern pass. If a durable workflow, UI style, backend convention, safety rule, or domain decision would help future agents, use the `skill-creator` skill to create or update a focused skill under `delopy_created_skills/`.

Keep locally created skills concise, remove template placeholders, and avoid one-off task details. Update `delopy_created_skills/README.md` in Chinese whenever skills are added or materially changed, describing each skill's purpose, when to use it, and the main files or domain it covers.
