---
name: skillctl-source-migration
description: Use when migrating manually installed, self-authored, vendored, or well-known Agent Skills into Skillctl-managed Git sources. Covers source classification, upstream identity confirmation, personal skill repo packaging, removing third-party skills from personal repos, activation migration, backups, adding sources through Skillctl logic, lock-file cleanup, and verifying DB/unified symlink state.
---

# Skillctl Source Migration

## When To Use

Use this skill when an existing set of skills lives directly in an agent global directory, such as `~/.agents/skills` or `~/.codex/skills`, and the goal is to replace those raw directories with Skillctl-managed Git sources.

Also use it when a third-party skill was vendored into a personal Skillctl source repository and should instead be maintained from its upstream Git repository.

## Safety Contract

- Confirm the upstream Git repository is official or otherwise intended before deleting local directories.
- Classify each local skill before migration: symlink/Git checkout/upstream code-search match means external; no origin or match means self-authored unless the user says otherwise.
- Stop at an approval point before removing any existing skill directories.
- Back up the old skill directories and any installer lock file before cleanup.
- Add the Git repository through Skillctl's source-management path, not by copying files into the agent skill directory.
- Treat "managed but not enabled" as success for fresh source migrations: source skills should exist in Skillctl's DB and unified skills directory, with zero activations unless the user explicitly asks to enable them.
- For vendored-skill removals, preserve intentional existing activations by moving them from the personal source skill ID to the upstream source skill ID.

## Migration Flow

1. Identify the upstream source.
   Verify repository owner, default branch, and the list of skill directories under `skills/`.

2. Compare local and upstream inventory.
   Enumerate matching local skill names, symlink status, and any installer lock entries. If local content differs from upstream, report that the migration will replace local content with the current source checkout.

3. Package self-authored skills when needed.
   Create a separate Git repository with `skills/<skill-name>/...` and a concise `README.md`. Copy only self-authored skill directories into `skills/`, commit, push to a private GitHub repository, then add that repository as a Skillctl source.

4. Ask for approval.
   Include the source URL, source ID to be used, local paths to delete, and the default-not-enabled result.

5. Back up before editing.
   Archive the old directories and copy lock files into a timestamped backup under Skillctl's data area.

6. Add the source through Skillctl.
   Prefer the HTTP API or existing manager/CLI flow so clone, DB insert, scanner, tag update, and unified symlink rebuild all run normally.

7. Clean old global installs.
   Delete only the validated matching directories or symlinks. If an installer lock file tracks them, remove only those exact entries and validate the JSON before replacing the file.

8. Verify.
   Confirm source skill count, unified symlink count, absence of old global entries, absence of matching lock entries, and zero activations for the new source. Run `skillctl doctor` when available.

## Vendored Personal Repo Flow

1. Confirm the source repository.
   Verify the upstream owner, default branch, and that `SKILL.md` is discoverable either at the repository root, under `skills/`, or in another directory that Skillctl's scanner will walk.

2. Inspect current Skillctl state.
   Query `sources`, matching `skills`, and matching `activations`. Check the personal source working tree is clean before editing.

3. Add the upstream as its own source.
   Use Skillctl's HTTP API or manager path so clone, DB insert, scanning, tags, and unified symlinks are created normally.

4. Migrate activations when needed.
   If a personal-source skill is enabled, disable that activation through Skillctl first, then enable the upstream skill with the same agent/scope/project root. This avoids symlink-target conflicts and preserves the user's active workflow.

5. Remove the vendored copy.
   In the personal repository, `git rm -r` only the vendored skill directory and remove only its entry from the README's included-skill list. Keep unrelated references that mention the skill name when the upstream source still provides the same name.

6. Rescan and verify.
   Rescan the personal source, confirm no matching skill remains under that source, confirm the upstream skill exists, and confirm any activation symlink points to `~/.local/share/skillctl/skills/<upstream-source-id>/...`.

7. Commit and push.
   Commit the personal repository removal with a concise message and push its branch.

## Verification Queries

- Source exists: query `sources` for the chosen source ID.
- Skills are managed: count `skills where source_id = ?`.
- Fresh migration not enabled: count `activations where skill_id like '<source-id>::%'` is `0`.
- Vendored replacement enabled as before: query `activations` for the upstream skill ID and inspect the symlink target with `readlink`.
- Unified entries exist: source-specific entries under `~/.local/share/skillctl/skills/<source-id>/` are symlinks.
- Old install is gone: no matching directories or symlinks remain under the agent global skill directory.
