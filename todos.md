# PicFetch — TODOs

## Done

### What's Changed

#### New Features

Explore your photos by where they were taken. Open **Window -> Location Map**
(`Shift+L`) to see photos with location information on an interactive map.
Nearby photos are grouped together: open a group to browse all its pictures,
then return to the same map position.

Use the arrow keys to select a photo or group, Enter to open it, and hold Shift
while pressing an arrow key to move around the map. Zoom with `+`/`-`, or press
`0` to show all photo locations. The map matches your light or dark theme,
keeps photos visible while you drag it, and shows progress while checking for
duplicate photos.

![Trane exploring a map of Europe with photo pins](https://raw.githubusercontent.com/frathe/picfetch/9521edb7087c1d229355419e3d3ce0f5299089fc/assets/trane/trane_europe_map.png)

#### Bugfix

- Location Map's background now matches your theme while loading or offline.
  Parts of the map can appear even if another part fails to download, and the
  current map stays visible while you drag to an area that is still loading.

- Returning to the map after viewing a photo or group now checks for changes to
  the images and refreshes their locations. This also works for images from
  sources that cannot report whether a file has changed.

- When a hidden duplicate provides a photo's location, the map preview now names
  that duplicate. Opening the preview still shows the photo pictured on the card.

- Opening Location Map from visual-search results now closes the search grid and
  shows locations from your loaded collection. **Find more like this** becomes
  available again after you leave the photos opened from the map.

- Copy shortcuts used on the map no longer copy the image hidden behind it.

- **Copy Selection** works correctly in photos opened from the map. Escape clears
  the selection before leaving the photo, and G clears it before opening Grid
  View. Neither key interrupts a copy that is still finishing.

- Opening a group on the map now shows every photo in that group and clears any
  previous selection, so an old selection cannot cause you to copy or delete
  photos outside the group. Returning restores your previous Grid View filter
  and selection.

- While photos are being checked for locations, the map adjusts its position at
  most four times per second, reducing repeated downloads. Photo counts and
  previews stay up to date; moving the map yourself still takes effect immediately,
  as does the final adjustment when the scan finishes.

- Large groups of duplicate photos no longer delay cancellation. Cleaning up old
  saved location data no longer causes long delays when leaving the map, and
  quitting stops the remaining cleanup after any file operation already underway.

- Fixed how Location Map manages saved location information for Favorites when
  the loaded photos change. Editing a photo while the map is open now preserves
  its updated location information even if an earlier cleanup is still finishing.

- Reduced Location Map's memory use by storing only the photo details needed for
  locations and limiting how much download information it keeps. Map images can
  still appear when that extra information is too large to store. Map downloads
  also no longer follow redirects to other web addresses.

- Dragging a selection box in Grid View now stays aligned with the pointer,
  even when the selection controls or search progress change.

- In-app updates on macOS and Linux are now safer and clean up temporary files
  left behind by interrupted updates.

- On Linux, applying a mosaic wallpaper keeps it on the monitor you selected,
  even if another monitor is connected while the wallpaper is being prepared.

- **File -> Close Files** now remains available while your first collection is
  loading or being sorted. Cancelling that work with Escape resets the menu correctly.

- Fixed "Reveal in file manager" on Linux for filenames containing commas.

#### Internal

- Added automated tests for Location Map, including repeated photos, unusually
  large photo details, read-only images, saved location data, preview and map
  position updates, overlapping image saves, and empty collections.

- Completed license notices for bundled fonts and Windows components, and added
  checks to keep those notices accurate.

- Reviewed fifteen license-check findings for third-party components and recorded
  the decisions for the relevant PicFetch versions. Required notices and existing
  license rules remain in place. See the
  [license review details](docs/fossa-license-ci-2026-09-26.md).

- Fixed license-check reports that could incorrectly show no findings when the
  scan results were incomplete.

- Completed editor-based code inspections with no errors or warnings in the
  inspected files.

- Restored the additional automated code checks provided by Qodana. The reviewed
  changes passed those checks, code and security reviews, automated tests, and
  license checks. Updated the developer guide to explain which checks are active.
  See the [verification details](plans/2026-09-26-pr61-license-notices.md).

- Fixed license checks for AVIF image support so they also work in a fresh
  developer setup.

- Improved map testing tools so test runs leave normal update files alone and
  correctly count images whose filenames have no extension.

- Made map performance measurements more reliable: exit timing now checks that
  the viewer actually closed, and movement or zoom timings are rejected if the
  tool cannot confirm that the requested action took place.

- Corrected tests for stored location information and Shift+arrow map movement.
  Added checks for the macOS testing tools and duplicate-photo shortcuts while
  browsing from the map.

- Windows package updates are published through WinGet only after a release
  succeeds, with clearer recovery instructions if publishing fails.

- Improved the safety of developer tools used to investigate failed builds.

- Improved automated checks for Linux desktop integration.

## Open

- **Similarity protocol race timeout:** the PR 61 final Docker race run hit
  `TestAnalysisProtocolPreservesLimitErrorsAndConfiguration/complete`'s 20-second
  helper deadline while the helper was at `os.Exit(0)`. Its code is unchanged;
  three focused host race reruns passed. Investigate load-sensitive helper exit
  behavior separately; do not skip the test or count the failed full run as
  passed. Evidence is in the [PR 61 record](plans/2026-09-26-pr61-license-notices.md).

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

<!--
Inactive pause record, retained for reuse after Ronin authorizes another CI
pause. Update the date and restore this section when that happens.

### Qodana CI paused

Disabled at Ronin's request on 2026-09-25 after the trial subscription expired.
Keep its configuration for possible restoration; this is not a passed scan.
GoLand inspections and CodeQL remain in use. The
[local inspection research](docs/local-qodana-inspections-2026-09-25.md) records
the IDE-only Qodana option, licensing distinction and historical inspection advice.
-->

### Fyne upgrade deferred

Keep Fyne at v2.8.0, including [PR #59](https://github.com/frathe/picfetch/pull/59).
Ronin reports an upstream library regression with v2.8.1. Revisit the upgrade
after an upstream fix is available and the affected behavior is verified.
PR #59 retains its grouped Sigstore v1.11.0 update. The earlier hold in
[PR #19](https://github.com/frathe/picfetch/pull/19) retained its four grouped
`golang.org/x/*` updates.


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
