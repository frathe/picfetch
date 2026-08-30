## What's Changed

### New Features

- Added manual update checks in Settings: **Check now** bypasses the automatic
  daily gate, shows checking/current/download/error states, verifies and
  stages updates, and offers **Later** or **Perform update** to apply and
  relaunch with What's New on the updated launch.

### Internal

- Qodana's false-positive suppressions are confirmed in CI, not just under
  GoLand's engine: at `210fee5` (run `33270269940`), the summary CSV counts
  the 12 findings covered by the 8 `//goland:noinspection` comments exactly —
  `GoMaybeNil` 2, `GoBoolExpressions` 3, `GoRedundantConversion` 1,
  `GoErrorStringFormat` 2, `GoVarAndConstTypeMayBeOmitted` 4, summing to 12 —
  and none of the 12 appear anywhere in `qodana.sarif.json`'s results, whose
  only `ruleId` values are `DuplicatedCode` and `GoTypeAssertionOnErrors`:
  12-of-12 suppressed. The other 18 suppressions, the `GoUnusedParameter` `_`
  renames, never produced a CSV row at all — the blank identifier removes the
  finding before the inspection pass counts it, rather than suppressing a
  counted one. The earlier prediction that the next CI run would show "30
  fewer problems (155 -> 125 on a full scan)" was wrong, and not because the
  suppressions failed: 155 was an IDE Project Default scan and 125 assumed CI
  would run that same profile, but CI runs `qodana.starter` instead, so an
  IDE Project Default total and a CI `qodana.starter` total were never
  comparable in the first place.

- CI does not under-report duplication against a full scan — the old framing
  had it backwards. Qodana emits one SARIF result per duplicate *cluster*,
  not per fragment: at `210fee5` the SARIF holds 33 `DuplicatedCode` results
  occupying 71 location slots, and the slots are not disjoint because 8
  fragments each belong to 2 clusters (55 fragments in exactly 1 cluster, 8
  in exactly 2: 55×1 + 8×2 = 71), so 71 slots reduce to 63 distinct
  fragments. The summary CSV counts 75 `DuplicatedCode` rows, and the gap
  closes completely rather than leaving a mystery: 75 CSV rows = 71
  test-file fragments + 4 production fragments already suppressed at source
  in the orientation pixel loops (see Orientation transforms below), and 71
  − 8 serialisation losses = 63 SARIF fragments, so the 12-fragment
  CSV-to-SARIF gap is 8 + 4 with nothing left unexplained. At `210fee5` the
  IDE finds 71 `DuplicatedCode` fragments and CI finds 63; CI's 63 are a
  strict subset of the IDE's 71, and the 8 fragments CI is missing — 7 in
  `internal/imaging/loader_test.go` and 1 in `internal/update/tufroot_test.go`
  — are accounted for by a genuine Qodana serialisation failure that the CI
  run recorded itself as 3 `DuplicatesProblem` "Can't find duplicate problem
  in db" warnings naming exactly those 2 files and no others. All 33
  clusters at `210fee5` were test-only — every fragment in every cluster
  lives in a `_test.go` file — so `qodana.yaml` now excludes those files from
  `DuplicatedCode` by explicit path; run `33274422606` at `ed3d4e6` confirmed
  it, returning 0 SARIF results.

- Making the `DuplicatedCode` exclusion actually take effect needed three
  attempts, but the mechanism itself was never broken — only the pattern.
  `exclude:` with a `paths:` glob compiles into a real scope:
  `log/effective.profile.xml` shows `<scope
  name="qodana.yaml.exclude.DuplicatedCode" level="INFORMATION"
  enabled="false" />` nested inside the `DuplicatedCode` inspection element
  itself, proving the entry reached the engine — but a compiled, disabled
  scope that matches no file suppresses nothing. Both `"**/*_test.go"` (run
  `33273666731` at `d38f6e8`) and `"**_test.go"` (run `33274030031` at
  `f481c4a`) failed exactly that way: 33 `DuplicatedCode` results both times,
  unchanged from baseline. The second glob is the dialect JetBrains uses for
  its own built-in scopes (`glob:**.md`, `glob:**.test.ts`), so it was not a
  wild guess, and it still did not match. Neither failure was a delivery
  problem — each run's `log/qodana-config.json` echoes back the exact
  pattern that run was given, and neither run's `idea.log` shows a cache hit
  or restore. Explicit file paths work: with `qodana.yaml`'s `paths:`
  replaced by the 30 flagged test files listed by path, run `33274422606` at
  `ed3d4e6` returned 0 SARIF results. That the CSV's `DuplicatedCode` count
  fell to 4 rather than 0 on that run is the proof the exclusion narrowed the
  inspection instead of disabling it outright — a full disable would have
  zeroed the CSV too, not just the SARIF.

- Update-check lifecycle ordering is restored: after its existing gates,
  `maybeStartUpdateCheck` prepares the verifier/client through the
  instance-owned `Updater` verifier-factory seam before `updateOp.begin()`,
  then calls `Updater.Start`. The focused
  `TestUpdateCheck_VerifierFailurePreservesLifecycle` regression test proves
  a verifier failure preserves the revision and prior token/context, leaves
  the client nil, and never begins completion. Moving `begin` above
  `EnsureClient` makes that test fail on the lifecycle preservation; restoring
  the intended order passes.

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

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.11...v0.2.12
