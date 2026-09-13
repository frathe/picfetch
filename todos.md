# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- Optimized the static assets bundled with PicFetch, reducing
  the macOS application binary by **19.3%** (from 50.63 MB to 40.84 MB).

<img src="https://raw.githubusercontent.com/frathe/picfetch/0c3bf8cddb83a53d34072ca9b54edc32d9fc39bb/assets/trane/trane_shrink_ray.png" alt="Trane using a shrink ray to illustrate smaller PicFetch downloads.">

- **Hidden easter egg:** enjoy a moving tunnel of pictures with smooth
  transitions and animated GIF playback. It ran stably at around 60 FPS
  during an hour of hands-on testing. Use your current image order or shuffle
  the pictures, with duplicate filtering respected.
- Pictures now fade into the Spiral over 0.75 seconds and can appear closer
  to the centre, creating a softer, fuller effect.
- Animated GIFs play independently inside the Spiral. Adjust picture
  transparency while it runs, or set newly arriving pictures to between
  half and twice their normal size. Pictures remain partly transparent,
  and your transparency setting is remembered until you quit PicFetch.

#### Bugfix

- Fixed the reference screenshot used to check **Copy Selection**, without
  changing its appearance, and repaired links to earlier development records.
- Fixed several Explorer build commands so they correctly prepare the required
  tag data before building the app.
- Fixed artwork checks that incorrectly rejected optimized images because of
  color changes in completely transparent areas. Visible colors and transparency
  remain unchanged.
- On Linux systems with multiple monitors, Mosaic wallpaper now offers
  **Set on All Displays** when the desktop cannot change just the selected
  monitor. This was confirmed working on Ubuntu 24.04 with two displays.
- Leaving Spiral fullscreen now restores a usable 960 × 600 window instead
  of shrinking it to a tiny, unusable size.
- Press **H** to show or hide Spiral's help overlay. **F1** opens the main manual.
- Fixed several Spiral issues: **F1** now works when the Spiral canvas has
  focus, stalled image loading no longer delays quitting, pictures resume
  appearing after reducing the centre size when it is off-screen, and shuffle
  avoids repeating the same picture at the end of one cycle and the start
  of the next, even when the source list contains repeated entries.

#### Internal

- Updated the third-party license notices included with the updater across
  all six supported desktop builds. All six standalone packages have been
  checked; signed releases and the final Windows MSIX package still need
  verification. The license check also rejects source links for the wrong
  dependency version.

- Reduced the measured macOS application binary from **50.63 MB to 40.84 MB**,
  a **19.3% reduction**, by optimizing bundled artwork, fonts and data.
  Trane and Finis keep all their used poses with their original pixels.

- Full Docker test runs now check that the computer running them supports
  the required Linux security features. Unsupported ARM emulation produces
  a clear explanation. The full checks pass on native Linux x64.

- Added a developer command, `make movie`, that creates an animated video
  of PicFetch's development history, including captions, project statistics,
  a growth chart and original music. Video length and output location
  are configurable.

- Redistributed automated interface tests so they finish sooner. The slowest
  test group completed about **24–25% faster** in measured runs, with every
  test still included.

## TODO

### Comparison test deadline under build contention

Completed on September 12: the regression now uses smaller synthetic frames,
the existing five-second comparison wait budget, cancellable worker waits and
reported cleanup failures. Both frames still require detail tiles, preserving
the cancellation dependency. The original fixture failed all three race runs
with four CPU contenders sharing one core; the final fixture passed all five
Ubuntu contention runs (2.9–3.8s). All three Settle regressions pass five Ubuntu
race repetitions each, and an overlay restoring the obsolete-tile deadlock
still fails the test as expected. Evidence: `/tmp/picfetch-compare-deadline/`.

Final `make verify` passes on this Ubuntu host: formatting, TUF, Qodana exclusions,
vet, build and all four Docker race partitions. The complete streams contain
2,605 top-level passes and five existing skips; all 682 UI assignments ran exactly once.
The changed regression also passes during the full suite (2.32s).
Raw gate evidence: `.scratch/race-runs/20260912T185421Z-jHjXwb/`.
GoLand inspection of `internal/ui/compare/vector_test.go` is complete, including
weak warnings. Its only findings are intentional duplicate setup in the unchanged
queued-completion and cancellation tests; the existing exact-file
`DuplicatedCode` exclusion in `qodana.yaml` covers both. No actionable findings
remain. A fresh native `go test -race -count=5 -timeout 2m -run
'^TestCompareSettle_' ./internal/ui/compare` also passes (10.338s).

### Qualify updater notices in native release CI

