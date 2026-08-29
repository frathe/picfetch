# Settings numeric-entry helper — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task.
> Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** after every task the parent agent reviews the
> subagent's diff line by line, fixes it up itself, runs that task's verification
> commands, and then **stops** and hands Florian a suggested commit message. Do not
> dispatch Task N+1 until Florian has confirmed the commit landed. Do not run
> `git commit` (`AGENTS.md`). Dispatch **one** implementer at a time — Tasks 1 and 2
> share `internal/ui/settingswin/settingswin.go` and will conflict if run in parallel.

**Goal:** One unexported helper builds every positive-integer Settings field so a
seeding or validation fix cannot silently leave five of the six copies wrong.

**Architecture:** Add `newPositiveIntEntry(get, set, max, validate)` next to `build`.
It is a literal extract of the six copied blocks: `widget.NewEntry()`, assign the
shared `positiveInt` validator, seed `Text` from `get()` without `SetText`, and
`OnChanged` that `strconv.Atoi`s, requires `n > 0`, and when `max > 0` also
requires `n <= max` before calling `set`. Width/height keep a one-line `int` ↔
`float32` wrapper at the call site. The picture-frame interval stays hand-rolled —
it is `ParseInt` + `time.Duration`, not this helper.

**Tech Stack:** Go, package `internal/ui/settingswin`, Fyne `widget.Entry`,
existing `go test` / `-race`. No new dependencies.

**Spec:** `todos.md` § "Qodana: settingswin's numeric entries are five copies of one
block".

## Global Constraints

- Do not run `git commit`. End each task with a suggested commit message for Florian.
- Do not add `TODO`/`FIXME` comments. Open work stays in `todos.md`.
- Unexported helper only. Do not export it, do not put it in `internal/ui/widgets`,
  do not introduce generics.
- Helper semantics are a literal extract of the six copies, not a new policy.
  Keep `strconv.Atoi` (not `ParseInt`). Keep ignoring invalid text rather than
  reverting the field or calling `set(0)`. Seed via `e.Text = strconv.Itoa(get())`,
  never `SetText` — `SetText` fires `OnChanged` and would round-trip into the host
  on open (the contract `TestShow_SeedsEveryControlFromHostWithoutRoundTripping`
  already pins).
- `max <= 0` means no ceiling (today: max-scan, max-width, max-height). `max > 0`
  means `n <= max` (today: the three memory fields pass `maxMemoryMB`).
- Do **not** fold `intervalEntry` into the helper. It uses `strconv.ParseInt` and
  `maxDurationSeconds` because a second count has to survive `time.Duration(n) *
  time.Second` without wrapping negative.
- Do **not** collapse `widget.NewFormItem` / `HintText` construction. Those labels
  and hints are not the duplicated block Qodana flagged.
- Do **not** change `Host`, `Window` fields, `Show`'s nil-out list, or any
  translation key.
- Do **not** edit `ARCHITECTURE.md`. No package was added, removed, or renamed,
  and no file moved between packages.
- Tests: no `time.Sleep`; no new mutable package-level seams. Drive the helper
  directly in Task 1; keep driving the widget fields through `Show()` in the
  existing characterization tests. Use the package's `TestMain` `testApp`; do
  not call `test.NewApp()` again.
- Format with `goimports -local github.com/frathe/picfetch`. `make fmt` is fine.
- Verification is `go test -timeout 2m -race -count=1 ./internal/ui/settingswin/`
  unless a task names a narrower `-run`. Do not run the 20-minute full-repo
  suite unless the parent asks.

## Open points (defaults used below)

Answer these before Task 1 if you disagree; otherwise the implementers follow the
defaults.

1. **Helper signature** — **default: four arguments**, sharing `build`'s
   `positiveInt` validator so all seven integer fields (the six plus interval)
   still use one regexp:
   `func newPositiveIntEntry(get func() int, set func(int), max int, validate fyne.StringValidator) *widget.Entry`.
   Alternative: three arguments, helper calls `validation.NewRegexp` itself.
   That matches the todo's wording more tightly but duplicates the regexp with
   `intervalEntry`.
