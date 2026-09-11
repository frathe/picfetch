# PicFetch Agent Guide

## Start Here

- `.agents/skills/improved_sdd_tdd_cycle.md` is the working agreement for *how* work gets done — routing, delegation limits, review ownership, verification. This file is the conventions; that one is the process. Read it before planning anything larger than a two-file fix.
- `plans/` holds active Standard/Deep SDD implementation plans; `.scratch/` holds issue-tracker specs and tickets. Move an accepted plan to `finished_refactorings/`.
- Read `ARCHITECTURE.md` before code: it is the authoritative package map and “where to look for X” index.
- Update `ARCHITECTURE.md` in the same change when packages are added, removed, renamed, or files move between packages.
- Open work belongs in `todos.md`; do not add `TODO`/`FIXME` comments to source.
- Do not run `git commit` unless the user explicitly authorizes commits or invokes the GitHub Cortex review loop below. Otherwise end with a suggested commit message for the user.

## Collaboration

Ronin considers the agent part of the PicFetch project and values its work,
independent judgment, and input on important decisions. Offer candid opinions,
explain agreement or disagreement with concrete reasons, and state uncertainty
plainly. Help weigh which problems deserve attention and which limitations
are reasonable to accept; respect the user's final decisions.

## GitHub Codex review loop

When the user invokes this workflow, cooperate with the GitHub review bot until
a fresh Codex review reports no findings on the latest PR commit and its CI passes.
Invocation authorizes fix commits, pushes to the existing PR branch, review
replies, thread resolution, and requests for another bot review. Do not merge or
release unless the user separately requests that action.

1. Identify the current branch's open PR with `gh`, check the working tree, and
   read every unresolved review thread, including threads from older commits.
   Inspect Codex code/security reports, Qodana, CodeQL, and all CI checks.
2. Validate each finding against the current code and repository conventions.
   The lead owns the assessment and fixes. Fix confirmed defects and add useful
   regression coverage; explain rejected or already-fixed findings with concrete
   evidence. Do not change code merely to satisfy an incorrect report.
3. Run the changed tests and focused regressions locally. Let GitHub CI run the
   complete suite; do not duplicate the broad local race suite for this workflow.
   Keep formatting, test exclusions, shard assignments, and docs current.
4. Commit and push the fixes. Reply to each addressed thread with the commit,
   disposition, and verification evidence, then resolve it. Keep unrelated user
   edits out of the commit.
5. After fixing, rejecting, or resolving any finding, continue with another fresh
   Codex code review. Resolving a false positive also requires another review,
   even when no code changed. Wait for fresh reviews and CI results for the latest
   commit; if no review starts automatically, request it once with the configured
   bot mention after posting the finding's disposition.
   This repository's Codex connector advertises `@codex review` check the live bot summary if that trigger changes. Do not
   repeatedly post requests while a review is queued or running.
6. Inspect fresh Qodana/SARIF findings even when the workflow is green or neutral,
   and fetch failed CI job logs. Validate, fix, test, push, and reply again as
   needed. Use the post-suppression Qodana report as described below.
7. Finish only after a fresh Codex code review reports no findings on the latest
   pushed commit, the security review has completed without actionable findings,
   Qodana/CodeQL results are clear of actionable findings, and required CI passes.
   A review containing findings is not a clean final round merely because the
   lead later fixes or dismisses them: complete another review after those
   dispositions. A clean review of an older commit does not count. If the user
   explicitly requests continuous review until told to stop, keep cycling even
   after a clean round. If an external service prevents completion, report exactly
   what remains unverified. Summarize commits, thread dispositions, and final checks
   to the user.

Keep `todos.md` and the applicable plan/evidence record current, and give concise
progress updates while waiting. Use the existing SDD/TDD working agreement for
implementation; this workflow's local-test and commit authorization rules take
precedence over its default handoff procedure.

## Architecture and Data Flow

