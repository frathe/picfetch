# PicFetch — Refactoring Backlog

Codebase-quality audit, 2026-08-27. Ordered by severity (High → Low); within a
tier, by priority score `(Impact + Risk) × (6 − Effort)`, each dimension 1–5
(effort inverted: cheaper fixes rank higher).

**Overall health is well above average** and the list below is graded on a
curve: `go vet` clean, zero `TODO/FIXME` markers in production code, all 30
test packages green, ~38k lines of tests against ~26.5k lines of code,
coverage 90–100% everywhere except thin OS-integration seams, an up-to-date
`ARCHITECTURE.md`, and a working refactoring cadence (`finished_refactorings/`).
The debt that remains is concentrated in one god object (item 3, the
`viewer`) and one upstream library bug (item 1). The second god object and
the inverted dependency — items 4 and 2 — were resolved on 2026-08-27 by
the `internal/dupes` extraction, along with items 11, 13 and 14.

---

## High severity

### 1. HEIC decodes leak native memory (dependency debt)

- **Impact 3 · Risk 5 · Effort 1 → priority 40**
- Where: `go.mod` — `github.com/gen2brain/heic v0.7.1`, imported by
  `internal/imaging/loader.go:29` and `internal/imaging/exif.go:12`.
- The upstream wazero-based decoder leaks native memory on every decode
  (gen2brain/heic issue #15; fix PR #16 filed by this project 2026-08-20).
  v0.7.1 is still the latest release and the tree carries no `replace`
  directive or local mitigation, so a session browsing HEIC-heavy folders
  grows RSS without bound — multi-GB growth was observed during diagnosis.
  This is invisible to Go heap profiling (native memory), so it will not
  resurface in routine pprof checks.
- **Why it matters**: crash-grade for end users with iPhone photo libraries —
  the app's core audience for this format.
- **Fix**: watch for the upstream release containing PR #16 and bump. If it
  stalls, add a `replace` to the patched fork — one line, immediately
  shippable. Either way, add a note in `AGENTS.md` so the pin isn't forgotten.
- **Mitigation (2026-08-26):** `go.mod` replace → `frathe/heic@0ac0a39` until
  upstream releases PR #16. Remove replace on bump.

### 3. `viewer` god object — 87 fields and still growing

- **Impact 4 · Risk 3 · Effort 4 → priority 14**
- Where: the struct definition alone spans
  [viewer.go:38–478](internal/ui/viewer.go:38); its methods spread across
  ~30 files of `internal/ui`. (Grew by one field, `dupes *dupes.Model`,
  when the duplicate-visibility model moved out from under the grid — item
  2's resolution; the navigation helpers it replaced were methods, not
  fields, so they didn't shrink the count on the way out.)
- The comments are exemplary and the tests thorough, which is why this is
  Risk 3 and not 5 — but the growth pattern is intact: autoupdate landed 6
  new fields, the info overlay 7, menu items account for **16 fields** on
  their own. Every feature keeps paying a "where in the 430-line struct does
  my field go" tax, and the concurrency notes per field only get harder to
  hold in one head.
- **Fix**: continue the existing feature-split practice with field-cluster
  extractions that have clean seams already visible in the comments:
  - menu-item state (`saveItem` … `actionsTrashItem`, 16 fields) → a
    `menus` type with a single recompute entry point (pairs with item 5);
  - updater state (`update`, `updateDir`, `updateOp`, `updateDone`,
    `updateCurrentVersion`, `updateDayMu`) → an `updater` type in
    autoupdate.go;
  - info-overlay state (`infoVisible`, `infoText`, `exifLink`, `infoCard`,
    `currentFileSize`, `currentHasEXIF`, `currentPreview`) → info.go;
  - display state (`displayFrames`, `displayFrameIdx`, `rotation`,
    `fadeAnim`) → load.go/rotate.go.

---

## Medium severity

### 5. Menu Checked/Disabled state is synchronized manually from every mutation site

- **Impact 3 · Risk 3 · Effort 2 → priority 24** (highest score in the list —
  cheap and unlocks item 3's menu extraction)
- Where: `updateFileMenuState`, `updateActionsMenuState`,
  `updateWindowMenuState`, `refreshMainMenu` must be remembered at every
  state-changing site — rotate.go, load.go, save.go, `clearToDropzone`,
  drop.go, and more.
- The tell: `HighlightChanged`
  ([viewer.go:536–559](internal/ui/viewer.go:536)) snapshots four booleans,
  calls `applyActionsMenuState`, then hand-diffs them to decide whether the
  native menu needs a refresh. That's an ad-hoc change-detection layer
  bolted onto push-based invalidation.
- **Fix**: one `syncMenus()` that recomputes all Checked/Disabled state from
  the model and internally diffs before touching the native bar, called from
  a small number of choke points (end of every user action). Deletes the
  per-site call discipline and the manual diffing.

### 6. `appState` is anemic — file-set invariants enforced from outside

- **Impact 1 · Risk 2 · Effort 2 → priority 12** (was 20; the revision half
  landed 2026-08-28)
- **Half done.** The revision no longer lives outside `appState`: the
  `fileSetRevision` counter is gone, and `setFiles` / `clearFiles` /
  `removeFile` / `reorder` each end in `publish()`, which republishes the
  file-set snapshot under a bumped generation as a matter of construction
  rather than caller convention. `viewer.Generation()` reads that snapshot.
  See the duplicate-snapshot work in `plans/2026-08-28-dupes-followups.md`.
- Where: [state.go](internal/ui/state.go) still leaves one invariant to the
  caller — `RemoveFile` ([viewer.go:820](internal/ui/viewer.go:820)) evicts
  the image cache itself, so a future mutator that forgets that call leaks
  decodes of files that are gone. drop.go and sort.go still apply
  merge/sort ordering from outside.
- **Why it matters**: cache eviction is now the only file-set invariant
  nothing but convention enforces. The generation and the
  files/unsortedFiles sync are both internal to `appState`.
- **Fix**: let the viewer subscribe to `appState` for cache eviction, so
  removal evicts without the caller remembering to. Still shrinks item 3 as
  a side effect.

### 7. Updater dependency tree: sigstore-go + TUF in a desktop image viewer

- **Impact 2 · Risk 3 · Effort 5 → priority 5** (accepted debt — document,
  don't act)
- Where: `go.mod` — `sigstore-go`, `sigstore`, `go-tuf/v2` pull in gRPC,
  OpenTelemetry, the go-openapi suite, certificate-transparency-go, and
  k8s klog: the majority of the ~120 indirect dependencies exist to verify
  release signatures for the in-app updater.
- **Why it matters**: binary size, build time, CVE-scanner noise, and the
  sigstore ecosystem's fast API churn all land on an app whose job is
  showing pictures. The choice is deliberate and security-motivated, and
  `internal/update` isolates it well — so this is a *watch* item: revisit
  whether a TUF-only + bundle-verification subset can replace the full
  verifier at the next major sigstore-go bump, and keep the dependency out
  of any package but `internal/update`.

---

## Low severity

### 8. Favorites preview prewarm competes with the foreground for one budget

- **Impact 2 · Risk 2 · Effort 3 → priority 12**
- Where: [favthumbs.go](internal/ui/favthumbs.go) (`gridSink`),
  [sync.go](internal/favthumbs/sync.go).
- Partially mitigated since first noted: `gridSink.Store` stops offering to
  the in-memory cache at `ThumbCacheFull()`, so the pass no longer churns
  its own entries. What remains: a pass over a huge favorite (50k files)
  still decodes everything for the disk cache at `syncConcurrency = 4`,
  competing for CPU/IO with whatever the user is doing, and the head-fills-
  the-budget strategy assumes the user will browse the favorite's head next.
- **Options** (open design question, not a bug): cap the prewarm's share of
  the thumb budget; idle-priority workers; skip the in-memory offer entirely
  above a set size and rely on the disk cache.

### 9. Mode-interaction guards scattered through `handleKeyEvent`

- **Impact 2 · Risk 2 · Effort 3 → priority 12**
- Where: [keys.go:70–338](internal/ui/keys.go:70). The dispatcher itself is
  flat and mostly commentary — length is not the issue. The issue is that
  mode-composition rules (slideshow vs. grid vs. inspect vs. scan/sort
  cancellation) are encoded positionally: an Escape priority chain plus
  per-key `v.dupes.Inspecting()`/`slides.Active()` exceptions inside `P`,
  `G`, and `D`. Each new mode multiplies cases. Item 2 (now resolved) moved
  where this state lives, not how these guards are structured — the
  positional encoding is unchanged; if tackled alone, a small
  mode-precedence table centralizes the rules.

### 10. OS-seam test coverage (accepted)

- **Impact 2 · Risk 2 · Effort 4 → priority 8**
- `winpos` 27.9%, `filepicker` 49.4%, `wallpaper` 61.7%, `trash` 66.1%,
  `clipboard` 66.9% — native-API glue the fyne test driver cannot reach.
  Acceptable as long as the seams stay thin; keep logic out of these
  packages so the uncovered surface stays pure OS calls.

### 12. cgo `copyTitle` returns a shared static buffer

- **Impact 1 · Risk 2 · Effort 1 → priority 15**
- Where: [windowmenu_darwin.go:33](internal/ui/windowmenu_darwin.go:33).
  Safe only while every caller stays on the AppKit main thread and never
  holds two results at once — neither constraint is written down. Return a
  `strdup`'d copy freed by the Go side (or document the constraint at the
  function). `testKeepAlive` also grows unboundedly, though it is test-only.
---

## Suggested sequencing

Alongside feature work, in this order:

1. **Now (minutes–hours)**: 12 — mechanical cleanup; and set a recurring
   check on the heic release for item 1 (or land the `replace`).
2. **Next (a day)**: 5 (menu recompute) and what is left of 6 (cache
   eviction into `appState`; the revision half landed 2026-08-28) — both
   shrink the viewer's surface and delete call-site discipline, preparing
   item 3.
3. **Done (2026-08-27, staged)**: 2 + 4 — moved the visibility/grouping
   model out of the grid into `internal/dupes`, then boxed the hash engine
   into `grid/hashengine.go`. See their own resolved notes above and
   `todos.md` Done → Internal.
4. **Next candidate (the big one)**: 3 — the `viewer` god object, one
   field-cluster extraction per sitting (menus first, since step 2 clears
   the way), continuing until the struct comment fits on two screens.
5. **Watch list**: 1 (bump on release), 7 (revisit at next sigstore-go
   major), 8 (design decision when favorites grow), 10 (keep seams thin).
