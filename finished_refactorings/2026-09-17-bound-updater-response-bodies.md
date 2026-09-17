# Bound Updater Response Bodies

## Problem

The updater HTTP client limits only response-header latency. A peer that has
sent headers can therefore stall any response body forever, and GitHub release
and attestation JSON is decoded without a byte limit.

## Decisions

- Preserve large, legitimately slow archive downloads by enforcing an idle
  read timeout rather than restoring a whole-request timeout.
- Apply the idle timeout to every updater response body at the shared HTTP
  transport boundary.
- Cap each GitHub API JSON document at 16 MiB before decoding.

## Acceptance criteria

1. A response body that makes no progress for the configured interval is
   closed, while a body that keeps making progress may outlive that interval.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/autoupdate -run '^TestUpdateHTTPClient_'`
2. Release and attestation API documents larger than their explicit limit are
   rejected before JSON decoding can allocate without bound.
   Verify: `go test -tags no_emoji,nodynamic ./internal/update -run 'TestDecodeGitHubJSON'`
3. Existing updater behavior remains covered.
   Verify: `go test -tags no_emoji,nodynamic ./internal/update ./internal/ui/autoupdate`

## Non-goals and honest limit

- This does not impose a total duration on downloads that continue to deliver
  data. Such downloads are intentional and remain cancellable by their request
  context.
- The archive's existing 200 MiB byte cap is unchanged.

## Tasks

### Task 1 - Pin body bounds

Owner: T0 inline

Files: `internal/ui/autoupdate/updater_test.go`, `internal/update/github_test.go`

Test: Cover stalled and progressing bodies plus bounded JSON decoding.

Verify: focused commands in acceptance criteria 1 and 2.

### Task 2 - Implement shared bounds

Owner: T0 inline

Files: `internal/ui/autoupdate/updater.go`, `internal/update/github.go`

Depends: Task 1

Contract: updater responses use an idle body timeout; GitHub JSON decoding uses
an explicit byte limit.

Verify: acceptance criterion 3.


## Evidence

- Focused idle-timeout tests passed five consecutive runs.
- Both changed packages passed their complete test suites.
- `make fmt` and `make vet` passed.
- GoLand command-line inspections were unavailable in this environment; no
  `inspect.sh` or GoLand executable was installed under `/opt`.
- The focused idle-timeout tests passed ten runs under the race detector.
- `make verify` could not start because Docker is not installed in this
  environment.
- `make test-native` passed all changed packages but encountered the unrelated
  `TestVisualSimilarityExplorer/settings` persistence failure in `internal/ui`;
  that exact test passed immediately when rerun alone.
