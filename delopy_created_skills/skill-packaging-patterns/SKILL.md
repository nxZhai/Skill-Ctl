---
name: skill-packaging-patterns
description: Use when creating, updating, reviewing, or organizing Agent Skill folders in this repository. Captures the reusable packaging pattern from Skillctl: concise SKILL.md frontmatter, optional agents/openai.yaml UI metadata, minimal resources, stable names, progressive disclosure, and validation before deploying skills under delopy_created_skills.
---

# Skill Packaging Patterns

## Overview

Use this skill to turn repeated project knowledge into deployable Agent Skills without bloating the context window or creating extra documentation inside each skill.

## Folder Contract

A skill folder should usually contain only:

```text
skill-name/
├── SKILL.md
└── agents/
    └── openai.yaml
```

Add `scripts/`, `references/`, or `assets/` only when the skill needs deterministic helpers, large reference material, or reusable output files.

## Naming And Metadata

- Use lowercase hyphen-case names: `research-ui-frontend`, `skillctl-local-backend`.
- Keep names stable because Skillctl identifies skills by source and relative path.
- `SKILL.md` frontmatter must include `name` and a description that states when to use the skill.
- `description` should be trigger-oriented, not a marketing summary.
- `agents/openai.yaml` should include `display_name`, `short_description`, and `default_prompt` when the skill is meant to appear in UI lists.
- Regenerate `agents/openai.yaml` if the skill's purpose changes.

## Writing The Skill Body

- Start with a short overview: one or two sentences explaining what the skill enables.
- Prefer reusable procedure over historical narrative. Mention history only when it explains a stable rule.
- Use clear sections such as `Product Model`, `Workflow`, `Patterns`, `Safety`, and `Verification`.
- Keep instructions compact enough to load cheaply. Split large details into `references/` only when they are not always needed.
- Include verification commands when the skill affects code, generated assets, or deployable artifacts.
- Do not add a `README.md` inside the skill folder. For this repository, directory-level Chinese documentation belongs in `delopy_created_skills/README.md`.

## Creation Workflow

1. Mine repeated decisions from Git history, diffs, README files, source code, and recent conversation context.
2. Reject one-off facts that are unlikely to repeat.
3. Choose the smallest set of skills that each have a clear trigger.
4. Create each folder with `skill-creator` tooling when available.
5. Replace all template placeholder content with concise instructions.
6. Generate or refresh `agents/openai.yaml`.
7. Update `delopy_created_skills/README.md` in Chinese with the skill name, purpose, and when to use it.
8. Validate the structure and report any skipped validation.

## What Not To Package

- Do not package ordinary task results that will not generalize.
- Do not create broad skills with vague triggers such as "project helper".
- Do not duplicate repository README content unless the skill needs it for execution.
- Do not include generated build output, local databases, logs, or dependency folders.
- Do not introduce resource directories just because the template mentions them.

## Validation

- Confirm each skill has `SKILL.md` with valid `---` frontmatter.
- Confirm every frontmatter `name` matches the folder name.
- Confirm no template placeholders remain.
- Confirm `agents/openai.yaml` has quoted interface strings and a default prompt that names the skill with `$skill-name`.
- If the official validator cannot run because dependencies are missing, perform these checks manually and state that limitation.
