# Migrate `exifwin.warmDone` onto `completion.Signal`

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) to implement this
> plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> Parent session reviews every task **and fixes the code** before
> dispatching the next. Do not start Task N+1 until Task N is reviewed and
> fixed. **Do not run `git commit`.**

**Goal:** Replace `internal/ui/exifwin`'s last hand-rolled one-shot
`chan struct{}` (`warmDone`) and `waitForWarm`'s nil-guard-plus-`select`
with `internal/completion.Signal`, the same type item 5 already put on
the nine viewer-owned operations.

**Architecture:** `exifwin.Window` already has the domain staleness rule
(`warmGen`, plus `locationMap == nil` after close). Keep that. Swap only
the test-facing completion mechanism: `startWarm` calls `warm.Begin()`
where it used to `make`+assign the channel, and the `fyne.Do` callback
`defer done()`s where it used to `defer close(done)`. `waitForWarm` keeps
its “never started” fatal (today’s `== nil`) via `Begun()`, then
`Wait`s with a 5s timeout. Do not export the Signal. Do not add it to
`newTestUI`’s `drain`.

**Tech Stack:** Go 1.26.7, Fyne v2.8, existing `internal/completion` and
`internal/ui/exifwin`. No new dependencies. No imaging or tile-fetcher
behavior change.

**Spec:** `todos.md` TODO “Migrate `internal/ui/exifwin`'s `warmDone`
onto `internal/completion.Signal`”.

**Precedent:** `planned_features/unify_completion_signals.md` (item 5)
and `internal/ui/toast.go` (`hidden completion.Signal` on a
feature-owned struct, not on `viewer`). Field naming drops the `Done`
suffix (`v.loadDone` → `v.load`). Helpers that used to fatal on a nil
channel keep a `Begun()` canary (`settleChooser` in
`internal/ui/harness_test.go`); they do **not** rely on `Signal.Wait`’s
never-begun immediate return.

**What is already done (do not re-implement):**

- `internal/completion.Signal` (`Begin` / `Wait` / `Begun` / `Current` /
  `Handle`) and its tests.
- `warmGen` staleness, `warming` spinner flag, `tiles.Warm`, map hide
  until the block is in, close callback bumping `warmGen` and nil-ing
  `locationMap`.
- All Location-section tests and `waitForWarm` call sites in
  `exifwin_test.go`. Those tests stay; only the helper’s internals
  change.
- `internal/ui/exifwin/tiles.go`’s own `warming` field (the fetcher’s
  in-flight flag). That is a different contract. Do not touch `tiles.go`.

---

## Open questions (proposed defaults — confirm before dispatch)

Implementers treat the **Proposed** column as spec unless Florian
overrides it before Task 1 starts.

| # | Question | Proposed |
|---|----------|----------|
| 1 | Field name after the type change? | **`warm completion.Signal`.** Drop the `Done` suffix, matching item 5 (`loadDone` → `load`). Coexists with `warming bool` (`w.warm` vs `w.warming`). Do not keep `warmDone`. |
| 2 | `waitForWarm` when no prefetch has begun? | **Keep the fatal.** `if !w.warm.Begun() { t.Fatal("no prefetch has been started") }` then `Wait`. Do **not** let `Signal.Wait`’s never-begun `nil` swallow a missed `ToggleLocation` / `startWarm`. Same canary as `settleChooser`. |
| 3 | Add `Window.Settle` / put `warm` in `newTestUI`’s `drain`? | **No.** `drain` is for viewer-owned work. Viewer tests never expand the map. `exifwin` tests already `waitForWarm` and `Close`. Exporting the Signal (or a Settle wrapper) is extra API. Leave `drain` alone. |
| 4 | New `Current()` / superseded-generation test? | **No.** `TestRefresh_ExpandedSectionRefetchesForANewPosition` already waits generation 1, `Begin`s generation 2, waits generation 2. `TestClose_StopsTheFetcherFromTouchingDeadWidgets` waits the same generation after close (close bumps `warmGen`, it does **not** `Begin`). `completion`’s own tests already lock “stale closer still finishes its generation.” |
| 5 | Where does `defer done()` run? | **Inside the `fyne.Do` callback**, replacing `defer close(done)` byte-for-byte in placement. Do **not** `defer done()` at the goroutine’s entry (toast’s pattern). Closing before the callback runs would let `waitForWarm` return before widget writes; the test driver hides that because `fyne.Do` is inline, production `fyne.Do` is async. |
| 6 | User manuals? | **Unchanged.** This is a test-sync refactor. No user-visible string or behavior change. |
| 7 | Uncommitted SVG-zoom edits in the tree (`ARCHITECTURE.md`, `vector.go`, manuals, `todos.md`)? | **Leave them.** Touch only the sentences this plan names. Do not revert or rewrite SVG-zoom prose. |

