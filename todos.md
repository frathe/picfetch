# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- Added current app version and build number to Settings > Updates.
- Split Settings into General, Appearance, and Updates tabs so related options
  are easier to find.
- Added manual update checks in Settings: **Check now** bypasses the automatic
  daily gate, shows checking/current/download/error states, verifies and
  stages updates, and offers **Later** or **Perform update** to apply and
  relaunch with What's New on the updated launch.
- Added an **Appearance** choice in Settings with Light, Dark, and System
  default modes. The System default follows operating-system theme changes.
- Added a **Keep a fixed window size** toggle in Settings > General. When it
  is on, PicFetch no longer resizes the window to fit each image, zoom step,
  or rotation; a size you set by dragging the edge is kept across launches.
  Escape back to the welcome screen also leaves that size alone.

#### Bugfix

- Windows: if PicFetch lived in a protected folder (Documents, Music, and the 
  like), Controlled Folder Access blocked every in-app update. The update now 
  applies without going through cmd.exe, so that block is gone.
  If Windows still refuses the write (the app isn't signed), you get a message
  on the next launch with a button to the download page instead of a silent fail. 
  That dialog actually fits the window, and German text no longer shows garbage 
  characters.

#### Internal

- Made macOS native-menu title reads return caller-owned C strings, removing
  shared-buffer aliasing and truncation. Native-menu test fixtures now release
  their retained AppKit menus after each test.

## ACTIVE DEVELOPMENT

## TODO

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
