# Spec: comparison link button

Status: approved

## Problem

Comparison pane linking is discoverable only through physical `Ctrl+L`, while
the resulting Unlinked status sits beside unrelated layout, Swap, and exit
actions at the top right. The comparison surface needs a visible pointer-driven
control whose state and availability exactly match the shortcut.

## Decisions

- A separate compact translucent card is anchored at the top left. It contains
  the link-action button first and the existing Unlinked status immediately to
  its right.
- The existing Swipe/Side by side, Swap, and Back to Grid card remains at the
  top right and no longer contains the Unlinked status.
- The button names its next action: Unlink while linked, Link while unlinked.
- The button is visible but disabled until both comparison images are ready.
  Physical `Ctrl+L` is also inert until that readiness boundary.
- The button and shortcut call the existing `compare.Feature.ToggleLink`
  action. No parallel state or callback is introduced.
- Open, close/failure, and Swap restore linked state, the Unlink label, and a
  hidden status. Existing Unlinked, Unlinked: Left, and Unlinked: Right status
  behavior is retained.

## Acceptance criteria

1. The ready-gated button is in its own compact translucent top-left card while
   the existing action card remains at the top right.
   Verify: `go test ./internal/ui/compare -run '^TestCompareLinkControl_TopLeftCardAndReadyGate$' -count=1`
2. Button and physical `Ctrl+L` use the same readiness gate and toggle state,
   label, and status together.
   Verify: `go test ./internal/ui -run 'Compare(LinkControl|LinkToggle)' -count=1`
3. Target-aware status, new-session reset, and Swap reset remain correct.
   Verify: `go test ./internal/ui/compare -run 'Compare(LinkControl|LinkToggle|OpenStartsLinked|SwapWhileUnlinked)' -count=1`
4. Link and Unlink are localized, both manuals document the control, and all
   manual/translation guards pass.
   Verify: `go test . -run '^TestTranslations_' -count=1`
   Verify: `go test ./internal/ui/help -run '^TestManual' -count=1`
5. The repository formatting, vet, build, and race gate passes.
   Verify: `make verify`

## Non-goals

- Changing comparison transform, pan, zoom, relink, or target-selection
  semantics.
- Adding a preference, icon-only control, menu action, or exported API.
- Moving or otherwise changing the top-right comparison actions.
- Rewriting completed historical comparison plans.

## Honest limit

Placement is guarded with deterministic geometry and widget-state assertions,
not a new screenshot golden. This avoids coupling the control to Fyne's
software painter while still proving its edge, ordering, and card ownership.
