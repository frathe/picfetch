# PicFetch — Refactoring Backlog

Updated by the project-wide maintainability audit, 2026-09-06.

Status: documentation-only audit complete; all findings remain open. Implementation is proposed separately in [the phased plan](plans/2026-09-06-maintainability-plan.md). [Validation evidence and reproducible probes](plans/2026-09-06-maintainability-validation.md) accompany this report.

## Baseline and scope

Audited commit: `2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f` ("housekeeping"), detached HEAD in the isolated audit worktree. The checkout was clean before the audit. Code links below are pinned to that revision so subsequent edits do not move the evidence.

The saved main checkout had the same HEAD but uncommitted XRandR extraction work: modified `internal/displays/linux.go`, renamed/modified `linux_test.go` to `xrandr_test.go`, new `xrandr.go`, and modified `qodana.yaml`. Those edits and other worktrees are excluded. This report does not describe uncommitted work elsewhere as reviewed.

The lead read the architecture, domain vocabulary, existing backlog, and working agreement, reviewed UI/composition and cross-feature flows, and consolidated two explicitly requested read-only component reviews: imaging/data, and platform/update/tooling. The lead checked the reported source locations and independently re-ran the component probes, clipboard fault probe, and UI reproductions. This update preserves the existing canonical filename, `needs_refactoring.md`, and reconciles its previous entries below. No production code, test source, dependency, workflow, or application data was changed. A later explicit delivery instruction authorizes a local documentation-only commit; publication, merge, and release remain outside this step.

## Assessment and ranking

The package structure is broadly sound. The highest-value work concerns contracts that cross existing boundaries: validated byte spans, bounded scratch memory, generation-aware result publication, and the distinction between queued UI work and completed UI work. The evidence does not justify a project-wide redesign or a generic controller framework.

There are **24 findings/watch items: 2 P1, 15 P2, 7 P3**. IDs are stable audit identifiers, not implementation order within a phase. Severity weighs consequence and plausible trigger; confidence separately records strength of evidence.

| Priority | Meaning |
| --- | --- |
| P0 critical | Demonstrated widespread irreversible loss or comparable emergency. None established. |
| P1 high | Reachable process crash or severe resource exhaustion from a supported input path. Address first. |
| P2 medium | Incorrect state/data/paths, concurrency or responsiveness failure under a specific workflow, or a material verification/lifecycle gap. |
| P3 low | Localized freshness, optional-check, or reproducibility debt with limited immediate impact. |

“Reproduced” means the behavior was observed with a controlled probe. “Confirmed contract/risk” means the implementation establishes the gap, while the described real-world timing or workload was not exercised. Passing baseline tests do not repair or disprove either category.