2. **Interval** — **default: out of scope.** It is not one of the six copies.
3. **File** — **default: keep the helper in `settingswin.go`**, immediately
   above `build`. Do not add `entry.go` for ~20 lines.
4. **`max` convention** — **default: `max <= 0` means unbounded.** No `*int`,
   no variadic, no second helper.
5. **Extra characterization tests** — **default: add two** in Task 2, covering
   behaviour the six copies already had but the suite never pinned: max-scan
   accepts a value above `maxMemoryMB`, and an image-cache value of exactly
   `maxMemoryMB` is accepted. These stop a "always pass `maxMemoryMB`" wiring
   bug from going green.

## Approaches considered

1. **Recommended (this plan).** One `newPositiveIntEntry` with `get` / `set` /
   `max` / `validate`. Width/height wrap `float32` at the call site. Interval
   stays as it is. Smallest change that makes Qodana's five hits go away and
   keeps the existing tests meaningful.
2. **Generic over the Host value type.** Window size is the only `float32`;
   a type parameter (or `set func(T)`) is more API for one conversion. Rejected.
3. **Two helpers (`newUnboundedPositiveIntEntry` / `newCappedPositiveIntEntry`).**
   The `max <= 0` convention already distinguishes them. Two names would re-split
   the thing Qodana asked us to join.

## File map

| File | Role after this plan |
|------|----------------------|
| `internal/ui/settingswin/settingswin.go` | **Modify.** Add `newPositiveIntEntry`; replace the six copied blocks in `build`. Interval construction unchanged. |
| `internal/ui/settingswin/settingswin_test.go` | **Modify.** Add `TestNewPositiveIntEntry` (Task 1). Add max-scan-above-ceiling and image-cache-at-ceiling tests (Task 2). Existing widget tests stay. |
| `todos.md` | **Modify.** Move this Qodana item from TODO to Done → Internal. |
| `ARCHITECTURE.md` | **Unchanged.** |

## Subagent assignment

| Task | Subagent | Model | Why |
|------|----------|-------|-----|
| 1 — helper + unit tests | `go-expert` | `composer-2.5-fast` | Mechanical extract; this plan contains the complete test and function text. Transcription + TDD, not design. |
| 2 — wire `build` + two characterization tests | `go-expert` | `composer-2.5-fast` | Mechanical replacement of six call sites against a locked signature. |
| 3 — backlog | `generalPurpose` | `composer-2.5-fast` | `todos.md` only. |

No task needs `claude-opus-5-thinking-high`: the work splits cleanly and every
function body is fully specified. Parent review after each task uses this
session's model.

Do **not** dispatch `best-of-n-runner` or parallel implementers.

---

### Task 1: `newPositiveIntEntry` + unit tests

**Subagent:** `go-expert` · **Model:** `composer-2.5-fast`

**Files:**
- Modify: `internal/ui/settingswin/settingswin.go` (add the helper only; do not
  touch `build` yet)
- Modify: `internal/ui/settingswin/settingswin_test.go` (append `TestNewPositiveIntEntry`)
- Test: `go test -timeout 2m -race -count=1 ./internal/ui/settingswin/ -run TestNewPositiveIntEntry`

**Interfaces:**
- Consumes: package const `maxMemoryMB`; Fyne `widget.Entry`; `fyne.StringValidator`.
- Produces:
  - `func newPositiveIntEntry(get func() int, set func(int), max int, validate fyne.StringValidator) *widget.Entry`

- [ ] **Step 1: Write the failing test**

Append this test to `internal/ui/settingswin/settingswin_test.go`. Do not
implement `newPositiveIntEntry` yet. Do not add imports — the stub validator
is a `func(string) error`, which is `fyne.StringValidator`.

