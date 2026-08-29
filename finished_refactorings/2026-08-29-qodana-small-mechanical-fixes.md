# Qodana Small Mechanical Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the `todos.md` item `Qodana: small mechanical fixes` — 16
findings across 8 files, in five unrelated inspection categories.

**Architecture:** Four of the five categories are local edits with no design
content. The fifth, the spiral FPS colours, is a small refactor by the user's
decision: the three backdrop colours move from literals duplicated between
production and test into package-level vars, with a new test pinning their
values so the existing threshold test does not become the only thing guarding
them.

**Tech Stack:** Go 1.27.0 toolchain on a Go 1.26.7 module. Verification is
GoLand's inspection engine through the `goland` MCP server (`lint_files`),
which is the same engine Qodana runs and needs no `QODANA_TOKEN`.

**Spec:** `todos.md`, section `Qodana: small mechanical fixes`.

## Baseline

Every finding below was confirmed by a GoLand `lint_files` run during
planning, with its exact inspection message.

| # | Site | Inspection message |
|---|---|---|
| 1 | `internal/imaging/save.go:183` | Comparison with errors using equality operators fails on wrapped errors |
| 2 | `internal/ui/exifwin/tiles_test.go:164` | (same) |
| 3 | `internal/ui/exifwin/tiles_test.go:218` | (same) |
| 4 | `internal/ui/exifwin/tiles_test.go:299` | (same) |
| 5 | `internal/ui/spiral/overlays.go:25` | Fields are assigned without explicit names |
| 6 | `internal/ui/spiral/overlays.go:169` | (same) |
| 7 | `internal/ui/spiral/overlays.go:171` | (same) |
| 8 | `internal/ui/spiral/overlays.go:173` | (same) |
| 9 | `internal/ui/spiral/overlays_test.go:66` | (same) |
| 10 | `internal/ui/spiral/overlays_test.go:67` | (same) |
| 11 | `internal/ui/spiral/overlays_test.go:68` | (same) |
| 12 | `internal/ui/spiral/overlays_test.go:69` | (same) |
| 13 | `internal/imaging/raw_test.go:463` | Redundant type conversion |
| 14 | `internal/update/apply_test.go:146` | Variable 'real' collides with the 'builtin' function |
| 15 | `scripts/releasenotes/main.go:18` | Unhandled error |
| 16 | `scripts/synctuf/main.go:22` | Unhandled error |

**16 findings, 8 files.**

### Scope boundary, established by a tree sweep

Two other sites compare `err == io.EOF` and are **not** part of this change,
because GoLand does not flag them:

- `internal/update/extract.go:105` — the `tar.Reader.Next` loop terminator.
- `scripts/plistdoctypes/doctypes_test.go:228` — the `xml.Decoder.Token` loop
  terminator.

Both are the documented stdlib iterator idiom, where the bare sentinel is
what the API promises. The inspection recognises those call shapes. Leave
them alone; do not "make them consistent".

A sweep for the other categories found no additional sites: `real` is the only
builtin-shadowing local in the tree, `raw_test.go:463` the only redundant
`copy` conversion, `releasenotes` and `synctuf` the only unhandled
`Fprintf`s, and all 8 positional `color.NRGBA` literals are the ones listed
above.

Two files carry unrelated pre-existing `DuplicatedCode` warnings
(`raw_test.go` at 275/318/348/374/395, `apply_test.go` at 17/154). Out of
scope; they must still be there when this is done.

## Resolved Decisions

Answered by the user during planning:

1. **`tiles_test.go`** — use `!errors.Is(err, errTilePending)`. It matches the
   24 existing `errors.Is` uses in the tree's tests and survives the package
   ever wrapping the sentinel. Rejected: keeping `!=` behind a
   `//goland:noinspection`. Accepted cost: a wrapped `errTilePending` would
   now also satisfy the assertion.
2. **Spiral FPS colours** — hoist the three to package-level vars and have the
   test reference them, removing the production/test literal duplication.
   Rejected: naming the fields in place and leaving the duplication.
3. **`apply_test.go`** — rename `real` to `target`, including the temp
   filename string, so the local and the file it names stay in step.
   Rejected: `realPath`, `realFile`.

Decided from evidence in the tree, not asked:

4. **`_, _ = fmt.Fprintf(...)`** for the two scripts.
   `scripts/plistdoctypes/main.go:29` is the identical `main()` shape and
   already writes it exactly that way, and `AGENTS.md` calls for marking
   intentionally ignored errors explicitly.

## Global Constraints

- Read `AGENTS.md` and `ARCHITECTURE.md` before editing.
- No behaviour changes anywhere. Every edit is either a
  semantics-preserving rewrite or a rename.
- Do not touch `qodana.yaml`, and add no `//goland:noinspection` anywhere.
  All 16 findings are correct and get fixed.
