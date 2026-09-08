# Qodana findings from revision 2698317

Route: Standard. Resolve the 13 supplied findings with behavior-preserving
production cleanup and inspection/file-specific ignores for tests and confirmed
non-issues. The report is the spec; the package breadth is mechanical and does
not require a cross-package design. No new interfaces, UI changes or test files.

## Decisions and acceptance

- Rename the two Zenity executable variables, make suite receivers consistently
  pointers, and remove only RAW slice lower-bound conversions. Keep widened
  upper-bound arithmetic to preserve overflow protection.
  Verify: IDE inspections on those three production files and
  `go test ./internal/filepicker ./internal/imaging ./scripts/nativeguards`.
- Ignore the reported naming inspections only in the two named UI test files,
  and error-result inspection only in `internal/imaging/mutations_test.go`.
  `WriteResult` is a value whose commit state is intentionally tested on errors.
  Verify: inspect `qodana.yaml` and `make check-qodana-test-exclusions`.
- Ignore the clipboard resource inspection only in `internal/clipboard/clipboard.go`:
  `writeTempPNGFile` closes on every path; consumers remove the returned filename.
  Verify: `go test ./internal/clipboard -run '^TestWriteTempPNG'`.
- Final gate: `make verify`; report any environmental limitation explicitly.
  Qodana CI itself may remain unverified if its linter is unavailable locally.

## Tasks

1. Clipboard ownership recon (read-only scout; no files changed).
   Contract: identify close/remove ownership with source locations.
   Verify: existing clipboard test command above, run by lead.
   Budget: 1 spawn, 1 lead review, no full suite.
2. Apply production cleanups and exact inspection exclusions (lead).
   Files: `internal/filepicker/filepicker.go`, `internal/imaging/raw.go`,
   `scripts/nativeguards/main.go`, `qodana.yaml`, `todos.md`.
   Verify: acceptance commands above; no new tests for mechanical changes.
   Budget: 0 spawns, 1 review, full suite at final gate only.
3. Final verification and evidence (lead), depends on tasks 1 and 2.

Tasks 1 and production recon are independent. Delegation gate: a short ownership
question (G1), existing failure-path tests (G2), read-only/disjoint work (G3),
helper/caller/test tracing isolated from other findings (G4), and fresh clipboard
context at dispatch (G5). A semantic ownership trace needs comprehension (S);
the task provides no implementation (W). Lead retains all fixes and review.

## Evidence and cost

Requested cleanup complete. Actual: 1 scout spawn, 1 lead review, 1 full gate,
plus one failed-shard retry justified by the gate failure.

- `go test ./internal/filepicker ./internal/imaging ./internal/clipboard
  ./scripts/nativeguards`: all four packages passed. This includes the existing
  clipboard failure-path tests and the mutation-result assertions.
- GoLand inspections: no warnings/errors in the three modified production files.
- `git diff --check` and `make check-qodana-test-exclusions`: passed.
- `make verify`: formatting, TUF root, exclusion check, vet, build and 667-test
  shard inventory passed. Non-UI, ui-1 and ui-2 race partitions passed; ui-3
  reported a package-only failure without individual test failures in the compact
  log. Full output: `/tmp/picfetch-qodana-verify-20260907.log`.
- Prepared Linux/amd64 Docker retry:
  `make test-race-ui-direct TEST_SHARD=ui-3 TEST_CAPTURE=/capture/ui-3.json`
  passed (305.538s). Host-mounted raw events and compact output:
  `/tmp/picfetch-qodana-check/ui-3.json` and `ui-3.log`.
- The first package failure remains unexplained and is tracked in `todos.md`.
  The retry establishes a passing isolated shard, not a passing concurrent gate.
  No unrelated test or runtime changes were made. Qodana CI was not rerun;
  inspection/file exclusions were added to its existing configuration.

Initial sandboxed Go testing could not write the shared build cache; the retry
and Docker verification ran with approved cache/Docker access.
