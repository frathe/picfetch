# Duplicate-visibility model extraction — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use `superpowers:subagent-driven-development`
> to execute this plan stage-by-stage. Steps use checkbox (`- [ ]`) syntax.
>
> **Controller extra (this session):** after every stage the parent agent reviews the
> subagent's diff line by line, fixes it up itself, runs that stage's verification
> commands, and then **stops** and hands Florian a suggested commit message. Do not
> dispatch Stage N+1 until Florian has confirmed the commit landed. Do not run
> `git commit` (`AGENTS.md`).

**Goal:** Move *which files are visible* out of `internal/ui/grid` and into a
viewer-independent `internal/dupes` package, so the core viewer stops polling an
overlay feature to answer navigation questions, and box the duplicate-hash engine
behind its own type inside the grid.

Closes `todos.md` "The duplicate-visibility model lives inside the grid feature"
(priority 16) plus `needs_refactoring.md` items **2**, **4**, **11**, **14**.

**Architecture after this plan:**

```
internal/dupes            (new, Fyne-free)
  ├── facts      hashes / native sizes / hash failures, keyed by URI string,
  │              generation-scoped (wipe vs. adopt)
  ├── distance   Hamming threshold, 0..32, default imaging.DuplicateMaxDistance
  ├── groups     Compute() off the UI goroutine → Groups snapshot; Install() on it
  ├── modes      hide-duplicates (standing) + inspect (survives grid close)
  ├── visibility IsVisible / NextVisible / FirstVisible / LastVisible /
  │              VisibleIndexesExcept — the file-set questions the viewer asks
  └── observers  OnChange(f) fan-out, fired on group install / mode / distance change

internal/ui               owns the Model, implements dupes.FileSet, subscribes for
                          jump-off-extra + menu resync, navigates through the model

internal/ui/grid          keeps presentation: badges, filter display, top bar,
                          marquee, browse-duplicates (an overlay filter like search),
                          and hashEngine — the pool-driven hashing pass that feeds
                          the model. Subscribes to the model to re-filter.
```

**Tech stack:** Go 1.26, Fyne v2.8, existing `internal/imaging` (`DifferenceHash`,
`DuplicateGroups`, `LoadThumbnailAndBounds`), `internal/decodepool`,
`internal/uitest.UIQueue`.

## Why this approach

The complaint in `todos.md` is precise: `nextVisibleIndex` / `firstVisibleIndex` /
`lastVisibleIndex` / `randomVisibleOther` ([viewer.go:940–1003](../internal/ui/viewer.go))
are filter-aware iteration implemented by poking a feature package, and plain arrow-key
navigation polls the grid overlay's state per index *while the overlay is closed*. That
inverts `ARCHITECTURE.md`'s own rule (features expose state; `internal/ui` composes them).

A separate package rather than a private `internal/ui` type, because the grouping logic
is the part that most wants isolated unit tests — today it is only reachable through a
`widget.GridWrap` and a Fyne test app. `internal/selection` and `internal/filesort` are
the precedent.

Alternatives rejected:

- **`internal/ui` private `visibleSet`:** least import churn, but the grid would have to
  receive the model back through `Host`, and the grouping maths stays untestable without
  a viewer.
- **`internal/ui/dupes` subpackage:** sits with the feature packages, which is exactly the
  "this is an overlay concern" framing the refactor is trying to undo.
- **Model owns browse too:** browse is cleared on every `Close`, filters only the overlay,
  and drives the escape ladder. Putting it in a viewer-independent package would drag a
  pure-presentation concern across the boundary for no caller's benefit.

## Locked decisions

Confirmed by Florian on 2026-08-27. Do not revisit without asking.

1. **Scope:** `needs_refactoring.md` items 2 + 4, plus the cheap cleanups 11 and 14 as a
   prep stage (both live in the files being moved).
2. **Package:** new top-level `internal/dupes`. Fyne-free — the file set is reached
   through a `FileSet` interface with string keys, not `fyne.URI`.
3. **Mode ownership:** the model owns **hide-duplicates** and **inspect**. The grid keeps
   **browse** (`browseHost`), its toast, its filter, and its escape ladder.
4. **Tests:** split by what they actually test. Grouping / representative / distance /
   visibility → new `internal/dupes` unit tests written against the model's own API.
   Overlay behaviour (filter, badges, browse, escape, throttled hash apply) stays in
   `internal/ui/grid`.
5. **Commits:** Florian commits after every stage the parent signs off. Each stage must
   leave the tree building, formatted, vetted and green on its own.
6. **Hash engine:** relocate `hashRemaining`'s completion closure **verbatim** — receiver
   and field paths only. Any restructuring into named steps is a separate later stage that
   is not part of this plan.
7. **No user-visible behaviour change.** Key bindings, menu enablement, titles, toasts,
   escape ordering, and the throttling cadence all stay exactly as they are. If a stage
   uncovers a latent bug, the subagent **reports** it and does not fix it.

## Global constraints

- **Do not run `git commit`.** End every stage with a suggested commit message.
- **Do not add `TODO`/`FIXME` comments.** Open work goes in `todos.md`.
- Every user-visible string stays `lang.L("English text")` with the same key in every
  `translations/*.json` bundle. **This refactor adds and removes none** — the two dupe
  strings (`"The images are currently being analyzed"`, `"Showing duplicates"`,
  `"Hiding duplicates"`) stay in `internal/ui/grid`, which keeps browse and the top bar.
- `internal/dupes` must not import `fyne.io/fyne/v2` or anything under `internal/ui`.
  Its only project import is `internal/imaging`.
- Feature packages own widgets/state and declare a narrow `Host`. Do not pass `appState`
  to anything. Do not invent a shared controller or registry.
- **Every `g.ui.Do` call must stay inside the `g.decodes.Go` body it belongs to.**
  `Settle`'s barrier is `decodes.Wait()`, which only covers completions the pool spawned;
  a completion queued from an untracked goroutine slips past it silently.
- Do not "simplify" `g.ui.Do` back to a direct `fyne.Do`, and do not add mutable
  package-level test seams. Runtime/test-configurable values belong on the owning type.
- Preserve `gofmt` and `goimports -local github.com/frathe/picfetch`. Tabs, not spaces.
- Comment density in this repo is high and deliberate. **Move the existing comments with
  the code they explain**, rewording only where the receiver or package name changed.
  A moved function that loses its "why" comment is a failed stage.
