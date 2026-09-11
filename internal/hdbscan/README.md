# HDBSCAN provenance

This directory contains the MIT-licensed HDBSCAN implementation from
[PhotoPrism pkg/vector/alg](https://github.com/photoprism/photoprism/tree/c48d23f6b03c25fc19d376d789fac56c32a26fdb/pkg/vector/alg),
pinned to commit `c48d23f6b03c25fc19d376d789fac56c32a26fdb`.
The independent [upstream subtree license](https://github.com/photoprism/photoprism/blob/c48d23f6b03c25fc19d376d789fac56c32a26fdb/pkg/vector/alg/LICENSE)
is retained verbatim in `LICENSE`. No PhotoPrism module dependency or code from
the previous GPL-licensed clustering package is included.

Copied production files: `hdbscan.go`, `labels.go`, `union_find.go`.
`common.go` contains only `dataDims` from upstream `common.go`, `DistFunc` and
`EuclideanDist` from `clusters.go`, and four required errors from `errors.go`.
The package is renamed from `alg` to `hdbscan`; Go formatting is normalized.

Copied synthetic tests: `hdbscan_test.go`, `labels_test.go`, `sets_test.go`.
The DBSCAN comparison test and `dbscanLabels` helper are omitted because they
require the unrelated DBSCAN implementation. The unused `sameClustering` helper
is omitted; the empty-slice declaration uses idiomatic nil initialization. Other
HDBSCAN tests are retained.

`minPts` includes the point itself. Clusters are numbered from 1; `Noise` is -1.
The upstream algorithm excludes the root cluster. PicFetch's adapter retains
the root as one cohort when no smaller cluster qualifies and at least four
distinct sources remain, preserving the existing small-collection behavior.
Noise remains unassigned when smaller clusters are selected. PicFetch uses one
worker, converts the existing float32 reduction exactly to float64, and derives
cohort IDs from membership.
