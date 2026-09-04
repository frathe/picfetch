# Refresh agent documentation for the mosaic feature

Type: task
Status: resolved
Priority: P2
Blocked by: 07

## Goal

Bring `AGENTS.md` and `CONTEXT.md` back in line with what the branch actually
shipped, so the conventions file does not describe an architecture that no longer
exists.

## Evidence

- `AGENTS.md`, Concurrency and Fyne: "`internal/ui/grid` and `internal/ui/compare`
  are the per-instance `UIQueue` exceptions ... `internal/ui`'s `newTestUI`
  installs a drainable `uitest.UIQueue` on both features". There is now a third:
  `internal/ui/mosaicwin/lifecycle.go:10` holds a `UIQueue` and
  `internal/ui/harness_test.go:93,248` wires it. The statement is false as
  written. `ARCHITECTURE.md` was updated in the same change; `AGENTS.md` was not.
- `AGENTS.md`, Architecture and Data Flow: "OS integrations live behind dispatcher
  vars in `internal/{clipboard,filepicker,trash,wallpaper}`". `internal/displays`
  is now a fifth such package (`internal/displays/displays.go:59`, `var Inspect`,
  stubbed through `uitest.StubDisplays`) and is not listed.
- `CONTEXT.md` gained no entries although the change introduced "target display",
  "source pool", and "Grid result" into code, both manuals, and both translation
  catalogues. `CONTEXT.md` already fixes "Grid selection" against "Selection"; the
  new terms have the same ambiguity risk.
- A top-level `plans/` directory (5 files) appeared with no mention in `AGENTS.md`
  or `ARCHITECTURE.md`, so its relationship to `.scratch/` and
  `finished_refactorings/` is undocumented.

## Decisions

- Name `mosaicwin` as the third `UIQueue` exception, with its own drain rule, or
  restate the rule so it does not enumerate features.
- Add `internal/displays` to the dispatcher-var list.
- Add "target display" and "source pool" to `CONTEXT.md` with their `Avoid:`
  alternatives, following the existing entry shape.
- Document what `plans/` holds and how it differs from `.scratch/` and
  `finished_refactorings/`, or fold it into an existing location.

## Acceptance Criteria

- No statement in `AGENTS.md` about `UIQueue` exceptions or dispatcher vars
  contradicts the code.
- `CONTEXT.md` defines the mosaic vocabulary used in the manuals and catalogues.
- The purpose of `plans/` is documented in one place.

```sh
go test . ./internal/ui/help -run 'Test(Translations|Manual)' -count=1
```

## Non-Goals

- Restructuring `AGENTS.md`
- Changing the `UIQueue` design itself
- Renaming shipped user-visible strings

## Comments

Found by a standards-axis review of the branch on 2026-09-04.

## Answer

`AGENTS.md` now names all three per-instance UI queues and their settle rules,
includes `internal/displays` in the dispatcher-backed OS integrations, and has
one concise lifecycle pointer distinguishing active `plans/`, tracker material
in `.scratch/`, and accepted plans in `finished_refactorings/`.

`CONTEXT.md` now defines Grid result, target display, and source pool with
explicit alternatives to avoid. The definitions remain domain-only and contain
no implementation details. The ticket acceptance tests and targeted accuracy
searches pass.
