# PicFetch — TODOs

## Done

### What's Changed

#### New Features

![Trane building a mosaic](https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/trane_mosaic.webp)

- **Create an Image Mosaic from Grid View.** Turn your selected images—or all images matching your current filters—into a collage sized to fill your screen. Everything is generated locally on your device.
- Generate a new arrangement, save it as a PNG or JPEG, or set it as your wallpaper.
- On Windows and macOS, choose which monitor gets the wallpaper. On Linux, you can save the mosaic or use the existing wallpaper option, but targeting a specific monitor isn’t supported.
- Use the mosaic controls with a keyboard.
- Choose **Start Over** from the finished preview to create another mosaic with your settings preserved—handy for setting up another monitor.
- Customize the look, including optional drop shadows, under **Advanced**.
- Added the hidden Finis companion: type `finis` in manual search, then move the
  pointer in his window to guide his gaze. Escape closes the companion.

#### Fixes and Improvements

- Fixed mosaic controls that appeared disabled or didn’t respond properly.
- Fixed overlapping text while generating a mosaic.
- Reduced excessive image overlap so more of each photo stays visible.
- Smoothed jagged borders and rotated image edges while keeping photos sharp.
- Improved image handling: photos follow their saved orientation, all unique images are used before any repeat, and selected duplicates use the highest-resolution version available in Grid View.
- Improved sizing for high-resolution Mac displays and handling of monitor changes before generation.
- Fixed target selection for identical monitors and translated fallback display names.
- Disconnected Windows monitors no longer block mosaic creation or accept wallpaper changes.
- Bounded the memory retained by repeated mosaic sources and preserved the full canvas of partial-frame GIFs.
- Removed artificial fading where rotated photos extend past the canvas boundary.

#### Behind the Scenes

- Improved error reporting and cleaned up internal code and documentation to make the mosaic feature easier to maintain.

## TODO

### Verify the Finis companion on Linux

Run `make verify` when connectivity permits. Finis’s focused tests, native vet
and build, and visual checks passed; Linux race verification was stopped during
dependency setup at the user’s request. See
`finished_refactorings/2026-09-05-finis.md`.

### Validate image mosaics on physical multi-display desktops

Run the supervised native smoke matrix in
`finished_refactorings/2026-09-04-bildmosaik/issues/16-release-hardening-and-native-smoke-tests.md` on
Windows, macOS, GNOME, and KDE hosts with the listed display arrangements.
Automated tests and cross-builds cover the implementation, but they cannot
prove a real desktop changed, retained its Fill/Fit options, or persisted after
PicFetch exited.

### Publish PicFetch in Microsoft Store

Build the Partner Center-reserved PicFetch product as one x64/ARM64 MSIX
bundle, make the Store build defer updates to Microsoft Store, retain the
portable GitHub/WinGet channel, validate on Windows with WACK, prepare the
English/German listing and privacy policy, then submit it for certification.
Implementation plan: `plans/2026-09-03-microsoft-store-msix.md`.

### Automate Microsoft Store updates after the initial publication

Do this only after the first PicFetch submission has passed certification and
the product is published and live. The current `Microsoft Store package`
workflow deliberately stops after building and validating the x64/ARM64 MSIX
bundle and uploading it as a GitHub Actions artifact; downloading the artifact,
uploading it to Partner Center, and submitting it for certification are still
manual steps.

The intended end state is a protected GitHub Actions deployment that takes the
already WACK-validated `picfetch-microsoft-store.msixbundle`, uploads it to the
existing Partner Center product (`9P0DM0KTH01K`), submits the package update for
certification, and reports the resulting submission status. Keep the existing
Store listings and other metadata unchanged unless a release explicitly ships
metadata changes. A tag must never bypass the existing CI, package validation,
or WACK gates.

Potential ticket split:

1. **Provision least-privilege Partner Center automation credentials.**
   - Confirm that the initial product is published, live, free, and eligible
     for Microsoft Store Developer CLI update automation.
   - Associate a Microsoft Entra tenant with Partner Center, register a
     dedicated automation application, add it to Partner Center with the
     minimum role that can manage submissions, and record its tenant ID,
     client ID, seller ID, and product ID.
   - Decide whether Microsoft's current tooling supports short-lived/OIDC
     authentication. If it still requires a client secret, store that secret
     only in a protected GitHub environment, document its expiry and rotation,
     and never write it to repository files or workflow logs.
2. **Add a read-only Store connectivity check.**
   - Install the official `microsoft/microsoft-store-apppublisher` action and
     configure the `msstore` CLI from GitHub secrets.
   - Add a manually triggered diagnostic job that reads PicFetch's product or
     current submission status without creating, changing, or publishing a
     submission. Give authentication failures actionable error messages.
3. **Automate package-only update submission.**
   - Extend `.github/workflows/microsoft-store.yml` after the package/WACK job,
     using the exact bundle produced and validated by that run rather than a
     separately rebuilt or downloaded file.
   - Upload the bundle to the existing product and create/submit an update while
     preserving the existing availability, properties, age rating, listings,
     screenshots, and restricted-capability explanation.
   - Poll Partner Center until it returns a stable accepted, failed, or
     certification-in-progress state; surface the submission ID and Partner
     Center status in the Actions summary.
4. **Protect and test the deployment boundary.**
   - Put the mutating Store step behind a dedicated protected GitHub environment
     with required human approval. Tag creation may build and validate
     automatically, but it must not upload or submit before that approval.
   - Add workflow contract tests for job dependencies, product ID, artifact
     identity, secret names, and the approval environment. Ensure pull requests,
     forks, prerelease tags, reruns of old commits, and ordinary branch pushes
     cannot publish Store updates.
   - Exercise one manual dry run/read-only check first, then submit the first
     automated update under supervision and record rollback/retry instructions.
5. **Document the release and recovery procedure.**
   - Update `docs/microsoft-store.md` with the normal automated path, credential
     rotation, how to inspect certification failures, how to retry the same
     release safely, and how to fall back to the current manual upload process.
   - Keep Store metadata automation out of the first iteration. If automated
     listing or screenshot updates are later needed, design them as a separate
     reviewed workflow using exported Partner Center metadata as the baseline.

Acceptance should require that a release tag still produces a usable artifact
when Store credentials or approval are unavailable, that no Store mutation can
happen before approval, and that an approved run submits exactly one package
version and exposes enough status to distinguish upload, validation,
certification, and publication failures.

### Functional test coverage

Audit baseline, 2026-09-01: package-local statement coverage came from a
passing run of:

```sh
go test -count=1 -skip '^TestE2E' \
  -coverprofile=/tmp/picfetch-cover-clean.out ./...
```

"Effective" coverage below additionally instruments the named package while
running its existing higher-level tests. Percentages locate candidates;
completion means pinning useful PicFetch behavior through an established
interface, not reaching a blanket coverage target.

1. **P0 - `internal/update` (77.2% local / 78.8% effective).**
   - Through `update.Apply`, force a plist-backup failure after binary
     replacement begins (a conflicting `Info.plist.old` is the deterministic
     fixture). Verify the installed binary and plist remain the original
     versions and partial replacement files are removed.
   - Through `Client.Download` with an HTTP test server and fake external
     verifier, reject ZIP symlinks and TAR symlink/hardlink entries. No usable
     stage or file outside the staging directory may be created.
   - Do not exercise the real Sigstore implementation or unreachable platform
     stubs merely to cover their lines.
2. **P1 - `internal/filesort` (73.7% local / 89.9% effective).**
   - Pin `FromPref` / `PrefValue` round-trips for every `Modes` entry and the
     unknown-value fallback to `ByName`.
   - Verify missing files use the documented zero-key fallback for capture
     date, modification time, and size sorts, while `Order` leaves its input
     slice unchanged.
3. **P1 - `internal/ui/copyselection` (85.4% local / 88.1% effective).**
   - Through `Feature` and canvas gestures, verify all eight image-region
     selection handles resize the correct edges, including corner and crossed
     movement.
   - Move a committed image-region selection beyond each image edge and verify
     it clamps to the image while preserving its dimensions.
   - Do not test `resizeRect`, renderer no-ops, or Fyne event methods directly.
