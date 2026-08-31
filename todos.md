# PicFetch — TODOs

## Done

### What's Changed

#### New Features

 - Copy selection: rectangular image-region copy from the normal viewer
   (Actions -> Copy selection, Opt/Alt+Shift+C).

#### Bugfix

#### Internal

## ACTIVE DEVELOPMENT

## TODO

Architecture review, 2026-08-31. Four deepening candidates, strongest first.
The HTML write-up is not in the repo (OS temp). No ADRs to contradict.

### Deepen Copy Selection through the pixel path

- **Strong** (top recommendation). In-process.
- Where: [copyselection.go](internal/ui/copyselection.go) (viewer adapter),
  [copyselection/](internal/ui/copyselection/),
  [animationpause.go](internal/ui/animationpause.go),
  [features.go](internal/ui/features.go), [load.go](internal/ui/load.go),
  [info.go](internal/ui/info.go) `displayedDimensions`.
- Copy Selection mode is deep for drawing an image-region selection and
  shallow for copying it: the Feature hands a rectangle out, and freeze /
  SVG crop / encode / clipboard / GIF pause live in the viewer adapter.
  Package tests cover gestures; the raster/SVG/GIF copy bugs sit in
  `copyselection_{sources,pixels,worker}_test.go`. `zoom.Geometry` is
  already the presentation seam and stays put.
- **Fix**: move freeze, crop, and encode into the Copy Selection module.
  The viewer starts the mode and supplies the existing clipboard adapter.
  Animation pause becomes an internal seam; oriented displayed size stops
  hiding in `info.go`.

### One yield for Copy Selection mode

- **Worth exploring.** In-process. Related: `needs_refactoring.md` item 9
  (mode-interaction guards in `handleKeyEvent`).
- Where: [copyselection.go](internal/ui/copyselection.go)
  `cancelRegionCopyBeforeAction` (~16 files), [keys.go](internal/ui/keys.go)
  (second key table: which keys cancel, zoom, or stay),
  [shortcuts.go](internal/ui/shortcuts.go),
  [actionmenu.go](internal/ui/actionmenu.go),
  [windowmenu.go](internal/ui/windowmenu.go), plus drop / rotate / save /
  export / openfiles / wallpaper / sort / slideshow / info / batch /
  clipboard.
- Copy Selection mode occupancy leaks across every command entry.
  `keys.go` duplicates `HandleKey` policy. Copy Selection is still the
  only occupant, so a generic occupancy seam would be hypothetical.
- **Fix**: concentrate yield policy in one module that command entry
  already has to cross (busy blocks, idle cancels, zoom does not yield,
  close still runs). Do not invent a feature registry; cross-feature
  decisions stay in `internal/ui`.

### Collapse the settings Host

- **Worth exploring.** In-process.
- Where: [settingswin.go](internal/ui/settingswin/settingswin.go) `Host`
  (~32 methods), [memlimits.go](internal/ui/memlimits.go),
  [theme.go](internal/ui/theme.go), [autoupdate.go](internal/ui/autoupdate.go),
  [preferences.go](internal/preferences/preferences.go) (`State` already
  exists), `fakeHost` in `settingswin_test.go`.
- The Host is field-for-field with the form. Deleting it does not remove
  that surface — it reappears 1:1 on the viewer. `fakeHost` already proves
  two adapters, so the seam is real; it is just shallow.
- **Fix**: drive the window from the standing preferences value plus a
  small apply path for live side effects (cache retune, appearance). Keep
  `CheckForUpdatesNow` / `PerformUpdate` as the only update verbs. Same
  exception menus already used (AGENTS.md: value snapshot when a Host
  would be a dozen-plus methods). Apply cannot be a pure snapshot.

### Give ShowImage a home

- **Speculative.** In-process.
- Where: [load.go](internal/ui/load.go) (~675 lines),
  [vector.go](internal/ui/vector.go) (~274 lines),
  [display.go](internal/ui/display/display.go) (already extracted),
  [animationpause.go](internal/ui/animationpause.go).
- `display` took frames and rotation; the load choreography stayed on the
  viewer. Copy Selection's freeze now reaches through `load.go`.
- **Fix**: only if the Copy Selection pixel-path deepening does not absorb
  the pause as an internal seam. Then concentrate ShowImage / finishLoad /
  animate / vector behind one load interface the viewer composes. Reopens
  the 2026-08-28 field-cluster plan, which stopped at `display` on purpose.
  A load Host risks becoming as wide as the pipeline.

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