---

## Dispatch order and models

Parent: this session. **One implementer at a time.** After each task:
parent reviews the diff, **fixes if needed**, then dispatches the next.

Available Cursor Task models for this repo: `composer-2.5-fast` (cheap
mechanical), `claude-sonnet-5-thinking-high` (standard Go/Fyne),
`claude-opus-5-thinking-high` (only if a task cannot be split).

| Task | What | Implementer | Reviewer |
|------|------|-------------|----------|
| 1 | Field + `startWarm` + `waitForWarm`; existing Location tests stay green | `go-expert` · `claude-sonnet-5-thinking-high` | parent (this session), not a subagent |
| 2 | `ARCHITECTURE.md`, `completion.go` package comment, `todos.md` | `generalPurpose` · `composer-2.5-fast` | parent |
| Final | Whole-branch review after Task 2 | — | parent; escalate to `go-expert` · `claude-opus-5-thinking-high` only if Task 1 races or `waitForWarm` returns before the spinner/map assertions |

Task 1 is the `fyne.Do` closer placement plus a `Begun()` canary — easy
to “simplify” into toast’s goroutine-level `defer done()` or into a
silent `Wait`. Do not downgrade it to the cheap model. Task 2 is docs.

Do **not** use Opus for Tasks 1–2. The work splits. The parent, not a
subagent, owns cross-task review and any fix-up.

---

## Global Constraints

Copied from `AGENTS.md`; every task’s requirements implicitly include these.

- **Do not run `git commit`.** Each task ends with a *suggested* commit
  message. The parent does not commit either unless Florian asks.
- Do not add `TODO`/`FIXME` comments to source. Open work belongs in
  `todos.md`.
- Update `ARCHITECTURE.md` in the same change when the completion-signal
  story gains a tenth owner (Task 2).
- Every user-visible string is `lang.L("English text")` with that exact
  key in every `translations/*.json` bundle. This plan adds **no** new
  strings.
- `internal/completion` stays viewer-independent: no Fyne types, no
  `fyne.Do`. `exifwin` importing it is the point; do not reverse that
  (do not move Signal, do not add a Fyne helper to `completion`).
- Mark intentionally ignored errors explicitly (`_ =` or `_, _ =`).
- No new dependencies. No mutable package-level test seams.
- Do not export `warm`. Do not add `Settle` / `WarmDone` / a public
  getter. `waitForWarm` stays in `exifwin_test.go` (same package).
- Do not change `tiles.go`, `confirm.go`, strip-button layout, GPS
  parsing, or map widget behavior.
- Do not `Begin()` on `startWarm`’s early return (`!hasPos` or
  `locationMap == nil`). Today that path leaves the previous channel
  alone; keep that.
- Do not `Begin()` or zero the Signal in the close callback. Close
  bumps `warmGen` and nils widgets; the in-flight closer still fires.
  `Begun()` stays true after close — `TestClose_*` relies on waiting
  that generation.
- Do not edit `internal/ui/harness_test.go` / `drain`.
- Do not rewrite SVG-zoom, strip-button, or other unrelated uncommitted
  edits in files this plan also touches.
