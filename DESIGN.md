---
name: tenda-n300
description: A living man page for a terminal tool that controls a Tenda N300 router
colors:
  paper: "#f5f1e8"
  paper-deep: "#ece6d9"
  ink: "#1a1917"
  ink-soft: "#55524b"
  rule: "#d8d0c0"
  rule-dark: "#a89e8a"
  led-green: "#0e7a3c"
  led-red: "#c2251d"
  term-bg: "#161616"
  term-ink: "#d9d6cf"
  term-dim: "#8f8b83"
  term-green: "#6fce8a"
  term-red: "#ff6b61"
  term-cyan: "#6cc7d9"
typography:
  body:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.55
  section-head:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "1.05rem"
    fontWeight: 700
    lineHeight: 1.3
  term:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "0.88rem"
    fontWeight: 400
    lineHeight: 1.55
  manual-meta:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "0.8rem"
    fontWeight: 400
    letterSpacing: "0.02em"
  table:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "0.92rem"
    fontWeight: 400
    lineHeight: 1.55
  label:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "0.72rem"
    fontWeight: 400
    letterSpacing: "0.06em"
  micro:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "0.78rem"
    fontWeight: 400
    lineHeight: 1.55
rounded:
  sm: "4px"
  md: "12px"
spacing:
  measure: "46rem"
  section-top: "3.25rem"
  term-pad: "1rem 1.1rem 1.1rem"
components:
  term-window:
    backgroundColor: "{colors.term-bg}"
    textColor: "{colors.term-ink}"
    rounded: "{rounded.md}"
  explorer-command:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
  copy-button:
    backgroundColor: "transparent"
    textColor: "{colors.term-ink}"
    rounded: "{rounded.sm}"
    padding: "0.35rem 0.7rem"
---

# Design System: tenda-n300

## Overview

**Creative North Star: "The Router Manual on Paper"**

This site is not a website about a command-line tool; it is the tool's man page, typeset as a printed Unix manual. A homelabber landing on it should feel they have already opened `man tenda-n300` — the grammar of the manual (NAME, SYNOPSIS, OPTIONS, FILES, SEE ALSO, the double-rule header and footer) *is* the design. Where the manual talks about running the tool, the page becomes the tool: output is rendered as a real dark terminal block, not described.

The surface is warm paper with black ink, IBM Plex Mono throughout, and hairline rules. The only color is router-LED status: green for online/allowed, red for blocked/danger — used where live network state is shown, never as decoration. Density is high and steady, like a real manual page; the scroll is long, and the only interactive devices are a copy button, a command explorer, and a less-style pager that reveals on scroll.

**Key Characteristics:**
- The whole page reads as one continuous man page, from header rule to footer rule.
- Terminal output is dark blocks on warm paper — the only dark surfaces, and they mean "the tool speaking."
- Mono type at every scale; hierarchy comes from weight and size, never a second family.
- Status color is LED language: green and red only, and only on live output.

## Colors

A warm-paper document with black ink; status color speaks in LED green and red, confined to terminal output.

