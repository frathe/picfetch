# Maintainability: safe inputs, correct file identity, and observable background work

Status: ready-for-agent
Date: 2026-09-06
Route: Deep implementation effort, delivered as independent work packages.

This specification turns the 2026-09-06 audit into implementation acceptance.
The [canonical backlog](../../needs_refactoring.md) owns finding severity,
evidence and resolution status; the [implementation plan](../../plans/2026-09-06-maintainability-plan.md)
owns work packages A-S and their ordering; the [validation record](../../plans/2026-09-06-maintainability-validation.md)
owns historical observations and temporary probes. Preserve those documents
and their stable MA identifiers. This publication implements no refactors and
does not mark any finding resolved.

## Problem Statement

PicFetch can crash on malformed camera metadata or request gigabytes of mosaic
scratch memory for a small, valid panoramic source. Other ordinary workflows
can leave the displayed file list inconsistent with a confirmed deletion,
change a path selected in a native dialog, show incomplete cached image
information, or save contradictory JPEG dimensions.

Background work has related contract gaps: a queued UI callback is treated as
already complete, an obsolete image read can publish into a newer file set,
and cancellation or window closure does not consistently stop owned work.
Large-folder grouping and image actions can occupy the UI thread. Native
tests and actual comparison shader rendering also have gaps in release
validation that the passing headless suite cannot cover.

The audit identifies 24 items: 20 required corrections or verification
improvements, two conditional improvements, and two accepted dependency
watches. Its production baseline is `2ae4e0f6fd3ec53b35fb20d60c583402e0045d3f`.
At spec creation, HEAD is `79f696f`; changes since that baseline are confined
to documentation. Historical observations remain evidence from the audit,
not newly executed reproductions.

## Solution

Strengthen the existing owning modules so opening, comparing, saving,
copying, deleting and generating images preserve the user's intended files
and remain responsive. Bound transient work, carry cancellation to the
operations that can honor it, and publish results only to the request and
source generation that produced them. Exercise the native contracts PicFetch
actually ships while preserving its deterministic reference tests.

Implement MA-001 through MA-020 in the phased plan's independently reviewable
work packages. Start with MA-001 and MA-002. Measure MA-021 before deciding
whether to change scheduling. Address MA-022 opportunistically when command
routes are already being changed. Keep MA-023 and MA-024 as explicit watches.

## User Stories

