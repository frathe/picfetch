# PicFetch — Refactoring Backlog

Codebase-quality audit, 2026-08-27. Ordered by severity (High → Low); within a
tier, by priority score `(Impact + Risk) × (6 − Effort)`, each dimension 1–5
(effort inverted: cheaper fixes rank higher).

**Overall health is well above average** and the list below is graded on a
curve: `go vet` clean, zero `TODO/FIXME` markers in production code, all 30
test packages green, ~38k lines of tests against ~26.5k lines of code,
coverage 90–100% everywhere except thin OS-integration seams, an up-to-date
`ARCHITECTURE.md`, and a working refactoring cadence (`finished_refactorings/`).
The debt that remains is one upstream library bug (item 1, mitigated) and
the updater's dependency footprint (item 7, accepted debt to watch). The
second god object and the inverted dependency — items 4 and 2 — were
resolved on 2026-08-27 by the `internal/dupes` extraction, along with items
11, 13 and 14. Items 5 (menu recompute) and 6 (`appState` cache eviction)
were resolved on 2026-08-28. Item 3, the `viewer` god object, is retired
rather than re-scored: it had already moved to `todos.md`'s TODO section
ahead of the plan that closed it, and now lives in `todos.md` Done →
Internal — four field-cluster extractions took `viewer` from 87 fields to
55, and what's left is largely widget references and already-grouped value
structs (`state`, `settings`, `vector`, `display`), a materially weaker
complaint than the original that no longer clears the bar for a scored
entry here.

---

## High severity

### 1. HEIC decodes leak native memory (dependency debt) [temp fix applied]

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

---

## Medium severity

### 5. Menu Checked/Disabled state is synchronized manually from every mutation site — DONE

- **Impact 3 · Risk 3 · Effort 2 → priority 24** (highest score in the list —
  cheap and unlocks item 3's menu extraction)
- Where: `updateFileMenuState`, `updateActionsMenuState`,
  `updateWindowMenuState`, `refreshMainMenu` had to be remembered at every
  state-changing site — rotate.go, load.go, save.go, `clearToDropzone`,
  drop.go, and more.
- The tell: `HighlightChanged` snapshotted four booleans, called
  `applyActionsMenuState`, then hand-diffed them to decide whether the
  native menu needed a refresh. That was an ad-hoc change-detection layer
  bolted onto push-based invalidation.
- **Fix**: one `syncMenus()` that recomputes all Checked/Disabled state from
  the model and internally diffs before touching the native bar, called from
  a small number of choke points (end of every user action). Deletes the
  per-site call discipline and the manual diffing.
- **Resolved (2026-08-28):** `internal/ui/menus.Menus.Apply(State) (changed
  bool)` recomputes the whole File/Window/Actions matrix as one pure
  function of a value snapshot; `menu.go`'s `menuState()` builds that
  snapshot in the one place, and `syncMenus()` applies it, refreshing the
  native bar only when `Apply` reports `changed`. Replaces
  `updateFileMenuState`/`updateActionsMenuState`/`updateWindowMenuState`/
  `applyActionsMenuState`/`applyWindowMenuState` and deletes
  `HighlightChanged`'s four-boolean hand-diff outright. The push sites
  themselves were then audited and cut from 23 call sites to 16 choke
  points (7 removed, each justified by a covering feature observer). See
  `internal/ui/menus`, `ARCHITECTURE.md`'s row for it, and `todos.md`
  Done → Internal.

### 6. `appState` is anemic — file-set invariants enforced from outside — DONE

- **Impact 1 · Risk 2 · Effort 2 → priority 12** (was 20; both halves landed
  2026-08-28)
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
- **Resolved (2026-08-28):** the eviction half landed too. `appState`
  carries an `onRemove func(fyne.URI)` hook, fired from `removeFile` after
  `publish()` so a subscriber always sees the new generation first;
  `build.go` wires it to the viewer's `imgCache` once the viewer literal
  exists. `RemoveFile` ([viewer.go:754](internal/ui/viewer.go:754)) no
  longer evicts anything itself. drop.go/sort.go's merge/sort ordering
  from outside `appState` is unchanged and not part of this item. See
  `todos.md` Done → Internal.

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
2. **Done (2026-08-28, staged)**: 5 (menu recompute) and the rest of 6
   (cache eviction into `appState`) — one `syncMenus()` replaced the
   per-site Checked/Disabled push, and `appState` now evicts the image
   cache itself on removal via an `onRemove` hook. Both shrank the
   viewer's surface and cleared the way for the field-cluster extraction
   below. See their own resolved notes above and `todos.md` Done →
   Internal.
3. **Done (2026-08-27, staged)**: 2 + 4 — moved the visibility/grouping
   model out of the grid into `internal/dupes`, then boxed the hash engine
   into `grid/hashengine.go`. See their own resolved notes above and
   `todos.md` Done → Internal.
4. **Done (2026-08-28, staged)**: 3 — the `viewer` god object (by the time
   it closed, already filed in `todos.md`'s TODO rather than here — see
   the retirement note above). Four field-cluster extractions,
   `internal/ui/menus`/`autoupdate`/`infoview`/`display`, took `viewer`
   from 87 fields to 55. See `todos.md` Done → Internal.
5. **Watch list**: 1 (bump on release), 7 (revisit at next sigstore-go
   major), 8 (design decision when favorites grow), 10 (keep seams thin).