- `ARCHITECTURE.md` is updated once, in Stage 8 — not piecemeal.
- Golden screenshots do not change. Do not run `make golden`.
- Subagents stop after their own stage's verification and report. They never start the
  next stage.

## Subagent assignments

Least-powerful-model-that-can-do-it, per Florian's instruction. `go-expert` for design
and new code; `refactor-planner` for stages that are mechanical relocation against a
written plan (its defining habit is verifying the plan against the tree before editing).

| Stage | Agent | Model | Why |
|-------|-------|-------|-----|
| 0 — prep cleanups | `go-expert` | `sonnet` | Two small, well-specified edits; the sentinel unification touches `finishBrowse`, so not haiku. |
| 1 — dupes facts + groups | `go-expert` | `sonnet` | New package, ported logic, TDD. Fully specified below. |
| 2 — dupes modes + visibility | `go-expert` | `sonnet` | Pure functions with exact bodies given. |
| 3 — grid delegates to model | `refactor-planner` | **`opus`** | Generation wipe-vs-adopt semantics, 28 field pokes + 41 helper calls in tests, race-sensitive. |
| 4 — split dupes_test.go | `go-expert` | `sonnet` | Test triage against a written rubric. |
| 5 — viewer owns the model | `refactor-planner` | **`opus`** | Ownership flip plus observer ordering — the stage most able to change behaviour silently. |
| 6 — viewer navigates the model | `refactor-planner` | **`opus`** | Upgraded from `sonnet` after Stage 5: it is no longer purely mechanical. Moving the hide toggle to the model hits the same observer-expressiveness wall Stage 5 documented, so it carries a design element (`pushHideDuplicates`) alongside the 29 call-site swaps. |
| 7 — box the hash engine | `refactor-planner` | **`opus`** | The repo's most delicate concurrent code. Verbatim relocation, `-race -count=2`. |
| 8 — docs and backlog | `go-expert` | `sonnet` | `ARCHITECTURE.md` prose accuracy matters more than speed. |
| 9 — full verification | *parent* | — | Not delegated. |

## Target file structure

```
internal/dupes/                        NEW
  dupes.go        Model, FileSet, facts storage, generation, distance, observers
  groups.go       Groups value type, Compute/Install, group queries
  visible.go      hide + inspect modes and the visibility/navigation helpers
  dupes_test.go   facts, generation wipe vs adopt, distance clamp
  groups_test.go  grouping, representative choice, group membership
  visible_test.go hide/inspect modes, NextVisible/First/Last/VisibleIndexesExcept

internal/ui/grid/
  dupes.go        SHRINKS: browse + inspect-adjacent overlay glue, badge/filter feed
  hashengine.go   NEW (Stage 7): hashEngine — pool, throttle, job accounting
  dupes_test.go   SHRINKS: overlay behaviour only

internal/ui/
  visibility.go   NEW (Stage 6): dupeFileSet adapter + the viewer's navigation helpers
  viewer.go       SHRINKS: loses nextVisibleIndex/first/last/randomVisibleOther bodies
```

## Current code the implementers must not break

These are the invariants the existing tests are actually guarding. Every stage's
verification is "these still hold".

1. **Generation wipe vs. adopt.** `ensureHashGenLocked(gen)` **wipes** hashes, hashFailed
   and native when `hashGen != gen`. `adoptHashGen()` **keeps** them and adopts the host's
   generation, because an incremental shrink (`RemoveFiles` → `FilesChanged`) is not a new
   drop. Never route adopt through ensure. Orphan keys for deleted URIs are allowed to
   linger until the next full-set change.
2. **`hashFailed` is sticky per generation.** A file whose thumbnail decode failed is never
   retried by the hashing pass; without this, every Shift+D on a mixed-format drop re-raises
   the analyzing toast with no work left to do.
3. **Distance is snapshotted under the maps' lock.** `computeDuplicateGroups` reads
   `dupeDist` inside `hashMu` because hashing workers run off the UI goroutine.
4. **Representative = highest native pixel count; lowest host index on a tie.**
5. **`groupSizes[i]`: 0 = unhashed, 1 = unique-and-hashed, ≥2 = in a group.** Unhashed files
   are never hidden extras.
6. **`IsHiddenExtra(i)`** requires hide on **and** size ≥ 2 **and** `i != rep[i]`.
7. **Throttled apply.** `hideApply` stays set until the in-flight UI install returns;
   mid-window installs are floored by `hideApplyMinInterval` (250 ms); the last job
   (`remaining == 0`) always applies. Without the floor an idle UI queues one install per file.
8. **Browse waits for the last job.** `finishBrowse` runs only at `remaining == 0`, so a
   partially hashed group is never displayed. A source that turns out unique leaves browse
   off with no toast.
9. **Close semantics.** `Close()` clears inspect even when the overlay is already hidden
   (drop, picture-frame). `closeOverlay(false)` — the Return/click commit out of the
   variants grid — deliberately preserves inspect. Hide survives `Close`; browse does not.
10. **Escape ladder order** inside the grid: marquee → selection → search → browse → hide →
    close. One layer per press.
11. **`Warm()` records the hash only on a thumb-cache hit** and must not probe; native size
    is the hashing pass's job.
12. **Distance clamp is 0..32**; `imaging.DuplicateMaxDistance` (6) is the default when the
    user has never set it.
13. **Test UI queue.** Grid completions marshal through `g.ui` (`*uitest.UIQueue` under test),
    never `fyne.Do`. `internal/ui`'s `newTestUI` installs the same drainable queue.
14. **`groupComputes`** exists so tests can prove a snapshot was computed off the UI queue
    rather than inside it. It must survive the move in some form.

---

## Stage 0 — Prep cleanups (`needs_refactoring.md` 11 + 14)

**Agent:** `go-expert` · **Model:** `sonnet` · **Touches:** `internal/ui/grid/dupes.go`,
`internal/ui/grid/search.go`, their tests.

Both cleanups live inside code later stages relocate. Doing them first means they are
reviewed on their own instead of hiding inside a move diff.

- [ ] Extract `ensureMapsLocked()` from the three repeated nil-map guards shared by
      `ensureHashGenLocked` ([dupes.go:26](../internal/ui/grid/dupes.go)) and
      `adoptHashGen` ([dupes.go:51](../internal/ui/grid/dupes.go)). The two callers keep
      their distinct wipe-vs-keep semantics — `ensureHashGenLocked` still reallocates all
      three maps on a generation mismatch *before* calling the new helper.
