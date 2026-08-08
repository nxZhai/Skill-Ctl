---
name: research-ui-frontend
description: Use when starting, migrating, designing, or reviewing a frontend that should have a research/operations product style: warm-neutral light theme, quiet dark theme, Inter/JetBrains Mono/Instrument Serif typography, compact data-dense layouts, small-radius controls, icon-led actions, understated logos, precise underlines, status-rich cards, modals, filters, drawers, and responsive dashboards. Applicable to any new or existing web project across repositories.
---

# Research UI Frontend

## Overview

Use this skill to quickly establish a portable frontend style for serious research tools, local operations dashboards, admin systems, and technical product UIs. It provides a reusable visual system and implementation workflow independent of any one repository.

## When To Use

- Starting a new React/Vite/Next/Svelte/plain web app that needs a polished research-operations UI immediately.
- Migrating an existing app away from generic SaaS cards, one-note gradients, or marketing-style hero layouts.
- Designing dense but readable dashboards with sources, records, tasks, experiments, evaluations, logs, rankings, or local files.
- Creating a style guide with fonts, weights, colors, icons, logo treatment, spacing, controls, cards, modals, and responsive behavior.
- Reviewing whether a frontend matches this style.

## First Pass Workflow

1. Identify the product domain and the primary objects users inspect or operate on.
2. Map screens around those objects: list/ranking/table, detail modal or side panel, filters, bulk actions, status/progress, settings.
3. Apply the style kit in `references/style-kit.md`, starting with CSS tokens, typography, buttons, links, cards, and layout primitives.
4. Choose icons from lucide or the app's existing icon library. Use icon-only buttons for common tools and icon+text for major commands.
5. Create a compact logo/brand mark using text plus a simple geometric or monogram treatment. Avoid illustrative mascots or decorative blobs.
6. Build the first usable screen directly. Do not start with a landing page unless the user explicitly requested marketing content.
7. Verify desktop and mobile views for text overflow, controls that jump, contrast, dark theme, and responsive stacking.

## Style Kit

Read `references/style-kit.md` when implementing or reviewing visual details. It contains:

- Font families, weights, sizes, letter spacing, line height, and mono usage.
- Light/dark palette tokens and semantic status colors.
- Link and underline rules.
- Layout, spacing, cards, borders, shadows, and radius rules.
- Button, input, select, tab, chip, badge, table/list, drawer, modal, toast, and progress patterns.
- Icon and logo guidance.
- Starter CSS tokens and component snippets.

## Adaptation Rules

- Use the user's framework and design system if one already exists; port the style through tokens and component conventions.
- Keep the interface work-focused: dense but organized, low decoration, visible status, strong hierarchy, clear affordances.
- Use cards only for repeated items, modals, or framed tools. Do not nest cards or make entire page sections float as cards.
- Keep border radii at 3-6px by default, 8px maximum unless an existing design system requires otherwise.
- Do not use gradient orbs, bokeh blobs, oversized generic hero sections, or stock-like atmospheric imagery.
- Prefer structured CSS variables over hard-coded colors in components.
- Keep section titles and descriptions as the top visual hierarchy; when multiple controls would crowd the header, move them into a left-aligned toolbar below the title and make segmented controls visually equal in height.
- For short dashboard pages, use a column app shell with `main` flexing to fill available height so small footers sit at the viewport bottom instead of floating after content.
- Every visible string should fit in English and any target locale; compact controls need shorter labels.
- Keep compact input-plus-action controls in one grid row (`minmax(0, 1fr) auto`) and verify matching top positions and heights in the browser.
- Derive row-level aggregate badges from the full underlying object set, not the currently filtered visible subset, so filters do not create misleading repo/project status.
- When configuration keys differ only by case, deduplicate their displayed options case-insensitively but preserve the chosen real key for API requests.

## Verification

- Run the project's build/typecheck command after implementation.
- Inspect at least one desktop width and one mobile width.
- Check light and dark themes if both exist.
- Confirm icons render, logo treatment is legible, links/underlines are clear, and long technical text does not break layout.
- For local apps, open the app in Browser/Playwright when a dev server or file target is available.
