# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- **Export options: an export size limit and metadata omission.** The export
  prompt (`Cmd/Ctrl+E`) now carries two rows above its PNG/JPEG buttons,
  reachable with Up/Down and reset to today's behaviour on every open: an
  **export size limit** (Original / 2400 / 1600 / 1000 on the longest edge,
  aspect preserved, never enlarging, with the Original rung labelled with the
  frame's own longest edge) and **Include camera metadata (JPEG only)**,
  which when unchecked writes the copy without the source's identifying tags
  while the source keeps all of them. A limit that actually changed the pixels
  joins the suggested filename and the completion toast; an export at the
  defaults is byte-identical to what it always wrote, with the short toast it
  always showed. `imaging.Export` grew an `ExportOptions` value (the wallpaper
  and mosaic paths pass defaults), `imaging.ScaleForExport` scales with
  CatmullRom while the thumbnail cache keeps ApproxBiLinear, and the in-place
  TIFF patcher learned to *remove* entries so a resized JPEG exported with its
  metadata no longer carries dimension tags claiming the original's size -
  MakerNote and DPI deliberately kept. See
  `plans/2026-09-06-export-options.md` and `.scratch/export-options/`.

  The TODO this closes claimed the metadata half was something
  `internal/imaging/save.go` "already gets close to". It was in fact complete
  and shipped as the EXIF window's in-place **metadata removal**; the work here
  was making an existing export default optional (**metadata omission**), not
  building a stripper.

- **Startup flags for scripting and picture-frame use.** `picfetch` now takes
  `--slideshow`, `--shuffle`, `--interval=8s`, `--sort=name|date|modified|size|drop`,
  `--merge`, `--max-files=N` and `--help`, parsed in the new `internal/launch`
  and applied by `viewer.applyLaunchOptions`. Flags may sit anywhere among the
  paths, an unknown one exits 2, and macOS's `-psn_*` is ignored so a Finder
  launch cannot be broken by it. Every flag is session-only: the pre-flag value
  is what gets saved at shutdown, so a scripted launch never rewrites the
  user's standing settings. See `plans/2026-09-05-launch-flags.md`.

#### Bugfix

#### Internal

## TODO

### Dimension tags survive a rotated or EXIF-oriented export

Found while reviewing the export-options work, pre-existing since export
started preserving metadata. `imaging.Export` drops the dimension tags a
resize invalidated (`SizeLimitApplies` decides), but a *rotation* invalidates
exactly the same tags and nothing fires:

- Exporting a 900x600 JPEG at Original size after a 90-degree turn in the
  viewer writes 600x900 pixels while IFD0 `0x0100` still reads 900. Verified
  directly against a written file.
- The same holds for any source whose EXIF Orientation is 5-8: export writes
  the upright frame and normalizes Orientation to 1, while `0x0100`/`0x0101`
  and `0xA002`/`0xA003` still describe the stored, unrotated frame.

Ticket 04 scoped its trigger to the size limit ("choosing 2400 for an 1800px
photo drops nothing"), so this is out of that ticket rather than a defect in
it - but the reason the tags are dropped at all ("no program reads the file as
claiming a size it does not have") covers this case just as well.

The fix is one predicate, not new machinery: compare the written frame's
dimensions against the source JPEG's own SOF header instead of against the
scaler's input, which catches the resize, the viewer rotation and the
orientation normalization in one. It changes what exports write for every
EXIF-rotated photo, so it wants a decision rather than a quiet patch.

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
