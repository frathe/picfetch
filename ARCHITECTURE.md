# PicFetch — Architecture

Navigation map for AI agents. PicFetch is a Fyne desktop image viewer: one
binary, split into `internal/...` packages. Start here to find a file.
Standing rules (data flow, concurrency, conventions, build) live in
`AGENTS.md` — do not duplicate them here.

## Package map

### `github.com/frathe/picfetch` (package main)

Entry point only. `main.go` calls `openwith.Install` (first statement, see
`internal/openwith`), builds the `fyne.App`, loads embedded
`translations/*.json`, converts CLI paths to URIs (`argsToURIs`), and calls
`ui.Run`. `main_darwin_test.go` asserts the graft landed — this is the only
test binary that links the Cocoa driver.

### `internal/ui`

The application. Unexported `appState` is the file-set model (scan/drop
order, displayed order, index, sort, merge). Unexported `viewer` is the
Fyne façade. Construction order, overlay order, data flow, and concurrency:
see `AGENTS.md`. Features expose state; `internal/ui` composes them.

The concurrency invariant: see `AGENTS.md` § Concurrency and Fyne.

#### Its own files

| File(s) | Responsibility |
|---------|----------------|
| `run.go` | `Run`: restore startup viewer, start runtime (`favstore.DefaultDir`, position polling), register shutdown and CLI drop, enter the Fyne loop. |
| `build.go` | `buildViewer` composes widgets and `registerFeatures` modules. Overlay tail: grid, delete confirm, export prompt, toast. |
| `startup.go` | `loadStartupState` / `restoreStartupGeometry` / `buildStartupViewer` — the one load→build→restore path shared by `Run` and tests. |
| `components.go` | Dropzone, scan, sort, and info-overlay constructors. Toast stays in `toast.go`. |
| `features.go` | `registerFeatures` assigns help, EXIF, zoom, grid, deletion, slideshow, settings, then favorites. |
| `shortcuts.go` | `wireGlobalShortcuts` plus per-action `desktop.CustomShortcut` wiring (open, favorites, clipboard, delete, select-all, save, export, wallpaper). |
| `gesture.go` | Position-poller callback fans samples to `winPos` and `spiralDrag`; a recognised spiral calls `help.OpenSpiral`. |
| `windowtrack.go` | Main-window size tracker and position poller; `widgetGeometry` / `prefGeometry` translate `preferences.WindowGeometry` ↔ `widgets.Geometry`. |
| `windowmenu.go` | Window-menu Disabled/actions (`applyWindowMenuState`, `updateWindowMenuState`) and `refreshMainMenu` / Darwin sync entry points. |
| `windowmenu_darwin.go` | After Show and every native rebuild, fold Window items into GLFW’s `NSApp.windowsMenu` and clear AppKit’s default Command mask on unmodified letter accelerators. |
| `windowmenu_notdarwin.go` | No-op twin of the Darwin native-menu merge. |
| `testdata/` | Golden screenshots for the e2e suite. |
| `state.go` | Unexported `appState`. Only `viewer` accesses it. |
| `lifecycle.go` | `requestLifecycle` / `requestToken`. Load, scan, sort, and vector each own an instance. |
| `viewer.go` | Façade: title (`baseTitle` / `gridTitle` / `applyTitle`), reset/close, merge, Host vocabulary (`CurrentFile`, `ShowImage`, `RemoveFiles`, …). |
| `visibility.go` | `dupeFileSet` (adapts the viewer to `dupes.FileSet`); `jumpIfHiddenExtra`; `pushHideDuplicates`; the navigation helpers (`nextVisibleIndex` / `firstVisibleIndex` / `lastVisibleIndex` / `randomVisibleOther`) that read `v.dupes` instead of polling the grid overlay. |
| `keys.go` | `handleKeyEvent` / `handleTypedRune`. Return immediately while `Canvas().Overlays().Top()` is set (Fyne dialogs/menus). |
| `menu.go` | Menu bar composition: File, Favorites, Actions, Window, Help. Grid/slideshow mutual exclusion lives here, not in those packages. |
| `actionmenu.go` | Actions-menu Checked/Disabled and handlers (`applyActionsMenuState`). |
| `drop.go` | `handleDrop` / `applyScanResult` / `applyScannedFiles` glue over `filescan.Images` / `filescan.Siblings`; scan lifecycle is `viewer.scanOp`. |
| `openwith.go` | macOS "Open With" delivery: `installOpenWithHandler` / `openInitialFiles` / `openFilesFromOS` over `internal/openwith`, both routed through `fyne.Do` so a launch carrying argv files and a delivery makes one `handleDrop`. |
| `memlimits.go` | `settings` value plus memory-limit get/set that retune `imgCache`, grid thumb cache, `imaging.SetMaxEncodedBytes`, and the SVG raster cap. |
| `favthumbs.go` | Viewer glue for `favthumbs.Sync`, `gridSink`, and the favorite-preview `completion.Signal`. |
| `load.go` | `ShowImage` / `attemptLoad` / `finishLoad` (named steps in this file), neighbor preload (`AddIfFits`), GIF `animate`, `resizeToImage` / `syncWindowToZoom`. |
| `toast.go` | Self-dismissing notification card and `ShowToast`. |
| `info.go` | Persistent info overlay (I); EXIF link; RAW `(preview)` mark; `displayedDimensions`. |
| `asyncop.go` | `asyncOpUI` (lifecycle, active, done, spinner) — used only by scan and sort. |
| `sort.go` | `toggleSort` / `SetSortMode` / `startSort` / `finishSort` over `filesort.Order`; lifecycle is `viewer.sortOp`. |
| `rotate.go` | View-only 90° rotation. Call `updateFileMenuState` *before* `applyRotationLayout`. |
| `vector.go` | Debounced SVG re-render. |
| `save.go` | File > Save Changes (`canSaveRotation` / `saveRotation` / `updateFileMenuState`). |
| `export.go` | File > Export image (`promptExport` / `exportAs`) via `widgets.ChoiceCard` + `filepicker.ChooseSave`. |
| `wallpaper.go` | Set as Wallpaper: write a PNG into `viewer.wallpaperDir`, then `wallpaper.Set`. |
| `autoupdate.go` | Opt-in daily GitHub update check/download (`maybeStartUpdateCheck`), apply-on-stop (`applyStagedUpdate`), What's New cache helpers. |
| `slideshow.go` | `togglePictureFrameMode` (closes grid first) plus shuffle/interval bindings. |
| `batch.go` | Only file that knows both grid selection and deletion/clipboard exist. |
| `session.go` | `restoreSession` glue over `internal/session`. |
| `clipboard.go` | Copy-path / copy-image glue over `internal/clipboard`. |
| `openfiles.go` | Native open-dialog glue over `internal/filepicker`. |

