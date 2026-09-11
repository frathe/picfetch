# Local explorer engine experiment

Asset setup supports Apple Silicon macOS and x64/ARM64 Linux and Windows.
Production worker/UI tests support those platforms. The original smoke
and full-library evidence commands below remain
macOS-only.
Smoke mode processes a bounded corpus; library mode opens the native viewer
for the complete supplied collection. Technical collection alone does not qualify
the intended 50,000-image trial. The source images, thumbnails, vectors, manifests and
reports stay local under the ignored `.scratch/visual-similarity-explorer/` tree.

From the repository root:

```sh
make explorer-setup
make explorer-test
make explorer-evaluate TRIAL=smoke
```

Setup downloads a public 372 MB float32 SigLIP 2 vision model, its processor
configuration, and the ONNX Runtime archive (11 MB on Linux x64, 10 MB on Linux ARM64, 42 MB on macOS,
80 MB on Windows x64, 82 MB on Windows ARM64). Published
model/archive SHA-256 values are checked before use/extraction; extracted
runtime and processor hashes are checked too. Nothing is installed globally.
The shared installer selects pinned platform archives in `internal/similarity/assets.go`;
extracted files are checked against `internal/similarity/assets.sha256`. Upstream native runtime license files remain in the extracted archive.

On Windows 11 x64/ARM64, install the assets from PowerShell without Bash or Make:

```powershell
go run ./scripts/explorereval -install -assets .scratch/visual-similarity-explorer/assets
```

`make explorer-setup` uses the same command. The download is about 451 MB on x64 or 453 MB on ARM64;
only the required DLLs and license notices are extracted from the Windows ZIP,
excluding its large debug symbols. Repeating setup verifies and reuses the local
files without another HTTP request. `make explorer-download-test` qualifies this
path without loading native code or launching an analysis worker.

