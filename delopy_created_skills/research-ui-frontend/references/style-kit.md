# Research UI Frontend Style Kit

## Design Intent

Build a quiet, precise, technical product UI: appropriate for research tools, experiment dashboards, local developer utilities, evaluation browsers, knowledge bases, and operations consoles. The interface should feel serious and crafted, not corporate-generic, playful, or marketing-led.

## Typography

- Primary UI font: Inter. Use `@fontsource/inter` in bundled apps or Google Fonts/CDN only when acceptable for the project.
- Mono font: JetBrains Mono for paths, hashes, IDs, code, metrics, time ranges, and technical metadata.
- Display accent: Instrument Serif for compact brand titles or top-level display headings only. Do not use it for dense controls or body copy.
- Font weight scale:
  - 400 for body and secondary metadata.
  - 500 for buttons, labels, tabs, chips, and table headers.
  - 600 for section headings and important row titles.
  - 700 only for brand mark, selected navigation, or critical emphasis.
- Base font size: `16px`; control text: `0.9rem` to `0.98rem`; compact metadata: `0.76rem` to `0.84rem`.
- Line height: `1.45` to `1.6` for prose, `1.2` to `1.3` inside buttons and badges.
- Letter spacing: `0` by default. If a design already uses slight tightening, keep it no more aggressive than `-0.005em` for body and `-0.01em` for headings.

## Palette

Use a warm-neutral light theme and quiet charcoal dark theme. Keep accent color sparse; state colors should carry meaning.

```css
:root {
  --sans: "Inter", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  --mono: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
  --serif: "Instrument Serif", "Iowan Old Style", Georgia, serif;

  --page-bg: #faf8f4;
  --surface: #fffefd;
  --surface-soft: #f2eee7;
  --surface-hover: #ebe5dc;
  --control-bg: #ffffff;
  --panel-soft: #f8f5ee;
  --border: #e1d9ca;
  --border-strong: #c8bdac;
  --ink: #17191f;
  --text-muted: #667085;

  --solid-bg: #17191f;
  --solid-hover: #050608;
  --solid-text: #ffffff;
  --accent-soft: #efebe3;

  --success: #0a7d4d;
  --success-surface: #ecfdf3;
  --success-border: #b6e7d0;
  --warning: #8a6d00;
  --warning-surface: #fff8eb;
  --warning-border: #f8d6a4;
  --danger: #b42318;
  --danger-surface: #fef3f2;
  --danger-border: #f3b4ab;

  --focus: rgba(23, 25, 31, 0.22);
  --overlay-bg: rgba(15, 23, 42, 0.5);
  --radius: 4px;
  --radius-sm: 3px;
  --shadow: 0 1px 2px rgba(16, 24, 40, 0.04), 0 10px 30px rgba(38, 31, 23, 0.08);
  --motion: 180ms cubic-bezier(0.2, 0.8, 0.2, 1);
}

[data-theme="dark"] {
  --page-bg: #15161a;
  --surface: #1d1f24;
  --surface-soft: #25272d;
  --surface-hover: #2d3037;
  --control-bg: #202228;
  --panel-soft: #22242a;
  --border: #343741;
  --border-strong: #4b505d;
  --ink: #f3efe7;
  --text-muted: #aab2c1;
  --solid-bg: #f3efe7;
  --solid-hover: #ffffff;
  --solid-text: #15161a;
  --accent-soft: #272a31;
  --success: #74d6a3;
  --warning: #f6c96f;
  --danger: #ff8f84;
  --focus: rgba(216, 221, 232, 0.24);
  --overlay-bg: rgba(2, 6, 23, 0.66);
  --shadow: none;
}
```

## Links And Underlines

- Default links should be text-colored or muted, not bright blue, unless external-link affordance is critical.
- Use `text-decoration-thickness: 1px`, `text-underline-offset: 0.16em` to `0.22em`, and underline only on hover/focus for dense operational UIs.
- For file paths, repository names, or row actions, use a button styled like a link rather than a large filled button.
- Never rely on color alone. Hover/focus should add underline, border change, or background change.

```css
a,
.linkButton {
  color: inherit;
  text-decoration: none;
  text-underline-offset: 0.18em;
  text-decoration-thickness: 1px;
}

a:hover,
a:focus-visible,
.linkButton:hover,
.linkButton:focus-visible {
  text-decoration: underline;
}
```

## Layout

- App shell: sticky or fixed-height topbar, constrained `main` width around `1180px` to `1280px`, `clamp(1rem, 2vw, 2rem)` page padding.
- Main screens: section header with title, concise description, right-aligned primary actions or stats.
- Data panels: use grid or flex rows with stable columns for selection, title/body, and metadata/actions.
- Bulk actions: use a side drawer or bottom panel. Do not scatter bulk controls across every row.
- Detail views: use modals or side panels for records, markdown, files, logs, and settings.
- Responsive rule: controls wrap into rows on tablet and stack on mobile while preserving primary action visibility.

## Cards And Panels

- Cards are functional repeated items or framed tools, not decorative page sections.
- Radius: 3-6px. Use 8px only for large modals if needed.
- Border: `1px solid var(--border)`. Hover may lift by `translateY(-1px)` with subtle shadow in light theme only.
- Avoid nested cards. Use dividers, soft panels, or table rows inside a card.

```css
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
}

.card:hover {
  border-color: var(--border-strong);
}
```

## Controls

