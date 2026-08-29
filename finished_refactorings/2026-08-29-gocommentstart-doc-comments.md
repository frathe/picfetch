# GoCommentStart Doc-Comment Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the `GoCommentStart` backlog item by making all 19 flagged doc
comments start with the element they document, and — by the user's scope
decision — normalize the six unflagged doc comments that use the same
slash-joined shape, so the tree reads one way.

**Architecture:** Comment text only. No declaration is renamed, moved, added,
or removed, with one exception: the `internal/ui/spiral` block comment is
split so that the paragraph describing `Show` moves down onto `func Show`,
which today has no doc comment at all.

**Tech Stack:** Go 1.27.0 toolchain on a Go 1.26.7 module. Verification is
GoLand's inspection engine, reachable from the parent session through the
`goland` MCP server (`lint_files` / `get_file_problems`) — the same engine
Qodana runs, so no `QODANA_TOKEN` and no CI round-trip is needed.

**Spec:** `todos.md`, section
`Qodana: 19 doc comments don't start with the element they document`.

## The Inspection Rule, Established Empirically

The rule was probed with a throwaway file (`internal/uitest/zzprobe.go`,
written, linted, and deleted during planning). GoLand's own message states the
required form: `Comment should have the following format 'X ...' (with an
optional leading article)`.

Measured outcomes:

| Comment opening | Result |
|---|---|
| `// ProbeA and ProbeB do a thing.` | clean |
| `// ProbeC together with ProbeD does a thing.` | clean |
| `// ProbeD reports a thing. ProbeE reports another thing.` | clean |
| `// The ProbeE value is a thing.` | clean |
| `// ProbeF and ProbeG bound a thing.` (inside a `const (` block) | clean |
| `// ProbeB, ProbeC, and ProbeD do a thing.` | **flagged** |
| `// ProbeH: does a thing.` | **flagged** |

**The rule:** the first token must be exactly the documented element's name
followed by whitespace, optionally preceded by `A`, `An`, or `The`. Any
punctuation glued to the name — `/`, `,`, `:` — breaks it. This is why nine
comments that already *begin* with the right name are still flagged.

A second, separate message covers file-level comments:
`Package comment should be of the form 'Package ui ...'`. It fires on any
comment block attached directly to a `package` clause. Inserting one blank
line between the block and the clause detaches it and clears the warning —
verified on `internal/ui/build.go` (linted clean, then reverted), and
confirmed to survive `goimports -local github.com/frathe/picfetch`.

That blank-line form is already the dominant house style: **69 files in the
tree** carry a detached file-level comment above the package clause,
including `internal/ui/load.go`, `menu.go`, `keys.go`, and `export.go` in the
very same package. The five offenders are the outliers, not the convention.
`internal/ui`'s real package doc lives at `internal/ui/run.go:1`.

## Global Constraints

- Read `AGENTS.md` and `ARCHITECTURE.md` before editing.
- **Comments only.** No signature, body, name, or declaration order changes.
  The single structural edit is relocating one comment block in
  `spiral.go` (Task 7).
- Do not add, remove, or reorder declarations.
- Do not touch `qodana.yaml`. This item is fixed in source, matching how the
  earlier suppression work was done.
- Do not add `//goland:noinspection` anywhere. Every one of these findings is
  correct; they get fixed, not suppressed.
- Wrap comment paragraphs at **≤76 columns**, counting a leading tab as one
  character — the width every touched paragraph already uses. Reflow only
  paragraphs you are otherwise editing; leave untouched paragraphs alone even
  if they are longer.
- Preserve every fact in the existing prose. These comments carry real
  reasoning; the edit changes how a sentence opens, not what it says.
- `-` is the house dash in these comments (not `—`). Keep it.
- No `git commit` (`AGENTS.md`). The parent session ends with a suggested
  message.

## Current State and Baseline

`git status` is clean except for untracked plan files. Baseline captured from
the GoLand MCP inspection during planning — all 19 flagged sites, with the
exact form each one demands:

