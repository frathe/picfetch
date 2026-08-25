# Trim ARCHITECTURE.md to a Navigation Map

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** After every task, the parent agent reviews the diff and fixes it before dispatching the next task. Do not start Task N+1 until that review lands. Do not commit.

**Goal:** Turn `ARCHITECTURE.md` into the leanest package map and “where to look for X” index that still lets an agent open the right file for a task.

**Architecture:** Keep the heading skeleton (package map → feature packages → supporting packages → translations → error handling → where to look → keeping current). Cells are locators (path + `rg`-able symbol + load-bearing order), not specs. Standing rules stay in `AGENTS.md` (one pointer, no paraphrase). Stale map entries (`windowmenu*.go` missing, grid `Host` still “9 methods”, `grid/nav.go` unnamed, `NewImgCache` dropped from `loader.go`) are corrected in the same rewrite.

**Tech Stack:** Markdown. `AGENTS.md` gains two concurrency sentences (named wait helpers vs `drain`). No other Go or doc changes. Verification is coverage, verbatim index keys, source-comment anchors, and per-cell lean-ness — not `go test` and not a byte/line cap.

**Lean-ness review:** [Opus](f25ef88e-b9c0-4eca-a3c4-5b3a6b51a8a9) reviewed this plan against “as lean as possible, still an agent index.” Replacement text below already applies that review.

## Status of `todos.md` (this todo is old)

Checked 2026-08-25 against the tree. **Do not implement the first three TODO bullets.**

| TODO | Still open? | Why this plan skips it |
|------|-------------|------------------------|
| `internal/ui/toast.go` import grouping | Yes. Two import groups; neighbours use three. | Bullet says fix it the next time that file’s imports change anyway. |
| `finishLoad` (`internal/ui/load.go` ~192–308) | Yes. Still one ~116-line pipeline. | Decompose only if it needs to change anyway. |
| `internal/imaging/exif.go` (~734 lines) | Yes. Still one file. | Parse/format split is cosmetic. |
| **`ARCHITECTURE.md` trim** | **Yes, and worse.** Todo said ~66 KB; file is now **~159 KB / 748 lines**. | This plan is only that item. |

If the user instead wants the toast.go one-liner, stop; do not execute this plan.

## Approaches considered

1. **Index only** — drop file tables. Smallest, but agents lose “which file in this package”.
2. **Locator tables (this plan)** — keep tables; one locator sentence per file; cut extraction dates, “why”, and `AGENTS.md` paraphrases.
3. **Split docs** — extra files to drift. Rejected.

## Target shape

```
# PicFetch — Architecture          pointer to AGENTS.md; no writing-contract block here
## Package map
### github.com/frathe/picfetch
### internal/ui                    pointer + composition one-liner + "concurrency invariant" anchor + two tables
### internal/imaging
### internal/{favstore,…,uitest}
## Translations                    map facts only (who embeds, who tests, filesort.Label)
## Error handling                  which files may LogError; rules in AGENTS.md
## Where to look for X             39 verbatim question keys; answers are paths
## Keeping this doc current        two lines (the writing contract lives here, once)
```

No byte or line target. Today: 163040 bytes / 748 lines. Smaller is better only when the delete test still passes (see Global Constraints).

## File structure

- Modify: `ARCHITECTURE.md` (Tasks 1–8)
- Modify: `AGENTS.md` (Task 6 only — two concurrency sentences)
- Modify: `todos.md` (Task 8 only — move this bullet to Done)
- Do **not** modify `CONTRIBUTING.md` or README (they already call this file a package map / where-to-look index)
- Do **not** modify `.go` files. Source comments that cite “ARCHITECTURE.md's concurrency invariant” and “internal/ui composes” must still resolve in the rewritten doc.

## Global Constraints