- Do not fix the pre-existing `DuplicatedCode` warnings in `raw_test.go` or
  `apply_test.go`.
- Keep import groups in the house order — stdlib, then `fyne.io`/third party,
  then `github.com/frathe/picfetch` — and run `make fmt` if an import moves.
- No `git commit` (`AGENTS.md`). The parent session ends with a suggested
  message.

## File Map

| File | Findings | Task |
|---|---|---|
| `internal/imaging/save.go` | 1 | 1 |
| `internal/imaging/raw_test.go` | 1 | 1 |
| `internal/ui/exifwin/tiles_test.go` | 3 | 2 |
| `internal/update/apply_test.go` | 1 | 3 |
| `scripts/releasenotes/main.go` | 1 | 4 |
| `scripts/synctuf/main.go` | 1 | 4 |
| `internal/ui/spiral/overlays.go` | 4 | 5 |
| `internal/ui/spiral/overlays_test.go` | 4 | 5 |

No file appears in two tasks.

## Subagent Routing

All tasks go to the `go-expert` agent (`Read`/`Edit`/`Write`/`Grep`/`Glob`/
`Bash`).

| Task | Model | Rationale |
|---|---|---|
| 1 | `haiku` | Two supplied one-line edits plus one import insertion. |
| 2 | `haiku` | Three supplied one-line edits; the import already exists. |
| 3 | `haiku` | A four-occurrence rename in one function. |
| 4 | `haiku` | Two identical supplied one-line edits. |
| 5 | `sonnet` | Introduces package vars, rewrites a branch, rewrites a test table, and adds a new test. The only task with judgment in it. |

Opus/Fable are not needed; nothing here requires cross-file design.

**Step 1** dispatches Tasks 1–4 in parallel (four `haiku` agents).
**Step 2** dispatches Task 5 (one `sonnet` agent).
**Step 3** is the parent session's verification and backlog close.

The parent session waits for the user's go-signal before each step.

---

## Task 1: `errors.Is` in `save.go`, Drop a Conversion in `raw_test.go`

**Subagent:** `go-expert`, model `haiku`

**Files:**
- `internal/imaging/save.go`
- `internal/imaging/raw_test.go`

**Steps:**

- [ ] `internal/imaging/save.go:183` — replace exactly:

      ```go
      		if err == io.EOF || err == io.ErrUnexpectedEOF {
      ```

      with exactly:

      ```go
      		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
      ```

- [ ] Add `"errors"` to `save.go`'s stdlib import group, between `"context"`
      and `"image"`. Do not reorder anything else.

- [ ] `internal/imaging/raw_test.go:463` — replace exactly:

      ```go
      	copy(head, []byte("FUJIFILMCCD-RAW "))
      ```

      with exactly:

      ```go
      	copy(head, "FUJIFILMCCD-RAW ")
      ```

      `copy` takes a string as its source directly. The string content is
      unchanged, including its trailing space.

- [ ] Leave the `DuplicatedCode` fragments in `raw_test.go` alone.

**Verification:**

- [ ] `make fmt-check`
- [ ] `go build ./...` succeeds.
- [ ] `go test ./internal/imaging/...` passes.
- [ ] `grep -c 'err == io\.' internal/imaging/save.go` reports `0`.

---

## Task 2: `errors.Is` in `exifwin/tiles_test.go`

**Subagent:** `go-expert`, model `haiku`

**Files:**
- `internal/ui/exifwin/tiles_test.go`

The file already imports `errors`; no import change is needed.

**Steps:**

- [ ] Line 164 — replace exactly:

      ```go
      	if err != errTilePending {
      ```

      with exactly:

      ```go
      	if !errors.Is(err, errTilePending) {
      ```

- [ ] Lines 218 and 299 — both read exactly the same apart from
      indentation. Replace each occurrence of:

      ```go
      if _, err := f.RoundTrip(req); err != errTilePending {
      ```

      with:

      ```go
      if _, err := f.RoundTrip(req); !errors.Is(err, errTilePending) {
      ```

      preserving each line's existing leading tabs.

- [ ] Change no `t.Fatalf`/`t.Errorf` message text.

**Verification:**

- [ ] `make fmt-check`
- [ ] `go test ./internal/ui/exifwin/...` passes.
- [ ] `grep -c 'err != errTilePending' internal/ui/exifwin/tiles_test.go` reports `0`.
- [ ] `grep -c 'errors.Is(err, errTilePending)' internal/ui/exifwin/tiles_test.go` reports `3`.

---

## Task 3: Rename the Builtin-Shadowing `real`

**Subagent:** `go-expert`, model `haiku`

**Files:**
- `internal/update/apply_test.go`

`real` is a local in `TestApplyUnix_EvalSymlinks` only, with four
occurrences: lines 146, 147, 151, and 163. The temp filename string moves
with it so the variable and the file it names stay in step.