4. **P2 - `internal/favthumbs` (86.9%).**
   - Verify `Read` falls through from a corrupt JPEG preview to a valid PNG
     sibling.
   - Verify `Sweep` preserves directories and other non-regular files whose
     names resemble preview files.
5. **P3 - `scripts/releasenotes` (86.8%).**
   - Exercise `Build` and `ClearDone` with CRLF input and with `## Done` as the
     final section, preserving surrounding Markdown and changelog output.

Future tests stay at the exported seams named above. Each must be seen failing
against a deliberate behavior break before it is accepted, then pass its
focused package test and `make verify` at final handoff.

Do not chase the low numbers in `main`, `internal/uitest`, `scripts/synctuf`,
Fyne widget adapters, or the accepted OS-integration seams. `internal/session`
and `internal/favstore` already cover their useful persistence behavior; their
remaining gaps are chiefly framework/filesystem failure plumbing.
`internal/ui/autoupdate` and `internal/appearance` reach 91.4% and 96.4%
respectively when their higher-level tests are counted. Re-audit
`internal/ui/compare` only after the current command-isolation work lands; its
present uncovered input-shield methods are deliberate no-ops.

## LATER

### Retire the GitHub-hosted Intel macOS runner before August 2027

GitHub plans to retire `macos-15-intel`, its final hosted x86_64 macOS
runner, in August 2027. Before then, decide whether PicFetch will stop
shipping an Intel macOS archive or retain it through another build path. If
Intel support remains, replace the `macos-15-intel` release job with a tested
alternative; otherwise remove the x86_64 artifact and update the release and
installation documentation. The native Apple-silicon build is not affected
by Rosetta's retirement.

## not deemed worth implementing (edge cases)

- Windows releases are not Authenticode-signed. Controlled Folder Access and
  SmartScreen both judge by signature and reputation as well as by which
  program is writing, so an unsigned `picfetch.exe` can still be blocked
  even with the in-process swap (see Done → Bugfix above, where the block
  would now name `picfetch.exe` instead of `cmd.exe`). The real remaining
  fix is signing the Windows release build — Azure Trusted Signing or a
  purchased certificate — in `.github/workflows/release.yml`, which runs no
  `signtool` today.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)

### Qodana drops detected duplicates during serialisation (upstream)

At `210fee5` (run `33270269940`), the IDE reports 71 `DuplicatedCode`
fragments and the CI SARIF reports 63, with CI's 63 a strict subset of the
IDE's 71. The 8 fragments CI is missing are 7 in
`internal/imaging/loader_test.go` and 1 at
`internal/update/tufroot_test.go:173`. That run's own `log/idea.log` carries
exactly 3 `#o.j.q.s.i.r.g.DuplicatesProblem` "Can't find duplicate problem in
db" warnings, naming exactly those two files and no others, emitted
immediately after the line `The Project analysis stage completed in 41s` —
so Qodana's own log shows detection succeeded and serialisation into the
report/SARIF failed afterwards. This is an upstream defect, not a picfetch
config problem: nothing here suppresses or excludes those two files, and the
drop happens before any project-side filtering runs.

`qodana.yaml`'s new `_test.go` exclusion (see Done → Internal above) makes
this defect invisible going forward in this repository, because every
dropped fragment happens to live in a test file that the exclusion now
removes from the inspection entirely — recorded here so the defect is not
lost along with the rule that used to surface it. Of the 12-fragment
CSV-to-SARIF gap at `210fee5`, these 8 serialisation losses are one part; the
other 4 are the source-suppressed production fragments in the orientation
pixel loops recorded above, so nothing about that gap is left open — only
the underlying serialisation defect itself is. See
`finished_refactorings/2026-08-29-qodana-evidence.md` for the decoded byte
offsets and anchoring detail, and `plans/2026-08-29-qodana-serialisation-bug-report.md`,
Task 8's draft of the upstream report text — as of this writing not yet
submitted to JetBrains; check that file for whether it has been sent since.
