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

---
