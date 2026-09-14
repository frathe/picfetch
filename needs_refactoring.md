# PicFetch — Open Refactoring Backlog

Updated 2026-09-14 after completing and qualifying MA-027.

This file contains proposed refactorings and accepted dependency watches.
Completed findings have been removed; their history remains in Git and the
[maintainability implementation plan](finished_refactorings/2026-09-06-maintainability-plan.md).
Existing MA identifiers and accepted decisions are preserved: MA-024 verifier
size refactoring remains declined, and MA-025 records acceptance of the decoded
map-cache limitation previously recorded under MA-015 and in [todos.md](todos.md).
MA-026 is complete in `28d65ef`, with full CI qualification recorded in its
[archived plan](finished_refactorings/2026-09-14-explorer-feature.md).
MA-027 is complete in `e6024dc`, with native CI and clean code/security reviews
recorded in its [archived plan](finished_refactorings/2026-09-14-ma-027-presentation.md).
MA-028 is the next architecture task; it does not reopen the completed audit.

Inspection baseline: `main` at `54fd7c3` (v1.1.2). The root `internal/ui` package
contains 55 production Go files, 10,835 non-test lines including comments, and
275 methods on `viewer`. The refactoring goal is to move feature state and
background-work ownership behind small module interfaces while retaining
explicit cross-feature composition in `internal/ui`.

| ID | Priority | Remaining work | Status |
| --- | --- | --- | --- |
| [MA-028](#ma-028) | P2 | Centralize shared command-admission decisions | Proposed; next refactoring |
| [MA-023](#ma-023) | P3 | Retire the HEIC fork when an official release contains its fix | Accepted dependency watch |

<a id="ma-026"></a>

## MA-026 — Make Explorer a complete feature module

**Complete (`28d65ef`), 2026-09-14.** Explorer owns its workflow, dialogs,
cancellation and workers; root retains source preparation and navigation.
[Native CI](https://github.com/frathe/picfetch/actions/runs/34829485376) passed
all four Linux/amd64 race partitions and every platform job. Qodana and CodeQL
are clear. The [archived plan](finished_refactorings/2026-09-14-explorer-feature.md)
holds the implementation and verification evidence. This anchor remains for
existing dependency links; MA-026 is no longer open refactoring work.

<a id="ma-027"></a>

## MA-027 — Give single-image presentation ownership of its lifecycle

**Complete (`e6024dc`), 2026-09-14.** The
[archived Deep SDD plan](finished_refactorings/2026-09-14-ma-027-presentation.md)
records the accepted contract, seven slices, red/green evidence and qualification.

[display.Feature](internal/ui/display/feature.go) now owns the surface, source
identities, captures, rotation/fades and load/GIF/SVG/preload workers. Root
[load.go](internal/ui/load.go) chooses requested/retry/neighbor sources and
composes display's synchronous handoff with zoom, window policy, title and EXIF.
[vector.go](internal/ui/vector.go) only forwards layout density. Root no longer
owns mutable frames, animation pauses, timers or SVG raster workers.

The public display contract covers stale delivery, coherent handoff, same-source
reopening, saved baselines, fresh-delay capture release and all-generation
teardown. Root tests retain actual input, file actions, cache sharing and window
composition. Foreground cache admission and budgets remain; display speculation
uses a separate atomic non-evicting operation. This corrects the design's mistaken
assumption that existing AddIfFits never evicts; other callers keep its semantics.

[Native CI](https://github.com/frathe/picfetch/actions/runs/34851844070) passed
all four Linux/amd64 race partitions, validation and Windows/macOS guards.
Qodana and CodeQL have zero findings; fresh Codex code/security reviews are
clean. The unchanged golden masters pass. All seven tickets are resolved;
this anchor remains for existing dependency links.

See the [accepted specification](.scratch/ma-027/spec.md),
[verification map](.scratch/ma-027/verification.md), and
[surface-ownership ADR](docs/adr/0001-single-image-presentation-ownership.md).

<a id="ma-028"></a>

## MA-028 — Centralize shared command-admission decisions

**Next refactoring: reduce the number of places a new mode must update.**

**Evidence:** [keys.go](internal/ui/keys.go),
[shortcuts.go](internal/ui/shortcuts.go), [menu.go](internal/ui/menu.go),
[actionmenu.go](internal/ui/actionmenu.go), and
[windowmenu.go](internal/ui/windowmenu.go) each encode parts of comparison,
Explorer, Copy Selection, and slideshow policy. The exact call
`v.comparisonActive()` appears 55 times across 20 root production files.
Repeated checks include intentional defenses at direct action entry points;
their count alone does not justify removing them.

**Proposed split:**

- Introduce a small decision module that reads a snapshot of feature-owned
  state and determines whether a command runs, yields Copy Selection, or
  reports a refusal. Keep feature state in its existing owner.
- Reuse demonstrably shared decisions in menus, shortcuts, and direct command
  entries. Keep Fyne input adapters, action execution, and cross-feature
  composition in `internal/ui`; retain intentional differences between routes.
- Build on completed MA-022 command-admission guards. Its tested matrix was
  an accepted outcome; this is an incremental consolidation, not a reopened
  defect or a broad mode-state rewrite.

**Preserve and verify:** exercise shared decisions directly and retain tests
through actual menu callbacks, registered shortcuts, and direct actions.
Preserve Escape priority, comparison's Help allowance and explicit Open
refusal, Explorer/cohort restrictions, and Copy Selection's zoom/copy
exceptions and pending-copy blocking.

**Done when:** a shared mode rule has one decision implementation that all
applicable entry points consult, with behavior parity and deliberate exceptions
covered by the existing integration guards.

<a id="ma-023"></a>

## MA-023 — Retire the HEIC fork after an upstream release includes its fix

**Accepted watch; trigger: the next separately scoped HEIC dependency update.**

[go.mod](go.mod) requests `github.com/gen2brain/heic v0.7.1` and replaces it with
`github.com/frathe/heic v0.0.0-20260820164529-0ac0a39f8206`, which carries the
native memory-leak fix documented in [AGENTS.md](AGENTS.md). The remaining debt
is maintaining this fork. Upstream release status was not rechecked for this
backlog rewrite.

**Work at the trigger:**

- Inspect the proposed official release/tag and verify that it contains the
  actual fix before removing the replacement. Keep the fork if inclusion cannot
  be established.
- Verify native decoding and the WASM/no-cgo fallback, including existing format,
  orientation and metadata regressions.
- Run imaging tests and the [opt-in RSS check](internal/imaging/heic_leak_test.go)
  below. The latter needs Linux RSS
  support; establish that the native decoder is actually exercised before
  using the result as evidence for the native leak fix. A skipped check or a
  fallback-only run does not establish that result.

```sh
go test ./internal/imaging -count=1
PICFETCH_HEIC_LEAK_TEST=1 go test -tags=heicleak ./internal/imaging -run '^TestHEICDecode_DoesNotGrowRSSUnbounded$' -count=1 -v
```

**Done when:** an official version containing the fix replaces the fork, the pin
notes are updated, and decode/RSS evidence supports the migration.