1. As a viewer user, I want malformed EXIF data to produce a tolerant result or an error, so that opening a photo cannot terminate PicFetch. (MA-001)
2. As a camera RAW user, I want malformed embedded-preview pointers handled safely, so that a damaged file does not interrupt viewing other images. (MA-001)
3. As a mosaic user, I want extremely wide or tall images to fit within a bounded preparation budget, so that a small source cannot exhaust memory. (MA-002)
4. As a mosaic user, I want placement, aspect ratio and rotated edges preserved, so that safer rendering retains the intended composition. (MA-002)
5. As a viewer user, I want animated frames to keep their individual delays under a busy UI queue, so that playback remains correct. (MA-003)
6. As a picture-frame mode user, I want leaving the mode to invalidate pending advances, so that a late callback cannot change my next session. (MA-003)
7. As a comparison user, I want native chooser admission and returned results checked on the UI thread, so that opening files respects the active mode. (MA-003)
8. As a Grid View user, I want confirmation to delete the files I selected even if ordering changes, so that the remaining Grid result matches the files still on disk. (MA-004)
9. As a Grid View user, I want partial deletion failures and a fresh drop during deletion handled by file identity, so that unrelated files remain available. (MA-004)
10. As a user choosing a destination, I want PicFetch to act on exactly the path I confirmed, so that a legal newline or trailing carriage return cannot redirect a save. (MA-005)
11. As a user opening several files, I want spaces, Unicode and legal delimiter characters preserved, so that my filenames survive native transport. (MA-005)
12. As a comparison user, I want later single-image viewing to show the same file size and EXIF availability as a normal open, so that information does not depend on cache history. (MA-006)
13. As a user saving a rotated JPEG, I want its numeric metadata dimensions to describe the written frame, so that other programs agree on the image size. (MA-007)
14. As a photographer, I want unrelated camera metadata, DPI and color profiles retained when saving a rotation, so that correcting dimensions does not discard valid information. (MA-007)
15. As a user searching a large Grid result, I want unchanged duplicate groups reused, so that each keystroke does not repeat expensive grouping. (MA-008)
16. As a duplicate-inspection user, I want group membership and representative rules preserved, so that a performance improvement does not change which files are hidden. (MA-008)
17. As a user replacing a source or opening a new file set, I want late thumbnail facts rejected, so that old pixels or failures cannot describe the new generation. (MA-009)
18. As a user cancelling a sort or favorite preview pass, I want cancellable reads and waiting work to stop, so that obsolete work releases capacity. (MA-010)
19. As a clipboard user, I want a temporary-file failure reported as the actual write or close failure, so that cleanup cannot turn failure into apparent success. (MA-011)
20. As a Windows user copying files, I want accented, CJK and emoji filenames decoded correctly, so that the copied list names the files I selected. (MA-012)
21. As a Linux user with a custom data directory, I want deletion to honor my normal XDG configuration, so that my desktop can find the trashed files. (MA-013)
22. As a viewer user, I want clipboard encoding, Save Changes and metadata operations to leave input and redraw responsive, so that large files do not freeze the window. (MA-014)
23. As a user changing images during an action, I want the action tied to its captured source and its result applied only where current, so that navigation cannot retarget a write or install obsolete UI state. (MA-014)
24. As an EXIF map user, I want navigation and window closure to cancel obsolete tile work, so that request concurrency and failure metadata remain bounded. (MA-015)
25. As a user closing a window, I want its position polling to stop observably, so that queued native reads cannot update a closed target. (MA-016)
26. As a release maintainer, I want native Windows, macOS and Store-tagged guards executed, so that a green release gate covers those shipped contracts. (MA-017)
27. As a comparison user, I want pan, zoom, rotation and swipe verified with the production GL renderer at representative display densities, so that reference-renderer success is backed by native evidence. (MA-017)
28. As a favorites user, I want a rapid same-size source edit to invalidate its preview, so that the cache uses the timestamp precision available from the filesystem. (MA-018)
29. As a decoder maintainer, I want falling RSS samples treated as falling memory use, so that an optional leak check cannot report an unsigned-underflow false alarm. (MA-019)
30. As a release maintainer, I want reviewed packaging tool and image inputs recorded in logs, so that rebuilding does not silently select different packaging dependencies. (MA-020)
31. As a user opening a cold favorite, I want preview prewarming to leave capacity for interaction when measurements show contention, so that visible images are served promptly. (MA-021, conditional)
32. As a keyboard and menu user, I want consistent command admission and Escape priority, so that equivalent actions respect comparison, Copy Selection mode, Grid View, inspect and picture-frame mode. (MA-022, conditional)
33. As a maintainer, I want the HEIC fork retired only after an official release contains its fix, so that dependency cleanup preserves the leak mitigation. (MA-023, watch)
34. As an update user, I want dependency-footprint improvements to retain signature and provenance verification, so that build simplification preserves the update trust contract. (MA-024, watch)

## Implementation Decisions

These decisions synthesize the supplied audit and plan. Treat them as settled;
change them only when current-code evidence contradicts their premise, and
record the replacement decision with the affected MA item.