#### Feature packages (`internal/ui/...`)

| Package | Responsibility | Reaches back via |
|---------|----------------|------------------|
| `internal/ui/zoom/` | Zoom/pan of the displayed image. Window growth is `syncWindowToZoom` in `internal/ui`. | `onChanged`, `modifiers`, `onScaleChanged`. |
| `internal/ui/grid/` | Overview (G): `GridWrap`, thumb cache, `decodepool`, `uiqueue.go`, search, badges, `marquee.go` (drag rectangle → `Targets()`), browse-duplicates (Shift+D), and `hashengine.go`'s pool-driven hashing pass that feeds `internal/dupes`. `nav.go`: `setHighlight` → `HighlightChanged`. Reads the model; does not own it. | 10-method `Host` including `Modifiers`. |
| `internal/ui/deletion/` | Shift+Delete confirm (`widgets.ChoiceCard`) then `trash.Move`. `RequestFiles` is the batch path; `Request` is the one-file wrapper. | 7-method `Host`. |
| `internal/ui/slideshow/` | Picture-frame mode (P): full-screen, auto-advance, interval, `winpos.Tracker` capture/restore. | 2-method `Host`. Knows nothing about the grid. |
| `internal/ui/exifwin/` | EXIF panel (E): tag list, optional JPEG strip, GPS map (`tiles.go`, `startWarm`). Geometry via `widgets.Singleton`. | 4-method `Host`. |
| `internal/ui/help/` | Manual, About, What's New (`whatsnew.go`), Help menu; embeds `manual.md` / `manual_de.md`. Secret search phrase and window-spiral both open `spiral/`. | Nothing — `New(app, title, art)` only. |
| `internal/ui/spiral/` | Full-screen shader easter egg. | `New(app)` only. |
| `internal/ui/settingswin/` | Settings window: form + checks, live `Host` setters, geometry via `Singleton`. | Getter/setter `Host` (sort, merge, slideshow, caps, caches, duplicate distance). |
| `internal/ui/favorites/` | Favorites menu and add/overwrite/manage/remove dialogs. `New` does no disk I/O; `SetDir` from `Run`. | 5-method `Host`. |
| `internal/ui/widgets/` | Shared UI mechanics: `ChoicePanel` / `ChoiceCard`, `TappableArea`, `Singleton` (+ geometry memory), `NewSizeTracker`, focus-ring style. | Leaf aside from `internal/winpos`. |
| `internal/ui/assets/` | `WelcomeWebP` / `PlaceholderWebP`. | Leaf. |

