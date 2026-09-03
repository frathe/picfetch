# 03: Enforce safe and complete shard assignment

**What to build:** Make an invalid UI shard assignment impossible to accept as
a successful test run. Maintainers receive a build-aware Linux guard and exact
test filters that fail closed whenever current test inventory and the reviewed
manifest diverge.

**Blocked by:** 02: Generate the deterministic shard manifest.

**Status:** resolved

- [x] The guard obtains the selected test files and runnable top-level test,
      fuzz, and example inventory from the current Linux build context.
- [x] The guard compares exact sets and rejects every unassigned current name,
      duplicate assignment, stale name, malformed row, unknown shard, and empty
      shard.
- [x] Every rejection names the offending test, row, or shard and exits
      nonzero; the guard never repairs or broadly reshuffles the manifest by
      itself.
- [x] A newly introduced parallel-test call in a Linux-selected test file of
      the exact main UI package fails with a message requiring explicit safety
      review.
- [x] The guard ignores `TestMain` as harness code and keeps every slash-named
      subtest under its top-level parent.
- [x] Filter generation validates the complete manifest before producing one
      shard's expression.
- [x] Every produced filter is valid for Go's regular-expression engine,
      escaped, anchored at both ends, non-empty, and incapable of matching a
      name assigned to another shard.
- [x] Deliberate fixtures demonstrate rejection of missing, duplicate, stale,
      malformed, unknown-shard, empty-shard, and parallel-call cases.
- [x] Running the canonical live check under Linux/amd64 succeeds twice without
      changing the checked-in manifest.
- [x] A Darwin-derived inventory cannot be mistaken for the canonical Linux
      acceptance result.

## Comments

- 2026-09-03: Claimed. The established seam is the `scripts/testshards`
  command boundary. Implementation will add fail-closed `check` and `regex`
  paths plus a public Linux/amd64 Docker check; no manifest regeneration or
  application behavior is in scope.
- 2026-09-03: Resolved. `check` derives build-selected test files with
  `go list`, derives runnable top-level names with `go test -list`, compares
  exact manifest membership, scans only those files for `.Parallel()` calls,
  and refuses canonical success outside a matching Linux/amd64 executable and
  Go build context. `regex` validates all rows and shards before emitting an
  escaped, anchored, non-empty expression.
- 2026-09-03: `go test ./scripts/testshards -count=1`, `make vet`,
  `make fmt-check`, `make check-qodana-test-exclusions`, and `git diff --check`
  pass. Two independent `make check-test-shards` Docker runs each reported
  567 runnable names across three shards. The checked-in manifest remained
  byte-identical at SHA-256
  `6468a50ce0925ade39d253be7e7b0e728aa3d38fdc9f32a6b5547347155c5a18`.