| Area | Decision and contract |
| --- | --- |
| Ownership | Keep the viewer as composition owner, feature-owned state behind narrow consumer-side Host interfaces, the established State-snapshot exceptions, and explicit feature/overlay order. Keep imaging and data modules viewer-independent. |
| Checked input | Imaging owns checked offset/span arithmetic shared by TIFF readers. Reader tolerance and writer malformed-block policies remain distinct. Check arithmetic before slicing or allocating. |
| Mosaic memory | Mosaic owns an explicit scratch budget covering simultaneous preparation buffers, resampling and masks. Clip or tile preparation to the visible contribution while preserving source geometry. Account for output, decoded sources and repeat-cache memory separately. |
| Deletion identity | Capture an immutable target list at prompt time. Reconcile successful removals with the current file set by URI identity; a saved index alone never authorizes removal from a changed list. Prevent overlapping confirmations from applying twice. |
| Path transport | Native adapters preserve structured path boundaries. A save returns one exact destination; multi-open preserves each path. Unsupported backend transport must fail explicitly rather than silently alter a filename. Cancellation remains distinguishable from failure. |
| Cache records | One complete LoadedImage construction contract supplies pixel data and file-size/EXIF facts to foreground, preload and comparison callers. Keep foreground Add and speculative AddIfFits admission policies explicit and preserve original animation frames and RAW preview behavior. |
| Saved JPEG dimensions | Save Changes uses the existing dimension-invalidated policy against written pixels. Correct representable dimension tags and remove geometry that cannot be made true; retain unrelated metadata. This deliberately supersedes the earlier policy that SaveRotated leaves dimension tags alone. |
| Queue ownership | Animation state has one goroutine owner. UI state reads and mutations happen on UI. Acknowledgement needed for pacing is cancellable; callbacks recheck staleness when applied. No shutdown path waits on UI for a callback that itself needs UI to finish. |
| Cancellation | Ancillary imaging reads accept caller context; sorting, previews and native-size fallback propagate their current request context. A cancelled decode-slot waiter never begins decoding. Document non-interruptible decoder work honestly. |
| Fact publication | Hashes, failures and native dimensions are admitted atomically against the generation captured before work began. A reorder/adoption can retain valid established facts; source replacement/reset invalidates them even when the URI is unchanged. |
| Grouping | The duplicate model owns immutable accepted groups keyed by file generation, hash/native-fact revision and distance. Search filters that snapshot. Changed groups compute off UI with cancellation and stale-install rejection. Preserve greedy complete linkage and highest-pixel-count/lowest-index representatives. Index selection follows measurement. |
| Image actions | Owning features capture immutable action inputs and perform expensive reads/encoding away from UI. Serialize mutations of the same source, preserve atomic writes, and apply completion/error UI once under the current request. Navigation cannot retarget an already captured mutation. |
| Map and polling | A tile fetcher enforces one aggregate request bound for foreground and warm work, deduplicates requests and bounds failure metadata. Superseded warm work and closed windows cancel. Position polling distinguishes cancellation requested from worker completion and rechecks cancellation before native reads/publication. |
| Local failures and environment | Clipboard cleanup preserves the primary failure, optionally joining cleanup failure. Trash preserves ordinary XDG overrides and normalizes only identified sandbox redirection. Keep these package-specific contracts separate. |
| Preview identity and RSS | Favorite-preview keys retain nanosecond modification time; older key entries become misses/sweepable cache data. RSS arithmetic handles growth, equality and decline without wrapping; the long-running leak test stays opt-in. |
| Native and packaging evidence | Retain Linux reference/golden tests and add execution of omitted native/Store contracts plus maintained production-GL smoke evidence. Centralize reviewed packaging tool/image inputs across local, release and Store builds and log their resolution. |
| Conditional work | Preview scheduling changes require measured contention. Command-policy consolidation is limited to repeated admission decisions after capturing current behavior and intentional exceptions. Neither justifies a universal scheduler or mode registry. |
| Dependency watches | Keep the patched HEIC fork until fix inclusion is verified. Keep the updater's supported verification boundary; reassess footprint at a major verifier upgrade without weakening trust or provenance. |

## Testing Decisions

Test observable behavior at the highest existing boundary that exposes the
defect: decoded metadata and written files, mosaic results and allocation
planning, feature actions through fake Hosts, and UI commands through the
production-shaped viewer harness. Assert file identity, numeric tag values,
rendered membership and callback effects rather than helper call order or
widget Visible flags alone.

The supplied plan and validation already establish these seams; this spec
retains them. Add an instance-owned queue/read seam only where the current
boundary cannot hold a callback or source read deterministically. Existing
OS dispatcher stubs remain appropriate; add no mutable package-level test seam.

