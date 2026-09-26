# PicFetch — TODOs

## Done

### What's Changed

#### New Features

Explore your photos by where they were taken. Open **Window -> Location Map**
(`Shift+L`) to see photos with location information on an interactive map.
Nearby photos are grouped together: open a group to browse all its pictures,
then return to the same map position. Use the arrow keys to select a photo or
group, Enter to open it, and Shift+arrow keys to move around the map. Zoom with
`+`/`-` or press `0` to fit everything. The map follows your light or dark theme,
keeps photos visible while dragging, and shows progress while checking duplicates.

![Trane exploring a map of Europe with photo pins](https://raw.githubusercontent.com/frathe/picfetch/9521edb7087c1d229355419e3d3ce0f5299089fc/assets/trane/trane_europe_map.png)

#### Bugfix

- Location Map's loading and offline background now follows the selected theme.
  Find more like this becomes available after leaving a map visit, keeping image
  navigation consistent. Large duplicate groups no longer hold up cancellation.

- Map clipboard shortcuts no longer act on the hidden image. Returning from a
  photo or cluster checks source changes before displaying locations or fetching
  tiles; readable provider images without file versions refresh their GPS too.

- Map previews name the source photo when a hidden duplicate supplies the GPS
  location, while continuing to open the displayed representative.

- Opening Location Map from visual-search results now closes the ranked Grid
  and maps the loaded collection. Tile response metadata is bounded and charged
  to the encoded cache budget; oversized cache headers do not prevent display.

- Location Map keeps only GPS fields in its metadata cache. Retired Favorite
  cache cleanup is bounded and cancellable, avoiding long cleanup during map exit.

- Opening Location Map retains Favorite GPS ownership only for the current
  sources. Initial map tiles can appear even when a neighboring request fails;
  dragging still keeps the previous map visible while replacements load.

- Editing an image while Location Map is open preserves its refreshed Favorite
  GPS cache even when older cleanup finishes after the new scan.

- Copy Selection now keeps keyboard priority in photos opened from Location Map.
  Escape cancels the selection before leaving the photo, G clears an idle
  selection before opening Grid, and both keys wait while a copy is finishing.

- Progressive location scans limit automatic map movement to four updates per
  second, reducing repeated tile downloads while keeping counts and photo cards
  current. Manual movement and the completed scan update immediately.

- Quitting cancels remaining Favorite GPS cache maintenance between filesystem
  operations; simply leaving the map still lets committed cleanup finish.

- Opening a location cluster now starts with every member visible and no inherited
  selection, preventing copy/delete actions from targeting an outside photo.
  Returning restores the original Grid filter and selection.

- Dragging a selection box in Grid View now stays aligned with the pointer,
  even when the selection controls or search progress change.

- Make in-app updates safer on macOS and Linux, and clean up temporary files
  left behind by interrupted updates.

- On Linux, applying a mosaic wallpaper keeps it on the monitor you selected,
  even if another monitor is connected while the wallpaper is being prepared.

- **File -> Close Files** now remains available while your first collection is
  loading or being sorted. Cancelling that work with Escape resets the menu correctly.

- Fixed "Reveal in file manager" on Linux for filenames containing commas.

#### Internal

- Complete embedded Fyne font and Windows GLFW header notices, with pinned
  upstream source/text checks in the existing notice gate.

- PR 58 review hardening: isolated map trials preserve normal updater files,
  trial reports count extensionless images, and map tiles reject redirects.
- Native map exit timing now requires the observed closed viewer, not an unrelated
  changing map frame. Added macOS observer-policy checks and duplicate-shortcut
  regression coverage for map visits.
- PR 58 follow-up: corrected GPS-only metadata assertions and native Shift+arrow
  pan input. Uncorrelated pan/zoom timings now fail qualification explicitly.
- Windows package updates are published through WinGet only after a release
  succeeds, with clearer recovery instructions if publishing fails.
- Improve the safety of developer tools used to investigate failed builds.
- Improve automated checks for Linux desktop integration.

## Open

- **PR 61 FOSSA decisions:** notice fixes are pushed in `f0ed64a`; review the
  remaining findings using [the disposition guide](docs/fossa-license-ci-2026-09-26.md).
  The second export identifies all 13 original issues, including five denied
  CC findings in non-distributed upstream docs/test samples, but still refers
  to pre-fix `cf24b84`. Export the latest PR revision (the `f0ed64a` check reports
  15), confirm File Matches and record project/version-scoped decisions. A fresh
  passing remote check is still required; local checks cannot approve policy
  findings, and the two additional live issues remain unidentified.

- **Application architecture:** the [cross-PR assessment](needs_refactoring.md)
  recommends shared command policy (MA-028), explicit browsing ownership and
  collection transitions (MA-029/030), followed by Favorite ownership, a bounded
  worker-lifetime pilot and launch policy (MA-031 through MA-033). Proposals only;
  keep feature state local and preserve explicit composition. Start with MA-028.

- **Native Location Map gesture timing:** replace hash-only change detection with
  independently verified pan/zoom transforms before enabling formal latency
  qualification again. The current helper rejects these measurements; manual
  trials and stage/RSS observation remain usable. Existing maintainer performance
  acceptance stands separately from measured timing evidence.

## Deferred

### Qodana CI paused

Disabled at Ronin's request on 2026-09-25 after the trial subscription expired.
Keep its configuration for possible restoration; this is not a passed scan.
GoLand inspections and CodeQL remain in use. The
[local inspection research](docs/local-qodana-inspections-2026-09-25.md) records
the IDE-only Qodana option, licensing distinction and historical inspection advice.

### Fyne upgrade deferred

Keep Fyne at v2.8.0 in [PR #19](https://github.com/frathe/picfetch/pull/19).
Ronin reports an upstream library regression with v2.8.1. Revisit the upgrade
after an upstream fix is available and the affected behavior is verified.
The four grouped `golang.org/x/*` updates remain in the PR.


### Retire the GitHub-hosted Intel macOS runner before August 2027

GitHub plans to retire `macos-15-intel`, its final hosted x86_64 macOS runner, in August 2027. Before then, decide
whether PicFetch will stop shipping an Intel macOS archive or retain it through another build path. If Intel support
remains, replace the `macos-15-intel` release job with a tested alternative; otherwise remove the x86_64 artifact and
update the release and installation documentation. The native Apple-silicon build is not affected by Rosetta's
retirement.

## not deemed worth implementing (edge cases)

- **Verifier size refactoring (MA-024):** Declined by Ronin on 2026-09-13.
  Keep upstream Sigstore/TUF verification. Potential savings of a few megabytes
  do not justify the security risk and maintenance burden of a custom or
  trimmed verifier. The refactoring plan and upgrade watch have been removed.

- **Retained decoded EXIF map tiles (MA-025):** Accepted by the user on 2026-09-09
  for occasional single-photo EXIF lookups. The upstream decoded-tile cache is
  unbounded and its long-session impact remains unmeasured. The new Location Map
  changes the scale assumption: bounded tile residency and sustained browsing
  measurements belong to that feature's open work above. The original decision
  does not qualify collection-scale map browsing.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via the space key works, but when trying it with
  mouse and Ctrl key, it does not. Holding the Ctrl key down and clicking on an image does not select it but instead
  opens it. Observation, when pushing the Ctrl key at exactly the same time as clicking on the image, it actually works,
  and the image is selected. (this seems to be a bug in fyne, created an issue, sorry Windows users)
