# Batched duplicate-visibility accessor — Implementation Plan

> **For agentic workers:** this plan is executed by dispatching one fresh
> subagent per stage, with a review gate between stages. Steps use
> checkbox (`- [ ]`) syntax for tracking. Do not run `git commit`.

**Goal:** Close both open items in `todos.md`:

1. Delete the dead duplicate-hash wrappers in `internal/ui/grid/dupes.go`
   (`rememberHashFail`, `hashFailedOf`) and the stale comment that names
   one of them.
2. Export a batched visibility accessor from `internal/dupes` and have
   `Overview.applyVisibleFilter` read it **once per filter pass** instead
   of taking the model mutex once — in fact twice — per file index.

**Tech stack:** Go, Fyne v2, `internal/dupes`, `internal/ui`,
`internal/ui/grid`. No new dependencies.

**Spec:** this document. There is no separate spec file.

---

## Design

### The defect

`applyVisibleFilter` ([search.go:120][s120]) walks every host file and,
per index, takes the `dupes.Model` mutex **twice**:

```go
if browseFilter && g.dupes.RepresentativeOf(i) != hostRep { continue } // lock
if hide && g.dupes.IsHiddenExtra(i)                      { continue } // lock
```

plus two more before the loop (`g.dupes.GroupSize(g.browseHost)` at
:126, `g.dupes.HideDuplicates()` at :129) and one conditional
`RepresentativeOf(g.browseHost)` at :137. At 50k files with hide or
browse on, one keystroke in the search bar is up to **2N + 3** mutex
acquisitions on the UI goroutine.

`todos.md` names only the `IsHiddenExtra` half. The `RepresentativeOf`
call on :144 is the same defect in the same loop and is in scope for
this change (confirmed with the user).

This is the identical one-lock-per-candidate pattern the `Snapshot` work
removed from `internal/dupes/visible.go`'s navigation walks. That work
solved it *inside* the package with an unexported pair — `visibility()`
returning `(hide, Groups)` and a lock-free `hiddenExtra(hide, groups, i)`
([visible.go:93-116][v93]). The grid cannot reach either.

### The fix

Promote that unexported pair into one exported immutable value, mirroring
how `Snapshot` already models "an immutable view the caller reads many
times from one acquisition":

```go
// Visibility is the model's hide flag and installed group snapshot
// frozen at one read.
type Visibility struct {
    Hide   bool
    Groups Groups
}

func (v Visibility) HiddenExtra(i int) bool
func (v Visibility) Visible(i int) bool
func (v Visibility) RepresentativeOf(i int) int
func (v Visibility) Size(i int) int

func (m *Model) Visibility() Visibility // exactly one mu acquisition
```

`Visibility` is a value: `Groups` holds two slices that are never mutated
in place (`Install` replaces the struct wholesale), so a caller holding a
`Visibility` reads a consistent hide-plus-groups pair however the model
changes underneath it. That is the same immutability argument
`snapshot.go`'s header comment makes for `Snapshot`.

This subsumes both the unexported `visibility()` and the free
`hiddenExtra()` helper — the package ends with **one** concept for
"visibility read once, tested many times" instead of two, and
`visible.go` gets shorter, not longer.

`applyVisibleFilter` then becomes **one** mutex acquisition for the whole
pass, down from 2N + 3:

```go
vis := g.dupes.Visibility()
browseFilter := browsing && vis.Size(g.browseHost) >= 2
hide := vis.Hide && !browsing
hostRep := vis.RepresentativeOf(g.browseHost)
for i := range g.host.FileCount() {
    ...
    if browseFilter && vis.RepresentativeOf(i) != hostRep { continue }
    if hide && vis.HiddenExtra(i)                        { continue }
    ...
}
```

### Where the value is read

`applyVisibleFilter` must take its `Visibility` **at its own top**, not
earlier. Both entry paths install groups immediately before calling it —
`rebuildFilter` calls `g.rebuildGroups()` first ([search.go:114][s114]),
and `applyHashSnapshot` calls `g.dupes.Install(snap)` first
([grid/dupes.go][gd]) — so a read at the top of the function always sees
the snapshot that pass is meant to filter against.

### Test seam

`Model` gains a `visibilityReads atomic.Int32` incremented by
`Visibility()`, exposed as `VisibilityReads() int32`. This mirrors the
existing `computes` counter exactly ("so tests can prove a snapshot was
computed off the UI queue rather than inside it",
[groups.go:155][g155]) and lets a grid test assert *one read per filter
pass regardless of file count* — the regression guard that stops the
per-index pattern coming back. Approved with the user; it is model state,
not a mutable package-level test seam, so it does not violate AGENTS.md.

