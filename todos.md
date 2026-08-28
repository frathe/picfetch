# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

## ACTIVE DEVELOPMENT

## TODO

### Qodana: the JPEG segment walk is copy-pasted four times

`internal/imaging/exif.go:39` (`jpegEXIFOrientation`), `exif.go:238`
(`jpegMetadata`), `internal/imaging/jpegexif.go:35`, and `jpegexif.go:117`
each carry their own copy of the same marker-skipping state machine: the
`data[pos] != 0xFF` bail, the no-payload marker set
(`0xD8 || 0x01 || 0xD0..0xD9`), the `0xDA` start-of-scan stop, and the
`segLen < 2 || pos+2+segLen > len(data)` bounds check. Only the body that
runs per segment differs. This is the one duplication finding with a real
correctness cost — a fix to the bounds check or the marker set in one copy
silently leaves the other three wrong. Extract one iterator, something like
`forEachJPEGSegment(data []byte, fn func(marker byte, payload []byte) bool)`,
and reduce all four to their per-segment body. 4 of the 90 `DuplicatedCode`
hits, and the highest value in the report.

### Qodana: settingswin's numeric entries are five copies of one block

`internal/ui/settingswin/settingswin.go:208,217,226,238,250` — the max-scan,
max-width, max-height, image-cache, thumb-cache, and max-file-size entries
are each the same six lines: `widget.NewEntry()`, `Validator = positiveInt`,
`Text = strconv.Itoa(get())`, and an `OnChanged` that does
`strconv.Atoi` + `n > 0` (+ `n <= maxMemoryMB` for the three memory ones)
before calling the setter. One helper taking a getter, a setter, and an
optional upper bound collapses all six. 5 `DuplicatedCode` hits.

### Qodana: orientation.go's five transforms differ only by coordinate map

`internal/imaging/orientation.go` — `flipH`, `flipV`, `rotate180`,
`rotate90CW`, and `rotate270CW` are the identical bounds/alloc/nested-loop
body with a different `out.Set(...)` destination. Reported at lines 50, 64,
94, and 110. Worth collapsing, but check the cost first: the obvious shape
(pass a `func(x, y, w, h int) (int, int)`) puts an indirect call in the
per-pixel inner loop, which is exactly where this code cannot afford one.
Prefer a form that keeps the mapping inlinable, or leave these alone and
suppress — the duplication here is honest and stable, unlike the JPEG walk
above. 4 `DuplicatedCode` hits.

### Qodana: uitest.go wraps an APP1 segment twice

`internal/uitest/uitest.go:296` and `:497` both end with the same six lines
that build `Exif\x00\x00` + TIFF, prepend the `0xFF 0xE1` length header, and
splice the result in after `data[:2]`. A `wrapAPP1(data, tiff []byte) []byte`
helper covers both. `:458` and `:547` are flagged as part of the same
clusters. 4 `DuplicatedCode` hits.

### Qodana: 19 doc comments don't start with the element they document

`GoCommentStart`, all mechanical, all correct per Go doc convention:

- Five package comments in `internal/ui` that need the `Package ui ...` form
  or a blank line to detach them from the package clause: `build.go`,
  `components.go`, `favthumbs.go`, `shortcuts.go`, `startup.go`.
- Fourteen exported-element comments that don't lead with the name:
  `ui/autoupdate.go:10,25`, `ui/autoupdate/updater.go:71,78,119`,
  `ui/favthumbs.go:18`, `ui/load.go:516,521`, `ui/memlimits.go:86`,
  `ui/spiral/spiral.go:130`, `ui/widgets/style.go:29,42,66`,
  `ui/widgets/tappable.go:49`.

Several are the "A, B, and C implement X" shape (`tappable.go:49`), where the
fix is to lead with the first name rather than to rewrite the sentence.

### Qodana: small mechanical fixes

- `internal/imaging/save.go:183` — `err == io.EOF || err == io.ErrUnexpectedEOF`
  should be `errors.Is`. `io.ReadFull` documents returning those two
  unwrapped, so this is correct today; the change is about not depending on
  that. Same pattern in `internal/ui/exifwin/tiles_test.go:164,218,299`.
- `internal/ui/spiral/overlays.go:25,169,171,173` — name the fields in the
  `color.NRGBA{0, 0, 0, 191}` literals. Cheap readability win on colour
  constants where positional args are easy to misread. Four more of the same
  in `overlays_test.go:66-69`.
- `internal/imaging/raw_test.go:463` — `copy(head, []byte("FUJIFILM..."))`,
  the conversion is redundant; `copy` takes a string directly.
- `internal/update/apply_test.go:146` — local `real` shadows the builtin.
- `scripts/releasenotes/main.go:18`, `scripts/synctuf/main.go:22` — the
  `fmt.Fprintf(os.Stderr, ...)` results are unhandled. `_, _ =` matches how
  the rest of the tree marks a deliberately ignored error.

### Favorites un-merges the macOS native menu bar

`internal/ui/favorites` calls `f.menu.Refresh()` from two sites
(`SetHasFiles`, `refreshMenu`) with no `syncNativeMenuBar` follow-up.
`fyne.Menu.Refresh` is a `SetMainMenu`, which rebuilds the Darwin native
bar; only `refreshMainMenu` folds it back together afterwards. So adding,
renaming, or deleting a favorite already leaves a duplicate "Window" menu
and Command-prefixed accelerators on the unmodified letters until the next
`refreshMainMenu`. Pre-existing, not caused by the `viewer` field-cluster
refactor. The invariant worth stating: nothing outside `refreshMainMenu`
may call `Refresh()` on a menu that lives in the main bar.

### The manual-opened observer registration is untested

After `52af098` reduced the menu push sites to choke points, F1 and
Window → Help have no hand-written `syncMenus`; they rely on
`help.SetOnManualOpened(view.syncMenus)` staying registered in
`buildMainMenu`. Deleting that registration passes the entire test suite —
verified by mutation. No viewer-level test is possible because
`ShowManual` panics under Fyne's test theme (see the note at the tail of
`internal/ui/e2e_test.go`). `internal/ui/help`'s own
`TestHelp_SetOnManualOpenedFiresOnShow` pins that the hook *fires*, not
that `internal/ui` subscribes to it.

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
