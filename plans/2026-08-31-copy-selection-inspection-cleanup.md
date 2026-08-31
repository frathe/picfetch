# Copy Selection Inspection Cleanup

Deliverable: remove the reported GoLand inspection warnings without changing
Copy Selection or Settings behavior.

Route: Standard. The cleanup spans two packages and seven implementation/test
files, but introduces no new interface, package, dependency, or user-visible
string. All work stays inline because the relevant context is already local.

## Acceptance criteria

1. The reported builtin-name collisions, unnamed-parameter findings, and
   incomplete `handleKind` switch are absent.
   Verify: lint `internal/ui/copyselection/geometry.go` and
   `internal/ui/copyselection/visual.go` with GoLand inspections.
2. Each reported duplicate pair has one shared implementation or test helper.
   Verify: lint `internal/ui/memlimits.go`,
   `internal/ui/copyselection_pixels_test.go`,
   `internal/ui/copyselection_worker_test.go`,
   `internal/ui/copyselection/png_test.go`, and
   `internal/ui/copyselection/source_test.go` with GoLand inspections.
3. Copy Selection geometry, PNG encoding, viewer integration, and Settings
   application behavior remain green.
   Verify: `go test ./internal/ui/copyselection` and
   `go test ./internal/ui -run '^(TestCopySelection.*|TestSettingsState.*|TestApplySettings.*|TestSetMax.*|TestVectorRasterPixelsFor.*|TestDuplicateDistance.*)$'`.

## Non-goals

- Reorder the `settings` struct for the separate memory-layout suggestion.
- Change public seams, selection geometry, or settings semantics.
- Touch the unrelated Compare feature specification already edited in
  `todos.md`.

## Tasks

### Task 1 - Production inspection findings

Owner: T0 inline
Files: `internal/ui/copyselection/geometry.go`,
`internal/ui/copyselection/visual.go`, `internal/ui/memlimits.go`
Contract: existing function and method signatures stay stable except for naming
ignored interface parameters `_`; settings setters run in their current order.
Test: existing geometry and settings suites.
Verify: acceptance criteria 1 and 3.
Budget: 0 spawns, 1 review round, no full suite.

### Task 2 - Duplicate test fragments

Owner: T0 inline
Files: `internal/ui/copyselection_pixels_test.go`,
`internal/ui/copyselection_worker_test.go`,
`internal/ui/copyselection/png_test.go`,
`internal/ui/copyselection/source_test.go`
Contract: share drag and pixel-assertion mechanics without weakening assertions.
Test: existing Copy Selection suites.
Verify: acceptance criteria 2 and 3.
Budget: 0 spawns, 1 review round, no full suite.

### Final gate

Run `make fmt-check`, `go vet ./...`, `go build ./...`, and
`go test -timeout 20m -race ./...` once.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|------|------------------------|---------------|------------|-------|
| T1 | 0 / 0 | 1 | no | Hot context; inline. |
| T2 | 0 / 0 | 1 | no | Hot context; inline. |
| gate | - | - | yes | Final verification only. |
