# Private Favorite Storage

## Problem

Favorite lists contain absolute image paths, but `Save` creates their directories
as `0755` and changes their JSON files to `0644`. `DefaultDir` also falls back to
a predictable path in the shared temporary directory. Another local account can
therefore read saved path metadata when the surrounding hierarchy is traversable.

## Decisions

- `DefaultDir` returns an error when it cannot select or create private storage.
- A missing user configuration directory falls back to a randomly named,
  `0700` temporary directory rather than a shared predictable directory.
- Every save repairs the named favorite directory to `0700` and publishes the
  list as `0600`, including favorites created by older versions.

## Acceptance criteria

1. Saved favorite directories are `0700` and list files are `0600`, even when
   overwriting a legacy favorite.
   Verify: `go test -tags no_emoji,nodynamic ./internal/favstore -run TestSaveUsesPrivatePermissions`
2. The no-config fallback is a private, unpredictable temporary directory.
   Verify: `go test -tags no_emoji,nodynamic ./internal/favstore -run TestDefaultDirFallbackIsPrivate`
3. Production handles directory-selection failure through its existing startup
   error return.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui -run '^TestRun_RejectsUnavailableFavoriteStorageBeforeBuildingViewer$'`

## Non-goals and limits

- Existing favorites are repaired when next saved; this change does not scan and
  rewrite every favorite at startup.
- Platform ACL policy remains the operating system's responsibility; the fix
  enforces the restrictive modes represented by Go's portable file API.
- The random temporary fallback is session-specific; later launches do not
  rediscover Favorites saved there.

## Tasks

### Task 1 - Guard private persistence

Owner: T0 inline  
Files: `internal/favstore/favstore_test.go`, `internal/favstore/favstore.go`  
Test: Assert fallback, directory, and file permissions.  
Verify: `go test -tags no_emoji,nodynamic ./internal/favstore`  
Budget: 0 spawns; 1 review round; full suite at final gate.

### Task 2 - Propagate default-directory failure

Owner: T0 inline  
Files: `internal/ui/run.go`  
Depends: Task 1  
Contract: `Run` returns a `DefaultDir` error before constructing the viewer.  
Verify: `go test -tags no_emoji,nodynamic ./internal/ui -run '^TestRun_RejectsUnavailableFavoriteStorageBeforeBuildingViewer$'`
Budget: 0 spawns; 1 review round; full suite at final gate.

PR review and verification: [PR #39 record](../plans/2026-09-17-pr39-review.md).