| # | Site | Demanded form |
|---|---|---|
| 1 | `internal/ui/build.go:1` | `Package ui ...` |
| 2 | `internal/ui/components.go:1` | `Package ui ...` |
| 3 | `internal/ui/favthumbs.go:1` | `Package ui ...` |
| 4 | `internal/ui/shortcuts.go:1` | `Package ui ...` |
| 5 | `internal/ui/startup.go:1` | `Package ui ...` |
| 6 | `internal/ui/autoupdate.go:10` | `CheckForUpdates ...` |
| 7 | `internal/ui/autoupdate.go:25` | `LastUpdateCheckDay ...` |
| 8 | `internal/ui/favthumbs.go:18` | `FavoritePreviewCache ...` |
| 9 | `internal/ui/load.go:516` | `MaxWindowWidth ...` |
| 10 | `internal/ui/load.go:521` | `SetMaxWindowWidth ...` |
| 11 | `internal/ui/memlimits.go:86` | `MaxImageCacheMB ...` |
| 12 | `internal/ui/autoupdate/updater.go:71` | `Dir ...` |
| 13 | `internal/ui/autoupdate/updater.go:78` | `Client ...` |
| 14 | `internal/ui/autoupdate/updater.go:119` | `LastCheckDay ...` |
| 15 | `internal/ui/spiral/spiral.go:130` | `ShowForGesture ...` |
| 16 | `internal/ui/widgets/style.go:29` | `DropzoneBorderWidth ...` |
| 17 | `internal/ui/widgets/style.go:42` | `ButtonRingWidth ...` |
| 18 | `internal/ui/widgets/style.go:66` | `ToastBGColor ...` |
| 19 | `internal/ui/widgets/tappable.go:49` | `MouseIn ...` |

Plus six unflagged sites pulled in by the user's scope decision (the
inspection does not check struct fields or mid-comment references, so these
are consistency, not findings):

| # | Site | Why |
|---|---|---|
| 20 | `internal/ui/load.go:506` | opens the same comment block as 510 |
| 21 | `internal/ui/load.go:510` | cross-reference, five lines above site 9 |
| 22 | `internal/preferences/preferences.go:88` | exported struct field |
| 23 | `internal/preferences/preferences.go:96` | exported struct field |
| 24 | `internal/preferences/preferences.go:109` | exported struct field |
| 25 | `internal/preferences/preferences.go:118` | exported struct field |

Total: **25 sites, 13 files.**

`internal/ui/memlimits.go` also reports an unrelated
`Struct 'settings' might be suboptimal: 8 bytes wasted (~12%)`. Out of scope;
leave it.

## Resolved Decisions

Answered by the user during planning:

1. **`tappable.go:49`** — `MouseIn implements desktop.Hoverable, as do
   MouseMoved and MouseOut below.` Rejected: three separate comments;
   the `and the ... below` phrasing.
2. **`memlimits.go:86`** — distribute the verb across all three names, so
   the doc says more than it does today. Rejected: an `and`-chain; dropping
   two names.
3. **`spiral.go:130`** — split the block. `Show` gets its own doc comment.
   Rejected: reordering in place, which would silence the inspection but
   leave `go doc Spiral.Show` empty.
4. **Scope** — normalize the unflagged slash-joined comments too
   (`load.go`, `preferences.go`). Sites 20–25 above.

Not asked, decided from evidence in the tree:

5. **Package comments get a blank line**, not a rewrite to `Package ui ...`.
   Five files cannot each be the package doc, and 69 files already use the
   detached form.
6. **`/` becomes ` and `** for two-name pairs. `internal/ui/zoom/zoom.go:317`
   (`In and Out are the +/- keys`), `widgets/choicepanel.go:310`,
   `favorites/manage.go:139`, `imaging/svg.go:20`, and `imaging/vector.go:18`
   already document pairs exactly that way.
7. **Mid-comment `X/Y` stays.** `Cmd/Ctrl+S`, `Left/Right/Home/End`,
   `maxWinW/maxWinH`, `startW/startH`, `getter/setter` are key chords,
   shorthand, or continuation text — not comment-leading element names. The
   scope boundary is: a slash-joined pair that opens a doc comment.

## File Map