**Steps:**

- [ ] Line 146 — replace exactly:

      ```go
      	real := filepath.Join(dir, "real")
      ```

      with exactly:

      ```go
      	target := filepath.Join(dir, "target")
      ```

- [ ] Lines 147, 151, and 163 — replace the identifier `real` with `target`:

      ```go
      	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
      	if err := os.Symlink(target, link); err != nil {
      	got, err := os.ReadFile(target)
      ```

- [ ] Touch nothing outside `TestApplyUnix_EvalSymlinks`. In particular leave
      the `link` and `staged` locals, the `t.Errorf` text, and the
      `DuplicatedCode` fragments at lines 17 and 154 alone.

**Verification:**

- [ ] `make fmt-check`
- [ ] `go test ./internal/update/...` passes, including
      `go test -run TestApplyUnix_EvalSymlinks -v ./internal/update/`.
- [ ] `grep -cw real internal/update/apply_test.go` reports `0`.
- [ ] `git diff --stat -- internal/update/apply_test.go` shows 4 insertions
      and 4 deletions.

---

## Task 4: Mark the Scripts' Ignored `Fprintf` Results

**Subagent:** `go-expert`, model `haiku`

**Files:**
- `scripts/releasenotes/main.go`
- `scripts/synctuf/main.go`

Both files have the identical `main()` shape, and
`scripts/plistdoctypes/main.go:29` already writes the fixed form.

**Steps:**

- [ ] `scripts/releasenotes/main.go:18` and `scripts/synctuf/main.go:22` —
      in each file replace exactly:

      ```go
      		fmt.Fprintf(os.Stderr, "%v\n", err)
      ```

      with exactly:

      ```go
      		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
      ```

- [ ] Change nothing else in either file. No comment is needed — `_, _ =` is
      the tree's marker for a deliberately ignored error, and the third
      script carries no comment either.

**Verification:**

- [ ] `make fmt-check`
- [ ] `go build ./...` succeeds.
- [ ] `go vet ./...` is clean.
- [ ] `go run ./scripts/synctuf --check` still succeeds (this is the
      `check-tuf-root` make target's command, and it exercises the edited
      `main`).
- [ ] `git diff --stat -- scripts/` shows 2 insertions and 2 deletions.

---

## Task 5: Hoist the Spiral FPS Backdrop Colours

**Subagent:** `go-expert`, model `sonnet`

**Files:**
- `internal/ui/spiral/overlays.go`
- `internal/ui/spiral/overlays_test.go`

Eight findings, plus the duplication the user asked to remove: the three FPS
backdrop colours are currently written as positional literals in `updateFPS`
and again in the test's table.

**Steps:**

- [ ] `overlays.go:25` — name the fields on the existing backdrop var:

      ```go
      var contentOverlayBackdropColor = color.NRGBA{R: 0, G: 0, B: 0, A: 191}
      ```

- [ ] `overlays.go` — insert this block between the closing brace of
      `newFPSOverlay` (line 154) and the doc comment of `updateFPS`
      (line 156), matching how `helpLineHeight` sits just above
      `newHelpOverlay`:

      ```go
      // fpsGoodColor, fpsWarnColor, and fpsBadColor are the performance
      // overlay's three backdrop colours, from healthy to stalling. They are
      // package-level so overlays_test.go can name the one it expects instead
      // of repeating the literals - see TestFPSBackdropColorValues, which is
      // what pins the values themselves.
      var (
      	fpsGoodColor = color.NRGBA{R: 0, G: 120, B: 0, A: 180}   // Dark Green
      	fpsWarnColor = color.NRGBA{R: 120, G: 120, B: 0, A: 180} // Dark Yellow
      	fpsBadColor  = color.NRGBA{R: 150, G: 0, B: 0, A: 180}   // Red
      )
      ```

- [ ] `overlays.go:167-174` — replace the branch's literals with the vars,
      dropping the now-duplicated trailing colour comments:

      ```go
      	var bgColor color.NRGBA
      	if fps > 60 {
      		bgColor = fpsGoodColor
      	} else if fps >= 40 {
      		bgColor = fpsWarnColor
      	} else {
      		bgColor = fpsBadColor
      	}
      ```

- [ ] `overlays_test.go:66-69` — reference the vars in the table:

      ```go
      		{"above 60fps is dark green", 1.0 / 61.0, fpsGoodColor},
      		{"between 40 and 60fps is dark yellow", 1.0 / 50.0, fpsWarnColor},
      		{"below 40fps is red", 1.0 / 20.0, fpsBadColor},
      		{"zero dt guards against divide by zero, reporting 0fps (red)", 0, fpsBadColor},
      ```

      Leave the test names, the `dt` values, and the rest of the function
      exactly as they are.