### `internal/imaging`

Viewer-independent probe → decode → EXIF-orient → cache pipeline (JPEG, PNG,
GIF including animated, WebP, BMP, TIFF, ICO, XPM, HEIC, AVIF, SVG, camera
RAW via embedded JPEG). RAW is preview-only (`LoadedImage.Preview`);
`CanEncode` is false. SVG is the only vector format (`svg.go` / `vector.go`).
Encode/write-back for a subset of formats lives in `save.go`.

| File | Responsibility |
|------|----------------|
| `bytecache.go` | `ByteCache[V]`: goroutine-safe LRU by estimated bytes. `Add` (displayed image) vs `AddIfFits` (speculative preload). |
| `loader.go` | `LoadedImage`, `NewImgCache`, `ReadAndProbe`, `DecodeLoaded`, `LoadImage`, `IsSupportedImage`, `SupportedExtensions`, `MaxEncodedBytes` / `InputTooLargeError`. |
| `raw.go` | Largest embedded JPEG from TIFF IFDs or SOI scan (CR3/RAF). |
| `svg.go` | SVG detection, logical-size floor (`MinVectorWidth`/`Height` = UI `startW`/`startH`), `ClampVectorRaster` / `MaxVectorRasterPixels`. |
| `vector.go` | `Vector` / `ParseVector` / `RasterAt`. |
| `exif.go` | Orientation tags + `ReadMetadata` / `Metadata` (including GPS IFD). JPEG APP1, then TIFF IFD0, then HEIC/AVIF, then RAW preview APP1. |
| `exififd.go` | Unexported IFD walker (`walkIFD`) and tag value helpers used by `exif.go` and `raw.go`. |
| `exifformat.go` | Unexported display formatters for exposure, focal length, and Exif dates (`formatExposureTime` / `formatFocalLength` / `formatExifDate` / `parseExifDateTime`). |
| `orientation.go` | `ApplyOrientation`, `RotateSteps`. |
| `gif.go` | Animated GIF compositing, `probeGIF`. |
| `thumbnail.go` | `LoadThumbnail` / `LoadThumbnailAndBounds` / `NewThumbCache`: same probe+decode, then downsample; `LoadThumbnailAndBounds` also returns native `ReadAndProbe` size for hide-duplicates. |
| `dhash.go` | `DifferenceHash` / `Hamming` / `DuplicateGroups` for grid hide-duplicates. |
| `jpegexif.go` | Unexported JPEG segment copy/strip for `save.go`. |
| `save.go` | `SaveRotated`, `Export`, `CanEncode` / `CanEncodeExt`, `StripJPEGMetadata`. |

### `internal/favstore`

Named file lists under a caller-supplied config directory. No UI.
`DefaultDir` is the production path helper.

| File | Responsibility |
|------|----------------|
| `favstore.go` | `Save` / `Load` / `Count` / `DefaultDir`; trash-backed remove. |

### `internal/favthumbs`

Disk-cached grid previews under `<favorite>/thumbs/`. `Sync` is the
background pass; `Sink` is the caller’s in-memory thumb cache.

| File | Responsibility |
|------|----------------|
| `store.go` / `name.go` | On-disk lookup and filename scheme. |
| `sync.go` | `Sync` walk: memory → disk → decode, then `Sink`. |
| `sweep.go` | Deletes stale preview files after a complete pass. |