### Deliberately out of scope

- `applyDupBadge` ([nav.go:211][n211]) and the cell updater
  ([grid.go:416][g416]) call `GroupSize(i)` for a **single** index each.
  Correct as they are.
- `restoreHighlight` ([search.go:207][s207]) calls `RepresentativeOf`
  once, not per index. Leave it; threading `vis` in would only churn a
  signature.
- No behaviour changes anywhere. Every stage below is either a deletion
  of dead code or a refactor with identical observable output.

### Stale-comment trap (read this before Stage 4)

`search.go:132-135` carries a comment explaining why `hostRep` is
computed *conditionally*:

> Computing it unconditionally spent a model-mutex acquisition per filter
> pass on a value nothing read.

That premise dies with this change — once `vis` is in hand,
`vis.RepresentativeOf(g.browseHost)` is a bounds-checked slice read
costing nothing, and `Groups.RepresentativeOf(-1)` already answers `-1`
for the not-browsing case. Stage 4 must compute `hostRep`
unconditionally **and delete that comment**, not leave it contradicting
the code.

[s114]: ../internal/ui/grid/search.go#L114
[s120]: ../internal/ui/grid/search.go#L120
[s207]: ../internal/ui/grid/search.go#L207
[v93]: ../internal/dupes/visible.go#L93
[g155]: ../internal/dupes/groups.go#L155
[gd]: ../internal/ui/grid/dupes.go
[n211]: ../internal/ui/grid/nav.go#L211
[g416]: ../internal/ui/grid/grid.go#L416

---

## Stage 1 — Delete the dead duplicate-hash wrappers

**Agent:** `general-purpose` · **Model:** `haiku`
**Why this model:** a mechanical deletion with the dead-ness already
verified; no design judgement involved.

**Files:** `internal/ui/grid/dupes.go`, `internal/ui/grid/hashengine.go`

- [x] 1.1 Delete `rememberHashFail` (`dupes.go:33-38`) and `hashFailedOf`
      (`dupes.go:59-65`). Both have zero callers in the tree, tests
      included.
- [x] 1.2 In `hashengine.go:120-126`, the comment reads "rather than
      through Overview's `hashOf`/`pixelCountOf`/`hashFailedOf` wrappers".
      Drop the `hashFailedOf` mention; `hashOf` and `pixelCountOf` are
      still live and must stay named.
- [x] 1.3 Do **not** touch `dupes.Model.PutFailed` / `Model.Failed` — the
      hash engine calls both directly (`hashengine.go:133`, `:175`).

**Review gate (I run these):**
```
grep -rn "rememberHashFail\|hashFailedOf" --include="*.go" .   # expect: no hits
go build ./... && go vet ./...
go test ./internal/ui/grid/...
```

---

## Stage 2 — Add `dupes.Visibility` (purely additive, TDD)

**Agent:** `go-expert` · **Model:** `sonnet`
**Why this model:** the API is fully specified above, so this is
careful-but-bounded Go with a high doc-comment bar — exactly what the
repo-specific Go agent is for. Escalate to `opus` only if the review gate
fails twice.

**Files:** `internal/dupes/visible.go`, `internal/dupes/visible_test.go`,
`internal/dupes/dupes.go`

Place `Visibility` in `visible.go`, immediately above `IsHiddenExtra`,
where the unexported `visibility()`/`hiddenExtra()` pair lives today.
Do **not** create a `visibility.go` — a `visible.go`/`visibility.go` pair
would be a coin-flip for every future reader.

- [x] 2.1 **Tests first.** Add to `visible_test.go`:
      - `TestVisibility_AgreesWithPerIndexAccessors` — for hide on and
        off, over a set containing a representative, a hidden extra, a
        hashed-unique file, an unhashed file, and out-of-range indices
        (`-1`, `Count()`), assert `vis.HiddenExtra(i) == m.IsHiddenExtra(i)`,
        `vis.Visible(i) == m.IsVisible(i)`,
        `vis.RepresentativeOf(i) == m.RepresentativeOf(i)`, and
        `vis.Size(i) == m.GroupSize(i)`.
      - `TestVisibility_IsAFrozenRead` — take a `Visibility`, then call
        `m.SetHideDuplicates(!hide)` and `m.Install(differentGroups)`, and
        assert the held value's answers are unchanged.
      - `TestVisibility_TakesOneModelReadPerCall` — `VisibilityReads()`
        goes up by exactly 1 per `Visibility()` call, and testing 64
        indices off one held value adds none.
      Existing tests must not be edited in this stage.
