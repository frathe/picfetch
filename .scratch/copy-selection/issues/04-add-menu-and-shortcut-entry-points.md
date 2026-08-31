# 04: Add menu and shortcut entry points

**Spec:** [Copy an Image-Region Selection](../spec.md)

**What to build:** Add Actions -> Copy selection between the existing image and
path copy actions, bind `Option`/`Alt+Shift+C`, and drive enablement from one
viewer-built menu-state value. Preserve the existing `Cmd`/`Ctrl+C` routing and
`Cmd`/`Ctrl+Shift+C` path copy.

**Blocked by:** 03

**Status:** ready-for-agent

## Entry-point contract

- Add `CopySelection func()` to `menus.Callbacks` and a corresponding
  `ActionItems` entry/accessor.
- Add one positive availability value to `menus.State`, built only in
  `viewer.menuState()`. The menus module applies that value; it does not
  rediscover loading, Grid View, Picture Frame, image, or modal state.
- Add viewer action `copyActionsSelection()` with the same guards as the menu
  state, because the shortcut bypasses the menu.
- Add `wireCopySelectionShortcut` to `wireGlobalShortcuts` using
  `desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier:
  fyne.KeyModifierAlt | fyne.KeyModifierShift}`.
- Use exact visible strings `Copy selection` and `Copy to clipboard` through
  `lang.L`; add both keys to every locale with English identity mapping.

## Behavior checklist

- [ ] Start with failing menu-composition and production-shortcut tests.
- [ ] Preserve exact Actions ordering: Copy image, Copy selection, Copy image
      path.
- [ ] Enable Copy selection only for a decoded image in the normal
      single-image viewer; disable it for empty/initial/loading, Grid View,
      Picture Frame, and modal-prompt states.
- [ ] Keep the item ordinary and unchecked while active; invoking it again is
      a no-op.
- [ ] Prove menu click and real `Alt+Shift+C` shortcut reach the same viewer
      action with one file loaded.
- [ ] Prove the new shortcut does not fire ordinary image copy or path copy,
      and those existing bindings remain unchanged.
- [ ] Keep all user-visible strings localized and locale parity green.

## Files

- Modify: `internal/ui/menus/menus.go`
- Modify: `internal/ui/menus/menus_test.go`
- Modify: `internal/ui/menu.go`
- Modify: `internal/ui/actionmenu.go`
- Modify: `internal/ui/shortcuts.go`
- Modify focused `internal/ui/*_test.go` files
- Modify: `translations/*.json`
- Do not modify: manuals, `internal/clipboard`, or `ARCHITECTURE.md`

## Verification

```sh
go test -race ./internal/ui/menus ./internal/ui -run 'Test(ActionsMenu_CopySelection|CopySelectionAvailability|WireCopySelectionShortcut)$'
go test . -run 'TestTranslations_(EveryLocaleCoversEnglish|EnglishMapsEachKeyToItself|NoArrowFollowedByASpace)$'
go test ./internal/ui -run TestTranslationsHaveNoUnicodeArrows
```

Negatively verify the shortcut guard, restore it, rerun all commands, set
`Status: resolved`, and add the result under `## Answer`. Do not commit.

## Answer

Pending.