```go
// TestNewPositiveIntEntry pins the helper in isolation, before build switches
// onto it. Invalid text is the mid-edit state (empty, garbage, zero, overflow)
// and must not reach set — the same contract
// TestMaxScanEntry_InvalidTextIsIgnored / TestMemoryEntries_InvalidTextIsIgnored
// already make through the widgets.
func TestNewPositiveIntEntry(t *testing.T) {
	validate := func(string) error { return nil }

	t.Run("seeds Text from get without calling set", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 42 }, func(n int) { calls = append(calls, n) }, 0, validate)
		if got, want := e.Text, "42"; got != want {
			t.Errorf("Text = %q, want %q", got, want)
		}
		if len(calls) != 0 {
			t.Errorf("set called on seed: %v", calls)
		}
		if e.Validator == nil {
			t.Error("Validator is nil, want the validate argument")
		}
	})

	t.Run("valid change calls set", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 1 }, func(n int) { calls = append(calls, n) }, 0, validate)
		e.SetText("100")
		if len(calls) != 1 || calls[0] != 100 {
			t.Errorf("set calls = %v, want [100]", calls)
		}
	})

	t.Run("invalid text is ignored", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 1 }, func(n int) { calls = append(calls, n) }, 0, validate)
		for _, text := range []string{"", "abc", "-1", "0", "99999999999999999999"} {
			e.SetText(text)
		}
		if len(calls) != 0 {
			t.Errorf("set calls = %v, want none for invalid input", calls)
		}
	})

	t.Run("max 0 has no ceiling", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 1 }, func(n int) { calls = append(calls, n) }, 0, validate)
		e.SetText("1048577")
		if len(calls) != 1 || calls[0] != 1048577 {
			t.Errorf("set calls = %v, want [1048577] (unbounded)", calls)
		}
	})

	t.Run("maxMemoryMB is accepted, one over is ignored", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 1 }, func(n int) { calls = append(calls, n) }, maxMemoryMB, validate)
		e.SetText("1048576")
		e.SetText("1048577")
		if len(calls) != 1 || calls[0] != maxMemoryMB {
			t.Errorf("set calls = %v, want [%d] (the ceiling, not one over)", calls, maxMemoryMB)
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test -timeout 2m -count=1 ./internal/ui/settingswin/ -run TestNewPositiveIntEntry
```

Expected: FAIL with `undefined: newPositiveIntEntry`.
If it passes, stop — the helper already exists and this task is done or the
tree is not the one this plan describes.

- [ ] **Step 3: Write the implementation**

In `internal/ui/settingswin/settingswin.go`, add this function immediately above
`build` (before the `// build lays out every control` comment). Do not change
`build` in this task.

```go
// newPositiveIntEntry is the numeric form field used by every standing integer
// preference except the picture-frame interval (that one is an int64-second
// count that has to survive a Duration multiply). Text is seeded from get
// without going through SetText, so opening the window does not round-trip the
// current value back into the host. OnChanged ignores anything that isn't a
// positive int, and when max > 0 also anything above that ceiling — the same
// mid-edit "leave the last good value in the host" behaviour the six copies had.
func newPositiveIntEntry(get func() int, set func(int), max int, validate fyne.StringValidator) *widget.Entry {
	e := widget.NewEntry()
	e.Validator = validate
	e.Text = strconv.Itoa(get())
	e.OnChanged = func(s string) {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && (max <= 0 || n <= max) {
			set(n)
		}
	}
	return e
}
```

`fyne.StringValidator` is already in scope via the existing
`"fyne.io/fyne/v2"` import. Do not add imports.

- [ ] **Step 4: Run the test to verify it passes**

```bash
goimports -local github.com/frathe/picfetch -w internal/ui/settingswin/settingswin.go internal/ui/settingswin/settingswin_test.go
go test -timeout 2m -race -count=1 ./internal/ui/settingswin/ -run TestNewPositiveIntEntry
```

Expected: PASS.

Then run the whole package so the unused-by-`build` helper has not broken
anything:

```bash
go test -timeout 2m -race -count=1 ./internal/ui/settingswin/
```

Expected: PASS. `build` still contains the six copies; that is this task's
point.

- [ ] **Step 5: Suggested commit (do not run git commit)**

```
add newPositiveIntEntry for the settings numeric fields

One helper seeds a positive-int Entry and applies OnChanged with an optional
ceiling, matching the six copied blocks in build. build itself still uses
the copies; the next commit switches them over.
```