### `internal/session`

Last-open file set via Fyne’s app-scoped cache.

| File | Responsibility |
|------|----------------|
| `session.go` | `Save`, `Load`. |

### `internal/update`

GitHub-release check, SHA-256 + immutable release attestation verify, stage, apply.

| File | Responsibility |
|------|----------------|
| `update.go` | `Client`, `AssetName`, `Newer`, `Due`. |
| `github.go` | Releases + release-attestation HTTP. |
| `checksums.go` | `VerifyHash` (optional API digest). |
| `download.go` / `extract.go` | Fetch, hash, unzip/tar, `Stage`. |
| `attest.go` | GitHub Fulcio Sigstore `Verifier` + in-toto release policy. |
| `tufroot.go` | Offline 60-day expiry check and verified sync of `embed/tuf-repo.github.com/root.json`. |
| `apply.go` / `apply_unix.go` / `apply_windows.go` | `Apply` dispatcher. |

### `internal/preferences`

Standing UI preferences via Fyne `Preferences` (not the session cache).
`SortMode` is a string on disk (`filesort.FromPref` / `Mode.PrefValue`).
Secondary-window geometry is `WindowGeometry` structs.

| File | Responsibility |
|------|----------------|
| `preferences.go` | `Save`, `Load`, `SaveLastUpdateCheckDay`, `State`, `WindowGeometry`. |

### `internal/wingesture`

Pure geometry: timestamped window positions in, spiral `Result` out. No
Fyne, no cgo. Positions are y-down (positive accumulated angle = clockwise
on screen). `realdrag_test.go` replays recorded title-bar drags.

| File | Responsibility |
|------|----------------|
| `wingesture.go` | `Direction`, `Result`, `Config`. |
| `detector.go` | Ring buffer, idle gap, one-shot `armed` latch. |
| `analyse.go` | Centroid, accumulated angle, sign consistency, radius-vs-angle fit. |

### `internal/winpos`

Fyne has no position getter and no move event. `Get` reads the native
handle. `Tracker` remembers the last good reading. `Poll` / `PollAt` sample
on a background goroutine and hop through `fyne.DoAndWait`.

| File | Responsibility |
|------|----------------|
| `winpos.go` | `Get`, `Set`, `Maximize`, `Unmaximize`. |
| `poll.go` | `PollAt` / `Poll` / `PollInterval` / `GestureInterval`. |
| `tracker.go` | `Tracker` atomics: `Store` / `Get` / `Capture` / `Restore`. |
| `darwin.go` / `windows.go` / `linux.go` / `other.go` | Platform position + maximize. Linux/Wayland: `Get` reports `ok=false`. |

OS integrations (`clipboard`, `filepicker`, `trash`, `wallpaper`) use
dispatcher vars and build-tagged platform files; tests stub them via
`internal/uitest` — see `AGENTS.md`.

### `internal/clipboard`

PNG image data (`CopyImage`) and file-reference lists (`CopyFiles`).

| File | Responsibility |
|------|----------------|
| `clipboard.go` | `CopyImage` dispatcher + per-OS image copy. |
| `copyfiles.go` | `CopyFiles` dispatcher + Linux/Windows file-list copy. |
| `darwin.go` / `other.go` | AppKit `NSPasteboard` file list / stub. |
| `windows.go` / `notwindows.go` | `hideConsoleWindow` pair. |

### `internal/filepicker`

Native open chooser (`Choose`) and save panel (`ChooseSave`). Linux/macOS
can pick folders; Windows is files-only.

| File | Responsibility |
|------|----------------|
| `filepicker.go` | `Choose` / `ChooseSave` / `ParseFileList` + Linux/Windows impls. |
| `darwin.go` / `other.go` | `NSOpenPanel` / `NSSavePanel` / stubs. |
| `windows.go` / `notwindows.go` | `hideConsoleWindow` pair. |

### `internal/trash`

Move to Trash/Recycle Bin (`Move`). Tests use `uitest.StubTrashMove`.

| File | Responsibility |
|------|----------------|
| `trash.go` | `Move` dispatcher + Linux/Windows impls. |
| `darwin.go` / `other.go` | AppKit recycle / stub. |
| `windows.go` / `notwindows.go` | `hideConsoleWindow` pair. |