- [ ] Unify the sentinel twins: `displayIndexOf` ([dupes.go:479](../internal/ui/grid/dupes.go))
      returns **0** on a miss, `displayIndexOfHost` ([search.go:173](../internal/ui/grid/search.go))
      returns **−1**. Delete `displayIndexOf`; make every caller use `displayIndexOfHost`
      and apply its own fallback explicitly.
      - Callers that wanted "default to the first cell": `Toggle` (grid.go, opening the
        grid on the current index) and `finishBrowse` (dupes.go, scrolling to the browse
        source). Both become `id := 0; if d := g.displayIndexOfHost(x); d >= 0 { id = d }`.
      - Re-read `restoreHighlight` (search.go) — it already uses the −1 form and must not
        change behaviour.
- [ ] Update the doc comment on `displayIndexOfHost`: it currently exists to warn about the
      twin that no longer exists. Say what it does instead.

**Verify:** `go build ./... && go vet ./... && gofmt -l internal/ui/grid && go test -race ./internal/ui/...`

**Report:** the two call sites that took an explicit fallback, and confirmation that no
behaviour depends on the old 0-sentinel anywhere else.

---

## Stage 1 — `internal/dupes`: facts, generation, distance, groups

**Agent:** `go-expert` · **Model:** `sonnet` · **Touches:** new `internal/dupes/` only.
Nothing imports it yet; the rest of the tree is untouched and must still build.

Write the tests first (`superpowers:test-driven-development`), then the code.

- [ ] `internal/dupes/dupes.go` — package doc explaining that this owns *which files are
      duplicates of which*, that it is Fyne-free, and that grouping facts are
      generation-scoped:

```go
// FileSet is the ordered file set the model groups over.
type FileSet interface {
    Count() int
    KeyAt(i int) string   // stable per file; the app passes URI strings
    Generation() uint64   // file-set revision; a change invalidates stored facts
}

const MaxDistance = 32    // was grid.maxDuplicateDistance

type Model struct{ /* set, mu, hashes, native, failed, gen, dist, groups, observers */ }

func New(set FileSet) *Model   // dist defaults to imaging.DuplicateMaxDistance
```