| File | Sites | Task |
|---|---|---|
| `internal/ui/build.go` | 1 | 1 |
| `internal/ui/components.go` | 1 | 1 |
| `internal/ui/shortcuts.go` | 1 | 1 |
| `internal/ui/startup.go` | 1 | 1 |
| `internal/ui/favthumbs.go` | 2 | 1 |
| `internal/ui/autoupdate.go` | 2 | 2 |
| `internal/ui/load.go` | 4 | 2 |
| `internal/ui/autoupdate/updater.go` | 3 | 3 |
| `internal/preferences/preferences.go` | 4 | 4 |
| `internal/ui/memlimits.go` | 1 | 5 |
| `internal/ui/widgets/style.go` | 3 | 6 |
| `internal/ui/widgets/tappable.go` | 1 | 6 |
| `internal/ui/spiral/spiral.go` | 1 | 7 |

No file appears in two tasks, so every task inside a step can run in
parallel without edit conflicts.

## Subagent Routing

All tasks go to the `go-expert` agent (`Read`/`Edit`/`Write`/`Grep`/`Glob`/
`Bash`), which is the repo's Go specialist and already knows the conventions.
Model is overridden per task by how much judgment the edit needs.

| Task | Model | Rationale |
|---|---|---|
| 1 | `haiku` | Insert a blank line; one supplied text swap. Zero judgment. |
| 2 | `haiku` | Four supplied text swaps, one short rewrap. |
| 3 | `haiku` | Three supplied text swaps. |
| 4 | `haiku` | Four supplied text swaps in one struct. |
| 5 | `haiku` | One comment, final text supplied verbatim. |
| 6 | `sonnet` | Three group comments must be re-authored to lead with a constant name while keeping the concept they explain; needs paragraph rewrapping. |
| 7 | `sonnet` | Relocates a 4-line paragraph across 30 lines of code onto a different declaration. Highest risk of collateral damage. |

Opus/Fable are not needed: no task requires cross-file reasoning or design.

**Step 1** dispatches Tasks 1–5 in parallel (five `haiku` agents).
**Step 2** dispatches Tasks 6–7 in parallel (two `sonnet` agents).
**Step 3** is the parent session's verification and backlog close.

The parent session waits for the user's go-signal before each step.

---

## Task 1: Detach the Five File-Level Comments

**Subagent:** `go-expert`, model `haiku`

**Files:**
- `internal/ui/build.go`
- `internal/ui/components.go`
- `internal/ui/favthumbs.go`
- `internal/ui/shortcuts.go`
- `internal/ui/startup.go`

**Steps:**

- [ ] In each of the five files, insert exactly one blank line between the
      final `//` line of the leading comment block and the `package ui` line.
      Change no comment text.
- [ ] In `internal/ui/favthumbs.go` only, replace line 18's opening:

      ```
      // FavoritePreviewCache/SetFavoritePreviewCache are the settings window's
      // getter/setter pair for the preference, the same shape memlimits.go uses
      // for the three memory limits.
      ```

      with:

      ```
      // FavoritePreviewCache and SetFavoritePreviewCache are the settings
      // window's getter/setter pair for the preference, the same shape
      // memlimits.go uses for the three memory limits.
      ```

      The `getter/setter` inside the sentence stays as is.

**Verification:**

- [ ] `gofmt -l internal/ui/build.go internal/ui/components.go internal/ui/favthumbs.go internal/ui/shortcuts.go internal/ui/startup.go` prints nothing.
- [ ] `go build ./...` succeeds.
- [ ] `git diff -U0 -- internal/ui/build.go internal/ui/components.go internal/ui/shortcuts.go internal/ui/startup.go | grep -c '^[+-][^+-]'` reports `0` — an added blank line is a bare `+` with nothing after it, so any match here means text was changed in a file that should only have gained a blank line.
- [ ] `grep -c '^// Package ui' internal/ui/run.go` reports `1` — the real package doc is untouched.

---

## Task 2: Normalize `autoupdate.go` and `load.go`

**Subagent:** `go-expert`, model `haiku`

**Files:**
- `internal/ui/autoupdate.go`
- `internal/ui/load.go`

**Steps:**

- [ ] `internal/ui/autoupdate.go:10` — replace:

      ```
      // CheckForUpdates/SetCheckForUpdates are the settings window's getter/setter
      // pair for the opt-in updates preference. Turning the setting on starts a
      // check when due; turning it off cancels an in-flight check but leaves an
      // already-complete stage on disk for apply-on-stop.
      ```

      with:

      ```
      // CheckForUpdates and SetCheckForUpdates are the settings window's
      // getter/setter pair for the opt-in updates preference. Turning the
      // setting on starts a check when due; turning it off cancels an
      // in-flight check but leaves an already-complete stage on disk for
      // apply-on-stop.
      ```