### `internal/wallpaper`

Set desktop wallpaper (`Set`). UI writes a PNG into the app cache dir
before calling this.

| File | Responsibility |
|------|----------------|
| `wallpaper.go` | `Set` dispatcher + Linux/Windows impls. |
| `darwin.go` / `other.go` | AppKit set-desktop-image / stub. |
| `windows.go` / `notwindows.go` | `hideConsoleWindow` pair. |

### `internal/openwith`

macOS "Open With", Dock drop, `open -a`, and double-clicked associations —
none of which put files in `argv` for a bundled `.app`. AppKit turns the
`kAEOpenDocuments` Apple Event into a delegate call, which `Install` grafts
onto GLFW's delegate class. That event fires inside `glfw.Init()`, before
`SetOnStarted`, so `Deliver` buffers until `SetHandler` installs the
viewer's handler and flushes in the same critical section.

| File | Responsibility |
|------|----------------|
| `openwith.go` | The queue (`Deliver` / `SetHandler`) and `URIsFromFileURLs`. |
| `openwith_darwin.{go,h,m}` | `Install` / `DelegateRespondsToOpen` + the `application:openURLs:` / `application:openFiles:` graft. |
| `openwith_notdarwin.go` | Both report false; other OSes use `argv`. |

### `internal/filescan`

Recursive image gather for drop/open, plus a non-recursive sibling listing
when the user opened a single file.

| File | Responsibility |
|------|----------------|
| `filescan.go` | `Images(ctx, uris, max, progress)` (recursive); `Siblings(ctx, file, max, progress)` (parent dir only, opened file seeded first); symlink-cycle + per-call dedupe. |

### `internal/filesort`

Five orderings the S key cycles, plus `Label` (`lang.L`) and preference
string translation (`FromPref` / `PrefValue`).

| File | Responsibility |
|------|----------------|
| `filesort.go` | `Mode`, `Next`, `Order`, `Label`, `FromPref` / `PrefValue`. |

### `internal/selection`

Integer index set + range anchor for grid multi-select. No Fyne import.

| File | Responsibility |
|------|----------------|
| `selection.go` | `Set`, `Toggle`, `Add`, `Range`. |

### `internal/dupes`

Which files in a file set duplicate which others. Fyne-free — reached
through a `FileSet` interface with string keys, not `fyne.URI` — and its
only project import is `internal/imaging`. Owns dHashes and native pixel
sizes keyed by file (generation-scoped: `WipeIfStale` on a fresh drop,
`AdoptGeneration` on an incremental shrink), the Hamming distance
threshold, the installed group snapshot (representative = highest native
pixel count, lowest index on a tie), the hide-duplicates and inspect
modes, and the visibility queries (`IsVisible` / `NextVisible` /
`FirstVisible` / `LastVisible` / `VisibleIndexesExcept`) plain navigation
asks. `internal/ui` owns the `Model` and implements `FileSet`;
`internal/ui/grid` reads and feeds it (hashing pass, browse, badges) but
does not own it.

| File | Responsibility |
|------|----------------|
| `dupes.go` | `Model`, `FileSet`, hash/native facts, generation (`WipeIfStale` / `AdoptGeneration`), distance clamp, `OnChange` observers. |
| `groups.go` | `Groups` snapshot, `Compute` / `Install` / `Rebuild`, `GroupSize` / `RepresentativeOf` / `Members`. |
| `visible.go` | Hide/inspect modes (`SetHideDuplicates`, `BeginInspect` / `ClearInspect` / `InspectMembers`), `IsHiddenExtra`, and the visibility/navigation queries. |

### `internal/decodepool`

Bounded workers + per-key in-flight claim. Grid thumbs:
`Pool[*fyne.Container, int]`. Viewer preloads: `Pool[string, struct{}]`.
Cell staleness stays in `grid/thumbs.go`.

| File | Responsibility |
|------|----------------|
| `decodepool.go` | `Pool[K,V]`, `Claim` / `Release`, `Go` / `Wait`. |

### `internal/completion`

One-shot “this background op finished” signal. Named wait helpers vs
`drain`: see `AGENTS.md` § Concurrency and Fyne.

