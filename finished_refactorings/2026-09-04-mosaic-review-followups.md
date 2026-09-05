# Mosaic review follow-ups (tickets 21-28)

Route: Deep

Deliverable: close the eight mosaic review findings in numeric order with a
red/green cycle for every behavioral change, accurate documentation, and the
repository's full verification gate.

## Frame

The attached ticket files are the implementation specification. Their stated
decisions are final. Work proceeds strictly in ticket order: a later ticket may
be inspected, but it is not implemented until every earlier ticket is green and
reviewed.

Non-goals are the union of each ticket's Non-Goals section. In particular, this
work does not redesign mosaic generation, add hot-plug subscriptions, change
Windows or Linux display behavior, add settings, or split the mosaic window.

Honest limit: macOS native-pixel enumeration and Windows COM behavior can be
compiled and exercised at their available seams here, but physical multi-panel
and native Windows smoke tests remain supervised follow-up work in `todos.md`.

## Approved seams and acceptance commands

The user approved the seams by requesting implementation of the attached
tickets. Tests stay at these existing package boundaries:

| Ticket | Seam | Acceptance command |
| --- | --- | --- |
| 21 | `internal/displays` inspection and mosaic target sizing | `go test ./internal/displays -run 'TestInspect' -count=1 && go test ./internal/ui/mosaicwin -run 'TestMosaicTarget' -count=1` |
| 22 | `mosaicwin.Window.Generate` through its `Host` display seam | `go test ./internal/ui/mosaicwin -run 'TestMosaicTarget' -count=1` |
| 23 | `mosaic.Layout` observable placements | `go test ./internal/mosaic -run 'TestLayout' -count=1` |
| 24 | mosaic window UI error boundary | `go test ./internal/ui/mosaicwin -count=1` |
| 25 | display and wallpaper platform adapters | `GOOS=windows GOARCH=amd64 go build ./internal/... && go test ./internal/displays ./internal/wallpaper -count=1` |
| 26 | translation/manual guards | `go test . ./internal/ui/help -run 'Test(Translations|Manual)' -count=1` |
| 27 | ticket command audit and mosaic suites | `go test ./internal/mosaic ./internal/ui/mosaicwin -count=1` |
| 28 | mosaic and wallpaper public behavior | `go test ./internal/mosaic ./internal/ui/mosaicwin -count=1 && GOOS=windows GOARCH=amd64 go build ./internal/...` |

## Task graph

`21 -> 22 -> 23 -> 24 -> 25 -> 26 -> 27 -> 28 -> final gate`

### Task 21 - macOS native panel pixels

Owner: T0 inline

Files: modify the Darwin display adapter and existing display tests; modify
Qodana exclusions only if a new test file is unavoidable.

Contract: `displays.Inspect` keeps opaque IDs and point-space default-screen
selection while returned width and height come from the active display mode.

Test: first pin native mode dimensions and scaled/unscaled default selection at
the existing display seam; then make the minimum adapter change.

Verify: ticket 21 acceptance command.

Budget: at most 1 read-only Scout, 1 review round, no full suite.

### Task 22 - revalidate target at Generate

Owner: T0 inline

Files: `internal/ui/mosaicwin/window.go` and its existing test file.

Contract: generation re-inspects through `Host.InspectMosaicDisplays`, uses a
fresh matching target, and clears a missing target without fallback.

Test: one vertical slice for removal, then one for changed dimensions, then the
unchanged case.

Verify: ticket 22 acceptance command.

Budget: 0 spawns, 1 review round, no full suite.

### Task 23 - minimum card-edge floor

Owner: T0 inline

Files: `internal/mosaic/layout.go` and its existing tests/goldens if required.

Contract: every unrotated shorter edge is at least `MinimumShortEdge` of the
target's shorter edge for every validated variation.

Test: replace the midpoint expectation with an independently calculated floor;
cover nonzero variation and the division guard.

Verify: ticket 23 acceptance command.

Budget: 0 spawns, 1 review round, no full suite.

### Task 24 - log mosaic UI failures

