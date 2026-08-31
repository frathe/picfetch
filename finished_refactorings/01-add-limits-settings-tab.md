# 01: Add the Limits settings tab

**What to build:** Add Limits as the final Settings tab and move the folder-scan, image-cache, thumbnail-cache, and file-size caps into it. Preserve General as the default tab and keep every setting's existing behavior while making the Settings window easier to scan.

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] Settings presents its tabs in this order: General, Appearance, Updates, Limits.
- [x] Settings continues to open with General selected.
- [x] Limits contains the maximum files-per-folder-scan, image-cache, thumbnail-cache, and file-size controls, and those controls no longer appear in General.
- [x] Maximum window width, maximum window height, and every other existing General control remain in General; Appearance and Updates retain their existing controls.
- [x] Every moved control retains its current value, default, unit, validation, persistence, hint text, and immediate-apply behavior.
- [x] The Limits tab label is translated in every supported locale.
- [x] User documentation identifies Limits as the location of the four moved controls.
- [x] The Settings UI-composition, control behavior, locale-parity, and manual guard tests pass.

## Answer

Added Limits as the final Settings tab and moved the four selected controls without changing their widgets, bindings, validation, or persistence. Added English and German tab labels, updated both manuals, extended the Settings UI-composition test, and updated the architecture map.

Verification passed: formatting, vet, full build, focused Settings/localization/manual tests, and the full race-enabled test suite.