- **Delete test.** Remove the sentence. If an agent would still open the same file, the sentence fails.
- **Locator, not spec.** A cell may name a path, a symbol you can `rg` for, and an ordering. It may not narrate runtime behavior. Bad: “`probeGIF` walks blocks *before* `gif.DecodeAll`”. Good: “`gif.go` — animated GIF compositing, `probeGIF`”.
- **No “why”.** Clauses with *because*, *rather than*, *deliberately*, *on purpose*, *not a controller*, or a parenthetical justification fail. Reasons stay in the code comment next to the code.
- **`AGENTS.md` is single-source.** If a sentence restates an `AGENTS.md` bullet, it fails even when it also cites `AGENTS.md`. One pointer at the top of a section, never a paraphrase plus a pointer.
- **No enumeration you could regenerate.** Full `Host` method lists and “nine `viewer` signals” fail (they go stale; the plan is itself fixing a stale “9-method Host”). Method *counts* and one surprising member (grid `Modifiers`) pass. `rg 'type Host interface' -A 30` is how you list methods.
- **Say it once.** A fact in a package intro, a table row, *and* a Where-to-look answer fails twice. The index is canonical for “where”; tables defer to it.
- **Question keys are verbatim.** Compress answers to paths + symbols. Do not delete, rephrase, or reorder the 39 quoted questions.
- **Every source cross-reference resolves.** `rotate.go` and `vector_test.go` cite “ARCHITECTURE.md's concurrency invariant”. `menu.go` and `help.go` cite that `internal/ui` composes features. `filesort.go` cites the preferences-string knot (`FromPref`). Keep those anchors as the words those comments search for.
- Do not delete a package, a production `internal/ui/*.go` file, or a Where-to-look question. Add `windowmenu.go` / `windowmenu_darwin.go` / `windowmenu_notdarwin.go` and name `grid/nav.go`.
- Do not invent packages, files, Host methods, or invariants absent from the tree or this plan’s replacement text.
- Cut: “Extracted from … on YYYY-MM-DD”, “Added YYYY-MM-DD”, “stage N”.
- Keep findable (here or via one `AGENTS.md` pointer): feature construction order; overlay stack order; grid `UIQueue`; `ByteCache.Add` vs `AddIfFits`; `appState` not passed to features; cross-feature composition in `internal/ui`.
- Sequential only: every task edits `ARCHITECTURE.md`. Never dispatch two implementers in parallel.
- Do **not** `git commit`. Do **not** run the full test suite for this markdown change.
- After the parent reviews a task, the next implementer reads the *current* `ARCHITECTURE.md`, not an older plan snapshot.
- Replacement text in each task is verbatim **after** the Opus lean-ness cuts. Do not re-expand it.

### Subagent models

| Role | Model | When |
|------|-------|------|
| Implementer, Tasks 1, 6, 8 | `composer-2.5-fast` | Short verbatim paste / checks / `todos.md` |
| Implementer, Tasks 4, 5 | `cursor-grok-4.5-high-fast` | Long verbatim paste with a short cut list |
| Implementer, Tasks 2, 3, 7 | `claude-opus-5-thinking-high` | Highest miss-the-file risk (36 ui rows + Host counts + 39 index keys) |
| Task reviewer (Tasks 1, 4–6, 8) | `cursor-grok-4.5-high-fast` | Spec + quality on that section |
| Task reviewer (Tasks 2, 3, 7) | `claude-opus-5-thinking-high` | Same judgment bar as the implementer |
| Parent controller | session model | Review + fix after every task |
| Final whole-doc pass | `claude-opus-5-thinking-high` | Coverage, anchors, Host counts, leftover commentary |

If a section rewrite is BLOCKED or drops required facts, re-dispatch that implementer on `claude-opus-5-thinking-high`.

---

### Task 1: Intro and package main

**Files:**
- Modify: `ARCHITECTURE.md` from the title through package main (stop at `### internal/ui`)

**Interfaces:**
- Consumes: nothing
- Produces: title + `AGENTS.md` pointer + package main blurb. Writing contract is **not** here; it lands once in Task 8.

**Model:** `composer-2.5-fast`

- [ ] **Step 1: Replace the opening through package main** with this text (verbatim):

