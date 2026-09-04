# Fix mosaic acceptance-criteria test gates

Type: task
Status: resolved
Priority: P3
Blocked by: 06, 10

## Goal

Make every mosaic ticket's verification command actually run tests, so a closed
ticket cannot have been closed on a command that matched nothing.

## Evidence

- Ticket 10's gate runs `go test -run 'Test(MosaicLargePool|MosaicRapidRegeneration)'`.
  Neither function exists anywhere in the repository, so the command exits 0
  having run no test at all. `go test -run` reports success for an empty match.
- Ticket 02's `Test(Generate_Bounds|Generate_Pool)` likewise matches nothing.
- Equivalent coverage does exist under different names, for example
  `internal/mosaic/generator_test.go:631` and
  `internal/ui/mosaicwin/mosaicwin_test.go:815`; the tests are not missing, the
  ticket text is stale.
- The spec's initial-view description reads "The initial view contains the
  target-display choice, the full-width Advanced toggle, Generate, and Cancel",
  but the shipped window also has a **Refresh Displays** button
  (`internal/ui/mosaicwin/window.go:176`). The button is wanted; the spec is what
  needs updating.

## Decisions

- Audit every `sh` block in `.scratch/bildmosaik/issues/*.md` and replace names
  that match no test with the real ones.
- Add `-count=1` where it is missing, consistent with the other tickets.
- Record **Refresh Displays** in the spec's initial-view description.
- Consider a small check that fails when a ticket's `-run` pattern matches no
  test, so this cannot recur silently.

## Acceptance Criteria

- Every `-run` pattern in the mosaic tickets matches at least one test function.
- The spec's initial-view list matches the shipped configuration view.

```sh
go test ./internal/mosaic ./internal/ui/mosaicwin -count=1
```

## Non-Goals

- Adding new test coverage
- Reopening resolved tickets whose behaviour is correct
- Changing the mosaic window's layout

## Comments

Found by a spec-axis review of the branch on 2026-09-04. This is documentation
accuracy, but it matters: three resolved tickets were verified by commands that
could not have failed.

## Answer

Audited all 67 documented `-run` commands with `go test -list` against the
build-selected package inventory. The one remaining empty expression was the
spec's large-pool/rapid-regeneration gate; it now selects `TestGenerate_LazyPool`,
`TestGenerate_SourcesUnchanged`, and
`TestMosaicSupersede_ReverseCompletionPublishesOnlyNewest`, and was observed
listing all three. Every documented `go test` command now carries `-count=1`.

The spec's initial view now includes Refresh Displays. The known-invalid Fyne
whole-repository Windows cross-build in ticket 28 was also aligned with CI's
`./internal/...` gate. A recursive test that launches dozens of nested `go test`
processes was not added; the build-selected audit is recorded here and the
repaired command itself remains directly executable.
