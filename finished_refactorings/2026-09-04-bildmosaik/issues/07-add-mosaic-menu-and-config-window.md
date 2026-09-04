# Add the mosaic menu action and configuration window

Type: task
Status: resolved
Priority: P0
Blocked by: 01, 02, 03

## Goal

Own the dedicated secondary-window shell and wire the only entry point from the
current Grid result, leaving generation lifecycle to ticket 06.

## Existing code anchors

- `internal/ui/menus.Menus` owns every stateful Actions item. Its
  `Callbacks`, `State`, `ActionItems`, `New`, `ActionsMenu`, `applyActions`,
  comparison isolation, `pairs`, and accessors all need the new item.
- `viewer.menuState()` is the only menu snapshot builder; Grid visibility,
  selection, and result-change observers already converge through `syncMenus`.
- `registerFeatures` owns explicit feature construction order. Secondary
  windows use `widgets.Singleton`, with geometry translated in `startup.go`,
  stopped in `run.go`, and persisted through `currentPreferences`.

## Scope

- Create `internal/ui/mosaicwin` with `Window`, a defensive `Snapshot` of
  sources plus source-kind/count display data, configuration controls, display
  selection/refresh, and a `widgets.Singleton`. Add only a narrow consumer-side
  Host for effects that truly cross back into `internal/ui`; do not pass
  `*viewer` or create a state registry.
- Add `Generate Image Mosaic...` immediately after `Compare selected images`
  in the first Actions-menu group. Add a `CanMosaic` menu-state fact; enable it
  only when Grid is visible and `ResultIndexes()` is non-empty. Comparison's
  final isolation override must disable it.
- Guard the callback itself, not only the menu item. Route it through
  `yieldingMenuCallbacks` so active Copy Selection/comparison behavior stays
  consistent with every ordinary command.
- Resolve and copy sources through ticket 03, inspect displays through ticket
  02, then call `mosaicwin.Show`. If the window is already open, raise the same
  window and retain its original source snapshot instead of silently
  retargeting it.
- Show display, minimum shorter edge, frame, Generate, and Cancel initially.
  Put size variation, overlap, and rotation in a collapsed Advanced section.
  Show whether the pool is explicit selection or current Grid result.
- Format each display as localized name plus native resolution and aspect
  ratio. Refresh retains an explicit display ID when still attached; if it has
  disappeared, clear selection, explain the stale target, and disable Generate
  until a new target is chosen.
- Construct the feature in `registerFeatures`, store it on `viewer`, and do not
  add it to `build.go`'s main-window overlay stack.
- Document `mosaicwin` and the new composition file in `ARCHITECTURE.md` in
  this change. Add every English key to both translation bundles now, and add
  every new test path to Qodana now.

## Acceptance Criteria

- Menu order/accessors and every `CanMosaic`, Grid, no-result, slideshow, and
  comparison state are deterministic in `internal/ui/menus` tests.
- A no-match Grid search disables the item immediately; widening the result
  enables it without requiring Grid close/reopen.
- Direct invocation with Grid closed or an empty result is a no-op.
- The secondary window is singleton, is absent from the main overlay stack,
  and holds a defensive source/display snapshot.
- Initial and Advanced controls enforce ticket 01's ranges without converting
  invalid in-progress input into settings.
- Removed targets cannot fall back to another display at Generate time.

```sh
go test ./internal/ui/menus -run 'TestApply_.*Mosaic|TestActionsMenu' -count=1 &&
go test ./internal/ui/mosaicwin -run 'TestMosaic(Window|Controls|Target|Snapshot)' -count=1 &&
go test ./internal/ui -run 'TestMosaic(Menu|Sources|Window)' -count=1 &&
go test . -run 'TestTranslations_' -count=1 &&
make check-qodana-test-exclusions
```

## Non-Goals

- Running the generator
- Preview/export/wallpaper behavior
- A new main-window overlay

## Comments

This ticket now precedes ticket 06 so one ticket owns creation of
`internal/ui/mosaicwin`; ticket 06 extends the existing shell.

Implemented and verified on 2026-09-04: Actions-menu ordering/state and the
singleton display/settings/Advanced configuration window are green.
