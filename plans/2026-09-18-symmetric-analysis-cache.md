# Symmetric loose-image analysis cache

Route: Deep. This was promoted from Standard when the correct narrow fix crossed
the `similarity`, `ui/explorer`, and root `ui` boundaries to preserve the
existing configured limit and automatic-eviction lifecycle. Deliverable: an
Explorer analysis warms a later **Find more like this** search when loose-image
analysis caching is enabled, using the same bounded cache policy as search.

## Problem and decisions

Explorer and search already read compatible Favorite/general representations,
but Explorer started its store in Favorite-only write mode. An Explorer-first
run over ordinary images therefore left no general records for a later search.
Search-first was warm; Explorer-first was cold.

| Decision | Contract |
| --- | --- |
| General representation ownership | Explorer and search write compatible loose-image representations only when the existing loose-cache preference is enabled. |
| Cache policy | Explorer receives the existing Cache-tab general byte limit at admission; no fallback to a different limit. |
| Capacity pressure | The analyzer retains the largest typed pressure request, exposes it only with its completed map, then root invokes the existing automatic eviction path. |
| Favorite retention | Preserve Favorite-first reads, Favorite opt-out, Favorite lifetime, and records outside the general size budget. |
| Live workers | Keep Explorer and search in separate native workers; do not transfer vectors through the UI or alter suspension barriers. |

Non-goals: physically merging Favorite and general stores; persisting map layout,
cohorts or UI state; changing Cache settings, memory limits, search ranking, or
the existing retention/eviction implementation.

Honest limit: the first enabled-cache Explorer pass still performs inference and
may fill the configured bounded general cache. A later search only reuses records
whose source and model/preprocessing identities still match.

## File map and routing

| Boundary | Contract |
| --- | --- |
| `internal/ui` -> `internal/ui/explorer` | `OpenRequest` captures the enabled general directory and exact byte limit. |
| `internal/ui/explorer` -> `internal/similarity` | The captured request configures `similarity.Client` before native admission. |
| `internal/similarity` -> `internal/ui/explorer` | `Event.CachePressureBytes` is nonzero only on the completed map; it retains the largest required temporary write reservation. |
| `internal/ui/explorer` -> `internal/ui` | The captured callback calls the existing analysis-cache `MakeRoom` mechanism after the map has completed. |

| Files | Ownership |
| --- | --- |
| `internal/similarity/{analyze.go,client.go,similarity.go}` | Analyzer cache policy, typed pressure propagation, worker request/event protocol. |
| `internal/similarity/analyze_test.go` | Non-native storage, configured-limit, and pressure regression coverage. |
| `internal/ui/explorer/{feature.go,workflow.go,feature_test.go}` | Captured request/pressure contracts and completed-map delivery. |
| `internal/ui/{explorer.go,features.go,explorer_test.go}` | Root policy capture and existing automatic eviction handoff. |
| `ARCHITECTURE.md`, manuals, active plans, `todos.md` | Current cache-policy documentation and work tracking. |

## Acceptance criteria

1. Explorer writes ordinary loose representations to the existing general store
   only when that cache is enabled, and honors the captured nondefault byte
   limit. Favorite records remain separate.
   Verify:
   `go test -tags no_emoji,nodynamic ./internal/similarity -run '^TestAnalyzerStorePersistsLooseRepresentations$' -count=1`
2. Analyzer records the maximum `CachePressureError` request without misreporting
   it as a cache warning; only the completed map transports that request.
   Verify:
   `go test -tags no_emoji,nodynamic ./internal/similarity -run '^TestAnalyzerCacheWriteReportsPressureAfterCompletion$' -count=1`
3. Explorer forwards pressure exactly once after a completed map, and root starts
   existing automatic eviction only then.
   Verify:
   `go test -tags no_emoji,nodynamic ./internal/ui/explorer -run '^TestFeatureReportsCompletedCachePressure$' -count=1`
   and
   `go test -tags no_emoji,nodynamic ./internal/ui -run '^TestVisualSimilarityExplorer$/cache_pressure_evicts_after_completed_map$' -count=1`
4. A completed Explorer pass over loose images with the existing loose-cache
   preference enabled warms a real later **Find more like this** worker.
   Verify:
   `go test -tags no_emoji,nodynamic,explorertrial ./internal/ui -run '^TestVisualSimilarityExplorerLocal$/explorer_general_cache_warms_search$' -count=1`
5. Existing producer scope remains safe when loose-image caching is disabled.
   Verify:
   `go test -tags no_emoji,nodynamic ./internal/similarity -run '^TestAnalysisCachePolicyProducerScope$' -count=1`
6. Architecture and manuals state the now-symmetric general-cache behavior and
   use no unsupported arrow glyphs.
   Verify:
   `go test -tags no_emoji,nodynamic ./internal/ui/help -run '^TestManualHasNoUnicodeArrows$' -count=1`
   and
   `go test -tags no_emoji,nodynamic ./internal/ui -run '^TestTranslationsHaveNoUnicodeArrows$' -count=1`

## Task graph