Owner: T0 inline

Files: mosaic window implementation and existing test file unless Fyne's logger
has no safely observable seam.

Contract: generation, display refresh, and wallpaper failures call
`fyne.LogError`; cancellation remains non-error behavior.

Test: pin observable logging where the framework permits, one failure path at a
time; retain existing status assertions.

Verify: ticket 24 acceptance command.

Budget: at most 1 bounded Scout if the Fyne logging seam is not discoverable by
shell, 1 review round, no full suite.

### Task 25 - shared Windows COM declarations

Owner: T0 inline (architecture and untestable platform behavior cannot leave
the Lead).

Files: add one small internal Windows COM package, update the display and
wallpaper adapters, their tests as needed, and `ARCHITECTURE.md`.

Contract: one owner exports the exact GUID/vtable/constants/HRESULT vocabulary
needed by both consumers without coupling the consumers to each other.

Test: preserve adapter tests and cross-build before consolidation; then remove
duplicates and explicitly bind discarded syscall results.

Verify: ticket 25 acceptance command plus a repository declaration search.

Budget: 0 spawns, 1 review round, no full suite.

### Task 26 - refresh agent documentation

Owner: T0 inline

Files: `AGENTS.md`, `CONTEXT.md`, and one standing architecture/process document
for `plans/`.

Contract: documentation matches the three UI queues, five dispatcher-backed OS
integrations, mosaic vocabulary, and plan lifecycle.

Test: run existing documentation, translation, and manual guards before and
after the edits.

Verify: ticket 26 acceptance command plus targeted text searches.

Budget: 0 spawns, 1 review round, no full suite.

### Task 27 - repair acceptance gates

Owner: T0 inline; exact command edits are scripted if a mechanical batch is
useful (Rule S).

Files: `.scratch/bildmosaik/spec.md` and ticket files under
`.scratch/bildmosaik/issues/`.

Contract: every documented `-run` expression selects at least one real test and
the initial-view description includes Refresh Displays.

Test: derive an executable audit from `go test -list` before changing command
text; add a repository guard only if it is small and stable.

Verify: ticket 27 acceptance command plus the audit.

Budget: 0 spawns, 1 review round, no full suite.

### Task 28 - remove misleading dead code

Owner: T0 inline

Files: mosaic generator, mosaic window, Windows wallpaper adapter, and existing
tests.

Contract: remove only the four named redundancies while preserving behavior.

Test: run focused behavior tests before and after each small cleanup; keep a
test seam only where it proves behavior rather than implementation structure.

Verify: ticket 28 acceptance command.

Budget: 0 spawns, 1 review round, no full suite.

## Delegation gate

Ticket 21 Scout: G1 yes (bounded recon prompt); G2 yes (file:line report and
focused commands); G3 yes (read-only); G4 yes (Darwin test/build-tag sweep is
smaller than loading that detail into the Lead before planning); G5 yes (spawned
before the Lead built detailed Darwin context). Rule S does not replace a
cross-file semantic inspection. Rule W is satisfied because the Scout writes no
implementation.

All implementation, architecture, documentation judgment, review, fixes, and
the final gate remain with T0.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| 21 | 1 / 1 | 1 | no | Read-only Darwin Scout; guard negatively verified |
| 22 | 0 / 0 | 1 | no | Both target guards negatively verified |
| 23 | 0 / 0 | 1 | no | Floor red/green; division guard negatively verified |
| 24 | 1 / 0 | 1 | no | Shell found standard-log seam; all three guards red/green |
| 25 | 0 / 0 | 1 | no | New deep wincom module; structural guard negatively verified |
| 26 | 0 / 0 | 1 | no | Agent pointer and domain glossary refreshed |
| 27 | 0 / 0 | 1 | no | 67 build-selected patterns audited; one empty gate repaired |
| 28 | 0 / 0 | 1 | no | Behavior preserved; Windows validation seam documented |
| final gate | - | - | yes | `make verify`, Windows amd64 vet/build, and Windows arm64 build passed |
