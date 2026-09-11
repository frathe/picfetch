# PR 18: cohort navigation, settings, and repeated sources

Route: Deep review batch across UI composition, menus, worker, and two manuals.
Root owns assessment, regression design, implementation, and review. The existing
read-only Scout maps worker test coverage while root works on UI behavior.

## Tasks and acceptance criteria

1. Enable Window -> Viewer from a cohort image and leave the cohort through the
   existing callback, preserving comparison isolation. Files: menus/menus.go,
   menus/menus_test.go, explorer_test.go. Verify TestApply_WindowViewer and the
   existing TestVisualSimilarityExplorer/cohort_viewer_menu subtest.
2. Duplicate-distance changes retire an Explorer built from filtered sources,
   cancel stale delivery, and allow fresh preparation. Same-value changes and
   analyses without duplicate hiding stay valid. Files: memlimits.go,
   explorer_test.go. Verify duplicate_distance_changes subtests.
3. Within an analysis, reuse a successful representation only while its source
   identity, size, and modification time match; retain all input entries and
   their accounting. Files: similarity/analyze.go, explorer_local_test.go.
   Verify real-worker repeated_source_cohorts accounting and existing
   favorite_cache version-invalidation coverage.
4. State Apple Silicon macOS or x86-64 Linux consistently in both manuals.
   Files: help/manual.md and manual_de.md. Inspect wording and run the existing
   manual Unicode-arrow guard. No new UI keys or source strings.

Tasks 1–4 are independent; root implements all. Task 3 Scout is a bounded
cross-package inventory with file:line and test-name output, no edits/review.
Budget: one reused Scout, focused local tests only, complete suite on GitHub CI.
No new packages or top-level UI tests. Update todos and move this record after
implementation; fresh external reviews and CI remain the PR exit gate.

## Evidence

Review source: c62fe35, discussions r3984330048, r3984330050, r3984330054,
and r3984330056. All four findings confirmed and fixed.

- Before fixes, the Viewer item was disabled in a cohort image, running and
  completed filtered analyses survived distance changes, and 40 repeated inputs
  caused 40 inference attempts with zero reuse. All failed for the reported cause.
- After fixes, focused cohort/settings/preparation and real-worker reuse/cache
  tests passed in 14.779s:

  ```sh
  go test -tags explorertrial ./internal/ui -run '^TestVisualSimilarityExplorer$/(cohort_viewer_menu|duplicate_distance_changes|duplicate_representatives|pending_duplicates)$|^TestVisualSimilarityExplorerLocal$/(repeated_source_cohorts|favorite_cache)$' -count=1 -timeout 3m
  ```

- A further menu case exposed that the cohort override must preserve comparison
  isolation. It failed before that correction; TestApply_WindowViewer now passes
  (0.006s), including the new cohort and comparison cases.
- The worker retains all 40 entries, performs eight inference attempts, and
  reports 32 reuses. Four copies of one source require one inference attempt.
  Its request-local lookup checks file identity, size, and modification time.
- Settings tests cover active/completed analysis, same-value changes, duplicate
  hiding off, stale completion rejection, and reopening after retirement.
- Both manuals now name Apple Silicon macOS and x86-64 Linux and describe map
  retirement after duplicate-distance changes. Existing embedding/Unicode-arrow
  guards pass (0.007s); goimports and git diff --check are clean.

Ledger: one reused read-only Scout for test inventory; no delegated reviews or
fixes. Focused local verification only. Full CI and subsequent automated review
results are recorded on PR 18; no merge or release is part of this workflow.