- [ ] `internal/ui/autoupdate.go:25` — replace:

      ```
      // LastUpdateCheckDay/SetLastUpdateCheckDay round-trip the local calendar
      // day (YYYY-MM-DD) of the last update check, or empty when none has run
      // yet - storage and persistence live on v.updater, see
      // internal/ui/autoupdate.Updater.
      ```

      with:

      ```
      // LastUpdateCheckDay and SetLastUpdateCheckDay round-trip the local
      // calendar day (YYYY-MM-DD) of the last update check, or empty when
      // none has run yet - storage and persistence live on v.updater, see
      // internal/ui/autoupdate.Updater.
      ```

- [ ] `internal/ui/load.go:506` — replace:

      ```
      // defaultMaxWindowWidth/defaultMaxWindowHeight cap how large the window is
      // ever allowed to auto-grow to fit a loaded image, until the settings
      // window (internal/ui/settingswin) changes them - see the viewer's
      // settings.maxWinW/maxWinH fields (memlimits.go) and
      // MaxWindowWidth/MaxWindowHeight below.
      ```

      with:

      ```
      // defaultMaxWindowWidth and defaultMaxWindowHeight cap how large the
      // window is ever allowed to auto-grow to fit a loaded image, until the
      // settings window (internal/ui/settingswin) changes them - see the
      // viewer's settings.maxWinW/maxWinH fields (memlimits.go) and
      // MaxWindowWidth and MaxWindowHeight below.
      ```

      `settings.maxWinW/maxWinH` is field shorthand and stays.

- [ ] `internal/ui/load.go:516` — replace:

      ```
      // MaxWindowWidth/MaxWindowHeight report the current window-size cap - the
      // settings window's getters.
      ```

      with:

      ```
      // MaxWindowWidth and MaxWindowHeight report the current window-size
      // cap - the settings window's getters.
      ```

- [ ] `internal/ui/load.go:521` — replace only the first two lines:

      ```
      // SetMaxWindowWidth/SetMaxWindowHeight set the window-size cap directly -
      // the settings window's binding. Floored at the drop-zone size
      ```

      with:

      ```
      // SetMaxWindowWidth and SetMaxWindowHeight set the window-size cap
      // directly - the settings window's binding. Floored at the drop-zone
      // size
      ```

      then rewrap the rest of that paragraph at ≤76 columns. `(startW/startH)`
      later in the paragraph stays as is.

**Verification:**

- [ ] `gofmt -l internal/ui/autoupdate.go internal/ui/load.go` prints nothing.
- [ ] `go build ./...` succeeds.
- [ ] `git diff -U0 -- internal/ui/autoupdate.go internal/ui/load.go | grep '^[+-][^+-]' | grep -vc '^[+-]\s*//'` reports `0` — every changed line is a comment line.
- [ ] `git diff -U0 -- internal/ui/autoupdate.go internal/ui/load.go | grep '^+' | grep -v '^+++' | awk '{ if (length($0) - 1 > 76) print }'` prints nothing.

---

## Task 3: Normalize `autoupdate/updater.go`

**Subagent:** `go-expert`, model `haiku`

**Files:**
- `internal/ui/autoupdate/updater.go`

**Steps:**

- [ ] Line 71 — replace:

      ```
      // Dir/SetDir round-trip the staged-update directory. SetDir exists for
      ```

      with:

      ```
      // Dir and SetDir round-trip the staged-update directory. SetDir exists
      // for
      ```

      then rewrap the rest of that paragraph at ≤76 columns.

- [ ] Line 78 — replace:

      ```
      // Client/SetClient round-trip the GitHub Releases client. nil until the
      ```

      with:

      ```
      // Client and SetClient round-trip the GitHub Releases client. nil until
      // the
      ```

      then rewrap the rest of that paragraph at ≤76 columns. `Check/Download`
      later in the paragraph stays as is.

- [ ] Line 119 — replace:

      ```
      // LastCheckDay/SetLastCheckDay round-trip the local calendar day
      // (YYYY-MM-DD) of the last update check, or empty when none has run yet.
      ```

      with:

      ```
      // LastCheckDay and SetLastCheckDay round-trip the local calendar day
      // (YYYY-MM-DD) of the last update check, or empty when none has run
      // yet.
      ```