- [x] 2.2 Add the `Visibility` type and its four methods. Each method
      delegates to the embedded `Groups` (`Size`, `RepresentativeOf`) or
      reproduces `hiddenExtra`'s exact three-step test (hide off → false;
      `Groups.Size(i) < 2` → false; else `i != RepresentativeOf(i)`).
      Keep the "unhashed files are never extras: size 0 fails the size
      check on its own" reasoning in the doc comment — it is the
      non-obvious part.
- [x] 2.3 Add `Model.Visibility() Visibility`: one `m.mu` acquisition
      returning `Visibility{Hide: m.hide, Groups: m.groups}`.
      Document *why* it exists (a caller testing many indices must not pay
      a lock per index) and point at `Snapshot` as the sibling idiom.
- [x] 2.4 Add `visibilityReads atomic.Int32` to `Model` (`dupes.go`) next
      to `computes`, incremented in `Visibility()`, with a
      `VisibilityReads() int32` accessor whose comment states the same
      purpose `Computes()`'s does.
- [x] 2.5 **Leave every existing caller alone.** This stage adds; Stage 3
      migrates. Nothing outside the new tests may change behaviour.

**Review gate (I run these):**
```
make fmt-check && go vet ./...
go test -race ./internal/dupes/...
git diff --stat          # expect: only visible.go, visible_test.go, dupes.go
```
Plus my read of the new doc comments against the package's existing
"explain the why, not the what" standard.

---

## Stage 3 — Migrate `internal/dupes`'s own callers

**Agent:** `go-expert` · **Model:** `sonnet`
**Why this model:** a pure refactor fenced in by an existing test suite
that already asserts the properties at risk.

**Files:** `internal/dupes/visible.go`

- [x] 3.1 Rewrite `IsHiddenExtra` and `IsVisible` on top of
      `Visibility()`.
- [x] 3.2 Rewrite the four walks — `NextVisible`, `FirstVisible`,
      `LastVisible`, `VisibleIndexesExcept` — to take one `Visibility()`
      and call `vis.HiddenExtra(i)` per index.
- [x] 3.3 Delete the now-unused `visibility()` method and the free
      `hiddenExtra()` function. Move any reasoning worth keeping from
      their comments onto `Visibility` / `Model.Visibility()` rather than
      dropping it.
- [x] 3.4 `NextVisible`'s documented branch order (inspect ring → hide-off
      passthrough → skipping walk) and its unclamped `from + delta` return
      are load-bearing and must not move. Same for `FirstVisible` /
      `LastVisible`'s fallback-to-0.
- [x] 3.5 No test file may be edited in this stage. If a test fails, the
      production change is wrong.
- [x] 3.6 `Visibility.HiddenExtra`'s doc comment (added in Stage 2)
      currently opens "reproduces IsHiddenExtra's exact test against the
      frozen hide flag and Groups, in the same order". Once 3.1 makes
      `IsHiddenExtra` delegate *to* `HiddenExtra`, that sentence points
      the wrong way — the primary would be describing itself as a copy of
      its own caller. Reword it to state the test on its own terms, and
      let `IsHiddenExtra`'s comment be the one that defers ("the
      single-index entry point; a walk should take a Visibility once
      instead"). The "unhashed files are never extras — size 0 fails the
      size check on its own" reasoning currently appears on three
      functions; after `hiddenExtra` is deleted it should survive on
      `Visibility.HiddenExtra` alone, with `IsHiddenExtra` pointing at it.

**Review gate (I run these):**
```
make fmt-check && go vet ./...
go test -race ./internal/dupes/...
git diff --stat internal/dupes    # expect: visible.go only, net negative
grep -n "func (m \*Model) visibility\|func hiddenExtra" internal/dupes/visible.go  # expect: no hits
```
The `countingSet` tests (`TestNextVisible_TakesOneSnapshot*`) staying
green is the proof the walks did not regress into per-candidate reads.

---

## Stage 4 — One visibility read per filter pass

**Agent:** `go-expert` · **Model:** `sonnet`
**Why this model:** the call-site rewrite is small, but it sits in the
grid's filter path where `matches`, `browseHost` and the hide flag
interact; it needs a reader who will check the branch conditions rather
than pattern-match. Escalate to `opus` if the review gate fails twice.

**Files:** `internal/ui/grid/search.go`, `internal/ui/grid/search_test.go`,
`internal/ui/visibility.go`