- [ ] Facts, ported from `rememberHash` / `rememberHashFail` / `rememberNative` /
      `hashOf` / `hashFailedOf` / `nativeSizeOf` / `pixelCountOf`:
      `PutHash(key string, h uint64)` (also clears the key's failed flag),
      `PutFailed(key string)`, `PutNativeSize(key string, sz image.Point)`,
      `Hash(key) (uint64, bool)`, `Failed(key) bool`,
      `NativeSize(key) (image.Point, bool)`, `PixelCount(key) (int, bool)`,
      `NativeSizeAt(i int) (w, h int, ok bool)` (the grid's exported `NativeSize`:
      false for out-of-range, unprobed, or a non-positive edge).
      Every read path calls `WipeIfStale()` first, exactly as today.
- [ ] Generation: `WipeIfStale()` (ensure semantics — wipe on mismatch),
      `AdoptGeneration()` (keep entries, adopt gen), `Clear()`. Carry invariant 1's
      comment across verbatim; it is the single most load-bearing comment in the file.
- [ ] Distance: `Distance() int`, `SetDistance(n int) bool` — clamps to `[0, MaxDistance]`,
      returns whether the stored value actually changed. Read/written under the same mutex
      as the maps (invariant 3).
- [ ] `internal/dupes/groups.go`:

```go
type Groups struct { Sizes, Reps []int; Dist int }
func (g Groups) Size(i int) int
func (g Groups) RepresentativeOf(i int) int

func (m *Model) Compute() Groups          // pure; safe off the UI goroutine
func (m *Model) Install(g Groups)         // UI goroutine
func (m *Model) Rebuild()                 // Install(Compute())
func (m *Model) GroupSize(i int) int
func (m *Model) RepresentativeOf(i int) int
func (m *Model) Members(i int) []int      // host-index order; nil when size < 2
func (m *Model) Computes() int32          // was groupComputes (invariant 14)
```

      `Compute` is `computeDuplicateGroups` moved verbatim: same `imaging.DuplicateGroups`
      call, same highest-pixel/lowest-index representative rule, same
      "hashed but ungrouped ⇒ size 1" pass.

- [ ] The installed snapshot is guarded by the same mutex as the maps. (Today
      `groupSizes`/`groupReps` are UI-goroutine-only by convention; guarding them is
      behaviour-neutral and removes a class of race. Do **not** use this as licence to
      change the `g.ui.Do` discipline in the grid — that stays.)
- [ ] `OnChange(f func())` appends an observer; `Notify()` fires them in registration
      order. Nothing fires them yet in this stage.
- [ ] Tests: a `fakeSet` implementing `FileSet`. Cover — hash/native round-trip; `PutHash`
      clearing a prior failure; wipe on generation change; **adopt keeping entries**;
      distance clamp at both ends and the changed/unchanged return; grouping of an exact
      pair; a three-file group; representative = highest pixel count; representative =
      lowest index on a pixel tie; unhashed files getting size 0 and never being extras;
      `Members` returning nil below two.

**Verify:** `go build ./... && go vet ./... && gofmt -l internal/dupes && go test -count=1 -race -cover ./internal/dupes/...`
Target ≥ 90 % statement coverage for the new package (the repo's norm). Use `-count=1`:
a `(cached)` result proves nothing about code written moments ago. Check the package's
**direct** imports only — see Stage 9 for why `go list -deps` is the wrong tool here.

**Report:** the exported API as built, coverage number, and anything in the ported code
whose intent was unclear enough that you had to guess.

---

## Stage 2 — `internal/dupes`: hide/inspect modes and visibility

**Agent:** `go-expert` · **Model:** `sonnet` · **Touches:** `internal/dupes/` only.

- [ ] `internal/dupes/visible.go`, modes first:
      `HideDuplicates() bool`, `SetHideDuplicates(on bool) bool` (returns whether it
      changed; **does not** hash, filter, or jump — those are the observers' jobs),
      `BeginInspect(i int)` (stores the key at `i`; out of range or missing clears
      inspect), `ClearInspect()`, `Inspecting() bool`, `InspectSource() int` (linear
      scan for the stored key, −1 when absent), `InspectMembers() []int`.
- [ ] `IsHiddenExtra(i int) bool` moves here from groups.go if it reads the hide flag —
      hide on, size ≥ 2, `i != rep[i]` (invariant 6).
- [ ] Visibility, ported from `viewer.go:940–1003`:

```go
func (m *Model) IsVisible(i int) bool                   // !IsHiddenExtra(i)
func (m *Model) NextVisible(from, delta int) int
func (m *Model) FirstVisible() int
func (m *Model) LastVisible() int
func (m *Model) VisibleIndexesExcept(current int) []int
```

      `NextVisible` reproduces `nextVisibleIndex` exactly, in order:
      1. `if members := m.InspectMembers(); len(members) >= 2 && delta != 0` → step within
         the members ring (port `stepInMembers` and `absInt` from viewer.go as unexported
         helpers here).
      2. `if !m.HideDuplicates() || delta == 0` → `return from + delta` **unclamped**.
         `ShowImage` is what normalises it; do not "fix" this.
      3. Otherwise walk `delta` steps modulo `Count()`, skipping hidden extras, returning
         `from` if the walk wraps all the way round.
      `FirstVisible`/`LastVisible` return **0** when nothing is visible (today's fallback),
      and do not check the hide flag — `IsHiddenExtra` is already false when hide is off.
- [ ] Randomness stays in `internal/ui`. `VisibleIndexesExcept(current)` returns the
      candidate list only; the viewer keeps its tested `randomOtherIndex` and does the draw.
- [ ] Tests: hide toggle return value; inspect begin/clear/out-of-range; inspect source
      after the key moves index; `NextVisible` in each of the three branches including the
      all-hidden wrap-around; `First`/`LastVisible` with the first and last file hidden;
      `VisibleIndexesExcept` excluding `current` and excluding extras.

**Verify:** `go build ./... && go vet ./... && gofmt -l internal/dupes && go test -race -cover ./internal/dupes/...`

**Report:** confirmation that `NextVisible`'s branch order matches viewer.go's, and any
place where the ported code's behaviour looked accidental rather than intended (report,
do not fix — locked decision 7).

---

## Stage 3 — `grid.Overview` delegates to an embedded `*dupes.Model`

**Agent:** `refactor-planner` · **Model:** `opus` · **Touches:** all of
`internal/ui/grid/`. The largest and most delicate mechanical stage.

The grid still *constructs* the model here — ownership moves to the viewer in Stage 5.
Every exported `Overview` method keeps its current name and signature, so `internal/ui`
and its tests are untouched by this stage.

- [ ] Add a `hostSet` adapter in `internal/ui/grid` implementing `dupes.FileSet` over
      `Host` (`Count` → `FileCount`, `KeyAt(i)` → `FileAt(i).String()`, `Generation` →
      `Generation`). Guard `KeyAt` against a nil URI the way the current code does.
- [ ] `Overview` gains `dupes *dupes.Model`, created in `New`. **Delete** these fields:
      `hashMu`, `hashes`, `native`, `hashFailed`, `hashGen`, `dupeDist`, `groupSizes`,
      `groupReps`, `groupComputes`, `hideDupes`, `inspectKey`. Keep `browseHost`,
      `hashing`, `hashJobs`, `hideApply`, `hideApplyAt` (Stage 7 moves those).
- [ ] Rewrite the deleted helpers as delegations: `rememberHash` → `g.dupes.PutHash(u.String(), imaging.DifferenceHash(img))`,
      `rememberNative`, `rememberHashFail`, `hashOf`, `hashFailedOf`, `nativeSizeOf`,
      `pixelCountOf`, `wipeHashesIfStale`, `clearHashes`, `adoptHashGen`,
      `rebuildGroups`, `computeDuplicateGroups`, `groupSize`, `groupMembers`,
      `inspectSource`.
      Keep the nil-URI and nil-image guards where they are today; the model takes strings
      and knows nothing about `fyne.URI`.
- [ ] Exported methods become one-liners onto the model: `NativeSize`, `IsHiddenExtra`,
      `RepresentativeOf`, `HideDuplicates`, `BeginInspect`, `ClearInspect`,
      `InspectingDuplicates`, `InspectMembers`, `SetDuplicateDistance`,
      `duplicateDistance`.
- [ ] `SetHideDuplicates` keeps its full body (hash pass, `applyFilter`,
      `jumpIfHiddenExtra`, `fireDupeState`) — only the flag read/write goes to the model,
      via `SetDuplicateDistance`-style "did it change" short-circuiting. Same for
      `SetDuplicateDistance`'s live regroup branches (browse → `finishBrowse`; hide →
      `applyFilter` + `jumpIfHiddenExtra`; else `rebuildGroups`).
- [ ] `hashRemaining`'s completion closure keeps its exact shape. `g.groupSizes, g.groupReps = sizes, reps`
      becomes `g.dupes.Install(snap)`; `g.duplicateDistance() != snapDist` becomes
      `g.dupes.Distance() != snap.Dist`. Nothing else in that closure moves in this stage.
- [ ] Grid tests: the ~28 direct field pokes and ~41 unexported-helper calls in
      `dupes_test.go` (and any in `grid_test.go`, `nav_test.go`, `selection_test.go`,
      `thumbs_test.go`) must be re-pointed at `g.dupes.*` or at the surviving helper.
      **Do not delete or weaken a single assertion in this stage** — Stage 4 is where
      tests move.
- [ ] `groupComputes` assertions become `g.dupes.Computes()`.

**Verify:** `go build ./... && go vet ./... && gofmt -l internal/ui/grid`, then
`go test -count=1 -race ./internal/ui/grid/...` and
`go test -count=2 -timeout 20m -race ./internal/ui/...`.

`-count=2` because the throttling and last-job assertions are timing-adjacent — a pass
that only happens once is not a pass. **`-timeout 20m` is mandatory with `-count=2`:**
`internal/ui` alone runs ~300s, so two passes blow through Go's default 10-minute limit
and abort with a goroutine dump that looks like a hang but is only a timeout. AGENTS.md's
CI line uses `-timeout 20m` for the same reason.

**Report:** the full list of deleted fields, any assertion that could not be re-pointed
without a rewrite (name it, do not rewrite it), and confirmation that
`ensureHashGenLocked`'s wipe and `adoptHashGen`'s keep are still distinct paths.

---

## Stage 4 — Split `grid/dupes_test.go` by what it tests

**Agent:** `go-expert` · **Model:** `sonnet` · **Touches:** `internal/ui/grid/dupes_test.go`,
`internal/dupes/*_test.go`.

Rubric — a test moves to `internal/dupes` when it asserts on grouping, representative
choice, distance, hash/native storage, generation wipe-vs-adopt, hidden-extra
classification, inspect membership, or visibility stepping, **and** needs no overlay,
no widget, and no `Host`. Everything else stays: filter/`matches` mapping, badge
rendering, top-bar text, browse, escape ordering, throttled apply, `Warm`, `Settle`,
job accounting, `FilesChanged` retarget.

- [ ] Move qualifying tests, rewriting them against the model's API and a `fakeSet`
      instead of `newOverview`/`openGrid`. Keep each test's name recognisable
      (`TestDuplicateGroups_...`) so `git log -S` still finds its history.
- [ ] Where a moved test also asserted something about the overlay, **split it in two**
      rather than dropping either half.
- [ ] Leave `internal/ui/grid/dupes_test.go` with the overlay half, its helpers intact.
- [ ] No net loss of assertions. State the before/after counts in the report.

**Verify:** `go test -race -cover ./internal/dupes/... ./internal/ui/grid/...` and confirm
`internal/ui/grid` coverage did not drop by more than 2 points (the moved lines are now
covered next door).

**Report:** test counts before/after in each package, both coverage numbers, and the list
of tests that had to be split.

---

## Stage 5 — The viewer owns the model; observers replace the callback chain

**Agent:** `refactor-planner` · **Model:** `opus` · **Touches:** `internal/ui/features.go`,
`menu.go`, `memlimits.go`, `viewer.go`, `internal/ui/grid/grid.go`, `dupes.go`,
`internal/ui/harness_test.go`.

This is the ownership flip. Behaviour must be identical; the risk is observer **ordering**.

- [x] **Lock-ordering watch item.** `dupes.Model.Compute` calls `set.KeyAt(i)` *while
      holding the model's mutex* (faithful to the original, which called
      `g.host.FileAt(i)` under `hashMu`). `dupeFileSet.KeyAt` must therefore stay a plain
      `v.state.files[i].String()` — it must not lock, call back into the model, or reach
      for anything that might. If it ever needs to, hoist the key slice out of the lock
      in `Compute` instead.
- [x] Add `dupeFileSet{v *viewer}` in `internal/ui` implementing `dupes.FileSet` from
      `v.state.files` and `v.fileSetRevision` (the viewer already exposes `FileCount`,
      `FileAt`, `Generation` — reuse them; `KeyAt(i)` is `v.FileAt(i).String()`).
- [x] `registerFeatures` ([features.go:47](../internal/ui/features.go)) creates
      `view.dupes = dupes.New(dupeFileSet{view})` **before** `grid.New`, and passes it:
      `view.grid = grid.New(view, window, view.dupes)`. Construction order in that file is
      load-bearing — the model must exist before the grid and before
      `view.grid.SetDuplicateDistance(...)`.
- [x] `SetDuplicateDistance` ([memlimits.go:153](../internal/ui/memlimits.go)) calls
      `v.dupes.SetDistance(n)` instead of `v.grid.SetDuplicateDistance(n)`. The
      settings-slider path (`settingswin` → `Host.SetDuplicateDistance`) is unchanged.
- [ ] Register two observers, in this order:
      1. **grid** (in `grid.New`): re-run the hashing pass when hide just turned on,
         then `applyFilter`/`applyVisibleFilter` and repaint. This is where
         `SetHideDuplicates`'s old body lands.
      2. **viewer** (in `registerFeatures`): `jumpIfHiddenExtra` (moved out of grid —
         it calls `ShowImage`, which is the viewer's job) followed by
         `updateActionsMenuState`.
      Registration order **is** fire order; say so in a comment at both sites. Grid
      filters before the viewer jumps, matching today's
      `applyFilter` → `jumpIfHiddenExtra` → `fireDupeState` sequence.
- [ ] Replace `view.grid.SetOnDupeStateChanged(view.updateActionsMenuState)`
      ([menu.go:186](../internal/ui/menu.go)) with the model observer. Keep
      `SetOnDupeStateChanged`/`fireDupeState` **only** if browse still needs its own
      notification; if it does, keep it and say why in the report.
- [x] `jumpIfHiddenExtra` moves to `internal/ui`: `if v.dupes.Inspecting() { return }`,
      then `if i := v.state.index; v.dupes.IsHiddenExtra(i) { v.ShowImage(v.dupes.RepresentativeOf(i)) }`.
- [x] `newTestUI` (`harness_test.go`) must build the model the same way production does.
      Check every test helper that constructs a viewer or a grid.

**Implementer's note (2026-08-27, Stage 5 as landed).** The two boxes above that are
still open could not be executed as written; they are a decision for Florian, not the
implementer, so they were left unticked and the behavioural invariant was preserved
instead. Evidence: in `hashRemaining`'s completion closure
([grid/dupes.go:398-411](../internal/ui/grid/dupes.go)) the jump fires on
`HideDuplicates() && !InspectingDuplicates()` for **every** throttled apply, while the
menu resync fires on `remaining == 0` regardless of hide. Those two conditions are not
nested, so a single parameterless observer list cannot carry both without either
double-firing `updateActionsMenuState` at `remaining == 0` (Trap A — `refreshMainMenu`
rebuilds the native macOS menu bar) or rebuilding the menu on every mid-pass apply
(≤4/s during a hash pass, where today it never fires at all). What landed instead:

- **One** model observer, registered in `registerFeatures`: `viewer.jumpIfHiddenExtra`.
  `grid.New` registers none — every grid transition already re-filters inline, and
  making the grid's reaction an observer would either double-filter or reset the
  highlight on the hide-off/no-browse distance path that today only calls
  `rebuildGroups`.
- `SetOnDupeStateChanged` / `fireDupeState` survive **unchanged, at all eight sites**,
  carrying only the menu resync. The menu reads `BrowsingDuplicates` and
  `SourceDuplicateGroupSize` (`actionmenu.go:30-32`), which the model does not own
  (locked decision 3), so this channel had to survive for browse in any case.
- The viewer-originated distance change reaches the grid's live regroup through the
  explicit `viewer.pushDuplicateDistance` → `Overview.DuplicateDistanceChanged` pair
  (`memlimits.go`, `grid/dupes.go`), which sequences "grid re-filters, then observers
  fire" at the call site rather than through registration order.

**Verify:** `go build ./... && go vet ./... && make fmt-check && go test -count=2 -timeout 20m -race ./...`

**Report:** the exact observer fire order at each entry point (hide toggle, distance
change, hash landing, `FilesChanged`), and any place where the old order could not be
reproduced.

**Verification (2026-08-27):** `go build ./...`, `go vet ./...`, `make fmt-check` all
exit 0. `go test -count=2 -race ./internal/ui/grid/... ./internal/dupes/...` exit 0.
`go test -count=1 -timeout 20m -race ./...` exit 0 (whole tree green, `internal/ui`
315s, golden e2e included). No pre-existing failures were found at HEAD `1e8825b`.
The `-count=2` pass over `./...` is the parent agent's gate, not run here.

---

## Stage 6 — Viewer navigation and mode checks read the model

**Agent:** `refactor-planner` · **Model:** `sonnet` · **Touches:** `internal/ui/viewer.go`,
`keys.go`, `actionmenu.go`, `windowmenu.go`, new `internal/ui/visibility.go`, tests.

The payoff stage: after this, plain navigation never touches the grid.

- [x] New `internal/ui/visibility.go` holding `dupeFileSet` (moved from Stage 5 if it
      landed elsewhere) and the four navigation helpers, now thin:
      `nextVisibleIndex(from, delta)` → `v.dupes.NextVisible(from, delta)`;
      `firstVisibleIndex()` → `v.dupes.FirstVisible()`;
      `lastVisibleIndex()` → `v.dupes.LastVisible()`;
      `randomVisibleOther(current)` keeps the `randomOtherIndex` fallback when hide is off
      and otherwise draws from `v.dupes.VisibleIndexesExcept(current)`.
      Delete the old bodies plus `stepInMembers` and `absInt` from `viewer.go`.
      **Leave `randomOtherIndex` and its tests in `load.go`/`slideshow_test.go` alone.**
- [x] **The hide toggle needs the same shape as the distance push (design settled in
      Stage 5; do not re-derive it).** `toggleHideDuplicates` cannot simply become
      `v.dupes.SetHideDuplicates(!v.dupes.HideDuplicates())`: when hide turns **on** the
      grid must run `hashRemaining()` and then `applyFilter()`, and when it turns **off**
      it must only `applyFilter()`. A parameterless `OnChange` observer cannot express
      that difference. Mirror `pushDuplicateDistance` exactly:

      ```go
      // internal/ui/memlimits.go or visibility.go, next to pushDuplicateDistance
      func (v *viewer) pushHideDuplicates(on bool) {
          if !v.dupes.SetHideDuplicates(on) {
              return
          }
          v.grid.HideDuplicatesChanged(on)   // grid: hash if on, then applyFilter
      }
      ```

      `Overview.HideDuplicatesChanged(on)` is the back half of today's
      `SetHideDuplicates` — `hashRemaining()` when on, `applyFilter()`, `Notify()` when
      on, `fireDupeState()` — in that exact order. `Overview.SetHideDuplicates(on)` then
      becomes `SetDistance`-shaped: model flag, then `HideDuplicatesChanged`. The grid's
      own D key keeps calling `SetHideDuplicates`; the viewer calls `pushHideDuplicates`.
      **Preserve the operation order at both entry points** and prove it in the report.
- [x] `Overview.SetDuplicateDistance` has had **no production caller** since Stage 5 —
      only 11 grid test call sites. Either delete it and give the tests a harness helper,
      or keep it and say why. Do not leave an exported production method alive solely for
      tests without saying so.
- [x] Swap the remaining viewer-side call sites from `v.grid.*` to `v.dupes.*`:
      - `viewer.go`: `NativeSize` (gridHighlightTitle), `ClearInspect` (clearToDropzone).
      - `keys.go`: `InspectingDuplicates` ×4 (Escape, P, G, D).
      - `actionmenu.go`: `HideDuplicates` ×4, `InspectingDuplicates`, `ClearInspect`,
        and `toggleHideDuplicates` → `v.dupes.SetHideDuplicates(!v.dupes.HideDuplicates())`.
      - `windowmenu.go`: `InspectingDuplicates`.
      **Leave alone:** `BrowsingDuplicates`, `SetBrowsingDuplicates`,
      `ToggleBrowseDuplicates`, `SourceDuplicateGroupSize`, `Visible`, `Toggle`, `Close`.
      Browse stays a grid concern (locked decision 3), and
      `SourceDuplicateGroupSize` deliberately follows the grid's highlight while the
      overlay is open.
- [x] Delete the `Overview` delegators that now have no caller. Keep the ones the grid
      itself still uses internally (`groupSize`, `RepresentativeOf`, `IsHiddenExtra` feed
      `applyVisibleFilter` and `applyDupBadge`) — those become direct `g.dupes.*` calls.
- [x] Update `internal/ui` tests that assert through `v.grid.HideDuplicates()` /
      `IsHiddenExtra` / `RepresentativeOf` / `InspectingDuplicates` (`step_test.go`,
      `actionmenu_test.go`, `grid_test.go`) to assert through `v.dupes`. Assertions
      about the *overlay* (`v.grid.Visible()`, `BrowsingDuplicates`) stay as they are.

**Verify:** `go build ./... && go vet ./... && make fmt-check && go test -count=2 -timeout 20m -race ./...`
Then prove the seam is gone:
`grep -rn "grid\.\(IsHiddenExtra\|InspectMembers\|InspectingDuplicates\|HideDuplicates\|RepresentativeOf\|NativeSize\)(" internal/ui/*.go`
(the trailing `(` is load-bearing: without it the pattern also matches
`grid.HideDuplicatesChanged`, the method Stage 6 deliberately adds)
must return nothing.

**Report:** that grep's output (expected: empty), the count of call sites changed, and the
`Overview` methods deleted.

**Implementer's note (2026-08-27, Stage 6 as landed).** Three details differ from the
bullets above, all of them the bullets being out of date rather than a design change:

- The `toggleHideDuplicates` body named in the call-site bullet
  (`v.dupes.SetHideDuplicates(!v.dupes.HideDuplicates())`) is superseded by the bullet
  above it: it is `v.pushHideDuplicates(!v.dupes.HideDuplicates())`, so the grid still
  hashes and re-filters. `actionmenu.go` holds three `HideDuplicates` *reads*, not four
  — the fourth site is that toggle.
- `Overview.SetDuplicateDistance` was **deleted**. Its grid-test call sites (10, not the 11 the bullet estimates) now go
  through `setDuplicateDistance(g, n)` in `harness_test.go`, which is
  `viewer.pushDuplicateDistance` over the grid's own model - same "did it change"
  short-circuit, so the idempotence assertions in
  `TestSetOnDupeStateChanged_SetDuplicateDistanceWhileHide` still bite.
  `Overview.SetHideDuplicates` survives by contrast because it has a production caller:
  the grid's own D key and escape ladder (`nav.go`).
- The stage's acceptance grep is **not** empty as written: `grid\.HideDuplicates`
  matches the new `v.grid.HideDuplicatesChanged(on)` in `visibility.go`, which the
  settled design requires. Anchoring the pattern with a `(` -
  `grep -rn "grid\.\(IsHiddenExtra\|InspectMembers\|InspectingDuplicates\|HideDuplicates\|RepresentativeOf\|NativeSize\)(" internal/ui/*.go`
  - returns nothing, which is the check that was meant.

`internal/dupes` changed in one test file only: `TestStepInMembers_Wraps` moved from
`internal/ui/step_test.go` into `visible_test.go`, because `stepInMembers` itself is
gone from `viewer.go`. No production code in that package was touched.

---

**Verification (2026-08-27):** `go build ./...`, `go vet ./...`, `make fmt-check` all
exit 0. `go test -count=2 -race ./internal/ui/grid/... ./internal/dupes/...` exit 0.
`go test -count=1 -timeout 20m -race ./...` exit 0 (whole tree green, `internal/ui`
320s, golden e2e included). No pre-existing failures were found at HEAD `8e08add`, so
no baseline worktree run was needed. The `-count=2` pass over `./...` is the parent
agent's gate, not run here.

## Stage 7 — Box the hash engine (`needs_refactoring.md` item 4)

**Agent:** `refactor-planner` · **Model:** `opus` · **Touches:** new
`internal/ui/grid/hashengine.go`, `internal/ui/grid/dupes.go`, `grid.go`, tests.

**Verbatim relocation only** (locked decision 6). The completion closure's body does not
get restructured, reordered, or renamed in this stage.

- [x] New `hashEngine` type in `internal/ui/grid/hashengine.go` owning the machinery
      fields moved off `Overview`: `hashing sync.Map`, `hashJobs atomic.Int32`,
      `hideApply atomic.Bool`, `hideApplyAt atomic.Int64`, plus the references it needs
      (`pool *decodepool.Pool[...]`, `thumbs *imaging.ByteCache[image.Image]`,
      `ui UIQueue`, `model *dupes.Model`, `host Host`). Move `hideApplyMinInterval` and
      `shouldScheduleHideApply` with it.
- [x] `Run(apply func(snap dupes.Groups, remaining int32, gen uint64))` is `hashRemaining`'s
      job-building and worker body, returning the same pending count. The engine keeps job
      accounting, dedup, the failure/probe path, and the throttle decision.
- [x] The completion's **UI-side body moves to `(*Overview).applyHashSnapshot`**, passed in
      as `apply`. Copy it byte-for-byte; only the receiver and the field paths change. It
      keeps needing `g.fileIndex(g.highlight)`, `g.browseHost`, `finishBrowse`,
      `g.dupes.Inspecting()` and `applyVisibleFilter` — all still on `Overview`, which is
      exactly why the split lands here.
- [x] `g.ui.Do` stays **inside** the `pool.Go` body (global constraint). Verify this by
      reading, not by assuming: `Settle`'s `decodes.Wait()` barrier depends on it.
- [x] `Overview` keeps `hashRemaining()` as a one-line call into the engine, so
      `SetHideDuplicates`, `SetBrowsingDuplicates` and the tests keep their entry point.
- [x] Re-point the tests that reach `g.hashJobs` / `g.hideApply` / `g.hideApplyAt` /
      `g.hashing` at `g.hashes.*` (or whatever the field is named). Do not relax a
      throttling assertion to make it pass — if one fails, stop and report.

**Verify:** `go build ./... && go vet ./... && make fmt-check`, then
`go test -count=3 -race ./internal/ui/grid/...` (the package that changed; three passes
because the throttle is timing-adjacent), then the full
`go test -timeout 20m -race ./...`. Never run `-count=N` over `./internal/ui/...`
without `-timeout 20m` — see Stage 3.

**Report:** a diff summary proving the closure body is unchanged apart from receiver and
field paths (e.g. `git diff -w` line counts), and the final field list on `Overview`.

**Implementer's note (2026-08-27, Stage 7 as landed).** The move is verbatim where the
plan says it must be and mechanical everywhere else; three details are worth recording:

- The completion's UI-side body landed in `(*Overview).applyHashSnapshot`
  ([grid/dupes.go:315](../internal/ui/grid/dupes.go)) **byte-for-byte** - the 24 lines
  dedented by four tabs hash identically to `88bb8f9`'s `dupes.go:366-389`. Not even the
  receiver changed: the body was already `g.*` and it moved onto `*Overview`. The one
  line that did not travel with it is `defer g.hideApply.Store(false)`, which stays in
  the engine wrapping the `apply` call (`hashengine.go:152-155`) because `hideApply` is
  now the engine's field. Semantics are unchanged: the deferred store still runs after
  the UI body returns, which is invariant 7.
- The engine calls `dupes.Model` directly where `hashRemaining` went through
  `Overview`'s `hashOf` / `pixelCountOf` / `hashFailedOf` / `rememberHash` /
  `rememberHashFail` / `rememberNative` wrappers. Those wrappers stay on `Overview`
  for the cell and `Warm` paths; the engine cannot reach them and moving them was not
  in this stage's scope. Each substitution is the wrapper's own body minus its
  nil-`fyne.URI` guard, which is unreachable in this loop - it already dereferences the
  URI for the string key it dedups and caches by. `key := u.String()` is hoisted three
  lines, above three pure model reads.
- `Overview.ThumbCacheFull`'s budget check became a package-level
  `thumbCacheFull(*imaging.ByteCache[image.Image])` in `thumbs.go`, so the engine's
  speculative-write guard and the exported method cannot drift apart.

**Verification (2026-08-27):** `go build ./...`, `go vet ./...`, `make fmt-check` all
exit 0. `go test -count=3 -race ./internal/ui/grid/...` exit 0. `go test -count=1
-timeout 20m -race ./...` exit 0 (whole tree green, `internal/ui` 322s, golden e2e
included). No pre-existing failures at HEAD `88bb8f9` - the same gates were run on the
clean tree before any edit - so no baseline worktree was needed. The `-count>1` pass
over `./...` is the parent agent's gate, not run here.

---

## Stage 8 — Docs and backlog

**Agent:** `go-expert` · **Model:** `sonnet` · **Touches:** `ARCHITECTURE.md`,
`todos.md`, `needs_refactoring.md`.

- [ ] `ARCHITECTURE.md`:
      - new package-table row for `internal/dupes` (near `internal/selection`, line ~252),
        stating that it is Fyne-free and owns hide/inspect + visibility;
      - update the `internal/ui/grid/` row (line ~72): it keeps browse, badges, filter,
        marquee and the hash engine, and reads the model;
      - update the `internal/ui` file table with `visibility.go`;
      - update "How does hide-duplicates work?" (line ~320) and the grid/thumbnail entry
        (line ~341) to point at `internal/dupes` first, then `grid/hashengine.go`;
      - `viewer.go`'s row loses the navigation helpers.
      `AGENTS.md` requires this update in the same change as the package move — it is
      landing one stage later by design, and Stage 9 is what makes that whole.
- [ ] `todos.md`: move "The duplicate-visibility model lives inside the grid feature" out
      of TODO. Add a line under **Done → Internal** naming the new package.
- [ ] `needs_refactoring.md`: mark items **2**, **4**, **11**, **14** resolved with the
      date, and update the "Suggested sequencing" section — step 3 is done, so item 3
      (the `viewer` god object) becomes the next candidate. Re-check item 3's field
      counts against the tree before restating them; this refactor changed them.
- [ ] **Log the latent findings surfaced during Stages 5–7 in `todos.md`.** All are
      pre-existing and were deliberately not fixed (locked decision 7: no behaviour
      change). Each needs enough detail to be actionable later:
      1. **`viewer.FileAt(i)` is unguarded and read off the UI goroutine.** It is a bare
         `v.state.files[i]`; `dupes.Model.Compute` calls `Count()` then `KeyAt(i)` from a
         hash worker while the UI goroutine can replace `v.state.files`. A shrink between
         the two reads panics the worker, and the slice is read there unsynchronised. The
         pre-extraction `hostSet` had the identical shape, so this is not new — but the
         generation check is the only thing making it rare.
      2. **`hideApplyAt` is never cleared between hashing passes.**
         `shouldScheduleHideApply(0)` sets `hideApply` but leaves `hideApplyAt` at the
         previous pass's last mid-window timestamp, so a pass starting within 250 ms of
         that skips its own first mid-window apply. The last job always applies, so
         nothing is lost permanently — it is a latency wart, not a correctness bug.
      3. **The hashing pass has no nil-URI guard** although every neighbouring helper
         does. `Host.FileAt(i)` returning nil panics the pass.
      4. **`NextVisible` is O(n) per arrow key** — `InspectMembers` → `InspectSource`
         scans the whole set for the inspect key on every step, then `IsHiddenExtra`
         takes the model mutex once per candidate. Unchanged from the pre-extraction
         cost, but now concentrated in one place and visible at 50k files.
      5. **`applyVisibleFilter` computes `hostRep` unconditionally**, including when
         browse is off and `browseHost == -1`. The value is unused there; costs one
         model-mutex acquisition per filter pass.
- [ ] Check `README.md` and `internal/ui/help` for any text describing hide-duplicates
      internals. User-facing behaviour is unchanged, so expect no edits — confirm rather
      than assume.

**Verify:** `make fmt-check && go build ./... && go test -race ./...` (docs stage, but the
tree must still be green) and `grep -rn "internal/dupes" ARCHITECTURE.md` returns the new rows.

**Report:** the sections edited and confirmation that no user-facing doc needed changing.

---

## Stage 9 — Full verification (parent agent, not delegated)

- [ ] `make fmt-check`
- [ ] `go vet ./...`
- [ ] `go build ./...`
- [ ] `go test -timeout 20m -race ./...` from the repository root
- [ ] `grep -rn "TODO\|FIXME" --include='*.go' internal/ main.go` → nothing new
- [ ] Confirm `internal/dupes` imports neither `fyne.io/fyne/v2` nor `internal/ui`
      **directly**:
      `go list -f '{{join .Imports "\n"}}' ./internal/dupes` → exactly
      `internal/imaging`, `image`, `sync`, `sync/atomic` (plus whatever Stage 2 adds).
      Do **not** use `go list -deps` here: it is transitive, and `internal/imaging`
      itself imports `fyne.io/fyne/v2` for its `fyne.URI` file I/O, so a `-deps` grep
      can never come back empty. `.Imports` is authoritative — it is the import set
      the compiler actually resolved. Do **not** bolt a `grep -rn "fyne" internal/dupes/`
      onto this either: the package's own doc comments legitimately use the words "Fyne",
      "internal/ui" and "math/rand" to explain what it must not import, so that grep
      always hits prose.
- [ ] Manual smoke via `make run`: D toggles hide; arrows skip extras with the grid closed;
      Shift+D opens the variants grid; Return commits and the inspect loop steps within the
      group; Escape reopens browse; G from inspect reopens variants; the settings distance
      slider regroups live; Home/End land on visible files.
- [ ] Suggested commit message for the final stage.

---

## Execution notes for the parent agent

- **One stage per dispatch.** After each: read the whole diff (`git diff`), fix what the
  subagent got wrong yourself rather than re-dispatching for small things, re-run that
  stage's verification, then stop and hand Florian a commit message.
- **Verify mechanically, not by report.** A subagent saying "all tests pass" is not
  evidence; run the command. (This repo has bitten us on exactly that before.)
- **Comment loss is the most likely silent failure.** After each move stage, spot-check
  that the invariants in "Current code the implementers must not break" still have their
  explanatory comments attached to the moved code.
- **If a stage fails verification twice, stop and escalate to Florian** rather than
  dispatching a third attempt.
- **Do not let a subagent widen scope.** If Stage 3 wants to restructure `hashRemaining`,
  that is Stage 7's verbatim move and then a follow-up plan — not now.

## Self-review

- No placeholders or TBDs; every stage names its files, its verification command, and its
  report contract.
- Stages 1–2 add code nothing imports, so both are independently revertable. Stages 3–7
  each leave the tree green on their own, which is what makes per-stage commits possible.
- Scope check: items 2, 4, 11 and 14 only. Item 3 (`viewer` god object) and item 5 (menu
  recompute) are explicitly out, though Stage 5 makes item 5's seam more visible.
- Ambiguity check: the one judgement call left to an implementer is Stage 4's move/stay
  rubric, which is written out rather than left to taste.
