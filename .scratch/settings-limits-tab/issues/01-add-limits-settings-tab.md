# 01: Add the Limits settings tab

**What to build:** Add Limits as the final Settings tab and move the folder-scan, image-cache, thumbnail-cache, and file-size caps into it. Preserve General as the default tab and keep every setting's existing behavior while making the Settings window easier to scan.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Settings presents its tabs in this order: General, Appearance, Updates, Limits.
- [ ] Settings continues to open with General selected.
- [ ] Limits contains the maximum files-per-folder-scan, image-cache, thumbnail-cache, and file-size controls, and those controls no longer appear in General.
- [ ] Maximum window width, maximum window height, and every other existing General control remain in General; Appearance and Updates retain their existing controls.
- [ ] Every moved control retains its current value, default, unit, validation, persistence, hint text, and immediate-apply behavior.
- [ ] The Limits tab label is translated in every supported locale.
- [ ] User documentation identifies Limits as the location of the four moved controls.
- [ ] The Settings UI-composition, control behavior, locale-parity, and manual guard tests pass.