- [ ] Leave the `New` and `Done` comments alone; they already pass.

**Verification:**

- [ ] `gofmt -l internal/ui/autoupdate/updater.go` prints nothing.
- [ ] `go build ./...` succeeds.
- [ ] `git diff -U0 -- internal/ui/autoupdate/updater.go | grep '^[+-][^+-]' | grep -vc '^[+-]\s*//'` reports `0`.
- [ ] `git diff -U0 -- internal/ui/autoupdate/updater.go | grep '^+' | grep -v '^+++' | awk '{ if (length($0) - 1 > 76) print }'` prints nothing.

---

## Task 4: Normalize `preferences.go` Struct-Field Comments

**Subagent:** `go-expert`, model `haiku`

**Files:**
- `internal/preferences/preferences.go`

These four are not flagged by the inspection — struct fields are outside its
scope. They are in the change by the user's explicit consistency decision.
Each line carries a leading tab; keep it.

**Steps:**

- [ ] Line 88 — `// MaxWindowWidth/MaxWindowHeight cap how large the window is ever`
      becomes `// MaxWindowWidth and MaxWindowHeight cap how large the window is`,
      then rewrap the rest of that paragraph at ≤76 columns.
      `maxWinW/maxWinH` later in the paragraph stays.

- [ ] Line 96 — `// MaxImageCacheMB/MaxThumbCacheMB are the byte budgets, in megabytes,`
      becomes `// MaxImageCacheMB and MaxThumbCacheMB are the byte budgets, in`,
      then rewrap the rest of that paragraph at ≤76 columns.

- [ ] Line 109 — `// WindowPosX/WindowPosY are the on-screen position (see`
      becomes `// WindowPosX and WindowPosY are the on-screen position (see`.
      No rewrap needed.

- [ ] Line 118 — `// SettingsWindow/ExifWindow are where those two secondary windows`
      becomes `// SettingsWindow and ExifWindow are where those two secondary`,
      then rewrap the rest of that paragraph at ≤76 columns.

**Verification:**

- [ ] `gofmt -l internal/preferences/preferences.go` prints nothing.
- [ ] `go build ./...` succeeds.
- [ ] `go test ./internal/preferences/...` passes.
- [ ] `git diff -U0 -- internal/preferences/preferences.go | grep '^[+-][^+-]' | grep -vc '^[+-]\s*//'` reports `0`.
- [ ] The existing `//goland:noinspection GoRedundantConversion` comment in this
      file is untouched (`grep -c goland:noinspection` still reports `1`).

---

## Task 5: Distribute the Verb in `memlimits.go`

**Subagent:** `go-expert`, model `haiku`

**Files:**
- `internal/ui/memlimits.go`

**Steps:**

- [ ] Line 86 — replace:

      ```
      // MaxImageCacheMB/MaxThumbCacheMB/MaxFileSizeMB report the current limits -
      // the settings window's getters.
      ```

      with exactly:

      ```
      // MaxImageCacheMB reports the decoded-image cache budget,
      // MaxThumbCacheMB the thumbnail cache's, and MaxFileSizeMB the
      // per-file encoded ceiling - the settings window's getters.
      ```

- [ ] Change nothing else. In particular, leave the `settings` struct at line
      29 alone — its field-padding warning is a separate, out-of-scope item.

**Verification:**

- [ ] `gofmt -l internal/ui/memlimits.go` prints nothing.
- [ ] `go build ./...` succeeds.
- [ ] `git diff --stat -- internal/ui/memlimits.go` shows 3 insertions, 2 deletions.

---

## Task 6: Re-author the `widgets` Group Comments

**Subagent:** `go-expert`, model `sonnet`

**Files:**
- `internal/ui/widgets/style.go`
- `internal/ui/widgets/tappable.go`

Three of these are group comments inside a `const`/`var` block. The
inspection attaches each one to the first name in its group and wants that
name first. The concept the comment explains has to survive the reordering —
these comments exist to say *why* the group is a group.

**Steps:**

- [ ] `style.go:29` — replace:

      ```
      	// The dropzone's rounded border box.
      ```

      with:

      ```
      	// DropzoneBorderWidth and DropzoneBorderRadius outline the dropzone's
      	// rounded border box.
      ```

