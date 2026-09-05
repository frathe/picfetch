# PicFetch — Architecture

Navigation map for AI agents. PicFetch is a Fyne desktop image viewer: one
binary, split into `internal/...` packages. Start here to find a file.
Standing rules (data flow, concurrency, conventions, build) live in
`AGENTS.md` — do not duplicate them here.

## Package map

### `github.com/frathe/picfetch` (package main)

Entry point only. `main.go` parses the command line (`launchArgs`, see
`internal/launch`) before any side effect, calls `openwith.Install` (first
statement after that, see `internal/openwith`), skips GitHub-update predecessor
cleanup for Store-managed builds, builds the `fyne.App`, loads embedded
`translations/*.json`, converts CLI paths to URIs (`argsToURIs`), and calls
`ui.Run`. `main_darwin_test.go` asserts the graft landed — this is the only
test binary that links the Cocoa driver.

### `scripts/testshards`

Repository tooling for measured Linux race-test sharding. It parses complete
`go test -json` streams, emits deterministic package/top-level-test summaries,
plans a reviewable manifest from median durations with stable LPT tie-breaking,
checks that manifest against the build-selected Linux test inventory, derives
the fail-closed non-UI package partition, and emits exact shard filters. Its
capture path streams compact partition/package/test diagnostics while retaining
the raw events outside the repository; the generated assignment lives at
`.github/testshards/internal-ui.tsv`.

| File | Responsibility |
|------|----------------|
| `main.go` | `summarize`, `plan`, `check`, `regex`, `partition`, and `capture` command paths: complete event-stream validation, deterministic assignment, live build-selected inventory and parallel-call checks, exact anchored filters, exact-package subtraction, concise diagnostics, and raw preservation. |
| `main_test.go` | Command-boundary fixtures for event streams and capture, deterministic planning, every manifest rejection, build-selected runnable forms, parallel-call refusal, exact filter generation, package partitioning, and Make contract expansion. |

### `internal/ui`

The application. Unexported `appState` is the file-set model (scan/drop
order, displayed order, index, sort, merge) and publishes an immutable
`dupes.Snapshot` — file keys plus generation — atomically on every write to
`files`; `viewer.Generation()` reads the generation out of that snapshot
rather than a separate counter. Unexported `viewer` is the Fyne façade.
Construction order, overlay order, data flow, and concurrency: see
`AGENTS.md`. Features expose state; `internal/ui` composes them.

The concurrency invariant: see `AGENTS.md` § Concurrency and Fyne.

#### Its own files

