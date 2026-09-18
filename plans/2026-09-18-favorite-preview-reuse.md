# Preserve large-Favorite preview reuse

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