- [ ] `style.go:42` — replace:

      ```
      	// Focus rings (see NewFocusRing): the delete-confirmation buttons use a
      	// thinner ring than the grid's cell highlight, whose stroke has to stay
      	// visible against a busy thumbnail behind it.
      ```

      with:

      ```
      	// ButtonRingWidth and GridRingWidth are the focus rings' stroke
      	// widths (see NewFocusRing), RingRadius their shared corner radius:
      	// the delete-confirmation buttons use a thinner ring than the grid's
      	// cell highlight, whose stroke has to stay visible against a busy
      	// thumbnail behind it.
      ```

- [ ] `style.go:66` — replace only the first line:

      ```
      	// The toast's fixed, deliberately loud warning colors: dark text for
      ```

      with:

      ```
      	// ToastBGColor and ToastTextColor are the toast's fixed, deliberately
      	// loud warning colors: dark text for
      ```

      then rewrap the rest of that paragraph at ≤76 columns. Keep the
      parenthetical about the info overlay intact.

- [ ] `tappable.go:49` — replace:

      ```
      // MouseIn, MouseMoved, and MouseOut implement desktop.Hoverable.
      ```

      with exactly:

      ```
      // MouseIn implements desktop.Hoverable, as do MouseMoved and
      // MouseOut below.
      ```

- [ ] Do not add doc comments to `DropzoneBorderColor` or `DropzoneHoverColor`,
      which are undocumented today. That is a different finding, not this one.

**Verification:**

- [ ] `gofmt -l internal/ui/widgets/style.go internal/ui/widgets/tappable.go` prints nothing.
- [ ] `go build ./...` succeeds.
- [ ] `go test ./internal/ui/widgets/...` passes.
- [ ] `git diff -U0 -- internal/ui/widgets/ | grep '^[+-][^+-]' | grep -vc '^[+-]\s*//'` reports `0`.
- [ ] `git diff -U0 -- internal/ui/widgets/ | grep '^+' | grep -v '^+++' | awk '{ if (length($0) - 1 > 76) print }'` prints nothing.
- [ ] Every constant named in the original comments still appears in the new
      ones: `grep -c 'NewFocusRing' internal/ui/widgets/style.go` is unchanged.

---

## Task 7: Split the `spiral.go` Block Comment

**Subagent:** `go-expert`, model `sonnet`

**Files:**
- `internal/ui/spiral/spiral.go`

One comment block at line 130 sits on `ShowForGesture` (line 149) but opens
by documenting `Show` (line 163), which has no doc comment of its own. The
split fixes both the inspection and the doc gap.

**Steps:**

- [ ] Move the first paragraph — the four lines beginning
      `// Show raises the window if it is already open,` and ending
      `// the one already up to the front rather than stack another on top of it.`
      — out of the block at line 130 and place it directly above
      `func (s *Spiral) Show() {`.

- [ ] Delete the now-orphaned `//` separator line that followed that
      paragraph, so the `ShowForGesture` block starts at
      `// ShowForGesture opens the spiral on the pattern the user's gesture asked`.

- [ ] Leave all three remaining `ShowForGesture` paragraphs — the gesture
      mapping, the "which direction maps to which pattern" note, and the
      uniform-writing note — verbatim, in order, still separated by `//`
      lines.

- [ ] Change no code. `func Open`, `func ShowForGesture`, and `func Show`
      keep their current order and bodies.

**Expected result:**

```go
// ShowForGesture opens the spiral on the pattern the user's gesture asked
// for, and is the window-drag gesture's way in (internal/wingesture, wired
// up in internal/ui/gesture.go): swirling the window clockwise brings up the
// Nautilus, counter-clockwise the Ripple. The manual's secret phrase goes on
// using plain Show, which opens whichever pattern the spiral was last left
// on - the same as the N key's own toggle.
//
// Which direction maps to which pattern is arbitrary and lives here rather
// than at the call site, since knowing what presets exist is this package's
// business and not internal/ui/help's.
//
// The uniform has to be written as well as the state whenever a window is
// already open: newShader seeds the uniforms from the state once, when the
// shader is built, so on an already-open spiral Show alone would raise the
// old window and change nothing.
func (s *Spiral) ShowForGesture(clockwise bool) {
	...
}

// Show raises the window if it is already open, or builds and shows a fresh
// full-screen one. Mirrors widgets.Singleton.Show's raise behaviour - the
// easter egg is a single window, and finding it a second time should bring
// the one already up to the front rather than stack another on top of it.
func (s *Spiral) Show() {
```

