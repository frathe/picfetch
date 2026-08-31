# PicFetch — TODOs

## Done

### What's Changed

#### New Features

 - Copy selection: rectangular image-region copy from the normal viewer
   (Actions -> Copy selection, Opt/Alt+Shift+C).

 - Settings: maximum window width, maximum window height, and fixed window
   size now live under Appearance.

#### Bugfix

#### Internal

 - Copy Selection crop/encode uses one `copyselection.Source` pipeline, even
   with the test encoder seam. Raster capture pins the displayed frame instead
   of allocating a duplicate rotation; SVG capture and crop share one oriented
   logical-size definition.

 - Copy Selection yield is enforced at command entry and at the `ShowImage`
   chokepoint. Favorites menu actions use the same host runner, callback
   completeness is guarded by reflection, and a busy-copy refusal now shows
   feedback. `0` yields because it resets rotation; pure zoom keys still keep
   the mode.

 - Settings window is seeded from `settingsState()` (`preferences.State`)
   and sends the snapshot before and after each edit to `ApplySettings`, which
   applies only that patch. Main-window shortcut changes made while Settings
   is open are no longer reverted by the next unrelated form edit.

 - Zoom geometry skips Copy Selection dispatch entirely while the mode is
   inactive and refreshes only the selection chrome while active.

 - `make test` and the race-test phase of `make verify` run in cached
   Linux/amd64 Docker environments, so local golden comparisons match CI.

## ACTIVE DEVELOPMENT

## TODO

- when a copy selection is active, and the user presses cmd-c, the selection
  is copied to the clipboard.

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
