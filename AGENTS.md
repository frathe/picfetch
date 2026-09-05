# PicFetch Agent Guide

## Start Here

- `.agents/skills/improved_sdd_tdd_cycle.md` is the working agreement for *how* work gets done — routing, delegation limits, review ownership, verification. This file is the conventions; that one is the process. Read it before planning anything larger than a two-file fix.
- `plans/` holds active Standard/Deep SDD implementation plans; `.scratch/` holds issue-tracker specs and tickets. Move an accepted plan to `finished_refactorings/`.
- Read `ARCHITECTURE.md` before code: it is the authoritative package map and “where to look for X” index.
- Update `ARCHITECTURE.md` in the same change when packages are added, removed, renamed, or files move between packages.
- Open work belongs in `todos.md`; do not add `TODO`/`FIXME` comments to source.
- Do not run `git commit`. End with a suggested commit message for the user.

## Architecture and Data Flow

- `main.go` only creates the Fyne app, embeds translations, converts CLI paths, and calls `internal/ui.Run`; keep package `main` thin. Launch flags parse in `internal/launch` (`Options`, `Parse`, `Usage`) and are applied by `viewer.applyLaunchOptions`; a new flag goes in that package's `flagSpecs` table, not into `main.go`. That package's usage and error text is deliberately English rather than `lang.L`: it goes to stderr for a shell or a journal, not to anything the app draws.
- `internal/ui/appState` owns the current/unsorted file lists, index, sort mode, and merge mode. The unexported `viewer` is its Fyne-facing façade; `ui.Run` is the package’s only exported entry point.
- Feature packages such as `internal/ui/grid`, `deletion`, and `slideshow` own their widgets/state and declare narrow consumer-side `Host` interfaces. Do not pass them `appState` or invent a shared controller/registry.
- `internal/ui/menus`, `infoview`, `display`, and `autoupdate` are the exception: they take a value `State` snapshot built in exactly one `internal/ui` function, not a `Host`. `settingswin` is the same exception on the read side (`settingsState()` at Show) plus a 3-method Host (`ApplySettings` and the two update verbs) because live side effects cannot be a pure snapshot. A narrow Host doesn't fit when a package reads state spread across five-plus features — `menus` alone would need ~13 methods — so default to `Host` and reach for a `State` snapshot only when the same case applies.
- Cross-feature decisions stay in `internal/ui`: for example, `batch.go` joins grid selection to deletion/clipboard, and `togglePictureFrameMode` prevents grid/slideshow overlap.
- Feature construction order in `internal/ui/features.go` and overlay order in `build.go` are load-bearing; preserve explicit composition rather than auto-registration.
- Input flows through CLI/open/drop → `handleDrop` scan → `filesort.Order` → `ShowImage` → `internal/imaging` probe/decode/orient/cache → Fyne display. Reuse this path rather than creating parallel open/load logic. On macOS, "Open With"/Dock drop/`open -a`/double-click arrive as an Apple Event instead of argv; `internal/openwith`'s queue + `internal/ui/openwith.go` fold that into the same `handleDrop` call, argv files first.
- `internal/imaging` is viewer-independent. Full images and grid thumbnails use separate byte-budgeted caches; preserve `ByteCache.Add` (displayed image) versus `AddIfFits` (speculative preload) semantics.
- Session file sets use `internal/session`/Fyne cache; standing settings and geometry use `internal/preferences`/Fyne preferences. Startup wiring is in `internal/ui/startup.go`, shutdown persistence in `run.go`.
- OS integrations live behind dispatcher vars in `internal/{clipboard,displays,filemanager,filepicker,trash,wallpaper}` with build-tagged platform files. Tests must replace them via `internal/uitest` stubs, never touch the real desktop.
- Keep `appID` synchronized across `main.go`, `FyneApp.toml`, and `Makefile`; changing it disconnects existing preferences/session data.

## Concurrency and Fyne

- Scan, load, sort, and vector work each own a `requestLifecycle`; capture its token, check staleness before expensive work and before applying results, and marshal background UI updates through `fyne.Do`.
- `internal/ui/grid`, `internal/ui/compare`, and `internal/ui/mosaicwin` are the per-instance `UIQueue` exceptions: their tests drain completions instead of letting the Fyne test driver run them inline on workers. `internal/ui`'s `newTestUI` installs a drainable `uitest.UIQueue` on all three; keep worker completions on `g.ui.Do` / `f.queueUI` / `w.ui.Do`. Every `g.ui.Do` call stays inside its owning `g.decodes.Go` body because grid `Settle` is bounded by `decodes.Wait()`. Compare `Settle` waits its load worker and both pane raster waitgroups, drains queued completions, then repeats because applying a queued load can start fresh vector work. Mosaic `Settle` waits its complete worker set, drains queued completions, and repeats because a drained action can start work.
- Do not add mutable package-level test seams. Runtime/test-configurable values belong on `viewer` or the owning feature.
- Every goroutine needs cancellation/staleness handling plus an observable stop/done signal. If adding background work, add it to `newTestUI`’s `drain` cleanup in `internal/ui/harness_test.go`.
- Fyne’s test driver runs `fyne.Do` inline. Use `dropAndWait`, `waitFor*`, feature `Settle`, and existing completion channels before assertions; never sleep to guess completion. Grid, comparison, and mosaic window work marshal through the `UIQueue` exception above.
- An in-flight counter is not a completion signal for the callback that follows it. `tileFetcher.release` decrements `pending` and *then* calls `onChange`, so a test polling for `Pending() == 0` may assert in between and see a callback that has not run. Wait on the callback's own observable effect — a channel it sends on — never on a counter its caller cleared first.
- `completion.Signal.Wait` on a never-begun signal returns immediately — `drain` and low-level `waitFor` rely on that. Named wait helpers (`waitUntilLoaded`, `waitForScan`, `waitForSort`, `waitForAnimStopped`, `waitForClipboard`) fatal when `!Begun()`.

