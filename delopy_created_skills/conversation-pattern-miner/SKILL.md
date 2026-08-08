---
name: conversation-pattern-miner
description: Use at the end of one or more project conversations to identify reusable workflows, UI patterns, coding conventions, safety rules, or domain decisions that should become or update Agent Skills under delopy_created_skills, with a Chinese README entry for each retained skill.
---

# Conversation Pattern Miner

## Overview

Use this skill as the closing pass after meaningful work. It turns repeated decisions into durable skills while filtering out one-off task details.

## When To Run

Run this pass when any of the following happened:

- A conversation introduced a repeatable implementation workflow.
- Multiple conversations reveal the same design, safety, testing, or review rule.
- A fix required non-obvious project knowledge that future agents should not rediscover.
- A new frontend/backend/domain pattern became clear from code and history.
- The user explicitly asks to summarize reusable patterns or create skills.

Skip skill creation when the work was trivial, entirely one-off, or already fully covered by an existing skill.

## Mining Workflow

1. Gather evidence from the latest user request, assistant actions, diffs, tests, relevant README/AGENTS text, and recent Git history.
2. List candidate patterns in plain language.
3. For each candidate, ask whether it has a future trigger, reduces rediscovery, and is not merely project trivia.
4. Merge candidates into existing skills when the trigger overlaps.
5. Create a new skill only when it has a distinct trigger and enough reusable procedure.
6. Store or update the skill under `delopy_created_skills/<skill-name>/`.
7. Update `delopy_created_skills/README.md` in Chinese.
8. Validate structure and mention any skipped validation in the handoff.

## Candidate Scoring

Keep a candidate if at least three are true:

- It would affect future code or design decisions.
- It is project-specific enough that a general model may miss it.
- It has a clear "use when" trigger.
- It can be stated as a workflow, guardrail, checklist, or compact reference.
- It was reinforced by code, tests, docs, commit history, or multiple user requests.

Reject a candidate if it is just a bug detail, temporary file path, personal preference without future action, or a broad reminder already covered by AGENTS.md.

## Skill Update Rules

- Use `skill-creator` tooling when available to initialize or regenerate metadata.
- Keep skill bodies concise; avoid copying conversation transcripts.
- Remove all template placeholder text.
- Do not add extra README files inside individual skills.
- Keep the Chinese directory README as the human index for all locally created skills.
- If a skill changes materially, refresh `agents/openai.yaml` so UI metadata stays accurate.

## Chinese README Entry

For each skill, include:

- Skill name.
- One-sentence purpose in Chinese.
- When to use it.
- Primary files or domain it covers.

Keep entries short. The README is an index, not the full instruction source.

## Verification

- Confirm every new or changed skill has valid frontmatter, no template placeholders, and matching UI metadata.
- Confirm `delopy_created_skills/README.md` lists every skill in the directory.
- Run repository tests/builds only if the skill update also changed executable code. For documentation-only skill updates, structural validation is enough.
