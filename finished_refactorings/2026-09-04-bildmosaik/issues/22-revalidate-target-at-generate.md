# Re-validate the target display when generation starts

Type: task
Status: resolved
Priority: P1
Blocked by: 07

## Goal

Prevent a mosaic from being generated for a display that has been detached since
the mosaic window was opened.

## Evidence

- The spec states: "If the selected display disappears before generation,
  require a new choice; do not silently render for another display." Ticket 07's
  acceptance criteria repeat it: "Removed targets cannot fall back to another
  display at Generate time."
- `Window.startGeneration` (`internal/ui/mosaicwin/window.go:371-379`) resolves
  the target only against the snapshot captured when the command was invoked.
- The only re-inspection is `Window.RefreshTargets`
  (`internal/ui/mosaicwin/window.go:491`), which runs solely when the user presses
  the **Refresh Displays** button.
- A monitor detached after the window opened therefore still generates at its
  stale dimensions. The failure surfaces later and less clearly, in the wallpaper
  adapter, or not at all if the user only exports.

## Decisions

- Re-inspect displays at the start of `startGeneration` and match the selected
  identifier against the fresh set before any layout work begins.
- If the identifier is gone, abort generation, show the existing localized
  "target display is no longer available" status, refresh the picker with the
  current set, and leave the selection empty rather than falling back to another
  display.
- If the identifier is present but its dimensions changed, use the fresh
  dimensions.
- Keep the manual **Refresh Displays** action; this check is in addition to it.

## Acceptance Criteria

- Removing the selected display between opening the window and pressing Generate
  aborts generation with the unavailable-target status and no result.
- No silent fallback to another display occurs in that case.
- A resolution change on a still-present display is picked up by the generation
  that follows it.
- A normal generation with an unchanged display set is unaffected.

```sh
go test ./internal/ui/mosaicwin -run 'TestMosaicTarget' -count=1
```

## Non-Goals

- Subscribing to OS display hot-plug notifications
- Re-validating during an in-flight generation
- Changing the wallpaper adapter's own target checks

## Comments

Found by a spec-axis review of the branch on 2026-09-04.

## Answer

`startGeneration` now refreshes the display topology through the existing Host
seam before it constructs a request. A still-attached target uses its fresh
dimensions; a missing target is cleared, the picker is refreshed, and generation
does not start or fall back. Manual Refresh Displays shares the same operation.

Both new guards were negatively verified against the stale-snapshot path, and
the entire `internal/ui/mosaicwin` package plus the ticket acceptance command
pass.
