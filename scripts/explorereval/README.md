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
`assets.sha256`. Upstream native runtime license files remain in the extracted archive.

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
the semantic verdict pending. HDBSCAN's quadratic implementation, unstable
batch layouts, reference-algorithm parity and the native UI trial remain open
before this experiment can become a full-library implementation.

Primary sources: [SigLIP 2 export](https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/tree/ba1f3b0843f24bc5417d38e19c37b287d719b2f4),
[Go ONNX binding](https://github.com/yalue/onnxruntime_go/tree/v1.36.0),
[ONNX Runtime](https://github.com/microsoft/onnxruntime/releases/tag/v1.29.0),
[Go UMAP](https://github.com/nozzle/umap/tree/f6085fb2514d623b8a7ebb2c478396238502aef6),
[Go HDBSCAN](https://github.com/alDuncanson/latent/tree/v0.1.4/projection).
