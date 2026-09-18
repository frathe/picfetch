# Preserve large-Favorite preview reuse

The initial 256-source policy below is now configurable, defaulting to 1000
at the user's request; see `2026-09-18-favorite-preview-limit-setting.md`.
The 256-entry cases remain useful coverage of a selected custom limit.

Route: Standard; lead owns implementation and review. No new dependency,
background lifetime, UI string or Grid interface. One read-only scout mapped
existing cache paths while the lead examined the measured regression.

The PR's 256-source prefix bounds automatic original-image decoding but also
deletes valid previews belonging to later entries. A 300-image synthetic Grid
comparison read 0 originals at the base and 44 on this PR. Preserve the decode
bound while reusing cheap existing previews throughout the Favorite. Pruning
must use the complete membership, not the decode-admission prefix.

1. Keep preparation for original decoding capped at 256 unique paths and
   decoding serial. Beyond that prefix, permit memory/disk reuse only, never
   original-source reads. Verify focused `internal/favthumbs` tests, including
   a reader that rejects original reads after the prefix.
2. Keep current tail previews, warm the byte-budgeted Grid sink from them, and
   still remove obsolete previews for sources no longer in the Favorite.
   Metadata/cached-preview traversal is cancellable; bookkeeping for pruning
   follows collection membership, while original decode admission stays capped.
   Verify package tests plus the existing 300-image comparison harness.
   Background admission must not evict existing Grid thumbnails when there
   is some free memory but insufficient room for the next thumbnail. Verify
   `TestSyncFavoritePreviews_DoesNotEvictGridThumbnails`; use the existing
   generation-bound `AddIfRoom` operation for atomic admission.
3. Preserve cancellation/offline retention and source-version checks. Verify
   the package race suite and focused favorite-preview UI regressions.
4. Update the two manuals, architecture and prior evidence to distinguish
   source decode limits from valid disk-cache retention. Run GoLand inspections,
   formatting, manual guards, vet/build, then exact-commit CI and bot review.

Already deleted previews cannot be recovered from their cache files. A cold
tail still loads on demand; this change preserves and reuses existing work.
This does not claim to restore eager original decoding over the whole list.

Files: `internal/favthumbs/{sync,sweep}.go`, their existing test files,
`internal/ui/help/manual{,_de}.md`, `ARCHITECTURE.md`, evidence and `todos.md`.
The near-full-cache finding also touches root `favthumbs.go` and its existing
tests, the UI shard manifest and outdated Grid cache accessor documentation.
Budget: one scout; lead-only fixes; focused local race tests, complete CI.

## Verification

- The new retention/reuse test failed before the fix: the tail preview was
  deleted and was not offered to Grid. After the fix the package race suite
  passed (3.100s), including cancellation and source-version regressions;
  focused viewer preview tests passed (0.295s).
- The original 300-image Grid comparison now warms all 300 previews, reads
  zero originals during Grid traversal and reaches the final item (1.85s),
  matching the base instead of the PR's earlier 44 unnecessary source reads.
  The harness ran under a separate 1 GiB service with no swap.
- Temporary overlays confirmed the tests reject decoding tail originals,
  skipping tail reuse, admitting 257 originals, and ignoring sweep cancellation.
- GoLand is clear on the changed implementation and sync test files. An
  existing duplicate fixture in the touched sweep test has a function-scoped
  suppression, preserving independent changed/removed/retained-source cases.
- Final GoLand inspections, formatting, exclusions, vet/build and the manual
  guard passed. No new test file or top-level internal/ui test was added in
  this follow-up. Exact-head hosted review remains pending.

### Near-full-cache follow-up

A 192-byte cache holding one 128-byte versioned thumbnail was below the old
`ThumbCacheFull` threshold. Warming another 128-byte thumbnail evicted the
existing one, so the previous pre-check did not provide its documented bound.
The new integration regression failed before switching the sink to AddIfRoom.
The new top-level UI test is assigned to ui-1 in the shard manifest; its file
already has a Qodana test exclusion. No cache API or dependency changes.
Focused UI race regressions passed (4.449s), including source-version and
cancellation coverage. The updated shard check accounts for 692 tests.
GoLand's existing package-shadowing warnings in the touched fixture file were
fixed by naming the URIs `source`; three independent lifecycle/open fixtures
have narrowly scoped duplicate-code suppressions with their rationale.
Final GoLand inspections, focused fixture checks, formatting, exclusions,
vet and build pass for this follow-up.

### Review follow-up: stale memory and corrupt disk entries

The next Codex review identified two gaps in the cache-only path. Lead will
reproduce and fix both, with one read-only scout locating existing disk-cache
tests while the lead examines memory admission. No implementation is delegated.

- Refresh a changed source's thumbnail atomically without evicting unrelated
  keys or promoting an existing key. If its new pixels cannot fit, remove the
  obsolete key so Grid can load the source on demand. Preserve AddIfRoom's
  existing display-preload contract and pre-purge writer rejection.
  Verify focused ByteCache tests and
  `TestSyncFavoritePreviews_RefreshesChangedGridThumbnails`.
- A failed disk-preview decode must allow a later versioned memory thumbnail
  to repair the cache, without reading a tail original in the eager pass or
  rewriting healthy existing previews. Verify
  `TestSyncBoundsOriginalDecodesAndRetainsCachedTail` and the package race suite.
- Update cache documentation and the UI shard manifest; run GoLand, formatting,
  vet/build and fresh hosted review. These fixes extend the existing Standard
  work to the shared byte cache; no new dependency or background worker.

Verification: both reports reproduced before the fixes. The stale-key test
failed with and without room for the replacement; the tail test could not
persist fresh memory pixels over the corrupt cache entry. Both pass after the
fixes. Focused race runs passed for Favorite previews, imaging and viewer
integration (3.132s, 11.636s, 18.994s). The complete Favorite-cache race suite
passed (4.147s), as did the new byte-cache guard and existing display preload
contract (1.056s, 1.065s). Temporary overlays were rejected for LRU promotion,
pre-purge publication and retaining an unadmitted stale key. GoLand is clear
on all six changed Go files, including warnings. The manifest now covers 693
UI tests. Full CI and security/Qodana/CodeQL passed on the previous `4eb5300`;
its two code-review findings are addressed here and require another fresh
review. The bounded native comparison remains pending.

The review of `5f8dae2` found three additional cases: the warming lookup itself
promotes entries before refresh; repeated tail paths decode cached previews
again when memory admission declines; and corrupt-file identity checking is
not atomic with removal. Lead will cover lookup recency and one-off tail work
in the existing regressions, and serialize preview replacement with failed-entry
cleanup using a runtime commit mutex. Encoding/decoding stays outside that
short critical section. Verify the Favorite-cache race suite and focused viewer
preview tests, then rerun exact-head hosted review. The newly requested saved
limit defaults to 1000 and is tracked in the separate settings plan.

These three reports were confirmed. The existing regressions failed for 267
preview offers instead of 257 distinct sources and for warming promoting an
older thumbnail over one the user viewed more recently. They now pass using
generation-bound Peek and one visit per tail path. The cleanup regression
also exposed recycled file identities after closing a delayed reader: retain
its handle until entering the same commit mutex as writers. Encoding/decoding
stays outside the critical section, with cancellation checked inside it.
The complete preview race suite passes (3.293s); overlays removing the cleanup
lock or identity guard fail for the expected reasons. All changed code files
pass GoLand inspections. One pre-existing duplicated source-version fixture
has a narrowly scoped suppression explaining its independent coverage.
