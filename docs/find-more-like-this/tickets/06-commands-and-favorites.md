# FML-006 — Wire commands, result actions, and Favorites

Status: planned.
Type: task.
Owner: T0 lead.
Depends on: FML-005 and [MA-026](../../../needs_refactoring.md#ma-026).
Acceptance: AC-06 in the [plan](../plan.md).
Budget: zero spawns; at most two lead review rounds; focused checks only.

## Files

- Add root `internal/ui/visualsearch.go` and `visualsearch_test.go` adapters.
- Modify `features.go`, `menu.go`, `menus/menus.go`, `shortcuts.go`, `keys.go`,
  `viewer.go`, and navigation/comparison/mosaic glue only where required.
- Extend the MA-026 Explorer module's setup continuation and frozen-visit seam.
- Add captured-list support/tests in `internal/ui/favorites/{favorites,add,confirm}.go`.
- Update translations with each UI string, exact Qodana exclusions, root UI
  shard assignments, and `ARCHITECTURE.md` in the implementation change.

## Contract and work

Resolve the reference exactly as the plan specifies. Wire Actions > Find more
like this and physical platform-default `Cmd/Ctrl+Shift+L`; it must not trigger
comparison's exact `Ctrl+L` link toggle. Use the same admission decision at menu,
shortcut, and direct entry. Honor focus/dialog ownership and pending-copy blocks.

After MA-026, route shared asset setup through a cancellable ready continuation;
capture source identity before setup and recheck on completion. A ready callback
starts only its requesting feature. Transfer from Explorer by capturing its
frozen map/cohort/origin state, stopping analysis, and joining it off UI before
admitting search. No two native analysis workers overlap for the main window.

Bridge Feature presentations to ranked Grid and keep image stepping inside the
ranked source list. Opening a result then Escape restores that ranked visit;
comparison close preserves it too. File open/drop and committed source changes
invalidate through the lifecycle ticket's hooks from the outset.

Implement `favorites.Feature.AddFiles([]fyne.URI)` with a defensive snapshot
that survives naming/overwrite dialogs. Existing Add Current List keeps its
original collection-wide behavior. The search Save to Favorites action captures
selected visible matches, or all visible matches when selection is empty, in
rank order; disable it for an empty result. Reuse preview sync and error/overwrite
handling. Never swap `appState.files` temporarily or silently overwrite a Favorite.

## Acceptance and verification

Drive actual menu items, registered shortcuts, and direct actions. Cover missing
reference, multiple selection, focused filename search, first-use retry/cancel,
comparison, Copy Selection, busy scan/sort/slideshow, and Explorer cohort return.
Verify actual opened/copied/compared/mosaic/favorite paths against ranked sources,
including files outside the visible viewport and a root order different from rank.
Favorites tests preserve captured scope through cancel/retry/overwrite dialogs.

```sh
go test -race ./internal/ui ./internal/ui/favorites -run '^TestFindMoreLikeThis' -count=1
make check-test-shards
```

Done when a real menu/shortcut-driven search-to-result-to-Favorite flow works
through injected analysis and OS adapters, preserving the originating visit.
