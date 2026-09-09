# Visual similarity explorer: local candidates

Research date: 2026-09-09. Scope: implications of SigLIP2, HDBSCAN and UMAP for the first offline trial on Ronin's current Mac and roughly 50,000-image library. These are evaluation candidates, not selected dependencies. The accepted behavior is one cohort per image, frozen membership while browsing a cohort, and a responsive map with visible progress during analysis.

## Verified source facts

**The candidates serve different stages.** SigLIP 2 is a family of
vision-language encoders that supplies visual representations. [Original
paper](https://arxiv.org/abs/2502.14786) UMAP's clustering guide demonstrates
dimensionality reduction followed by HDBSCAN, with a separate visualization
embedding. It notes that clustering need not use two dimensions and that
UMAP can distort density or introduce artificial splits. [Official clustering
guide](https://umap-learn.readthedocs.io/en/latest/clustering.html)

**HDBSCAN prediction preserves an existing clustering.** HDBSCAN is transductive: adding data can create, split or merge clusters. `approximate_predict()` assigns new samples against the fitted condensed tree without changing it. It needs prediction data, generated with `prediction_data=True` during fitting or `generate_prediction_data()` afterward. [Official prediction tutorial](https://hdbscan.readthedocs.io/en/latest/prediction_tutorial.html)

Its predictions need not match a fresh fit of old and new samples together. The API documentation identifies reclustering the combined dataset as the way to obtain that result. Prediction therefore provides assignment to existing groups, not incremental discovery of new groups. [Official prediction API](https://hdbscan.readthedocs.io/en/latest/api.html#hdbscan.prediction.approximate_predict)

**HDBSCAN can leave images unassigned.** Its hard labels give each sample one cluster number or `-1` for noise. Membership strengths are separate values; noise has strength zero. [Official basic usage](https://hdbscan.readthedocs.io/en/latest/basic_hdbscan.html)

**UMAP can place new data in an existing learned space.** After fitting, `transform()` embeds unseen samples into that space; the original training embedding is available as `embedding_`. The tutorial assumes reasonably consistent training and incoming distributions. It also notes first-call JIT overhead and that larger datasets increase transform costs. These examples do not establish performance for PicFetch. [Official transform tutorial](https://umap-learn.readthedocs.io/en/latest/transform.html)

**AlignedUMAP addresses correspondence across separate embeddings.** It requires relations between samples in consecutive datasets. Its `update()` method appends an embedding for a new slice aligned with prior slices. Alignment introduces overhead; online updates use backward relations and trade quality for incremental operation. The documented implementation retains prior data and describes generally consistent positions while allowing structural changes. [Official AlignedUMAP guide](https://umap-learn.readthedocs.io/en/latest/aligned_umap_basic_usage.html)

**A first-party image-embedding entry point exists.** Google's `google/siglip2-base-patch16-224` model card supplies `AutoModel`, `AutoProcessor` and `get_image_features()` examples and declares Apache-2.0 licensing. This is a concrete checkpoint for evaluation. [Google model card](https://huggingface.co/google/siglip2-base-patch16-224)

Transformers distinguishes fixed-resolution SigLIP2 models from NaFlex models, which preserve native aspect ratio and support varying resolutions. [Official Transformers SigLIP2 documentation](https://huggingface.co/docs/transformers/model_doc/siglip2)

Transformers supports offline use after required files have been acquired: `HF_HUB_OFFLINE=1` prevents Hub HTTP calls and `local_files_only=True` restricts loading to local files. [Official offline setup](https://huggingface.co/docs/transformers/installation#offline-mode) PyTorch provides the macOS `mps` device and an availability check via `torch.backends.mps.is_available()`. This establishes a possible evaluation route, not verified SigLIP2 compatibility or throughput on this Mac. [Official PyTorch MPS documentation](https://docs.pytorch.org/docs/2.14/notes/mps.html)

## Engineering implications to evaluate

These are inferences and trial questions, not architecture decisions:

- A fitted UMAP plus HDBSCAN prediction could populate an initial map progressively, but new subject groups require later discovery work. Compare it with occasional refits; measure time to useful results and movement between map revisions. Evaluate clustering separately from the appearance of the two-dimensional map.
- Freezing an opened cohort's member list is a separate product responsibility. Algorithmic refits can change both membership and labels; neither raw cluster numbers nor shared map coordinates establish durable cohort identity.
- Measure the unassigned fraction and inspect those images. Preserve access to them while evaluating singleton piles, a visible unassigned collection, or another assignment policy; none is selected here. Do not interpret the accepted single-membership rule as permission to hide noise images.
- Evaluate embedding quality, preprocessing effects, peak memory, cold/warm processing time, cancellation and map responsiveness on representative data before extrapolating to 50,000 images. No models, packages or library images were downloaded or processed for this note.
