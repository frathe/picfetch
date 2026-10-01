# Metadata source binding — 2026-10-01

Route: Deep (display/cache, EXIF confirmation, filesystem writes, Windows).
Owner: Pico / T0 inline; zero delegates. Hot context remains with the lead.

## Problem and decisions

PR #75 threads 4148307899 and 4148307904 identify source replacement while a
confirmation or save chooser is open. Removal can follow a different JPEG;
export can publish its metadata alongside the previously displayed pixels.

- Bind removal to the opened file identity and encoded-byte digest captured by
  inspection, and capture that value in the confirmation continuation.
- Preserve unchanged symlink sources; a retargeted link or replaced inode is a
  refusal even when its bytes happen to match. Changed bytes are also a refusal.
- Bind viewer exports to the digest decoded for the displayed pixels, including
  cache hits. A changed/unavailable source supplies no metadata; export succeeds.
- Capture the digest of our own committed Save output without rereading a path,
  so a subsequent export retains ordinary metadata behavior.
- Retain scoped URIs, transaction admission, cancellation, atomic staging and
  Store file-only replacement directories. No dependencies or new UI strings.
- Recursive scan budgeting (4148307910) remains availability backlog.

## Acceptance and tasks

1. Removal rejects entry retarget/replacement and in-place content changes after
   confirmation, leaving both originals untouched; unchanged symlink works.
   `go test -tags no_emoji,nodynamic ./internal/ui/exifwin -run 'TestMetadataRemovalBindsConfirmedSource|TestRequestStrip|TestMetadataRemoval'`
2. Export after a chooser-time source replacement writes the captured pixels
   without the replacement's metadata; unchanged source keeps GPS.
   `go test -tags no_emoji,nodynamic ./internal/ui -run TestExportAs_JPEGSourceKeepsGPSExif`
3. Cache/display captures retain their byte digest; committed-save reconciliation
   adopts only the digest of our own encoded output.
   `go test -tags no_emoji,nodynamic ./internal/ui/display ./internal/imaging -run 'TestPresentationContract|TestFileMutation|TestExport|TestSaveRotated'`
4. Focused race checks, Store-tag compilation/tests, Windows compile/vet,
   `make verify-build`, GoLand inspections of every changed code file, and latest
   hosted Codex/security/CodeQL/Qodana/CI evidence close the gate.

Task graph: regression tests -> identity/digest contracts -> EXIF/display wiring
-> focused checks/negative guard checks -> commit/push -> hosted review -> build.
All tasks T0 inline. Files: existing imaging loader/save/mutations tests,
fileaccess reader, EXIF metadata/confirmation/mutation/tests, display
presentation/capture/tests, root export/save/tests and this evidence record.
New files, if needed, receive exact Qodana test exclusions. Root UI regression is
an existing top-level test's subtest, so the shard assignment stays unchanged.
Budget: zero spawns; focused local suites; broad suite only native amd64 CI;
review rounds repeat until current-head clean. No claims of an OS atomic
compare-and-swap replacement: guards bind inspected and reread source identity;
existing filesystem transaction/staging semantics remain in force.

## Evidence

- End-to-end tests reproduced three destructive swaps (retargeted symlink,
  identical-byte replacement inode, and changed bytes) and chooser-time GPS
  publication before the guards. Untouched symlink behavior was green.
- Full affected EXIF/display/imaging/fileaccess/uitest race suites and focused
  root Export/Save races passed. Store-tagged focused suites passed; Windows
  EXIF compilation and affected-package vet passed (runtime qualification CI).
- Removing the identity guard again caused all three mutation regressions to
  fail; removing digest verification caused export GPS disclosure again. Both
  implementations were restored. Fresh post-restoration focused checks follow.
- `make verify-build` passed after repository import formatting. Existing root
  test was extended with subtests; no shard or test-exclusion changes required.
- GoLand inspected every changed file including weak warnings. Redundant casts,
  struct padding, obsolete helper and result-flow warning fixed and reinspected.
  Six existing EXIF test duplication warnings match its exact Qodana exclusion.
  The loader has an IDE configuration error (required `nodynamic` tag absent),
  despite real tagged build/vet/tests passing. An ignored module-tag edit did not
  refresh the active IDE; it was restored. That IDE resolution gate remains
  unverified, requiring current configured hosted Qodana evidence. No blanket
  source suppression was introduced. IDE build tool reported limited diagnostics.
- No dependency changes; existing notices remain the shipped closure. Parent
  directory replacement races outside these source-read/confirmation guards
  remain outside the claim of filesystem atomic compare-and-swap protection.
- Ledger: zero spawns; one lead local review plus targeted inspection fixes;
  one local race round; broad native amd64 suite delegated to hosted CI.

### Final code gate and local test package

Verified code revision: f36f1cade1504a3f126c564be91f7554e098b710.
[Fresh code review](https://github.com/frathe/picfetch/pull/75#issuecomment-5927983355)
and the [separate security-focused round](https://github.com/frathe/picfetch/pull/75#issuecomment-5928247620)
report no findings after all eight thread dispositions. Recursive scan work
budget is explicitly deferred availability hardening, not an implemented fix.

[Full CI](https://github.com/frathe/picfetch/actions/runs/36838100555),
[CodeQL](https://github.com/frathe/picfetch/actions/runs/36838100685) and FOSSA pass.
Open PR CodeQL alerts are empty. [Qodana](https://github.com/frathe/picfetch/actions/runs/36838100783)
artifact 11150212850 has this exact revision and 12 post-suppression SARIF
results, all unused-export false positives: the prior ten plus
StripJPEGMetadataVerified (EXIF constructor) and ReadAndProbeSnapshot (metadata
worker). Production callers were checked; no actionable findings remain.
Original local inspection scope/profile and its loader IDE-tag limitation remain
as recorded; tagged compiler checks and configured Qodana verify that source.

Universal ad-hoc sandbox package: version 1.1.11, build 483,
bin/apple-store-e2e-2026-10-01-f36f1ca/PicFetch.app. Native signatures,
entitlements, dependency closure, deployment minima, resources and seals pass;
all 24 manifest payload hashes were independently checked. No submission,
distribution signing or merge occurred. The manifest preserves the original
source/worktree provenance; later evidence-only commits carry forward unchanged
code results with this revision stated.
All source-binding guards and regression evidence are unchanged at the compiled
code gate. The availability backlog was recorded and the bot thread resolved as
a deferral. No OS atomic compare-and-swap guarantee is added by these guards.