**Parent review focus:** `Atoi` not `ParseInt`; seed assigns `Text` not
`SetText`; `max <= 0` is unbounded; `build` diff is empty.

---

### Task 2: Switch the six `build` copies onto the helper

**Subagent:** `go-expert` · **Model:** `composer-2.5-fast`

**Files:**
- Modify: `internal/ui/settingswin/settingswin.go` (`build` only, from
  `w.maxScanEntry =` through `w.maxFileSizeEntry`'s `OnChanged`)
- Modify: `internal/ui/settingswin/settingswin_test.go` (add two characterization
  tests; do not rewrite existing ones)
- Test: `go test -timeout 2m -race -count=1 ./internal/ui/settingswin/`

**Interfaces:**
- Consumes: `newPositiveIntEntry(get func() int, set func(int), max int, validate fyne.StringValidator) *widget.Entry`
  from Task 1; `positiveInt` local in `build`; `maxMemoryMB`.
- Produces: the six `Window` entry fields still populated, same names, same
  Host setters.

- [ ] **Step 1: Add the two missing characterization tests first**

These must fail if Task 1's helper is wired with the wrong `max` (always
`maxMemoryMB`, or always `0`). Append after `TestMaxScanEntry_InvalidTextIsIgnored`
and after `TestImgCacheEntry_ValidChangeCallsSetMaxImageCacheMB` respectively —
or at the end of the file, next to the tests they extend. Do not implement
the `build` rewrite until these exist.

```go
// TestMaxScanEntry_AcceptsAValueAboveMaxMemoryMB locks the unbounded path:
// scan count is not a memory budget, so a value that the three memory
// entries would reject must still reach SetMaxScan. Without this, wiring
// maxScan through newPositiveIntEntry(..., maxMemoryMB, ...) would still
// pass TestMaxScanEntry_ValidChangeCallsSetMaxScan (250000 < maxMemoryMB).
func TestMaxScanEntry_AcceptsAValueAboveMaxMemoryMB(t *testing.T) {
	host := &fakeHost{}
	w := New(testApp, host)
	w.Show()
	t.Cleanup(func() { w.win.Window().Close() })

	w.maxScanEntry.SetText("1048577")

	if len(host.maxScanCalls) != 1 || host.maxScanCalls[0] != 1048577 {
		t.Errorf("SetMaxScan calls = %v, want one call with 1048577 (no memory ceiling on scan count)", host.maxScanCalls)
	}
}

// TestImgCacheEntry_AcceptsMaxMemoryMB locks the inclusive ceiling the three
// memory OnChanged blocks already had (`n <= maxMemoryMB`). The invalid-text
// table only checks one-over (1048577).
func TestImgCacheEntry_AcceptsMaxMemoryMB(t *testing.T) {
	host := &fakeHost{}
	w := New(testApp, host)
	w.Show()
	t.Cleanup(func() { w.win.Window().Close() })

	w.imgCacheEntry.SetText("1048576")

	if len(host.imgCacheCalls) != 1 || host.imgCacheCalls[0] != maxMemoryMB {
		t.Errorf("SetMaxImageCacheMB calls = %v, want one call with %d", host.imgCacheCalls, maxMemoryMB)
	}
}
```

Run them **before** rewriting `build`. They describe existing behaviour, so
they must already pass against the six copies:

```bash
go test -timeout 2m -count=1 ./internal/ui/settingswin/ -run 'TestMaxScanEntry_AcceptsAValueAboveMaxMemoryMB|TestImgCacheEntry_AcceptsMaxMemoryMB'
```

Expected: PASS. If either fails, stop — the copies do not match this plan's
reading of `build` and the parent needs to amend the plan.

- [ ] **Step 2: Replace the six copies in `build`**

In `build`, keep `positiveInt` and the `intervalEntry` block exactly as they
are. Replace everything from `w.maxScanEntry = widget.NewEntry()` through the
`maxFileSizeEntry` `OnChanged` (inclusive) with:

```go
	w.maxScanEntry = newPositiveIntEntry(w.host.MaxScan, w.host.SetMaxScan, 0, positiveInt)

	maxScanItem := widget.NewFormItem(lang.L("Max files per folder scan"), w.maxScanEntry)
	maxScanItem.HintText = lang.L("Caps how many images a single recursive folder scan will gather")

	w.maxWidthEntry = newPositiveIntEntry(
		func() int { return int(w.host.MaxWindowWidth()) },
		func(n int) { w.host.SetMaxWindowWidth(float32(n)) },
		0,
		positiveInt,
	)

	w.maxHeightEntry = newPositiveIntEntry(
		func() int { return int(w.host.MaxWindowHeight()) },
		func(n int) { w.host.SetMaxWindowHeight(float32(n)) },
		0,
		positiveInt,
	)

	w.imgCacheEntry = newPositiveIntEntry(w.host.MaxImageCacheMB, w.host.SetMaxImageCacheMB, maxMemoryMB, positiveInt)

	imgCacheItem := widget.NewFormItem(lang.L("Max image cache (MB)"), w.imgCacheEntry)
	imgCacheItem.HintText = lang.L("Memory kept for recently viewed images")

	w.thumbCacheEntry = newPositiveIntEntry(w.host.MaxThumbCacheMB, w.host.SetMaxThumbCacheMB, maxMemoryMB, positiveInt)

	thumbCacheItem := widget.NewFormItem(lang.L("Max thumbnail cache (MB)"), w.thumbCacheEntry)
	thumbCacheItem.HintText = lang.L("Memory kept for grid-view thumbnails")

	w.maxFileSizeEntry = newPositiveIntEntry(w.host.MaxFileSizeMB, w.host.SetMaxFileSizeMB, maxMemoryMB, positiveInt)

	maxFileSizeItem := widget.NewFormItem(lang.L("Max file size (MB)"), w.maxFileSizeEntry)
	maxFileSizeItem.HintText = lang.L("Larger files are not opened at all")
```

Rules for this replacement:

- Use Host method values for the four `int` getters/setters (`MaxScan`,
  `SetMaxScan`, `MaxImageCacheMB`, `SetMaxImageCacheMB`, `MaxThumbCacheMB`,
  `SetMaxThumbCacheMB`, `MaxFileSizeMB`, `SetMaxFileSizeMB`). Do not wrap
  those in closures.
- Width and height **must** wrap: `get` is `int(w.host.MaxWindowWidth())` /
  `int(w.host.MaxWindowHeight())`, `set` is `w.host.SetMaxWindowWidth(float32(n))`
  / `SetMaxWindowHeight`. Do not change `Host`.
- `maxScan` / width / height pass `0`. The three memory fields pass
  `maxMemoryMB`.
- Leave `intervalEntry` as a hand-rolled `ParseInt` + `maxDurationSeconds`
  block. Leave every `NewFormItem` / `HintText` / check / slider as it is.
- Do not rename `Window` fields.

- [ ] **Step 3: Run the package tests**

```bash
goimports -local github.com/frathe/picfetch -w internal/ui/settingswin/settingswin.go internal/ui/settingswin/settingswin_test.go
go test -timeout 2m -race -count=1 ./internal/ui/settingswin/
```

Expected: PASS, including
`TestShow_SeedsEveryControlFromHostWithoutRoundTripping` (no Set* on open),
every `*Entry_ValidChange*` / `*InvalidTextIsIgnored`,
`TestMemoryEntries_InvalidTextIsIgnored`,
`TestMaxScanEntry_AcceptsAValueAboveMaxMemoryMB`, and
`TestImgCacheEntry_AcceptsMaxMemoryMB`.

- [ ] **Step 4: Suggested commit (do not run git commit)**

```
switch settings numeric fields onto newPositiveIntEntry

Max-scan, window size, and the three memory limits share one Entry helper.
The picture-frame interval stays a ParseInt path so a second count cannot
wrap time.Duration.
```

**Parent review focus:** `intervalEntry` diff is empty except for remaining
next to the helper calls. No leftover `widget.NewEntry()` for the six fields.
`maxScan` is `0`, memory fields are `maxMemoryMB`. Width/height still convert
through `int` / `float32`. Seeding test still passes.

---

### Task 3: Close the todo