| Boundary | Existing prior art and how to extend it |
| --- | --- |
| Imaging | Synthetic oriented/GPS/dimension-tagged JPEG and RAW fixtures in `internal/uitest`; metadata/orientation tests and numeric export assertions in `internal/imaging`. Replace the presence-only `TestSaveRotated_LeavesDimensionTagsAlone` policy with value-based saved-output assertions. |
| Mosaic | Seeded layout, render fidelity, cancellation and repeat-cache tests in `internal/mosaic`; generation/stale-result tests in `internal/ui/mosaicwin`. Observe proposed scratch sizes before allocation; never induce a real OOM. |
| Viewer and features | `newTestUI`, `newTestViewer`, `dropAndWait`, named waits and feature Settle methods. Use `uitest.UIQueue` and gated timers/readers for delayed completion. Preserve grid/compare/mosaic wait-and-drain ordering. |
| Deletion and actions | Deletion fake Host and temporary-file trash helper, plus `uitest` chooser/clipboard/trash/wallpaper stubs. Gate work while changing ordering, generation or active image, then assert disk and current-list identity. |
| Duplicate and preview data | `TestDuplicateGroups_ChainIsNotTransitive`, `TestAdoptGeneration_KeepsFactsAcrossGenerationChange`, grid hash/thumbnail tests and favorite sync/cancellation tests. Gate the old read across same-URI replacement, including success and failure; inspect model facts before UI installation. |
| Native and transport | Test the real serialization/decoding boundary with temporary paths. Execute Windows PowerShell decoding without invoking the clipboard. Exercise macOS bridge transport separately from real desktop mutation. Use controlled local HTTP servers for maps. |
| Release validation | `TestSetWindows_TargetPreservesOpaqueIDAndUnicodePath`, `TestSetWindows_TargetValidationFailsBeforeMutation`, main's `TestInstall_GraftsOntoGLFWsDelegate` and `TestStoreManaged_MicrosoftStoreBuildIsTrue`. Packaging prior art includes `TestMicrosoftStoreWorkflowAndBuildTarget` and `TestPackagingToolsUseCurrentFyneCLI` in `scripts/msixstage`. Keep shader runtime checks separate from canvas-reference assertions. |

Each mandatory acceptance criterion below must acquire maintained named
regressions covering its stated cases. The package commands are the execution
scope, not a claim that today's tests already prove the new behavior. Record
the exact added/changed test names and their actual non-skipped execution in
the implementation record. For focused `-run` commands, check the build-selected
inventory with `go test <same flags and packages> -list <pattern>` first;
`[no tests to run]`, build-tag exclusion or a skip cannot close a criterion.

For reproduced defects, run the regression against the defective behavior and
record the expected failure before implementing the fix. For new guards,
deliberately violate the guarded contract, observe failure, then restore it.
The temporary audit probes are inputs to this work: some pass by observing a
defect, and the queued-animation probe expects premature scheduling. They
cannot be copied unchanged as success criteria.

Use channels, controlled queues and observable done signals rather than
sleeps. A zero pending counter is insufficient when its callback runs later.
Attach all new workers to production cancellation and the viewer harness drain.
New test files require exact Qodana exclusions; new top-level viewer tests
require shard assignments and updated counts.

## Acceptance Criteria

Commands run from the repository root on a suitable Go/C/GL toolchain.
Native-only commands run on the named OS. The common Docker race gate remains
mandatory for every mergeable implementation change.

### AC01 — Safe TIFF spans (MA-001; A)

Malformed root, nested and next-IFD offsets; entry/value length overflow;
near-end spans; truncation; and MaxUint32 offsets cannot panic metadata,
orientation or RAW preview reading in either byte order. Valid metadata and
previews remain intact. Add a maintained `FuzzTIFFReaders` target covering
these reader entry points with deterministic seeds, then run it boundedly.

Verify: `go test ./internal/imaging -count=1` and
`go test ./internal/imaging -run '^$' -fuzz '^FuzzTIFFReaders$' -fuzztime=30s`.
The fuzz target is to be implemented; it does not exist at spec publication.

### AC02 — Bounded mosaic scratch (MA-002; B)

Wide and tall source ratios through 10000, including the audited 1920x1080
target/defaults/seed-42 case, cannot exceed the declared aggregate scratch
budget. Check dimensions and byte arithmetic before allocating. Assert this
with an allocation-plan observer; preserve ordinary seeded output, crop and
rotated-edge fidelity. Cancellation stops further preparation.

