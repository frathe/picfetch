# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

- `buildMainMenu`'s exact `help.SetOnManualOpened(view.syncMenus)` observer
  registration is covered by
  `TestBuildMainMenu_ManualOpenedObserverSyncsWindowHelp`. The test scopes
  `theme.DefaultTheme()` to this case because the shared Fyne test theme
  cannot render the manual's complete Markdown font combinations; it identifies
  and closes the newly created manual window and restores the original theme
  during cleanup. Removing the observer registration makes the test fail,
  leaving Window → Help enabled after the manual opens; the restored
  registration passes.

- Favorites no longer un-merges the macOS native menu bar. The original note
  here had the diagnosis half wrong, so for the record: `SetHasFiles` was
  never an unguarded site - `syncMenus` folds on the very next line,
  deliberately - `SetDir` is covered by the startup fold at `run.go:45`, and
  there is no rename path at all, since `favstore` exposes only `Save` and
  `Remove`. The two real paths were adding a favorite (`favorites.go:249`)
  and deleting one (`manage.go:362`), both reaching `refreshMenu` with
  nothing to fold the bar afterwards. `favorites.Host` gained
  `RefreshMenus`, which the viewer implements as `refreshMainMenu`;
  `refreshMenu` calls it instead of `f.menu.Refresh`, and `SetHasFiles` now
  only flips `Disabled`. The invariant is structural rather than documented:
  `internal/ui/favorites` never calls `fyne.Menu.Refresh`, so
  `refreshMainMenu` is the only place a main-bar menu is re-published.
  Pinned at both ends - three tests in `favorites` that the feature makes
  the call (and that `SetHasFiles` does not), one in `internal/ui` that the
  viewer's method reaches `refreshMainMenu`. Fyne's test driver records
  nothing when the bar is re-published: `MainMenu.Refresh` re-hands the same
  pointer to `test.window.SetMainMenu`, which just stores it, and both
  `syncNativeMenuBar` steps dead-end on NSApp's nil menus in a test binary.
  The only trace is `refreshMainMenu` reading the window's menu, so the
  viewer test observes that through a decorator on the per-viewer `v.win`.
  All four verified by mutation.

- Small mechanical Qodana fixes, five categories. `imaging/save.go` and
  `exifwin/tiles_test.go` compare errors with `errors.Is` now;
  `update/extract.go` and `plistdoctypes/doctypes_test.go` deliberately keep
  `err == io.EOF`, because the inspection does not flag the documented
  `tar.Reader.Next` / `xml.Decoder.Token` loop terminators and the bare
  sentinel is what those APIs promise. `raw_test.go` drops a redundant
  `[]byte` conversion (`copy` takes a string source). `apply_test.go`'s
  local `real` became `target`, filename string included, so it no longer
  shadows the builtin. The two scripts mark their ignored `Fprintf` results
  `_, _ =`, matching `scripts/plistdoctypes/main.go`. The spiral FPS
  backdrop colours were the only non-trivial one: they were positional
  `color.NRGBA` literals written out in both `overlays.go` and
  `overlays_test.go`, so they moved into package-level
  `fpsGoodColor`/`fpsWarnColor`/`fpsBadColor` that both sides name. That
  alone would have gutted the test - it would compare a var against itself
  and only catch a wrong threshold - so `TestFPSBackdropColorValues` was
  added to pin the values, and verified by mutation: bumping
  `fpsGoodColor`'s G to 121 fails it while the threshold test still passes.

- Doc-comment openings: `GoCommentStart` wants the documented element's name
  as the comment's first token, followed by whitespace - an optional leading
  `A`/`An`/`The` is the only thing allowed in front. Punctuation glued to
  the name breaks it - a slash, a comma, or a colon all fail even when the
  comment already opens with the right word, which is why `Dir/SetDir` and
  `MouseIn, MouseMoved, and MouseOut` were both flagged. The slash-joined
  pairs became the `X and Y` form `zoom.In`/`Out`,
  `widgets.FocusGained`/`FocusLost` and `imaging.MinVectorWidth`/`Height`
  already used; the three `widgets/style.go` group comments now lead with
  the first constant in their `const`/`var` group; and `internal/ui`'s five
  file-level notes gained a blank line before `package ui`, which detaches
  them from the package clause - the form 69 other files in the tree already
  use, with `run.go` keeping the one real package doc.
  `spiral.ShowForGesture` and `Show` shared one comment block that opened by
  describing `Show`, so it was split: `go doc Spiral.Show` printed nothing
  before and prints its paragraph now. Also normalized the six unflagged
  sites with the same shape in `load.go` and `preferences.go`; the
  inspection skips struct fields and mid-comment references, so those were
  consistency rather than findings. Verified through GoLand's inspection
  engine over all 13 files, which is the same engine Qodana runs and needs
  no `QODANA_TOKEN`.