- [x] 4.1 In `applyVisibleFilter`, take `vis := g.dupes.Visibility()`
      once at the top of the function, and source **all five** of these
      from it: the pre-loop `GroupSize(g.browseHost)`, `HideDuplicates()`,
      and `RepresentativeOf(g.browseHost)`, plus the in-loop
      `RepresentativeOf(i)` and `IsHiddenExtra(i)`.
- [x] 4.2 Compute `hostRep` unconditionally from `vis` and **delete the
      three-line comment above it** — see "Stale-comment trap" in the
      Design section. Leaving that comment in place would be worse than
      leaving the code alone.
- [x] 4.3 Preserve every branch condition exactly: `hide` is
      `vis.Hide && !browsing`; `browseFilter` is
      `browsing && vis.Size(g.browseHost) >= 2`; the outer
      `if nameFilter || hide || browseFilter` guard that decides whether
      `matches` stays `nil` (nil means "no filter", which `count()` and
      `fileIndex()` both depend on) is unchanged.
- [x] 4.4 In `internal/ui/visibility.go`, `jumpIfHiddenExtra` calls
      `IsHiddenExtra(i)` then `RepresentativeOf(i)` — two acquisitions for
      one decision. Fold to one `Visibility()` read. Behaviour identical,
      including the inspect-session early return above it.
- [x] 4.5 **Tests.** In `search_test.go`:
      - `TestApplyVisibleFilter_TakesOneVisibilityReadPerPass` — build a
        grid over enough files that a per-index read would be obvious,
        turn hide on, record `g.dupes.VisibilityReads()`, run one filter
        pass, assert the delta is exactly 1 and independent of file count.
      - Parity tests over the four filter combinations — search only,
        hide only, browse only, and search+hide — asserting `g.matches`
        holds the same host indices as before this change.
      Follow the package's `UIQueue`/`Settle` conventions from
      `harness_test.go`; never sleep to await a pass.

**Review gate (I run these):**
```
make fmt-check && go vet ./...
go test -race ./internal/ui/... ./internal/dupes/...
git diff internal/ui/grid/search.go     # read every branch condition by hand
```

---

## Stage 5 — Docs and backlog

**Agent:** `general-purpose` · **Model:** `sonnet`
**Why this model:** prose against a house style, no code.

**Files:** `ARCHITECTURE.md`, `todos.md`

- [x] 5.1 `ARCHITECTURE.md:299` — the `internal/dupes` file table row for
      `visible.go` should name `Visibility` alongside the existing
      `BeginInspect` / `InspectMembers` / `IsHiddenExtra` / `NextVisible`.
      Line 366's "How does hide-duplicates work?" index entry gets the
      same treatment. Do not restructure either table.
- [x] 5.2 `todos.md` — remove both closed items from `## TODO` and add
      entries under `## Done`, matching the existing voice (state what
      changed and why it mattered, not "fixed a bug"). The batched-accessor
      entry belongs under `#### Internal`.
- [x] 5.3 Do not touch `needs_refactoring.md` — neither item appears there.

**Review gate (I run these):** read the diff; confirm no invented claims
about what shipped.

---

## Stage 6 — Full verification (I run this, not a subagent)

- [x] 6.1 `make fmt-check`
- [x] 6.2 `go vet ./...`
- [x] 6.3 `go build ./...`
- [x] 6.4 `go test -timeout 20m -race ./...` from the repo root
- [ ] 6.5 Move this plan to
      `finished_refactorings/2026-08-28-batched-visibility.md`
- [ ] 6.6 Hand over a suggested commit message. **Do not `git commit`** —
      AGENTS.md reserves that for the user.

---

## Risks

| Risk | Mitigation |
|---|---|
| A `Visibility` taken before groups are installed filters against a stale snapshot. | Stage 4.1 fixes the read site at the top of `applyVisibleFilter`; both callers install groups immediately before calling it. Called out in Design. |
| Stage 3 quietly changes `NextVisible`'s branch order or its unclamped return. | Stage 3.5 forbids editing tests; the existing `NextVisible` suite covers all three branches plus the wrap-to-start case. |
| The `VisibilityReads` counter tempts later code into treating `Visibility()` as expensive. | Its doc comment states it is one mutex acquisition and the counter exists only for tests, same as `Computes()`. |
| Stale comments left contradicting the new code. | Explicit steps: 1.2 (`hashFailedOf` mention), 3.3 (`visibility()` reasoning), 4.2 (`hostRep` conditional). |
| Subagent reports success without running the suite. | Every stage has a review gate whose commands are run by the coordinating session, not taken on the subagent's word. |
