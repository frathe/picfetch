# 01: Open and close a fitted side-by-side comparison

**What to build:** Let a Grid View user invoke comparison for exactly two
explicitly selected images and see them in a fitted, fixed 50/50 side-by-side
overlay. The overlay appears immediately, loads both images concurrently, and
can be dismissed without disturbing any grid state.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

## Acceptance criteria

- [ ] **Actions -> Compare selected images** exists with `Cmd+D` on macOS and
  `Ctrl+D` elsewhere. It is enabled only while Grid View has exactly two
  explicitly selected images. An invalid shortcut invocation remains in Grid
  View and reports **Select exactly 2 images to compare**; it never falls back
  to the highlighted image or chooses two items from a larger selection.
  Verify: `go test ./internal/ui/... -run 'CompareEntry' -count=1`
- [ ] The two selected host-file indices remain eligible when a filename or
  duplicate filter hides either thumbnail. Their ascending grid/file order,
  not gesture order, determines left and right.
  Verify: `go test ./internal/ui/... -run 'CompareSelection' -count=1`
- [ ] Invoking the command immediately places an opaque comparison overlay
  above the still-open grid. Once ready, both images are fitted and centered
  in fixed 50/50 panes, with the earlier file on the left. It does not open a
  second window.
  Verify: `go test ./internal/ui/... -run 'Compare(Overlay|SideBySide)' -count=1`
- [ ] Both sources decode concurrently behind one spinner per pane. **Back to
  Grid** remains usable while loading; comparison controls that require two
  ready images are disabled until both finish. The work has cancellation,
  staleness protection, and an observable completion signal for deterministic
  tests.
  Verify: `go test ./internal/ui/... -run 'CompareLoading' -count=1`
- [ ] **Back to Grid** and `Escape` remove only the comparison overlay, cancel
  pending work, and reveal the original selection, active filter, highlight,
  scroll position, and highlighted-file title unchanged.
  Verify: `go test ./internal/ui/... -run 'Compare(Restoration|Cancel|Exit)' -count=1`
- [ ] If either decode fails, comparison closes automatically, reports a
  non-blocking error, preserves both selections and all other grid state, and
  removes neither file from the set. A late completion cannot repaint after
  the overlay has closed.
  Verify: `go test ./internal/ui/... -run 'Compare(Failure|Stale)' -count=1`
- [ ] Strings introduced by this slice are localized in every catalogue and
  the manuals document how to enter and leave the basic comparison.
  Verify: `go test ./... -run 'Translations|Manual' -count=1`

## Comments

