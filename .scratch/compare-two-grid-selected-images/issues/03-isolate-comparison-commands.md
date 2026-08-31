# 03: Isolate comparison-mode commands

**What to build:** Treat comparison as an exclusive main-window mode so only
comparison interactions, help, and normal window closing can affect the app
until the user returns to Grid View.

**Blocked by:** 01: Open and close a fitted side-by-side comparison

**Status:** ready-for-agent

## Acceptance criteria

- [ ] While comparison is active, its own toolbar and transform controls,
  `Escape`, F1 help, and normal window closing remain available. F1 opens help
  without closing or replacing the comparison.
  Verify: `go test ./internal/ui/... -run 'Compare(AllowedCommands|Help)' -count=1`
- [ ] Viewer/grid switching, file navigation, rotation, sorting, copying,
  deletion, export, favorites, wallpaper, merge, information, EXIF, and
  picture-frame commands are disabled in menus and ignored when reached by a
  keyboard shortcut or direct callback.
  Verify: `go test ./internal/ui/... -run 'Compare(CommandIsolation|MenuState)' -count=1`
- [ ] Unrelated typed keys and pointer gestures cannot leak through the opaque
  comparison overlay to the still-open grid or the hidden viewer beneath it.
  Verify: `go test ./internal/ui/... -run 'Compare(InputIsolation|GridLeak)' -count=1`
- [ ] File-dialog requests, drops, and native Open With deliveries are refused
  with **Return to Grid View before opening files**. They neither queue work nor
  replace or close the active comparison.
  Verify: `go test ./internal/ui/... -run 'CompareOpenRefusal' -count=1`
- [ ] Isolation applies at every command entry point rather than relying only
  on disabled menu items, so accelerators and programmatic feature callbacks
  cannot bypass it.
  Verify: `go test ./internal/ui/... -run 'CompareCommandEntryPoints' -count=1`
- [ ] The refusal message is localized in every catalogue and both manuals
  describe the exclusive-mode behavior.
  Verify: `go test ./... -run 'Translations|Manual' -count=1`

## Comments

