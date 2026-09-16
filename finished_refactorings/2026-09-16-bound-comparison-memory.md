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

## Original implementation verification

- The focused comparison and cache-writer regressions pass.
- Imaging GIF fallback regressions pass.
- `make fmt` and `make vet` pass.
- `make verify` and `make check-test-shards` could not start because Docker is
  not installed in the environment. `make test-native` reached the full UI
  suite; it exposed and prompted correction of the animated-cache contract,
  while an unrelated Explorer settings test failed and then passed in isolation.
- GoLand/Qodana inspection tooling is unavailable in the environment.

## PR #31 review follow-up

Route: Standard. The lead owns the fixes and review. One read-only scout locates
existing fixture and catalogue-verification patterns; no implementation or
review is delegated. No dependencies or distribution obligations change.

Acceptance criteria and tasks, all owned by T0:

1. Replace the renamed comparison test in the shard manifest, assign any new
   comparison regression, and keep the entry counts current.
   Verify: `make check-test-shards`.
2. Header-only decoded-pixel estimates cover eight-byte RGBA64/NRGBA64 pixels.
   Cached comparisons charge the detached first-frame record's actual retained
   bytes, including vector storage, and fresh decodes receive the same final
   check. Cover accepted and refused tiny 16-bit sources with and without a
   cache hit, and preserve complete cached animation records.
   Verify: `go test -tags no_emoji,nodynamic ./internal/imaging ./internal/ui -run '^(TestEstimateDecodedBytes|TestImageBytes|TestCompare|TestImageCacheWriters_PreserveCompleteRecords)' -count=1`.
3. Budget refusals reach a localized toast, with English/German catalogue keys.
   Verify: the comparison memory regressions above and
   `go test -tags no_emoji,nodynamic . -run '^TestTranslations_' -count=1`.

Files: comparison adapter/tests, imaging byte accounting/tests, shard manifest,
English/German catalogues, this evidence record and `todos.md`. Update the
preload test's estimate comment because it uses the shared conservative helper.
The shared helper also makes speculative preload admission more conservative;
actual byte-cache weights remain format-specific. These are decoded-image
admission limits, not a bound on all decoder scratch, renderer or process memory.

Verification: targeted regressions first, then GoLand inspections and
`make verify`; native amd64 CI supplies the complete suite if the local Docker
daemon cannot meet the required platform. Commit and push are user-authorized.
Budget: one scout, one final review, one complete-suite gate.

### Follow-up verification evidence

- Before the fix, the tiny-fixture tests failed for underestimated RGBA64 and
  NRGBA64 storage, admission of a 16-bit source above its pane allowance on
  both fresh and cached paths, and the untranslated budget-refusal toast.
- After the fix, focused comparison, full-cache-record, byte-accounting,
  preload and GIF-fallback regressions pass. The complete native `imaging`,
  `ui/display` and `ui/compare` package suites also pass.
- `go test -tags no_emoji,nodynamic . -run '^TestTranslations_' -count=1` passes.
- `make check-test-shards` passes in Linux/amd64 Docker: 687 runnable tests,
  three shards. Its earlier run failed on the unassigned comparison test.
- `make verify-build` passes: formatting, TUF root, Qodana test exclusions,
  generated assets/notices, vet and build. `git diff --check` is clean.
- GoLand inspected all PR code files and both changed catalogues, including
  weak warnings. The only findings are two unchanged duplicate setup fragments
  in `imgcache_test.go`, in tests of different removal entry points. The exact
  file is already excluded from `DuplicatedCode` in `qodana.yaml`; retain that
  existing test-only exclusion. No production issue or new inspection finding
  remains. The unused `assertFrozen` parameter introduced by the PR was removed
  by using its existing assertion helper directly.
- `make verify` refuses the local Docker daemon's `linux/aarch64` platform.
  The full race suite remains pending native Linux/amd64 CI after the push;
  no isolation policy or test was weakened to bypass this prerequisite.

Actual cost: one read-only scout, one lead review, one full-gate attempt.
The user authorized the fix commit and push to PR #31; the unrelated untracked
GitHub AI scan failure report is excluded from the commit.