- Import grouping in `exifwin.go`: stdlib, blank, Fyne, blank, picfetch
  (`completion` next to `imaging` / `widgets`). Do not copy `toast.go`’s
  merged import block.
- Verification per task, from the repository root, after the task’s own
  focused tests pass: `gofmt -l .` (must print nothing), `go vet ./...`,
  `go build ./...`, then the focused tests named in the task. The parent
  runs `go test -race ./...` after Task 1.

---

## File map

| File | Role |
|------|------|
| `internal/ui/exifwin/exifwin.go` | `warm completion.Signal`; `startWarm` uses `Begin` + `defer done()` inside `fyne.Do` |
| `internal/ui/exifwin/exifwin_test.go` | `waitForWarm` uses `Begun` + `Wait`; add `"context"` import |
| `internal/completion/completion.go` | Package comment: not “nine viewer copies” only |
| `ARCHITECTURE.md` | Concurrency invariant + `internal/completion` section + `exifwin/` row if it names the channel |
| `todos.md` | Move this TODO under Done |

No new packages. No new translation keys. No manuals. No `tiles.go`.
No `harness_test.go`.

---

## Assumptions (locked for implementers)

1. **Same goroutine shape.** `startWarm` still increments `warmGen`,
   snapshots `lat`/`lon`, sets `warming`, hides the map, `syncLoading`,
   then `go` → `tiles.Warm` → `fyne.Do`. Only the channel
   make/assign/close trio becomes `Begin` + captured `done`.
2. **Closer inside `fyne.Do`.** `waitForWarm` is a completion wait, not
   a poll, because under the test driver the callback runs inline on the
   fetching goroutine and writes widgets before `done()` returns. Moving
   `done()` outside the callback is a production-vs-test semantic split.
3. **`waitForWarm` call sites unchanged.** Nine existing calls; do not
   rename the helper; do not switch them to `w.warm.Wait` inline.
4. **Timeout stays 5s.** Restate `5 * time.Second` in the helper (do
   not import `internal/ui` for `testTimeout`). Use
   `context.WithTimeout` + `Signal.Wait`, not a raw `select`.
5. **Fatal message text stays.** `"no prefetch has been started"` and
   `"timed out waiting for the map prefetch"` so a failing test still
   reads the same.
6. **Zero value is fine.** `Window.warm` needs no constructor init.
   `New` does not mention it.
7. **`warming` / `warmGen` stay.** They are UI/staleness, not the
   completion mechanism.

---

## Task 1: Replace `warmDone` with `completion.Signal`

**Files:**
- Modify: `internal/ui/exifwin/exifwin.go` (`Window` field, `startWarm`)
- Modify: `internal/ui/exifwin/exifwin_test.go` (`waitForWarm`, imports)

**Interfaces:**
- Consumes: `completion.Signal` with `Begin() (done func())`,
  `Wait(ctx context.Context) error`, `Begun() bool`.
- Produces: `Window.warm completion.Signal` (unexported).
  `waitForWarm(t *testing.T, w *Window)` with the same signature and
  call sites as today.

Do not touch docs in this task. Task 2 does `ARCHITECTURE.md` /
package comment / `todos.md`.

- [ ] **Step 1: Rewrite `waitForWarm` onto the Signal API**

This is the failing-test step: after this step the test file must not
compile until Step 3 lands the field.

Add `"context"` to the stdlib import group in `exifwin_test.go`.

Replace `waitForWarm` (today ~lines 286–303) with:

```go
// waitForWarm blocks until the prefetch the last expand started has
// finished, so a test can assert on the loading indicator without racing
// it. Deliberately a completion wait rather than polling widget state:
// the Fyne test driver runs fyne.Do inline, so widget state is written
// from the fetching goroutine before that generation's closer runs.
func waitForWarm(t *testing.T, w *Window) {
	t.Helper()

	if !w.warm.Begun() {
		t.Fatal("no prefetch has been started")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := w.warm.Wait(ctx); err != nil {
		t.Fatal("timed out waiting for the map prefetch")
	}
}
```