```markdown
# PicFetch — Architecture

Navigation map for AI agents. PicFetch is a Fyne desktop image viewer: one
binary, split into `internal/...` packages. Start here to find a file.
Standing rules (data flow, concurrency, conventions, build) live in
`AGENTS.md` — do not duplicate them here.

## Package map

### `github.com/frathe/picfetch` (package main)

Entry point only. `main.go` builds the `fyne.App`, loads embedded
`translations/*.json`, converts CLI paths to URIs (`argsToURIs`), and calls
`ui.Run`.
```

- [ ] **Step 2: Leave everything after `### \`internal/ui\`` untouched.**

- [ ] **Step 3: Confirm the rest of the file is still present.**

```bash
rg -n '^### `internal/ui`|^## Where to look for X|^## Keeping this doc current' ARCHITECTURE.md
```

Expected: all three headings still match.

---

### Task 2: `internal/ui` overview + own-files table

**Files:**
- Modify: `ARCHITECTURE.md` from `### \`internal/ui\`` through the line before `#### Feature packages`

**Interfaces:**
- Consumes: Task 1 pointer
- Produces: composition one-liner; **“concurrency invariant”** wording (source comments grep this); complete own-files table including `windowmenu*`

**Model:** `claude-opus-5-thinking-high`

Replace that span with the following (verbatim). Keep `#### Feature packages` and everything after it.

- [ ] **Step 1: Write the replacement**

```markdown
### `internal/ui`

The application. Unexported `appState` is the file-set model (scan/drop
order, displayed order, index, sort, merge). Unexported `viewer` is the
Fyne façade. Construction order, overlay order, data flow, and concurrency:
see `AGENTS.md`. Features expose state; `internal/ui` composes them.

Concurrency invariant: see `AGENTS.md` § Concurrency and Fyne.

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
| `keys.go` | `handleKeyEvent` / `handleTypedRune`. Return immediately while `Canvas().Overlays().Top()` is set (Fyne dialogs/menus). |
| `menu.go` | Menu bar composition: File, Favorites, Actions, Window, Help. Grid/slideshow mutual exclusion lives here, not in those packages. |
| `actionmenu.go` | Actions-menu Checked/Disabled and handlers (`applyActionsMenuState`). |
| `drop.go` | `handleDrop` / `applyScanResult` / `applyScannedFiles` glue over `filescan.Images`; scan lifecycle is `viewer.scanOp`. |
| `memlimits.go` | `settings` value plus memory-limit get/set that retune `imgCache`, grid thumb cache, `imaging.SetMaxEncodedBytes`, and the SVG raster cap. |
| `favthumbs.go` | Viewer glue for `favthumbs.Sync`, `gridSink`, and the favorite-preview `completion.Signal`. |
| `load.go` | `ShowImage` / `attemptLoad` / `finishLoad`, neighbor preload (`AddIfFits`), GIF `animate`, `resizeToImage` / `syncWindowToZoom`. |
| `toast.go` | Self-dismissing notification card and `showToast`. |
| `info.go` | Persistent info overlay (I); EXIF link; RAW `(preview)` mark; `displayedDimensions`. |
| `asyncop.go` | `asyncOpUI` (lifecycle, active, done, spinner) — used only by scan and sort. |
| `sort.go` | `toggleSort` / `SetSortMode` / `startSort` / `finishSort` over `filesort.Order`; lifecycle is `viewer.sortOp`. |
| `rotate.go` | View-only 90° rotation. Call `updateFileMenuState` *before* `applyRotationLayout`. |
| `vector.go` | Debounced SVG re-render. |
| `save.go` | File > Save Changes (`canSaveRotation` / `saveRotation` / `updateFileMenuState`). |
| `export.go` | File > Export image (`promptExport` / `exportAs`) via `widgets.ChoiceCard` + `filepicker.ChooseSave`. |
| `wallpaper.go` | Set as Wallpaper: write a PNG into `viewer.wallpaperDir`, then `wallpaper.Set`. |
| `slideshow.go` | `togglePictureFrameMode` (closes grid first) plus shuffle/interval bindings. |
| `batch.go` | Only file that knows both grid selection and deletion/clipboard exist. |
| `session.go` | `restoreSession` glue over `internal/session`. |
| `clipboard.go` | Copy-path / copy-image glue over `internal/clipboard`. |
| `openfiles.go` | Native open-dialog glue over `internal/filepicker`. |
```

- [ ] **Step 2: Confirm `windowmenu.go` is a table row**, the phrase `Concurrency invariant` appears, and `#### Feature packages` still follows.

```bash
rg -n 'windowmenu\.go|Concurrency invariant|#### Feature packages' ARCHITECTURE.md
```

---

### Task 3: Feature packages table + composition note

**Files:**
- Modify: `ARCHITECTURE.md` from `#### Feature packages` through the paragraph(s) immediately before `### \`internal/imaging\``

**Interfaces:**
- Consumes: Task 2 overview (already has the compose / concurrency-invariant sentences)
- Produces: compact feature table with **counts** not full Host lists; `grid/nav.go` named

**Model:** `claude-opus-5-thinking-high`

- [ ] **Step 1: Replace the feature-packages table and the essays between it and `### internal/imaging`** with:

```markdown
#### Feature packages (`internal/ui/...`)

| Package | Responsibility | Reaches back via |
|---------|----------------|------------------|
| `zoom/` | Zoom/pan of the displayed image. Window growth is `syncWindowToZoom` in `internal/ui`. | `onChanged`, `modifiers`, `onScaleChanged`. |
| `grid/` | Full-window thumbnail overview (G): virtualized `GridWrap`, own byte-budget thumb cache, `decodepool`, `uiqueue.go`, `/` search, hide-duplicates (D), multi-select, `Targets()`. `nav.go` is the highlight funnel (`setHighlight` → `HighlightChanged`). | 10-method `Host` including `Modifiers`. Knows nothing about slideshow, deletion, or clipboard. |
| `deletion/` | Shift+Delete confirm (`widgets.ChoiceCard`) then `trash.Move`. `RequestFiles` is the batch path; `Request` is the one-file wrapper. | 7-method `Host`. |
| `slideshow/` | Picture-frame mode (P): full-screen, auto-advance, interval, `winpos.Tracker` capture/restore. | 2-method `Host`. Knows nothing about the grid. |
| `exifwin/` | EXIF panel (E): tag list, optional JPEG strip, GPS map (`tiles.go`, `startWarm`). Geometry via `widgets.Singleton`. | 4-method `Host`. |
| `help/` | Manual, About, Help menu; embeds `manual.md` / `manual_de.md`. Secret search phrase and window-spiral both open `spiral/`. | Nothing — `New(app, title, art)` only. |
| `spiral/` | Full-screen shader easter egg. | `New(app)` only. |
| `settingswin/` | Settings window: form + checks, live `Host` setters, geometry via `Singleton`. | Getter/setter `Host` (sort, merge, slideshow, caps, caches, duplicate distance). |
| `favorites/` | Favorites menu and add/overwrite/manage/remove dialogs. `New` does no disk I/O; `SetDir` from `Run`. | 5-method `Host`. |
| `widgets/` | Shared UI mechanics: `ChoicePanel` / `ChoiceCard`, `TappableArea`, `Singleton` (+ geometry memory), `NewSizeTracker`, focus-ring style. | Leaf aside from `internal/winpos`. |
| `assets/` | `WelcomeWebP` / `PlaceholderWebP`. | Leaf. |
```

Leave `### internal/imaging` in place. Do not re-add the concurrency essay — Task 2 already holds the anchor.

- [ ] **Step 2: Confirm `nav.go` and `10-method` appear**, and `### internal/imaging` still follows.

---

### Task 4: `internal/imaging`

**Files:**
- Modify: `ARCHITECTURE.md` from `### \`internal/imaging\`` through the line before `### \`internal/favstore\``

**Interfaces:**
- Consumes: writing contract
- Produces: short imaging blurb + locator file table; `NewImgCache` on the `loader.go` row; `Add` vs `AddIfFits` as a file mapping

**Model:** `cursor-grok-4.5-high-fast`

- [ ] **Step 1: Replace the imaging section** with:

```markdown
### `internal/imaging`

Viewer-independent probe → decode → EXIF-orient → cache pipeline (JPEG, PNG,
GIF including animated, WebP, BMP, TIFF, ICO, XPM, HEIC, AVIF, SVG, camera
RAW via embedded JPEG). RAW is preview-only (`LoadedImage.Preview`);
`CanEncode` is false. SVG is the only vector format (`svg.go` / `vector.go`).
Encode/write-back for a subset of formats lives in `save.go`.

| File | Responsibility |
|------|----------------|
| `bytecache.go` | `ByteCache[V]`: goroutine-safe LRU by estimated bytes. `Add` (displayed image) vs `AddIfFits` (speculative preload). |
| `loader.go` | `LoadedImage`, `NewImgCache`, `ReadAndProbe`, `DecodeLoaded`, `LoadImage`, `IsSupportedImage`, `MaxEncodedBytes` / `InputTooLargeError`. |
| `raw.go` | Largest embedded JPEG from TIFF IFDs or SOI scan (CR3/RAF). |
| `svg.go` | SVG detection, logical-size floor (`MinVectorWidth`/`Height` = UI `startW`/`startH`), `ClampVectorRaster` / `MaxVectorRasterPixels`. |
| `vector.go` | `Vector` / `ParseVector` / `RasterAt`. |
| `exif.go` | Orientation tags + `ReadMetadata` / `Metadata` (including GPS IFD). JPEG APP1, then TIFF IFD0, then HEIC/AVIF, then RAW preview APP1. |
| `orientation.go` | `ApplyOrientation`, `RotateSteps`. |
| `gif.go` | Animated GIF compositing, `probeGIF`. |
| `thumbnail.go` | `LoadThumbnail` / `NewThumbCache`: same probe+decode, then downsample. |
| `dhash.go` | `DifferenceHash` / `Hamming` / `DuplicateGroups` for grid hide-duplicates. |
| `jpegexif.go` | Unexported JPEG segment copy/strip for `save.go`. |
| `save.go` | `SaveRotated`, `Export`, `CanEncode` / `CanEncodeExt`, `StripJPEGMetadata`. |
```

- [ ] **Step 2: Delete any surviving “Extracted from `library.go` …” line.** Confirm `NewImgCache` is on the `loader.go` row.

---

### Task 5: Remaining packages (`favstore` through `uitest`)

**Files:**
- Modify: `ARCHITECTURE.md` from `### \`internal/favstore\`` through the end of `### \`internal/uitest\`` (stop before `## Translations`)

**Interfaces:**
- Consumes: writing contract
- Produces: one short blurb + locator file table per remaining package; `FromPref` kept (filesort.go cites it); completion Wait/Begun rule **not** duplicated here (Task 6 moves it to `AGENTS.md`)

**Model:** `cursor-grok-4.5-high-fast`

Replace that entire span with the following (verbatim). Keep `## Translations` and after.

- [ ] **Step 1: Write the replacement**

```markdown
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

### `internal/preferences`

Standing UI preferences via Fyne `Preferences` (not the session cache).
`SortMode` is a string on disk (`filesort.FromPref` / `Mode.PrefValue`).
Secondary-window geometry is `WindowGeometry` structs.

| File | Responsibility |
|------|----------------|
| `preferences.go` | `Save`, `Load`, `State`, `WindowGeometry`. |

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

### `internal/filescan`

Recursive image gather for drop/open.

| File | Responsibility |
|------|----------------|
| `filescan.go` | `Images(ctx, uris, max, progress)`; symlink-cycle + per-call dedupe. |

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
```

- [ ] **Step 2: Confirm `## Translations` still follows** and `FromPref` is still in the filesort/preferences blurbs.

```bash
rg -n 'FromPref|^## Translations' ARCHITECTURE.md
```

---

### Task 6: `AGENTS.md` wait-helper rule + Translations + error handling

**Files:**
- Modify: `AGENTS.md` § Concurrency and Fyne (append two sentences)
- Modify: `ARCHITECTURE.md` from `## Translations` through the line before `## Where to look for X`

**Interfaces:**
- Consumes: Task 5 completion section already points here
- Produces: standing wait-helper rule in `AGENTS.md`; map-only translations/error sections

**Model:** `composer-2.5-fast`

- [ ] **Step 1: Append this bullet** to `AGENTS.md` under `## Concurrency and Fyne` (after the existing `drain` / `fyne.Do` bullets):

```markdown
- `completion.Signal.Wait` on a never-begun signal returns immediately — `drain` and low-level `waitFor` rely on that. Named wait helpers (`waitUntilLoaded`, `waitForScan`, `waitForSort`, `waitForAnimStopped`, `waitForClipboard`) fatal when `!Begun()`.
```

Do not restate other concurrency bullets.

- [ ] **Step 2: Replace the Translations and Error handling sections** in `ARCHITECTURE.md` with:

```markdown
## Translations

`main.go` owns `//go:embed translations/*.json` and `lang.AddTranslationsFS`.
`filesort.Label` is the one non-widget `lang.L` call site. `main_test.go`
checks locale parity and that `en.json` is an identity map. String rule:
see `AGENTS.md`.

## Error handling

`fyne.LogError` is used in `main.go`, `internal/ui` glue, and
`internal/session`. Viewer-independent packages return errors. Ignore rule:
see `AGENTS.md`.
```

- [ ] **Step 3: Confirm `## Where to look for X` still follows.**

---

### Task 7: Where to look for X

**Files:**
- Modify: `ARCHITECTURE.md` from `## Where to look for X` through the line before `## Keeping this doc current`

**Interfaces:**
- Consumes: compressed package map from Tasks 2–6
- Produces: the same 39 question keys as `HEAD`, answers are paths (include `grid/nav.go` on the grid / title questions)

**Model:** `claude-opus-5-thinking-high`

Keep every quoted question **verbatim** (byte-identical to `git show HEAD:ARCHITECTURE.md` keys). Replace only the answers.

- [ ] **Step 0: Snapshot keys before editing**

```bash
git show HEAD:ARCHITECTURE.md | rg '^- "[^"]+"' -o > /tmp/arch-keys-before.txt
```

- [ ] **Step 1: Replace the section** with:

```markdown
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
- "How does drag-and-drop / folder scanning work?" → `internal/filescan.Images` + `drop.go` `handleDrop`.
- "How is an image shown/preloaded/animated once loaded?" → `load.go`.
- "Which keys do what?" → `keys.go` (`handleKeyEvent` / `handleTypedRune`) + `shortcuts.go`.
- "How do I find one file by name in a big drop?" → `internal/ui/grid/search.go` + `keys.go` `handleTypedRune`.
- "How does hide-duplicates work?" → `internal/imaging/dhash.go` + `internal/ui/grid/dupes.go`.
- "How do I act on several images at once?" → `internal/selection` + `grid/selection.go` `Targets` + `batch.go` + `deletion.RequestFiles` / `clipboard.CopyFiles`.
- "How does zoom/pan work?" → `internal/ui/zoom`; keys in `keys.go`; window resize in `load.go` `syncWindowToZoom`.
- "How does an SVG stay sharp when I zoom?" → `internal/imaging/vector.go` `RasterAt` + `svg.go` + `internal/ui/vector.go` + zoom `SetLogicalSize` / `onScaleChanged`.
- "How does rotation work, and how is it saved to disk?" → `internal/ui/rotate.go` + `internal/ui/save.go` + `internal/imaging/save.go`.
- "How do I write an image out in a different format?" → `internal/ui/export.go` + `filepicker.ChooseSave` + `imaging.Export`.
- "How does 'Set as Wallpaper' work?" → `internal/ui/wallpaper.go` + `internal/wallpaper`.
- "How does the slideshow / picture-frame mode work?" → `internal/ui/slideshow` + `slideshow.go`.
- "How does delete work?" → `internal/ui/deletion` + `internal/trash` + `shortcuts.go` / `batch.go` `requestDelete`.
- "How are native file dialogs implemented?" → `internal/filepicker` + `openfiles.go` / `export.go`.
- "How is the last session saved/restored?" → `internal/session` + `session.go` `restoreSession`.
- "How do Favorites work?" → `internal/favstore` + `internal/ui/favorites` + `shortcuts.go` + `viewer.OpenFiles`.
- "How are favorite previews cached on disk?" → `internal/favthumbs` + `internal/ui/favthumbs.go` + `favorites` + `grid` thumb accessors.
- "Where is the File menu / Settings window?" → `menu.go` `buildMainMenu` + `actionmenu.go` + `settingswin` + `viewer.closeFiles`.
- "How are preferences (sort order, merge mode, slideshow interval/shuffle, folder-scan cap, window-size cap, window size/position, favorite-preview-cache toggle) persisted?" → `internal/preferences` + `startup.go` + `features.go` + `windowtrack.go` + `run.go` `currentPreferences`.
- "How do the Settings and EXIF windows come back where I left them?" → `widgets.Singleton.Remember` / `Geometry` / `StopTracking` + `winpos.Poll` + `preferences.WindowGeometry`.
- "How is the window's on-screen position read back, since Fyne has no getter for it?" → `internal/winpos` + `windowtrack.go` `startWindowPosPolling`.
- "How can dragging the window open something?" → `internal/wingesture` + `gesture.go` + `help.OpenSpiral` / `spiral.ShowForGesture`.
- "How does copy-image-to-clipboard work?" → `internal/clipboard` + `clipboard.go`. Batch file copy: `copyfiles.go` + `batch.go` `copySelection`.
- "How does the grid overview / thumbnail generation work?" → `imaging/thumbnail.go` + `grid/grid.go` + `grid/thumbs.go` + `grid/nav.go` + `grid/uiqueue.go`.
- "What decides the window title?" → `viewer.go` `setTitle` / `applyTitle` / `HighlightChanged` + `load.go` + `grid/nav.go`.
- "How do I write a test that needs an image / a viewer?" → `internal/uitest` + `newTestViewer` / `newTestUI` + `dropAndWait` in `harness_test.go`.
- "How do I add or translate a user-visible string?" → `lang.L` at the call site and the same key in every `translations/` bundle. See `AGENTS.md`.
- "Why isn't feature X its own package?" → `AGENTS.md` (features expose state; `internal/ui` composes them).
- "How/where are errors reported, and when is it OK to ignore one?" → `AGENTS.md`.
```

- [ ] **Step 2: Diff keys against the snapshot**

```bash
rg '^- "[^"]+"' -o ARCHITECTURE.md > /tmp/arch-keys-after.txt
diff /tmp/arch-keys-before.txt /tmp/arch-keys-after.txt && echo "index keys intact"
```

Expected: no diff.

---

### Task 8: Keeping-current note, verification, `todos.md`

**Files:**
- Modify: `ARCHITECTURE.md` `## Keeping this doc current`
- Modify: `todos.md` (move the ARCHITECTURE bullet to Done)

**Interfaces:**
- Consumes: finished map from Tasks 1–7
- Produces: writing contract in one place; coverage/anchor/lean-ness checks pass; todo list updated

**Model:** `composer-2.5-fast`

- [ ] **Step 1: Replace `## Keeping this doc current`** with:

```markdown
## Keeping this doc current

Update this file in the same change when a package is added, removed,
renamed, or files move between packages. Cells are locators (path, symbol,
order) — one sentence. Standing rules go in `AGENTS.md`, not here.
```

- [ ] **Step 2: Coverage, anchors, Host counts, lean-ness** (no `wc` budget)

```bash
# Every internal package with a production .go file appears in the map.
for d in $(find internal -type d); do
  ls "$d"/*.go 2>/dev/null | grep -qv _test || continue
  rg -q "\b${d//\//\\/}\b" ARCHITECTURE.md || echo "MISSING PACKAGE: $d"
done

# Every production internal/ui/*.go is named in the doc.
for f in internal/ui/*.go; do
  case "$f" in *_test.go) continue;; esac
  rg -qF "\`$(basename "$f")\`" ARCHITECTURE.md || echo "MISSING FILE: $f"
done

# Source comments that cite this file still resolve.
rg -n 'ARCHITECTURE\.md' --glob '*.go' .
rg -q 'concurrency invariant' ARCHITECTURE.md || echo "MISSING ANCHOR: concurrency invariant"
rg -q 'composes them' ARCHITECTURE.md || echo "MISSING ANCHOR: composes them"
rg -q 'FromPref' ARCHITECTURE.md || echo "MISSING ANCHOR: FromPref"
rg -qF 'nav.go' ARCHITECTURE.md || echo "MISSING FILE: grid/nav.go"
rg -qF 'NewImgCache' ARCHITECTURE.md || echo "MISSING SYMBOL: NewImgCache"

# Host counts match source.
for p in grid deletion favorites exifwin slideshow; do
  f=$(rg -l 'type Host interface' internal/ui/$p)
  n=$(sed -n '/^type Host interface/,/^}/p' "$f" | rg -c '^\t[A-Z][A-Za-z]*\(')
  rg -q "${n}-method" ARCHITECTURE.md || echo "$p: source says $n methods; doc does not say ${n}-method"
done

# Over-long table cells (locator, not essay).
awk '/^\|/ && length($0) > 240 { print FNR": "length($0)" chars" }' ARCHITECTURE.md

# Commentary sniff — expected: few or none; each hit needs a yes/no.
rg -n -i 'deliberately|on purpose|rather than|because|why we|Extracted from|Added 20|stage [0-9]|not a controller' ARCHITECTURE.md || true

# AGENTS.md rules must not be restated here (pointer-only is fine).
rg -n 'appID|lang\.L\("English|fyne\.LogError|gen2brain|make golden|go test -race|never touch the real desktop' ARCHITECTURE.md || true
```

Pass: no `MISSING`, Host counts match, `settingswin` is the getter/setter Host (no method count in source the same way — skip if the loop fails to find a count), no table row over 240 characters unless it is a file-list cell that only names files. Commentary/duplication hits: remove or justify in the handoff.

`settingswin`’s `Host` is getter/setter pairs, not a `{n}-method` phrase — the loop may print a mismatch; that is OK if the doc still says “Getter/setter `Host`”.

- [ ] **Step 3: Update `todos.md`**

Under `## Done`, add:

```markdown
- Trimmed `ARCHITECTURE.md` from per-function commentary to a locator
  package map plus the “Where to look for X” index (2026-08-25).
```

Remove the ARCHITECTURE.md bullet from `## TODO`. Leave the toast.go / `finishLoad` / `exif.go` bullets in `## TODO`.

---

## Parent review checklist (after every task)

The controller (not the implementer) does this before Task N+1:

1. Apply the delete test to new sentences. Reject leftover essays, dropped files, dropped Where-to-look keys, or Host counts that do not match source.
2. Confirm cited anchors still exist (`concurrency invariant`, `composes them`, `FromPref`).
3. Fix typos in the same session. Re-dispatch on `claude-opus-5-thinking-high` only if the section is wrong in substance.

## Final pass (after Task 8)

Dispatch a reviewer on `claude-opus-5-thinking-high` with: current `ARCHITECTURE.md`, this plan’s Global Constraints, and `git diff` against the branch base. It must confirm:

- Every `internal/` package with a production `*.go` file is in the map
- Every production `internal/ui/*.go` file is named (including `windowmenu*`)
- `grid/nav.go` and `NewImgCache` are findable
- Feature `Host` *counts* match source (not full method lists)
- Source comments that cite this file still resolve
- 39 Where-to-look keys match `HEAD`
- No `AGENTS.md` paraphrase beyond a pointer
- Delete test holds

No Go tests required. Suggested commit message (human commits):

```
docs: trim ARCHITECTURE.md back to a navigation map

Cut per-function essays so the file is a package map and where-to-look
index. Move the named-wait vs drain rule into AGENTS.md.
```

## Open points

Locked unless the user overrides before Task 1:

1. **Which TODO?** Skip the three self-deferred bullets; implement this trim.
2. **Shape?** Locator tables (approach 2), not index-only and not split docs.
3. **Size?** No byte/line cap. Lean-ness = delete test + locator cells + no `AGENTS.md` paraphrase. (User, 2026-08-25.)
4. **Historical notes?** Deleted.
5. **Commits?** None from subagents.
6. **Opus?** `claude-opus-5-thinking-high` for Tasks 2, 3, 7, those reviews, BLOCKED retries, and the final pass.
