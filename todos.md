# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- Hypno Spiral tunnel image stream is complete. Ronin tested it for an hour
  and accepted stable, relatively smooth operation around 60 FPS, including
  smooth GIF playback. Frozen duplicate-aware sources, Main/Random order,
  image controls, soft entrances and lifecycle handling are implemented.
  Existing native resource runs, automated checks and GoLand inspections
  support the result; the known local amd64 seccomp failures remain separate.
  See the [final acceptance](docs/spiral-qualification-2026-09-12.md#accepted-result).
- Spiral pictures now fade in from fully transparent over 0.75 seconds and
  can launch closer to the centre, with a 3% protected disc instead of 8%.
  Focused and full UI race checks, GoLand inspections and native rendering
  pass; the full gate retains only the known local amd64 seccomp failures.
  See the [soft-entry evidence](finished_refactorings/2026-09-12-spiral-soft-entry.md).
- Spiral tunnel pictures now play bounded GIF previews with independent timing.
  A live image-transparency slider shifts the range, preserves the 85% visibility
  ceiling, and retains its setting across reopening in the current process.
  Image size is adjustable from 0.5x to 2x for new arrivals, with centre clearance
  preserved. Ronin confirmed all sliders work as intended in the native trial.
  See the [follow-up evidence](finished_refactorings/2026-09-12-spiral-help-and-gif-playback.md).

#### Bugfix

- Mosaic wallpaper on Linux with multiple displays now offers an explicit
  **Set on All Displays** action when the desktop rejects a selected-display
  change. Single-display behavior is preserved. Ronin's Ubuntu ARM64 machine
  has two displays; the working x64 machine has one, which explains the reported
  difference. Ronin confirmed the global mosaic wallpaper action on native
  Ubuntu 24.04 hardware with two displays on September 12, 2026.
  See the [fix and verification record](finished_refactorings/2026-09-12-mosaic-global-wallpaper.md).
- Leaving Spiral fullscreen now restores a usable 960x600 window instead of
  collapsing to a 1x1 canvas. Regression, native resize and qualification
  evidence are in the [September 12 record](docs/spiral-qualification-2026-09-12.md).
- H toggles Spiral's local help overlay; F1 retains the main manual binding.
- PR #20 fixes route F1 from the Spiral canvas to the manual, keep blocked
  preview reads from delaying process exit, restore arrivals after shrinking
  an off-screen centre, and prevent random-cycle boundary repeats when a URI
  occurs more than once. Regression evidence is in the
  [review record](finished_refactorings/2026-09-12-spiral-help-and-gif-playback.md#pr-20-review-loop).

#### Internal

- Embedded artwork and font/data inputs reduce the measured native macOS
  binary from 50.63 MB to 40.84 MB (19.3%). Trane and Finis retain only their
  17 used poses with identical original pixels; viewer and manual illustrations
  target 2x display sizes. Make builds omit the emoji font and generate exact
  binary tag vectors from retained JSON. Focused race checks, Linux goldens,
  GoLand inspections, native build/signature checks and Windows internal-package
  cross-checks pass. Ronin accepted the completed plan on September 13, 2026.
  See the [evidence](finished_refactorings/2026-09-13-embedded-asset-size.md).

- Complete local Docker suites now check for a native Linux/amd64 daemon before
  setup, with a clear explanation of the worker seccomp limitation under ARM
  emulation. Worker enforcement tests and native amd64 CI remain unchanged;
  golden rendering and shard inventory remain available under emulation.
  Native Linux amd64 `make verify` passes, including both worker regressions.
  See the [investigation and verification record](finished_refactorings/2026-09-12-linux-worker-test-container.md).
- Added `make movie` to render the current committed Git history with Gource
  and FFmpeg in Docker, including dynamic counters, captions, a growth chart
  and original soundtrack. `MOVIE_SECONDS` sets the duration and `MOVIE_DIR`
  sets the output parent. Seven replay tests and full 180/30-second movie
  checks pass; all 681 UI race tests pass, with only the existing local
  amd64 seccomp failures remaining in `make verify`.
  See `scripts/historymovie/README.md` and the
  [implementation evidence](plans/2026-09-12-history-movie.md).
- Confirmed the UI shard rebalance in hosted CI: the slowest UI job fell from
  14m18s to 10m55s on the rebalance commit (23.7%), and to 10m42s in the
  September 12 mosaic run (25.2%). All assigned tests are accounted for.
  Measured maximum test loads were 11.9% and 9.8% above the mean; the original
  5% balance target was a projection, with runner variation still visible.
  See [evidence and validation](finished_refactorings/2026-09-11-ui-shard-rebalance.md#hosted-ci-confirmation--september-12-2026).

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

## TODO

### Run the embedded-asset final gate on native Linux/amd64

The asset-size plan is complete and accepted. This verification follow-up
remains open: `make verify` correctly stops on the local ARM64 Docker daemon;
run the complete race gate in native amd64 CI before release. The existing unrelated scratch
file `.scratch/mosaic-wallpaper/capture_test.go` also prevents a repository-wide
local format gate; all changed Go files pass formatting. See the
[implementation record](finished_refactorings/2026-09-13-embedded-asset-size.md).

### Complete updater dependency notices before the next release

The September 12 dependency inventory found that `THIRD-PARTY-NOTICES.md`
omits the pinned Sigstore/TUF core module entries. Reconcile actual shipped
source/file licenses across supported targets, include applicable license and
NOTICE text (including go-tuf's NOTICE), and inspect delivery in the final
archives/MSIX. This applies to the dependencies already shipped and is
independent of the declined verifier refactoring.

### WinGet package identifier migration

The local publishing workflow and README now use `frathe.picfetch`. Move the
existing `io.github.frathe.picfetch` manifests in `microsoft/winget-pkgs` to the
new identifier before the next WinGet publication; the new manifest directory
is not present upstream yet. See the
[maintainer's suggestion](https://github.com/microsoft/winget-pkgs/pull/433339#issuecomment-5639706559).

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
  See [MA-025](needs_refactoring.md#ma-025).

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