**Subagent:** `generalPurpose` · **Model:** `composer-2.5-fast`

**Files:**
- Modify: `todos.md` only

**Interfaces:**
- Consumes: Tasks 1–2 landed; `newPositiveIntEntry` is what `build` uses.
- Produces: docs that match the tree.

- [ ] **Step 1: Backlog**

In `todos.md`:

1. Cut the whole `### Qodana: settingswin's numeric entries are five copies of one block`
   section (the heading plus its two-sentence paragraph, through `5 DuplicatedCode
   hits.`) out of `## TODO`.
2. Under `## Done` → `#### Internal`, add this bullet **below** the JPEG
   header-segment walk bullet:

```
- Settings numeric entries: `newPositiveIntEntry` in
  `internal/ui/settingswin/settingswin.go` is the one positive-int Entry
  constructor; max-scan, max-width, max-height, image-cache, thumb-cache,
  and max-file-size keep only their Host getter/setter (and the float32
  wrap on window size). The picture-frame interval stays a ParseInt +
  Duration path.
```

Do not edit other TODO items. Do not touch `ARCHITECTURE.md`.

- [ ] **Step 2: Verify docs-only**

```bash
go test -timeout 2m -race -count=1 ./internal/ui/settingswin/
```

Expected: PASS (no code change in this task). Parent confirms the Done bullet
is accurate and the TODO section no longer contains this Qodana item.

- [ ] **Step 3: Suggested commit (do not run git commit)**

```
close the settingswin numeric-entry Qodana todo
```

---

## Controller protocol (parent session)

1. Confirm Florian accepted the Open points defaults (or edit this plan first).
2. Dispatch Task N's implementer with **only that task's section** (via
   `task-brief`), the Global Constraints, and the produced signatures from
   earlier tasks. Do not paste the whole plan.
3. On DONE: read the diff. Fix anything that drifted from the specified code
   (`ParseInt`, `SetText` on seed, folding interval, a `*int` max, generics,
   a new file, `Host` changes, translation edits). Re-run the task's test
   command.
4. Hand Florian the suggested commit message. **Stop.**
5. After the commit lands, dispatch Task N+1.
6. After Task 3: optional whole-package glance at `settingswin.go`, still no
   `git commit` from the agent.

## Not in this plan

- **`intervalEntry`.** Different parse (`ParseInt` int64) and a different
  ceiling (`maxDurationSeconds`) so `time.Duration(n) * time.Second` cannot
  wrap. Forcing it onto `newPositiveIntEntry` would either drop that guard or
  infect the helper with a duration conversion it does not own.
- **Form labels / hints.** Not the duplicated six-line block.
- **`Host` type changes** (e.g. making window size `int`). The helper adapts
  at the call site.
- **Moving the helper into `internal/ui/widgets`.** Nothing else needs it.
- Qodana items below this one in `todos.md` (orientation transforms,
  `wrapAPP1`, doc comments, mechanical fixes, favorites menu bar, help
  observer, `maybeStartUpdateCheck` token order).
- Regenerating goldens, touching translations, or running Qodana CI.

## Risks and how each is contained

| Risk | Containment |
|------|-------------|
| Seed uses `SetText` and round-trips into the host on open | Helper assigns `e.Text`; `TestShow_SeedsEveryControlFromHostWithoutRoundTripping` + Task 1 seed subtest |
| `maxScan` accidentally gets `maxMemoryMB` | Task 2 `TestMaxScanEntry_AcceptsAValueAboveMaxMemoryMB` |
| Memory fields accidentally get `max == 0` | Existing `TestMemoryEntries_InvalidTextIsIgnored` (`1048577`) plus Task 2 ceiling-accepted test |
| Interval "simplified" onto the helper | Global constraint + parent review that `intervalEntry` still uses `ParseInt` |
| Width/height lose the `float32` conversion | Call-site wrappers in the specified `build` replacement; existing width/height tests |
| Helper exported or moved to `widgets` | Global constraint; parent rejects a new package path |
| Dirty-tree files get mixed into the change | Tasks list allowed files; parent `git diff --stat` before suggesting a commit |