| ID | Priority | Finding |
| --- | --- | --- |
| [MA-001](#ma-001) | P1 | Malformed EXIF offsets can panic the process |
| [MA-002](#ma-002) | P1 | Mosaic scratch allocation ignores the visible intersection |
| [MA-003](#ma-003) | P2 | Worker logic assumes UI dispatch has already completed |
| [MA-004](#ma-004) | P2 | Deletion confirmation retains indices from an earlier order |
| [MA-005](#ma-005) | P2 | Native chooser transport changes legal POSIX paths |
| [MA-006](#ma-006) | P2 | Comparison admits incomplete records to the shared image cache |
| [MA-007](#ma-007) | P2 | Save Changes leaves rotated EXIF dimensions contradictory |
| [MA-008](#ma-008) | P2 | Duplicate grouping is quadratic and re-enters the UI path |
| [MA-009](#ma-009) | P2 | Old thumbnail workers can publish facts into a new generation |
| [MA-010](#ma-010) | P2 | Ancillary image APIs discard caller cancellation |
| [MA-011](#ma-011) | P2 | Clipboard temp-file cleanup hides the original failure |
| [MA-012](#ma-012) | P2 | Windows copied-file lists cross an unspecified encoding boundary |
| [MA-013](#ma-013) | P2 | Trash workaround overrides legitimate XDG configuration |
| [MA-014](#ma-014) | P2 | Several image actions still do expensive work on UI |
| [MA-015](#ma-015) | P2 | Map tile concurrency and failure state lack a shared bound |
| [MA-016](#ma-016) | P2 | Position-poller stop does not establish worker completion |
| [MA-017](#ma-017) | P2 | Release gates do not execute several platform and renderer contracts |
| [MA-018](#ma-018) | P3 | Favorite-preview identity truncates available timestamp precision |
| [MA-019](#ma-019) | P3 | Optional HEIC leak check underflows on declining RSS |
| [MA-020](#ma-020) | P3 | Packaging tool versions float outside the reviewed module graph |
| [MA-021](#ma-021) | P3 | Favorite prewarm scheduling |
| [MA-022](#ma-022) | P3 | Distributed mode precedence |
| [MA-023](#ma-023) | P3 | HEIC fork retirement (accepted watch) |
| [MA-024](#ma-024) | P3 | Updater dependency footprint (accepted watch) |

<a id="ma-001"></a>

## MA-001 — Malformed EXIF offsets can panic the process

**P1 · Reproduced defect. Confidence: High.**

Locations: [internal/imaging/exififd.go:13-28](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/exififd.go#L13-L28); [internal/imaging/exif.go:148-168](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/exif.go#L148-L168); [internal/imaging/raw.go:150-161](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/raw.go#L150-L161); [internal/imaging/jpegexif.go:275-283](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/jpegexif.go#L275-L283).

**Evidence and scenario.** The TIFF header bytes `49 49 2a 00 ff ff ff ff` carry an IFD offset of `0xffffffff`. Adding 2 in uint32 wraps to 1, passes the length guard, then panics while slicing. External test overlays reproduced the panic in `ReadMetadata`, `parseExifOrientation`, and `nextIFDOffset`: `slice bounds out of range [4294967295:1]`. The JPEG orientation path also consumes TIFF data from EXIF.

**Impact.** A malformed metadata block can terminate a loader worker and therefore the application. The loader's tolerant error policy is bypassed. This establishes a crash, not arbitrary code execution.

**Recommended direction.** Use checked, widened offset arithmetic before every header/entry/value span calculation. The writer already demonstrates a safer uint64 pattern. Share the arithmetic contract across readers without forcing readers and writers to share malformed-block policy. Add both-endian, root/sub-IFD, near-end, overflow and truncated-value regressions; fuzz metadata and orientation entry points for no panics.

<a id="ma-002"></a>

## MA-002 — Mosaic scratch allocation ignores the visible intersection

**P1 · Reproduced allocation geometry; OOM not attempted. Confidence: High.**

Locations: [internal/mosaic/layout.go:114-121](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/mosaic/layout.go#L114-L121); [internal/mosaic/render.go:23-27](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/mosaic/render.go#L23-L27); [internal/mosaic/render.go:94-120](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/mosaic/render.go#L94-L120); [internal/mosaic/generator.go:72-80](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/mosaic/generator.go#L72-L80).

**Evidence and scenario.** `prepareSourceLayer` doubles the entire placement rectangle before clipping it to the destination. A geometry-only probe using a 1920x1080 target, defaults and seed 42 produced first-layer requests of 92,425,480 bytes at aspect 100, 924,097,660 bytes at aspect 1000, and 9,240,821,400 bytes at aspect 10000. No large allocation was executed. A 10000x1 source itself is small enough to satisfy the ordinary decoded-pixel guard.

**Impact.** A modest valid input can request gigabytes of transient memory for an approximately 8 MB output, outside the bounded repeat cache. Host memory determines whether the result is severe pressure or process termination.

**Recommended direction.** Prepare only the source region needed for the visible intersection, or tile it under an explicit scratch-memory budget that includes resampling/masks. Preserve placement, crop, rotation and edge quality; do not fix this by distorting source aspect ratios. Validate both wide and tall sources and cancellation before expensive preparation.

<a id="ma-003"></a>

## MA-003 — Worker logic assumes UI dispatch has already completed

**P2 · Queued-dispatch reproduction plus static race evidence. Confidence: High.**

Locations: [internal/ui/load.go:488-522](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/load.go#L488-L522); [internal/ui/slideshow/slideshow.go:309-333](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/slideshow/slideshow.go#L309-L333); [internal/ui/openfiles.go:33-58](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/openfiles.go#L33-L58); [internal/ui/compare.go:21-30](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/compare.go#L21-L30).

**Evidence and scenario.** `animate` reads `idx` on the worker, updates it inside `fyne.Do`, and reads `stale` immediately after enqueueing its writer. A controlled queued driver observed the next 1-second delay being scheduled before the pending callback could advance to the frame with a 2-second delay. Slideshow uses the same cross-goroutine `stale` local. Separately, `openFileDialog` starts a worker whose `runFileChooser` repeats a comparison-state/UI-toast guard directly on that worker. Fyne v2.8.0 `Do` dispatches without waiting; its test driver executes inline.

**Impact.** Native asynchronous execution exposes unsynchronized state access and incorrect animation pacing hidden by ordinary software-driver tests. The queued timing mismatch was reproduced; a native GL race run was not performed.

**Recommended direction.** Assign frame index ownership to one goroutine and acknowledge queued work only when required by pacing. Check request staleness within UI callbacks. Keep chooser admission on UI and revalidate results there. Add per-instance controllable queue tests for animation/slideshow/chooser; use cancellable acknowledgement where necessary, avoiding shutdown waits that block the event loop.

<a id="ma-004"></a>

## MA-004 — Deletion confirmation retains indices from an earlier order

**P2 · Reproduced defect. Confidence: High.**

Locations: [internal/ui/deletion/deletion.go:158-174](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/deletion/deletion.go#L158-L174); [internal/ui/deletion/deletion.go:232-266](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/deletion/deletion.go#L232-L266); [internal/ui/sort.go:45-57](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/sort.go#L45-L57).

**Evidence and scenario.** Target URI/index pairs are captured when the prompt opens, but the protecting generation is captured only when deletion starts. In the probe, prompt for A at index 0, reorder `[A,B]` to `[B,A]` and increment generation, then confirm. A is correctly removed from temporary disk storage, but the list removes B and retains missing A. An already-running sort can publish its reorder while the prompt is open.

**Impact.** The file list disagrees with the actual deletion and can appear to lose another item. The reproduction does not show the wrong disk file being deleted: the captured URI remains correct.

**Recommended direction.** Reconcile successful deletions by stable URI identity in the current file set, or capture and enforce the prompt generation before interpreting indices. Define changes both while the prompt is open and while trash work runs. Copy target slices at the boundary. Test reorder, fresh drop, partial failure and overlapping confirmations with stubbed trash operations.

<a id="ma-005"></a>

## MA-005 — Native chooser transport changes legal POSIX paths

**P2 · Confirmed parser-contract defect; native overwrite not exercised. Confidence: High for parsing; medium for platform consequence.**

Locations: [internal/filepicker/darwin.go:28-32](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/filepicker/darwin.go#L28-L32); [internal/filepicker/darwin.go:50-53](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/filepicker/darwin.go#L50-L53); [internal/filepicker/filepicker.go:173-186](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/filepicker/filepicker.go#L173-L186); [internal/ui/export.go:179-185](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/export.go#L179-L185).

**Evidence and scenario.** Open results are newline-joined; the common parser splits on newline and trims trailing CR/LF. A legal path `/tmp/a\nb.png` becomes two paths. Even the single save-panel result goes through this parser, and export takes the first URI. A trailing CR in a filename is also discarded.

**Impact.** The path acted on can differ from the one confirmed in the native panel. For save, the truncated destination could be a different existing file whose overwrite was never confirmed. That overwrite scenario follows from the path flow; the audit did not execute it.

**Recommended direction.** Return structured paths from native adapters and distinguish one save destination from multi-open results. Choose an unambiguous subprocess transport supported by each backend; handle unsupported filenames explicitly rather than silently changing them. Add newline, trailing-CR, whitespace, Unicode, cancellation and multiple-selection round trips.

<a id="ma-006"></a>

## MA-006 — Comparison admits incomplete records to the shared image cache

**P2 · Reproduced defect. Confidence: High.**

Locations: [internal/ui/compare.go:60-74](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/compare.go#L60-L74); [internal/ui/load.go:144-147](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/load.go#L144-L147); [internal/ui/load.go:273](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/load.go#L273); [internal/ui/load.go:439-440](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/load.go#L439-L440).

**Evidence and scenario.** Comparison reads and decodes the image then caches it without assigning `FileSize` or `HasEXIF`, unlike foreground load and preload. A GPS JPEG probe verified the shared cached object had `HasEXIF=false` and `FileSize=0` instead of 739.

**Impact.** Opening an image through comparison first changes the information later shown by normal navigation: byte size is zero and the EXIF affordance is absent. Repeated cache-admission assembly has allowed an invariant to drift.

**Recommended direction.** Create one canonical complete LoadedImage construction step for these callers, while keeping foreground versus speculative cache admission explicit. Assert all three paths produce equivalent metadata, preserve cancellation checks, and retain `Add` versus `AddIfFits` semantics.

<a id="ma-007"></a>

## MA-007 — Save Changes leaves rotated EXIF dimensions contradictory

**P2 · Reproduced defect with historical intent noted. Confidence: High.**

Locations: [internal/imaging/save.go:109-122](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/save.go#L109-L122); [internal/imaging/save.go:211-216](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/save.go#L211-L216); [internal/imaging/save.go:248-257](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/save.go#L248-L257); [internal/imaging/save_test.go:1268-1288](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/save_test.go#L1268-L1288).

**Evidence and scenario.** A 90x60 fixture saved with rotated 60x90 pixels produced a 60x90 JPEG header and IFD0 width=90, height=60. SaveRotated intentionally passes zero corrected dimensions; Export now corrects equivalent dimension changes. The existing rotation test checks tag presence, not their numeric correctness.

**Impact.** Consumers choosing EXIF dimensions disagree with consumers reading the encoded image. This was an earlier deliberate policy, but the resulting metadata is false; changing it should explicitly revise that policy and its test.

**Recommended direction.** Reuse the existing dimension-invalidated logic for SaveRotated, preserving correctable dimension tags and removing geometry that cannot be made true. Check numeric values for quarter turns, EXIF orientation 5-8, and unchanged geometry; retain unrelated metadata, DPI and color profiles.

<a id="ma-008"></a>

## MA-008 — Duplicate grouping is quadratic and re-enters the UI path

**P2 · Measured scalability risk. Confidence: High.**

Locations: [internal/imaging/dhash.go:124-166](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/dhash.go#L124-L166); [internal/dupes/groups.go:48-99](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/dupes/groups.go#L48-L99); [internal/ui/grid/search.go:96-120](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/grid/search.go#L96-L120); [internal/ui/grid/dupes.go:189-203](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/grid/dupes.go#L189-L203); [internal/filescan/filescan.go:34](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/filescan/filescan.go#L34).

**Evidence and scenario.** Unrelated hashes require approximately n(n-1)/2 comparisons. Fresh native probes measured 35 ms for 10,000, 141 ms for 20,000 and 857 ms for 50,000 hashes. The default scan cap is 200,000, corresponding to about 20 billion pair comparisons, not a measured duration. Search calls `rebuildFilter` then `rebuildGroups` for each edit even when hashes/distance have not changed; duplicate browse can rebuild twice. `Compute` cannot be cancelled.

**Impact.** Large warm folders can stall typing or settings changes despite the separate background hash pipeline. Superseded computations continue consuming CPU. Timings are hardware-specific non-race measurements.

**Recommended direction.** Cache grouping by file generation/hash revision/distance; filter names against the accepted snapshot. Compute changed groups off UI, cancel/coalesce superseded work, and publish only current snapshots. Benchmark indexing only after this separation. Preserve the current greedy complete-linkage grouping and representative rules; transitive connected components are not equivalent.

<a id="ma-009"></a>

## MA-009 — Old thumbnail workers can publish facts into a new generation

**P2 · Confirmed generation-contract gap; coordinated race scenario not run. Confidence: High for code path; medium for user impact.**

Locations: [internal/ui/grid/hashengine.go:164-201](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/grid/hashengine.go#L164-L201); [internal/dupes/dupes.go:171-207](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/dupes/dupes.go#L171-L207); [internal/ui/grid/thumbs.go:257-278](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/grid/thumbs.go#L257-L278).

**Evidence and scenario.** Hash workers check generation before loading, then call `PutHash`, `PutFailed` and `PutNativeSize` after loading. Those model methods adopt the file set's current generation. A drop/reset during the read can therefore label old results as current, including same-URI replacements. The thumbnail path also stores facts before its UI-install staleness guard.

**Impact.** Duplicate groups/native dimensions may use obsolete pixels, or an old failure can suppress retries in a fresh set. Checking only the final widget callback does not protect shared model/cache writes.

**Recommended direction.** Make fact publication conditional on the generation captured when work began, checked atomically with model mutation. Decide explicitly which cache entries can survive a file-set reorder versus source replacement. Add a gated decoder test: pause old read, replace/reset the same URI, complete old success/failure, and verify current facts remain untouched.

<a id="ma-010"></a>

## MA-010 — Ancillary image APIs discard caller cancellation

**P2 · Confirmed cancellation-contract gap. Confidence: High.**

Locations: [internal/imaging/loader.go:297-304](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/loader.go#L297-L304); [internal/imaging/thumbnail.go:42-69](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/thumbnail.go#L42-L69); [internal/filesort/filesort.go:143-149](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/filesort/filesort.go#L143-L149); [internal/favthumbs/sync.go:116-135](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/favthumbs/sync.go#L116-L135); [internal/ui/grid/hashengine.go:195-201](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/grid/hashengine.go#L195-L201).

**Evidence and scenario.** CaptureDate and thumbnail helpers use `context.Background()`. Capture-date sorting checks cancellation between files, but an in-flight call reads the whole source; the encoded input default permits up to 512 MiB. Favorite preview cancellation must wait for already-running uncancellable thumbnail reads, and grid native-size fallback also drops context.

**Impact.** New work can remain behind obsolete reads on large files or slow/network storage. Outer cancellation tokens communicate a stronger promise than the nested operations honor.

**Recommended direction.** Add context-bearing ancillary APIs and thread existing sort/pass/generation contexts through read/probe/decode boundaries. Describe limits where a third-party decoder cannot interrupt in-memory work. Test a reader paused between chunks, cancellation before acquiring a slot, and completion of cancelled passes; avoid sleep-based timing assertions.

<a id="ma-011"></a>

## MA-011 — Clipboard temp-file cleanup hides the original failure

**P2 · Reproduced defect. Confidence: High.**

Locations: [internal/clipboard/clipboard.go:42-55](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/clipboard/clipboard.go#L42-L55); [internal/clipboard/copyfiles.go:96-105](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/clipboard/copyfiles.go#L96-L105).

**Evidence and scenario.** The write/close error is shadowed by `os.Remove`'s result and that cleanup result is returned. An isolated subprocess executing the exact function with RLIMIT_FSIZE=1 returned `path="" err=<nil>` when writing failed and removal succeeded. Stdout was piped to an unrestricted parent so the failure limit did not truncate the evidence log.

**Impact.** Clipboard callers proceed with an empty filename and the original disk error disappears. The neighboring file-list helper already preserves the original error correctly.

**Recommended direction.** Return the primary write/close error and optionally join a cleanup error. Add fault coverage through a narrow writer seam or isolated subprocess, ensuring no desktop clipboard mutation occurs. This is a small independent fix.

<a id="ma-012"></a>

## MA-012 — Windows copied-file lists cross an unspecified encoding boundary

**P2 · Source/documentation-confirmed mismatch; Windows runtime untested. Confidence: High.**

Locations: [internal/clipboard/copyfiles.go:89-96](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/clipboard/copyfiles.go#L89-L96); [internal/clipboard/copyfiles.go:131-139](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/clipboard/copyfiles.go#L131-L139); [internal/clipboard/copyfiles_test.go:125-168](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/clipboard/copyfiles_test.go#L125-L168).

**Evidence and scenario.** Go writes BOM-less UTF-8 paths, while the generated Windows `powershell` script uses `Get-Content` without `-Encoding`. Microsoft documents that Windows PowerShell reads BOM-less text using the active ANSI code page. Paths such as `C:\Fotos\München.jpg` therefore fail on conventional non-UTF-8 code pages. Existing tests inspect the script/UTF-8 bytes using ASCII fixtures rather than execute the boundary. Source: [Microsoft character encoding documentation](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_character_encoding?view=powershell-7.6), Windows PowerShell section, checked 2026-09-06.

**Impact.** Batch copy can fail to resolve the files the user selected. This depends on Windows PowerShell/code-page configuration; it is not a claim about PowerShell 7's different default.

**Recommended direction.** Specify UTF-8 at the consumer and test actual Windows PowerShell list decoding with accented, CJK and emoji filenames while stubbing clipboard mutation. Inspect chooser stdout encoding when touching native path transport, but do not assume that separate boundary has the same defect.

<a id="ma-013"></a>

## MA-013 — Trash workaround overrides legitimate XDG configuration

**P2 · Confirmed conditional configuration defect. Confidence: High.**

Locations: [internal/trash/trash.go:67-92](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/trash/trash.go#L67-L92); [internal/wallpaper/wallpaper.go:135-141](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/wallpaper/wallpaper.go#L135-L141); [internal/wallpaper/wallpaper.go:177-178](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/wallpaper/wallpaper.go#L177-L178).

**Evidence and scenario.** Trash handling replaces XDG_DATA_HOME unconditionally with `$HOME/.local/share` as a sandbox-launcher workaround. That also overwrites a deliberate ordinary user setting. The wallpaper package already distinguishes ordinary overrides from identified snap redirection.

**Impact.** Files can land in a Trash directory different from the one the user's desktop file manager displays, reproducing the invisible-trash behavior the workaround was meant to prevent.

**Recommended direction.** Preserve ordinary XDG overrides and normalize only demonstrated sandbox redirection, or resolve host configuration explicitly. Test default, legitimate override and sandbox cases. Keep package-specific environment semantics unless a shared contract is demonstrated.

<a id="ma-014"></a>

## MA-014 — Several image actions still do expensive work on UI

**P2 · Confirmed synchronous call paths; latency not benchmarked. Confidence: High for placement; medium for workload impact.**

Locations: [internal/ui/clipboard.go:38-55](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/clipboard.go#L38-L55); [internal/ui/save.go:46-60](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/save.go#L46-L60); [internal/ui/exifwin/exifwin.go:263-290](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/exifwin/exifwin.go#L263-L290); [internal/ui/exifwin/exifwin.go:391-412](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/exifwin/exifwin.go#L391-L412).

**Evidence and scenario.** Clipboard PNG encoding happens before the worker starts. Save Changes directly calls SaveRotated, and EXIF refresh reads/probes the full file with Background context while strip directly rewrites it. These actions originate on UI, unlike the established background export/load paths.

**Impact.** Large images and slow storage can freeze input/redraw. Merely moving the OS clipboard command to a goroutine does not remove the preceding encoding stall. No interactive latency threshold was measured in this audit.

**Recommended direction.** Move reads/encoding into owning feature workers using immutable source snapshots and current-request completion checks. Serialize mutations of the same source, define close/navigation behavior and retain atomic-write guarantees; blindly moving file writes to unrelated goroutines would create new races. Reuse the cancellation contracts from MA-010 and test observable completion and errors.

<a id="ma-015"></a>

## MA-015 — Map tile concurrency and failure state lack a shared bound

**P2 · Confirmed resource-lifetime risk; stress/network run omitted. Confidence: High for structure; medium for workload impact.**

Locations: [internal/ui/exifwin/tiles.go:206-215](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/exifwin/tiles.go#L206-L215); [internal/ui/exifwin/tiles.go:236-275](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/exifwin/tiles.go#L236-L275); [internal/ui/exifwin/tiles.go:337-371](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/exifwin/tiles.go#L337-L371); [internal/ui/exifwin/exifwin.go:504-543](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/exifwin/exifwin.go#L504-L543).

**Evidence and scenario.** RoundTrip starts a goroutine per newly claimed URL. Warm creates its own four-slot semaphore for each call; concurrent warm passes do not share that cap. Changing location invalidates UI application by warmGen but does not cancel old fetches. Failed URL timestamps persist until that URL later succeeds; the 16 MiB byte-cache bound does not bound this metadata.

**Impact.** Rapid navigation/panning or prolonged tile failure can accumulate outstanding requests and failure entries for the window lifetime. The HTTP timeout bounds an individual request, not aggregate concurrency.

**Recommended direction.** Use a per-fetcher bounded worker/queue and cancellation for superseded warm passes/window close, with a bounded or expiring failure set. Give all workers and onChange callbacks an observable completion contract. Preserve request deduplication and cache behavior; verify with a controllable local server, not public tile traffic.

<a id="ma-016"></a>

## MA-016 — Position-poller stop does not establish worker completion

**P2 · Confirmed lifecycle-contract gap; native shutdown hang unverified. Confidence: High for contract; medium for consequence.**

Locations: [internal/winpos/poll.go:70-102](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/winpos/poll.go#L70-L102); [internal/winpos/poll_test.go:16-71](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/winpos/poll_test.go#L16-L71); [internal/ui/windowtrack.go:55-58](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/windowtrack.go#L55-L58); [internal/ui/run.go:111-119](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/run.go#L111-L119).

**Evidence and scenario.** Stop closes a channel but does not acknowledge worker exit. A worker already waiting in `fyne.DoAndWait` cannot observe that cancellation; its queued callback does not recheck it. A ready tick can also win against cancellation. Tests use non-native windows, which return a no-op stop and never exercise the worker.

**Impact.** A callback/native read can remain queued after stop returns, and a worker can await an event loop that has ended. This contradicts the documented stop guarantee. No actual native shutdown hang was reproduced; slideshow.Active is atomic and is not implicated as a race.

**Recommended direction.** Make queued reads cancel-aware and expose worker completion. Arrange shutdown while the event loop can still drain callbacks; never wait on UI for work that needs UI to finish. Test stop before enqueue, after enqueue and during callback using an instance-owned queue/native-read seam.

<a id="ma-017"></a>

## MA-017 — Release gates do not execute several platform and renderer contracts

**P2 · Confirmed coverage gap. Confidence: High.**

Locations: [.github/workflows/ci.yml:57-72](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/.github/workflows/ci.yml#L57-L72); [.github/workflows/ci.yml:138-160](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/.github/workflows/ci.yml#L138-L160); [.github/workflows/release.yml:19-50](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/.github/workflows/release.yml#L19-L50); [internal/wallpaper/target_windows_test.go:1-35](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/wallpaper/target_windows_test.go#L1-L35); [main_darwin_test.go:23-28](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/main_darwin_test.go#L23-L28); [internal/distribution/distribution_store_test.go:1-11](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/distribution/distribution_store_test.go#L1-L11); [internal/ui/compare/compare.go:116-118](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/compare/compare.go#L116-L118); [internal/ui/compare/renderer.go:99-103](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/compare/renderer.go#L99-L103).

**Evidence and scenario.** Windows CI tests only update/autoupdate, omitting existing wallpaper native guards. macOS packaging does not run the Objective-C class-graft test. Ordinary CI does not select the microsoftstore-tagged distribution test. Comparison production uses the shader renderer while deterministic unit tests use a canvas reference adapter. Windows cross-vet/build and Linux golden/race checks do exist.

**Impact.** Green gates do not establish that these existing regression guards pass or that the production GPU output matches the reference. This is a verification gap, not evidence of a current shader or platform failure.

**Recommended direction.** Add focused native macOS/Windows and Store-tag checks, enumerate the tests selected, and establish a bounded GL smoke/fidelity procedure for comparison. Keep the deterministic Linux/reference renderer tests. A full golden suite on every OS is unnecessary; exercise the actual native boundary that each guard protects.

<a id="ma-018"></a>

## MA-018 — Favorite-preview identity truncates available timestamp precision

**P3 · Reproduced cache defect. Confidence: High.**

Locations: [internal/favthumbs/name.go:41-52](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/favthumbs/name.go#L41-L52).

**Evidence and scenario.** The entry key uses `info.ModTime().Unix()`. Replacing same-size bytes at one path with mtimes 1700000000.100000000 and 1700000000.900000000 produced identical keys in a filesystem probe.

**Impact.** A rapid edit/replace can retain an old favorite preview until size or whole-second timestamp changes. This is limited to preview freshness.

**Recommended direction.** Use nanosecond modification time in version identity; treat old entries as cache misses/sweepable files. Add a same-size subsecond replacement test. Do not promise to detect an external editor preserving both exact timestamp and size.

<a id="ma-019"></a>

## MA-019 — Optional HEIC leak check underflows on declining RSS

**P3 · Statically confirmed arithmetic defect. Confidence: High.**

Locations: [internal/imaging/rss_heicleak_linux.go:11-24](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/rss_heicleak_linux.go#L11-L24); [internal/imaging/heic_leak_test.go:50-59](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/imaging/heic_leak_test.go#L50-L59).

**Evidence and scenario.** RSS samples are uint64 and `growth := rssAfter - rssMid` subtracts without first comparing. If reclamation reduces RSS, the result wraps to a huge apparent increase.

**Impact.** The optional `heicleak`/PICFETCH_HEIC_LEAK_TEST check can falsely report a leak. This does not affect ordinary CI and does not challenge the documented decoder leak-fix pin. No Linux RSS experiment was run.

**Recommended direction.** Compare samples before subtracting or calculate a safe signed delta; cover increasing/equal/decreasing arithmetic. Keep the actual long-running RSS check opt-in and preserve the HEIC fork until an appropriate official release is verified.

<a id="ma-020"></a>

## MA-020 — Packaging tool versions float outside the reviewed module graph

**P3 · Confirmed reproducibility risk. Confidence: High.**

Locations: [.github/workflows/release.yml:40](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/.github/workflows/release.yml#L40); [.github/workflows/release.yml:71](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/.github/workflows/release.yml#L71); [.github/workflows/microsoft-store.yml:30](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/.github/workflows/microsoft-store.yml#L30); [Makefile:9](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/Makefile#L9); [Makefile:368-388](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/Makefile#L368-L388).

**Evidence and scenario.** Packaging tools are installed with `@latest`; the Windows cross-image default is a mutable tag, and Linux packaging relies on the cross-tool's default image. These inputs are independent of the version-pinned application module graph.

**Impact.** Rebuilding the same release commit can change packaging output or fail without a reviewed source/dependency change. No claim is made that any current external version is broken or insecure.

**Recommended direction.** Centralize reviewed tool versions/image references for local and CI packaging and print resolved versions in build logs. Upgrade deliberately with a packaging check. This is a reproducibility improvement, not a proposal for a dependency upgrade or release during this audit.

<a id="ma-021"></a>

## MA-021 — Favorite prewarming competes with interactive work

**P3 · Retained performance/design risk from old item 8. Confidence: high for scheduling policy; medium for user-visible contention.**

Locations: [internal/favthumbs/sync.go:13-24](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/favthumbs/sync.go#L13-L24), [internal/ui/favthumbs.go:98-113](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/favthumbs.go#L98-L113).

**Evidence and impact.** A favorite preview pass still visits every disk-cache miss with four independent workers while foreground grid/display work has separate pools. Store stops offering to the memory cache once full, preventing the old self-eviction churn, but does not stop disk-preview decoding. CPU/I/O contention and the assumption that the favorite's head deserves the memory budget remain. No contention benchmark was run. This is distinct from MA-010: cancellation helps obsolete passes, not a current pass competing with interaction.

**Direction.** After cancellation is fixed, measure foreground latency with a large cold favorite and test an explicit prewarm share, pause/yield policy, or disk-only threshold. Keep interactive worker capacity reserved; do not merge the pools into a semaphore that lets background work starve visible thumbnails. This remains an optional performance policy, not a proven functional defect.

<a id="ma-022"></a>

## MA-022 — Mode precedence remains distributed across command routes

**P3 · Retained maintainability risk from old item 9. Confidence: high for distribution; medium for future defect likelihood.**

Locations: [internal/ui/keys.go:137-302](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/keys.go#L137-L302), [internal/ui/menu.go:62-118](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/menu.go#L62-L118), [internal/ui/menus/menus.go:455-475](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/ui/menus/menus.go#L455-L475).

**Evidence and impact.** Escape precedence and per-key picture-frame/grid/inspect rules remain positional; comparison and Copy Selection have added admission rules across keyboard, menu wrappers and menu enabled-state calculation. The older extraction moved state ownership but did not eliminate this policy distribution. Existing tests cover many combinations, so the code's length is not itself a failure. MA-003 identifies a concrete related guard crossing the wrong goroutine; do not count that defect again here.

**Direction.** First capture a small command/mode acceptance matrix from current behavior, then centralize only repeated admission decisions in internal/ui when touching these routes. Preserve Escape precedence and explicit composition; avoid a global mode registry or shared feature controller. Validate menu/shortcut parity and intentional exceptions, including open/help during comparison.

<a id="ma-023"></a>

## MA-023 — HEIC fork retirement remains an accepted dependency watch

**P3 · Mitigated dependency debt from old item 1. Confidence: high for local mitigation and checked upstream metadata.**

Locations: [go.mod:11-20](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/go.mod#L11-L20), `AGENTS.md` HEIC decoder pin, and the optional leak check discussed in MA-019.

**Evidence and impact.** The old entry contradicted itself: it claimed there was no replace/mitigation, then noted the applied fix. The current module replaces upstream HEIC with `github.com/frathe/heic v0.0.0-20260820164529-0ac0a39f8206`. The unmitigated high-severity leak description is therefore obsolete for this build. Read-only GitHub metadata checked on 2026-09-06 listed v0.7.1 as the newest tag and [upstream PR 16](https://github.com/gen2brain/heic/pull/16) as open/unmerged; the latest-release endpoint returned 404. The fork still needs eventual retirement. This audit did not remeasure native RSS or claim the fork's entire decoder is leak-free.

**Direction.** Keep the pin and documented reason. At a dependency update, verify an official tag actually contains the fix, replace the fork only then, run format/decode regressions and the optional Linux RSS check after MA-019. No standing notification/monitor was created by this document.

<a id="ma-024"></a>

## MA-024 — Updater verification dependency footprint remains accepted

**P3 · Accepted watch item retained from old item 7 (mentioned in the previous introduction). Confidence: high for dependency boundary; unmeasured present cost.**

Locations: [go.mod:12-15](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/go.mod#L12-L15), [internal/update/attest.go:15-19](https://github.com/frathe/picfetch/blob/2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f/internal/update/attest.go#L15-L19).

**Evidence and impact.** Sigstore and TUF remain direct dependencies of the isolated update verifier, with a substantial transitive graph visible in go.mod. Signature/provenance verification is intentional. The older approximate dependency counts, binary/build-time implications and CVE-noise claims were not remeasured, so they are not presented as current quantified defects.

**Direction.** Retain the verifier boundary and security properties. At the next major verifier dependency upgrade, measure dependency reachability, binary size and build time; consider a smaller supported verification subset only if it preserves the trust/provenance contract. No dependency removal or custom cryptographic replacement is recommended now.

## Reconciliation of the previous backlog

The baseline canonical file had four detailed open entries (1, 8, 9, 10), plus an accepted updater watch and historical resolution statements in its introduction. All are accounted for here. The old relative priority formula and stale global health/coverage/line-count claims are replaced by the current severity/confidence rubric and measured validation record.

| Previous entry | Current disposition | Current-code evidence / reason |
| --- | --- | --- |
| 1: HEIC native leak, temporary fix | Materially updated and downgraded to MA-023; preserve fork watch | go.mod:20 has the patched fork, AGENTS.md records it. Remove the contradictory “no replace/no mitigation” claim; no current unmitigated leak was established. |
| 8: favorite prewarm | Retained as MA-021; cancellation portion cross-links MA-010 | syncConcurrency remains 4; gridSink stops memory offers at full budget but disk decode work continues. |
| 9: scattered mode guards | Retained as MA-022 | keys.go still has positional Escape/P/G/D rules; menu wrappers and comparison add policy routes. Existing coverage reduces urgency, not validity. |
| 10: accepted OS-seam coverage | Merged and materially updated to MA-017, with concrete native poller gap MA-016 | Old percentages are not current measurements. CI source establishes omitted native/Store tests; thin unavoidable native code remains acceptable. |
| 7: updater dependencies (intro watch) | Retained as MA-024 | Sigstore/TUF imports remain isolated in update/attest.go; remove unsupported old quantitative claims. |
| 2 and 4: inverted duplicate dependency / grid god object (already marked resolved) | Remain retired; no scored duplicate entry | internal/dupes owns model/visibility through immutable FileSet snapshots, internal/ui/visibility.go reads the model independently of grid. New grouping/generation defects are specifically MA-008/009. |
| 3: viewer god object (already retired) | Remains retired | Current viewer composes extracted display/settings/vector/update and feature state. Broad composition responsibility or field count alone is insufficient evidence to reopen it. |
| 5: manual menu recompute (already marked resolved) | Remains retired | internal/ui/menus.Menus.Apply computes from a State snapshot and returns changed; internal/ui/menu.go:226-229 refreshes only when necessary. MA-022 concerns admission policy, not stale per-item recompute. |
| 6: external appState eviction invariants (already marked resolved) | Remains retired | internal/ui/state.go:134-135 invokes onRemove; build.go:115 wires cache eviction once. MA-004 concerns prompt identity, not this eviction hook. |
| 11: twin display-index sentinels (recovered from git history) | Remains resolved | Only displayIndexOfHost remains in internal/ui/grid/search.go:180; its -1 sentinel is handled explicitly by callers. The old zero-returning twin is absent. |
| 13: empty internal/favorites directory (recovered from git history) | Remains resolved in the audited checkout | That directory is absent; the maintained feature/storage/preview packages are the architecture-listed locations. This was a local untracked-directory observation, not a current source defect. |
| 14: duplicated lazy map initialization (recovered from git history) | Remains resolved | internal/dupes/dupes.go:108-117 centralizes ensureMapsLocked; wipe/adopt keep their intentionally different data semantics. |

No still-valid detailed older finding was removed. Already-resolved introductory history is condensed into this accounting; accepted debt remains visible. The competing hyphenated draft was removed before commit, leaving only the established underscore filename.

## Coverage and limits

| Area | Review performed | Limits / outcome |
| --- | --- | --- |
| Entry/composition/domain | main, launch/open-with, appState/viewer, feature/overlay assembly, startup/shutdown, preferences/session, selection, menu/shortcut admission | Detailed boundary review; preserve thin main and explicit orchestration. |
| Viewer and features | Load/scan/sort/vector/animation, display/zoom, grid/duplicates, comparison render pipeline, mosaic window, copy selection, deletion/save/export, clipboard/wallpaper glue, favorites, settings, EXIF/map, help/manual/widgets, autoupdate | Detailed failure/lifecycle review; simpler widget/layout sections sampled. No claim of native visual QA or every UI branch exercised. |
| Imaging/data | Probe/decode/orient/metadata/JPEG write/RAW/GIF/SVG/vector, caches/hashes, mosaic layout/render, dupes, scan/sort, completion/decodepool, favorites storage/previews, preferences/session | Production implementation reviewed; relevant tests inspected. Not every fixture-builder line or binary fixture visually inspected. |
| Native integration | appearance, clipboard, displays, distribution, filemanager/filepicker, trash, wallpaper, openwith, winpos, wincom, wingesture, launch | Source/build-tag review and host tests. No real desktop mutation, Windows runtime or platform packaging validation. |
| Update and tooling | update verification/download/staging/apply, rollback/provenance, six scripts, Makefile, CI/release/Store workflows, go.mod pins, Qodana/shard policy | Reviewed code and gate selection; no release signing, Partner Center submission, live update, external dependency-health survey or Qodana analysis run. |
| Tests and docs | Test inventory, relevant regression/concurrency/golden assertions, ARCHITECTURE, CONTEXT, existing refactoring backlog, agent agreement, plans/archive/todos, translation/manual constraints | Complete baseline gate passed; assertions and archived plans were sampled, not exhaustively re-proven. Documentation quality is assessed by concrete stale contracts, not prose/line-count metrics. |

`make verify` passed at the audited production revision: format, offline TUF-root and Qodana exclusion checks, vet, build, shard-manifest check (617 UI runnables), and Linux/amd64 Docker race suite. A focused native darwin/arm64 package/script suite also passed. Initial sandbox denials for Go cache writes/local test-server binding were resolved through approved execution; they were environment restrictions, not product failures. The audit probes intentionally expose defects despite that passing baseline. Details and commands are in the validation document.

The review did not attempt a real OOM, native GL race run, interactive latency benchmark, public map-service load test, Linux RSS leak experiment, Windows runtime path test, or exploitability assessment. Source-based conditional consequences are identified individually above. No finding claims coverage proves absence of other defects.

## Keep as-is and opportunities deferred

- Preserve consumer-side narrow Host interfaces, snapshot-based settings/menus/info exceptions, explicit construction/overlay order, and cross-feature policy in internal/ui. The viewer has broad composition responsibility; field count and comments alone do not establish a god-object defect.
- Preserve viewer-independent imaging/update/data packages, separate byte-budgeted full/thumbnail caches, and foreground `Add` versus speculative `AddIfFits`. Share complete-record construction at MA-006, not every loading/cache policy.
- Preserve deterministic comparison reference rendering and mosaic output/fidelity tests. Add native boundary coverage; do not remove the reference path to make tests resemble production superficially.
- Preserve update traversal/symlink defenses, download bounds, hash/provenance validation, Windows rollback ordering, and intentional HEIC fork pin. No broad security or decoder rewrite is supported by this audit.
- Preserve specialized orientation pixel loops and explicit embedded-preview RAW behavior. Neither duplication in hot loops nor the absence of RAW demosaicing is itself a defect.
- Do not split `scripts/testshards/main.go` or the viewer solely for line count. If modified later, cohesive file separation can improve navigation without introducing new packages or APIs.
- Existing platform dispatcher seams are intentional and tests are sequential. Replace them with instance-owned seams when lifecycle changes require it, not as an unrelated sweeping cleanup.
- Shutdown/harness cancellation lists and completion semantics deserve follow-through with MA-003/015/016: the test drain also stops favorite previews, slideshow and toast work beyond the production shutdown list. This audit does not establish an additional independent shutdown failure. Do not add a universal task registry or block UI waiting for UI callbacks.
- Map transport installs a process-global log filter (`internal/ui/exifwin/tiles.go:88-90`). Reassess that integration workaround when changing tile ownership, but no separate user-visible logging failure was reproduced.
- Minor stale comments/API references (for example the old LoadImage caller description) and the TODO link to an archived Store plan can be corrected during the relevant work. They do not warrant standalone architecture projects.

## Handoff

Start with bounded EXIF arithmetic and mosaic scratch-memory containment. The plan keeps quick local fixes separate from larger async/path work and gives each work package observable acceptance criteria. Rebase the findings against current source before implementation, especially the excluded display extraction. Maintain these IDs in tickets/PRs and append resolution evidence rather than silently deleting findings.