| File(s) | Responsibility |
|---------|----------------|
| `run.go` | `Run`: restore startup viewer, start runtime (`favstore.DefaultDir`, position polling), register shutdown and CLI drop, enter the Fyne loop. Shutdown cancels an active comparison before the event loop stops. Store-managed builds skip GitHub update startup and staged-binary apply. |
| `build.go` | `buildViewer` composes widgets and `registerFeatures` modules and snapshots `distribution.StoreManaged` onto the viewer. Overlay tail: copy selection, grid, comparison (including its pointer shield), delete confirm, export prompt, toast. Desktop canvases also receive the chained comparison key-down hook for exact physical `Ctrl+L`; ordinary typed-key and shortcut wiring remains separate. |
| `startup.go` | `loadStartupState` / `restoreStartupGeometry` / `buildStartupViewer` — the one load→build→restore path shared by `Run` and tests. |
| `components.go` | Dropzone, scan, sort, and info-overlay constructors. Toast stays in `toast.go`. |
| `features.go` | `registerFeatures` assigns help, EXIF, zoom, copy selection, grid, comparison, mosaic window, deletion, slideshow, settings, then favorites. |
| `shortcuts.go` | `wireGlobalShortcuts` plus per-action shortcut wiring (open, favorites, clipboard, copy selection, comparison, delete, select-all, save, export, wallpaper). Comparison registers the native `Cmd/Ctrl+D` plus physical `Ctrl+D` when those differ. `yieldingShortcuts` blocks ordinary commands during comparison and otherwise yields Copy Selection; Open is admitted only far enough to show comparison's refusal. Copy Selection and clipboard bindings also defend their own direct entries. |
| `gesture.go` | Position-poller callback fans samples to `winPos` and `spiralDrag`; a recognised spiral calls `help.OpenSpiral`. |
| `windowtrack.go` | Main-window size tracker and position poller; `widgetGeometry` / `prefGeometry` translate `preferences.WindowGeometry` ↔ `widgets.Geometry`. |
| `windowmenu.go` | Window-menu action handlers (`showViewer`, `showWindowExif`, `showWindowGrid`, `showWindowPictureFrame`, `showWindowHelp` — grid/picture-frame mutual exclusion lives in the first two) plus `refreshMainMenu` / `syncNativeMenuBar` and the Darwin sync entry points. The Checked/Disabled matrix itself lives in `internal/ui/menus`. |
| `windowmenu_darwin.go` | After Show and every native rebuild, fold Window items into GLFW’s `NSApp.windowsMenu` and clear AppKit’s default Command mask on unmodified letter accelerators. |
| `windowmenu_notdarwin.go` | No-op twin of the Darwin native-menu merge. |
| `testdata/` | Golden screenshots for the e2e suite. |
| `state.go` | Unexported `appState`. Only `viewer` accesses it. |
| `lifecycle.go` | `requestLifecycle` / `requestToken`. Load, scan, sort, vector, and copy-selection encode each own an instance. |
| `viewer.go` | Façade: title (`baseTitle` / `gridTitle` / comparison ownership / `applyTitle`), reset/close, merge, Host vocabulary (`CurrentFile`, `ShowImage`, `RemoveFiles`, …). |
| `visibility.go` | `dupeFileSet` (adapts the viewer to `dupes.FileSet` by forwarding `appState`'s published `dupes.Snapshot`); `jumpIfHiddenExtra`; `pushHideDuplicates`; the navigation helpers (`nextVisibleIndex` / `firstVisibleIndex` / `lastVisibleIndex` / `randomVisibleOther`) that read `v.dupes` instead of polling the grid overlay. |
| `keys.go` | `handleKeyEvent` / `handleTypedRune`, plus a chained desktop key-down hook that requests the ready-gated comparison link toggle on exact physical `Ctrl+L` without key-repeat flapping. Return immediately while `Canvas().Overlays().Top()` is set (Fyne dialogs/menus). Comparison owns all main-window typing: Escape closes it, F1 opens Help, `0` / `1` / `+` / `-` reach its shared camera or hovered photo pose, swipe-mode `Left` / `Right` / `Home` / `End` reach its divider, and every other key/rune stops before the still-open grid. Copy Selection: `HandleKey` consumes Escape/copy/navigation; unowned keys `yieldCopySelection` except modifier-only and zoom keys. |
| `menu.go` | `buildMainMenu` builds `internal/ui/menus.Menus` and assembles the bar: File, Favorites, Actions, Window, Help. `yieldingMenuCallbacks` enforces comparison isolation and Copy Selection yielding at callback entry. `menuState()` is the one function that builds the `menus.State` snapshot; `syncMenus()` applies it, pushes comparison/file availability into Favorites, and refreshes the native bar only when something actually changed. |
| `actionmenu.go` | Comparison-guarded Actions-menu handlers (`setActionsSort`, `toggleActionsHideDuplicates`, `showActionsVariant`, `rotateActionsImage`, …). The Checked/Disabled matrix lives in `internal/ui/menus`. |
| `drop.go` | `handleDrop` / `applyScanResult` / `applyScannedFiles` glue over `filescan.Images` / `filescan.Siblings`; scan lifecycle is `viewer.scanOp`. A non-empty drop is refused before any state change while comparison is active. |
| `openwith.go` | macOS "Open With" delivery: `installOpenWithHandler` / `openInitialFiles` / `openFilesFromOS` over `internal/openwith`, both routed through `fyne.Do` so a launch carrying argv files and a delivery makes one `handleDrop`. The combined pending set is cleared before that shared path refuses an active comparison, so deliveries cannot queue behind it. |
| `memlimits.go` | `settings` value, `settingsState` / `ApplySettings`, memory-limit get/set that retune caches and `imaging.SetMaxEncodedBytes`. |
| `theme.go` | Settings-facing appearance getter/setter; applies `internal/appearance` modes live. |
| `favthumbs.go` | Viewer glue for `favthumbs.Sync`, `gridSink`, and the favorite-preview `completion.Signal`. |
| `load.go` | `ShowImage` / `attemptLoad` / `finishLoad` (named steps in this file), neighbor preload (`AddIfFits`), GIF `animate`, `autoResizeToImage` / `resizeToImage` / `syncWindowToZoom` (static-size gate). |
| `toast.go` | Self-dismissing notification card and `ShowToast`. |
| `info.go` | Persistent info overlay (I); EXIF link; RAW `(preview)` mark. |
| `asyncop.go` | `asyncOpUI` (lifecycle, active, done, spinner) — used only by scan and sort. |
| `sort.go` | `toggleSort` / `SetSortMode` / `startSort` / `finishSort` over `filesort.Order`; lifecycle is `viewer.sortOp`. |
| `rotate.go` | View-only 90° rotation (state lives in `internal/ui/display`). `displayedDimensions` is the oriented raster/SVG size. Call `syncMenus` *before* `applyRotationLayout` — a documented `-race` fix under the fake test driver, not call-site discipline. |
| `vector.go` | Debounced SVG re-render. |
| `save.go` | File > Save Changes (`canSaveRotation` / `saveRotation`, ending in `syncMenus`). |
| `export.go` | File > Export image (`promptExport` / `exportAs`) via `widgets.ChoiceCard` + `filepicker.ChooseSave`. |
| `mosaic.go` | Cross-feature mosaic composition: snapshot explicit Grid selection or the complete filtered result, inspect displays, open the singleton workflow, and adapt generation/display refresh into its narrow Host. |
| `wallpaper.go` | Shared ordinary/mosaic wallpaper lifecycle: serialize work, export captured pixels to a global or hashed-target cache scope, call `wallpaper.Set(Request)`, and sweep only copies no longer backing another scope. |
| `autoupdate.go` | Viewer-side update glue: `maybeStartUpdateCheck` gates the opt-in daily check; `CheckForUpdatesNow` adapts manual worker callbacks through `fyne.Do` with an inner staleness check; `PerformUpdate` records relaunch intent and requests quit; `maybeShowWhatsNew` opens cached release notes. Every update entry point refuses a Store-managed viewer. Policy, staging, and cache live in `internal/ui/autoupdate`. |
| `slideshow.go` | `togglePictureFrameMode` (closes grid first) plus shuffle/interval bindings. |
| `batch.go` | Routes delete/copy commands by the current subject: image-region selection, grid selection, or displayed image. |
| `session.go` | `restoreSession` glue over `internal/session`. |
| `clipboard.go` | Copy-path / copy-image glue over `internal/clipboard`. |
| `reveal.go` | Actions > "Reveal in file manager" (`Cmd/Ctrl+R`) and the info overlay's link of the same name: current-file path, own goroutine behind `viewer.reveal`, toast on failure. Glue over `internal/filemanager`. |
| `copyselection.go` | Viewer adapter for `internal/ui/copyselection`: availability, start/cancel, zoom `Geometry` to `View`, animation pause, clipboard worker, `yieldCopySelection`. Command entry yields through `yieldingMenuCallbacks`, `yieldingShortcuts`, `handleKeyEvent`, and `handleDrop`. |
| `compare.go` | Viewer adapter for `internal/ui/compare`: validates exactly two explicit grid selections, resolves ascending host indices to URIs, unfocuses the covered grid so desktop modifier hooks remain reachable, and loads through the canonical full-image cache/probe/decode path. That path preserves EXIF-corrected pixels, RAW previews, animation decoding/budget policy, encoded-input limits, and the original first frame; the feature deliberately freezes animation. The adapter owns the exact comparison-window title callback and reports failures without mutating the grid or file set. `comparisonActive()` is the composition layer's sole exclusive-mode fact; `refuseOpenDuringComparison()` owns the localized discard policy. |
| `animationpause.go` | Serializes animated-frame advancement with Copy Selection's stable source capture. |
| `openfiles.go` | Native open-dialog glue over `internal/filepicker`; both dialog entry and chooser execution refuse an active comparison before starting external work. |

#### Feature packages (`internal/ui/...`)

| Package | Responsibility | Reaches back via |
|---------|----------------|------------------|
| `internal/ui/zoom/` | Zoom/pan of the displayed image. `Geometry` / `HandleScroll` / `SetOnGeometryChanged` are the presentation seam Copy Selection uses; this package does not import `copyselection`. Window growth is `syncWindowToZoom` in `internal/ui`. | `onChanged`, `modifiers`, `onScaleChanged`. |
| `internal/ui/copyselection/` | Transient Copy Selection mode: image-region geometry, overlay, and captured `Source` crop/encode. `HandleKey` reports whether the mode consumed the key. | `Copy`, `Ended`, `Scroll`. |
| `internal/ui/grid/` | Overview (G): `GridWrap`, thumb cache, `decodepool`, `uiqueue.go`, search, badges, explicit host-index selection plus its change observer, `marquee.go` (drag rectangle → `Targets()`), browse-duplicates (Shift+D), and `hashengine.go`'s pool-driven hashing pass that feeds `internal/dupes`. `nav.go`: `setHighlight` → `HighlightChanged`. Reads the model; does not own it. | 10-method `Host` including `Modifiers`. |
| `internal/ui/compare/` | Opaque main-window comparison surface: switchable gapless 50/50 and full-viewport swipe layouts compose two persistent photo transforms with one shared camera transform; each image has one reveal clip so swipe keeps aligned image coordinates. In Swipe, each pane input mirrors its current reveal even though the render viewport remains full-size, and reveal-local wheel coordinates are translated back into that viewport. The ready-gated top-left Unlink/Link control and physical `Ctrl+L` share `ToggleLink`; its adjacent status reports only the active unlinked target, while layout/Swap/Back stay in a separate top-right card. `ToggleLink` changes only input ownership and never changes rendered geometry. Linked pan/zoom moves the camera, linked `0` frames both current photo poses without rewriting them, and linked `1` returns the camera home. Unlinked pointer input and transform keys target the hovered or last-hovered photo; its `0` / `1` fit or show that photo at decoded-pixel size in the current camera. Photo centers and camera movement stop when an image edge reaches its pane center. Resize and layout preserve both photo poses and the camera; Swap deliberately clears divergence from the last-targeted visible pose before exchanging sources. A private `paneRenderer` scene seam keeps transforms independent from presentation; production owns two stable `canvas.Shader` objects while tests can inject the canvas reference adapter. Each immutable render source retains the canonical decoded frame, a long-edge-1024 overview, and a 64 MiB detail-tile cache. The planner uses physical display density and the actual side-by-side/swipe reveal, skips details when the overview is sufficient, and binds at most seven guttered detail tiles without shuffling stable sampler slots. One cancellable worker per pane generates tiles; publications are coalesced and marshalled through the feature's `UIQueue`. Pan/zoom changes shader geometry and uniforms without repainting the viewer root. Each SVG still gets a pane-local device-pixel raster, clamped by `imaging.ClampVectorRaster`, before entering the same overview/tile path. `Settle` covers load, vector, tile, and causal queued completions with reusable channel-epoch barriers. Fyne's software test painter does not render `canvas.Shader`, so deterministic pixel tests use the reference adapter; native runtime acceptance uses the GL painter. The feature also owns divider input, permanent chrome, ready-gated layout/link/Swap controls, the input shield, and the replaceable completion signal. It receives an ordered URI pair and never reads or mutates grid/viewer state. | `Loader` plus `Callbacks` (`Repaint`, `Closed`, `Failed`, `OrderChanged`, `Modifiers`). |
| `internal/ui/mosaicwin/` | Dedicated configuration/preview window with immutable command-entry sources/topology, accessible controls and focus order, distinct labels for identically named displays while preserving target IDs, cancellable generation lifecycle, stale-result rejection, exact-result export, targeted wallpaper callback, and remembered geometry. It is a secondary window, not a main-window overlay. | 3-method `Host` (`GenerateMosaic`, `InspectMosaicDisplays`, `SetMosaicWallpaper`). |
| `internal/ui/deletion/` | Shift+Delete confirm (`widgets.ChoiceCard`) then `trash.Move`. `RequestFiles` is the batch path; `Request` is the one-file wrapper. | 7-method `Host`. |
| `internal/ui/slideshow/` | Picture-frame mode (P): full-screen, auto-advance, interval, `winpos.Tracker` capture/restore. | 2-method `Host`. Knows nothing about the grid. |
| `internal/ui/exifwin/` | EXIF panel (E): tag list, optional JPEG strip, GPS map (`tiles.go`, `startWarm`). Geometry via `widgets.Singleton`. | 4-method `Host`. |
| `internal/ui/help/` | Manual, About, What's New (`whatsnew.go`), Help menu; embeds `manual.md` / `manual_de.md`. Secret search phrase and window-spiral both open `spiral/`; `finis` in manual search opens the cursor-following companion (`finis.go`, embedded `finis.webp`). | Nothing — `New(app, title, art)` only. |
| `internal/ui/spiral/` | Full-screen shader easter egg. | `New(app)` only. |
| `internal/ui/settingswin/` | Settings: General/Appearance/Updates/Limits, update dialogs, snapshot seed, live apply, Singleton geometry. `Show(State, storeManaged)` replaces the GitHub update controls with Store-owned-update copy when applicable. | `Show(State, bool)` + Host (`ApplySettings`, `CheckForUpdatesNow`, `PerformUpdate`). |
| `internal/ui/favorites/` | Favorites menu and add/overwrite/manage/remove dialogs. `New` does no disk I/O; `SetDir` from `Run`. `SetCommandsEnabled` preserves Add's file availability while disabling both static and dynamically rebuilt menu entries during comparison. | 6-method `Host`. |
| `internal/ui/menus/` | The stateful File/Window/Actions menu items and their whole Checked/Disabled matrix as `Apply(State) (changed bool)`, a pure function of a value snapshot. `ComparisonActive` applies a final all-ordinary-items-disabled override while leaving Help available. Fyne-typed but viewer-free, unit-testable with no app. Menu-bar assembly, the Darwin native-bar fold, the real shortcut bindings, and every action the items run all stay in `internal/ui`. | No Host: `Apply(State)` over a value snapshot built by `menu.go`'s `menuState()`. |
| `internal/ui/autoupdate/` | Shared serialized automatic/manual update worker: lazy verifier/client preparation, check/download progress events, matching-stage reuse, all-worker settle, last-check-day persistence, staged apply/relaunch intent, the What's-New cache (`whatsnew.go`), and the apply-failure cache (`applyfailure.go`) — `ApplyStagedUpdate` writes it when `update.Apply` fails, and `internal/ui` reads and clears it on the next launch. Both caches are one JSON document each in `app.Cache()`, over the `saveCacheJSON` / `loadCacheJSON` / `clearCacheJSON` helpers in `cache.go`; a failed relaunch is deliberately *not* recorded, since it happens after the new binary is installed and verified. | No Host: takes a `context.Context` and a staleness func per call (`Start` / `StartManual`), plus `Persist` and per-`Updater` verifier-factory seams — cancellation stays the viewer's own `requestLifecycle`, not promoted here. |
| `internal/ui/infoview/` | The persistent info overlay (I key): its four widgets - text, the EXIF link, the reveal link, the card - the current file's raw facts (byte size, EXIF presence, RAW-preview flag), its own toggle preference, and `formatFileSize`. The EXIF link follows `HasEXIF`; the reveal link is shown with the card itself. | No Host: `Update(State)` / `Sync(bool, State)` over a value snapshot built by `info.go`'s `infoState()`. |
| `internal/ui/display/` | What's currently on the canvas: the decoded frames, which one is up, the view-only rotation (composing `imaging.RotateSteps` itself, in `Rotated`), and the picture-frame crossfade. | No Host: a value `State` field on `viewer`, mutated through its own methods, never copied. |
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
| `bytecache.go` | `ByteCache[V]`: goroutine-safe LRU by estimated bytes. `Add` (displayed image) vs `AddIfFits` (speculative preload). `LoadedImage.DecodedBytes` shares retained pixel/vector accounting with the mosaic repeat cache. |
| `loader.go` | `LoadedImage`, `NewImgCache`, `ReadAndProbe`, `DecodeLoaded`, `LoadImage`, `IsSupportedImage`, `SupportedExtensions`, `MaxEncodedBytes` / `InputTooLargeError`. |
| `raw.go` | Largest embedded JPEG from TIFF IFDs or SOI scan (CR3/RAF). |
| `svg.go` | SVG detection, logical-size floor (`MinVectorWidth`/`Height` = UI `startW`/`startH`), `ClampVectorRaster` / `MaxVectorRasterPixels`. |
| `vector.go` | `Vector` / `ParseVector` / `RasterAt`. |
| `exif.go` | Orientation tags from JPEG APP1, PNG eXIf, WebP EXIF, and TIFF IFD0 + `ReadMetadata` / `Metadata` (including GPS IFD). Metadata scans JPEG APP1, then TIFF IFD0, then HEIC/AVIF, then RAW preview APP1. |
| `exififd.go` | Unexported IFD walker (`walkIFD`) and tag value helpers used by `exif.go` and `raw.go`. |
| `exifformat.go` | Unexported display formatters for exposure, focal length, and Exif dates (`formatExposureTime` / `formatFocalLength` / `formatExifDate` / `parseExifDateTime`). |
| `orientation.go` | `ApplyOrientation`, `RotateSteps`. |
| `gif.go` | Animated GIF compositing, `probeGIF`, and logical-canvas restoration for a frozen partial first frame without decoding later frames. |
| `thumbnail.go` | `LoadThumbnail` / `LoadThumbnailAndBounds` / `NewThumbCache`: same probe+decode, then downsample; `LoadThumbnailAndBounds` also returns native `ReadAndProbe` size for hide-duplicates. |
| `dhash.go` | `DifferenceHash` / `Hamming` / `DuplicateGroups` for grid hide-duplicates. |
| `jpegseg.go` | Unexported JPEG header-segment walker (`walkJPEGSegments`) used by `exif.go` and `jpegexif.go`. Stops at SOS; does not walk entropy-coded scans (`jpegLength` in `raw.go`) or copy/strip (`stripJPEGSegments` in `jpegexif.go`). |
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
| `download.go` / `extract.go` | Fetch with optional `DownloadProgress`, hash, attest, unzip/tar, and persist `Stage` provenance plus extracted-file hashes for reuse/apply revalidation. |
| `attest.go` | GitHub Fulcio Sigstore `Verifier` + in-toto release policy. |
| `tufroot.go` | Offline 60-day expiry check and verified sync of `embed/tuf-repo.github.com/root.json`. |
| `apply.go` / `apply_unix.go` / `apply_windows.go` | `Apply` dispatcher with `ApplyOptions`, normal shutdown without relaunch, explicit Perform-update relaunch. Unix (`apply_unix.go`) writes `<dest>.new` beside the target, renames it into place, and rolls back through `<dest>.old` on failure. Windows (`apply_windows.go`) replaces the running executable in-process via `swapBinary` — it used to run a generated `<dest>.apply.cmd` through `cmd.exe`, which Controlled Folder Access refuses outright regardless of `cmd.exe`'s own Microsoft signature; that script is gone. |
| `swap.go` | `swapBinary`: the Windows in-process replace. Renames the running executable to `<dest>.old` (the one replacement Windows allows on a running image), copies the staged binary over `dest`, SHA-256-verifies the copy against the stage, and *tries* to restore `<dest>.old` on any failure past the rename — the rename back is retried a few times and then falls back to copying the backup over `dest`, but if all of that is refused too, `dest` is left truncated or missing and the reported `Op` is `restore`. That is the one outcome PicFetch cannot recover from or even report on the next launch, since reading the record needs the executable that is broken. `<dest>.old` deliberately survives a successful swap — it is still this process's own running image — for the next launch to sweep (`await.go`). |
| `applyerr.go` | `ApplyError` (`Op`/`Path`/`Err`) and `FailureReason` (`ReasonAccessDenied` / `ReasonVirusBlocked` / `ReasonSharingViolation` / `ReasonUnknown`); `ClassifyApplyError` maps a failed `Apply` to the reason the next launch reports, preferring Windows errno classification (`applyerr_windows.go`) over the portable `fs.ErrPermission` fallback (`applyerr_other.go`). |
| `await.go` | `AwaitPIDEnv` (`PICFETCH_UPDATE_AWAIT_PID`) relaunch handshake. `CleanupPredecessor`, called from `main.go` before `app.NewWithID`, waits (bounded, 15s) for the process that installed this executable to exit before preferences are touched, then sweeps `<dest>.new` / `<dest>.apply.cmd` left by pre-2026-08-30 updates. `SweepBackup`, called from `internal/ui` startup once the Fyne app cache exists, removes `<dest>.old` — skipped when the last recorded apply failure has `Op == "restore"`, the one state where the backup is the user's only intact executable. |

### `internal/preferences`

Standing UI preferences via Fyne `Preferences` (not the session cache).
`SortMode` is a string on disk (`filesort.FromPref` / `Mode.PrefValue`).
Secondary-window geometry is `WindowGeometry` structs.

| File | Responsibility |
|------|----------------|
| `preferences.go` | `Save`, `Load`, `SaveLastUpdateCheckDay`, `State`, `WindowGeometry`; persists normalized mosaic visual settings and secondary-window geometry, never mosaic sources or display IDs. |

### `internal/appearance`

Application-wide System/Light/Dark mode, independent of the viewer. Forced
modes wrap Fyne's current theme and override only its color variant; returning
to System restores the underlying theme so Fyne's OS appearance watcher keeps
switching automatically.

| File | Responsibility |
|------|----------------|
| `appearance.go` | `Mode`, translated picker labels, stable preference values, and `Apply`. |

### `internal/distribution`

Compile-time distribution policy. `StoreManaged` is false for ordinary builds
and true only with the `microsoftstore` build tag. `internal/ui` snapshots it
when constructing a viewer so Microsoft Store packages cannot use the GitHub
self-updater while all other distributions retain the existing behavior.

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

OS integrations (`clipboard`, `displays`, `filemanager`, `filepicker`, `trash`,
`wallpaper`) use
dispatcher vars and build-tagged platform files; tests stub them via
`internal/uitest` — see `AGENTS.md`.

### `internal/wincom`

COM declarations and result handling shared by native Windows adapters.
`desktopwallpaper_windows.go` owns the `IDesktopWallpaper` GUIDs, vtable layout,
and activation constants. Portable `monitor.go` owns the HRESULT failure
predicate and `MonitorAttached`, which distinguishes an attached monitor
(`S_OK`) from a retained, detached entry (`S_FALSE`). `internal/displays` and
`internal/wallpaper` retain their native calls, COM lifetime, feature-specific
behavior, and error reporting.

### `internal/displays`

Native attached-display inspection. `Inspect(fyne.Window)` returns an ordered
snapshot of opaque platform IDs, user-facing names, native-pixel bounds, and
the display containing the greatest part of the PicFetch window. macOS uses
`NSScreen`/CoreGraphics IDs, Windows uses `IDesktopWallpaper` monitor device
paths while skipping retained detached entries, Linux uses XRandR under X11 and
explicitly reports Wayland unsupported. The shared snapshot builder supplies
localized fallback names. Callers compare IDs but never parse them.

### `internal/mosaic`

Viewer-independent mosaic generation. Its small public contract snapshots and
validates source URIs, native-pixel target size, deterministic seed, and visual
settings, then returns immutable rendered pixels. Layout, lazy canonical image
loading, coverage, frame/shadow geometry, and rendering stay inside the
package. Repeated sources use a generation-local 64 MiB byte cache; oversized
sources render without being retained. Rotated masks include a filter margin
before clipping so canvas boundaries do not fade. The package has no Grid,
display-enumeration, picker, wallpaper, preference, or widget behavior.

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

Set desktop wallpaper through `Set(Request)`, where a zero target retains the
legacy global/all-screen action and an opaque display ID requests one screen.
UI writes a persistent PNG into its cache before calling this. A typed
`TargetUnsupportedError` distinguishes an honest platform limitation from a
native execution failure.

| File | Responsibility |
|------|----------------|
| `wallpaper.go` | `Request` (including `Solo`), `Set` dispatcher, typed limitation, legacy Linux/GNOME/KDE and Windows PowerShell global paths. Linux rejects a target before lookup or mutation unless it is the only attached display. GNOME writes `picture-uri` *and* the 42+ `picture-uri-dark` — the only key on screen in dark mode — skipping the dark key only when the schema genuinely lacks it and otherwise reporting its failure. `hostSchemaEnv` scrubs `GSETTINGS_SCHEMA_DIR` / `XDG_DATA_HOME` / `XDG_DATA_DIRS` so a snap-wrapped launcher cannot redirect a host desktop write at its own stale bundled schemas. |
| `darwin.go` / `other.go` | AppKit all-screen or exact preflighted `NSScreen` set / non-Darwin stub. |
| `target_windows.go` | Targeted `IDesktopWallpaper`: COM-thread lifetime, exact device-path validation, and single-monitor set. |
| `windows.go` / `notwindows.go` | `hideConsoleWindow` pair plus non-Windows target stub. |

### `internal/filemanager`

Show one file selected in the current OS's own file manager, through
`Reveal(path)`. macOS runs `open -R` - a LaunchServices binary rather than an
Apple Event, so unlike this repo's other macOS integrations it needs no cgo to
avoid an Automation prompt. Windows runs `explorer.exe /select,"<path>"` with
the command line built by hand, because os/exec's escaping would quote the
`/select,` prefix along with the path; explorer's own non-zero exit is
discarded, which is why the path is stat'ed first. Linux calls
`org.freedesktop.FileManager1.ShowItems` over `dbus-send --print-reply` (the
only portable way to get the file *selected*) and falls back to `xdg-open` on
its parent directory.

| File | Responsibility |
|------|----------------|
| `filemanager.go` | `Reveal` dispatcher, the three platform paths, `fileURI`, and the portable `explorerCmdLine`. |
| `windows.go` / `notwindows.go` | `applyExplorerCommandLine` pair: `SysProcAttr.CmdLine` on Windows, no-op elsewhere. |

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

### `internal/launch`

Command-line flag parsing into the `Options` value `ui.Run` applies at startup.
Hand-rolled rather than `flag`, so flags may appear anywhere among the paths;
rejects an unknown flag, ignores macOS's `-psn_*`, and validates `--sort`
against the `preferences.SortBy*` vocabulary. No Fyne import.

| File | Responsibility |
|------|----------------|
| `launch.go` | `Options`, `Parse`, `Usage`, `ErrHelp`; the `flagSpecs` table every flag is declared in. |

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
through a `FileSet` interface with string keys, not `fyne.URI` —
`Snapshot() Snapshot` is `FileSet`'s only method, handing back an
immutable {keys, generation, key→index} view rather than answering a live
count and key lookup — and its only project import is `internal/imaging`.
Owns dHashes and native pixel sizes keyed by file (generation-scoped:
`WipeIfStale` on a fresh drop, `AdoptGeneration` on an incremental
shrink), the Hamming distance threshold, the installed group snapshot
(representative = highest native pixel count, lowest index on a tie), the
hide-duplicates and inspect modes, and the visibility queries (`IsVisible`
/ `NextVisible` / `FirstVisible` / `LastVisible` / `VisibleIndexesExcept`)
plain navigation asks. A caller testing many indices reads the model once
via `Visibility()`, a frozen hide-flag-plus-groups value with its own
`HiddenExtra` / `Visible` / `RepresentativeOf` / `Size`. `internal/ui`
owns the `Model` and implements `FileSet`; `internal/ui/grid` reads and
feeds it (hashing pass, browse, badges) but does not own it.

| File | Responsibility |
|------|----------------|
| `dupes.go` | `Model`, `FileSet`, hash/native facts, generation (`WipeIfStale` / `AdoptGeneration`, read through a `Snapshot`), distance clamp, `OnChange` observers. |
| `groups.go` | `Groups` snapshot, `Compute` / `Install` / `Rebuild`, `GroupSize` / `RepresentativeOf` / `Members`. |
| `snapshot.go` | `Snapshot`: the immutable {keys, generation, key→index} view every `Model` method reads through. |
| `visible.go` | Hide/inspect modes (`SetHideDuplicates`, `BeginInspect` / `ClearInspect` / `InspectMembers`), `IsHiddenExtra`, `Visibility`, and the visibility/navigation queries. |

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
| `uitest.go` | Temp URIs, synthetic images (including animated GIF, EXIF-oriented/GPS JPEG, RAW preview, SVG), `ApproxEqual`. |
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
- "Why is the info overlay's 'Show EXIF data' link missing?" → `info.go` `syncInfoOverlayVisibility` + `internal/ui/infoview` `Card.Sync` / `HasEXIF`.
- "How does the persistent info overlay (I key) work?" → `internal/ui/infoview` + `info.go`.
- "Why is a menu item greyed out (Checked/Disabled)?" → `internal/ui/menus` `Apply` + `menu.go` `menuState` / `syncMenus`.
- "Where is a photo's GPS position read, and where is it shown?" → `internal/imaging/exif.go` `parseGPSIFD` + `exifwin` Location section.
- "How is EXIF orientation handled?" → `internal/imaging/exif.go` + `orientation.go`.
- "How is a camera RAW file shown?" → `internal/imaging/raw.go` + `LoadedImage.Preview` + `load.go` / `info.go`.
- "How does drag-and-drop / folder scanning work?" → `internal/filescan.Images` / `filescan.Siblings` (single-file case) + `drop.go` `handleDrop`.
- "How is an image shown/preloaded/animated once loaded?" → `load.go`.
- "Which keys do what?" → `keys.go` (`handleKeyEvent` / `handleTypedRune`) + `shortcuts.go`.
- "How do I find one file by name in a big drop?" → `internal/ui/grid/search.go` + `keys.go` `handleTypedRune`.
- "How does hide-duplicates work?" → `internal/dupes` (the model: grouping, hide/inspect modes, visibility — `BeginInspect` / `InspectMembers` / `IsHiddenExtra` / `NextVisible` / `Visibility`) + `internal/imaging/dhash.go` (the hash and its Hamming linkage) + `internal/ui/grid/hashengine.go` (the pool-driven pass that fills the model) + `grid/dupes.go` (browse). Escape reopen: `internal/ui` `reopenVariantGrid`.
- "How do I act on several images at once?" → `internal/selection` + `grid/selection.go` `Targets` + `grid/marquee.go` + `batch.go` + `deletion.RequestFiles` / `clipboard.CopyFiles`.
- "How does zoom/pan work?" → `internal/ui/zoom`; keys in `keys.go`; window resize in `load.go` `syncWindowToZoom`.
- "How does an SVG stay sharp when I zoom?" → `internal/imaging/vector.go` `RasterAt` + `svg.go` + `internal/ui/vector.go` + zoom `SetLogicalSize` / `onScaleChanged`.
- "How does rotation work, and how is it saved to disk?" → `internal/ui/display` (frames/rotation state) + `internal/ui/rotate.go` + `internal/ui/save.go` + `internal/imaging/save.go`.
- "How do I write an image out in a different format?" → `internal/ui/export.go` + `filepicker.ChooseSave` + `imaging.Export`.
- "How do I open the current image's folder in Finder/Explorer/my file manager?" -> `internal/ui/reveal.go` + `internal/filemanager` + `shortcuts.go` `wireRevealShortcut` + `internal/ui/infoview` `RevealLink`.
- "How does 'Set as Wallpaper' work?" → `internal/ui/wallpaper.go` + `internal/wallpaper`.
- "How is an image mosaic sourced, generated, previewed, exported, and targeted to a display?" → `internal/ui/mosaic.go` + `internal/ui/mosaicwin` + `internal/mosaic` + `internal/displays` + `internal/ui/wallpaper.go`.
- "How does the slideshow / picture-frame mode work?" → `internal/ui/slideshow` + `slideshow.go`.
- "How does delete work?" → `internal/ui/deletion` + `internal/trash` + `shortcuts.go` / `batch.go` `requestDelete`.
- "How are native file dialogs implemented?" → `internal/filepicker` + `openfiles.go` / `export.go`.
- "How is the last session saved/restored?" → `internal/session` + `session.go` `restoreSession`.
- "How do in-app updates work?" → `internal/update` + `internal/ui/autoupdate` (serialized automatic/manual checks, staging, apply intent, What's-New cache, apply-failure cache) + `internal/ui/autoupdate.go` (`maybeStartUpdateCheck` / `CheckForUpdatesNow` / `PerformUpdate` / `maybeShowWhatsNew` / `maybeShowUpdateFailure`) + `settingswin` (manual dialogs) + `help/whatsnew.go` (the window). Automatic checks are off by default (`preferences.CheckForUpdates`) and stage silently. Apply remains OnStopped: normal shutdown installs without relaunch; explicit Perform update adds a post-apply relaunch. On Windows that relaunch starts the new executable with `PICFETCH_UPDATE_AWAIT_PID` set to the installing process's PID; `update.CleanupPredecessor` (`internal/update/await.go`), called from `main.go` before `app.NewWithID`, waits on that PID before preferences are touched, unsets the variable, and sweeps leftovers from pre-2026-08-30 updates. If `update.Apply` fails, `ClassifyApplyError` (`internal/update/applyerr.go`) records the reason via `autoupdate.SaveApplyFailure`, and `maybeShowUpdateFailure` explains it on the next launch with a button to the releases page — releases are unsigned, so Controlled Folder Access can still deny the write even to `picfetch.exe` itself. GitHub TUF bootstrap expiry: `tufroot.go`.
- "How are GitHub release notes written?" → `todos.md` `## Done` + `scripts/releasenotes` + `make release` + `.github/workflows/release.yml` `body_path`.
- "How are Linux race-test shards measured and assigned?" → `scripts/testshards` + `.github/testshards/internal-ui.tsv` + the measured CI sharding plan.
- "How is a WinGet publish gated after Release?" → `.github/workflows/winget.yml` + `scripts/wingettag` (vX.Y.Z allowlist; `workflow_run` must be `release.yml` on a published tag).
- "How does a macOS Open With reach the viewer?" → `internal/openwith` (queue + Objective-C graft) + `main.go` `openwith.Install` + `internal/ui/openwith.go` + `run.go` `SetOnStarted`.
- "How does the packaged macOS app declare file/folder associations (Open With)?" → `internal/imaging/loader.go` `SupportedExtensions` + `scripts/plistdoctypes` + `Makefile` `package-mac`.
- "How do Favorites work?" → `internal/favstore` + `internal/ui/favorites` + `shortcuts.go` + `viewer.OpenFiles`.
- "How are favorite previews cached on disk?" → `internal/favthumbs` + `internal/ui/favthumbs.go` + `favorites` + `grid` thumb accessors.
- "Where is the File menu / Settings window?" → `menu.go` `buildMainMenu` + `actionmenu.go` + `settingswin` + `viewer.settingsState` / `ApplySettings` + `viewer.closeFiles`.
- "How are preferences (sort order, appearance, merge mode, slideshow interval/shuffle, folder-scan cap, window-size cap, static window size, window size/position, favorite-preview-cache toggle, check-for-updates checkbox) persisted?" → `internal/preferences` + `startup.go` + `features.go` + `windowtrack.go` + `run.go` `currentPreferences`.
- "How does Light/Dark/System appearance work?" → `internal/appearance` + `internal/ui/theme.go` + `settingswin`.
- "How do the Settings and EXIF windows come back where I left them?" → `widgets.Singleton.Remember` / `Geometry` / `StopTracking` + `winpos.Poll` + `preferences.WindowGeometry`.
- "How is the window's on-screen position read back, since Fyne has no getter for it?" → `internal/winpos` + `windowtrack.go` `startWindowPosPolling`.
- "How can dragging the window open something?" → `internal/wingesture` + `gesture.go` + `help.OpenSpiral` / `spiral.ShowForGesture`.
- "How does copy-image-to-clipboard work?" → `internal/clipboard` + `clipboard.go`. Batch file copy: `copyfiles.go` + `batch.go` `copySelection`.
- "How does Copy Selection (image-region copy) work?" → `internal/ui/copyselection` (`Source` / `Encode`, `HandleKey`) + `copyselection.go` (pause, clipboard worker, `yieldCopySelection`) + `menu.go` `yieldingMenuCallbacks` + `shortcuts.go` `yieldingShortcuts` + zoom `Geometry` / `HandleScroll` + overlay order in `build.go`.
- "How does two-image comparison open, preserve raster/vector fidelity, render through stable overview/detail shaders, identify/swap sides, switch side-by-side/swipe, toggle photo editing with physical Ctrl+L, route camera/photo/divider input, isolate commands, load, settle, and return to the grid?" -> `internal/ui/compare` (`renderer.go` scene/source seam, `shader.go` stable GPU panes and tile lifecycle, `tile.go` reveal-aware planning/cache/generation, plus surface/chrome/reveal clips/layout/two photo transforms/shared camera/hover target/divider/input shield/load/vector lifecycles/UI queue) + `internal/ui/compare.go` (selection, focus release, canonical loader, title, active-mode fact, and open refusal) + `features.go` / `keys.go` / `shortcuts.go` (physical Ctrl+D, toggle hook, transform-key, divider-key, and command routing) + `favorites` / `menus` / `menu.go` (disabled state and guarded entries) + open paths in `drop.go` / `openfiles.go` / `openwith.go` + overlay order in `build.go`.
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
