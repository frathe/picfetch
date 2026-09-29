## What's Changed

### New Features

- In picture-frame mode on Linux and Windows, the menu bar no longer sits
  on top of your photos. Rest the mouse along the top of the screen for a
  moment and the menu slides in. Move the mouse away and it slides back
  out. While a menu is open, it stays put. On a Mac, the menu bar is
  unchanged.

- Help -> Privacy policy opens the installed build's policy in a scrollable
  offline window, alongside Release Notes. Menu labels use the existing English
  and German translations; the policy retains its published English text.

### Bugfix

- Location Map now redraws immediately when opened with Shift+L and maximizes
  the window when you enter or return to it.

- Menus and keyboard shortcuts now respect what you’re doing. When a dialog is
  open, commands won’t change the photo behind it, and Copy and Select All work
  on the text you’re editing. Actions that could interrupt copying a selected
  part of a photo are temporarily disabled. Choosing an unavailable action also
  leaves your selection intact.

- Keep Export/Delete keyboard focus on the prompt through Tab and checkbox
  clicks, and release it on dismissal. Windows native qualification found
  Escape swallowed by Export's checkbox or Delete's hidden panel; focused
  regressions and native repeats verify the repair. See the
  [Windows evidence](docs/command-admission-windows-qualification-2026-09-27.md).

- Escape/Close Files now leave ordinary native maximization before restoring
  the compact welcome window. Preserve fixed-size and Grid-owned restore
  behavior. A native-boundary regression and repeated Linux desktop checks
  cover the canvas/native size mismatch; fresh `make verify` and changed-file
  GoLand inspections passed.

### Internal

- Improve consistency when switching collections, opening Favorites, saving,
  cancelling background work, and closing PicFetch. The completed architecture
  cleanup also makes future changes easier to maintain.

- Stabilize the similarity protocol race test by dispatching its helper before
  the test runner starts and bounding its lifetime from the parent. Preserve
  configuration/error transport and cancellation/exit coverage. See the
  [diagnosis and verification record](plans/2026-09-26-similarity-protocol-timeout.md).

**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.1.10...v1.1.11