## Project Conventions

- Every user-visible string is `lang.L("English text")`; add that exact key to every `translations/*.json` bundle. English is an identity map and `main_test.go` enforces locale parity.
- No Unicode arrows in anything the app draws — not in `lang.L` keys or catalogue values, not in the manuals, not even inside backticks. The theme font (NotoSans) has no arrow glyphs, the shaper falls back to a 23-glyph symbol subset with no space, `/` or `-`, and the character *after* the arrow is painted as `�`. Write menu paths and cycles as ASCII `->` and keys as `Left` / `Right` / `Up` / `Down`. Guarded by `TestManualHasNoUnicodeArrows` and `TestTranslationsHaveNoUnicodeArrows`.
- Report UI-boundary failures with `fyne.LogError`; viewer-independent packages return errors. Mark intentionally ignored errors explicitly (`_ =` or `_, _ =`) so IDE/`errcheck` inspections see intent.
- In concrete functions and methods, name intentionally unused parameters `_` (for example, `func f(_ context.Context)`); Qodana's `GoUnusedParameter` inspection flags unnamed required parameters such as `func f(context.Context)`.
- Use `internal/uitest` for synthetic image formats, temp URIs, approximate comparisons, and OS seam stubs. UI tests should build through `newTestUI`/`newTestViewer`, which mirror production startup.
- `CanvasObject.Visible()` is the object's own hidden flag, not a statement about the tree it is in: a widget that was built and then left out of its container still reports `true`. A test that asserts only `Visible()` therefore passes on a widget that never reaches the screen. When the fact under test is "this is *in* the surface", walk the container from its root — see `infoview`'s `inCard` in `card_test.go`. `Visible()` alone is enough only for a widget already known to be in the tree whose show/hide is what moves.
- Keep platform-specific behavior in existing build-tag pairs and preserve no-cgo HEIC/AVIF decoding through `gen2brain` WASM; Fyne itself still requires a C/OpenGL toolchain.

## Build and Verification

- Use the Makefile: `make run`, `make build`, `make test`, `make fmt`, `make vet`; `make test` runs in Linux/amd64 Docker so golden rendering matches CI, while `make test-native` is the explicit host-platform check. `make help` lists packaging/security targets.
- `make package-mac` runs `go run ./scripts/plistdoctypes` to derive the packaged app's `CFBundleTypeExtensions` from `imaging.SupportedExtensions()`; it no longer depends on `python3`.
- Match CI before handoff with `make verify`: formatting/TUF checks, vet and build run from the repository root, and the race suite runs through the same Linux/amd64 Docker path as `make test`.
- **HEIC decoder pin:** `go.mod` replaces `github.com/gen2brain/heic` with a fork commit containing [gen2brain/heic#16](https://github.com/gen2brain/heic/pull/16) (fixes native memory leak [issue #15](https://github.com/gen2brain/heic/issues/15)). Remove the `replace` and bump to an official release once upstream tags a version that includes that fix. Optional manual RSS check: `PICFETCH_HEIC_LEAK_TEST=1 go test -tags=heicleak -run TestHEICDecode_DoesNotGrowRSSUnbounded ./internal/imaging/...`.
- Run focused tests while iterating, e.g. `go test -run TestE2E -v ./internal/ui/...`; the complete suite remains the final check.
- Golden screenshots are under `internal/ui/testdata/`. Regenerate only with `make golden` (Docker linux/amd64), inspect `internal/ui/testdata/failed/*.png`, and never commit failed renders.
- Tests/golden rendering and Windows/Linux packaging use Docker; macOS packaging is native. `fyne package` may bump `FyneApp.toml`’s build number.
- **Qodana test exclusions:** Whenever adding a `_test.go` file, add its exact repository-relative path to `qodana.yaml` under `exclude` -> `DuplicatedCode` -> `paths`; test-file globs do not work there.
- **UI shard manifest:** Whenever adding a top-level test to `internal/ui`, assign it a shard in `.github/testshards/internal-ui.tsv` (rows are sorted within each `ui-N` block) and refresh that block's entry count in the header comments. `scripts/testshards check` demands an exact assignment for every runnable and runs *before* the race suite in `make verify` / CI, so a missing row fails the gate after the Docker image is already built. `make check-test-shards` is the same check on its own, in seconds.
- **Reading a Qodana report:** `qodana.sarif.json` is the post-suppression result set and
  counts one result per duplicate *cluster*; `log/qodana_inspections_summary.csv` counts
  every finding *before* both source-level suppressions and `qodana.yaml`'s config-level
  scope exclusions, and one row per *fragment*. The two disagree by design — compare
  fragment sets, never totals. CI runs the `qodana.starter` profile, not the IDE Project
  Default, so IDE and CI totals are not comparable either.

## Agent skills

### Issue tracker

Issues and specs live as local Markdown files under `.scratch/<feature-slug>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five-role vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context. `ARCHITECTURE.md` at the repo root is the standing package map (read it before code); `CONTEXT.md` + `docs/adr/` are created lazily by `/domain-modeling`. See `docs/agents/domain.md`.