Do not change any `waitForWarm(t, w)` call site.

- [ ] **Step 2: Confirm the tests do not compile**

Run: `go test -count=1 ./internal/ui/exifwin/ -run 'TestToggleLocation_|TestRefresh_ExpandedSectionRefetchesForANewPosition|TestClose_StopsTheFetcherFromTouchingDeadWidgets|TestPaint_DoesNotBlockOnSlowTiles' -v`

Expected: compile error on `w.warm` (field is still `warmDone chan struct{}`)
or on `Begun`/`Wait` called on a channel. Do not skip ahead if it
compiles — then Step 1 did not actually switch the helper.

- [ ] **Step 3: Field + `startWarm`**

In `exifwin.go` imports, add to the picfetch group (keep the blank line
after the Fyne group):

```go
	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/widgets"
```

Replace the `warmDone` field and its comment (today ~lines 113–120)
with:

```go
	// tiles downloads and caches the map's tiles off the UI goroutine -
	// see tiles.go for why the widget's own fetching can't be left to it.
	// warming and warmGen track the prefetch that fills the first view,
	// warm is the completion.Signal tests wait on - see internal/completion.
	tiles   *tileFetcher
	warming bool
	warmGen int
	warm    completion.Signal
```

In `startWarm`, replace the channel make/assign and the `defer close(done)`
inside `fyne.Do`. The rest of the function is unchanged, including the
early return, `warmGen++`, `warming = true`, map hide, and the stale
`gen != w.warmGen || w.locationMap == nil` guard:

```go
func (w *Window) startWarm() {
	if !w.hasPos || w.locationMap == nil {
		return
	}

	w.warmGen++
	gen := w.warmGen
	lat, lon := w.lat, w.lon

	done := w.warm.Begin()

	w.warming = true

	// Drawing the map before its tiles are cached would have it ask for
	// every one of them and get nothing (see tiles.go), logging a failure
	// per tile per frame and showing a grid of holes. Keeping it hidden
	// until the block is in trades that for a spinner and one clean frame.
	w.locationMap.Hide()
	w.syncLoading()

	tiles := w.tiles

	go func() {
		tiles.Warm(lat, lon, mapZoom)

		fyne.Do(func() {
			defer done()

			if gen != w.warmGen || w.locationMap == nil {
				return
			}

			w.warming = false
			w.syncLoading()
			w.locationMap.Show()

			// Revealing the map has to re-run the stack's layout for the
			// same reason expanding the section does.
			w.body.Refresh()
		})
	}()
}
```

Grep the package for `warmDone`. Expected: no remaining hits in
`internal/ui/exifwin/`. Hits in `todos.md` / `planned_features/` /
historical docs are Task 2 / out of scope.

- [ ] **Step 4: Run the Location tests**

Run: `go test -count=1 ./internal/ui/exifwin/ -run 'TestToggleLocation_|TestRefresh_ExpandedSectionRefetchesForANewPosition|TestClose_StopsTheFetcherFromTouchingDeadWidgets|TestPaint_DoesNotBlockOnSlowTiles' -v`

Expected: PASS (the nine `waitForWarm` tests plus any other ToggleLocation
tests in that regex).

Then the whole package: `go test -count=1 ./internal/ui/exifwin/`

Expected: PASS.

Then from the repository root:

```
gofmt -l .
go vet ./...
go build ./...
```

Expected: `gofmt -l .` prints nothing; vet and build succeed.

- [ ] **Step 5: Suggested commit message (do not commit)**

```
refactor: migrate exifwin map prefetch onto completion.Signal

The Location-section warm-up was the tenth copy of the replace-on-start
/ close-on-finish channel the viewer already collapsed into
internal/completion. Keep the closer inside fyne.Do so tests still wait
until widget state has been written.
```

**Task 1 acceptance:**

