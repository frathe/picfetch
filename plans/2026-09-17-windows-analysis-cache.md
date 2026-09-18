# Windows analysis cache path regression

Restore Favorite and general analysis representation reuse for the file URI
paths supplied by the Windows UI. Route: Deep (Windows-specific behavior), with
one bounded implementation task. No dependencies or distribution changes.

## Evidence and decisions

Fyne URI paths use forward slashes on Windows. `decodeRepresentation` compared
them directly with `filepath.Clean`, which emits backslashes. General writes
failed validation; Favorite writes succeeded but their reads and stale checks
rejected the records. Existing fixtures used native paths and missed this.

- Preserve the exact source path and hash identity in saved records and events.
- Permit the platform's equivalent slash spelling during validation; continue
  rejecting relative paths and redundant path components.
- Preserve cache preferences, leases, capacity and producer write scopes.
  Explorer and search write compatible loose-image representations only while
  the existing loose-cache preference is enabled.
- Do not change unrelated Windows fixture assumptions or native file behavior.

## Acceptance and task graph

1. UI-originated paths persist, reopen with the same vector/preview and identity,
   appear in usage, and survive stale cleanup in both stores.
   `go test -tags no_emoji,nodynamic ./internal/similarity -run '^TestAnalysisCacheFileURIPathsReopen$' -count=1 -v`
2. Native and UI paths are valid; empty, relative and redundant paths remain
   invalid without rewriting valid source identities.
   `go test -tags no_emoji,nodynamic ./internal/similarity -run '^TestAnalysisCachePayloadPathValidation$' -count=1 -v`
3. Existing cache policy and lifecycle checks retain their behavior.
   `go test -tags no_emoji,nodynamic ./internal/similarity -run 'TestAnalysisCache|TestFavoriteAnalysis' -count=1`
   Record pre-existing native fixture failures separately; run Linux verification.

Recon -> regression tests (red) -> shared decoder fix (green) -> lead review and
verification. The read-only UI request scout ran alongside storage diagnosis.

### Task 1 - Shared path validation

Owner: T0 inline. Files: `internal/similarity/cache_payload.go`,
`internal/similarity/cache_store_test.go`, this evidence record, `todos.md`.
Contract: unchanged `decodeRepresentation`; no exported APIs or new packages.
Test/verify: acceptance commands above, native Windows first.
Budget: one read-only scout, one lead review, one final `make verify` attempt.
Scout delegation: independent UI wiring search, no edits, file/line evidence;
small prompt, bounded oracle (`rg`/source references), no shared mutations or hot
storage context. No implementation or review delegated.

## Verification evidence

- Red: `TestAnalysisCacheFileURIPathsReopen` fails natively on Windows: general
  reports `incompatible analysis record`; Favorite cannot reopen its record.
- Before changes, existing native cache tests already fail in
  `TestAnalysisCacheMaintenancePartialFailure` (Unix unreadable fixture),
  `TestAnalysisCacheConfinementManagedUsageAndTemps` (URI/native path comparison),
  and `TestFavoriteAnalysisFollowsOpenedDirectory` (open-directory rename).
- Red: the path-validation test rejects the Fyne URI path while the native
  path and all invalid-path expectations behave correctly.
- Green on Windows: both new tests pass, including both store reopenings,
  unchanged vector/preview/path, usage inspection and stale cleanup retention.
- Existing native cache regressions pass with the two pre-existing failing
  `TestAnalysisCache` fixtures excluded explicitly:
  `go test -tags no_emoji,nodynamic ./internal/similarity -run 'TestAnalysisCache' -skip 'TestAnalysisCache(MaintenancePartialFailure|ConfinementManagedUsageAndTemps)$' -count=1`
  Output: `ok github.com/frathe/picfetch/internal/similarity 5.299s`.
- Cache/search pipeline selection also passes:
  `go test -tags no_emoji,nodynamic ./internal/similarity -run 'TestSearch.*(Cache|Pipeline|Prepare)|TestAnalysisCache(FileURIPathsReopen|PayloadPathValidation)' -count=1`
  Output: `ok github.com/frathe/picfetch/internal/similarity 43.503s`.
- Package `go vet` and `go build` with `-tags no_emoji,nodynamic` pass.
  Changed-file `go tool goimports -local github.com/frathe/picfetch -l` emits
  nothing, and `git diff --check` passes. Repository-wide goimports reports
  existing files throughout the CRLF checkout and ignored scratch directory;
  no unrelated formatting was changed.
- Full `make verify` attempted with Git's sh: failed at `check-test-platform`
  because the Windows shell could not fork (`Resource temporarily unavailable`).
  Docker itself reports native `linux/x86_64`, but only 16,592,285,696 bytes,
  below the required 16 GiB. The memory requirement was not reduced.
- Independent `make vet` fails because native cgo is disabled and the OpenGL
  package has no selected files; `make build` cannot launch `mkdir -p` with the
  default Windows shell. Native application build/full race suite are unverified.
- Linux cache execution was also attempted by cross-compiling the test binary;
  no-cgo Linux compilation fails at `internal/displays` (`platformInspect`
  undefined). Available Docker images lack the native compiler/X11 toolchain.
  No Linux test result is claimed.
- GoLand inspection is unverified: the configured MCP endpoint on port 64422
  refuses connections; the active IDE endpoint on 64522 requires authorization
  unavailable through this session. No GoLand inspection tools are exposed.
- Lead reviewed the final diff against the acceptance criteria: only shared
  validation changes; existing entries retain their keys and do not require a
  format migration. Existing test file is already in Qodana exclusions; no new
  root UI tests or package-map updates are needed.

Implementation and focused verification are complete. Keep this record in
`plans/` until the outstanding environment-dependent gates are completed.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| Recon and fix | 1 / 1 read-only | 1 lead | no |
| Final gate | 0 / 0 | 1 lead | attempted; blocked by environment |
