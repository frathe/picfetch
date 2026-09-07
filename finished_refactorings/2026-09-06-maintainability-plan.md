# PicFetch maintainability implementation plan — 2026-09-06

Status: implementation in progress, authorized by the user's `/implement sdd tdd` request. Canonical findings and severity are in [needs_refactoring.md](../needs_refactoring.md). This plan references its stable MA IDs and does not maintain a competing findings list. [Audit validation](2026-09-06-maintainability-validation.md) records the baseline and probes. [Published tickets](../.scratch/maintainability/ticket-breakdown.md) track execution; no finding closes without its required evidence.

## Current execution record

Implementation baseline: `d608c17`. Work follows the published required frontier; conditional improvements and accepted watches retain their activation conditions. The lead owns contracts, review, fixes and final gates. No git commit is authorized. Task contracts and evidence are added here as each slice begins.

### Ticket 01 — Safe TIFF reader spans (A / MA-001)

Owner: T0 inline. Route: Standard within the overall Deep effort.
Files: modify `internal/imaging/exififd.go`, `exif.go`, `raw.go`; add `internal/imaging/tiffbounds_test.go` and its exact Qodana exclusion. Reader-helper placement remains within the existing imaging package.
Depends: none; independent of ticket 02.
Contract: shared private `tiffSpan(data []byte, offset, length uint64) ([]byte, bool)` validates without wrapping before slicing. TIFF readers widen offsets before arithmetic. Reader tolerance for partial readable entries stays separate from the writer's complete-IFD policy. Public signatures stay unchanged.
Test: malformed root/sub/next IFD spans in both byte orders cannot panic metadata, orientation or RAW-preview entry points; valid and partially readable tags survive. `FuzzTIFFReaders` exercises these readers with bounded deterministic seeds.
Verify: `go test ./internal/imaging -count=1`; `go test ./internal/imaging -run '^$' -fuzz '^FuzzTIFFReaders$' -fuzztime=30s`; final `make verify`.
Budget: 0 implementation spawns; up to 2 lead review rounds; one final common gate per mergeable change.

Recon delegation: one read-only scout inventories independent mosaic transforms/allocations while the lead implements TIFF readers. G1: bounded source inventory; G2: reported symbols checked by `rg`; G3: no writes; G4: only mosaic-local context; G5: lead has not built that rendering context. S/W: no edit or implementation is delegated. No spec, architecture or review decisions leave the lead.

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| 01 | 0 / 0 | 1 | passed | Root offsets reproduced 12 panics across three readers/two byte orders; fixed reader tests and imaging package pass. Original-source overlay also rejects new nested-pointer guard. `FuzzTIFFReaders` passed 31.457s / 3,181,251 executions. Focused vet and `make verify` pass (617 UI tests plus the remaining Docker race packages). Lead standards/spec review closed. |
| 02 recon | 1 / 1 | lead-owned | no | Read-only allocation inventory |

### Ticket 02 — Bounded mosaic preparation (B / MA-002)

Owner: T0 inline. Route: Standard/Deep boundary, one feature across generator/render and its tests.
Files: modify `internal/mosaic/render.go`, `generator.go` and existing `generator_test.go`; add private preparation planning in `internal/mosaic/preparation.go` if needed. Update the architecture's mosaic resource description when the contract lands.
Depends: none; begins after ticket 01's running gate so source changes do not race verification.
Contract: per-generator `beforePrepare` observes the actual preparation plan before allocating. A 64 MiB scratch ceiling includes NRGBA preparation, Catmull-Rom temporary/distribution storage, masks/rasterizer/filtering, and any extra vector rasterization. Output canvases and already-decoded source/cache memory remain separate. Preserve the current full preparation path when its whole working set fits. Otherwise clip source preparation to bounded destination tiles, preserving the original affine/photo geometry and filter margins, and use bounded-storage resampling. Limit additional SVG rasterization within the same budget; check cancellation before preparation and between bounded bands/tiles.
Test: first expose the existing proposed allocation via a non-mutating instance callback and reject it in a geometry-only regression, before any huge allocation. Test wide/tall ratios through 10000, hidden Catmull-Rom scratch, actual bounded panoramic rendering, cancellation at preparation, and clipped/rotated patterned-source fidelity against the retained full path. Existing seeded output and RAW/GIF/SVG fidelity remain checked.
Verify: `go test ./internal/mosaic ./internal/ui/mosaicwin -count=1`; guarded allocation tests negatively verified against the full-only preparation path; final `make verify`.
Budget: 0 implementation spawns; up to 2 lead review rounds before reassessment; one final common gate. Recon found the pinned scaler's `32 * preparedWidth * sourceHeight` temporary plus distribution tables, which must be included rather than measuring only the explicit layer.

Implementation evidence: the preflight observer reproduced requests of 9,978,524,688 bytes (10000:1), 10,027,519,384 bytes (1:10000), and 157,310,180 bytes from a square source through the hidden Catmull-Rom intermediate. The full-only negative overlay rejects both raster and SVG guards before allocating. Removing cancellation admission produces the expected missing-pixels failure. Actual panoramic rendering crosses multiple tiles under budget. Patterned enlargement/minification at 0, 7 and -12 degrees is compared with retained full preparation (maximum premultiplied channel difference 4/255, mean <= 0.1/255); original ordinary rendering is retained. Rebased small rasterizers initially introduced edge errors up to 165/255: clipping long edges and selecting floating-point rasterization eliminated those errors. Padding that selects the floating-point path is included in the budget. The band-cancellation test now triggers on its first drawn pixel rather than a fragile count of context checks.

Known bounds: live scratch is distinct from GC-retained heap/RSS and already decoded inputs. Additional SVG rasters may use lower resolution under the budget; geometry is preserved. Cancellation is checked before source preparation and between bounded bands/tiles; an already-running SVG rasterizer call is not interruptible. Source geometry outside x/image's coordinate representation returns an error before preparation.

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| 02 | 0 / 0 | 1 | passed | Mosaic and mosaic-window suites pass (2.305s / 1.694s); negative allocation/cancellation overlays fail as expected. `make verify` passes, including the new named regressions under race and all 617 UI tests. Lead standards/spec review closed. |
| 03 recon | 1 / 1 | lead-owned | no | Read-only deletion prompt/list/generation inventory, symbols verified with `rg`. |

### Ticket 03 reconnaissance delegation

One read-only scout inventories deletion prompt identity, list mutation and existing gated tests while T0 implements ticket 02. G1: one bounded deletion call-path question; G2: cited symbols checked with `rg`; G3: no writes; G4: deletion-local context only; G5: the lead has not built the deletion lifecycle context. Budget: one scout, no implementation or review delegation.

### Ticket 03 — Reconcile deletion identity (C / MA-004)

Owner: T0 inline. Route: Deep; feature/host contract and queued lifecycle cross the composition boundary.
Files: modify `internal/ui/deletion/deletion.go`, existing deletion tests and batch tests; add feature-local `uiqueue.go`; modify `internal/ui/viewer.go`, `batch.go`, `harness_test.go`, `run.go`, existing `delete_test.go` and its shard block. Update architecture/concurrency documentation for deletion queue ownership. No new test file or translations are expected.
Depends: none; implementation begins after ticket 02's common gate.
Contract: `Target` carries immutable captured URI identity; `RequestFiles` copies the caller slice and deduplicates URI strings so one physical path is moved once even if merge mode repeats it. Replace deletion Host's index-based removal and generation dependency with `ReconcileDeletedFiles([]fyne.URI) bool`, returning whether the current set changed. Resolve successful URI strings against the current list on UI, removing every occurrence of a file that no longer exists; preserve existing ordinary `RemoveFiles([]int)` admission. Reconciliation closes a comparison opened after the confirmed operation if its current file set is affected, then updates cache/grid/list through the existing removal path. Unrelated replacement sets remain untouched. Each confirmation owns its targets/results independently. The feature owns a `UIQueue` (`Do`, `Drain`), `SetUIQueue`, and nonblocking `Close`; `Settle` waits workers and drains their completions in tests. Closing stops unstarted batch moves and suppresses queued UI publication; an OS move already submitted remains noninterruptible. Harness and shutdown wire this ownership explicitly.
Test: reproduce prompt A/reorder/confirm using temp-file trash stubs and assert disk plus current-list identity. Gated workers cover reorder/replacement while moving, partial failure, overlapping confirmations, caller-slice mutation, duplicate URI occurrences, and closure with queued callbacks. Existing fresh-drop prompt cancellation remains. Composition tests exercise URI reconciliation/cache eviction, reorder-before-confirmation and a comparison opened during a held delete. New tests are assigned exact UI shards.
Verify: `go test ./internal/ui/deletion ./internal/ui -count=1` (UI rendering via the canonical Docker path if native goldens differ); negatively verify the named identity/queue guards; `make check-test-shards`; final `make verify`.
Budget: one completed read-only scout; 0 implementation spawns, up to 2 lead review rounds and one common gate. All contract, UI-policy and review decisions stay with T0.

### Ticket 06 reconnaissance delegation

One read-only scout inventories native chooser result transports, error/cancellation distinctions and existing protocol tests while T0 reviews deletion. G1: bounded filepicker transport inventory; G2: reported identifiers and tests checked with `rg`; G3: no writes; G4: only picker/platform context; G5: the lead has not built that context. Budget: one scout, no implementation or review delegation. This is the second Phase 2 scout (after deletion).

Ticket 03 review evidence: identity, queued completion, overlapping/partial moves and shutdown regressions pass; deletion's race package passes (1.904s). Removing cancellation/comparison reconciliation with overlays fails the corresponding guards. The broader native UI run exposed shared-app session persistence in the shutdown fixture; that fixture now owns its app/cache. All remaining native tests pass except `TestE2E_CopySelection`, which also fails against the pre-03 source overlay. Canonical Linux rendering and shard verification remain pending. The native shard checker additionally sees an existing Darwin-only menu test that deliberately is absent from the Linux manifest.

Gate scheduling: tickets 03–05 remain separate reviewed slices, with targeted red/green and negative guard evidence per ticket. Their common `make verify` gate runs once against their combined mergeable state, before any is resolved. This avoids repeating the expensive Linux UI build while iterating and preserves the common gate before handoff.

### Ticket 04 — Complete shared image records (D / MA-006)

Owner: T0 inline. Route: Standard, imaging and UI.
Files: modify `internal/imaging/loader.go`, `internal/ui/load.go`, `compare.go`, existing `imgcache_test.go` and UI shard manifest.
Depends: none.
Contract: `imaging.DecodeRecord(context.Context, []byte, int64) (*LoadedImage, error)` uses the existing canonical decode and fills encoded byte size and metadata presence. All three full-image cache writers call it; `DecodeLoaded` remains the pixel-only boundary for thumbnails. Read/probe size limits, cache budgets, request guards and Add/AddIfFits admission remain owned by their current callers.
Test: foreground, preload and comparison produce equivalent complete records for ordinary/GPS/oriented JPEG, RAW and animated GIF. Comparison-first cached navigation renders real size and an EXIF link reachable from the window's content tree.
Verify: exact new UI tests, existing imaging/cache/comparison fidelity regressions, full imaging package, and the shared `make verify` gate above. Original comparison source is the negative guard.
Budget: 0 spawns, up to 2 lead review rounds, shared common gate.

### Ticket 05 — Truthful saved JPEG dimensions (E / MA-007)

Owner: T0 inline. Route: Standard, imaging and viewer integration tests.
Files: modify `internal/imaging/save.go`, existing `save_test.go`, `internal/ui/save_test.go` and UI shard manifest.
Depends: none; writer's checked dimension correction already exists.
Contract: SaveRotated uses the existing `dimensionTagsInvalidated` predicate against the original encoded frame and supplies the actual output size to the metadata writer when geometry changes. This supersedes the old policy that Save Changes always leaves dimensions alone. Unchanged geometry and unreadable source-frame fallback retain existing metadata; malformed IFD tolerance is unchanged. Existing writer policy corrects representable dimensions, removes uncorrectable/invalidated coordinate tags, and preserves unrelated metadata.
Test: replace the presence-only save assertion with numeric IFD0/Exif/Interop values after both quarter turns and normalization of orientations 5–8; compare saved frame geometry and retain camera/DPI/ICC facts. Pin unchanged geometry and malformed metadata fallback. Exercise actual viewer Save Changes and its existing failure behavior.
Verify: ticket's exact imaging JPEG/export/EXIF and UI Save selections, negative original-source overlay, shared `make verify` gate.
Budget: 0 spawns, up to 2 lead review rounds, shared common gate.

Tickets 04–05 review evidence: five encoded fixtures across all three cache writers and comparison-first UI navigation pass (1.150s), including the EXIF link in the actual window tree. Existing focused cache/decode/fidelity checks pass (imaging 0.504s, UI 2.292s). Save regressions first failed numerically across all six dimension tags for both quarter turns and orientations 5–8; corrected save/export/EXIF selections pass (imaging 1.652s, UI 7.246s). Full imaging passes (1.757s). Original-source overlays reject both defects; a deliberately incorrect source-header fallback fails the malformed-source guard. Focused vet and diff checks pass. The prior “leave dimensions alone” assertion is replaced; unchanged geometry, ICC, malformed IFDs and viewer save failure remain covered. Lead standards/spec review closed; shared common gate pending.

### Phase 5 reconnaissance delegation

One read-only scout inventories existing native/Store CI and GL smoke entry points while T0 runs the Phase 2 common gate and investigates picker transports. G1: bounded CI/test inventory; G2: symbols and workflow commands can be checked with `rg`; G3: no writes; G4: isolated platform-validation context; G5: lead has not built that context. Budget: one Phase 5 scout, no implementation, policy or review delegation.

Shared 03–05 gate completed: `make verify` passed formatting/TUF/Qodana, native vet/build, the exact 623-test Linux shard inventory, all three UI race partitions and remaining race packages. Log: `/private/tmp/picfetch-maintainability-03-05-verify.log`. Tickets 03–05 are resolved; each used 0 implementation spawns and one lead review/fix round.

### Ticket 06 — Exact chooser paths (F / MA-005)

Owner: T0 inline. Route: Deep, native transport and export contracts.
Files: filepicker dispatch/platform adapters and tests, uitest chooser stubs and their callers, UI open/export and mosaic export, imaging export options/tests, exact manifests and architecture map. Native adapter test helpers are immutable functions, not mutable global seams.
Depends: none; implementation begins after the running 03–05 gate.
Contract: `Choose func() ([]fyne.URI, error)` and `ChooseSave func(string) (fyne.URI, error)` expose validated paths; nil without error means cancellation. Successful empty/malformed native selection is an error. Darwin uses JSON serialization of the actual NSURL path array across the C boundary; native transport tests call the same Objective-C serializer without opening a user dialog. Windows emits structured UTF-8 JSON with explicit process error propagation. Linux retains Zenity: single-save strips exactly its one output newline, and multi-open uses a delimiter containing an internal double slash, which cannot occur in its canonical GFile paths; validate canonical absolute paths and framing before use. Unsupported backend output returns an error, never a guessed path. Zenity exit status 1 is cancellation, other process errors remain errors.
Export contract: preserve the exact confirmed URI. `imaging.ExportOptions.FallbackExt` chooses the selected encoder only when the destination has no supported extension; a supported typed extension still wins. Callers provide their selected format, and no post-confirmation suffix is appended. Zero-value imaging export behavior stays unchanged.
Test: native path transport and chooser-to-open/save cases cover embedded newline, trailing CR/space, Unicode, order, empty/cancel/error and malformed framing; saves assert the exact file and encoded format without overwriting user files. Update superseded automatic-suffix tests. Execute Darwin's real serializer and Linux Zenity in an isolated virtual display; Windows native execution remains a required separately recorded gate. Ticket 14 retains chooser UI admission ownership.
Verify: filepicker suite, exact chooser/open/export selections, mosaic export and imaging export tests, native transport cases, negative guards, and `make verify` before resolving the slice.
Budget: completed read-only transport scout only; 0 implementation spawns; up to 2 lead review rounds before reassessment; one common gate shared with independent Phase 2 local fixes if ready.

