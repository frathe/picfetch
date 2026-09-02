## What's Changed

### New Features

![Trane comparing images](https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/trane_lightwall.png)

- Comparison pane linking can be toggled with the ready-gated **Unlink** / **Link**
  control at the top left or physical `Ctrl+L` on every platform. Its adjacent
  status identifies the targeted side while unlinked. Pan, wheel/Shift+wheel,
  and `0` / `1` / `+` / `-` then affect only that photo; once linked again, the
  same controls move one overhead camera while both photos retain their current
  arrangement.

### Bugfix

- Linking or unlinking comparison panes no longer moves or resizes either
  photo.
- In Swipe comparison, pointer hover, pan, wheel, and transform-key targeting
  now follow the revealed photo while panes are unlinked instead of always
  selecting the right photo.
- Moving the Swipe divider now refreshes detail tiles for the newly revealed
  area without repainting or re-decoding the comparison surface.
- Large images keep sharp comparison detail on GLES because tile lookup uses
  visible pane pixels instead of source-normalized values that mediump can
  round to zero.
- Physical `Ctrl+L` now respects Fyne dialogs and popup menus instead of
  changing comparison link state behind their canvas overlay.
- Pan and zoom in side-by-side and Swipe comparison now render through bounded
  GPU tiles instead of resampling the full images on every gesture.
- Physical `Ctrl+D` opens comparison alongside the platform-native shortcut.

### Internal

- Qodana's duplication exclusions now cover every `*_test.go` file. Run
  `make sync-qodana-test-exclusions` after adding or removing tests;
  `make verify` checks that the list stays synchronized.
- Updated the indirect gRPC-Go dependency to `v1.83.2`, resolving
  `CVE-2026-84304` / `GHSA-vp52-pcj8-j9qc`. Was never used (better save than sorry)
- Raised the minimum Go version to `1.27.1`, pinned `govulncheck` at `v1.7.0`,
  and updated Rekor to `v1.5.4`, which replaces its unmaintained
  `x/crypto/openpgp` implementation with the maintained Proton fork. The scan
  finds no reachable or imported vulnerabilities; `GO-2026-5932` remains only
  a module-level notice because latest `x/crypto v0.55.0` is still required for
  unaffected cryptography and the advisory has no patched release.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.16...v0.2.17