Verify: `go test ./internal/mosaic ./internal/ui/mosaicwin -count=1`.

### AC03 — Correct delayed dispatch (MA-003; I)

A held UI queue proves heterogeneous animation delays follow the applied
frame, without worker/UI shared mutable locals. Closing picture-frame mode
invalidates queued advancement. Chooser admission and returned-result
revalidation occur on UI, including comparison activation while the chooser
is outstanding. Cancellation with a callback pending terminates without a
UI-wait cycle; releasing it later cannot mutate a newer session.

Verify: `go test ./internal/ui ./internal/ui/slideshow -count=1`.

### AC04 — Deletion follows confirmed identity (MA-004; C)

Prompt for A in `[A,B]`, reorder to `[B,A]`, then confirm: only A is removed
from temporary disk and the current list. Cover a fresh drop while prompting,
generation changes during trash work, partial failures and overlapping
confirmations. Caller mutation of the original target slice cannot retarget
the operation. Every successful deletion is reconciled by identity, and
failed or unrelated files remain.

Verify: `go test ./internal/ui/deletion ./internal/ui -count=1`.

### AC05 — Exact chooser paths (MA-005; F)

Multi-open and single-save transport preserve supported legal POSIX paths
containing embedded newlines, trailing CR, spaces and Unicode. The save
destination is exactly the confirmed path; cancellation, empty selection and
errors remain distinct. Exercise each changed backend's actual transport,
including native macOS, using temporary paths and no user-file overwrite.
If a backend cannot represent a path, assert an explicit failure before use.

Verify: `go test ./internal/filepicker -count=1` on each changed platform, and
`go test ./internal/ui -run 'Test.*(Chooser|OpenFile|ExportAs)' -count=1 -v`.
Record native transport case names and results separately from parser tests.

### AC06 — Complete cached image records (MA-006; D)

The same encoded bytes loaded by foreground, preload and comparison produce
equivalent size/EXIF facts and canonical pixels. Comparison-first navigation
shows the real byte size and EXIF affordance in the UI tree. Preserve RAW,
orientation and animation policy, stale-result checks, and Add/AddIfFits
cache admission behavior.

Verify: `go test ./internal/ui ./internal/imaging -count=1`.

### AC07 — Truthful saved JPEG geometry (MA-007; E)

Save Changes writes numeric dimension tags matching the encoded frame after
either quarter turn and EXIF orientations 5-8. Verify IFD0, Exif and Interop
dimension values, orientation normalization, fallback removal of uncorrectable
tags and invalidated coordinate tags, and preservation of unrelated metadata,
DPI and color profiles. Unchanged geometry retains its existing policy.

Verify: `go test ./internal/imaging -run 'Test.*(SaveRotated|Export|JPEG|Exif|EXIF)' -count=1 -v`
and `go test ./internal/ui -run 'Test.*Save' -count=1 -v`.

### AC08 — Reusable cancellable duplicate groups (MA-008; O)

Search edits reuse unchanged groups. File/fact revision or distance changes
schedule cancellable computation away from UI, and superseded snapshots
cannot install. Compare membership and representatives with the current
greedy complete-linkage algorithm on bounded randomized and adversarial
inputs, including non-transitive chains and representative ties. Benchmark
unrelated and dense inputs at 10k, 50k and 200k and retain reproducible data;
never run a quadratic 200k baseline on UI.

Verify: `go test ./internal/imaging ./internal/dupes ./internal/ui/grid ./internal/ui -count=1`.
Add `BenchmarkGrouping` in `internal/dupes` for those sizes/distributions;
verify with `go test ./internal/dupes -run '^$' -bench '^BenchmarkGrouping$' -benchmem -count=3`.
The benchmark is a required future harness, not an existing timing claim.

### AC09 — Conditional fact publication (MA-009; K)

Pause old same-URI reads, replace/reset the source generation, then release
success and failure separately. Neither old hashes, failure markers nor
native dimensions enter the new model through hash or thumbnail workers.
Check admission atomically with mutation. Cover reorder/adoption separately
to retain valid established facts without treating replacement as adoption.

Verify: `go test ./internal/dupes ./internal/ui/grid ./internal/ui -count=1`.