Windows inference requires the [Microsoft Visual C++ v14 Redistributable for its architecture](https://learn.microsoft.com/en-us/cpp/windows/latest-supported-vc-redist).
The asset installer does not install system components or change firewall,
AppContainer, or filesystem permissions. The ARM64 runtime and executable can be cross-built; run inference on an ARM64 Windows device to qualify them.

The Windows worker is a normal hidden subprocess using local pipes, with no
listening port and no OS-enforced network block. ONNX Runtime telemetry is
disabled through its API before session creation, and failure stops analysis.
Windows events retain `OfflineVerified: false`; this field specifically records
OS enforcement, not whether processing is local or can run without internet.
The viewer explains this policy before first use. macOS/Linux isolation remains.

On x64/ARM64 Linux:

```sh
make explorer-setup
make explorer-install-test
make explorer-ui-test
make build
```

The native runtime needs glibc 2.28+ and libstdc++ providing GLIBCXX_3.4.22;
the kernel must allow seccomp filters. This is independent of distro/package
manager and honors the normal XDG user cache directory for in-app installation.
Alpine/musl is not qualified. ARM64 setup and worker support are implemented;
native ARM64 inference, kernel enforcement and GUI acceptance still require
hardware testing. A binary built with `make build`
uses the host's C library baseline; for older distros, build through the Makefile
in a container with an older glibc and inspect the binary's required GLIBC symbols.
The pinned fyne-cross packaging image has its own, potentially newer, baseline.
Runtime requirements do not replace the viewer's OpenGL/window-system requirements.
The 2026-09-10 qualification used Ubuntu 24.04 and Debian 12, including an
unprivileged Debian worker with networking disabled and read-only assets. The
Debian-built `bin/linux-portable/picfetch` requires GLIBC_2.34 (the native Ubuntu
build requires GLIBC_2.38). Build details and retained local evidence are in
[the implementation record](../../finished_refactorings/2026-09-10-explorer-ubuntu-setup.md).

Defaults:

- Inputs: `.scratch/visual-similarity-explorer/demo`.
- Assets: `.scratch/visual-similarity-explorer/assets`.
- Evidence: `.scratch/visual-similarity-explorer/evidence`.
- Provider: CPU, six native intra-op threads.

Override the three directories with `EXPLORER_LIBRARY`, `EXPLORER_ASSETS`, and
`EXPLORER_EVIDENCE`. `EXPLORER_PROVIDER=coreml` is an experimental comparison
switch; CoreML availability does not prove model acceleration. The native
test target uses the default pinned asset directory. `TRIAL=library` launches
the isolated native viewer and processes the complete supplied folder; see
"Isolated native collection" below.

The telemetry environment opt-out is set before loading the library, and the
runtime API disables telemetry before session creation.
The macOS worker uses `sandbox-exec` with `(deny network*)`; the Linux worker
installs a seccomp filter on all threads, denying sockets and io_uring before
reading requests. No root access, package manager, or namespace helper is needed.
It verifies OS permission
denial of TCP connection and empty UDP send attempts to a documentation address
before reading library contents. Network outages, timeouts and offline flags do
not satisfy the check. The Go build and public-asset setup happen separately
before the sandboxed processing. On a host already inside a restrictive sandbox,
launching a second sandbox may fail; that is a failed trial, not permission to
remove network denial.

The command processes at most 512 images using a fixed seeded order across
folder/format buckets. It runs a fresh 15D UMAP fit and HDBSCAN at the halfway
point, then admits remaining inputs and refits. A separate 2D UMAP fit determines
display coordinates. Cohort IDs derive from membership, avoiding arbitrary
numeric label renumbering. Each successful source appears exactly once in the
result; unreadable files have explicit failures and no cohort. The report
includes every cohort's full membership, up to 15 preview images, an Unassigned
collection, local source links, first/final maps and measured timings/memory.

Every run uses a new evidence directory. A failure retains its console/status
and partial evidence. `result.json` is written only after all required artifacts
are generated. Ctrl+C cancels the launcher and terminates/waits for its worker;
the process boundary is necessary because the trial clustering libraries do not
offer cooperative cancellation. A canceled worker must not publish a completed
result. The default Go test suite needs no assets or native runtime at execution;
`make explorer-test` runs additional real-model tests under explicit OS denial
and fails if assets or required observations are missing.

The generated `review.html` is a local artifact with a restrictive content
policy and no external resources. Use it to judge depicted-content grouping,
including any cross-media examples. A technically successful command leaves
full-library qualification pending. HDBSCAN's quadratic implementation, batch
layout continuity and reference-algorithm parity remain open before this
experiment can become a full-library implementation. The native completed-map
trial is described below.

Primary sources: [SigLIP 2 export](https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/tree/ba1f3b0843f24bc5417d38e19c37b287d719b2f4),
[Go ONNX binding](https://github.com/yalue/onnxruntime_go/tree/v1.36.0),
[ONNX Runtime](https://github.com/microsoft/onnxruntime/releases/tag/v1.29.0),
[Go UMAP](https://github.com/nozzle/umap/tree/f6085fb2514d623b8a7ebb2c478396238502aef6),
[Go HDBSCAN](https://github.com/alDuncanson/latent/tree/v0.1.4/projection).

## Production throughput profiling

`make explorer-profile` measures the production worker used by the viewer, on
up to 512 images selected by the same deterministic folder/format sampling as
the smoke experiment. Override `EXPLORER_LIBRARY`, `EXPLORER_ASSETS` and
`EXPLORER_EVIDENCE` as above. It retains the exact unstripped executable,
console, exit status, `result/events.jsonl` and `result/profile.json` in a fresh
`throughput-*` evidence directory. It does not launch the UI or qualify 50k.

The command uses an isolated temporary favorite for two passes: cold analysis,
then reuse of every unchanged successful representation. Failed sources are
retried and counted separately. Neither pass touches standing Favorites. The
temporary cache contains normal local representations during the run and is
removed before a completed summary is written. A changed input, incomplete
worker, cache warning, failed output write or cancellation refuses completion;
partial event evidence remains. Cancel with Ctrl+C. The report contains only
aggregate metadata, including a digest of source identities and content hashes,
executable digest, runtime/model/provider settings and temporary cache size.
It contains no paths, source images, previews or vectors.

macOS/Linux workers enforce TCP/UDP OS network denial before reading images;
Windows reports that this protection is absent. The
parent scans local directory metadata and records progress. Both use the
production CPU configuration (six native inference threads). The default
publishes only the completed map, matching the viewer's manual update default.
To include automatic map publications every 30 sources, use the retained binary:

```sh
bin/explorereval -trial throughput -automatic -out /tmp/picfetch-throughput-new
```

The output directory must be new. Each JSONL row includes pass, received time,
completed success/failure/reuse counts, stage and cumulative worker measurements.
Subtract successive rows to measure a source window. Inference throughput is
the change in `InferenceAttempts` divided by the change in `EncodeSeconds`;
it excludes decoding, cache work and grouping. Reused sources never count as
new inference. Failed decode attempts accrue decode time but no inference.

Timing definitions (seconds of wall time, not CPU time):

- `SetupSeconds`: worker offline/asset verification, tag setup and local file
  registration. `ModelSeconds`: lazy native inference initialization.
- `DecodeSeconds`: source read/probe, content hash and full oriented decode.
  `EncodeSeconds`: image preprocessing plus native inference and normalization.
  `PreviewSeconds`: scaling and JPEG encoding; `TagSeconds`: semantic labeling.
- `CacheSeconds`: favorite-cache opening, lookups, validation and writes.
- `GroupingSeconds`: all map grouping work. Its four named sub-stages are
  `ReductionSeconds`, `HDBSCANSeconds`, `ProjectionSeconds`, `HierarchySeconds`;
  they are already included in grouping and must not be added to it again.
- `ElapsedSeconds`: worker time before the current event is serialized; includes
  earlier delivery, source stat checks and unclassified bookkeeping. It excludes
  process startup/shutdown and the current event's transport.
- `FirstMapSeconds`: first map received by the profiling client;
  `WorkerExitedSeconds`: client wall time through observed child exit, including
  transport. These measure command delivery, not native map construction or paint.

`make explorer-test` runs profiling acceptance outside the outer sandbox used
by the old evaluator tests, because the production client creates its own
worker with OS network denial on macOS/Linux and ordinary local process pipes
on Windows. Native tests fail for missing assets, rather than skip.
This command supplies stage/count evidence for a modest corpus; UI latency,
native RSS, full-library scaling and the user's semantic verdict remain separate.

## Native viewer trial

Standalone installs offer the model/runtime download on first Explorer use;
Store installs download only the model data and include the runtime.
`make explorer-install-test` explicitly downloads the actual pinned public files
to a temporary directory, verifies progress and integrity, checks reuse without
HTTP, retains runtime notices, and runs synthetic-image inference with the
platform's network policy. This download qualification is separate from the
offline suite.

The accepted engine is shared with PicFetch in `internal/similarity`.
After `make explorer-setup`, run `make run`, open the demo directory, then choose
**Window -> Visual Similarity Explorer**. The map analyzes opened images, taking only the highest-resolution representative
of each duplicate group when duplicate filtering is active;
Grid search and selection do not narrow its input. Entering the explorer
maximizes the window. Drag or Shift-scroll to pan, scroll or use
`+`/`-` to zoom, and use **Fit map** to reset the view. A pile opens its complete
cohort in Grid View. Open an image normally; `Escape` returns to the cohort,
then `Escape` or **Back to map** returns to the preserved map camera.
**Unassigned** opens the noise collection. **Back to Viewer** leaves the map;
leaving unfinished analysis cancels and waits for the worker at shutdown.
Leaving releases map image resources; reopening rebuilds from saved favorite
representations where available. Replaced map revisions release their old image
sources immediately instead of waiting for Fyne renderer-cache expiry.

`make explorer-ui-test` requires the pinned assets on a supported Mac, Linux, or
Windows x64/ARM64 host. It runs the ordinary UI acceptance scenarios plus real worker
inference, the platform's network policy, missing-source accounting, and cancellation. Default assets
are `similarity-assets` beside the executable or the developer scratch assets
under the working directory; `PICFETCH_SIMILARITY_ASSETS` overrides that path.
No assets or images are downloaded during analysis. The model is still a
separate local setup asset, not included in ordinary release packaging.

**Update map** builds a map from currently retained representations. Optional
automatic updates every 30 sources default off; completion always builds a final
map. Continuing piles retain their positions and the camera expands when needed
to include newly discovered groups. Settings can disable this automatic fit. An open cohort keeps its captured
members until it is reopened. Piles have a minimum gap and thin fitted frames.

The 446-image progressive benchmark completed in 101.088 seconds, with its first
map at 9.840 seconds, 46 cohorts and 55 unassigned images. The prior 5/3 density
settings left 90 unassigned in the same input order; the new settings are 4/2.
UMAP is sensitive to input order, so these counts are benchmark evidence, not
a guaranteed count or semantic accuracy score. Favorite representations now persist
in each favorite’s `analysis` directory and are reused while source size/mtime
and model/preprocessing version match. Settings can disable persistence for
subsequent scans. Non-favorite images remain transient. Interruption, retry and
source-change recovery are implemented and covered by the native integration
suite. Ronin accepted native reuse and recovery in tickets 05-06. The
50,655-input completed-map trial records actual pipeline timings and accepted
responsiveness; integrated progressive/cache/recovery qualification at that
scale remains open in ticket 07.

**Hide tags**/**Show tags** collapse and restore the sidebar without changing
its choices. Each tag's count opens only its matching images in Grid View,
including matching Unassigned images. **Granularity** at the top right joins
related cohorts toward **Broader** or restores original cohorts toward **Finer**.
The worker supplies a centroid hierarchy in the 15D grouping space; changing
the slider cuts it locally without rescanning. Dragging applies the cut when
the slider is released; clicks and keyboard adjustments apply immediately.
New analysis results use the last released value during a drag. Unassigned stays separate, and
open grids retain their captured membership. Arrow keys highlight a directional
neighbor and reveal it; Enter opens the active stack, and +/- zoom.

Display events omit inference vectors while the worker and favorite cache keep
them. The bounded evaluator still writes vectors for engine-quality evidence.
Stack packing uses nearby-cell lookups and perimeter searches. These reduce
measured map-publication overhead; they do not qualify full-library throughput
or remove the batch UMAP/HDBSCAN scaling limit.


## Isolated native collection (`TRIAL=library`)

```sh
make explorer-evaluate TRIAL=library EXPLORER_LIBRARY=/absolute/local/folder
```

This builds and retains a native app bundle, runs it under effective OS network
denial, and opens the folder through PicFetch's normal scan/sort/Explorer path.
It processes all discovered inputs, without the 512-source smoke sampler.
An explicitly capped/truncated scan cannot produce a collected session.
Only run a private library when its owner has authorized that trial.

The new evidence directory contains the retained executable, `native.log`,
`exit-status.txt`, `runner.json`, and one-second process-group RSS samples in
`memory.jsonl`. The samples include the app and its analysis child, exclude
zombies, and are a sampled sum rather than an exact peak or private-memory metric.
Closing PicFetch finalizes `session/events.jsonl` and `session/session.json`.
The launcher forwards interruption to the app, observes its exit, and escalates
only its owned process group if graceful termination fails after ten seconds.

The public launch flag `--explorer-trial DIR` requires a new directory and actual
TCP/UDP OS denial before source reads. It supplies a unique Fyne identity and
separate Favorites, presets and updater directories; all update paths are disabled.
Ordinary application preferences, Favorites and preset definitions are untouched.
Preset/Favorite files may contain their normal user data; structured trial records
contain no source paths, pixels, embeddings or image metadata. Native diagnostic
logs are local and can contain errors with paths. Keep the evidence local unless
its contents have been reviewed for sharing.

Numbered records distinguish analysis start, received worker events, UI-applied
maps, worker exit, cohort opening, map return and Explorer exit. Queue/apply times
measure those boundaries, **not visible paint latency**. A final map must account
for the exact input identity digest and every source, with effective offline
verification and an observed worker exit. Failed/canceled runs retain their logs.
`Collected` means technical collection succeeded; `Qualified` remains false.
Human quality, large-map usability and full-library performance verdicts are
recorded separately. Tests and public-fixture runs cannot close ticket 07.

Each received worker event has a session-unique `Event` number, retained by its
`map-applied` record. `view-observed`, `cohort-open` and `map-return` carry that
map's event number plus `Surface` (`map`, `grid`, `image`, `comparison` or
`overlay`). Grid observations record the actual filtered results'
`VisibleTotal`/`VisibleSHA256`; image observations identify the current navigation
target. Map, comparison, overlay and empty observations omit the digest. Equal
grid digests across later publications establish unchanged source membership.
These are UI-state observations, not
proof of completed image loading or framebuffer paint. An update behind Grid
View therefore remains a grid observation even though the map widget is shown
underneath it.

The ordinary Explorer's **Presets** library supports AND rules over existing
visual tags and image facts, explicit matching previews, and reviewed Unassigned
membership. Favorite memberships persist independently of reusable definitions.
See the app manual for editing, pending members and compatibility recovery.

## Microsoft Store assets

Build with the `microsoftstore` tag and stage using `scripts/msixstage` with the
architecture-matched `-runtime-archive`. The executable loads only the packaged
runtime; cache/environment overrides select model data only. Its installer
transfers about 372 MB from Hugging Face and never downloads runtime code.
The MSIX includes upstream runtime/license/privacy notices and declares the
Microsoft Desktop C++ framework dependency. See `docs/microsoft-store.md`.

To qualify model-only downloads and inference, compile the explicit installer
test executable with `-tags "microsoftstore explorerinstall"`, stage it with its
matching runtime archive, and run the staged executable with
`'-test.run=^(TestAssetInstall|TestRealAssetInstall)$' '-test.v=true'`. This runs
the HTTP failure fixtures and checks that cached DLL copies
cannot override the bundled runtime. An ordinary `go test` temporary executable
has no bundled runtime and intentionally fails Store installation admission;
the HTTP fixtures skip there and run in the staged test executable instead.
