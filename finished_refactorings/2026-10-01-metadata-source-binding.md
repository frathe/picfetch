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
  implementations were restored and the fresh post-restoration focused checks passed.
- `make verify-build` passed after repository import formatting. Existing root
  test was extended with subtests; no shard or test-exclusion changes required.
- GoLand inspected every changed file including weak warnings. Redundant casts,
  struct padding, obsolete helper and result-flow warning fixed and reinspected.
  Six existing EXIF test duplication warnings match its exact Qodana exclusion.
  The loader has an IDE configuration error (required `nodynamic` tag absent),
  despite real tagged build/vet/tests passing. An ignored module-tag edit did not
  refresh the active IDE; it was restored. That IDE resolution gate remains
  unverified; the configured hosted Qodana scan subsequently verified this source
  separately, as recorded below. No blanket
  source suppression was introduced. IDE build tool reported limited diagnostics.
- No dependency changes; existing notices remain the shipped closure. Parent
  directory replacement races outside these source-read/confirmation guards
  remain outside the claim of filesystem atomic compare-and-swap protection.
- Ledger: zero spawns; one lead local review plus targeted inspection fixes;
  one local race round; broad native amd64 suite delegated to hosted CI.

### Qualified code gate and local test package at f36f1ca

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

### Completion-record maintenance

The final documentation review at 2e392a9 identified active-directory placement
and stale historical gate wording. This completed record is now archived under
finished_refactorings, with todo links updated and earlier snapshots labeled or
reconciled. Application code and the qualified test package were unchanged by
that documentation update. Later source follow-ups are qualified separately below.
Fresh review and CI results for subsequent documentation revisions are posted
in PR #75; original local code evidence retains its analyzed revision above.

### Subsequent review fix: changed clean inputs

Thread 4154549323 confirmed that a replacement already-clean JPEG bypassed source
verification through the no-op shortcut. Verified removal now compares inspected
identity and bytes immediately after rereading, before preparing removal or
returning a clean no-op. Unverified clean-file behavior remains unchanged.
The confirmation-path regression uses an independently qualified clean replacement:
it failed before the fix (no refusal), failed again with the guard moved behind
the shortcut, and passes after restoration. Replacement bytes remain unchanged
and no committed-source notification is emitted. Existing unchanged-symlink,
changed-byte, replaced-inode, confirmation-refresh and clean no-op cases pass.

Focused EXIF/imaging races, actual appleappstore-tagged confirmation tests and
make verify-build pass. GoLand Project per-file inspections of imaging/save.go
and exifwin/stripwork_test.go include weak warnings: no findings or timeouts.
No added test file/shard/exclusion or dependency changes. Logs:
/private/tmp/picfetch-clean-replacement-{red,negative,race}.log and
/private/tmp/picfetch-review-three-fixes-{race,store,verify}.log.
Original f36f1ca package evidence above remains source-specific; latest review
and hosted gates are recorded through PR #75 before refreshing the test package.

### Hosted CI follow-up: legacy clean-source expectation

At fa21e17, hosted non-UI race job 110345447844 found the older
TestJPEGMetadataRemovalUI case still expecting success when the inspected JPEG
was rewritten clean during confirmation. This expectation predates source binding.
The case now requires one refusal message, no committed-source notification,
unchanged clean bytes and removal of the stale action after reinspection.
Application code is unchanged. The full EXIF package race run and affected actual
appleappstore-tagged tests pass; logs are
/private/tmp/picfetch-clean-refusal-{exif-race,store}.log.
GoLand Project inspection of exifwin/exifwin_test.go included weak warnings.
Six duplicate fragments match the existing exact qodana.yaml exclusion; no new
actionable issues or timeouts. No new test files or shard assignments.
Latest-head hosted CI and fresh review still follow this test-only correction.

### Subsequent review fix: final source content check

Thread 4154911230 at reviewed fa21e17 confirmed that an in-place equal-length
rewrite with restored modification time could bypass the final metadata-only
check and be overwritten by removal derived from the older bytes. The final
pre-rename check now streams a bounded current source digest, verifies opened
identity/size/time, and checks the path identity/size/time again after hashing.
The reader is bounded to the originally admitted size plus one byte and uses
context cancellation, without retaining another full source buffer. Both verified
and compatibility removal use this guard. A later external mutation between the
final observation and rename remains possible; filesystem compare-and-swap is
not claimed. No authority or dependency changes.

