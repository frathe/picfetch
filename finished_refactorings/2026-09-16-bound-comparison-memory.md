# Bound comparison memory

## Problem

Comparison starts both selected-source loads concurrently. Each load previously
used the complete image-cache budget, so two decoded sources and unused GIF
frames could remain live beyond the shared budget.

## Acceptance criteria

1. Each pane is admitted against half the current shared image-cache budget
   before pixel decoding.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui -run '^TestCompareMemory_' -count=1`
2. Comparison decodes only the first frame of an animated GIF.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui -run '^TestCompareAnimated_' -count=1`
3. Existing comparison behavior remains green.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui -run '^TestCompare' -count=1`

## Non-goals and limit

This does not downsample sources that exceed admission. It rejects them through
the existing comparison failure path. Splitting the budget equally is
deliberately conservative: it preserves concurrent loading without needing an
allocation reservation service, at the cost of rejecting an asymmetric pair
whose combined estimate might otherwise fit.

## Tasks

### Task 1 - Guard comparison decoding

Owner: T0 inline
Files: `internal/ui/compare.go`, `internal/imaging/gif.go`, existing UI tests
Test: comparison budget and animated-source regressions in the existing test file
Verify: acceptance commands above
Budget: 0 spawns, 1 review round, full suite yes

## Verification evidence

- The focused comparison and cache-writer regressions pass.
- Imaging GIF fallback regressions pass.
- `make fmt` and `make vet` pass.
- `make verify` and `make check-test-shards` could not start because Docker is
  not installed in the environment. `make test-native` reached the full UI
  suite; it exposed and prompted correction of the animated-cache contract,
  while an unrelated Explorer settings test failed and then passed in isolation.
- GoLand/Qodana inspection tooling is unavailable in the environment.
