# Local explorer engine experiment

These commands evaluate the shared local engine on Apple Silicon macOS.
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
configuration, and the approximately 42 MB ONNX Runtime archive. Published
model/archive SHA-256 values are checked before use/extraction; extracted
runtime and processor hashes are checked too. Nothing is installed globally.
The assets are pinned to the revision and checksums in `setup.sh` and
`internal/similarity/assets.sha256`. Upstream native runtime license files remain in the extracted archive.

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

The native runtime's full telemetry opt-out is set before its library is loaded.
The worker uses `sandbox-exec` with `(deny network*)`. It verifies OS permission
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

Each worker enforces TCP/UDP OS network denial before reading images. The
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
denied-network worker. Native tests fail for missing assets, rather than skip.
This command supplies stage/count evidence for a modest corpus; UI latency,
native RSS, full-library scaling and the user's semantic verdict remain separate.

## Native viewer trial

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
leaving unfinished analysis cancels and waits for the isolated worker at shutdown.
Leaving releases map image resources; reopening rebuilds from saved favorite
representations where available. Replaced map revisions release their old image
sources immediately instead of waiting for Fyne renderer-cache expiry.

`make explorer-ui-test` requires the pinned assets on this Apple Silicon Mac.
It runs the ordinary UI acceptance scenarios plus real worker inference, denied
network access, missing-source accounting, and cancellation. Default assets
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
