# 04: Run the sharded race contract locally

**What to build:** Give contributors one canonical local verification path
that exercises precisely the package partition and three UI shards intended
for hosted CI. It runs sequentially in one Linux/amd64 container for practical
local use while preserving the flags, inventory, diagnostics, and raw evidence
that parallel CI will use.

**Blocked by:** 03: Enforce safe and complete shard assignment.

**Status:** resolved

- [x] The normal non-race Docker test command remains a complete unsharded
      suite with its current behavior.
- [x] The race and full verification commands first run the canonical Linux
      manifest guard, then execute the non-UI partition and all three UI shards
      sequentially in one Linux/amd64 container.
- [x] The non-UI partition is the complete module package inventory minus only
      the exact main UI package; root, tooling, and every UI feature subpackage
      remain included.
- [x] An empty or inconsistent package partition fails before tests run.
- [x] Every race partition shares the English UTF-8 locale, race detector,
      fresh execution count, and 30-minute package timeout.
- [x] Direct in-container commands are available for the guard, non-UI tests,
      and one selected UI shard so hosted CI can call the same contract without
      nesting Docker.
- [x] Test capture prints concise package/test/failure information while
      preserving a readable raw event stream.
- [x] A deliberate failed stream proves that either a test failure or capture
      failure keeps the overall pipeline failed and identifies the exact
      package, shard, and test.
- [x] The public manifest-check command enters Linux/amd64 Docker; a local
      macOS inventory is never treated as authoritative.
- [x] The full repository verification command passes once through this final
      sequential race path, with no golden screenshots regenerated and no raw
      timing files left in the repository.

## Comments

- 2026-09-03: Claimed. The established seams are the public and prepared-runner
  Make targets plus the `scripts/testshards` command boundary. The normal
  unsharded `make test` behavior remains fixed; application code, hosted-CI
  topology, translations, and golden files are out of scope.
- 2026-09-03: Resolved. `test-race` now enters one Linux/amd64 container and
  runs the live manifest guard, the module inventory minus only exact
  `internal/ui`, then `ui-1`, `ui-2`, and `ui-3`. Prepared runners can call
  `check-test-shards-direct`, `test-race-non-ui-direct`, or
  `test-race-ui-direct TEST_SHARD=ui-N`; all race commands visibly share the
  locale, race, fresh-count, timeout, JSON-capture, and pipefail contract.
- 2026-09-03: `go test ./scripts/testshards -count=1 -v`,
  `make check-test-shards`, and the final `make verify` pass. The final race
  rerun checked 567 UI runnables, passed the complete non-UI partition, then
  passed `ui-1` in 229.604s, `ui-2` in 228.490s, and `ui-3` in 211.661s. The
  repository inventory was 45 packages = 44 non-UI + exact UI.
- 2026-09-03: A deliberate failed test stream returned pipeline status 1 and
  named `non-ui`, `scripts/testshards`, and
  `TestDeliberateCaptureFailure`; a forced capture-open failure returned
  capture status 1. Capture also reproduced an existing 2,481,918-byte raw CI
  stream byte-for-byte. The temporary failure fixture was removed, and no raw
  stream or failed golden render remains in the repository.