- Test image fixture duplication: `wrapAPP1` now owns Exif APP1 framing,
  `littleEndianTIFF` shares TIFF header and integer emission across the
  capture-date, RAW-preview, and GPS fixtures, and imaging's oversized-PNG
  test reuses `uitest.TruncatedPNGHeader`; byte-level tests lock both helper
  formats.

- Orientation transforms: the five direct pixel loops stay separate to avoid
  callback dispatch or transform branching in the per-pixel hot path. The
  four reported `DuplicatedCode` copies carry source-local suppressions and
  explanations; characterization tests cover offset source bounds.

- JPEG header-segment walk: `walkJPEGSegments` in `internal/imaging/jpegseg.go`
  is the one SOI-to-SOS marker loop; `jpegEXIFOrientation`, `jpegMetadata`,
  `jpegMetadataSegments`, and `jpegHasRemovableMetadata` keep only their
  per-segment bodies. `stripJPEGSegments` and `jpegLength` stay separate —
  they copy/error and walk entropy-coded scans, respectively.

- Settings numeric entries: `newPositiveIntEntry` in
  `internal/ui/settingswin/settingswin.go` is the one positive-int Entry
  constructor; max-scan, max-width, max-height, image-cache, thumb-cache,
  and max-file-size keep only their Host getter/setter (and the float32
  wrap on window size). The picture-frame interval stays a ParseInt +
  Duration path.

## ACTIVE DEVELOPMENT

## TODO

### `maybeStartUpdateCheck` begins its lifecycle token before the verifier is built

`9fb2f79` moved the update client construction inside
`autoupdate.Updater.Start`, so on `NewSigstoreVerifier()` failure the
token has already been superseded, where previously it had not.
Currently unobservable — the build path only runs when the client is nil,
which is the first call, when no check goroutine exists — but it is a
real ordering difference from the pre-extraction code.

### Qodana: false positives are flagged in code (done, needs CI confirmation)

All 30 non-issues are now suppressed at the source rather than in
`qodana.yaml`, so the reason travels with the code instead of sitting in a
config file nobody reads. Two mechanisms:

- **`GoUnusedParameter`, 18 hits — named `_` rather than suppressed.** Every
  one is either a build-tag stub twin whose signature is fixed by the real
  implementation (`clipboard`, `filepicker`, `trash`, `wallpaper`'s
  `notwindows.go`/`other.go` pairs, `ui/windowmenu_notdarwin.go`,
  `update/apply_unix.go`) or a Fyne interface method
  (`widgets/tappable.go` x3, `widgets/choicepanel.go`,
  `ui/favorites/manage.go`, `ui/help/mascot.go`). The blank identifier is the
  idiomatic Go way to say "deliberately ignored" and the inspection honours
  it, so this needs no suppression comment at all.
- **8 `//goland:noinspection` comments**, each with a sentence saying why the
  finding is wrong: `imaging/gif.go` x2 (`GoMaybeNil`),
  `update/attest_test.go` (`GoBoolExpressions`),
  `plistdoctypes/doctypes.go` and `wallpaper/wallpaper_test.go`
  (`GoErrorStringFormat`), `preferences/preferences.go`
  (`GoRedundantConversion`), `imaging/dhash_test.go` x2
  (`GoVarAndConstTypeMayBeOmitted`).

Note the doc-comment trap: a `//goland:noinspection` line is a directive and
`go doc` drops it, but ordinary prose placed next to it is absorbed into the
doc comment. Explaining a suppression above an exported declaration leaks the
explanation into the public API docs - `preferences.Load` did exactly that
until the note was moved inside the function body. Keep the reasoning in the
body, or on unexported/test declarations.

Not yet confirmed end to end: the release Qodana linter refuses to run
without `QODANA_TOKEN`, so this was verified with GoLand's inspection engine
(same engine, looser profile) plus placement experiments proving statement-,
function- and declaration-level suppression all take. The next CI run should
show 30 fewer problems (155 -> 125 on a full scan); if any survive, the likely culprit is the
declaration-level directive on `dhash_test.go`'s `const` block.

### Qodana: CI under-reports duplication against a full scan

The `qodana_code_quality.yml` run on `4cc8bb5` reported 107 problems
(High 26 / Moderate 81); a full IDE scan at `e9cfe7b` reports 155
(High 26 / Moderate 129). The High counts match exactly and every non-
duplication inspection matches one-for-one — the entire 48-problem gap is
`DuplicatedCode`, 42 in CI against 90 locally. Worth understanding before
trusting the CI number as a gate: `pr-mode: true` narrows what gets
analysed, and with `upload-result: false` there is no SARIF artifact to
check the run against without Qodana Cloud access. Flipping `upload-result`
to `true` would at least make the CI report retrievable.

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