**Verification:**

- [ ] `gofmt -l internal/ui/spiral/spiral.go` prints nothing.
- [ ] `go build ./...` succeeds.
- [ ] `go test ./internal/ui/spiral/...` passes.
- [ ] `git diff -U0 -- internal/ui/spiral/spiral.go | grep '^[+-][^+-]' | grep -vc '^[+-]\s*//'` reports `0` — comment lines only.
- [ ] `go doc ./internal/ui/spiral Spiral.Show` now prints the raise-behaviour
      paragraph (it prints nothing today).
- [ ] `go doc ./internal/ui/spiral Spiral.ShowForGesture` opens with
      `ShowForGesture opens the spiral`.

---

## Parent Controller Protocol After Every Task

The parent session, not the subagent, owns verification. After each task
returns:

1. `git diff -- <the task's files>` read in full. Confirm comment-only.
2. `mcp__goland__lint_files` over the task's files. Confirm zero
   `Comment should have the following format` and zero
   `Package comment should be of the form` entries. Pre-existing unrelated
   warnings (the `settings` struct padding in `memlimits.go`) are expected
   and stay.
3. `gofmt -l` over the task's files, then `go build ./...`.
4. Fix up anything the subagent got wrong directly, rather than
   redispatching, unless the task needs redoing wholesale.
5. Report the result and wait for the user's go-signal before the next step.

## Final Repository Verification

Run from the repository root, after Step 2 is reviewed:

- [ ] `mcp__goland__lint_files` over all 13 touched files — zero
      `GoCommentStart` findings of either message shape.
- [ ] `git diff | grep '^[+-][^+-]' | grep -v '^[+-]\s*//' | grep -v '^[+-]\s*$'`
      prints nothing — the whole change is comments and blank lines.
- [ ] `make fmt-check`
- [ ] `go vet ./...`
- [ ] `go build ./...`
- [ ] `go test -timeout 20m -race ./...`
- [ ] `go doc ./internal/ui/spiral Spiral.Show` and
      `go doc ./internal/ui/spiral Spiral.ShowForGesture` both read correctly.
- [ ] `go doc ./internal/ui` still shows the `run.go` package doc and nothing
      from the five detached file comments.

## Backlog Close

- [ ] In `todos.md`, delete the
      `### Qodana: 19 doc comments don't start with the element they document`
      section from `## TODO`.
- [ ] Add an entry under `## Done` → `#### Internal` recording: the two
      message shapes, the blank-line detach as the fix for file-level
      comments (with the 69-file precedent), the `X and Y` form as the fix for
      slash-joined pairs, the `spiral.go` split that also gave `Show` its
      first doc comment, and the scope extension to the six unflagged
      `load.go`/`preferences.go` sites.
- [ ] Record the one durable, non-obvious fact for a future reader: the
      inspection requires the element name followed by **whitespace**, so
      `Name/Other`, `Name, Other`, and `Name:` all fail even though they lead
      with the right word.
- [ ] `ARCHITECTURE.md` needs no update — no package added, removed, renamed,
      or moved.
- [ ] Do not commit. Hand the user a suggested commit message.

## Out of Scope

- `internal/ui/memlimits.go:29` — the `settings` struct-padding warning.
- `DropzoneBorderColor` / `DropzoneHoverColor` in `style.go` — exported and
  undocumented, a different finding.
- Mid-comment `X/Y` shorthand: `Cmd/Ctrl+S`, `Left/Right/Home/End`,
  `maxWinW/maxWinH`, `startW/startH`, `getter/setter`, `Check/Download`,
  `CanvasObject/Hoverable`, and the same shape in test files
  (`windowsize_test.go:57`, `session_test.go:16`, `completion_test.go:180`,
  `openwith_test.go:15`). Not element-name comment openings.
- The remaining `todos.md` items: the mechanical `errors.Is` / named-field /
  shadowing fixes, the favorites native-menu-bar bug, the untested
  manual-opened observer, the `maybeStartUpdateCheck` ordering note, and the
  Qodana CI duplication under-reporting.
- `qodana.yaml`. Untouched.
