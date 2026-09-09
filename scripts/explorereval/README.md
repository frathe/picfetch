# Local explorer engine experiment

This command implements ticket 02's bounded real-pipeline evaluation on Apple
Silicon macOS. It does not add an explorer to the viewer or qualify the intended
50,000-image library. The source images, thumbnails, vectors, manifests and
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
test target uses the default pinned asset directory. `TRIAL=library` is
deliberately refused until full-library integration is ready.

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
subsequent scans. Non-favorite images remain transient. Extended recovery and
full-library qualification remain tickets 06-07. Batch
HDBSCAN is not qualified at 50k.