| File | Responsibility |
|------|----------------|
| `completion.go` | `Signal` (`Begin` / `Wait` / `Begun` / `Current`) and `Handle`. |

### `internal/uitest`

Test-only fixtures and OS-seam stubs. Never imported from production files.

| File | Responsibility |
|------|----------------|
| `uitest.go` | Temp URIs, synthetic images (including GPS JPEG, SVG), `ApproxEqual`. |
| `stubs.go` | `StubChooser`, `StubSaveChooser`, `StubClipboardCopy` / `CopyFiles`, `StubTrashMove`, `StubWallpaperSet`. |
| `uiqueue.go` | Drainable `UIQueue`. |

Wait helpers (`waitUntilLoaded`, `dropAndWait`, …) stay in
`internal/ui/harness_test.go`.

## Translations

`main.go` owns `//go:embed translations/*.json` and `lang.AddTranslationsFS`.
`filesort.Label` and `internal/filepicker` are the `lang.L` call sites
outside the UI tree. `main_test.go`
checks locale parity and that `en.json` is an identity map. String rule:
see `AGENTS.md`.

## Error handling

`fyne.LogError` is used in `main.go`, `internal/ui` glue, and
`internal/session`. Viewer-independent packages return errors. Ignore rule:
see `AGENTS.md`.

## Where to look for X

