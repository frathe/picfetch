# Close Files During Initial Work

## Problem

An empty viewer disabled **File > Close Files** solely because no files had
been committed. During an initial scan and its first sort, that made the menu
action unavailable even though it is the visible command for cancelling those
operations.

## Acceptance criteria

1. Close Files remains disabled for an idle, empty viewer, but is enabled while
   an initial scan or sort is active.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui -run 'TestCloseFilesItem_' -count=1`
2. Other file-dependent commands remain disabled while no files are committed.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/menus -run '^TestApply_FileItems$' -count=1`
3. Scan and sort transitions publish the new menu state when work starts and
   ends.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/... -count=1`

## Implementation

- Extend the menu snapshot with the active scan/sort fact without changing the
  meaning of `NoFiles` for other commands.
- Synchronize menus at scan and sort lifecycle boundaries.
- Extend the existing menu-state tests; no new top-level UI test or shard entry
  is required.

## Non-goals and limit

This does not change Escape handling, scan limits, sorting, or cancellation
semantics. It only restores access to an already-supported cancellation action.