### AC10 — Cancellation reaches ancillary reads (MA-010; J)

Controlled chunked reads stop on capture-date sort, favorite-preview and grid
native-size cancellation. A cancelled slot waiter performs no decode.
Cancelled passes expose completion and do not sweep previews that they have
not visited. Context is checked before and after non-interruptible decoder
work so its obsolete result is discarded; no promise of interrupting such a
decoder is introduced.

Verify: `go test ./internal/imaging ./internal/filesort ./internal/favthumbs ./internal/ui/grid -count=1`.

### AC11 — Preserve clipboard failure (MA-011; G)

Failed temporary PNG write and close return the primary error when removal
succeeds. If removal also fails, the primary cause remains inspectable.
Success returns a usable path. Fault tests reach the package behavior through
an instance/local writer seam or isolated subprocess and never mutate the
desktop clipboard.

Verify: `go test ./internal/clipboard -count=1`.

### AC12 — Explicit Windows file-list encoding (MA-012; F)

Execute the generated file-list decoding with Windows PowerShell and accented,
CJK and emoji UTF-8 paths. Decoded paths match byte-for-byte after Unicode
conversion; clipboard mutation is stubbed. Assert explicit decoding so an
ANSI-default environment cannot change behavior. Inspect chooser stdout as a
separate boundary without assuming it has the same defect.

Verify on Windows: `go test ./internal/clipboard -count=1 -v`.
The native decoding test must appear as executed, not skipped; source/script
text inspection and cross-compilation alone do not satisfy this criterion.

### AC13 — Respect Trash configuration (MA-013; G)

Test absent/default XDG settings, legitimate custom XDG_DATA_HOME and identified
sandbox redirection. Preserve ordinary overrides and normalize only the
demonstrated redirected environment. Assert the environment passed to the
adapter without moving any real desktop file.

Verify: `go test ./internal/trash ./internal/wallpaper -count=1`.

### AC14 — Responsive image actions (MA-014; N)

Gate clipboard encoding, Save Changes, EXIF reading and metadata removal
individually; a queued UI interaction still completes while each is blocked.
Capture source identity and pixels before work, serialize same-source writes,
retain atomic replacement, and report completion/failure once. Navigation or
close cannot retarget work or apply old UI state. Test cancellation before a
write and completion after an atomic replacement has already committed;
cancelled presentation must not pretend to undo an accomplished disk write.

Verify: `go test ./internal/ui ./internal/ui/exifwin ./internal/imaging -count=1`.

### AC15 — Bounded map lifetime (MA-015; L)

Concurrent warm passes and foreground requests share the declared fetcher
limit, preserve deduplication/cache behavior, and cancel obsolete/closed work.
Failure metadata has a tested capacity or expiry bound. A controlled local
server verifies limits, cancellation and retry behavior. Completion assertions
wait for onChange's own effect, including cancellation and close races.

Verify: `go test ./internal/ui/exifwin -count=1`.

### AC16 — Observable position-poller stop (MA-016; M)

Test stop before enqueue, after enqueue and during a gated native read.
Queued callbacks recheck cancellation, no stopped target receives a position,
and the worker exposes completion. Calling cancellation from UI never blocks
on that UI queue. Tests exercise the polling worker through an instance-owned
native-read/queue seam; a non-native-window no-op is insufficient evidence.

Verify: `go test ./internal/winpos ./internal/ui/widgets ./internal/ui -count=1`.
Retain native shutdown evidence under AC17 for platform claims.

### AC17 — Execute shipped platform and renderer contracts (MA-017; R)

CI selects and executes the Windows wallpaper target guards, main's macOS
delegate-graft guard and Store-tagged distribution policy. Logs enumerate
those exact tests; they retain stubbed desktop mutations. Commands:

```sh
# Native Windows with its C/GL toolchain.
go test ./internal/wallpaper ./internal/clipboard ./internal/filepicker ./internal/update ./internal/ui/autoupdate -count=1 -v
# Native macOS; main links the Cocoa driver for the graft test.
go test . ./internal/openwith ./internal/displays ./internal/winpos -count=1 -v
# Explicit Store build selection on an appropriate native runner.
go test -tags=microsoftstore ./internal/distribution ./internal/ui/autoupdate -count=1 -v
```