- "How is an image loaded/decoded/cached?" → `internal/imaging/loader.go` (`raw.go` for camera-RAW previews).
- "How is image memory bounded, and where are the limits set?" → `internal/imaging/bytecache.go` + `internal/ui/memlimits.go` + `gif.go` + `loader.go` `MaxEncodedBytes`.
- "Where does the EXIF panel live?" → `internal/ui/exifwin`.
- "Why does the help package import a GLSL shader?" → `internal/ui/spiral`, opened from `help/manual.go`’s `secretPhrase` or `internal/ui/gesture.go`.
- "Why is the log not full of `tile fetch error`?" → `internal/ui/exifwin/tiles.go` `quietPendingTiles` / `tileLogFilter`.
- "Why doesn't the EXIF window's map freeze the app while it loads?" → `exifwin/tiles.go` + `startWarm` / `syncLoading`.
- "Why is the info overlay's 'Show EXIF data' link missing?" → `info.go` `syncInfoOverlayVisibility` + `viewer.currentHasEXIF`.
- "Where is a photo's GPS position read, and where is it shown?" → `internal/imaging/exif.go` `parseGPSIFD` + `exifwin` Location section.
- "How is EXIF orientation handled?" → `internal/imaging/exif.go` + `orientation.go`.
- "How is a camera RAW file shown?" → `internal/imaging/raw.go` + `LoadedImage.Preview` + `load.go` / `info.go`.
- "How does drag-and-drop / folder scanning work?" → `internal/filescan.Images` / `filescan.Siblings` (single-file case) + `drop.go` `handleDrop`.
- "How is an image shown/preloaded/animated once loaded?" → `load.go`.
- "Which keys do what?" → `keys.go` (`handleKeyEvent` / `handleTypedRune`) + `shortcuts.go`.
- "How do I find one file by name in a big drop?" → `internal/ui/grid/search.go` + `keys.go` `handleTypedRune`.
- "How does hide-duplicates work?" → `internal/dupes` (the model: grouping, hide/inspect modes, visibility — `BeginInspect` / `InspectMembers` / `IsHiddenExtra` / `NextVisible`) + `internal/imaging/dhash.go` (the hash and its Hamming linkage) + `internal/ui/grid/hashengine.go` (the pool-driven pass that fills the model) + `grid/dupes.go` (browse). Escape reopen: `internal/ui` `reopenVariantGrid`.
- "How do I act on several images at once?" → `internal/selection` + `grid/selection.go` `Targets` + `grid/marquee.go` + `batch.go` + `deletion.RequestFiles` / `clipboard.CopyFiles`.
- "How does zoom/pan work?" → `internal/ui/zoom`; keys in `keys.go`; window resize in `load.go` `syncWindowToZoom`.
- "How does an SVG stay sharp when I zoom?" → `internal/imaging/vector.go` `RasterAt` + `svg.go` + `internal/ui/vector.go` + zoom `SetLogicalSize` / `onScaleChanged`.
- "How does rotation work, and how is it saved to disk?" → `internal/ui/rotate.go` + `internal/ui/save.go` + `internal/imaging/save.go`.
- "How do I write an image out in a different format?" → `internal/ui/export.go` + `filepicker.ChooseSave` + `imaging.Export`.
- "How does 'Set as Wallpaper' work?" → `internal/ui/wallpaper.go` + `internal/wallpaper`.
- "How does the slideshow / picture-frame mode work?" → `internal/ui/slideshow` + `slideshow.go`.
- "How does delete work?" → `internal/ui/deletion` + `internal/trash` + `shortcuts.go` / `batch.go` `requestDelete`.
- "How are native file dialogs implemented?" → `internal/filepicker` + `openfiles.go` / `export.go`.
- "How is the last session saved/restored?" → `internal/session` + `session.go` `restoreSession`.
- "How do in-app updates work?" → `internal/update` + `internal/ui/autoupdate.go` + `help/whatsnew.go`. Off by default (`preferences.CheckForUpdates`). Apply is OnStopped, not a relaunch. GitHub TUF bootstrap expiry: `tufroot.go`.
- "How are GitHub release notes written?" → `todos.md` `## Done` + `scripts/releasenotes` + `make release` + `.github/workflows/release.yml` `body_path`.
- "How does a macOS Open With reach the viewer?" → `internal/openwith` (queue + Objective-C graft) + `main.go` `openwith.Install` + `internal/ui/openwith.go` + `run.go` `SetOnStarted`.
- "How does the packaged macOS app declare file/folder associations (Open With)?" → `internal/imaging/loader.go` `SupportedExtensions` + `scripts/plistdoctypes` + `Makefile` `package-mac`.
- "How do Favorites work?" → `internal/favstore` + `internal/ui/favorites` + `shortcuts.go` + `viewer.OpenFiles`.
- "How are favorite previews cached on disk?" → `internal/favthumbs` + `internal/ui/favthumbs.go` + `favorites` + `grid` thumb accessors.
- "Where is the File menu / Settings window?" → `menu.go` `buildMainMenu` + `actionmenu.go` + `settingswin` + `viewer.closeFiles`.
- "How are preferences (sort order, merge mode, slideshow interval/shuffle, folder-scan cap, window-size cap, window size/position, favorite-preview-cache toggle, check-for-updates checkbox) persisted?" → `internal/preferences` + `startup.go` + `features.go` + `windowtrack.go` + `run.go` `currentPreferences`.
- "How do the Settings and EXIF windows come back where I left them?" → `widgets.Singleton.Remember` / `Geometry` / `StopTracking` + `winpos.Poll` + `preferences.WindowGeometry`.
- "How is the window's on-screen position read back, since Fyne has no getter for it?" → `internal/winpos` + `windowtrack.go` `startWindowPosPolling`.
- "How can dragging the window open something?" → `internal/wingesture` + `gesture.go` + `help.OpenSpiral` / `spiral.ShowForGesture`.
- "How does copy-image-to-clipboard work?" → `internal/clipboard` + `clipboard.go`. Batch file copy: `copyfiles.go` + `batch.go` `copySelection`.
- "How does the grid overview / thumbnail generation work?" → `imaging/thumbnail.go` + `grid/grid.go` + `grid/thumbs.go` + `grid/hashengine.go` + `grid/nav.go` + `grid/uiqueue.go`.
- "What decides the window title?" → `viewer.go` `setTitle` / `applyTitle` / `HighlightChanged` + `load.go` + `grid/nav.go`.
- "How do I write a test that needs an image / a viewer?" → `internal/uitest` + `newTestViewer` / `newTestUI` + `dropAndWait` in `harness_test.go`.
- "How do I add or translate a user-visible string?" → `lang.L` at the call site and the same key in every `translations/` bundle. See `AGENTS.md`.
- "Why isn't feature X its own package?" → `AGENTS.md`.
- "How/where are errors reported, and when is it OK to ignore one?" → `AGENTS.md`.

## Keeping this doc current

Update this file in the same change when a package is added, removed,
renamed, or files move between packages. Cells are locators (path, symbol,
order) — one sentence. Standing rules go in `AGENTS.md`, not here.