Linux transport evidence: [Zenity's output implementation](https://raw.githubusercontent.com/GNOME/zenity/master/src/fileselection.c) prints GFile paths separated by the configured literal and adds one newline. [GLib's local-file implementation](https://raw.githubusercontent.com/GNOME/glib/main/gio/glocalfile.c) constructs canonical filenames. The delimiter's unambiguity follows from that canonical-path constraint, not from assuming an ordinary filename character is forbidden. Test the actual backend and reject output outside that contract.

Ticket 06 progress: actual Darwin NSURL-to-JSON transport passes for newline, trailing CR/newline, spaces and Unicode (native decomposed accent spelling). Exact-path viewer exports pass, and Open preserves supported names with embedded control characters; unsupported extensions still follow the existing scanner's admission policy. Real process exit fixtures distinguish Zenity cancellation from failure, which now produces the existing error toast on Linux too. Complete picker/imaging/mosaic suites pass (0.850s / 2.532s / 1.442s), chooser UI selection passes (3.832s), and focused vet plus Windows test cross-compilation pass. Windows runtime evidence remains pending; a runner-access question is outstanding. Linux native Zenity 4.0.1 transport now passes six exact-path selections and two cancellations in a disposable display with a private controlled selection portal; replay through the production Go decoder passes (0.420s). See `.scratch/maintainability/evidence/README.md` for environment, commands, limitations and retained outputs. No native execution is inferred from compilation or builder assertions.

### Ticket 07 — Preserve clipboard temporary-file failures (G / MA-011)

Owner: T0 inline. Route: Standard, one package and a private fault boundary.
Files: modify `internal/clipboard/clipboard.go` and existing `clipboard_test.go`.
Depends: none; independent of the remaining native picker validation.
Contract: `writeTempPNG` retains OS temp-file creation and delegates the write/close/cleanup transaction to private `writeTempPNGFile(tempPNGFile, []byte, func(string) error) (string, error)`, where tempPNGFile has Write/Close/Name. Creation and cleanup are bound locally, with no mutable global seam. On failure the original write or close cause remains inspectable; cleanup errors may be joined. Success returns the usable temp path. A short write is an error.
Test: extract the fault boundary without changing its current behavior, then fail write/close with successful/failed cleanup using real temporary files and per-call injected faults. Assert errors.Is on the primary cause and cleanup ordering. Existing clipboard dispatch tests remain stubbed.
Verify: full clipboard package, negatively verified error guard, focused vet and the next shared Phase 2 `make verify` gate.
Budget: 0 spawns, up to 2 lead review rounds, shared common gate.

### Ticket 08 — Explicit Windows copied-file decoding (F / MA-012)

Owner: T0 inline. Route: Deep, native Windows consumer.
Files: modify `internal/clipboard/copyfiles.go`, existing `copyfiles_test.go`; add `copyfiles_windows_test.go` and exact Qodana entry.
Depends: none.
Contract: retain the BOM-less UTF-8 CRLF list and existing dispatch/error/cleanup boundaries; PowerShell Get-Content explicitly requests UTF8. Native test intercepts only Set-Clipboard with a script function while executing the production command and reads its JSON result; no clipboard mutation. A per-test ASCII default makes implicit decoding fail independently of host defaults.
Test: existing dispatch contract first rejects missing explicit encoding; native Windows guard exercises temporary accented, CJK, emoji and metacharacter paths in order, single/multiple entries and failed decoding with temp-list cleanup.
Verify: full clipboard suite, Windows test cross-compilation as preparation, then the actual named Windows runtime test and shared common gate. A pending runner is not a native pass.
Budget: 0 spawns, up to 2 lead review rounds, shared Phase 2 gate.

### Ticket 09 — Preserve ordinary Trash configuration (G / MA-013)

Owner: T0 inline. Route: Standard, one package.
Files: modify `internal/trash/trash.go` and existing `trash_test.go`.
Depends: none.
Contract: homeTrashEnv preserves absent/default/custom XDG_DATA_HOME, including unrelated paths containing a snap directory name. Only the demonstrated canonical `<home>/snap/<app>/<revision>/.local/share` redirection (numeric revision, current or common) gets the home default; do not broaden wallpaper's independent schema environment contract. All child OS mutations remain stubbed.
Test: both gio and trash-put environments cover absent/empty/default/custom, true snap redirects and near misses; other variables survive unchanged. Update the older blanket-override expectation.
Verify: full trash/wallpaper suites, negative normalization guard and common gate.
Budget: 0 spawns, up to 2 lead review rounds, shared Phase 2 gate.

### Ticket 10 — Subsecond favorite preview identity (H / MA-018)

Owner: T0 inline. Route: Standard, one package.
Files: modify `internal/favthumbs/name.go` and existing name/store/sync tests as needed.
Depends: none.
Contract: private entryName(hash string, modified time.Time, size int64) string retains a stable path prefix and adds available nanoseconds using separate seconds/fraction fields, avoiding UnixNano range overflow. All new keys differ from the old whole-second format. Existing Sweep prefix and cancellation rules remain unchanged.
Test: controlled timestamp identity plus same-size changed file bytes on the current filesystem; old format misses and is removed by a complete sweep, while cancelled sync retains unvisited previews. Report filesystem precision limits explicitly; deterministic key evidence always runs.
Verify: full favthumbs suite, negative timestamp guard and common gate.
Budget: 0 spawns, up to 2 lead review rounds, shared Phase 2 gate.

### Ticket 11 — Non-wrapping RSS growth (H / MA-019)

Owner: T0 inline. Route: Thin within this execution record.
Files: existing `internal/imaging/heic_leak_test.go` only.
Depends: none.
Contract: test-only rssGrowth(before, after uint64) uint64 returns positive growth or zero. The optional experiment uses it, retaining build tag, opt-in and HEIC fork.
Test: named TestRSSGrowth covers increase/equal/decrease and uint64 boundaries without enabling the long experiment.
Verify: Linux `go test -tags=heicleak ./internal/imaging -run '^TestRSSGrowth$' -count=1 -v`, negative arithmetic guard and shared common gate.
Budget: 0 spawns, 1 lead review round, shared Phase 2 gate.

Phase 2 verification progress: clipboard primary/cleanup/short-write failures reproduced six red cases; Windows list builder rejected implicit decoding; both stubbed Trash adapters reproduced lost absent/empty/custom XDG values; same-size changed bytes at mtimes .1s/.9s reused a stale preview; RSS decline wrapped to 18446744073709551576. All four local package suites now pass (clipboard 0.325s, trash 0.573s, wallpaper 0.799s, favthumbs 1.144s). The deterministic tagged RSS test executes on Linux without opt-in and passes all five cases (imaging 0.058s). All 17 deliberate path/error/environment/timestamp/cancellation/RSS mutations were rejected by their guards; log `/private/tmp/picfetch-maintainability-06-11-negative.log`. The new Windows clipboard test cross-compiles; native execution is pending alongside the Windows picker test. The shared common gate first rejected two import groups; `make fmt` corrected those. The rerun passes formatting/TUF/Qodana, native vet/build, all 625 Linux UI tests and the remaining race packages. Lead review closes tickets 07/09/10/11. Tickets 06/08 retain the required Windows runtime gap. Ticket 12 overlays reproduced premature scheduling and pause scheduling, then passed the acknowledgement design plus existing animation/Copy Selection race tests (6.882s); source edits began only after the common gate completed.

### Phase 3 reconnaissance delegation

One read-only scout inventories the existing delayed-UI audit probes, request/clock seams and lifecycle ownership for animation, picture-frame mode and chooser admission while T0 verifies the Phase 2 changes. G1: one bounded queue/lifecycle inventory; G2: cited symbols checked with rg; G3: no writes; G4: only three feature entry paths and their tests; G5: the lead has not read their current implementations. No implementation, policy or review decisions leave T0. Budget: first of at most two Phase 3 scouts, reused existing Scout agent.

### Ticket 12 — Queue-aware animation pacing (I / MA-003)

Owner: T0 inline. Route: Standard, one viewer lifecycle.
Files: modify `internal/ui/load.go`, `viewer.go`, `build.go` and existing `animate_test.go`; update the exact UI shard manifest and concurrency documentation as needed.
Depends: none; source edits land after the running Phase 2 gate. An isolated overlay may establish red while that gate reads unchanged sources.
Contract: per-viewer `frameDo func(func())` defaults to fyne.Do, matching the existing frameAfter/vector dispatch seams. The worker owns its frame index and receives a buffered acknowledgement describing whether a queued frame actually applied. It starts the next frame delay only after that acknowledgement. Each callback captures an immutable next index and rechecks its load token at application. Cancellation wins independently of a missing callback; a late callback cannot block sending its acknowledgement or paint a replacement request. animationPause still excludes source capture and blocks the existing worker without creating a second loop. Existing v.anim finisher and harness invalidation/wait remain the observable lifetime contract.
Test: a held per-instance uitest.UIQueue with a controlled frame clock and testing/synctest.Wait proves absence of premature scheduling without sleeps. Apply heterogeneous frames and assert actual pixels/index and subsequent delays; cancel with a held callback, install a replacement, then drain it; hold Copy Selection's pause across queued application and resume from the unchanged frame. Construct the viewer through newTestViewer before the isolated worker bubble; each test owns and stops its local request/worker. Retain existing Copy Selection integration coverage.
Verify: named queued guards, existing animation/Copy Selection tests under race, negative acknowledgement/staleness overlays, exact Linux shards and common gate. The pre-existing native CopySelection golden difference remains separate from canonical Docker goldens.
Budget: completed Phase 3 read-only Scout; 0 implementation spawns, up to 2 lead review rounds, common gate shared with tickets 13/14 only after each has independent red/green and review evidence.

| Task | Spawns budget/actual | Review rounds | Common gate | Result |
| --- | --- | --- | --- | --- |
| 06 | 0 / 0 implementation | 2 | passed | Darwin/Linux runtime + decoder replay pass; Windows runtime pending. Native probe artifacts retained. |
| 07 | 0 / 0 | 1 | passed | Primary/cleanup/short-write causes preserved; resolved. |
| 08 | 0 / 0 | 1 so far | passed | Explicit UTF8 portable guard passes; Windows runtime pending. |
| 09 | 0 / 0 | 1 | passed | Ordinary XDG preserved, bounded snap normalization; resolved. |
| 10 | 0 / 0 | 1 | passed | Subsecond key, actual cache miss, legacy migration and cancellation; resolved. |
| 11 | 0 / 0 | 1 | passed | Deterministic Linux tagged RSS guard; resolved. |
| Phase 3 recon | 1 / 1 | lead-owned | n/a | Queue/lifecycle inventory checked with rg. |

### Ticket 13 — Queue-aware picture-frame lifetime (I / MA-003)

Owner: T0 inline. Route: Standard across slideshow and viewer shutdown composition.
Files: modify `internal/ui/slideshow/slideshow.go` and existing tests; add owning `slideshow/uiqueue.go`; modify viewer harness/shutdown and an existing shutdown integration test, exact shard/architecture documentation as needed.
Depends: none; the worker is independent of ticket 12.
Contract: Controller owns a UIQueue (Do/Drain, default Fyne dispatcher), private after timer seam and per-session stop channel. A buffered application acknowledgement paces the next countdown; worker/UI exchange values through channels instead of a shared stale local. Exit invalidates generation and closes the session stop channel. Kick resets the countdown and invalidates any already queued timed advance for that countdown. Close permanently stops admission/work without restoring geometry or painting a closing window; registerShutdown calls it. Settle waits workers and drains stale test completions, never requires UI to acknowledge cancellation. Ordinary Exit retains fullscreen/position restoration and active callbacks.
Test: extract only timer/dispatch seams and a temporary Close-to-Exit implementation for red. Held per-instance queues with synctest.Wait cover delayed pacing, Exit/restart stale delivery, Kick after enqueue, cancelled acknowledgement wait, and terminal Close. Existing interval/shuffle/fullscreen tests remain. Add a shutdown integration assertion using an existing isolated-app shutdown fixture.
Verify: complete slideshow package under race, focused picture-frame/shutdown viewer tests, negative generation/stop/acknowledgement guards and shared 12–14 common gate.
Budget: completed Phase 3 Scout; 0 implementation spawns, up to 2 lead review rounds, shared gate.

Tickets 12–13 progress: the animation regression reproduces a premature old-frame delay and scheduling during Copy Selection pause. The landed frame acknowledgement implementation passes focused race checks (6.882s overlay, then ordinary source), including existing animated Copy Selection integration. Picture-frame tests reproduce premature countdown, a timed advance surviving a manual Kick and nonterminal Close. The controller race suite passes (1.409s), and focused viewer/slideshow race coverage passes (39.928s / 1.345s). A shutdown integration test first proved the controller remained active, then passed after registerShutdown closes it. A held-submission guard proves generation rejection before the worker can observe stop; that guard and the animation replacement fixture pass under race (1.363s / 2.207s). Seven deliberate stale-token, unbuffered acknowledgement, stale-generation, Kick, admission, stop and shutdown mutations fail; log `/private/tmp/picfetch-maintainability-12-13-negative.log`. Source/contract review complete pending the shared common gate; no ticket is closed early.

### Ticket 14 — UI-owned open chooser requests (I / MA-003)

Owner: T0 inline. Route: Deep within the viewer because admission, request lifetime and shutdown composition cross multiple files.
Files: modify `internal/ui/openfiles.go`, `viewer.go`, `build.go`, `drop.go`, `run.go`, harness and existing openfiles/shutdown tests, exact shard and architecture/concurrency documentation. Any new queue type remains private to ui; Run stays the sole exported package entry point.
Depends: independent of ticket 06's exact typed paths.
Contract: openFileDialog is the UI admission boundary, captures the current filepicker.Choose function, begins an openChooserLifecycle token, and owns all mode checks. runFileChooser(token, choose) only invokes the captured native function and submits immutable results through a per-viewer chooserUI queue. A new chooser, an accepted drop or explicit reset/Close Files, and shutdown invalidate old tokens. Delivery rechecks token and comparison state on UI; stale/closed/comparison-covered results are discarded. Current failures are logged/toasted once directly on UI, without a second unguarded hop. closeOpenChooser stops admission and invalidates delivery. openChooserWorkers accounts for every native worker even when the legacy shared chooser completion.Signal points at a later open/export. Worker completion means native work/submission ended; settleChooser and harness drain then drain the owning queue before waiting for child scan/sort/load work. No worker waits for a UI callback.
Native limit: the existing blocking native panel API has no cancellation parameter. Invalidation prevents its result from applying, but an already open panel's worker ends when that panel returns. Shutdown signals invalidation without waiting for that external call; tests hold and then release a stub and observe the tracked worker. Do not hide an untracked goroutine or claim the panel itself was forcibly dismissed.
Test: first extract only the per-instance queue seam while retaining behavior. Held native calls and queued results cover reverse completion of two requests, an intervening accepted drop/Close Files, shutdown and comparison activated before delivery. Direct/menu/shortcut comparison refusal happens before a native call. Current errors are visible only after UI drain; stale errors remain silent. Update direct worker tests to use the actual UI entry and settle helper. All OS operations stay stubbed.
Verify: focused chooser/open/export and comparison refusal tests under race, negative token/delivery/queue guards, full Linux UI inventory and the shared 12–14 make verify gate. Verify each new test is present and non-skipped.
Budget: completed Phase 3 Scout; 0 implementation spawns, up to 2 lead review rounds, shared common gate.

Ticket 14 progress: reverse completion replaced the newer choice, a held result reopened files after a newer drop/Close Files, and obsolete/current errors painted on the worker. All were observed as failing regressions before the lifecycle change. Focused chooser/open/export/comparison refusal race tests pass (56.999s); delivery/admission/shutdown tests pass (12.401s), and the queued-reset plus shutdown scan-start assertions pass (3.725s). Native work captures its function at UI admission, every worker remains tracked, and queue draining precedes child scans in the harness. An already blocking panel is explicitly not force-cancelled. The exact UI manifest now contains 637 tests. Source/contract review is complete; the shared 12–14 make verify gate is running.

### Ticket 19 reconnaissance delegation

The second Phase 3 Scout inventories map metadata/tile fetch/cache lifetime and its window consumers while T0 verifies queue changes and starts ancillary read cancellation. G1: bounded map lifetime/consumer inventory; G2: symbols and tests checked with rg; G3: read-only; G4: only map transport/state context; G5: the lead has not read these implementations. Return ownership, admission/cancellation/completion/cache behavior and existing test seams, without recommendations or code. Budget: second and final Phase 3 Scout; no implementation/review delegation.

Shared 12–14 gate completed: `make verify` passed native vet/build, formatting/TUF/Qodana checks, all 637 Linux UI tests and remaining race packages. Slideshow race passed in 1.106s; UI partitions passed in 226.652s, 260.386s and 326.025s. Log: `/private/tmp/picfetch-maintainability-12-14-verify.log`. Seven animation/slideshow negative mutations and nine chooser mutations were rejected before this gate. Tickets 12–14 and MA-003 are resolved. Each used 0 implementation spawns and one lead review/fix round; the shared full suite ran once.

### Ticket 15 — Context-bearing capture-date sorting (J / MA-010)

Owner: T0 inline. Route: Standard plus viewer integration coverage.
Files: imaging loader and existing loader tests; filesort implementation/tests; existing UI sort tests and exact shard manifest; add `internal/uitest/reader.go` for instance-owned controlled storage sources and update architecture description.
Depends: none; starts after the 12–14 common gate.
Contract: `imaging.CaptureDateContext(context.Context, fyne.URI) (time.Time, bool, error)` carries context into the bounded byte reader and checks cancellation before accepting parsed metadata. Existing `CaptureDate(fyne.URI) (time.Time, bool)` remains a compatibility wrapper. Private sort key extraction returns an error; cancellation aborts key collection and cannot fall back to mtime, while ordinary unreadable/missing metadata retains fallback. Order's exported signature and viewer token/completion ownership remain unchanged. Shared raw reads check context after the final underlying read, including EOF. `uitest.ReaderURI(fyne.URI, func() (io.ReadCloser, error)) fyne.URI` uses one immutable repository adapter registered at init and holds factories on URI instances; no mutable package-level seam or per-test global repository mutation.
Test: controlled multi-chunk reads cancel after their first chunk and prove no second read, reader closure and no mtime stat; normal metadata/missing metadata/read-error fallback remains. A viewer sort held in that reader is cancelled, its own completion handle ends, and no obsolete callback installs ordering. Cancel on an EOF-bearing final read must also reject a valid date.
Honest limit: context is checked between reads. It cannot interrupt a reader already blocked inside Read; the controlled test releases that call and observes prompt termination afterward. No wrapper goroutine hides blocked I/O.
Verify: exact named imaging/filesort/viewer regressions under race, full imaging/filesort, negative propagation/fallback/stale-install guards, and a common `make verify` shared with tickets 16–18 if ready. Verify required tests are listed and non-skipped.
Budget: 0 spawns; up to 2 lead review rounds; one shared common gate after separately reviewed red/green slices. MA-010 stays open until tickets 15–17 all satisfy their evidence.

### Ticket 16 — Cancellable favorite thumbnails (J / MA-010)

Owner: T0 inline. Route: Standard, imaging and favthumbs with existing viewer integration coverage.
Files: imaging thumbnail implementation/tests; favthumbs sync/store implementations and existing tests; architecture contract descriptions. No new test files or viewer tests expected.
Depends: none; reuses ticket 15's controlled instance-owned storage fixture and checked raw read.
Contract: `LoadThumbnailContext(context.Context, fyne.URI) (image.Image, error)` and `LoadThumbnailAndBoundsContext(context.Context, fyne.URI) (image.Image, image.Rectangle, error)` expand the imaging API beside compatibility wrappers. Context crosses source read/probe and is checked around raster decode, vector parse/rasterization and scaling before accepting pixels. A private per-call thumbnail decoder parameter permits a held non-interruptible decoder test without global state. Favorite `ReadContext` / `WriteContext` preserve existing hit/atomic-write behavior while checking cancellation at read/encode/commit boundaries; compatibility Read/Write retain background contexts. Private `syncFile` takes the pass context, checks after memory/disk calls and before writes/Store. Sync's existing bounded admission and waitgroup remain the completion boundary, and cancelled passes never sweep.
Test: actual chunked thumbnail sources stop after cancellation; four held favorite reads exhaust capacity and a fifth cancelled waiter never opens/decodes. Completion precedes assertions, no cancelled result reaches disk or Sink, and an unvisited old preview survives. A following idle pass converges. Held decoder return is rejected after cancellation. Memory-hit/miss cancellation suppresses later disk/sink work; cancellation during preview encode preserves an existing file and removes temporary output. Context-bearing disk decode checks cancellation between reads and after decoder return. Existing format/orientation/animation and preview cache tests remain.
Honest limit: already running decoder, SVG rasterizer, image sampling, filesystem call or underlying Read returns before a boundary can observe cancellation. No forced interruption or detached worker is claimed.
Verify: exact named imaging/favthumbs race guards and negative mutations, complete imaging/favthumbs plus relevant viewer favorite regressions, then the shared 15–18 `make verify` gate.
Budget: 0 spawns; up to 2 lead review rounds; shared common gate. MA-010 remains open pending all 15–17 evidence.

Ticket 15 review evidence: the read regression reproduced eight reads plus an mtime fallback after cancellation; the viewer reproduction likewise read eight chunks. Current race guards pass (imaging 1.504s, filesort 1.541s, UI 3.078s); complete imaging/filesort pass (2.236s/0.458s). Five negative mutations reject missing source context, read-loop cancellation, cancellation-vs-fallback distinction, propagated key errors and stale viewer installation. The key-error guard was strengthened after its first mutation survived: it now asserts the next file is never read, rather than relying on the incidental ordering of a zero-time key. Focused vet and diff checks pass. The common gate remains pending, so the ticket stays claimed.

Ticket 16 review expansion: production shutdown never invalidated favThumbLifecycle, and harness drain waited only the latest completion.Signal although superseded passes can still be inside a read. Extend this owning-boundary slice to `internal/ui/favthumbs.go`, viewer fields, shutdown, harness and existing favorite tests/shards. Pin `closeFavoritePreviews()` (terminal admission guard plus cancellation), `favThumbWorkers sync.WaitGroup` (every pass, including superseded ones), and harness waiting for that full set. Shutdown signals cancellation without blocking on external I/O. A synctest-held old pass must keep the full-set wait pending after a newer pass completes; a real shutdown test must stop delivery and reject fresh admission. This follows the parent's explicit shutdown/harness comparison requirement; no broader Signal semantics change. Budget remains 0 spawns and the same lead review round/common gate.

Ticket 16 evidence: initial guards reproduced three source reads and accepted pixels after cancellation, four active favorite writes/Store calls, stale cache-hit delivery, and replacement of an existing preview during cancelled encoding. Context-bearing imaging/favthumbs race guards pass (1.504s/1.900s). Full imaging/favthumbs/filesort pass (2.163s/1.523s/0.893s); existing viewer favorite/cancellation tests pass under race (6.707s). Nine source/decode/cache/encode/sweep/admission mutations are rejected. The shutdown/tracking expansion separately reproduced three reads, post-shutdown disk/memory writes and new admission, plus a full worker wait that ignored an older pass. Both corrected lifecycle guards and existing viewer favorite tests pass under race (6.896s). Their synctest fixture discards completed bubble-owned handles before outer harness cleanup; it does not leave live work behind. The exact UI inventory is now 640 tests. All three lifecycle negative mutations are rejected; focused vet and diff checks pass. Lead standards/spec review for tickets 15–16 is closed. The shared 15–18 common gate remains pending, so both tickets stay claimed.

### Ticket 17 — Grid work cancellation (J / MA-010)

Owner: T0 inline. Route: Deep, grid lifetime plus viewer shutdown composition.
Files: grid overview/thumbnail/hash engine/dupe-entry/selection code, private `work.go`, existing thumbnail/hash tests and harness; viewer shutdown/harness and a shutdown integration test if needed; architecture/queue convention documentation and exact manifests.
Depends: ticket 16's context-bearing APIs are implemented, negatively verified and reviewed; both slices share their final common gate with 15/18.
Contract: per-overview work session captures a cancellable context, host generation and monotonically increasing revision. Explicit close cancels current work; reopen or an explicit hidden-grid hash command may start a fresh session. FilesChanged/source-generation admission supersedes the previous session. `Stop()` is terminal and is called by viewer shutdown and harness cleanup. A fresh session owns a fresh hashEngine over the same pool/cache/model so old accounting/throttles/URI claims cannot suppress a new pass. `hashEngine.Run` takes context. Thumbnail pool claims carry cell id plus session revision, preventing an old same-cell/id completion from releasing a new claim. Both acquired=false and cancelled acquired=true jobs release their own claim without decode or failure publication. Source reads and cached-thumbnail native backfill receive context; checks after hash/decode prevent cancelled facts. All queued callbacks stay inside their owning pool worker and recheck its context. Settle retains wait/drain/repeat over the common pool. Search/cell recycling retains its separate wanted checks; useful URI cache entries survive ordinary close and search.
Test: hold real thumbnail/hash/native-backfill reads across close, source supersession and stop; no later read, cancelled failure/native fact, cache write or UI callback may land. Occupy all pool slots, cancel a pending cell and observe released claim/completion before occupied slots are freed, without decoding it. Close/reopen the same cell and source while its old read is held; old release cannot erase the new claim or delay the new hash engine's final callback. Preserve ordinary decode failure retry and all hide/browse/cache behavior. Observe exact worker completion before assertions; no sleeps or native desktop effects.
Honest limit: already blocked underlying reads and active decoders/group computation return before cancellation is observed. Ticket 18 supplies atomic model-generation admission independently; ticket 24 owns cancellable group computation.
Verify: focused new grid race guards and existing hash/browse/cell tests, full grid/imaging and relevant viewer integration selections, negative guards, then the shared 15–18 make verify gate. Verify named tests and exact manifest entries.
Budget: 0 spawns; up to 2 lead review rounds; shared common gate. No new shared concurrency framework or mutable global seam.

Ticket 17 progress: close/stop originally allowed three reads, cache/cell/hash/native publication, cancelled failures and late callbacks. The corrected controlled-read/slot tests pass under race (1.571s); same-cell reopen, independent new-pass completion, source retry and terminal admission pass (1.763s), and already-queued delivery guards pass (1.397s). Fourteen negative mutations are rejected. The complete grid race package passes (2.717s), and viewer grid/duplicate/inspect plus shutdown selection passes (81.975s). The initial slot synctest fixture stalled because its semaphore was created outside the bubble; creating the test pool inside it restores a deterministic blocked wait. No timing sleep was added. Review found that closing a partial hide pass now required reopening to resume analysis beyond visible cells; a new guard reproduced zero analysis jobs and missing duplicate groups. Reopen explicitly resumes that pass. Final focused verification and the shared 15–18 common gate remain pending; current UI inventory is 641.

Ticket 17 review follow-up: after reopening began resuming partial hide analysis, the full grid suite exposed an older partial snapshot arriving after the final one. A held first submission reproduces the wrong group size deterministically (0 instead of 2). The session's workers now number snapshots before computing/submitting them; UI-owned last-applied numbering rejects older queued results. This is a local delivery-order guard, not ticket 24's reusable/cancellable group cache. The test preserves the pool-owned submission/wait contract.

Ticket 17 final focused evidence: full grid race suite passes (2.707s), viewer grid/duplicate/inspect/shutdown race selection passes (83.097s). The three review mutations (reopen admission, snapshot ordering and queued cancellation) are rejected. The ordering fixtures use patterned images because uniform images intentionally produce excluded zero dHashes. Lead review is complete; common gate remains pending.

### Ticket 18 — Generation-safe duplicate facts (K / MA-009)

Owner: T0 inline. Route: Deep; model admission contract and both grid worker paths.
Files: internal/dupes facts/groups and existing tests; grid work/hash/thumbnail adapters and existing tests; architecture and ticket evidence.
Contract: capture a model-owned FactWriter before work, binding immutable file-set generation and model reset epoch. Each conditional PutHash/PutFailed/PutNativeSize validates both under the same model lock as its mutation and reports rejection. Clear invalidates even same-generation writers. Explicit adoption retains already-established facts, but never authorizes old writers. Read paths and Compute ignore mismatched namespaces without destroying facts before an explicit UI adoption. Grid sessions own the captured writer; a reset creates a fresh session on next admission. Both workers reject stale facts before cache/UI publication; queued callbacks also reject reset writers. Incremental FilesChanged adopts before capturing its next writer. Compatibility synchronous model puts remain, but background code cannot recapture identity after reading.
Tests: gate hash success/failure, cell success/failure and cached-thumbnail native backfill across same-URI replacement and same-generation Clear, deliberately leaving their contexts alive. Check current facts, cache, queued callbacks and retry. Model tests cover replacement/reset/adoption and an instance-owned FileSet observer using TryLock at the actual generation read, proving validation shares mutation's critical section. Adoption after intervening reads/Compute retains established facts while old writers remain rejected.
Verify: named race regressions, complete dupes/grid packages and viewer integration, negative generation/reset/publication/adoption guards; common make verify shared with 15–17.
Budget: 0 spawns; up to 2 lead review rounds; shared common gate. Group reuse/cancellable computation remains ticket 24; source-file cache policy and cache budgets remain unchanged.

Ticket 18 evidence: held success/failure reads reproduced old hash 0, sizes 8x8/0x0, failure markers, cache pixels and same-generation callbacks replacing current facts. Captured-writer guards pass under race (dupes 1.428s, grid 1.500s); complete dupes/grid pass (1.320s/2.549s). Reset queued-delivery/retry and adoption selections pass (1.364s/1.903s). Fourteen deliberate generation/reset/lock/adoption/producer/queued-delivery mutations are rejected; log `/private/tmp/picfetch-maintainability-18-negative.log`. Production fact writers all use captured handles. Focused vet and diff checks pass. Lead standards/spec review is complete; shared 15–18 common gate is next.

Phase 4 recon budget: one reused read-only Scout (1/1) inventories original-file mutation paths for ticket 23 while the lead verifies ticket 18. It owns no design, edits or review.

Budget extension: a second reused read-only Scout inventories existing clipboard/EXIF refresh tests and completion seams for tickets 21–22 while T0 fixes the manual sort finding. G1: two bounded action paths; G2: cited symbols checked with rg; G3: no writes; G4: isolated existing test/worker context; G5: T0 has not built those tests' context. No design or review delegation; two Phase 4 scouts total.

## Objective and constraints

Prevent input-driven crashes/resource blowups, restore identity/metadata correctness, and make asynchronous work bounded and observable. Improve existing package contracts where the audit demonstrates a gap. Preserve public behavior except the identified defects; record deliberate behavior changes in each implementation PR.

This is a sequence of reviewable work packages, not one bulk refactor. Effort estimates are focused engineer-days including meaningful tests and review, not calendar promises. S = up to 1 day, M = 1–3 days, L = 3–5 days. Native setup can extend elapsed time. Risk describes implementation/regression risk, not audit severity.

Follow the current agent working agreement when executing a package; re-read it rather than treating historical plan instructions as current. Use existing instance-owned lifecycle/queue seams. Add tests that prove the behavior, not mechanical tests of new helpers. New `_test.go` files require exact Qodana exclusions; new internal/ui top-level tests require shard assignments/header counts. Update ARCHITECTURE only if package/file placement changes. All open implementation work stays linked from todos.md. Move accepted/completed plans to the established archive with links updated according to repository process. This audit's explicitly authorized documentation commit does not grant general permission to commit future implementation work.

## Phase 1 — Contain the high-impact input failures

Recommended first implementation phase. Each package can be reviewed independently; neither should wait for a UI redesign.

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| A: Checked TIFF spans | MA-001 | None | M / medium | Arbitrary offsets/lengths cannot panic orientation, metadata or RAW IFD walking. Both endian formats, MaxUint32, near-end entry addresses and nested/truncated IFDs return tolerant results/errors. Existing valid metadata survives. |
| B: Bounded mosaic preparation | MA-002 | None | L / high | Wide/tall source ratios through 10000 cannot request scratch beyond an explicit budget. Check planned allocations before executing them. Ordinary seeded layouts and rotated edge fidelity remain correct; cancellation prevents further preparation. |

A should deepen the existing metadata boundary by sharing checked arithmetic, not merge read/write semantics. B should preserve crop/placement mathematics: clipping or tiled sampling changes allocation strategy, not photo geometry. Do not use a real OOM test as the regression test.

Commands during iteration:

```sh
go test ./internal/imaging -run 'Test.*(EXIF|Exif|Metadata|RAW|Raw|Orientation)' -count=1
go test ./internal/mosaic ./internal/ui/mosaicwin -count=1
```

Add a bounded fuzz run for the new TIFF entry-point target after deterministic seeds pass. Use its actual added name in the implementation record. Existing golden output must remain stable; if an intentional visible change is necessary, use `make golden` in Linux/amd64 Docker and inspect differences. Finish each mergeable change with the common gate below.

## Phase 2 — Correct identity, file transport and complete image records

This phase is separable into small changes. A platform path fix should not wait for a new async framework.

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| C: Deletion target identity | MA-004 | None | M / medium | Prompt A, reorder, confirm: only A leaves disk and current list; B stays. Cover fresh drop, partial trash failure and generation change during worker completion. All OS mutations stubbed. |
| D: Complete shared image records | MA-006 | None | S–M / low | Foreground, preload and comparison preserve identical size/EXIF facts for the same bytes; comparison-first navigation displays correct info. Preserve Add/AddIfFits and stale-request rules. |
| E: Rotated JPEG metadata | MA-007 | A useful but not required | S–M / medium | Save Changes writes actual numeric dimensions for both quarter turns/orientations 5–8; unchanged geometry and unrelated metadata remain intact. Replace the old presence-only assertion. |
| F: Lossless native path transport | MA-005, MA-012 | None | M / medium | Save result is exactly the confirmed path. Open lists round-trip newline/trailing-CR/space/Unicode filenames. Windows PowerShell decodes accented/CJK/emoji UTF-8 paths correctly. Cancellation remains distinct from error. |
| G: Correct local error/environment behavior | MA-011, MA-013 | None | S each / low for clipboard, medium for trash | A failed temp write/close returns its primary error even if removal succeeds; normal XDG overrides survive while demonstrated sandbox redirection is normalized. |
| H: Small freshness/test fixes | MA-018, MA-019 | None | S total / low | Same-size subsecond changes invalidate previews; increasing/equal/decreasing RSS arithmetic cannot wrap. Preserve optional leak-test gating. |

Prefer separate commits/PRs for unrelated entries in G/H during eventual implementation, subject to that session's commit authorization. They are grouped here only for scheduling.

Commands:

```sh
go test ./internal/ui/deletion -count=1
go test ./internal/ui -run 'Test.*(Compare|Cache|Info)' -count=1
go test ./internal/imaging -run 'Test.*(SaveRotated|Export|JPEG|Exif|EXIF)' -count=1
go test ./internal/clipboard ./internal/filepicker ./internal/trash ./internal/favthumbs -count=1
```

Run the added RSS arithmetic test by its actual name; `heicleak` files are excluded from ordinary commands. F additionally requires native Windows PowerShell decoding execution without invoking the real clipboard, and native macOS chooser transport validation without overwriting user files. Unit tests inspecting a generated script alone do not close MA-012.

## Phase 3 — Make worker, cancellation and publication contracts explicit

Use narrow per-feature changes, starting with deterministic failing tests. Do not introduce a universal worker registry or hold the UI thread while waiting for a queued callback.

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| I: Queue-aware animation and admission | MA-003 | None; can start beside phases 1/2 | M / high | A queued driver proves correct heterogeneous frame delays and no worker/UI shared mutable locals. Slideshow stale callbacks cannot advance a closed session. Chooser guard/results run on the right goroutine. Cancel while a callback is pending without deadlock. |
| J: Context-bearing ancillary reads | MA-010 | None | M / medium | Controlled chunked read stops on sort/preview cancellation; a cancelled semaphore waiter never decodes. Worker completion remains observable. Decoder interruption limits are documented honestly. |
| K: Generation-conditional fact publication | MA-009 | J useful; identity tests independent | M / high | Pause old same-URI success/failure, advance generation/reset, then release: no old hash/failure/native dimensions enter the new model. Test reorder/adoption separately from source replacement. |
| L: Bounded tile lifetime | MA-015 | I/J patterns useful, no shared abstraction required | M–L / medium | Concurrent map warms and foreground tiles share an enforced limit; superseded/closed work cancels; failed metadata has an expiry/cap. Completion covers callback effects, not only pending counters. |
| M: Native position-poller lifecycle | MA-016 | I queue test pattern | M / high | Stop before/after enqueue or during native read has defined cancellation and completion behavior. No queued read updates a closed target. Event-loop shutdown cannot wait cyclically on itself. |
| N: Background image action work | MA-014 | I/J; preserve D record contract | M–L / high | Clipboard encoding, metadata read/removal and Save Changes leave UI responsive with a gated slow source. Capture source identity; serialize same-file mutations, preserve atomic writes and report errors once. Close/navigation cannot install old UI state. |

While changing shutdown ownership, compare production shutdown against harness drain for favorite preview, toast, picture-frame, grid and secondary-window workers. Fix proven omissions in the owning feature; do not silently convert every completion.Signal into a broader promise. Grid/compare/mosaic queue/wait ordering in AGENTS.md remains load-bearing.

Commands:

```sh
go test ./internal/ui/slideshow ./internal/ui/grid ./internal/ui/exifwin ./internal/winpos -count=1
go test ./internal/imaging ./internal/filesort ./internal/favthumbs ./internal/dupes -count=1
go test ./internal/ui -run 'Test.*(Anim|Slideshow|Chooser|Clipboard|Save|Exif|Shutdown)' -count=1
make check-test-shards
```

These focused selections are examples over existing names, not a substitute for verifying that each new regression actually ran. Record new exact test names in the PR. Use the canonical Docker race gate at completion; a passing inline-driver race suite alone does not close queue-related findings.

## Phase 4 — Remove avoidable large-folder work and reduce policy drift

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| O: Reusable cancellable group snapshots | MA-008 | K; I/J patterns | L / high | Search changes reuse unchanged groups; distance/hash changes schedule cancellable computation. Superseded snapshots never publish. Benchmarks at 10k/50k/200k cover unrelated and dense hashes without running a quadratic 200k baseline on UI. Golden membership/representative fixtures preserve greedy complete linkage. |
| P: Foreground-aware preview scheduling | MA-021 | J | M / medium; optional after measurement | Measure cold-favorite/interactive latency first. Adopt a bounded yield/share policy only if needed; foreground gets capacity, disk cache still converges when idle, and cancellation does not sweep unseen valid previews. |
| Q: Small shared command admission policy | MA-022 | I; no broad mode refactor | M / medium; opportunistic | Document and test menu/keyboard parity for comparison, Copy Selection, grid, inspect and picture-frame mode, including Escape priority. Centralize repeated decisions only; retain intentional exceptions. |

For O, compare algorithm results against the current implementation on bounded randomized and adversarial inputs. Do not substitute transitive grouping. First eliminate recomputation; choose an index only when profiling establishes the remaining bottleneck. Record hardware, non-race/race status, inputs and wall time with benchmarks.

Commands:

```sh
go test ./internal/imaging ./internal/dupes ./internal/ui/grid ./internal/favthumbs -count=1
go test ./internal/ui ./internal/ui/menus -run 'Test.*(Key|Menu|Compare|Selection|PictureFrame|Inspect)' -count=1
```

Add benchmark names to the work package when implemented and run those exact benchmarks with `-benchmem`. Compare foreground latency as well as throughput for P.

## Phase 5 — Exercise shipped platform contracts and stabilize packaging

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| R: Native/tag/GL coverage | MA-017 | Can start early; validates F/I/M | M–L / low production risk | CI logs enumerate wallpaper Windows guards, actual macOS class-graft test, and Store-tagged distribution test. A maintained native GL smoke verifies comparison pan/zoom/rotation/swipe at representative DPI and large-source detail. Keep Linux golden baseline. |
| S: Reviewed packaging inputs | MA-020 | None | S–M / low | Local/release/Store tool versions come from reviewed constants; cross images are pinned or explicitly versioned with recorded resolution. Build logs identify inputs; packaging smoke runs without publishing. |

Native runner examples (run on their named OS with required C/GL toolchain):

```sh
# Windows: include the existing platform-only guards, not only updater tests.
go test ./internal/wallpaper ./internal/clipboard ./internal/filepicker ./internal/update ./internal/ui/autoupdate -count=1
# macOS: main package includes the Objective-C graft guard.
go test . ./internal/openwith ./internal/displays ./internal/winpos -count=1
# Store behavior is selected explicitly.
go test -tags=microsoftstore ./internal/distribution ./internal/ui/autoupdate -count=1
```

Do not present cross-compilation as execution of these tests. Record the actual native GL environment and checks; headless reference images cannot validate shader compilation/output. Packaging acceptance is artifact inspection and a smoke run, not a release or Store submission.

## Accepted watch items

MA-023 remains mitigated: inspect upstream HEIC fix inclusion at the next dependency update, then run imaging regressions and the optional Linux `PICFETCH_HEIC_LEAK_TEST=1 go test -tags=heicleak -run TestHEICDecode_DoesNotGrowRSSUnbounded ./internal/imaging/...` after MA-019. Keep the fork until fix inclusion is established.

MA-024 requires no immediate implementation. At the next major verifier upgrade, measure reachable dependencies/build time/binary size and review supported footprint reductions without weakening trust or provenance. Do not replace Sigstore verification with hand-written cryptography to reduce a module count.

## Quick wins, ordering and completion

Small independent changes are G's clipboard error propagation, D's record completeness, H's timestamp/RSS arithmetic and S's tool input pinning. They can accompany the first phase as separate reviews; they must not postpone MA-001/002. E is also small but changes an intentional metadata policy and needs value-based tests.

Dependency sketch: I/J enable safe K/L/M/N; K enables O; J enables P. R can start at any time and provides the native evidence needed to close F/I/M. All other work packages are independent unless the current code changes that assessment. Do not batch all UI concurrency changes into one PR.

For each implemented work package:

1. Recheck the pinned finding against current HEAD and preserve unrelated changes.
2. Prove the regression with controlled inputs/queues; assert behavior and file identity, not only widget Visible flags or counters.
3. Implement the smallest owning-boundary change, check targeted tests, and update exact shard/Qodana entries where required.
4. Run `make verify` for each mergeable change. Use `make golden` only for intended render changes and inspect generated differences; never commit failed images.
5. Add native/benchmark evidence required by that package. A passing common gate does not waive an unmet platform acceptance criterion.
6. Mark the MA entry resolved with commit/test evidence, update todos/plan status, and retain the historical audit reference. Keep unresolved watch items open.

The documentation-only audit already passed the baseline gate; no code has been changed by this plan. The existing architecture's Host/snapshot seams, cache semantics, deterministic render references, explicit feature composition and update defenses are explicit keep-as-is constraints.

### Ticket 19 — Bounded map requests and window lifetime (L / MA-015)

Owner: T0 inline. Route: Deep; fetch scheduling plus EXIF window lifetime and harness composition.
Files: exifwin tiles, window and existing tests; a local UIQueue adapter, viewer shutdown/harness integration; architecture/concurrency documentation and exact manifests.
Contract: one fetcher owns at most four active workers and 64 queued URL jobs, shared by Warm and foreground RoundTrip. URL claims deduplicate current jobs, including foreground interest in a warm job. RoundTrip never waits for queue capacity; warm submission waits for capacity or its captured context, without spawning a goroutine per URL. All job contexts derive from a fetcher session captured before work. Source navigation, collapse/no-GPS, close and shutdown cancel obsolete work; queued cancellation releases claims/completions, and active work retains its shared worker slot until its HTTP call returns. Reopen starts a fresh session over the retained useful cache; terminal Stop refuses further admission. Failures expire after the existing 30 seconds and have a 256-entry capacity; cancelled work never creates a failure. Per-job completion includes onChange submission; window warm workers and tile workers are both tracked. The EXIF window's per-instance UIQueue rejects old map generations and Settle waits workers/drains/repeats. All public tile requests remain behind the nonblocking transport.
Tests: a controlled local server records active requests and cancellation, with channels for entry/release/completion. Overlap warm and foreground requests beyond capacity; assert active<=4, queued<=64, deduplication and subsequent retry. Replace/cancel while queued and during HTTP; no stale failure/cache/UI callback, old same-URL completion cannot erase the new claim. Exercise failure capacity/expiry and callback completion independently of Pending. Close/reopen and shutdown use held UI delivery and tracked completion. Preserve lazy expansion, location changes, cache hits and existing rendering.
Honest limits: the 16 MiB PicFetch cache stores encoded tile bytes, not decoded pixels or total RSS. The pinned upstream map widget separately retains decoded tiles in a global unbounded map; this ticket does not claim to bound that dependency's cache. Correct the misleading local cache comment and record this separately in todos. The HTTP transport must return after cancellation/timeout before an active worker can finish; no detached replacement worker bypasses the aggregate bound.
Verify: named controlled-network/queue guards and negative violations; complete exifwin and affected viewer tests under race, then a shared 19–20 common gate if both are ready. No public tile traffic. Inventory required tests and exact manifests.
Budget: 0 new scouts (existing phase-3 inventory reused), up to 2 lead review rounds; one shared common gate. No source edits begin before the 15–18 verification run finishes.

Shared tickets 15–18 common gate: PASS. `make verify` completed formatting/TUF/Qodana, native vet/build and all Docker race partitions; UI shards 212/201/228 tests (641 total), 331.904s/273.269s/258.454s. No failures or race reports. Tickets 15–18 and MA-009/010 are resolved. Log `/private/tmp/picfetch-maintainability-15-18-verify.log`.

Ticket 19 red/green scheduling progress: an external test overlay reproduced 105 admitted requests against the declared 68-job maximum. The candidate shared scheduler passes the existing transport/warm/callback/backoff tests plus aggregate-bound guard under race (1.624s), before repository source installation. Cancellation/window integration remains in progress.

Ticket 19 reviewed evidence: the shared scheduler bounds aggregate requests and queue pressure, retries rejected requests later, joins foreground interest to warm jobs, expires/caps failure records, and tracks callbacks through completion. Gated same-URL success/failure cannot erase a replacement claim or cache obsolete bytes. Full EXIF race suite passes (4.695s); viewer EXIF/shutdown selection passes (18.720s); focused vet and whitespace checks pass. Fifteen deliberate violations are rejected (`/private/tmp/picfetch-maintainability-19-negative.log`). Closing the window originally left HTTP reads alive; the new guard observed cancellation after integration. Review also reproduced an obsolete active request keeping current loading at 1 after replacement completion. Pending/notice counts now include only live-session claims, while physical worker/queue bounds and Wait still cover obsolete work. The redundant aggregate pending counter was removed. All six feature queues are installed by the root harness; UI inventory is 642. Shared 19–20 gate remains pending, so ticket 19 stays claimed.

### Ticket 20 — Observable position polling (M / MA-016)

Owner: T0 inline. Route: Deep; native-read scheduling, secondary window lifetime and shutdown/harness composition.
Files: winpos poll implementation and existing tests; widgets Singleton and existing tests; viewer window tracking/build/runtime/harness and focused existing test files; secondary feature tracking-forwarders; architecture/concurrency docs and exact manifests.
Contract: Poll/PollAt return a Poller exposing idempotent nonblocking Stop, worker Done and Wait. Non-native windows return an already-completed handle. A per-call private read/dispatch/tick seam exercises the actual loop. Each tick queues at most one read and awaits its acknowledgement or cancellation. A queued read has atomic pending/running/discarded admission: cancellation can discard an unstarted callback and end the worker without needing that UI callback, while a callback already inside the native read must finish before Done closes. Both pre-read and post-read guards suppress cancelled delivery, and skip is read on UI. No UI Stop waits for UI; a held native call is an honest completion limit. Main-window startup retains its existing stop callback binding and adds the corresponding wait binding. Singleton retains unfinished poll handles across close/reopen, prunes completed ones, and exposes tracking completion; root cleanup stops all main/secondary tracking before waiting. Runtime shutdown cancels while the event loop is still available.
Tests: explicit tick/queue/read gates cover stop before enqueue, after enqueue without draining, during native read, and while dispatch itself is held; no stopped position publication. Prove worker Done remains open until an active read/dispatch ends, idempotent Stop, live successful reads, failed reads and UI-owned skip. Integration guards cover main stop/wait binding and secondary retained tracking handles. A bounded native make run exercise records OS, window movement/secondary close and observed app exit; retained GL smoke may share that session when available.
Verify: named deterministic winpos/widget/viewer race guards and negative violations, complete owning packages and relevant viewer tests; shared 19–20 make verify gate. Native evidence remains explicit until the actual run occurs.
Budget: 0 spawns; up to 2 lead review rounds; shared common gate. No package-global mutable test seam or general scheduling framework.

### Ticket 15 follow-up — Cancelled sort presentation

Manual testing reported that Escape preserves the previous order but leaves the cancelled mode selected in the menu. Reopen ticket 15 for this regression. Owner: T0 inline; no delegation, one lead review. Contract: cancelling or invalidating a pending mode change restores the mode belonging to the retained order, including menu/title and persisted preference. Superseding requests must retain the last completed mode as their cancellation baseline; a stale completion cannot overwrite it. Selecting the cancelled menu item again must start fresh work and commit the new mode only on successful completion. Keep immediate feedback while a request is active and preserve first-drop cancellation.
Tests: existing viewer/menu action and held ReaderURI seams; assert cancellation before releasing the read, eventual stale completion, and retry success. Verify the named regression under race, related sort/menu/drop tests, a deliberate rollback violation, and include the fix in the next shared common gate. No source fixtures or desktop mutations.

Ticket 15 follow-up red/green: the held-reader menu-action regression reproduced Name unchecked, Capture date checked, the date title prefix retained, and the retry remaining in name order (`/private/tmp/picfetch-maintainability-15-menu-red.log`). The viewer now retains the last completed mode while requests overlap and restores it on invalidation; only current successful completion clears the rollback point. The focused sort/menu/key race selection passes (14.361s). Both new top-level tests are assigned to ui-2; inventory totals 644. Six deliberate mode/menu/title/baseline/commit/stale violations are rejected in `/private/tmp/picfetch-maintainability-15-20-negative.log`. Shared gate is running; manual retry requested.

Ticket 20 review/native evidence: winpos/widgets/settingswin/mosaicwin race suites pass (1.478s/3.034s/4.171s/6.488s). Eight poller/integration negative guards reject queued-wait, premature completion, stopped publication, skipped UI policy, broken Wait, discarded old window handles, missing main wait binding, and reading before worker cancellation. Review strengthened the Wait test after an empty Wait initially survived: it now waits concurrently during the held native read and asserts no premature return. Exact selected test inventory is `/private/tmp/picfetch-maintainability-15-20-inventory.log`. Native current-source window movement, secondary close/reopen and Quit with Settings open completed on macOS 26.6.2 arm64; the process exited and both window positions persisted. Details and screenshot: `.scratch/maintainability/evidence/20-native-poller-smoke.md`. Shared `make verify` is running (`/private/tmp/picfetch-maintainability-15-19-20-verify.log`).

### Ticket 21 — Background whole-image clipboard encoding (N / MA-014)

Owner: T0 inline; Deep for viewer action, clipboard routing and shutdown/test completion.
Files: clipboard implementation and new clipboard work tests, viewer/build/load/shutdown/harness, batch and region-copy admission bindings, exact shard/Qodana entries and architecture docs. Reuse the existing clipboard dispatcher stubs; no platform command changes.
Contract: capture the displayed immutable image reference on UI, then PNG-encode and dispatch off UI. A per-viewer encoder and UI queue allow held work and delayed presentation. A context-aware writer and post-encode check reject cancellation before OS dispatch; navigation/clear cancels whole-image work, shutdown terminates admission. Already-running encoder calls/OS dispatch must return before the worker completes; no claim of undoing an OS write. One viewer clipboard operation is admitted at a time across whole-image, grid files and region copies. A repeated/competing copy uses the existing finishing-copy toast and does not replace its completion signal; synchronous path copy also yields while native work is busy. This bounds encoding and prevents older OS work overwriting a later admitted copy. Preserve region selection's existing completion/queue contract.
Completion: keep the shared clipboard Signal, track every admitted worker, and queue whole-image/grid result effects through a per-viewer clipboard queue. Workers finish after queue submission; the Signal finishes after the UI callback, or immediately after actual cancelled work returns without a callback. An idempotent finisher releases admission only for its own operation. The harness cancels first, waits workers, drains results, then waits the Signal; current tests use the same helper before assertions. Encoding and dispatch errors use the existing image-copy failure text once on UI.
Tests: actual PNG pixel reads gated behind a synthetic image prove an unrelated queued M-key interaction runs while encoding is held; the temporary overlay already fails for this exact symptom (1.331s, `/private/tmp/picfetch-maintainability-21-red.log`). Add cancellation during encode/navigation/close, queued stale/error results, captured pixels across image replacement, one-operation overlap/late completion and shared grid/region/path routing guards. Inventory and negatively verify each boundary; focused UI/clipboard race tests and the next common gate. No real clipboard writes.
Budget: completed read-only phase-4 Scout only, 0 implementation spawns, up to 2 lead review rounds; shared next common gate with ticket 22 if ready. Repository source remains frozen until the 15/19/20 gate ends.

Shared 15 follow-up / 19 / 20 gate: PASS. `make verify` finished all checks and Linux race partitions, 644 UI tests (212/204/228), 325.882s/278.304s/260.193s. Tickets 15, 19 and 20 are resolved; native ticket-20 evidence is retained. Source freeze ended. No commit was created.

Ticket 21 reviewed progress: actual gated PNG reads originally blocked an unrelated queued M-key interaction; moving encoding passed the tracer (3.194s). Navigation originally dispatched the cancelled copy (red 2.092s), then passed with request cancellation (2.963s). Grid/region commands originally replaced the held image copy's Signal (red 4.501s); shared one-operation admission now preserves it, and all routes retry after completion. The focused UI/clipboard race selection passes (72.961s/1.280s), with captured literal pixels across rotation, error/success UI delivery, output-write cancellation and shutdown guards. The retry/output-writer selection passes (6.107s); strengthened active-OS shutdown guard passes (2.211s). Fourteen deliberate violations are rejected (`/private/tmp/picfetch-maintainability-21-negative.log`). Negative review exposed a test-cleanup panic when invalid code queued a closed result; that test now diagnoses missing completion before draining the invalid callback and cleaning its toast. Vet and exact Qodana exclusions pass. The live Linux shard check passes with all 652 UI tests assigned. Common gate remains pending, so ticket 21 stays claimed.

### Ticket 22 — Cancellable asynchronous EXIF panel reads (N / MA-014)

Owner: T0 inline; Deep for panel reads, map/action presentation and viewer notification integration.
Files: exifwin metadata work implementation and window/queue wiring, existing/new panel tests, viewer EXIF/harness tests, exact exclusions/manifests, architecture/concurrency docs.
Contract: Show builds and focuses all widgets before starting Refresh as its last step. Refresh captures the displayed URI on UI, cancels the previous panel read, clears prior text/GPS/strip availability using the existing Loading... text, and runs ReadAndProbe with its captured context plus metadata parsing off UI. The panel owns a metadata generation, context/cancel, per-request completion Signal and all-read worker waitgroup, separate from map-warm work. Only a current generation/context may apply text, GPS and strip availability through the existing EXIF UIQueue. Current read errors keep the existing metadata-unavailable message and log once on UI; cancellation is silent. Close/no-source/Stop cancels; Stop refuses future reads. An active underlying read/parser must return before actual worker completion. Settle waits metadata, warm and tile workers, drains callbacks and repeats; metadata application may start map work.
Notification contract: successful stripping still calls Host.AfterMetadataRemoved exactly once. Capture the local metadata generation around that callback; perform the panel's fallback Refresh only if the Host did not already request one. This preserves existing Host behavior while avoiding the production double reread and still superseding a same-URI read after mutation.
Tests: held real ReaderURI reads during Show/Refresh with a queued window key interaction; replacement/navigation, queued stale results, close/reopen/no-source/terminal Stop and same-URI mutation. Assert visible text/GPS and actual strip-row membership, not Visible alone. Prove cancelled reads stop before additional chunks, old completion cannot change a new Signal, and one post-removal refresh occurs with either host behavior. Existing synchronous tests settle at established action boundaries; never drain while a deliberately held read is active. Inventory, negative violations, full exifwin and affected viewer race tests; common gate shared with ticket 21 once both pass review.
Budget: reused completed read-only Scout only, 0 implementation spawns, up to 2 lead review rounds; no parent-spec edits or general worker abstraction.

### Phase 5 recon budget — native CI and packaging inputs

One read-only Scout inventories the native CI/test entry points and packaging tool/image inputs across Makefile, workflows and their script tests while T0 finishes tickets 21–22. The initial rg found separate release, Store and cross-build paths, so the remaining question is their combined ownership and evidence coverage. G1: bounded entry-point/input inventory; G2: every cited symbol/command checked with rg; G3: no writes; G4: only CI and packaging context; G5: these implementations have not been read by the lead. No review, version selection or architecture is delegated. Budget: one of at most two Phase 5 Scouts, reused existing read-only agent; Phase 4's two-Scout budget remains consumed.

Ticket 22 navigation refinement: the real viewer regression showed old GPS/text/actions installing after both navigation-start and reset (2.763s). Panel Invalidate now cancels and clears presentation at those entry points. EXIF Host.DisplayedFile rejects loading selections, and finishLoad requests metadata only after clearing the loading state, preventing E/Show from admitting metadata for a selected URI while outgoing pixels are still displayed. This refinement remains within the current-read acceptance contract.

Ticket 22 lead review evidence: held metadata I/O blocked an unrelated queued Right-key interaction (red 0.943s), then passed after the asynchronous read contract. Deterministic post-removal admission exposed four reads instead of initial + held + one refresh (red 1.064s); both refreshing and notification-only Hosts now perform one reread. Real viewer navigation/reset reproduced obsolete GPS/text/actions (red 2.763s), and a separate loading-admission regression rejected the selected-but-not-loaded source (red 2.875s). Both are fixed at load entry/completion. Full EXIF race suite passes (6.223s); affected viewer race selection passes (15.728s). Sixteen temporary-overlay violations are rejected, including actual read cancellation, queued/current result delivery, stale GPS/action state, close/Stop, independent completion, all-read Settle, host notification, redundant/fallback rereads and viewer navigation/admission. Evidence: `/private/tmp/picfetch-maintainability-22-negative.log`, `-22-final-package.log`, `-22-focused.log`. Initial asynchronous integration exposed direct-New confirmation fixtures painting concurrently under Fyne's inline test driver; they now use newTestWindow and its drainable queue. Exact inventory, Qodana exclusions, vet, formatting and shard check pass (653 UI tests: 212/213/228). Lead standards/spec review closed; shared 21/22 common gate pending. No Windows/native claims are inferred from these checks.

### Ticket 23 — Serialized cancellable original-file transactions (N / MA-014)

Owner: T0 inline; Deep for write transactions, Save/EXIF/export lifetimes and cache reconciliation.
Depends: preserve 04/05 image/metadata policy and 21/22 request/completion contracts; source edits start only after the 21/22 common gate.
Files: imaging write implementation plus a transaction module and tests; viewer save/export/mutation work and shutdown/harness; EXIF removal work/window tests; existing mosaic export binding where needed; cache/fact invalidation bindings and focused regressions; exact manifests/exclusions and architecture/concurrency docs.
Contract A: retain existing SaveRotated, StripJPEGMetadata and Export signatures as compatibility wrappers. Add context-aware variants returning WriteResult{Path string, Committed bool}; Path identifies the resolved destination, and Committed records an accomplished atomic replacement even if presentation is later cancelled. A production-only per-path coordinator, shared by all imaging write routes, holds admission across resolution-bound read/transform/encode/temp-write/rename. Entries exist only while owned or awaited; waits are context-cancellable, and unrelated destinations can proceed independently. Save/Strip follow existing file symlinks. Export resolves parent directories and rejects destination symlink leaves, preserving the confirmed destination spelling without redirecting a write to an unconfirmed target (PR #17 security correction). Keep permissions, same-directory replacement and JPEG dimension/metadata policy. Context checks bracket reads/parsing and encoder output and precede rename; an already-running read/encoder/native rename must return before completion. This coordinates this process's writers; it cannot serialize an external editor or revoke an already-accomplished rename.
Contract B: Save captures URI, immutable pixels, load revision and rotation on UI; EXIF confirmation captures its URI before admitting a panel-owned strip worker. Both use per-instance operation seams/queues, track every worker, stop admission on shutdown and cancel obsolete work on navigation/close. A repeated active Save/strip is not a second concurrent operation for that owner. Export's existing background route participates in the same transaction coordinator and gains owned cancellation/causal completion. Completion includes current UI delivery, with stale presentation discarded while committed disk/cache effects remain factual. Failed Save leaves the user's rotation intact; a successful result folds captured pixels into display only when the captured source/view still matches. Strip retains one successful-removal Host notification and one post-removal metadata refresh. All native chooser/OS tests remain stubbed.
Contract C: a committed mutation invalidates source decode/thumbnail/fact records, including captured aliases, and prevents older in-flight reads from repopulating invalidated caches. Current source metadata/info refresh once; navigation to another source must retain that source's presentation. Use the existing cache, fact and feature ownership boundaries, not a universal task registry. Specify any needed cache admission identifier in this record before its implementation.
Tests: gate real Save encoding and independently gate strip dispatch; unrelated queued input must run. Hold Save while Strip/Save/Export target the same path or a supported alias (parent-directory alias for Export, file alias for Save/Strip): the later transaction must wait, then see the latest bytes (final dimensions/tags prove the read was also serialized). An unrelated file completes while the first is held. Cancel before output/rename leaves original bytes and no temp file; hold delivery after an actual commit, navigate/close, then verify committed bytes/cache invalidation and unchanged unrelated UI. Assert captured intended pixels, failed-save rotation, exact destination/permission/link behavior, alias export participation, current/stale error delivery and all-worker/causal completion. Add cache repopulation and once-only metadata/info regressions where the asynchronous binding exposes them.
Verify: build-selected inventory; focused imaging and Save/strip/export/mosaic/cache race tests with negative overlays; full required imaging/UI/EXIF coverage via shared make verify once the complete ticket passes lead standards/spec review.
Budget: completed read-only mutation inventory only; 0 implementation spawns, up to 2 lead review rounds, one common gate for the complete ticket. Current Phase 5 Scout returned CI/packaging facts only; its cited symbols were checked locally.

Shared 21/22 common gate: PASS, Shared `make verify` passed formatting/TUF/Qodana, native vet/build and all Linux race partitions: 653 UI tests (212/213/228), shard times 332.664s/287.301s/255.054s. Log: `/private/tmp/picfetch-maintainability-21-22-verify.log`. No commit was created. MA-014 remains open for ticket 23. Source freeze ended. Ticket 23 transaction tracer uses a temporary test overlay: all six Save/Strip/Export same-file or symlink cases reproduce overlapping completion and overwritten dimensions/tags; alias export also replaces the symlink (red 0.595s). Independent-file concurrency already passes. Log `/private/tmp/picfetch-maintainability-23-transaction-red.log`.

Ticket 23 progress (2026-09-07): transaction red reproduced all six shared-file/alias Save/Strip/Export interleavings and symlink replacement (0.595s); the serialized version passes (1.627s). Full imaging race suite passes (19.043s); cancellation, cancelled waiters, commit/no-op/error identity, exact alias paths and claim cleanup guards pass (1.655s). Save Changes blocked queued M-key input (red 2.097s), then passed with worker/queue ownership (2.998s). Existing Save/menu/shortcut tests pass through completion (15.677s); captured pixels, later rotation, before/after-commit cancellation, busy/error/retry and stale failures pass (9.968s). Metadata removal independently blocked queued Right-key input (red 1.018s), then passed (1.993s); full EXIF race suite with removal cancellation/current/stale/busy/error/retry guards passes (5.912s). Strip workers and metadata reads remain separate signals, both included in Settle. Export navigation reproduced a write after the native panel returned (red 2.085s); its owned-context/queue conversion is in progress. The first fixed export run reached the intended no-toast behavior and rejected the tracer's unconditional toast cleanup; the test now asserts silence and cleans up only an unexpected toast. Cache/alias reconciliation, older cache-write rejection, mosaic binding, further export guards, negative verification and the complete ticket common gate remain outstanding; ticket 23 stays claimed.

Ticket 23 cache contract refinement: use ByteCache.Capture() -> CacheWriter[V] with Current/Add/AddIfFits admission under the cache's own lock. Purge advances its revision so pre-commit foreground, preload, comparison, grid/hash and favorite-sink producers cannot repopulate invalidated pixel records. Preserve ordinary Add versus speculative AddIfFits behavior. Committed file writes conservatively invalidate the bounded image/thumbnail caches and current duplicate facts (all aliases are covered without resolving every file on UI); cancellation of obsolete grid/favorite producers accompanies that invalidation. This trades cache warmth after an explicit file mutation for a small, correct identity boundary and does not change cache budgets. Source-change presentation must preserve selection, mode and unrelated view rotation; any comparison reload retains camera/photo transforms and source ordering. Resolve the currently displayed URI against the committed target on a tracked worker before deciding whether its pixels/info need refresh; recheck the captured display generation when applying that decision. File-action drain repeats worker wait and UI drain because applying a write result can start reconciliation work. Preserve each feature's explicit worker/signal ownership; no shared task registry or background/UI filesystem probes are added merely to navigate an unchanged cache.

Ticket 23 cache progress: a pre-purge CacheWriter remained current and admitted stale data (red 0.538s); cache revision admission now passes existing ByteCache tests (1.569s), preserving oversized display retention and speculative refusal. The real viewer tracer then reproduced a cached symlink alias and a held pre-commit preload retaining old 8x16 pixels after a 16x8 save (red 1.964s). Save/export commit effects purge the image cache, foreground/preload/comparison producers capture cache writers before reading, and foreground/comparison retry after invalidation instead of displaying an old decode. The alias/preload tracer passes (2.944s). Thumbnail/fact/favorite invalidation, committed stale UI reconciliation and mosaic binding still remain open. The source-cache unit guard was strengthened to execute Current, Add and AddIfFits checks independently rather than short-circuiting after the first violation. UI shard inventory now has 660 assigned tests (212/220/228); a new exact shard command is still pending after the full ticket's test additions.

Phase 5 Scout 2/2: bounded read-only inventory of duplicate-group semantic oracles and benchmark fixtures for ticket 24, while T0 finishes mutation/cache code. Initial rg identifies groups/dHash assertions but does not connect their semantic constraints to the grid's current recomputation and benchmark inputs. G1: one semantic-test/benchmark inventory; G2: cited declarations checked with rg; G3: no writes; G4: narrow grouping/test context; G5: the lead has read only the Clear/Rebuild call sites for mutation invalidation, not this grouping implementation or its tests. No algorithm, spec, review or implementation decisions leave T0. This consumes the final Phase 5 Scout allowance.

Ticket 23 preview binding: Overview.CaptureThumbs returns a cache writer for a background favorite pass; InvalidateContent cancels/restarts work, clears derived facts and pixels, and rebuilds without clearing selection/query or resetting the highlight. Grid decoders capture cache admission before work. Favorite sync captures EntryName before cached lookup/read/decode and passes that immutable name to its private write helper; a changed source version is never offered back to the memory sink. This preserves WriteContext's public signature and the existing timestamp/size identity policy while preventing a held old decode from being labelled with the replacement's newer disk-cache name. Focused regressions gate the old reader and assert both disk and sink outcomes.

Ticket 23 reconciliation interface refinement: fileMutationWork also owns a shutdown context for resolved-path/current-view reconciliation workers. afterFileWrite(WriteResult, reload, refreshEXIF, done) is UI-owned and carries the action's causal completion through its queued reconciliation; it captures the current URI/load revision, resolves and reads on the worker, then rejects changed display generations on delivery. Save retains an unchanged current view's captured/later rotation behavior; Export or a committed obsolete action reloads a matching current alias. EXIF's Host notification adds the actual WriteResult alongside the original URI, so no later symlink resolution can change what was committed; the panel keeps its existing once-only refresh fallback. Compare.Refresh reloads its existing source pair through its own lifecycle while retaining transforms/layout/link/source order; this is conservative for an active comparison after explicit file writes. Mosaic's Host gains AfterFileExported(WriteResult), and its exporter takes context and returns the committed result. These remain explicit consumer-owned boundaries, with no task registry.

Ticket 23 review refinement: a favorite pass can start after atomic replacement but before its UI notification invalidates the old grid cache. Capturing the current EntryName alone is then insufficient: an old memory hit could be persisted under that new name. Add favthumbs.Preview, an image.Image wrapper carrying the source version captured before decode; Sync only reuses matching versioned memory hits and offers versioned previews to its existing Sink. Grid read producers capture the version off UI before decoding and retain it in the cached image value. Unknown/unversioned memory hits fall back to the disk preview/source path. This keeps the Sink interface and bounded cache unchanged and adds no per-file registry; source version retains the existing path/mtime/size policy. The regression replaces source bytes while leaving the grid cache old, starts favorite sync before committed UI reconciliation, and checks the persisted preview's actual orientation. Finish this guard before the ticket 23 common gate.

Ticket 23 review and verification progress (2026-09-07): grid/favorite stale-preview tracers were red (0.738s/0.501s), then passed (2.104s/1.691s); their complete race suites passed 2.791s/2.042s. Committed Export alias reconciliation was red (0.838s), and comparison queued-pixel refresh was red (0.602s); combined Save/alias/comparison guards passed 12.716s/2.542s. Metadata alias info was red (0.821s), then passed with root/EXIF guards 3.743s/3.527s. Mosaic's committed notification was red (0.690s); its first fixed full-package run exposed a test expectation using /var instead of the resolved /private/var path, corrected to EvalSymlinks. Concurrent unrelated commits reproduced a dropped current-source refresh (0.795s); retry plus mosaic export guards passed 2.996s/3.572s. Save also mutated a decoded Frames slice held by another reader (red 0.754s); it now installs its own one-frame slice when folding the saved rotation.

The focused race selection passed imaging 11.628s, root UI 77.585s, EXIF 4.432s, compare 1.901s, mosaic 3.960s, favorites 3.127s and grid 3.903s (`/private/tmp/picfetch-maintainability-23-focused.log`). Added busy/queued-error guards passed root UI 3.478s and EXIF 2.039s. All 41 initial negative overlays were rejected; source-version review added two more and reverified the two existing preview guards (43 distinct rejected mutations; `/private/tmp/picfetch-maintainability-23-negative.log`). The last memory-hit/version regression was red 0.753s and its affected race selection passed root UI 15.866s, favorites 2.102s and grid 3.000s. No working source was mutated by negative overlays. Source versions are retained in the bounded thumbnail value, not a separate file registry.

Formatting/TUF/Qodana checks pass after separating the new imaging test's import groups. The attempted native `check-test-shards-direct` is not the canonical platform inventory: it reports the existing Darwin-only TestMergeWindowMenus_FoldsEveryDuplicate, which is absent from the Linux manifest. The public Docker check initially lacked socket permission; canonical Linux validation remains in the upcoming `make verify` gate. Root UI manifest now assigns 665 tests (212/225/228), 98 post-baseline additions. Ticket 23 remains claimed until that gate completes; user native sort retest and actual Windows evidence remain pending. Lead review retains existing permissions, metadata/dimension behavior, exact destinations and no-op/error distinctions. Cache invalidation is deliberately conservative after explicit writes; unrelated displayed pixels/rotation and comparison transforms remain preserved. Source version identity retains the existing path/mtime/size policy.

Ticket 24 reconnaissance while ticket 23 is frozen for verification: the Phase 5 Scout's cited declarations were checked locally. `Groups` currently has only Sizes/Reps/Dist; Compute always builds fresh arrays and performs greedy complete linkage, Install unconditionally accepts, and Rebuild calls both. The only production consumers are grid's rebuildGroups, applyHashSnapshot, and hashEngine.Run. Search currently reaches Rebuild per keystroke. FactWriter admissions already guard file generation/reset, but there is no revision for changed hash/native values. MaxDistance is 32 (default 6). Existing grouping oracles pin non-transitive chains, first-seen assignment, uniform hash zero exclusion and largest-native-pixel/lowest-index representatives. No maintained grouping benchmark exists; the historical overlay has unrelated 10k/20k/50k xorshift inputs only. Do not choose an index before reuse is implemented and measured. Any changed grouping must leave UI, including distance changes, browse completion and post-delete inspect retargeting; existing synchronous tests will need causal Settle boundaries at those transitions. No ticket 24 source edits or benchmarks have begun during ticket 23's common gate.


Ticket 23 common gate: PASS. `make verify` completed with native formatting/TUF/Qodana/vet/build and every Linux race partition, exact 665-test UI inventory (212/225/228); UI shard times 307.645s/297.999s/246.786s. `/private/tmp/picfetch-maintainability-23-verify.log`; process exited 0. Ticket 23 and MA-014 are resolved. Code freeze ended. Agent-guide documentation now records fileWork drain, committed Host notifications, and versioned thumbnail admission. No commit was created. Required ticket 24 follows; Windows-native evidence remains pending separately.

### Ticket 24 A — Reusable generation/fact/distance model snapshots

Owner: T0 inline; Deep ticket, first vertical slice. Depends: ticket 18 facts and completed ticket 23 invalidation.
Files: internal/dupes groups/facts/model and existing tests; internal/imaging dHash context variant and existing tests. Later slices integrate grid scheduling and supply benchmarks before choosing any index.
Contract: GroupingKey is an opaque comparable value containing file generation, fact revision, reset and distance; Model.GroupingKey returns the current key. Groups carries its computed key and valid marker. CurrentGroups() (Groups, bool) recognizes only an accepted current key. ComputeContext(context.Context) (Groups, error) captures one immutable file/fact input and checks cancellation during grouping; existing Compute wraps Background. Install(Groups) bool rejects obsolete/untagged inputs under the model lock. Rebuild retains its synchronous compatibility signature but reuses an unchanged accepted snapshot; production UI will move changed work to a feature worker in slice B. Accepted PutHash/PutNativeSize advance fact revision only when the stored value/knownness changes; generation/reset/distance contribute directly to the key. DuplicateGroupsContext preserves the current greedy complete-linkage ordering, zero-hash exclusion and membership; DuplicateGroups remains a Background wrapper. No index is selected in this slice.
Tests: repeated rebuild of unchanged facts performs no new computation; same-value writes preserve the key; changed hash/native facts, distance, replacement/adoption/reset reject old installation. Cancel before or during grouping returns cancellation without a partial snapshot. Existing non-transitive/representative oracles remain authoritative. Verify targeted imaging/dupes tests with negative overlays; the complete ticket gate follows grid integration and required measured benchmarks.
Budget: T0 implementation/review/fixes, no remaining Phase 5 Scout slots; no source changes were made during ticket 23 verification.

Ticket 24 A progress: new reuse/stale-install/cancellation tests reproduced unchanged recomputation, every obsolete/untagged install, and ignored cancellation (dupes red 0.424s, imaging red 0.551s; `/private/tmp/picfetch-maintainability-24-model-red.log`). GroupingKey and fact revision, CurrentGroups reuse, keyed Install rejection and DuplicateGroupsContext checks now pass the existing grouping/representative oracle selection (dupes 1.451s, imaging 1.537s) and the complete duplicate-model race suite (1.362s). Fabricated visibility fixtures now explicitly tag their current input; no production caller receives a way to bless an arbitrary snapshot. Grid's two fabricated search snapshots use a computed current stamp. Its compatibility race suite is running after removing the obsolete fixture import. Changed grid grouping remains synchronous until slice B; ticket 24 is not complete, and no new common gate has run.

### Ticket 24 measurement fixture

Owner: T0 inline; existing internal/dupes/groups_test.go. Add BenchmarkGrouping with unrelated/dense xorshift-seeded inputs at 10k/50k/200k, each cold Compute and accepted Rebuild reuse. Use a prepublished immutable FileSet snapshot so benchmark setup does not accidentally time the test fake's per-call key copying; fill deterministic native sizes outside timing. Dense inputs share a nonzero base with one toggled bit (all pair distances <= 2), exercising complete-linkage membership and representative selection. First measure only 10k/50k, one iteration per case and no race instrumentation, after ticket 23's gate has ended. Record machine/Go/runtime context and allocations. Those measurements determine whether cold grouping needs an index; full required matrix/count=3 follows the accepted implementation.

Ticket 24 measurement: Apple M5 Max, 18 logical CPUs, 48 GiB RAM, Go 1.27.1 darwin/arm64, no race instrumentation; the previous common gate had exited before measurement. The maintained 10k/50k baseline ran one iteration per case (`/private/tmp/picfetch-maintainability-24-bounded-baseline.log`, total 3.941s). Unrelated cold: 59.721ms/1.152305s, 587336/2871368 B, 10014/50014 allocations. Dense cold: 24.283ms/601.868ms, 859416/4420888 B, 27/33 allocations. Accepted reuse had 0 B/0 allocations (single-iteration timings are only a smoke measurement, not steady-state latency evidence). Grid model compatibility also passes its full race suite (2.715s). The 50k cold costs justify indexing; no 200k quadratic baseline was run.

### Ticket 24 indexed grouping decision

Owner: T0 inline. Files: raw-hash grouping in internal/imaging/grouping.go, dHash API location/doc updates, existing model semantic-oracle tests and benchmarks. Preserve DuplicateGroups and DuplicateGroupsContext. Assign each input to the earliest compatible existing group; this is equivalent to the original first-unassigned scan because adding members can never make a prior rejection become a fit. Equal hashes always retain their first assignment and can reuse it directly; complete-linkage checks need only distinct hashes. For distance <= 7 and sufficiently large input, candidate anchors come from four 16-bit projections: for distance 0..3 some block must match exactly; for 4..7 some block differs by at most one bit. Enumerate those keys, verify the full Hamming distance and every distinct group member, and pick the lowest compatible group index. Keep an exact scan fallback for other distances/small inputs. Zero hashes never enter groups. Candidate storage and per-hash assignments are computation-local and O(n); fixed projection heads and per-group links avoid per-query allocation. Cancellation checks bound candidate/member work and never return a partial partition. Do not change representative selection in Model.

Tests: retain existing chain/hub/order/zero tests; compare complete Sizes/Reps against a preserved brute-force oracle over randomized repeated/near/unrelated/unhashed/zero inputs, all important distance boundaries and native-size ties, including lengths that exercise both scan and indexed paths. Verify the bounded benchmark again, then run the required full matrix/count=3 after the final implementation and asynchronous UI integration are accepted. This is a measured cold-path improvement after accepted-snapshot reuse, not a new grouping policy.

Ticket 24 indexed bounded repeat: same native hardware/no-race/one-iteration fixture passed in 0.436s. Unrelated cold 10k/50k: 4.228ms/29.735ms, 4123896/11697712 B, 80/267 allocations. Dense cold: 0.415ms/1.567ms, 822232/4075368 B, 39/59 allocations. Reuse remains zero allocations. The index exchanges bounded computation-local storage for lower cold latency; steady-state full matrix still pending. Randomized and projection/order/complete-linkage adversarial guards pass under race instrumentation.

### Ticket 24 B — Grid-owned asynchronous grouping

Owner: T0 inline. Files: grid grouping worker, hash engine, filter/browse/selection lifecycle and existing tests; duplicate-model snapshot readers; architecture locators.
Contract: one pending grouping request per Overview, with a cancellable context and opaque input key. Further changed requests cancel it and coalesce into the latest request after its queued completion; unchanged requests reuse accepted groups. Group computation runs in the existing decode pool and submits its completion inside that pool body, preserving Settle's existing Wait/Drain barrier and bounded worker capacity. Hash completions only request grouping through the UI queue; they never perform a synchronous regroup on UI. Installation checks the captured session and Model.Install's current input stamp. Close/Stop cancel work; explicit reopen resumes. Search filters the last accepted same-file snapshot while changed facts/distance compute. A snapshot from a different file generation/reset must not hide or retarget any current file. Post-delete inspect retarget waits for accepted new groups. Browse completes only after its hash pass and grouping are current. Completion preserves the current query and highlighted host file.
Tests: park the decode pool and prove hide/distance/search handlers return without a Compute; coalesce repeated changes; drain an obsolete queued result and prove only current groups install; cancel queued completion on Close/Stop; source-change visibility and inspect retarget retain identity. Existing semantic/UI behavior assertions gain causal Settle only where they require completed grouping. Verify focused race suites and negative overlays, then required full benchmark and common gate. No additional delegation.

Ticket 24 B review: the first viewer integration run found four tests assuming synchronous grouping and one real behavior regression: Shift+D on a warm unique source opened Grid before the asynchronous uniqueness result. Preserve the existing no-op by letting the viewer's duplicate-state observer open Grid only when Overview.BrowseReady reports a current, fully hashed duplicate group. Cross-feature opening remains in internal/ui. Groups.Key exposes the captured opaque stamp for notification deduplication; using the model's newer live stamp after Install was reproduced as suppressing the next accepted UI result (red 0.536s). The callback must deduplicate by the snapshot it actually installed.

Ticket 24 B evidence: parked-pool UI/coalescing/queued-close regressions were red (0.568s), then passed under race instrumentation. Stale source/reset readers were red (0.345s); the model race suite passes (1.522s), retaining one-snapshot navigation. Grid's completed-behavior tests now use causal Settle; its full race suite passes (2.857s). The first viewer integration selection found five failures (105.502s): four missing completion waits and the real unique-source opening regression. After delayed BrowseReady opening, the affected viewer selection passes (25.969s) and grid selection passes (1.732s). A fact arriving between Install and notification reproduced dropped current presentation; notifications now use Groups.Key, the captured stamp, rather than the live model key. The held-compute seam is instance-owned; its initially unwired fixture was not counted as a behavior red. Removing active cancellation after wiring does reproduce the intended held-work failure.

All 26 distinct negative overlays were rejected, including input-key components, unchanged reuse, zero-hash exclusion, exact-hash membership, indexed projection coverage, earliest compatible group, complete linkage, representative size/ties, readable file identity, UI execution, active cancellation/coalescing, queued close, latest retry, delayed inspect retarget and unique-source opening. `/private/tmp/picfetch-maintainability-24-negative.log`; only temporary overlay sources were changed. Full benchmark count=3 passes (44.990s), with retained hardware/input/range/allocation evidence in `.scratch/maintainability/evidence/24-grouping-benchmarks.md` and raw output beside it.

The required native package command passed imaging (2.116s), model (0.712s) and grid (1.349s). Its root UI run exposed a synctest cleanup boundary: Save invalidation now creates grouping context channels inside the test bubble, so grid Stop/Settle must also happen inside that bubble before outer viewer cleanup. Fixed test passes under race (root UI 2.889s, grid grouping guards 1.629s). The native run also failed TestE2E_CopySelection: visually inspected actual/master differ by the copy button, not merely antialiasing. No golden was changed. The previous canonical Linux gate passed this test; the new canonical gate must establish whether that difference is platform-specific. Native command failure is retained, not reported as a pass. Formatting/Qodana preflight is clean. Common gate follows with source frozen; root UI manifest remains 665 (212/225/228), and no new test file was added for ticket 24.

Ticket 24 first common gate did not pass: UI-2 found startup grouping before a test marshaler exists and TestStepImage_SkipsHiddenExtras missing its group completion boundary. Other partitions completed; Linux TestE2E_CopySelection passed (1.310s), so the native image difference did not require changing the canonical golden. Empty-distance startup now does no background work (red 0.588s; dedicated negative overlay rejected), and navigation waits for accepted groups. The common gate will be rerun with the adjacent native-CI slice below; ticket 24 remains claimed until then. No commit created.

### Ticket 25 — Native guard execution and evidence

Owner: T0 inline, Deep; no remaining Phase 5 Scout allocation. Files: new scripts/nativeguards command and tests, .github/workflows/ci.yml, architecture locator and exact Qodana test exclusion. Preserve release/Store reuse of the CI gate and existing Linux partitions.
Contract: `go run ./scripts/nativeguards -suite windows|macos|store -capture <json-file>` validates the actual runtime OS for Windows/macOS, rejects unknown suites, lists each package's build-selected tests with identical tag selection, then runs the full named package suite with `-json -count=1 -v`. The Store suite explicitly passes `-tags=microsoftstore`; Windows also includes the ordinary distribution guard. Required package/test pairs must appear in inventory and have run/pass events without skip/fail; malformed events or process failure fail the command. Raw JSON survives failures for CI artifact upload. A per-call subprocess function allows deterministic runner tests without native desktop mutation or package-level seams.
Windows required guards include wallpaper opaque-ID/Unicode/pre-mutation validation, the chooser UTF-8 serializer, clipboard non-UTF8-default transport, native updater replacement/rollback/relaunch/error/wait guards, updater transaction behavior and ordinary distribution policy. macOS includes root main's Cocoa-linked delegate graft, actual open-with Unicode delivery, display and poller guards. Store includes its build-tagged distribution test and existing autoupdate behavior. Actual Windows execution remains pending runner access; cross-build or a simulated runner test will not close that requirement.
Tests: wrong OS/suite refuses before invoking Go; missing selected guards refuse execution; matching inventories/tags/package commands run; required skip/fail/missing run/pass or malformed JSON fails; subprocess failure cannot become success; full valid events yield named evidence. Exact CI command/configuration coverage is coupled to these declared suites. Native macOS and Store commands run locally after the harness passes, and raw evidence is retained. Combined tickets 24/25 common gate follows accepted code, while unavailable Windows evidence stays open.

Ticket 25 implementation evidence: runner tests were red for absent suite selection, missing inventory/execution, accepted incomplete events and missing CI wiring; they now pass under race (1.473s). Twelve negative overlays were rejected (`/private/tmp/picfetch-maintainability-25-negative.log`). Actual native macOS and explicit Store suites both exited 0, all required guards ran/passed and no tests skipped; package times and raw events are retained in `.scratch/maintainability/evidence/25-native-guards.md`. Windows execution remains pending actual runner access, and tickets 06/08/25 remain open for that evidence. Formatting/Qodana preflight is clean. Ticket 24 startup/navigation corrections pass the focused root/grid race selection (12.709s/2.152s), including the empty-startup negative guard (27 total for ticket 24). The combined common gate follows with production sources frozen.

Preparatory inventory for the remaining native/package slices: system_profiler with desktop read access reports Apple M5 Max (40 GPU cores), built-in Color LCD, 3456x2234 Retina; no standard-density external display is listed. Fyne's local v2.8.0 source separates FYNE_SCALE's user/system scaling from the native framebuffer texture scale; a scale override alone must not be reported as physical display-density evidence. Comparison has no rotation command; transformed source boundaries need explicit treatment in its smoke record. Computer Use operates only through node_repl/@oai/sky.

Packaging inventory (no source changes yet): installed fyne CLI is fyne.io/tools v1.7.2 and fyne-cross is v1.6.3, both built with Go 1.26.5. Public registry inspection confirms multiarch amd64/arm64 OCI indexes: Windows sha256:5663afcd79d447e56c57226c3a06cc895fd83a4aa99d2d1974cf0338e9d08f36; Linux sha256:7502500e2224dbbc207df49b13c98b9116a6f6967ff3f8ceab6798be75918706. Native arm64 manifests: Windows 7900518d5ff5655381fe3662b7881133f53efc850a770ddfe03860656c57cd63, Linux 77c4a34d8dbd5ec996241c93b0acefb2b35912f24f5531a1b70343cf0661c5db. amd64 manifests: Windows 76914463edc15322d99a5ed312a12585e6230586d37d378511b72274bfaa88d3, Linux d213d442250c3db3260d61e7b9377f6422abe72c29943b4b1583aafe670a3c4f. Exact inspection output is in /private/tmp/picfetch-maintainability-27-{windows,linux}-image.txt. Existing Windows routes share image/engine/cache/warm-up, Linux omits those explicit choices; CLI installation floats in Makefile and both packaging workflows. Pinning must preserve both host architectures and existing app identity.

### Ticket 26 — Native comparison renderer smoke

Owner: T0, independent manual evidence while the 24/25 common gate runs; no production source mutation. Use generated temporary raster fixtures with known corner/grid landmarks and a rotated source, an isolated temporary app bundle, and the actual production comparison shader/lifecycle. A Go build overlay changes only main's appID for test preferences/session isolation; the temporary bundle uses that same identity. Production appID and packaging files remain unchanged. Record exact fixture/build/launch commands, environment, screenshots, linked/unlinked pan and zoom, swap, side-by-side/swipe/divider extremes, close during work and process shutdown. Comparison has no rotation action: verify its documented canonical-source orientation and that a temporary single-view rotation is not inherited, rather than adding a new feature. The one available panel is Retina; user-scale overrides are supplemental geometry checks and cannot substitute for an unavailable standard-density native framebuffer. Retain any required unsupported display result as open. Computer Use goes through node_repl/@oai/sky; only test fixtures are opened. A later source fix that affects this renderer needs renewed relevant native evidence.


Tickets 24/25 common gate: PASS (2026-09-07). `make verify` exited 0 after native formatting/TUF/Qodana/vet/build and every Linux race partition, exact 665-test UI inventory (212/225/228), shard times 319.878s/307.214s/244.663s. Log `/private/tmp/picfetch-maintainability-24-25-verify.log`. Source freeze ended. Ticket 24 and MA-008 are resolved with the maintained benchmark evidence and 27 rejected mutations. Ticket 25's implementation and native macOS/Store executions pass; actual Windows execution remains open for 06/08/25. No commit or publication occurred.


Ticket 15 native follow-up: user confirmed on 2026-09-07, “sorting cancel works.” Retained order and selected menu sort are accepted.

### Ticket 26 native shutdown regression

The actual bundled production renderer crashed on Cmd+Q with comparison active (process exit 2). Retained crash: `/private/tmp/picfetch-comparison-smoke/app.log`, NSInternalInconsistencyException: native menu mutation on a non-main thread. The exact stack is OnStopped -> compare.Close -> comparisonClosed -> syncMenus -> refreshMainMenu -> mergeAppWindowMenus. Fyne's drained event loop executes OnStopped on its lifecycle goroutine; dispatching through fyne.Do again cannot restore a stopped main thread. The existing shutdown seam must retire UI notifications before feature cancellation, preserving worker cancellation and synchronous preferences/session save. Lead owns fix/review; no delegation.

Test boundary: extend the existing startup/shutdown comparison integration guard to assert that stopping cancels comparison without changing the final window title or menu admission snapshot. Add a viewer stopping flag set before shutdown callbacks; title/menu refresh entry points and queued native-menu follow-up ignore a retired viewer. Verify the existing shutdown regression red before code, focused shutdown/menu/compare race selection, negative overlays and repeat native Cmd+Q with comparison active. Full common gate follows this implementation.


Ticket 26 shutdown evidence: the strengthened integration guard reproduced changed title/menu state and six native-menu reads (0.751s). Viewer retirement is set before shutdown callbacks; title, state recomputation, direct refresh and already-queued native folds now reject the stopped viewer. The test was renamed to TestShutdownClosesComparisonWithoutRefreshingRetiredUI and its existing ui-2 assignment preserved (665 total). All five negative overlays fail the named guard; affected native UI race tests pass 145.623s. Repeated ordinary native comparison/source-detail Cmd+Q exits 0. A separate temporary overlay holds the existing per-renderer tile seam until context cancellation: two calls were observed pending, Escape cancelled both and preserved grid selection; reopening and Cmd+Q cancelled both new calls and exited 0. Retained setup, screenshots, raw crash/fixed/held traces and limitations: `.scratch/maintainability/evidence/26-native-comparison-smoke.md`. Native drag instrumentation shows down/up at the initial point, then movement with Button:0; automated pan/divider drag is inconclusive. Only the Retina panel is available. Full common gate is running; production source remains frozen for it.


### Ticket 27 — Reviewed packaging inputs and real artifacts

Owner: T0 inline, Deep; Phase 5 Scout allowance exhausted. Files: new packaging/tools.mk constants and existing Makefile/release/Store workflows; existing scripts/msixstage test file, packaging guide, architecture locator and .gitignore for local tools. No application identity change. Contracts: FYNE_VERSION=v1.7.2 and FYNE_CROSS_VERSION=v1.6.3, selected from installed module build metadata; FYNE_CROSS_WINDOWS_IMAGE and FYNE_CROSS_LINUX_IMAGE use the reviewed multiarch index digests recorded above. Both image CLIs were executed and report Fyne v1.7.2 with Go 1.25.10 linux/arm64; per-project GOTOOLCHAIN=auto warmup remains required for Go 1.27.1. Install targets put version-specific binaries in ignored .tools directories so an arbitrary PATH binary cannot select the packaging version. Packaging targets log actual host CLI module metadata and container CLI/Go/image identity. Local/release/Store builds consume the same Make targets and constants; deliberate overrides remain visible in logs.

Tests: execute all five cross-package routes with controlled CLI/container boundaries in a disposable directory and inspect argv plus emitted artifact names for both architectures, Store tag and debug flags. Verify quoted cache paths, common engine/image/cache selection, toolchain warmup UID, provenance and failure propagation; existing actual plist/MSIX manifest guards remain. Workflow checks ensure shared install targets and preserve test/signing/publication/WACK gates. First reproduce floating/default input selection, then implement the central inputs. Negative overlays must reject pin drift, lost routing inputs, Store/debug flag changes and omitted provenance. Actual install-tools and macOS/Windows/Store/Linux packaging run on a disposable copy of the complete dirty working source; inspect Mach-O/PE/ELF architecture and distribution metadata. Native startup is separate evidence for each target OS, and actual Windows MakeAppx/WACK remains open without a Windows runner. No commit, tag, push, release or Store submission. Source edits begin after ticket 26's common gate completes.


Ticket 26 common gate: PASS. `make verify` exited 0: native formatting/TUF/Qodana/vet/build and all Linux race partitions, exact 665-test manifest, UI shard times 308.207s/307.879s/246.080s. `/private/tmp/picfetch-maintainability-26-verify.log`. Source freeze ended. The native shutdown defect is resolved; ticket 26 remains open only for unavailable standard-density and inconclusive native drag coverage. Temporary diagnostic app runs have exited and the bundle was rebuilt without instrumentation. Ticket 27 implementation follows.


## User-requested pause, 2026-09-07

User is moving and will temporarily have no internet. Stop implementation and network work until they resume. Current TDD state is intentionally incomplete: ticket 27's Linux executed-route regression reproduced omitted engine/cache/pinned image/UID/provenance, then passed (1.075s). Expanded Linux/debug/Windows/Store/debug routes reproduced missing cache quoting and local-tool/provenance bypasses; those routes plus warmup/Store workflow guards now pass (2.755s). `packaging/tools.mk` exists with reviewed versions and multiarch digests, version-specific .tools installs, common warmup/provenance and all five cross routes. The next test, TestPackagingToolsUseCurrentFyneCLI, has been expanded and is RED: macOS packaging and release/Store workflows still bypass the shared install targets. `/private/tmp/picfetch-maintainability-27-install-red.log`. Next: inspect this failure, wire package-mac to install-fyne/FYNE_BIN and replace workflow @latest installs with make install-fyne or make install-fyne-cross, then run the complete script suites, strengthen failure/pin guards and negative verification. No ticket 27 actual package build/common gate has run. New test edits still need goimports. No commit.

Actual native environments discovered: UTM contains Windows 11 UUID FEA5F12D-9650-4019-9FDD-4FA4F363F418 and Ubuntu UUID 6227E83A-B088-44B3-AEB1-9E5600104354, initially both stopped. Windows was started and reaches its Desktop. User explicitly says Windows executables must be placed on that Desktop because the needed libraries live there; preserve the existing picfetch.exe and libraries and use a distinct smoke filename. The Desktop visibly has opengl32.dll, libEGL.dll and GLES libraries. utmctl exec reports no running/installed QEMU guest agent (OSStatus -2700). Opening PowerShell via automated mouse/keyboard has not yet succeeded; the VM desktop remains open, no guest test files or commands were written/executed. A mouse-capture alert was dismissed during local user interaction; latest capture state is off. UTM screenshot dimensions include a blank right region; use fresh state/coordinates. All GUI interaction is through node_repl/@oai/sky. No Windows native test or package/WACK result may yet be claimed. Ubuntu has not been started.

All temporary macOS comparison processes have exited. The temporary comparison bundle has been rebuilt without diagnostic overlays. Ticket 26's common gate passed (all 665 UI tests and remaining Linux race packages); native ordinary quit and held tile cancellation/quit passed. Native mouse drag remains inconclusive due the automation's down/up-before-motion sequence and standard-density output is unavailable. Ticket 15 user confirmation: “sorting cancel works.”


## Resume, 2026-09-07

User resumed `/implement use ssd and tdd`. Ticket 27 native/workflow installation regression is green. All five cross routes now reject first-architecture artifact-copy failures (red 5.217s; both packaging suites green). Negative verification rejected 17 mutations; it exposed and corrected a weak whole-log UID assertion, which now checks each container invocation. Reviewed tools installed and actual macOS plus Windows/Store/Linux amd64+arm64 packages built successfully in `/private/tmp/picfetch-maintainability-27-package`. Artifact inspection confirms Mach-O/PE/ELF architectures, Store-only tag, production macOS identity/version/associations. Actual packaged macOS ARM64 app rendered the generated Alpha fixture and Cmd+Q exited 0; app preferences/session directories were temporarily isolated and restored automatically. Retained evidence starts at `evidence/27-*`.

First ticket 27 common gate failed only scripts/testshards TestMakeCIFailuresFindsLatestCompletedRunOrAcceptsRunID: adding tools.mk to MAKEFILE_LIST caused grep to prefix every help row with its source filename. Existing guard reproduced natively; Make help now uses grep -hE. All three Linux UI shards passed; rerun affected script suites and the full gate after this correction.

User confirmed the Windows VM has no network and mounted the disposable checkout as Z:. Offline Windows/arm64 test executables (CGO_ENABLED=0, Go 1.27.1), standard Go test2json and a PowerShell harness are prepared under its offline-windows directory; Z:\go.cmd runs eight complete packages and checks all 14 required native Windows/Store guards without skips. No Go installation/network required in the guest. Binaries copy to distinct Desktop filenames alongside existing libraries. The guest agent is absent; automated GUI input is unreliable, so the user was asked asynchronously to run the prepared launcher. No Windows result yet. Ubuntu VM also boots to its desktop, reports failed network activation, and lacks its guest agent. Its shared-folder control is disabled; native Linux execution remains pending. No publication or commit.

Ticket 27 final gate rerun: every comparison test printed PASS, but its process was then killed. Docker cgroup evidence confirms oom_kill=1, memory.peak=7931346944 bytes, Docker VM MemTotal=8319213568 bytes; only the test container was running. All root UI shards subsequently passed. This is an environmental OOM, not a claimed successful gate. Rerun the unchanged `make verify` via a temporary PATH Docker wrapper injecting only GOGC=25 into its child test container. This makes Go collect more frequently without reducing inventory, package concurrency or race instrumentation; no Docker settings or repository test policy changed. Log `/private/tmp/picfetch-maintainability-27-verify-gc25.log`. Ubuntu was confirmed stopped after unsuccessful GUI access; its prepared smoke ISO remains mounted.

Ticket 27 common gate: PASS with temporary Docker-test GOGC=25 wrapper, same complete make verify path, race instrumentation, package concurrency and 665-test inventory. UI 334.273s/324.107s/258.171s; all other packages/native gates passed; process exit 0. Evidence `27-verify.txt`. Source freeze ended. User ran offline Windows go.cmd and reports completion, but returned logs are blocked by the intentionally read-only Z: share. Retrieve existing results rather than rerun. User also ran pack.cmd and supplied the exact missing SDK/MakeAppx/SignTool/WACK prerequisite error; no package/trust changes were made. Preparing Microsoft's offline SDK ISO 10.0.26100.9169 (1,170,829,312 bytes) from the official download page, currently mounted read-only on the Mac at /private/tmp/picfetch-windows-sdk. No SDK installed.

### Native Windows full-package follow-up

The temporary share was changed to writable and Windows restarted; the user ran the copy-only results launcher. Retained Windows run 20260907-115026 is RED: TestHostSchemaEnv_DropsSandboxInjectedSchemaSources loses the host XDG entries. Both required Windows wallpaper guards passed, but the complete wallpaper package failed and the harness stopped before the other seven packages. The Linux XDG parser used os.PathListSeparator, which becomes a semicolon in the Windows test binary. Owner: T0 inline; existing regression is the red test. Fix only hostDataDirs to parse its colon-delimited XDG protocol independent of the execution host, then run the full wallpaper suite locally and on Windows. The offline harness will collect all package results before reporting failure, and use a new versioned wallpaper test filename to preserve the prior binary. Required native evidence and the common gate remain pending for this follow-up; no test skips or weakened assertions.

The user now owns VM keyboard input after confirming dropped automated keystrokes. Updated Z:\go.cmd is ready: complete package results are collected despite individual failures, and the changed wallpaper binary uses a new r2 filename. Native Mac wallpaper package passes (0.391s); Windows rerun is pending. SDK option-manifest inspection retains source/hash and x86/amd64-only WACK payload conditions; ARM64 installation viability remains unproven. The temporary host ISO mount was ejected, leaving the downloaded ISO on the share.

Windows rerun 20260907-121327: seven complete packages pass, including the fixed wallpaper package and actual chooser/clipboard/Store guards. Full update package aborts when testing.Chdir cannot restore a directory handle into the WebDAV share; TempDir cleanup then encounters the still-current directory. Retained raw results: evidence/25-windows-r2/. Update tests generate their fixtures and have no repository testdata dependency. The unchanged update executable will be rerun from a new local Desktop results subdirectory via Z:\update.cmd; no updater source or test change is justified by this setup failure.

Native Windows follow-up complete: run 20260907-122203 passes the unchanged updater binary from local storage (99 top-level tests, 2.116s). Combined accepted runs cover all eight packages, 254 top-level test executions and no skips. Production suite/guard replay passes all 14 required guards (0.286s). The common gate passed with the established temporary GOGC=25 Docker wrapper, exact 665 UI tests (212/225/228), shard times 347.975s/333.033s/267.607s, all remaining race packages, native checks/vet/build and exit 0. Windows/amd64 cross-vet passes. Source freeze ended; tickets 06/08/25 and MA-005/012 are resolved. MA-017 retains ticket 26's native renderer gap. Lead reviewed/fixed inline; no new delegation.

Refreshed all seven reviewed packages from the final verified source via make package-mac package-windows package-windows-store package-linux in the disposable checkout (exit 0). Architecture/Store tags/CGO and production identity/version inspection pass; fresh Mac build 450, root FyneApp.toml unchanged. The user launched Z:\smoke.cmd, which runs the four Windows variants in sequence with process-specific test profile/cache directories and hash-suffixed Desktop filenames. First Alpha window visibly renders all four landmarks; retained 27-windows-first-alpha.jpg. Process identity/exit/DPI logs return after the sequence finishes; no clean-quit or remaining-variant pass is claimed yet. The user owns keyboard/mouse input; the next requested action is Right to view Beta while leaving the first window open.


### Windows graphical results and corrected attribution

The user completed the four-attempt launcher. Returned process logs identify ordinary ARM64 PID 6672 and Store ARM64 PID 1900, both window DPI 96 and exit 0 with empty stdout/stderr. Ordinary ARM64 passed Alpha/Beta navigation and user-operated comparison linked/unlinked pan, divider drag, linked wheel zoom and unlinked full-source detail. Store ARM64 visibly rendered Alpha and quit cleanly. Both amd64 builds instead exited 2 before a visible window: WGL OpenGL context unavailable, followed by Fyne applying a Windows theme to a nil window. The previous visual-order interpretation was wrong: the empty drop area was in the still-running ordinary ARM64 process, not a new package failing image load. Evidence documents are corrected and raw process/transcript/stdout/stderr records retained in `evidence/27-windows-package-results/`; duplicate fixtures/cache files are omitted. All hashes match the refreshed inspected artifacts. `Z:\details.cmd` is a read-only DLL-architecture/driver/monitor-scale probe to determine the environment before a runtime or source change. Windows x64 startup, verified graphics/display context, Linux native startup and WACK remain open. Evidence-only changes need no repeated common gate. Lead retains diagnosis/review; no new delegation.

The environment probe returned successfully: Windows 11 Pro ARM64, VirtIO GPU DOD driver 22.7.38.43, a 1728x1043 virtual monitor at 100% scale (API success), and four ARM64 Desktop GL libraries. Together with ordinary ARM64 process DPI 96, this completes ticket 26's standard-density/environment criterion; native user pan/divider/zoom/detail, prior Retina orientation/swipe/detail and held-work cancellation cover the remaining criteria. Ticket 26 and MA-017 are resolved with documented environment limits. Ticket 27's x64 runtime hypothesis will be tested by unchanged binaries beside an isolated amd64 Mesa llvmpipe 26.2.2 opengl32.dll; archive digest matches the release API, PE architecture/hash retained. Z:\x64.cmd is prepared; no existing Desktop libraries or system settings change.

The first isolated x64 retry stopped before launch because WebDAV rejected reading the 58,609,152-byte runtime DLL. Raw failure/transcript retained. The same DLL now transfers as a 21,995,876-byte ZIP, with archive and extracted hashes checked; host ZIP round-trip verification passes. Z:\x64.cmd was updated to expand locally and return only JSON/text records, then the user was asked to rerun. No source change or system/share-limit setting change.

Compressed x64 transfer run 20260907-134213 loaded the correct isolated OPENGL32.dll in both packages; each created a window at DPI 96 but exited 2 with illegal instruction 0xc000001d during initial glDrawArrays / Fyne rectangle painting. User confirms immediate window closure. Raw modules/crashes retained. Hypothesis order: generated CPU instruction unsupported by emulation; Mesa build/code-generation bug; app/Fyne-specific trigger. First probe changes only child GALLIUM_OVERRIDE_CPU_CAPS=sse2 with capability logging; Z:\x64-sse2.cmd prepared. No production source or system setting change; x64 acceptance remains open.

User steering: stop Windows experiments, write a test todo, continue non-Windows validation. User will test later on native x64 systems. Checklist is `.scratch/maintainability/windows-test-todo.md`, linked from todos and ticket 27. SSE2 experiment remains unexecuted; no more Windows runs requested. Next: inspect the refreshed actual macOS package and complete available Linux graphical validation, retaining any native-environment limitation explicitly.

Non-Windows verification complete: actual refreshed macOS build 450 rendered Alpha, Cmd+Q exited 0, log empty and original preferences/session restored. Both unchanged Linux packaged executables passed Alpha/Beta rendering and navigation, then native window close exited 0 with empty application logs. ARM64 also passed production comparison side-by-side/swipe and quit with comparison open. Isolated Debian 12 desktops used Mesa 22.3.6 llvmpipe / LLVM 15.0.6 and Xvfb 1280x960 at 96 DPI. ARM64 runs natively in Docker's Linux VM; amd64 uses CPU emulation on the ARM64 host. Retained `evidence/27-linux-packaged-smoke.md` documents exact hashes, runtime identities, setup, screenshots, initial environment corrections and terminal records. User Windows VM was not touched after deferral. Both disposable Linux containers and their browser tabs are cleaned up after completed app exits. Required outstanding work is the user's later native x64 Windows smoke and WACK checklist; conditional tickets 28–30 remain unselected. No production source changed, so no repeat common gate is warranted.


## Phase 6 — Selected remaining conditional work, 2026-09-07

User requests the remaining open tickets and owns the Windows checklist before
merge. Continue 28 and 30; activate 29 only from measured contention. Tickets
31/32 retain their separately selected dependency-upgrade triggers. Ticket 27
stays deferred; do not perform more VM experiments. Baseline is 524fcf8, clean.
Route: Standard slices within this existing Deep plan. T0 owns implementation,
design, review, negative verification and final gate. One shared full make verify
after the selected slices, with the established GOGC=25 Docker wrapper if needed.

### Task 28 — Measure the existing prewarm boundary
Owner: T0 inline. Files: internal/favthumbs benchmark test, qodana exact exclusion,
evidence and tracker docs. Contract: BenchmarkPreviewForegroundContention calls
public Sync against the real imaging thumbnail boundary used by foreground work;
controlled source admission records actual overlap, latency, throughput,
allocations and completed disk previews. No production scheduling change before
measurement. Source set and cache conditions must be reproducible, and reader
cancellation versus a running decoder distinguished. Verify: ticket 28 benchmark
command (three samples), focused package tests, negative instrumentation check,
then shared common gate. Budget: zero implementation spawns, one lead review;
measurements may justify a separately recorded ticket 29 policy.

### Task 30 — Capture and guard command admission
Owner: T0 inline. Files: existing keys/shortcuts/comparison/Copy Selection tests,
menus tests as needed, exact UI shard inventory, evidence matrix and tracker docs.
Contract: exercise viewer actions, actual menu callbacks and registered canvas
shortcuts in the existing UI harness; retain mode state in its owning features.
Pin comparison, Copy Selection, grid, inspect and picture-frame decisions and
Escape priority, including comparison Help and explicit Open refusal. Consolidate
only a demonstrably repeated decision; a tested matrix is an accepted outcome.
Verify: named non-skipped guard inventory, negative overlays, ticket 30 package
command and the shared gate. Budget: one read-only Scout, lead review/fixes.

Delegation gate for command inventory: G1 yes (one bounded inventory question);
G2 yes (verify returned locators using rg -n against the named tests/routes);
G3 yes (zero files changed); G4 yes (short question, no plan transfer);
G5 yes (lead has only route names from shell searches, no matrix context).
Rule S: the cross-file behavioral mapping needs reading, not text substitution.
Rule W: no implementation prescribed or delegated. Scout returns locators only
while T0 builds ticket 28. Harness mapping: T3 uses available gpt-5.6-luna,
read-only, cold context. Phase 6 Scout budget 1; no peer review delegation.


### User-reported regression — Progressive hide updates
User confirms duplicate detection finishes correctly but the view now waits for
the entire list; it previously updated successively. Prioritize this ticket 24
follow-up before completing Phase 6. Existing focused hide/grouping and command
tests pass (root UI 5.864s, grid 1.149s), so add a missing partial-pass guard.
Owner T0, no delegation. Seam: Overview.SetHideDuplicates with controlled source
read admission and the real UIQueue. Complete two matching sources while the
rest remain blocked; draining ready UI work must hide one extra while preserving
unhashed files. Verify the named regression first, then all grid/UI interactions
and negative cancellation/staleness checks. Design decision follows the red.
Existing Phase 6 make verify remains the final shared gate.


Progressive hide red: TestHideDuplicatesPublishesWhileSourceReadsRemainPending
fails with 8 visible instead of 7 (grid 0.442s). Two completed matches plus six
held sources reproduce the user symptom. Ranked probes: grouping queued behind
all hash reads; notification throttle; stale partial installation. First change
only grouping admission to an independently tracked single worker. Keep strict
snapshot freshness and coalescing unchanged. Grid Settle must wait both worker
owners before draining UI and repeating; Close/Stop continue cancelling the
shared session context. Update tests that waited only the old pool, architecture
and the documented grid completion invariant. No public interface change.


### Task 29 — Measured CPU-aware preview admission
Ticket 28's corrected leaf-scoped GOMAXPROCS measurement demonstrates contention:
on P2 median cold foreground p95 is 175.5 ms versus 56.91 ms alone; foreground
throughput 6.834/s versus 17.82/s. All 64 disk previews converge, median 2220 ms.
P18 remains bounded by four background workers and has much smaller contention.
Activate 29 with the smallest policy: each Sync captures a background worker cap
of min(4, max(1, GOMAXPROCS-1)), retaining a separate foreground pool and leaving
one Go execution slot outside prewarm on multi-processor runs. This is a worker
budget, not an OS CPU reservation; a single-processor runtime cannot reserve a
second processor. No pause timer, global semaphore, new UI seam or API.
Measured acceptance target: P2 median cold foreground p95 <= 1.5 times the paired
alone case, foreground throughput >= 80% of alone, preview convergence <= twice
the measured 2220 ms baseline. Use the identical 64+8 JPEG workload, three samples;
retain all baseline and after metrics/allocations, including any missed target.
Tests at public Sync: held reads expose actual admitted workers for P1/P2/P3/P8
(expected 1/1/2/4), a foreground decode completes while background reads remain
held, cancellation joins workers and the next idle pass converges. Existing
cancel-without-sweep and full-memory-cache guards remain required.
Files: internal/favthumbs/sync.go and existing sync_test.go, benchmark/evidence,
architecture locator. T0 inline, zero spawns, one lead review plus negative
mutation, focused package tests and the shared common gate. Rebuild test artifacts
from the final source for the user's later Windows validation; no VM runs.


Phase 6 pre-gate review: all eight deliberate overlays rejected; named benchmark
and new guard inventory retained. Full favthumbs race suite passes (1.888s), grid
race suite passes (2.811s), grouping/delivery selection passes (1.529s). Expanded
command/hide selection passed UI 5.864s and grid 1.149s. Native full affected
packages: favthumbs 0.741s, grid 0.933s and menus 0.524s pass; UI 49.098s fails
only the previously recorded TestE2E_CopySelection macOS golden mismatch. No
golden changed; canonical Linux gate must pass that case. Final corrected before
benchmark passes 72.529s; after passes 97.615s. P2 cold median p95 204.5 -> 63.16
ms, foreground throughput 6.637 -> 16.27/s, preview convergence 2176 -> 4084 ms;
all measured policy targets pass. P18 retains four workers and comparable
results. Retained evidence: 28-29-preview-contention.md, 24-progressive-hide.md,
30-command-matrix.md and phase6-* records.

Production source frozen for the shared make verify gate. Formatting/Qodana
preflight and diff whitespace pass. Same temporary Docker GOGC=25 wrapper, full
race inventory and package concurrency retained. Root UI inventory is 667
(212/225/230); the Help test expanded existing subtests, two new top-level tests
were assigned ui-3. No benchmark runs inside race CI. Reviewed Windows ordinary
and Store amd64/arm64 artifacts rebuilt successfully in the disposable Phase 6
copy; application source bytes match the reviewed files. All four binaries plus
fixtures/SHA256SUMS are copied to ignored bin/maintainability-validation for the
user's later native testing. Architecture/build tags/source hashes and packaging
provenance are retained. No VM execution or SDK/trust changes.


Phase 6 final gate: make verify PASS, process exit 0. Formatting, offline TUF,
Qodana exclusions, vet/build, exact 667-test root UI inventory and all canonical
Linux/amd64 race partitions pass. The Copy Selection golden passes in its
canonical environment; the previously recorded macOS-only mismatch remains in
the native output. Source freeze ends. Tickets 24 (progressive follow-up), 28,
29 and 30 are resolved. Ticket 27 awaits user Windows testing; 31/32 remain
unactivated accepted watches. No commits or Windows VM actions were made.

Phase 6 cost ledger: one read-only T3 Scout (budget 1 / actual 1), zero delegated
implementations/reviews/fixes. T0 completed review and the user-reported grouping
fix inline. One shared full make verify gate; focused red/green and eight negative
overlays are retained. Three-sample final before/after benchmarks met the measured
policy targets. A preliminary incorrect parent-scoped CPU setup was rejected and
corrected before accepting measurements.

Terminal package results:

```text
package	partition=non-ui	package=github.com/frathe/picfetch/internal/decodepool	action=pass	elapsed=1.068
package	partition=non-ui	package=github.com/frathe/picfetch/internal/distribution	action=pass	elapsed=1.072
package	partition=non-ui	package=github.com/frathe/picfetch/internal/clipboard	action=pass	elapsed=1.114
package	partition=non-ui	package=github.com/frathe/picfetch/internal/appearance	action=pass	elapsed=1.149
package	partition=non-ui	package=github.com/frathe/picfetch/internal/displays	action=pass	elapsed=1.149
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/assets	action=skip	elapsed=0.002
package	partition=non-ui	package=github.com/frathe/picfetch/internal/filemanager	action=pass	elapsed=1.154
package	partition=non-ui	package=github.com/frathe/picfetch/internal/completion	action=pass	elapsed=1.161
package	partition=non-ui	package=github.com/frathe/picfetch/internal/favstore	action=pass	elapsed=1.189
package	partition=non-ui	package=github.com/frathe/picfetch/internal/launch	action=pass	elapsed=1.197
package	partition=non-ui	package=github.com/frathe/picfetch/internal/filesort	action=pass	elapsed=1.214
package	partition=non-ui	package=github.com/frathe/picfetch	action=pass	elapsed=1.230
package	partition=non-ui	package=github.com/frathe/picfetch/internal/filescan	action=pass	elapsed=1.237
package	partition=non-ui	package=github.com/frathe/picfetch/internal/dupes	action=pass	elapsed=1.267
package	partition=non-ui	package=github.com/frathe/picfetch/internal/filepicker	action=pass	elapsed=1.399
package	partition=non-ui	package=github.com/frathe/picfetch/internal/favthumbs	action=pass	elapsed=1.425
package	partition=non-ui	package=github.com/frathe/picfetch/internal/openwith	action=pass	elapsed=1.128
package	partition=non-ui	package=github.com/frathe/picfetch/internal/selection	action=pass	elapsed=1.053
package	partition=non-ui	package=github.com/frathe/picfetch/internal/preferences	action=pass	elapsed=1.132
package	partition=non-ui	package=github.com/frathe/picfetch/internal/session	action=pass	elapsed=1.120
package	partition=non-ui	package=github.com/frathe/picfetch/internal/trash	action=pass	elapsed=1.089
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/display	action=pass	elapsed=1.144
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/infoview	action=pass	elapsed=1.146
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/deletion	action=pass	elapsed=1.394
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/menus	action=pass	elapsed=1.180
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/autoupdate	action=pass	elapsed=1.484
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/slideshow	action=pass	elapsed=1.135
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/spiral	action=pass	elapsed=1.187
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/zoom	action=pass	elapsed=1.188
package	partition=non-ui	package=github.com/frathe/picfetch/internal/wincom	action=pass	elapsed=1.129
package	partition=non-ui	package=github.com/frathe/picfetch/internal/wallpaper	action=pass	elapsed=1.172
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/copyselection	action=pass	elapsed=2.647
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/grid	action=pass	elapsed=2.761
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/widgets	action=pass	elapsed=1.779
package	partition=non-ui	package=github.com/frathe/picfetch/internal/uitest	action=pass	elapsed=1.623
package	partition=non-ui	package=github.com/frathe/picfetch/internal/wingesture	action=pass	elapsed=1.087
package	partition=non-ui	package=github.com/frathe/picfetch/internal/winpos	action=pass	elapsed=1.200
package	partition=non-ui	package=github.com/frathe/picfetch/scripts/nativeguards	action=pass	elapsed=1.100
package	partition=non-ui	package=github.com/frathe/picfetch/scripts/plistdoctypes	action=pass	elapsed=1.128
package	partition=non-ui	package=github.com/frathe/picfetch/scripts/releasenotes	action=pass	elapsed=1.114
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/settingswin	action=pass	elapsed=2.920
package	partition=non-ui	package=github.com/frathe/picfetch/scripts/synctuf	action=pass	elapsed=1.179
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/favorites	action=pass	elapsed=4.510
package	partition=non-ui	package=github.com/frathe/picfetch/scripts/wingettag	action=pass	elapsed=2.021
package	partition=non-ui	package=github.com/frathe/picfetch/internal/update	action=pass	elapsed=4.086
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/exifwin	action=pass	elapsed=6.098
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/mosaicwin	action=pass	elapsed=7.522
package	partition=non-ui	package=github.com/frathe/picfetch/scripts/msixstage	action=pass	elapsed=6.993
package	partition=non-ui	package=github.com/frathe/picfetch/scripts/testshards	action=pass	elapsed=13.855
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/compare	action=pass	elapsed=32.323
package	partition=non-ui	package=github.com/frathe/picfetch/internal/imaging	action=pass	elapsed=34.065
package	partition=non-ui	package=github.com/frathe/picfetch/internal/mosaic	action=pass	elapsed=39.231
package	partition=non-ui	package=github.com/frathe/picfetch/internal/ui/help	action=pass	elapsed=49.341
package	partition=ui-3	package=github.com/frathe/picfetch/internal/ui	action=pass	elapsed=299.333
package	partition=ui-2	package=github.com/frathe/picfetch/internal/ui	action=pass	elapsed=302.810
package	partition=ui-1	package=github.com/frathe/picfetch/internal/ui	action=pass	elapsed=324.193
```
