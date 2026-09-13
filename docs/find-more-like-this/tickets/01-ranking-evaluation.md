# FML-001 — Evaluate ranking quality and resource costs

Status: planned.
Type: prototype/research.
Owner: T0 lead.
Depends on: none.
Acceptance: AC-01 in the [plan](../plan.md).
Budget: zero spawns; at most two lead review rounds; focused checks only.

## Files

- Add `scripts/explorereval/search.go` and `search_test.go`; extend its `main.go`
  and `README.md` with a development-only `-search-evaluate` entry.
- Add real-model search evaluation coverage under the existing `explorertrial`
  build tag. Add exact test paths to `qodana.yaml`.
- Record aggregate findings in `docs/find-more-like-this/evaluation.md` when run;
  keep the reproducible corpus/manifest, labeled judgments, and image report in
  the selected local evidence directory.

## Contract and work

Reuse the pinned encoder, asset verification, oriented decoding, and existing
offline experiment launcher. Use at least 20 reference images across multiple
subjects, compositions, lighting conditions, and folders. Supply a manifest
with reference IDs, human-labeled relevant candidate IDs, and intent labels
such as subject or visual appearance. Store that manifest as
`search-corpus.json` in the directory supplied by `-library`; source entries use
stable IDs and relative image paths. Include duplicates and hard negatives.

Emit exact cosine top-30 results, per-query precision at 10, median precision,
separate intent summaries, skipped sources, corpus/model/preprocessing identity,
cold preparation time, warm query p50/p95, and retained-vector/native RSS
measurements with machine details. Record first-index and warm behavior
separately; report larger synthetic-vector rank timings at 10,000 candidates.

Initial usefulness target: median precision at 10 of at least 0.6 over the
labeled queries, with missing result slots counted as non-relevant. Initial
warm-ranking target: p95 under 200 ms for 10,000
prepared vectors on the recorded development machine. These are evaluation
targets, not shipping claims for every computer or every visual intent.
Record a proceed/revise decision and the human assessment; do not count a
subject match as proof of reliable style/composition matching. An unmet target
requires a documented scope/algorithm decision before FML-002, not a new model
or dependency selected implicitly.

## Acceptance and verification

- Fixture tests reject self-matches, malformed manifests, mislabeled result
  counts, and incomplete reports; repeated runs retain deterministic IDs/order.
- Real evidence covers all 20 references and includes judgments, timing/RSS,
  model identity, and an explicit proceed/revise decision.
- No native/model result is inferred from synthetic fixtures.

```sh
go test ./scripts/explorereval -run '^TestSearchEvaluation' -count=1
```

Real run, with the three variables pointing to prepared local directories:

```sh
go run ./scripts/explorereval -search-evaluate -library "$PICFETCH_SEARCH_CORPUS" -assets "$PICFETCH_SEARCH_ASSETS" -out "$PICFETCH_SEARCH_EVIDENCE"
```

Done when the report is reproducible, its human-quality decision is recorded,
and its measured limitations are reflected in the plan.