- [ ] `overlays_test.go` — add this test directly above
      `TestUpdateFPSBackdropColorThresholds`:

      ```go
      // TestFPSBackdropColorValues pins the three backdrop colours themselves.
      // TestUpdateFPSBackdropColorThresholds below checks which one updateFPS
      // picks for a given frame time, but it now compares against these same
      // vars, so on its own it would pass just as happily if a colour were
      // changed. This is the test that fails when one is.
      func TestFPSBackdropColorValues(t *testing.T) {
      	tests := []struct {
      		name string
      		got  color.NRGBA
      		want color.NRGBA
      	}{
      		{"good", fpsGoodColor, color.NRGBA{R: 0, G: 120, B: 0, A: 180}},
      		{"warn", fpsWarnColor, color.NRGBA{R: 120, G: 120, B: 0, A: 180}},
      		{"bad", fpsBadColor, color.NRGBA{R: 150, G: 0, B: 0, A: 180}},
      	}

      	for _, tt := range tests {
      		if tt.got != tt.want {
      			t.Errorf("%s color = %v; want %v", tt.name, tt.got, tt.want)
      		}
      	}
      }
      ```

      This exists because hoisting the colours would otherwise leave nothing
      asserting what they are — the threshold test would compare a var against
      itself. If the user strikes this step, say so rather than silently
      skipping it.

- [ ] Change no rendered pixel: the three colour values, the `fps > 60` and
      `fps >= 40` thresholds, the overlay geometry, and
      `contentOverlayBackdropColor`'s value all stay exactly as they are.

**Verification:**

- [ ] `make fmt-check`
- [ ] `go build ./...` succeeds.
- [ ] `go test ./internal/ui/spiral/...` passes, and
      `go test -run 'TestFPSBackdropColorValues|TestUpdateFPSBackdropColorThresholds' -v ./internal/ui/spiral/`
      shows both tests running and passing.
- [ ] `grep -c 'color.NRGBA{[0-9]' internal/ui/spiral/overlays.go internal/ui/spiral/overlays_test.go`
      reports `0` for both files.
- [ ] Mutation check, then revert it: change `fpsGoodColor`'s `G` from 120 to
      121 and confirm `TestFPSBackdropColorValues` fails while
      `TestUpdateFPSBackdropColorThresholds` still passes. That is the whole
      point of the new test; report the output.

---

## Parent Controller Protocol After Every Task

The parent session, not the subagent, owns verification. After each task
returns:

1. `git diff -- <the task's files>` read in full.
2. `mcp__goland__lint_files` over the task's files. The task's findings must
   be gone; the pre-existing `DuplicatedCode` warnings in `raw_test.go` and
   `apply_test.go` must still be present and are expected.
3. `make fmt-check`, then `go build ./...`.
4. Fix up anything the subagent got wrong directly rather than redispatching,
   unless the task needs redoing wholesale.
5. Report the result and wait for the user's go-signal before the next step.

## Final Repository Verification

- [ ] `mcp__goland__lint_files` over all 8 touched files — zero findings in
      the five categories above.
- [ ] `make fmt-check`
- [ ] `go vet ./...`
- [ ] `go build ./...`
- [ ] `go test -timeout 20m -race ./...`
- [ ] `go run ./scripts/synctuf --check` succeeds.
- [ ] `git diff` reviewed whole: no behaviour change, no stray edits.

## Backlog Close

- [ ] In `todos.md`, delete the `### Qodana: small mechanical fixes` section
      from `## TODO`.
- [ ] Add an entry under `## Done` → `#### Internal` recording: the five
      categories and where each landed; that `extract.go` and
      `doctypes_test.go` keep their `err == io.EOF` because the stdlib
      iterator idiom is not flagged; that the FPS colours became package vars
      with `TestFPSBackdropColorValues` added to keep the values pinned once
      the threshold test started comparing a var against itself; and that
      `_, _ =` matches `scripts/plistdoctypes/main.go`.
- [ ] `ARCHITECTURE.md` needs no update — no package added, removed, renamed,
      or moved.
- [ ] Move this plan to `finished_refactorings/`.
- [ ] Do not commit. Hand the user a suggested commit message.

## Out of Scope

- `internal/update/extract.go:105` and
  `scripts/plistdoctypes/doctypes_test.go:228` — unflagged stdlib iterator
  terminators.
- The `DuplicatedCode` findings in `internal/imaging/raw_test.go` and
  `internal/update/apply_test.go`.
- `internal/ui/memlimits.go:29`, the `settings` struct-padding warning.
- The remaining `todos.md` items: the favorites native-menu-bar bug, the
  untested manual-opened observer, the `maybeStartUpdateCheck` ordering note,
  the false-positive suppression CI confirmation, and the Qodana CI
  duplication under-reporting.
- `qodana.yaml`. Untouched.