### Primary
- **LED Green** (#0e7a3c): online / allowed / connected status. Also the focus ring and the "copy" success state. Never a background.
- **LED Red** (#c2251d): blocked devices and danger in terminal output. Never a background.

### Neutral
- **Warm Paper** (#f5f1e8): the document ground. All body copy sits on it.
- **Deep Paper** (#ece6d9): inline-code and synopsis fills, and the explorer's nav rail.
- **Ink** (#1a1917): body text and reverse-video section controls.
- **Soft Ink** (#55524b): secondary text, man-page header/footer rules, muted list markers.
- **Hairline** (#d8d0c0): 1px dividers between options, borders on deep-paper fills.
- **Dark Hairline** (#a89e8a): the header/footer double-rule, the explorer frame.

### Terminal palette (dark blocks only)
- **Terminal Black** (#161616): the terminal window ground.
- **Terminal Ink** (#d9d6cf): terminal body text and prompts.
- **Terminal Dim** (#8f8b83): terminal window chrome (title bar, LED at rest).
- **Terminal Green** (#6fce8a): the prompt `$` and the lit LED.
- **Terminal Red** (#ff6b61): blocked-device rows and danger output.
- **Terminal Cyan** (#6cc7d9): user-typed input echoed in transcripts.

### Named Rules
**The Two-LED Rule.** Green and red are the only chromatic colors on paper, they mean network status, and they never leave the terminal blocks. If a region is not showing the tool speaking, it is ink on paper.

## Typography

**Display Font:** IBM Plex Mono (self-hosted woff2)
**Body Font:** IBM Plex Mono (with ui-monospace, SFMono-Regular, Menlo, Consolas fallbacks)

**Character:** The entire page is one monospace family — the manual is set in typewriter faces, and no second family interrupts that. Hierarchy is carried by weight (400 → 700), size (0.72rem labels → 1.05rem section heads → body), and rules, not by face change.

### Hierarchy
- **Section head** (700, 1.05rem, 1.3): man-page `.SH` headings — NAME, SYNOPSIS, INSTALLATION, etc. Bold, uppercase-by-convention, tight.
- **Body** (400, 16px/1.55, measure ~46rem): the manual paragraphs, max width 46rem so terminal blocks and tables fit.
- **Terminal** (400, 0.88rem, 1.55): all terminal block content, white-space pre.
- **Table / Synopsis** (400, 0.92rem, 1.55): option rows, synopsis, explorer descriptions, inline code.
- **Manual meta** (400, 0.8rem, 0.02em tracking): the man-page header/footer rules (General Commands Manual, version/date).
- **Label** (400, 0.72rem, 0.06em tracking): terminal title bar and status chrome.
- **Micro** (400, 0.78rem, 1.55): the less-style pager and the copy button.
- **Command tab** (400, 0.82rem): explorer command labels.
- **Mobile body** (400, 15px, 1.55): below 720px the body steps down one notch.

### Named Rules
**The Typewriter Rule.** Nothing on the page leaves IBM Plex Mono. No display serif, no grotesque, no system sans. A second face anywhere is a broken promise of the manual.

## Layout

A single centered paper column of `46rem` (max-width), `1.25rem` side padding on the viewport, generous `3.25rem` above each section heading. The man-page header and footer rules (`tenda-n300(1)  ·  General Commands Manual  ·  tenda-n300(1)`) frame the document; the footer carries the version and date.

- Option lists are hanging-indent rows (`--ip <addr>` bold on the left, description in soft ink to the right, hairline between rows).
- The command explorer is a tab rail (deep paper) over a stage: the clicked command inverts to ink-on-paper, its synopsis renders, and its transcript types into a terminal block below.
- The `less`-style pager is fixed to the viewport bottom, hidden at the top of the page, revealed after scrolling; it carries section jumps on the left and a scroll position percent on the right.

**Responsive:** Below 720px the option rows collapse to single column, the middle "General Commands Manual" header span hides, the pager drops its jump rail (position percent only), and the copy button sits under the install command.

## Elevation & Depth

Depth is reserved for the terminal block — the one place the page "lifts" off the paper. Paper surfaces are flat; the explorer is flat with a dark hairline frame; the pager and the inverse command tab are flat color swaps.

### Shadow Vocabulary
- **Terminal lift** (`0 1px 0 rgba(0,0,0,0.15), 0 8px 24px -12px rgba(0,0,0,0.5)`): under terminal windows, so output reads as a physical device sitting on the desk.

### Named Rules
**The Flat-Paper Rule.** Paper regions never cast shadows. Only a terminal block may lift, and only with the single documented shadow. A soft shadow anywhere else is decoration.

## Shapes

Radii speak in two notes only: `12px` for terminal windows and the explorer frame (the only large surfaces), and `4px` for small controls (copy button, inline code). Corners are the quiet rounding of a physical device; there are no pills, no hard offsets, no clipped corners.

- **Terminal window** — 12px radius, dark fill, 1px darker chrome bar on top with a status LED (dim at rest, green glow when on).
- **Copy button** — 4px radius, transparent fill, hairline border, uppercase micro label that inverts to green on success.
- **Explorer tab (pressed)** — flat ink fill, paper text, no border; the rail it sits in is deep paper.

## Components

### Terminal Window
A dark block, the tool speaking. Dark fill (#161616), ink text, a title bar with a dim-or-green LED, `12px` radius, the single lift shadow. Body is `pre` text in Terminal Mono (0.88rem). Prompt `$` is terminal green; echoes are cyan; blocked rows are red.

### Copy Button
Micro uppercase label on a hairline square. Hover: border and text go terminal green. Success: label swaps to "copied" and holds green ~1.6s. Keyboard focus shows the green ring.

### Command Explorer
A tab rail of every command (`status`, `devices`, `block`, `wifi`, `--json`, ...). Rest: paper-on-deep-paper, hairline dividers. Pressed: full ink fill, paper text. Clicking types that command's real-format transcript into the terminal below; `aria-pressed` tracks state and the synopsis is announced politely.

### The Pager
A fixed bottom bar in terminal black with the same chrome as a terminal title bar. Hidden at page top, it slides up after scroll, lists jump targets (hover goes green), and reports scroll percent in tabular digits. On mobile the jump rail hides and only the position remains.

### The Install Block
The conversion point: a terminal window whose single line is the `curl | bash` one-liner with a copy button docked to its right. It sits high in the document, immediately under SYNOPSIS, so the manual's first actionable moment is also its first demo.

## Do's and Don'ts

### Do:
- **Do** keep every surface on warm paper with ink text unless it is a terminal block.
- **Do** use green and red only to mean network state, and only inside terminal output (plus the focus ring and copy success).
- **Do** render the tool's output as terminal blocks — author transcripts in the real format and label sample data as such.
- **Do** set section heads as bold `.SH`-style manual heads; the man-page grammar is the identity.
- **Do** reserve the lift shadow for terminal windows alone.

### Don't:
- **Don't** add a second typeface — the page is one mono family end to end.
- **Don't** put status color on the paper or on chrome; LEDs live on dark terminal glass.
- **Don't** use gradient text, glass, glyph icons, or emoji — iconography is the LED and the `$` prompt.
- **Don't** use cards, hero metrics, or kicker labels — the manual has no such scaffolds.
- **Don't** let an em dash stand in for the man-page colon cadence; the manual speaks in colons and hyphens.
