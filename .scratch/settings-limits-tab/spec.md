# Split Limits from General Settings

Status: ready-for-agent

## Problem Statement

The Settings window's General tab contains too many controls, making routine preferences harder to scan. Four resource and workload caps are mixed with everyday behavior settings even though they are adjusted less often:

- Max files per folder scan
- Max image cache (MB)
- Max thumbnail cache (MB)
- Max file size (MB)

## Solution

Add a Limits tab at the end of the Settings window and move those four existing controls into it. Keep General as the default tab. This is an information-architecture change only: every moved control retains its current value, validation, default, persistence, and immediate-apply behavior.

The resulting tab order is:

1. General
2. Appearance
3. Updates
4. Limits

## User Stories

1. As a PicFetch user, I want infrequently changed limits separated from general behavior settings, so that General is easier to scan.
2. As a PicFetch user, I want Settings to continue opening on General, so that the most common preferences remain immediately accessible.
3. As a user tuning large folder scans, I want the maximum files-per-scan control under Limits, so that workload caps are grouped predictably.
4. As a user tuning memory use, I want the image-cache limit under Limits, so that image memory controls are easy to find.
5. As a user tuning grid memory use, I want the thumbnail-cache limit under Limits, so that related memory controls sit together.
6. As a user protecting PicFetch from oversized inputs, I want the maximum file-size control under Limits, so that input caps are grouped with other safeguards.
7. As a user adjusting window behavior, I want maximum window width and height to remain under General, so that this change does not redefine those display preferences as resource limits.
8. As an existing user, I want all moved settings to retain their saved values, so that reorganizing the interface does not alter my configuration.
9. As an existing user, I want edits in the new Limits tab to apply immediately, so that the interaction remains consistent with the rest of Settings.
10. As a user entering an invalid limit, I want the existing validation behavior to remain unchanged, so that the move introduces no new acceptance or rejection rules.
11. As a user of a translated PicFetch interface, I want the Limits tab translated consistently, so that the new organization works in every supported locale.
12. As a user consulting the manual, I want its Settings descriptions to name the new location, so that documentation matches the application.

## Implementation Decisions

- The Settings window gains one tab named "Limits."
- Limits is the last tab, after Updates.
- General remains the initially selected tab.
- Move only the folder-scan cap, image-cache cap, thumbnail-cache cap, and file-size cap into Limits.
- Maximum window width and maximum window height remain in General.
- Picture-frame interval and duplicate match distance remain in General because they configure behavior or matching rather than the selected resource and workload caps.
- All other General controls remain in General.
- Reuse the existing controls and host bindings. The change reorganizes their container ownership without adding preference fields or changing interfaces.
- Preserve live application, current values, saved preference keys, defaults, units, validation ranges, and hint text.
- Keep the current Settings window dimensions and scrolling behavior unless the real four-tab layout demonstrates a usability defect.
- Add "Limits" as a user-visible translation key in every locale.
- Update user documentation that describes the location of the moved controls.

## Testing Decisions

- Test at the existing Settings UI-composition seam, which builds the real tab container and inspects its user-visible structure.
- Extend the composition test to assert the exact tab order: General, Appearance, Updates, Limits.
- Assert that General remains selected when Settings opens.
- Assert that Limits contains the folder-scan, image-cache, thumbnail-cache, and file-size controls.
- Assert that those four controls are absent from General.
- Assert that maximum window width and height remain in General.
- Preserve the existing assertions that Appearance and Updates contain their current controls and exclude unrelated controls.
- Keep the existing control-level tests as proof that moved controls retain their current values, validation, and host callbacks; do not duplicate those behaviors in new tests.
- Use the repository's locale-parity test to verify that the new user-visible key exists in every translation bundle.
- Run the relevant manual guards after updating the English and German documentation.

## Out of Scope

- Changing any limit's value, default, unit, validation, persistence, or runtime behavior.
- Moving maximum window width or maximum window height into Limits.
- Reorganizing the remaining General settings into additional categories.
- Renaming existing controls or rewriting their hint text.
- Changing Settings window geometry or remembering the last selected tab.
- Migrating or rewriting saved preferences.

## Further Notes

This spec deliberately treats "Limits" as a narrow category for the four controls selected during design. It does not use the presence of "Max" in a label as a general rule for category membership.
