# Bound Save and Export JPEG rereads

Route: Standard, lead inline, zero delegates. The repository agreement keeps
hot-context investigation, fixes and every review with the lead; the security
skill's investigation and candidate review are separate inline passes.

## Contract

After display admission, an external writer can grow or replace a regular JPEG.
Save Changes rereads it through `readFileContext`; JPEG Export acquires source
authority and rereads through `jpegFileBytesContext`. Both currently allocate
without a byte limit. Every complete reread must capture `MaxEncodedBytes`,
reject an oversized stat before allocation, and enforce the bound while reading
even when the file grows after stat. Cancellation remains observable.

Save rejects oversized metadata input without replacing the source. Export
keeps its existing unreadable-source behavior: write displayed pixels without
source metadata. Exact-limit JPEGs, mislabeled JPEGs, non-JPEG fast rejection,
permissions, atomic writes and URI authority remain supported. Metadata-removal
limits and the separate source-identity findings are outside this patch.

## Tasks and acceptance commands

1. Extend existing `mutations_test.go` with replacement/growth-after-display,
   exact-limit metadata preservation and bounded streaming regressions.
   Red/green: `go test -tags no_emoji,nodynamic ./internal/imaging -run
   'TestSecondaryJPEG|TestReadMutationSource' -count=1`.
2. Add a shared private bounded reader in `mutations.go`; apply it to Save's
   full reread and Export's magic-confirmed reread in `save.go`. Inspect all
   direct callers and challenge alternate filenames and growth during reads.
3. Run imaging race tests, focused root Save/Export regressions, Apple Store
   tagged coverage, GoLand inspections including weak warnings and
   `make verify-build`. Commit/push this fix, complete fresh GitHub code/security
   reviews, full CI, CodeQL and post-suppression Qodana assessment before the
   next issue. No broad local race suite; the Docker daemon is ARM64.

Graph: regression -> bounded reads -> qualification -> commit/review loop.
Budget: zero spawns, one inline candidate review, focused tests and one local
build/static gate. No dependency, package, UI string or test-file addition.

## Evidence

- Red: all six growth/replacement cases reproduced unbounded rereads after a
  successful display load. Save committed; normal and mislabeled Export sources
  returned bytes past the configured ceiling. Exact-limit controls passed.
- Green: focused Save/Export, magic detection and cancellation tests pass.
  Shared stream coverage consumes exactly limit+1 bytes from a larger producer,
  returns no partial bytes on overflow and reads nothing after cancellation.
  Removing the stream limiter (while retaining stat admission) made this guard
  fail with 4096 bytes consumed instead of 65. Source restored afterward.
- `go test -race -tags no_emoji,nodynamic ./internal/imaging`: PASS (33.404s).
  Focused root Save/Export race regressions: PASS (34.226s). Apple Store-tagged
  bounded read and live-authority tests using the actual `appleappstore` tag:
  PASS (0.492s). The initial `applestore` spelling selected ordinary files;
  that run is not counted as Store verification.
- `make verify-build`: PASS, including full host vet/build, format, generated
  inputs, notices, TUF and exact test exclusion checks. Initial sandboxed race
  and build attempts could not access Go cache entries; authorized reruns passed.
- GoLand inspected all three changed Go files with weak warnings included.
  Production files are clear. The existing missing-source test at line 152 has
  an invalid nullable-result warning: `WriteResult` is a nonpointer value whose
  zero `Committed` field intentionally remains observable alongside an error.
  No suppression or unrelated test change is needed. Final helper reinspection
  after the negative probe is clear. The restored final helper also passed
  its focused race rerun (1.314s) and `make verify-build` passed again.
- Lead candidate review: both complete rereads share the helper; no extension
  alias bypasses Export's content check. Both outcomes preserve existing error
  behavior and acquired URI authority; non-JPEG sources still stop after two
  bytes. Stat is only an early optimization, never the sole memory guard.
  Metadata-removal's tighter independent budget is unchanged. No new test file,
  UI test or package move requires exclusions, shard or architecture changes.

Hosted code/security reviews, full native Linux/amd64 CI, CodeQL and a fresh
post-suppression Qodana assessment are pending for the pushed fix. Final results
will be retained on PR 75 before proceeding to the recursive-scan finding.
On resumption, the lead rechecked the complete candidate and direct callers;
the focused security race regressions passed again (1.644s), and
`git diff --check HEAD` was clear. Ronin explicitly authorized commit/push and
a separate completed GitHub review loop after each security fix.
Ledger: zero spawns, one inline candidate review, one negative streaming probe,
focused regression/race checks and local build/static gate.