- `warmDone` is gone from `internal/ui/exifwin/`.
- `startWarm` uses `w.warm.Begin()`; `defer done()` is inside `fyne.Do`.
- `waitForWarm` fatals on `!Begun()` and `Wait`s with a 5s timeout.
- `tiles.go` / `confirm.go` / harness / drain / docs untouched.
- Focused package tests pass; `gofmt`/`vet`/`build` clean.

---

## Task 2: Docs — tenth Signal is feature-owned

**Files:**
- Modify: `internal/completion/completion.go` (package comment only)
- Modify: `ARCHITECTURE.md` (concurrency invariant, `internal/completion`
  section; one clause on the `exifwin/` row if it still implies a raw
  channel)
- Modify: `todos.md` (move this TODO under Done)

**Interfaces:**
- Consumes: Task 1’s `Window.warm completion.Signal` and `waitForWarm`.
- Produces: docs that say the type now has a tenth owner in `exifwin`,
  not on `viewer`, not in `drain`.

Do not change Go behavior. Do not touch manuals. Do not rewrite the SVG
zoom Done paragraph or any other Done item.

- [ ] **Step 1: Package comment in `internal/completion/completion.go`**

Replace the first paragraph of the package comment (the sentence that
says the viewer grew **nine** hand-rolled copies) so it still names
those nine *and* the EXIF window’s prefetch, without implying Signal
lives on `viewer` only. Exact replacement for lines 1–17:

```go
// Package completion is the one-shot "this background operation has
// finished" signal that the viewer grew nine hand-rolled copies of, and
// that internal/ui/exifwin grew a tenth for the Location-section map
// prefetch: a channel replaced at the start of each request and closed
// when that request finishes, which the test suite waits on instead of
// polling widget state a producer goroutine may still be writing.
//
// The rule it makes unbreakable is the one those copies could only
// state in prose: a request that has been superseded must still close its
// own channel, without touching the field a newer request now owns. Begin
// hands back a func closed over this generation's channel, so a stale
// producer cannot reach the newer one even by accident.
//
// It is deliberately viewer-independent: no Fyne types, no fyne.Do, no UI
// marshaling. The caller decides what counts as stale and what finishing
// means; Signal answers only "has the generation I am looking at finished
// yet".
```

Do not change any function in this file.

- [ ] **Step 2: `ARCHITECTURE.md`**

Three surgical edits. Do not reflow the rest of the file. Do not revert
unrelated uncommitted SVG-zoom wording.

**A. Concurrency invariant** (the paragraph that lists `v.anim`,
`scanOp.done`, `v.load`, … as `completion.Signal` and says they replaced
**nine** hand-rolled fields). After the sentence that names those nine
viewer-side Signals, add one sentence (same paragraph is fine):

`internal/ui/exifwin.Window.warm` is a tenth `completion.Signal`, owned
by the EXIF panel for the Location-section prefetch (`startWarm`); tests
wait via `waitForWarm` in `exifwin_test.go`. It is not a `viewer` field
and is not in `newTestUI`’s `drain` — viewer tests never expand the map,
and the panel’s own tests wait the prefetch out themselves.

Keep the existing list of the nine viewer Signals. Do not add `warm` to
the `drain` table in prose as if it were wired there.

**B. `### internal/completion` section.** Update “nine hand-rolled
copies” / “all nine now share” / “each its own zero-value instance on
`viewer`” so the section also names `exifwin.Window.warm` as a
zero-value instance on the panel, waited by `waitForWarm`, not by
`harness_test.go`’s `waitFor`. Keep the `Begin`/`Wait`/`Current`
mechanics paragraph; it is still correct.

The “Added 2026-08-22…” footer may gain a short clause that the EXIF
prefetch joined on a later date (use 2026-08-23), still extracted from
the same contract. Do not pretend item 5 migrated it.

**C. `exifwin/` package-map row.** If that row describes the prefetch
only as `Warm` + spinner, add that tests wait the prefetch on
`Window.warm` (`completion.Signal`), not a raw `chan struct{}`. One
clause, not a rewrite of the strip-button / GPS / `Host` story.