`T1 (red cache routing) -> T2 (analyzer policy and pressure) -> T3 (Explorer/root handoff) -> T4 (native path and docs) -> review/gate`

### Task 1 — Cache-routing and configured-limit regressions

Owner: T0 inline
Files: modify `internal/similarity/analyze_test.go`,
`internal/ui/explorer/feature_test.go`, `internal/ui/explorer_test.go`
Depends: none
Contract: pin loose-store routing, exact limit capture, typed pressure semantics,
completed-map delivery, and root eviction with no model assets.
Test: a disabled loose cache writes nothing; an enabled one writes one general
record at its exact configured limit; pressure is handled only after completion.
Verify: AC1, AC2 and AC3 commands.
Budget: 0 implementation spawns; 1 lead review round; full suite no.

### Task 2 — Preserve cache policy in the analyzer protocol

Owner: T0 inline
Files: modify `internal/similarity/analyze.go`, `internal/similarity/client.go`,
`internal/similarity/similarity.go`
Depends: Task 1
Contract: Analyzer selects enabled-store writes only with a general directory,
uses the request byte limit, and carries typed pressure only on a completed map.
Test: Task 1 regressions.
Verify: AC1, AC2 and AC5 commands.
Budget: 0 implementation spawns; 1 lead review round; full suite no.

### Task 3 — Capture policy and hand pressure to root maintenance

Owner: T0 inline
Files: modify `internal/ui/explorer.go`, `internal/ui/features.go`,
`internal/ui/explorer/feature.go`, `internal/ui/explorer/workflow.go`
Depends: Task 2
Contract: root captures the current Cache-tab policy before worker admission;
Explorer invokes its callback once after final delivery; root reuses `MakeRoom`.
Test: Task 1 regressions.
Verify: AC3 command.
Budget: 0 implementation spawns; 1 lead review round; full suite no.

### Task 4 — Native regression and policy record

Owner: T0 inline
Files: modify `internal/ui/explorer_local_test.go`, `ARCHITECTURE.md`,
`internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`,
`plans/2026-09-17-windows-analysis-cache.md`, `todos.md`, this plan
Depends: Task 3
Contract: retain an asset-qualified Explorer-to-search integration case and
record the shared bounded-store policy without changing settings or strings.
Test: AC4 and AC6.
Verify: AC4, AC6, `git diff --check`.
Budget: 0 implementation spawns; 1 lead review round; full suite no.

## Verification evidence

- Red before the original routing change:
  `go test -tags no_emoji,nodynamic ./internal/similarity -run '^TestAnalyzerStorePersistsLooseRepresentations$' -count=1`
  failed for `loose-enabled=true` because the general store contained zero
  records. This was the intended old Favorite-only behavior.
- Focused green after the policy/pressure fix:
  `go test -tags no_emoji,nodynamic ./internal/similarity -run '^(TestAnalyzerStorePersistsLooseRepresentations|TestAnalyzerCacheWriteReportsPressureAfterCompletion)$' -count=1`
  passed.
- Explorer delivery and root automatic-eviction regressions passed:
  `go test -tags no_emoji,nodynamic ./internal/ui/explorer -run '^TestFeatureReportsCompletedCachePressure$' -count=1`
  and
  `go test -tags no_emoji,nodynamic ./internal/ui -run '^TestVisualSimilarityExplorer$/cache_pressure_evicts_after_completed_map$' -count=1`.
- Broader focused race gates passed:
  `go test -race -tags no_emoji,nodynamic ./internal/similarity -count=1`
  (25.517s after the final event-protocol refinement),
  `go test -race -tags no_emoji,nodynamic ./internal/ui/explorer -count=1`
  (6.612s), and
  `go test -race -tags no_emoji,nodynamic ./internal/ui -run '^TestVisualSimilarityExplorer$' -count=1`
  (267.050s). The targeted **Find more like this** race selection also passed.
- Manual and translation arrow guards passed. `make fmt-check`, `make vet`, and
  `make build` passed. `git diff --check` was clean before final review.
- GoLand re-inspected every changed Go file after the final refinement. It
  reported only two unchanged weak duplicate-code warnings in
  `internal/ui/explorer_local_test.go` at lines 895 and 1014; both are
  pre-existing 80-name fixture blocks outside this diff, so no unrelated cleanup
  was made.
- The asset-qualified integration test compiles with `explorertrial`; its local
  execution remains unavailable because
  `.scratch/visual-similarity-explorer/assets/vision_model.onnx` is absent.
  No native cache claim is made until the pinned assets are installed.
- The fresh hosted PR review/CI evidence is appended after the push.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| Recon | 3 / 3 read-only | 1 lead | no | Independent routing, scope, and pressure scouts; T0 owned all fixes. |
| T1 | 0 / 0 | 1 lead | no | Non-native storage and lifecycle regressions. |
| T2 | 0 / 0 | 1 lead | no | Hot analyzer context. |
| T3 | 0 / 0 | 1 lead | no | Captured existing root maintenance seam. |
| T4 | 0 / 0 | 1 lead | no | Policy record and asset-qualified case. |
| Gate | — | 1 lead | CI | Existing PR loop supplies the broad hosted gate after push. |
