# Return to mosaic settings from the preview

Type: task
Status: resolved
Priority: P1
Blocked by: 08

## Goal

Let a user return from a finished mosaic preview to its configuration so they
can generate and set wallpapers for several attached displays without closing
and reopening the workflow.

## Decisions

- Add a localized **Start Over** action at the preview's top-left.
- Preserve the command-entry source snapshot, selected display, visual
  settings, and export format.
- Discard the completed result and status when returning to configuration, so
  hidden preview actions cannot operate on stale pixels.
- Focus the target-display selector after the transition.
- Keep the action disabled while generation, export, or wallpaper work is busy.

## Acceptance Criteria

- A finished preview exposes an accessible and keyboard-reachable **Start
  Over** button at the top-left.
- Activating it shows configuration, hides the preview, clears its result and
  status, and does not invoke generation.
- Sources, target, visual settings, and export format survive the transition.
- Selecting a second display and generating produces that display's native
  dimensions.

```sh
go test ./internal/ui/mosaicwin -run 'TestMosaic(StartOver|Keyboard|Accessibility)' -count=1 &&
go test . ./internal/ui/help -run 'Test(Translations|Manual)' -count=1
```

## Non-Goals

- Changing mosaic generation or wallpaper routing
- Persisting display selection between separate mosaic windows
- Keeping the discarded preview reachable after returning to configuration

## Comments

Requested after using the result screen to set wallpapers on multiple monitors.

## Answer

Implemented a localized, accessible **Start Over** action in the preview's
top-left. It returns the existing mosaic window to configuration, clears the
discarded result and status, focuses the target-display selector, and retains
the source snapshot, selected display, visual settings, advanced expansion,
and export format. A user can then select another display and generate its
native-sized wallpaper without reopening the workflow.

The UI behavior was developed test-first and is covered for keyboard focus,
accessibility, top-left placement, state preservation, second-display output,
and busy-state rejection during regeneration, export, and wallpaper work. The
bilingual manuals and translation catalogues were updated. Visual inspection
confirmed that the new row does not obstruct the preview. `make verify` and
the focused Linux/amd64 race acceptance tests pass.
