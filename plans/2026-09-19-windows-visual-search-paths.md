# Restore Windows visual-search results

Route: Standard, bounded Windows bug fix. The lead owns diagnosis, design,
tests, implementation and review. A read-only scout located native coverage
and local verification prerequisites while the lead traced production paths.

## Problem and decisions

Fyne file URI paths use forward slashes on Windows. `runSearchSession` cleans
them into backslash paths and publishes those identities to Grid, whose
collection index compares exact paths. Existing transport tests pass but do
not connect production ranking to URI-backed Grid results.

Preserve the first admitted source spelling through preparation and published
matches. Continue using cleaned paths for deduplication and reference lookup.
Do not change shared collection identity, ranking, worker policy, dependencies,
user-visible strings, or unrelated Windows behavior.

## Acceptance and tasks

1. Source paths round-trip unchanged for native, URI and mixed-spelling inputs;
   equivalent paths prepare once, and a later reference still ranks correctly.
   Lead: `internal/similarity/search.go`, `search_session_test.go`.
   Verify: `go test -tags no_emoji,nodynamic ./internal/similarity -run
   TestSearchSessionPreservesSourcePaths -count=1` on Windows (red, then green).
2. Run this regression in Windows CI without model downloads.
   Lead: `.github/workflows/ci.yml`, in the existing Windows job.
   Verify: `go test -tags no_emoji,nodynamic ./internal/similarity -run
   '^TestSearch' -count=1 -timeout 3m`. Keep the complete Linux suite unchanged.
3. Verify real Windows inference, retained search and visible Grid results with
   installed pinned assets: `go test -tags no_emoji,nodynamic,explorertrial
   ./internal/ui -run '^TestVisualSimilarityExplorerLocal/(explorer_general_cache_warms_search|new_search_favorite_reuses_general_analysis|saving_search_favorite_without_loose_cache_reuses_vectors)$'
   with the available native cgo compiler. Run focused UI search regressions,
   GoLand inspections of all changed code, and `make verify` using Linux/amd64
   Docker. Record unavailable or incomplete evidence explicitly.

Task graph: regression -> implementation -> CI guard -> final verification.
Budget: one read-only scout, no implementation delegation, one final full suite;
lead handles any findings inline. Scout scope is independent read-only coverage
discovery with file:line evidence, no shared writes, small context and no review.

## Evidence

- Initial native Windows search/worker tests pass (3.868s), confirming the gap.
- Docker reports Linux/x86_64; a portable Windows cgo compiler and the pinned
  production model/runtime are available locally.
- No dependency or distribution changes.
- New regression fails on Windows for URI/mixed source spelling; the existing
  real-worker `explorer_general_cache_warms_search` test also fails before the
  fix with zero reused records and native-spelling matches missing from Grid.
- All three real-worker Windows search/cache acceptance tests pass after the
  fix (8.664s). GoLand reports no findings in the changed search code/test.
- The complete similarity package exposed separate Windows cache-test issues
  (Unix permission assumptions, native/URI path assertion and open-directory
  rename sharing). Windows CI therefore runs all search tests in its own step,
  rather than adding unrelated cache tests to the native guard suite.
- Re-running those three cache tests with a Go overlay containing HEAD's
  pre-fix `search.go` reproduces the same failures (0.412s); tracked in `todos.md`.
- All native Windows `TestSearch*` tests pass (31.586s), including the observed-red
  identity regression. Root `TestFindMoreLikeThis*` regressions pass (4.847s).
- GoLand also reports no findings in the Windows CI workflow.
- Full Linux verification uses a fresh source snapshot with LF line endings,
  preserves binary assets, and runs the unchanged Make suite in native amd64
  Docker. `TEST_MEMORY_GIB=14` fits the daemon's 15.45 GiB available memory;
  the prepared Ubuntu verification image avoids re-downloading build tools.
- Linux formatting/generated-file checks, vet and build pass. The first race
  invocation started all UI shards but its non-UI inventory could not resolve
  the snapshot's shared Git objects inside Docker. Repacked the snapshot's
  objects locally and removed only its alternate-object pointer, then restarted
  the unchanged `test-race-non-ui-direct` target in the same container, writing
  `non-ui-retry.json`. UI shards continue from their original invocation.
- Native Windows similarity vet passes. macOS test/build cross-compilation
  without cgo is blocked by existing Cocoa-dependent platform integrations;
  native macOS runtime validation remains for CI.