Notice reconciliation and delivery guards are implemented. PR #22's
[native Linux/amd64 full CI gate](https://github.com/frathe/picfetch/actions/runs/34755687251)
passes, along with its Windows and macOS native guards. Before release, observe
the new inspection of final signed Windows archives and both native MSIX bundle
payloads. This host has no Windows SDK; local unsigned archives for all six
targets pass. The audit preserves inferred MPL matcher attribution for
`go-pathspec`; its historical `fnmatch.translate()` source does not identify an
exact CPython version, so the retained Python license is explicitly an ancestry
reference. See the [evidence and limits](plans/2026-09-13-updater-notices.md).

### Fyne upgrade deferred

Keep Fyne at v2.8.0 in [PR #19](https://github.com/frathe/picfetch/pull/19).
Ronin reports an upstream library regression with v2.8.1. Revisit the upgrade
after an upstream fix is available and the affected behavior is verified.
The four grouped `golang.org/x/*` updates remain in the PR.

### Antivirus verdicts on unreleased builds

The September 11 local builds have likely false positives: Microsoft flags both
Linux architectures as `Trojan:Script/Wacatac.C!ml`; Trapmine alone flags Windows
AMD64 as `Malicious.high.ml.score`. Microsoft reports both Windows builds as
undetected. Windows ARM64 is 0/67, but Trapmine cannot process that file type.
All 99 distinct cached dependency directories and archives match the build
metadata and `go.sum`. Vendor review remains pending; no samples have been
submitted by the agent. Record final vendor determinations and rescan the final
release artifacts. See [the investigation](docs/antivirus-triage-2026-09-11.md).

## LATER

### WinGet package identifier migration

Move the existing `io.github.frathe.picfetch` manifests in `microsoft/winget-pkgs` to the
new identifier `frathe.picfetch` before the next WinGet publication; See the
[maintainer's suggestion](https://github.com/microsoft/winget-pkgs/pull/433339#issuecomment-5639706559).


### Revisit HEIC at its next dependency upgrade

[MA-023](needs_refactoring.md#ma-023) tracks retiring the HEIC fork when an
approved official release includes its leak fix. Its separate upgrade trigger
remains unchanged.

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

- **Retained decoded map tiles (MA-025):** Accepted by the user on 2026-09-09. The map loads only when opened, and
  checking the geolocation of thousands of images is outside expected use. The upstream decoded-tile cache remains
  unbounded; its long-session impact is unmeasured. No further measurement or implementation work is planned.

- Windows releases are not Authenticode-signed. Controlled Folder Access and SmartScreen both judge by signature and
  reputation as well as by which program is writing, so an unsigned `picfetch.exe` can still be blocked even with the
  in-process swap (see Done → Bugfix above, where the block would now name `picfetch.exe` instead of `cmd.exe`). The
  real remaining fix is signing the Windows release build — Azure Trusted Signing or a purchased certificate — in
  `.github/workflows/release.yml`, which runs no
  `signtool` today.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via the space key works, but when trying it with
  mouse and Ctrl key, it does not. Holding the Ctrl key down and clicking on an image does not select it but instead
  opens it. Observation, when pushing the Ctrl key at exactly the same time as clicking on the image, it actually works,
  and the image is selected. (this seems to be a bug in fyne, created an issue, sorry Windows users)

### Qodana drops detected duplicates during serialisation (upstream)

At `210fee5` (run `33270269940`), the IDE reports 71 `DuplicatedCode`
fragments and the CI SARIF reports 63, with CI's 63 a strict subset of the IDE's 71. The 8 fragments CI is missing are 7
in
`internal/imaging/loader_test.go` and 1 at
`internal/update/tufroot_test.go:173`. That run's own `log/idea.log` carries exactly 3
`#o.j.q.s.i.r.g.DuplicatesProblem` "Can't find duplicate problem in db" warnings, naming exactly those two files and no
others, emitted immediately after the line `The Project analysis stage completed in 41s` — so Qodana's own log shows
detection succeeded and serialisation into the report/SARIF failed afterwards. This is an upstream defect, not a
picfetch config problem: nothing here suppresses or excludes those two files, and the drop happens before any
project-side filtering runs.

`qodana.yaml`'s new `_test.go` exclusion (see Done → Internal above) makes this defect invisible going forward in this
repository, because every dropped fragment happens to live in a test file that the exclusion now removes from the
inspection entirely — recorded here so the defect is not lost along with the rule that used to surface it. Of the
12-fragment CSV-to-SARIF gap at `210fee5`, these 8 serialisation losses are one part; the other 4 are the
source-suppressed production fragments in the orientation pixel loops recorded above, so nothing about that gap is left
open — only the underlying serialisation defect itself is. See
`finished_refactorings/2026-08-29-qodana-evidence.md` for the decoded byte offsets and anchoring detail, and
`plans/2026-08-29-qodana-serialisation-bug-report.md`, Task 8's draft of the upstream report text — as of this writing
not yet submitted to JetBrains; check that file for whether it has been sent since.