Maintain a native GL smoke procedure launched with `make run` on a prepared
desktop and temporary fixtures. Record OS/GPU, display density, source sizes,
shader compilation/output and screenshots for linked/unlinked pan, zoom,
rotation, side-by-side/swipe transitions, large-source detail and close during
work. Test representative standard/high DPI. The command launching the app is
only the entry point; the recorded observations are required acceptance
evidence. Keep Linux golden/reference checks. Cross-vet/build or headless
rendering cannot substitute for the native run.

### AC18 — Subsecond preview freshness (MA-018; H)

Different same-size bytes at the same path with different subsecond mtimes
produce different preview identities. Old-format entries become misses and
can be swept after a complete pass. Normal hits and cancelled-pass retention
remain correct. A filesystem lacking that precision is identified explicitly
and the arithmetic/key contract is still tested with controlled inputs.

Verify: `go test ./internal/favthumbs -count=1`.

### AC19 — Safe optional RSS arithmetic (MA-019; H)

Increasing, equal and decreasing uint64 RSS samples produce correct bounded
growth without underflow. Add a deterministic `TestRSSGrowth` arithmetic test
that does not require the long leak experiment. Preserve the optional
heicleak build tag and runtime opt-in for the actual RSS experiment.

Verify on Linux: `go test -tags=heicleak ./internal/imaging -run '^TestRSSGrowth$' -count=1 -v`.
This named regression is to be added. The optional experiment remains
`PICFETCH_HEIC_LEAK_TEST=1 go test -tags=heicleak ./internal/imaging -run '^TestHEICDecode_DoesNotGrowRSSUnbounded$' -count=1 -v`.

### AC20 — Reviewed packaging inputs (MA-020; S)

Local, release and Store packaging resolve tools from reviewed constants;
cross images are pinned or explicitly versioned with recorded resolution.
Build logs identify tool versions and resolved images. Exercise changed
packaging routes without publication and inspect their artifacts and native
startup. This requires packaging inputs to be stable, not byte-identical
artifacts despite timestamps/signatures.

Verify the relevant builds in a disposable checkout: `make install-tools`,
`make package-mac` on macOS, and `make package-windows package-windows-store package-linux`
with Docker. Run `go test ./scripts/msixstage ./scripts/plistdoctypes -count=1`
for maintained packaging contracts. Inspect produced architecture/distribution metadata and smoke
run artifacts on their named OS. The Store bundle path additionally executes
its existing package-validation/WACK steps on Windows without submission.
Record the exact artifact and smoke commands in the work-package evidence;
Make dry runs alone are insufficient.

### AC21 — Evidence before preview scheduling changes (MA-021; P, conditional)

After AC10, measure foreground latency and preview convergence for a large
cold favorite, with and without competing preview work. Add the repeatable
`BenchmarkPreviewForegroundContention` harness in `internal/favthumbs` when
this work package is selected; run
`go test ./internal/favthumbs -run '^$' -bench '^BenchmarkPreviewForegroundContention$' -benchmem -count=3`.
Record hardware, data and foreground-latency measurements as well as throughput.
If contention warrants a policy change, demonstrate reserved interactive
capacity, eventual idle disk-cache convergence and cancellation without an
incomplete sweep using `go test ./internal/favthumbs ./internal/ui/grid ./internal/ui -count=1`.
Otherwise record the measurement and retain the current policy explicitly.
Unselected work remains open and does not block MA-001 through MA-020.

### AC22 — Preserve command policy during consolidation (MA-022; Q, conditional)

When touching these routes, capture the existing command/mode matrix for
comparison, Copy Selection mode, Grid View, inspect and picture-frame mode.
Tests cover Escape priority, menu/keyboard/direct-entry parity, and intentional
exceptions such as comparison Help and Open refusal. Consolidate repeated
decisions only if it simplifies the touched routes while retaining that matrix.

Verify: `go test ./internal/ui ./internal/ui/menus -count=1`.
Unselected consolidation remains open; AC03's chooser-thread fix is mandatory
independently and is not counted twice.