- [ ] **Step 3: `todos.md`**

Cut the whole “Migrate `internal/ui/exifwin`'s `warmDone`…” heading and
paragraph from `## TODO`. Paste it under `## Done` (after the SVG zoom
item, before `## ACTIVE DEVELOPMENT`), rewritten in past tense, without
stale line numbers. Match the voice of item 5’s Done paragraph.

Suggested Done text:

```markdown
### Migrate `internal/ui/exifwin`'s `warmDone` onto `internal/completion.Signal`

The EXIF panel's Location-section prefetch was the tenth hand-rolled
copy of the replace-on-start / close-on-finish / wait-in-test channel
item 5 collapsed on `viewer`. `Window.warm` is now a
`completion.Signal`; `startWarm` `Begin`s it and the `fyne.Do` callback
still finishes that generation after writing widgets (including when
`warmGen` or a closed window has made the prefetch stale).
`waitForWarm` keeps a `Begun()` canary and `Wait`s with a timeout, the
same shape `settleChooser` already uses. It stayed out of item 5
because that plan named only the nine viewer fields; it is still not in
`drain`, because it is not a viewer-owned signal.
```

Leave `## TODO` with no items (or empty) rather than a placeholder.
Leave “not deemed worth implementing” untouched. Do not mention
`warmDone` in the SVG zoom Done paragraph.

- [ ] **Step 4: Verify docs-only + tests still pass**

`gofmt -l .` (must print nothing — the package-comment edit is the only
Go change).

`go test -count=1 ./internal/ui/exifwin/ ./internal/completion/`

Expected: PASS.

Grep: `rg 'warmDone' internal/ui/exifwin internal/completion` prints
nothing. `todos.md` may still mention `warmDone` in the Done heading
(that is the historical name; fine). `planned_features/` historical
plans that said “do not migrate warmDone” stay as they were.

- [ ] **Step 5: Suggested commit message (do not commit)**

```
docs: record exifwin map prefetch as a completion.Signal

The type is no longer viewer-only: the EXIF panel owns the tenth
instance, waited in-package, not through drain.
```

**Task 2 acceptance:**

- Package comment, `ARCHITECTURE.md`, and `todos.md` agree: tenth
  Signal, feature-owned, not drained.
- Manuals untouched. No behavior change. `gofmt` clean.
- `## TODO` no longer lists this migration.

---

## Parent verification (after Task 2, before “done”)

From the repository root, parent runs (does not take the implementer’s
word):

```
gofmt -l .
go vet ./...
go build ./...
go test -race ./...
```

Expected: `gofmt -l .` prints nothing; vet, build, and the race suite
pass.

Also confirm:

1. Diff matches this plan (exifwin field + helper, completion package
   comment, architecture, todos). No `tiles.go`, no `drain`, no manuals,
   no SVG-zoom rewrites.
2. `defer done()` is inside `fyne.Do` in `startWarm`.
3. `waitForWarm` still fatals when `!Begun()`.
4. `warmDone` is gone from `internal/ui/exifwin/`.

---

## Spec coverage (self-review)

| Requirement | Task |
|-------------|------|
| Replace `warmDone chan struct{}` with `completion.Signal` | 1 |
| `startWarm` Begin + closer on the same generation | 1 |
| Closer inside `fyne.Do` (widget writes happen first under the test driver) | 1 Q5 |
| `waitForWarm` drops the hand-rolled select; keeps never-started fatal | 1 Q2 |
| No import cycle (`completion` has no Fyne) | 1 (exifwin imports completion) |
| Do not put it in `drain` / do not export | 1 Q3, global constraints |
| `ARCHITECTURE.md` + todos Done | 2 |
| No user-visible change | Q6, no manuals |

No placeholders. No “add tests for the above” without code — Task 1
reuses the existing Location tests as the suite. No Opus: two tasks.