TestMetadataRemovalRechecksSourceAtCommit deterministically changes the source
after staging through its private operation writer, without a mutable package seam.
It covers unchanged success, equal-length rewritten bytes with restored time,
and replaced identity with identical bytes. It was red before the fix and again
with the content comparison disabled; restored source passes. Refusal preserves
intervening bytes and removes staging files. Focused imaging races, full affected
EXIF package races and make verify-build pass. An initial Store-tagged
regression run found a test assumption about sibling staging incompatible with
the native private replacement directory. The operation writer seam now captures
its own staging write on either platform; requalification follows below. Logs: /private/tmp/picfetch-final-content-{red,negative,
race,exif-race,store,verify}.log.
GoLand inspected save.go and mutations_test.go including weak warnings; save.go
has no findings. The existing WriteResult error-path warning in mutations_test.go
is covered by the exact GoDfaErrorMayBeNotNil exclusion: a value result retains
meaningful Committed even on error. No timeouts or new actionable warnings.
At that fix snapshot, latest-head code/security review and hosted gates were
pending; their completed results are recorded below.

Requalified with the portable staging observer replaced by the private writer:
focused imaging races and actual Store-tagged imaging/EXIF tests pass, and
disabling the content comparison still reproduces the equal-length overwrite.
The per-file GoLand result is unchanged (only the assessed/excluded value-result
warning); final build/vet checks pass. The earlier Store test failure is not
claimed as a pass. No test or isolation case is skipped.

### Current source qualification at 3453f1b

All source follow-ups are committed/pushed at
3453f1bed2ed448355f7d66edd1ebeff73b9a880. The
[fresh code review](https://github.com/frathe/picfetch/pull/75#issuecomment-5930654642)
and [separate security-focused round](https://github.com/frathe/picfetch/pull/75#issuecomment-5930759320)
report no findings. The latter uses the advertised regular review trigger with
a security scope; it is not claimed as a separate named security scanner. All
20 historical threads have dispositions, including confirmed final source-content
checking in 3453f1b and the legacy confirmation expectation corrected in 63382e7.
[Full CI](https://github.com/frathe/picfetch/actions/runs/36857299096),
[CodeQL](https://github.com/frathe/picfetch/actions/runs/36857299056) and FOSSA
pass; current PR CodeQL alerts are empty.
[Qodana](https://github.com/frathe/picfetch/actions/runs/36857299114) artifact
11160590347 has this exact revision in post-suppression SARIF: eleven prior
unused-export false positives remain, with checked production callers. No new
results; ReadAndProbeSnapshot is no longer reported. There are no actionable
findings. Original per-file local inspection profiles/revisions remain stated,
including the loader IDE-tag limitation and configured exact test exclusions.

Refreshed universal ad-hoc sandbox bundle: version 1.1.11, build 483,
bin/apple-store-e2e-2026-10-01-3453f1b/PicFetch.app. The packager validates
signatures, entitlements, architectures, deployment minima, dependencies,
privacy/notices and seals; all 24 manifest payload hashes were independently
checked. Source/worktree provenance is in the adjacent manifest. The preserved
working-tree packaging metadata supplies build 483. No merge, submission or
distribution signing occurred.

The exact refreshed native app starts with -fixed-size-mode=1280x800. Its
consent panel opens at the correct new generated fixture folder with spaces.
Automatic approval review first rejected clicking Allow Folder Access because
specific app/folder permission was required. Ronin then explicitly approved the
exact generated test folder; the retry selected it and clicked the real button.
The panel closes and first.jpg loads as 1/2; Right reaches second.jpg as 2/2.
A complete quit (exit 0), relaunch with the same flag and native Open With load
first.jpg as 1/2 without another permission panel; the second quit also exits 0.
Logs: /private/tmp/picfetch-store-final-3453f1b-{run,relaunch}.log. No native
approval check remains blocked. The earlier cancellation/Return approval and
measured captures remain qualified at f36f1ca, with consent/sizing source
unchanged. Sky's refreshed app-state image has padded dimensions and is not used
as exact window-capture proof. Actual mixed-monitor movement and direct Finder
drag retain their qualification limits.

This completion change is documentation only. It carries current unchanged-code
inspection/build/package evidence at 3453f1b. Fresh reviews, CodeQL, configured
post-suppression Qodana and full CI for the documentation head are recorded in
PR #75 before the loop is declared complete; no duplicate broad local race run.
Filesystem CAS is not claimed and traversal budgets remain deferred hardening.