### AC23 — Keep the HEIC mitigation (MA-023; accepted watch)

At the next HEIC dependency update, establish that the chosen official tag
contains the leak fix before removing the fork. Preserve the mitigation until
then. Verify any eventual replacement with `go test ./internal/imaging -count=1`
and AC19's optional Linux RSS command after its arithmetic fix. Link upstream
fix-inclusion evidence at that time; passing decode tests alone cannot prove
it. No upgrade or recurring monitor is created by this spec.

### AC24 — Preserve updater verification (MA-024; accepted watch)

At the next major verifier upgrade, record reachable dependencies, build time
and binary size using `go list -deps ./internal/update`, `time make build`
and `wc -c bin/picfetch`, with platform and cache conditions. Any reduction
must retain supported signature/provenance checks, traversal/symlink defenses,
download bounds and rollback behavior. Verify an eventual change with
`go test ./internal/update ./internal/ui/autoupdate -count=1` and the common
gate. No current footprint regression or dependency-removal task is claimed.

### AC25 — Implementation completion gate

Every mergeable work package runs `make verify` successfully, including
format, offline TUF, Qodana exclusions, vet/build, exact shard inventory and
Linux/amd64 Docker race tests. Update test manifests before this gate. Run
`make golden` only for an intentional render change, inspect its Linux/amd64
differences, and retain no failed renders. Record native, benchmark and manual
evidence required above separately; a passing common gate does not waive it.

The lead checks acceptance against command output and resolves review findings.
Update the canonical MA status with implementation/test evidence and the
linked open work. Update the architecture map if package/file placement
changes, and archive the accepted implementation plan only when its remaining
work and links are accounted for. Creating this spec does not authorize a git
commit or close these implementation criteria.

## Out of Scope

- A bulk refactor, new universal controller/task registry, or package splitting based solely on file length.
- Reopening retired architectural findings or reviewing the previously excluded display extraction as part of this specification.
- Altering duplicate membership to transitive grouping, distorting mosaic aspect ratios, or merging full-image and thumbnail cache policies.
- Removing deterministic render references, promising interruption inside unsupported third-party decoders, or substituting a real OOM for bounded allocation tests.
- New product modes, RAW demosaicing, additional image formats, or unrelated translations/settings changes.
- Immediate HEIC/updater dependency upgrades, custom cryptographic verification, standing monitoring, release publication, Store submission or automated deployment.
- Production implementation or implementation-ticket creation during this `/to-spec` publication.

## Further Notes

### Sequence and ownership

Use the [existing phased plan](../../plans/2026-09-06-maintainability-plan.md)
for files and task decomposition. A/B are independent high-priority starts;
small D/G/H/S changes can proceed separately without delaying them. I/J
establish patterns useful to K/L/M/N; N also preserves D's record contract;
K precedes O; J precedes P; I precedes Q. R can start early and supplies native
evidence for F/I/M. These are dependencies and useful precedents, not a reason
to create a shared async framework. Split unrelated G/H entries during ticketing.

Before implementing a work package, add its concrete files/contracts, named
tests, verification command, owner and budget to the active plan under the
current working agreement. The lead owns spec decisions, architecture, review
and the final gate. Spec preparation used one read-only test-seam scout;
acceptance and document edits stayed with the lead. The scout's bounded
inventory crossed no write ownership, needed no spec decisions, and its
reported identifiers were checked by repository search (G1-G5; no mechanical
edit or implementation was delegated).

### The honest limit

Malformed-input hardening does not prove absence of all decoder defects.
The mosaic scratch budget is distinct from total process memory. Context
cancellation cannot preempt a decoder or blocking operation without interrupt
support, and cannot undo an already committed atomic write. URI and mtime/size
identity do not detect external replacement that preserves all observed
identity fields. A dimension-only policy still cannot infer invalidated
coordinates for a 180-degree turn with unchanged bounds; this retains the
existing dimension-policy limit rather than introducing viewer rotation state
into imaging. Native renderer results are limited to the environments exercised.

MA-021/022 remain conditional and MA-023/024 remain accepted watches after the
mandatory implementation work. None should be silently marked fixed merely
because the required work packages or this documentation pass validation.