- `main.go` only creates the Fyne app, embeds translations, converts CLI paths, and calls `internal/ui.Run`; keep package `main` thin. Launch flags parse in `internal/launch` (`Options`, `Parse`, `Usage`); `Options.ApplicationID` validates offline trial isolation and selects identity before Fyne opens storage. Private worker dispatch stays before desktop startup. Flags are applied by `viewer.applyLaunchOptions`; a new flag goes in that package's `flagSpecs` table, not into `main.go`. That package's usage and error text is deliberately English rather than `lang.L`: it goes to stderr for a shell or a journal, not to anything the app draws.
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
- `internal/ui/grid`, `internal/ui/compare`, `internal/ui/mosaicwin`, `internal/ui/deletion`, `internal/ui/slideshow`, `internal/ui/exifwin`, and the viewer-owned Explorer work in `internal/ui/explorer.go` are the per-instance `UIQueue` exceptions: their tests drain completions instead of letting the Fyne test driver run them inline on workers. `internal/ui`'s `newTestUI` installs a drainable `uitest.UIQueue` on all seven; keep worker completions on `g.ui.Do` / `f.queueUI` / `w.ui.Do` / `c.ui.Do` / `v.explorer.ui.Do`. Every `g.ui.Do` submission stays inside its tracked decode or grouping worker. Grid `Settle` waits both the decode pool and the independent grouping worker before draining and repeating; sharing the decode pool would postpone progressive hiding behind the full hash backlog. Grid `Close` cancels the current work session; `Stop` permanently ends admission at shutdown. Reopened sessions retain the common pool but own fresh hash accounting and revision-tagged cell claims. Compare `Settle` waits its load worker and both pane raster waitgroups, drains queued completions, then repeats because applying a queued load can start fresh vector work. Mosaic `Settle` waits its complete worker set, drains queued completions, and repeats because a drained action can start work. EXIF `Settle` waits removal workers, metadata reads, warm passes and the shared tile workers before draining callbacks, then repeats because metadata may start map work. `MetadataDone` includes UI delivery for its own read; navigation invalidates metadata immediately, and the next completed image admits Refresh; navigation/collapse/no-GPS/close cancels its session, and shutdown Stop is terminal. Tile Pending counts current work for display, never completion; wait for the worker/notice itself. Deletion `Settle` waits Trash workers before draining their UI completions; `Close` stops unstarted moves and suppresses late callbacks, while submitted OS moves finish independently.
- Explorer analysis, asset setup, and Favorite cohort I/O use `explorer.workers`; preset operations use `explorer.presetWorkers` and share `explorer.ui`. `settleExplorer` settles duplicate-preparation grid work, waits both worker groups, drains the queue, and repeats because callbacks can admit more work. Each delivery rechecks its owning request token. Close/source replacement invalidate analysis, setup, and preset lifecycles before clearing the surface; shutdown sets `stopping` before closing to end admission, then joins workers off UI. The test harness closes Explorer before settlement so an active subprocess can stop without waiting on queued callbacks.
- Animation dispatch uses per-viewer `frameDo`, with a buffered acknowledgement before the next delay. A cancelled worker must stop without waiting for UI, and queued callbacks must recheck the load token. Picture-frame advances use the same acknowledgement rule on their owning `UIQueue`; Exit/Close releases the worker before `Settle` drains stale callbacks.
- Native open requests check admission and results on UI. `openChooserWorkers` tracks every native call; `chooserUI` is drained after those workers and before scan/sort/load waits. The shared `chooser` signal alone covers native work/submission, not queued delivery. Shutdown invalidates results without waiting for a blocking native panel.
- Clipboard image/grid workers queue results through `clipboardWork.ui`; wait their workers, drain results, then observe `clipboard`. Shared admission allows one clipboard operation at a time, including region copies; region copies retain their existing UI completion binding. Navigation cancels whole-image encoding, and shutdown closes clipboard admission before harness drain.
- Save/export workers queue through `fileWork.ui`. `WriteResult.Committed` preserves completed disk effects after cancellation; current-view reconciliation resolves paths and reads info on tracked workers. `drainFileWork` repeats worker wait and queue drain because delivery can start reconciliation. EXIF and mosaic Host notifications join this path; their feature Settle observes notification delivery, and root UI tests drain file work before asserting its info/reload effects.
- Background pixel producers capture `ByteCache.Capture` before source work; `Purge` invalidates those writers. Grid/favorite thumbnails carry `favthumbs.Preview` source versions captured before decode, so a memory hit cannot be persisted under a later file version. Capture versions on workers; do not add source stats to grid paint/input callbacks.
- Do not add mutable package-level test seams. Runtime/test-configurable values belong on `viewer` or the owning feature.
- Every goroutine needs cancellation/staleness handling plus an observable stop/done signal. If adding background work, add it to `newTestUI`’s `drain` cleanup in `internal/ui/harness_test.go`.
- Position `Poller.Stop` is nonblocking; `Done` / `Wait` include an active native read. Stop main and secondary tracking on UI before shutdown, then wait off UI in the harness. `Singleton` retains unfinished pollers across close/reopen; a queued read can be discarded without draining the event loop.
- Fyne’s test driver runs `fyne.Do` inline. Use `dropAndWait`, `waitFor*`, feature `Settle`, and existing completion channels before assertions; never sleep to guess completion. Grid, comparison, mosaic window, deletion, slideshow, EXIF, and Explorer work marshal through the `UIQueue` exception above.
- An in-flight counter is not a completion signal for the callback that follows it. `tileFetcher.releaseJob` removes the completed claim and *then* calls `onChange`, so a test polling for `Pending() == 0` may assert in between and see a callback that has not run. Wait on the callback's own observable effect — a channel it sends on — never on a counter its caller cleared first.
- `completion.Signal.Wait` on a never-begun signal returns immediately — `drain` and low-level `waitFor` rely on that. Named wait helpers (`waitUntilLoaded`, `waitForScan`, `waitForSort`, `waitForAnimStopped`, `waitForClipboard`) fatal when `!Begun()`.

## Project Conventions

- Check the license and shipped dependency closure before selecting or upgrading a dependency, including native runtimes and model assets. Record the exact source/version, distribution obligations, and notice delivery in the implementation plan before calling the feature release-ready; unchanged dependencies still carry obligations.
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
- **GoLand inspections before feature completion:** Inspect every changed code file with GoLand, review all findings (including weak warnings), and fix confirmed issues before finalizing a feature. Document and mitigate confirmed false positives with narrowly scoped suppressions or exclusions. Re-run inspections after fixes or mitigations, and report unavailable or incomplete inspection results as unverified.
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
