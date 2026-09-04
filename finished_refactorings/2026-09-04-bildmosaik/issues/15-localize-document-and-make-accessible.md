# Localize, document, and make the workflow accessible

Type: task
Status: resolved
Priority: P2
Blocked by: 07, 08, 09, 10, 11, 12, 13, 14

## Goal

Audit the finished workflow so English/German users can understand it and all
actions are operable and meaningfully exposed without a mouse.

## Existing code anchors

- English is an identity map in `translations/en.json`; `main_test.go` enforces
  exact locale parity with `de.json`.
- `internal/ui/help/manual.md` and `manual_de.md` are selected by locale and
  guarded against Unicode arrows. `internal/ui/translations_test.go` covers the
  same renderer constraint for catalogue values.
- Fyne 2.8 standard widgets implement `fyne.Accessible`; custom controls must
  provide `AccessibilityLabel`/`AccessibilityRole` themselves when their
  visible text does not already describe them.
- Every earlier ticket adding a user-visible string or `_test.go` file must
  update translations/Qodana in that same change. This ticket is the final
  audit, not a deferred dumping ground.

## Scope

- Audit every new `lang.L` key and provide complete idiomatic English and
  German values, including validation, busy, stale-display, export, wallpaper,
  and typed-limitation states.
- Use standard labelled widgets wherever possible. For custom widgets, assert
  non-empty accessible label/role and expose changing numeric/status values as
  localized visible text; do not promise unsupported generic screen-reader
  live-region behavior.
- Define and test deterministic Tab/Shift+Tab focus order for both configuration
  and preview states. Enter/Space activates the focused action and Escape closes
  or cancels according to the current state without reaching the main viewer.
- Ensure disabled/busy/error/selected states are communicated by text or
  standard widget state, never color alone.
- Update both manuals with menu location, selection-vs-result source rules,
  display choice, settings/Advanced controls, regeneration, export, source
  immutability, and the exact Windows/macOS/Linux wallpaper scope.
- Audit `ARCHITECTURE.md` for entries already required in tickets 01, 02, and
  07; fix drift, but do not defer package documentation until this ticket.
- Run `make sync-qodana-test-exclusions` if any test path is missing, then
  verify the exact-list guard.

## Acceptance Criteria

- Keyboard-only tests reach Generate, Regenerate, Save Image, Set as Wallpaper,
  and Close in deterministic order, including the collapsed Advanced section.
- Every interactive custom object satisfies `fyne.Accessible` with meaningful
  localized output; dynamic status is also visible as text.
- English/German catalogue key sets match, English maps every key to itself,
  and no rendered/manual string contains a Unicode arrow.
- Both manuals describe that targeted Linux wallpaper is refused before a
  global change and that Save Image remains available.
- Architecture and Qodana guards report no drift.

```sh
go test ./internal/ui/mosaicwin -run 'TestMosaicAccessibility|TestMosaicKeyboard' -count=1 &&
go test . -run 'TestTranslations_' -count=1 &&
go test ./internal/ui -run 'TestTranslationsHaveNoUnicodeArrows' -count=1 &&
go test ./internal/ui/help -run 'TestManual' -count=1 &&
make check-qodana-test-exclusions &&
make fmt-check
```

## Non-Goals

- Claiming screen-reader behavior Fyne does not expose to automated tests
- Deferring earlier tickets' translation, architecture, or Qodana obligations

## Comments

Do not put Unicode arrows in either manual, any catalogue key/value, or any
other rendered string.

Implemented and verified on 2026-09-04: English/German parity, manuals, no-arrow
guards, visible status/value text, accessible controls, and keyboard paths are green.