- Minimum touch target: `44px` for buttons, inputs, selects.
- Buttons: small radius, medium weight, clear border, compact icon slot.
- Primary buttons: solid ink background in light theme and inverted solid in dark theme.
- Danger actions: outlined or text-danger by default; only fill red on hover or confirmation.
- Segmented controls: use for views, filters, ranges, and modes.
- Inputs/selects: neutral surface, visible border, focus ring using `--focus`.
- When compacting a scoped action group, reduce its font, padding, icon gap, and minimum width together. Keep adjacent inputs and submit buttons exactly the same height, and do not change the global control size for a page-specific density adjustment.
- Inline metadata editors should stay one row when space allows: text input first, primary/secondary actions directly to the right, and compact counters inside or adjacent to the input. Enforce copy limits in both the UI and backend.

```css
button,
input,
select,
textarea {
  min-height: 44px;
  font-family: var(--sans);
  font-size: 0.95rem;
}

button {
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--control-bg);
  color: var(--ink);
  font-weight: 500;
  padding: 0.58rem 0.95rem;
  transition: background var(--motion), color var(--motion), border-color var(--motion), transform var(--motion);
}

button:hover {
  background: var(--surface-hover);
  border-color: var(--ink);
  transform: translateY(-1px);
}

button.primary,
button.active {
  background: var(--solid-bg);
  border-color: var(--solid-bg);
  color: var(--solid-text);
}

button:focus-visible,
input:focus-visible,
select:focus-visible,
textarea:focus-visible {
  outline: 3px solid var(--focus);
  outline-offset: 2px;
}
```

## Icons

- Prefer lucide icons when available. If a project already has an icon library, use that one consistently.
- Use 16-20px icons in controls; keep stroke around `1.75` to `2`.
- For icon+text buttons, reserve a stable icon slot so labels do not shift.
- Icon-only buttons need `aria-label` and visible hover/focus.
- Avoid manually drawn SVGs except for logos, diagrams, or missing domain-specific icons.

```css
.buttonContent {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  line-height: 1.2;
  white-space: nowrap;
}

.buttonIconSlot {
  display: inline-flex;
  width: 1.16em;
  height: 1.16em;
  flex: 0 0 1.16em;
}
```

## Logos And Brand Marks

- Use a compact wordmark or monogram aligned with the operational UI. The brand should be visible in the first viewport but not oversized.
- Preferred logo treatments:
  - Serif wordmark with small mono version suffix.
  - Square monogram with 1px border and ink/surface contrast.
  - Simple geometric mark made from two or three strokes/shapes.
- Avoid cartoon mascots, gradient blobs, complex SVG illustrations, and glossy product-marketing logos.
- Logo text weight: 700 for compact wordmarks, 500-600 for utility headers.
- Keep logo color within `--ink`, `--solid-bg`, or a single restrained accent.

## Status, Tags, And Badges

- Use compact pills with text plus optional dot/icon.
- Status colors must be semantic: success, warning, danger, neutral.
- Badges should not dominate row titles; keep font size `0.72rem` to `0.82rem`.
- Group multiple markers tightly near the object title or right metadata area.
- For active relationships that can be removed directly, such as enabled agents or applied filters, prefer tag-like removable chips with a small `x` inside the chip over a separate full-size destructive button.

## Tables, Lists, And Rankings

- Prefer scan-friendly rows over decorative tiles for operational data.
- Put primary name/title left, technical path/ID in mono beneath or adjacent, status/counts/actions right.
- For compact object cards, place a muted one-line description between the title and technical path. Use ellipsis truncation and preserve the complete text in a tooltip or detail view.
- Rankings use a small rank column, strong title, muted metadata, and compact count pill.
- Long values should wrap in controlled columns or truncate with title attributes when exact content is available elsewhere.

## Modals, Drawers, Toasts

- Modal overlay uses `--overlay-bg`; modal surface uses `--surface`; radius stays small.
- Modal header: title plus muted mono/path subtitle when useful, close button right.
- Destructive confirmations should use `role="alertdialog"`, separate what will be removed from what will be preserved, focus the safe cancel action by default, and prevent dismissal while the operation is running. On mobile, keep the dialog content-height instead of stretching it to the viewport.
- Drawers/panels should be anchored and information-dense; avoid oversized empty padding.
- Toasts should be short, dismissible, and recorded only if the app has a history affordance.

## Motion

- Use 120-200ms transitions for hover, focus, modal entry, toasts, and progress.
- Avoid large page choreography. This style should feel responsive, not animated.
- Include:

```css
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
  }
}
```

## Copy

- Labels should be concrete nouns or verbs: `Sync`, `Check`, `Open`, `Enable`, `Export`, `Status`, `Source`, `Run`, `Path`.
- Section descriptions are allowed, but keep them one short sentence.
- Do not add visible text explaining basic UI mechanics, keyboard shortcuts, style choices, or feature lists.
- Empty states should say what is missing and the next useful action.

## Review Checklist

- Fonts load or have acceptable fallbacks.
- Body, controls, mono text, and headings use the intended weights.
- Underlines are visible on hover/focus and not noisy by default.
- Light and dark palettes are not one-note purple/blue/cream/brown.
- Icons are consistent in size and stroke.
- Logo is compact, legible, and not decorative clutter.
- Cards are not nested and sections are not floating card stacks.
- Text fits in buttons, badges, cards, rows, and mobile layouts.
- Technical paths/IDs do not break the layout.
- Focus states are visible.
- Build/typecheck passes.
