# Mac App Store preparation

Status: active; submission readiness is not established.
Route: Deep (macOS distribution and cross-package permission lifetimes).

## Deliverable

Prepare a separate Mac App Store distribution while retaining PicFetch's
features and preserving ordinary and Microsoft Store distribution behavior.
Ronin authorized committing the preparation patch after reviewing the initial
handoff. No upload, release, or change to signing credentials is authorized.

## Observed gaps

- `package-mac` produces the ordinary, unsandboxed distribution.
- `internal/distribution` recognizes only ordinary and Microsoft Store builds.
- Session and Favorite file lists retain paths, without security-scoped grants.
- Darwin HEIC and similarity launch through `/usr/bin/sandbox-exec`.
- Ordinary similarity setup downloads native ONNX libraries.
- The runtime verifier compares complete upstream library bytes; signing a
  Mach-O library changes those bytes, so this verifier cannot simply be reused
  after Mac App Store signing.
- The development host selects Command Line Tools; full Xcode 27 is available through a per-command
  `DEVELOPER_DIR`, and
  `security find-identity -v -p codesigning` reports no valid signing identities.

Apple requirements and sources are recorded in
`docs/apple-app-store-requirements-2026-09-29.md`.

## Decisions

| Decision | Reason |
| --- | --- |
| Separate `appleappstore` build channel | A Store package must never activate GitHub binary updates or download native code. |
| Preserve current application ID | Changing it would change preferences/session identity; confirm it can be registered with the developer account. |
| Preserve full feature set (Ronin confirmed) | Do not silently remove Explorer, deletion, or wallpaper support. |
| No broad filesystem or temporary exception entitlements | User-selected grants must carry actual authority; saved paths do not. |
| No weaker offline guarantee | Replacing `sandbox-exec` with an ordinary network-enabled child is not sufficient. |

## Acceptance criteria

AC1: Ordinary, Microsoft Store, and Apple Store builds select their intended
update policy; Apple builds refuse native runtime download fallback.
`go test -tags no_emoji,nodynamic ./internal/distribution ./internal/launch`
`go test -tags no_emoji,nodynamic,microsoftstore ./internal/distribution ./internal/similarity -run 'TestStore|TestRuntimeDownloadSelection'`
`go test -tags no_emoji,nodynamic,appleappstore ./internal/distribution ./internal/similarity -run 'TestAppleStore|TestStore|TestRuntimeDownloadSelection'`

AC2: Store explanations name the correct store, with locale parity.
`go test -tags no_emoji,nodynamic,appleappstore ./internal/ui/settingswin ./internal/ui/explorer ./internal/ui -run 'TestUpdatesTab|TestFeatureSetupDownloadRetryCancel|TestUpdateCheck_StoreManaged|TestLaunchPolicyIntegration'`
`go test -tags no_emoji,nodynamic . -run 'TestTranslations|TestLaunchStartupContract'`

AC3: Grants survive relaunch and are retained through all workers using them;
cancel/close/shutdown do not release grants while producers still use them.
Verification: native signed sandbox qualification on both architectures,
including picker, Dock/Open With, dropped directory, session restoration,
Favorites, renamed/missing files, exports, committed writes, and shutdown.
The test runner and commands must be pinned before implementation starts.

AC4: Public sandboxed workers preserve actual TCP/UDP denial and tracked exit.
Verification: signed native helper qualification, including real HEIC and ONNX
analysis, cancelled runs, worker shutdown, and parent crash. The native fixture command is `DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer make apple-worker-test`.
Real image/model execution, IPC grant transport and Intel qualification remain open.

AC5: Native dependencies are pinned before signing, bundled with notices,
validated by signature after signing, and cannot be replaced by model-cache
overrides. Verification: final signed bundle validation, tampered-library
rejection, real inference, and pre-sign archive checksum rejection.

AC6: Submission package and metadata pass Apple's validation; privacy labels,
export compliance, licensing, screenshots, support and policy URLs are accurate.
Verification: final bundle inspection, `codesign --verify --strict`,
`pkgutil --check-signature`, Apple's current upload validation and TestFlight.
Credentials and account details are needed for these external checks.

## Task graph and routing

`T1 -> T2 -> T3 -> T4 -> T5; pre-sign staging in T4a is independent of T2/T3`; T6 research informs every task.
All implementation, architecture, review and fixes belong to T0 inline.
One research delegation covers public Apple documentation only; no platform
implementation or review is delegated. No model overrides.

### T1 — Distribution policy and accurate Store messages
Owner: T0 inline.
Files: internal/distribution, similarity/assets_install.go and assets_test.go,
UI update/settings/setup code and existing tests, translations, architecture.
Contract: `distribution.StoreManaged` remains immutable; a new
`distribution.AppleAppStore` identifies the Apple channel. Existing launch
permission remains the authority for update effects.
Test: captured policy disables updates; missing bundled runtime fails before
any HTTP request; UI names the correct store.
Verify: AC1/AC2 commands.
Budget: 0 spawns; 2 review rounds; full suite at final gate only.

### T2 — Persistent user-selected filesystem access
Owner: T0 inline.
Files: new permission owner package; filepicker, openwith, UI input/startup/
shutdown, session and Favorite authority integration.
Depends: T1.
Contract: design a bounded owner for grants and bookmarks before coding; keep
native URL authority through admission and retain grants through worker exit.
Test: denied sibling access never silently broadens authority; stale or absent
grants offer reselection; complete Favorite/session sources retain valid grants.
Verify: AC3 qualification and focused lifetime tests.
Budget: 0 spawns; 2 review rounds; no extra complete suite.

### T3 — Public sandbox worker architecture
Owner: T0 inline.
Files: main private dispatch, HEIC and similarity Darwin launchers/protocols,
packaged helper entitlements and native qualification tooling.
Depends: T2 for source authority; the process-boundary fixture is independent.
Contract: a signed worker with no networking, documented grant transfer, and
existing stop/done and offline-verification contracts.
Test: worker cannot network, accepts only current source grants, and exits after
cancellation/shutdown. Do not introduce temporary sandbox exceptions.
Verify: AC4 native qualification.
Budget: 0 spawns; 2 review rounds; no extra complete suite.

### T4 — Bundle runtime and build/sign package
Owner: T0 inline.
Files: similarity assets/package verification, Apple packaging scripts,
Makefile, entitlements and metadata templates, third-party notices.
Depends: T3.
Contract: Mac runtime loaded exclusively from the signed application bundle;
release archive digest verified before signing, signature after signing.
Test: tampering, wrong architecture, bad archive and missing notices fail closed.
Verify: AC5 plus signed package inspection.
Budget: 0 spawns; 2 review rounds; no extra complete suite.

### T4a — Pre-sign native staging and prerequisite tooling
Owner: T0 inline.
Files: similarity/assets_package.go and existing assets_test.go;
Makefile, scripts/apple-store-preflight.sh, packaging/apple-app-store/README.md.
Depends: no runtime permission contract; can precede T2/T3.
Contract: `StageMacRuntime(ctx, arch, archivePath, root)` verifies pinned archives
and stages only library/notices before signing; prerequisites are read-only.
Test: invalid/unsupported/cancelled archives cannot change package output;
real pinned arm64 archive yields verified runtime plus upstream notices.
Verify: targeted StageMacRuntime/StageWindowsRuntime tests with the explicit
existing arm64 archive; `bash -n scripts/apple-store-preflight.sh`;
`make apple-store-preflight` reports actual missing inputs.
Budget: 0 spawns; 2 review rounds; no additional complete suite.

### T5 — Submission evidence and final gate
Owner: T0 inline.
Files: packaging/apple-app-store/listing.md, privacy and submission records,
todos.md, plan evidence.
Depends: T4.
Contract: every claim binds to an actual final artifact; report unverified gates.
Test: metadata mirrors shipped behavior and has no placeholders at submission.
Verify: AC6, GoLand inspections, `make verify`, latest CI where available.
Budget: 0 spawns; 1 review round; complete suite once.

### T6 — Apple primary-source research
Owner: delegated research agent.
Files: docs/apple-app-store-requirements-2026-09-29.md only.
Contract: Apple citations, scope distinctions and explicit uncertainty.
Verify: T0 checks links/claims against fetched Apple documentation.
Budget: 1 spawn; 1 review round; no suite.
Delegation gate: G1 yes (bounded public research), G2 no (judgment required),
G3 yes, G4 yes, G5 yes. This delegation is specifically required by the invoked
research skill; that instruction is the reason for the G2 exception. The
agent supplies evidence only; T0 owns assessment and review.

## Dependencies and licensing

No dependency upgrades. Retain the existing six-target notice checks, AVIF WASM
source/license manifest, Fyne/GLFW licensing and SigLIP model notices. Apple
packages must include architecture-pinned ONNX 1.23.2 (Intel) / 1.29.0 (arm64),
their exact upstream LICENSE, ThirdPartyNotices.txt and available Privacy.md.
Record the complete shipped closure and notice delivery before release readiness.

## Honest limit

A build tag is preparation, not evidence of sandbox compliance. Until T2–T5
are complete, Apple builds are experimental and must not be submitted. Ad-hoc
signed native fixtures qualify selected boundaries on arm64; complete production
sandbox qualification, Apple Distribution signing, validation and App Review
remain outstanding.

## Evidence and cost ledger

Initial reconnaissance: branch `feature/apple-appstore-preparations`, clean
working tree; host arm64; selected developer directory
`/Library/Developer/CommandLineTools`; zero valid code-signing identities.

| Task | Spawns budget/actual | Review rounds | Complete suite | Status |
| --- | --- | --- | --- | --- |
| T1 | 0/0 | 2 | no | verified foundation |
| T2–T5 | 0/0 | pending | unverified | open |
| T4a | 0/0 | 2 | no | verified pre-sign staging |
| T6 | 1/1 | 1 | no | research recorded; T0 checked key requirements |

### Preparation evidence (2026-09-29)

Analyzed revision: HEAD `09dea07` plus the uncommitted preparation patch.
Changed code scope digest (sorted path/NUL/content/NUL): SHA-256 `379807100409ec6246a78142d7f8802d55c1ee1528c72e9bd01b07a785b4a061` (18 files, Go source/tests, shell and Makefile).

- `make verify-build` passed: formatting, TUF freshness, exact Qodana exclusions,
  generated assets, dependency notice checks, vet and ordinary build.
- Complete affected-package regressions passed for distribution, launch,
  similarity, Settings, Explorer and MSIX staging with ordinary build tags.
- Focused Apple-channel tests passed in main, distribution, similarity,
  Settings, Explorer and root UI: startup policy, translation parity, update
  refusal, Store runtime-download refusal and correct Store explanations.
- Focused Microsoft Store policy, download-selection and setup/Settings tests
  passed. No native Windows Store artifact was newly qualified.
- Real pre-sign arm64 staging passed against the existing pinned
  `onnxruntime-osx-arm64-1.29.0.tgz`; library checksum and all upstream
  notices including Privacy.md were verified. Intel pinned-archive staging
  was not exercised against a real archive.
- Guards were deliberately violated: false Apple channel, allowed GitHub
  updates and skipped runtime archive digest. All three tests failed for the
  intended behavioral reason; original sources were restored and tests passed.
- GoLand `get_file_problems(errorsOnly=false)` inspected every changed Go file
  and the shell preflight, including weak warnings; no timeouts were reported.
  Four duplicate-fragment weak warnings in `internal/ui/autoupdate_test.go`
  describe existing unchanged test blocks. They are intentional test
  duplication covered by that file's existing exact Qodana exclusion; no
  refactoring or new suppression was added. No actionable finding remained.
  New Apple-tagged test and runtime-staging files were inspected after final edits.
- GoLand build reported success but limited diagnostic collection. The actual
  `go build` in `make verify-build` supplies command-level build evidence.
- `bash -n scripts/apple-store-preflight.sh` passed. Native preflight found Xcode
  27.0 (27A266a) through its per-command developer directory and reports missing
  Team ID/profile. An independent keychain check found zero valid code-signing
  identities. The profile itself is not validated by this prerequisite script.
- `make verify` stopped at the required platform guard: the Docker daemon is
  Linux/aarch64. Complete Linux/amd64 race verification remains unverified;
  no isolation test was skipped and no sandbox filter was relaxed.
- Fresh Qodana CI/SARIF, CodeQL, signed sandbox qualification, signed bundle/
  installer validation and Apple processing/review have not run for this patch.
  No commit, push, submission or release was performed.

Remaining implementation is T2/T3 and the signed portion of T4/T5. Signing
identity/profile/account inputs have been requested from Ronin. The foundation
is reviewable; it does not establish a full-feature Store app's readiness.

## Continuation evidence — XPC worker boundary

Base revision: `3883bf5`; work below is the next implementation slice.
T3 process boundary is implemented; full AC3/AC4/AC5 are still open.

- Full Xcode compiler now works after Ronin accepted its license locally.
  Preflight checks `xcrun clang --version` as well as `xcodebuild -version`.
- A separately sandboxed XPC service owns the image-worker process group.
  The inherited image helper must be inside the service bundle: the first
  fixture failed `posix_spawn` with EPERM from the parent app's MacOS directory;
  moving it into service Contents/MacOS passed without extra entitlements.
- `make apple-worker-test` on macOS arm64 passes both modes, pipe transfer,
  TCP/UDP EPERM, invalid mode refusal, SIGTERM cancellation, SIGKILL broker
  crash, and normal leader exit retiring descendants. Leader reaping is checked
  after broker completion; descendant pipe EOF is observed with bounded waits.
- Negative verification added network.client to the temporary service and saw
  the guard fail on TCP EINPROGRESS/UDP success; restored entitlements pass.
  The first negative run timed out on blocking TCP, so the fixture now uses
  nonblocking sockets to discriminate policy denial without external responses.
- Apple-tagged HEIC/similarity launchers select the fixed bundled broker.
  Native completion is synchronized; descriptor duplication prevents dup2
  collisions; unreaped leaders prevent PID reuse during group retirement.
  Nested HEIC retains the XPC-owned group. Existing process-group fixtures
  explicitly own their group and retain descendant-retirement coverage.
- Protocol tests use an instance-owned command factory so ordinary Go test
  executables need no signed app bundle. No isolation test is skipped.
- `go test -tags no_emoji,nodynamic,appleappstore ./internal/macworker
  ./internal/heic ./internal/similarity` passes. The same ordinary-tag command
  passes. Apple-tagged focused `go vet` passes.
- GoLand get_file_problems, errorsOnly=false, reviewed all 20 changed code files
  in this slice. One shell warning about empty CDPATH assignment was corrected
  and re-inspected clean. Objective-C/C also compile with -Wall -Wextra -Werror;
  GoLand native-language inspection depth is limited by installed language support.
  IDE build reports success with limited diagnostics, so CLI tests/vet remain
  the build evidence. Qodana CI/profile scan is not established by these checks.
- New command_test.go is exactly excluded from Qodana duplicate inspection.
  No top-level root UI test or new dependency was added.

Honest limit: the fixture establishes native containment and lifecycle, not a
complete sandboxed PicFetch app. Security-scoped source/cache authority, signed
runtime validation, production packaging, Intel and final Store qualification
remain required. No upload or release has been performed.

T3 budget so far: 0 spawns; 2 inline review rounds; no complete suite rerun.

### Next authority design and native feasibility evidence

Base revision: `2ec93e7`. T2/T3 source authority remains open.

Apple's current [file-access guidance](https://developer.apple.com/documentation/security/accessing-files-from-the-macos-app-sandbox)
distinguishes app-scoped bookmarks stored for relaunch from bookmarks created
with options 0 for interprocess access. Resolve temporary bookmarks in the
process doing the actual file I/O and balance its implicit access with Stop.
Persistent bookmarks cannot simply be sent to a differently signed XPC helper.

The native qualification now includes the production layout's inherited broker.
A fixture app creates one file and an unrelated sibling in its own container,
then sends an options-0 bookmark through the existing input pipe. The separately
sandboxed image worker is denied the file before resolving the bookmark,
reads its exact content after resolution, and remains denied the sibling.
Both TCP/UDP checks and retirement checks still pass. Native sources compile
with -Wall -Wextra -Werror; strict ad-hoc bundle signature verification passes.
GoLand inspections of driver.m, fixture.m, check.py and run.sh return no findings.
Negative verification replaced the transferred bookmark with invalid bytes;
only the source-access guard failed (granted=0); the fixture was restored.

Planned contract before source implementation:

- An immutable source URI carries persistent bookmark data and the relative
  child identity when its authority is a selected directory. Paths never
  manufacture authority. Folder scanning propagates only that captured grant.
- Each file I/O operation resolves/acquires its source scope, retains it through
  actual read/write/native completion, and releases it exactly once. The URI
  carries no indefinitely retained native scope, and features share no global
  mutable permission registry. Capture/resolution must happen on tracked workers.
- Session and Favorite records retain grants aligned with the complete recorded
  membership and occurrences. Missing, stale or inaccessible authority needs
  reselection; it must not be interpreted as an empty successful collection.
- Analysis holds its parent source/cache leases for the whole existing Analyze/
  Search call. Add bounded temporary bookmark bytes to the existing request
  protocol; the image worker resolves and holds them through all worker exit
  and nested decoder retirement. No new broker/XPC side channel is needed.
- File-only grants do not imply parent-directory rights. Sibling browsing and
  atomic source replacement need verified folder authority or a deliberate
  native authorization flow; do not silently broaden the selected scope.

Focused lifetime tests must cover acquisition failure rollback, cancelled reads,
exact-once release, directory-child inheritance/traversal rejection, restored
membership/occurrences, and delayed native completion. Their package-specific
commands and source boundary map will be pinned before those edits.

A scratch native picker probe is open for user-assisted qualification because
both computer-use accessibility requests timed out. It uses separate identifier
io.github.frathe.picfetch.grantqualification and only the generated
.scratch/apple-store-grant-probe/selected folder. User selection/relaunch/moved
folder evidence is pending. No production bookmark readiness is inferred from
the successful temporary interprocess grant fixture.

### Persistent grant probe — actual user-selected folder

Ronin selected only the generated fixture folder in the temporary native
NSOpenPanel. Its separate ad-hoc signed app has app-sandbox,
files.user-selected.read-write and files.bookmarks.app-scope entitlements.

- Capture returned a nonempty security-scoped bookmark with no native error.
- Fresh-process restore: original-path read failed with Cocoa error 257,
  startAccessingSecurityScopedResource returned true, content read succeeded,
  and stale=false. The matching stop call ran before process exit.
- Renamed generated folder: original-path read failed with error 260; the
  bookmark resolved the moved directory, returned stale=true, read succeeded,
  and regenerating persistent data succeeded. The fixture folder was restored
  to its original path afterward.
- Evidence: .scratch/apple-store-grant-probe/restored.json and moved.json;
  native source and signed app are alongside them. No user photo was selected.
- Computer-use launched the probe but AX requests timed out. The completed
  native output, rather than inaccessible AX state, establishes these results.
  Ronin performed the initial selection; subsequent launches need no picker.
- Full Apple-tagged application build to a temporary executable succeeds.
  This is compilation evidence only; no production sandbox bundle is ready.

Source boundary map before T2 edits:

| Boundary | Existing entry points | Required authority lifetime |
| --- | --- | --- |
| Selection/open | filepicker Darwin panel serializer; openwith native URL bridge; UI open/drop admission | Capture selected URL bookmark before flattening, admit immutable URI metadata; cancelled requests publish no new sources. |
| Decode/EXIF | imaging.readRawBytes | Acquire before storage.Reader; release after ReadCloser closes, including cancelled/failed reads. |
| Scan | filescan.gather and SiblingsWithAdmission | Acquire during each directory listing/stat; attach captured directory grant to discovered children; file grants cannot authorize parent scans. |
| Save/export/metadata | imaging.SaveRotatedContext, ExportContext, StripJPEGMetadataContext | Hold source/destination grant across transaction, temp replacement and committed effects; source metadata reads need their own grant. |
| Version/reconciliation | favthumbs.EntryName; UI filework stats and alias comparisons | Acquire on existing tracked workers; keep captured source/version semantics. |
| Delete | deletion.performDelete -> imaging.WithFileMutation -> trash.Move | Hold captured target grant through submitted native completion even if view closes. |
| Persistence | session state; favstore.Definition.Files and saving list encoding | Preserve complete ordered grant metadata/occurrences; no filesystem widening from path lists. |
| OS handoff | reveal, clipboard file copy and wallpaper native URLs | Hold scope through actual native call; verify recipient access using public URL mechanisms. |
| Analysis | similarity.Client Analyze/Search request and WorkerMain | Parent leases through complete worker wait; temporary bookmark array bounded by existing protocol limits; worker scopes through complete producer/nested decoder retirement. |

T2 remains open. These probes establish the native bookmark behavior, not
production grant propagation, stale collection reconciliation or complete
sandboxed feature operation.

### Recovery continuation — source permission persistence

Resume the uncommitted fileaccess/picker/scan/read patch on `a293137`.
Owner: T0 inline; no delegation. Seams: fileaccess Acquire/Reader/Child/Parent,
filescan Images/Replay/Siblings, session Save/Load, and favstore Save/Open/Files.
Preserve occurrence order and distinct grants for repeated paths; ordinary
path-only records remain readable. Favorite publication must return the same
authority that reopening its complete saved membership returns, without native
scope acquisition during encoding or decoding. Malformed authority fails closed.
Verify: focused ordinary and Apple-tagged tests for fileaccess, filepicker,
filescan, session and favstore; then affected regressions, vet and IDE inspections.
Budget: 0 spawns, 2 inline review rounds, complete verification once at handoff.
This slice does not establish signed full-application qualification.

### Recovery checkpoint — 2026-09-29

Base: `a293137`, plus the recovered and continued working-tree patch; no commit
or push in this continuation. Code scope: 21 Go files, sorted path/NUL/content/NUL
SHA-256 `bcfd4b0b4531e8301d11b5c2b7ad3c0f6a8d7db881cf2091846bc78c3bfac05a`.

Concurrent detached Codex sessions were still modifying this checkout after the
interrupted run. Ronin explicitly requested stopping all other sessions. They
were terminated, remaining detached processes were force-stopped, and a process
check confirmed only the current session remained. Late manifest/session edits
were reconciled before the final tests. Some shell tool calls fail to report
completion even after command exit; explicit command exit markers and complete
package results are the verification evidence, not those tool session handles.

Implemented checkpoint:

- Native open-panel URL ownership survives UI delivery to the tracked chooser
  worker, where Apple-channel persistent bookmarks are captured. Canonical image
  reads hold one acquired scope through ReadCloser.Close, with idempotent release.
- Folder scans propagate the captured directory grant to descendants; file-only
  selections do not acquire parent/sibling authority. Replay preserves occurrences.
- `fileaccess.Manifest` stores each shared scope once and retains every ordered
  source occurrence. Session `access` and Favorite `$access` metadata are checked
  against the complete saved list before restoration. Ordinary path-only writes
  retain their existing format; old path-only records remain readable. Older
  PicFetch versions do not understand the new scoped Favorite metadata.
- Committed Favorite save results carry the same source authority as reopening
  the published membership. No native scope is acquired while serializing or
  restoring metadata. A 1,024-occurrence collection sharing a 128 KiB bookmark
  saves and reopens within the existing 64 MiB Favorite definition limit.

Verification:

- Session and Favorite authority regressions first failed because bookmarks were
  lost; both pass with the implementation. Malformed, mismatched and incomplete
  manifests fail closed; duplicate numeric positions and legacy size limits
  retain their existing regression coverage.
- Ordinary and Apple-tagged fileaccess, filepicker, filescan, session and favstore
  suites pass. Ordinary imaging, favthumbs, UI Favorites, Location Map, similarity
  and macworker regressions pass.
- Apple-tagged root UI collection/Favorite and Store-policy integration passes
  (`TestCollectionReplay|TestCollection|TestFavorite|TestLaunchPolicyIntegration|TestUpdateCheck_StoreManaged`).
- Temporary Go overlays deliberately discarded bookmarks, omitted cancellation
  release, discarded child grants and widened file-only sibling access. Guards
  failed on each intended behavior; real sources were unchanged and restored
  focused suites pass. The first scan overlay had an unused variable compile
  error; it was corrected before the reported behavioral failures.
- `make verify-build` passes (format, TUF/assets/notices/exclusions, vet, build).
  Focused Apple-tagged vet and Windows/amd64 no-cgo vet pass for fileaccess,
  session, favstore, filescan and filepicker. The two new scan structural tests
  use the non-Store resolver explicitly; signed native grant behavior remains
  a separate qualification gate.
- GoLand `get_file_problems(errorsOnly=false)` inspected all 21 changed Go files.
  Import ordering and a previously unhandled test Close error were corrected.
  The build-channel constant warning has a narrow documented suppression.
  Reinspection leaves intentional duplicate test fragments in session,
  favstore/saving and filepicker; existing exact Qodana exclusions cover them.
  The current IDE profile does not select `nodynamic`, so imaging/loader.go has
  an AVIF build-constraint diagnostic. CLI vet/build with required tags passes;
  that IDE profile's imaging result remains unverified, not suppressed. This is
  IDE fallback evidence, not a fresh Qodana SARIF/profile result.
- `make verify` cannot pass its prerequisite: the selected Docker daemon reports
  Linux/aarch64. Complete native Linux/amd64 race verification remains unverified;
  no isolation test or worker restriction was bypassed.

Next: native save/Open With/drop capture, missing/stale-grant reselection and
renewal, mutation/version/OS-handoff scope lifetimes, worker grant transport,
signed runtime validation/packaging, and full Intel/Apple Silicon qualification.
Session cache corruption retains its existing nil-result behavior; presenting a
reselection/error surface for unusable authority is still part of T2. The raw
record and manifest tests establish serialization/lifetimes, not a sandboxed
production application's readiness. Signing inputs and Apple submission gates
remain outstanding. No release, upload, credential change or dependency upgrade.

Recovery budget: 0 spawns; 3 inline review rounds (one additional reconciliation
round for concurrent edits); one attempted complete gate, blocked at the daemon
platform guard. Focused tests and build-only verification completed separately.

### Next slice — owned native opening requests

Base: `03a2eaf`. Owner: T0 inline; Deep route, no delegation.
Files: fileaccess selected-input ownership; openwith Darwin bridge/queue;
root collection scan, composition and harness; focused tests, manifests/docs.
Contract: `fileaccess.NewSelection` carries one native selected URL through
admission; `CaptureSelected(ctx, files)` consumes it on the scan worker into
immutable bookmark URIs, and `ReleaseSelected(files)` discards unstarted input.
Release never retires a native URL while capture is active. Capture is one-shot;
repeated occurrences inside one batch share the captured immutable result.
The native bridge retains original URLs before returning from the Apple Event,
performs metadata/bookmark I/O on the tracked collection worker, and balances
implicit access on capture/discard. Queue shutdown discards pending selections.
Collection scans with selected/scoped inputs perform no filesystem work on UI;
scan workers and their per-instance delivery queue are settled by the harness.
Tests: ownership exact-once/active cancellation/duplicate occurrence; queue stop
and cold-start delivery; blocked capture returns to UI, cancellation cannot apply
stale inputs, rejected input releases, capture error preserves current collection.
Verify: ordinary/Apple fileaccess and openwith tests; focused root opening,
collection and shutdown tests; native bridge serialization fixture; shard and
Qodana exclusions, vet/build and GoLand inspections. Native signed production
Open With/drop qualification remains explicit until actually exercised.
Budget: 0 spawns, 2 inline review rounds, complete suite only at final gate.
Apple's file-access guide confirms implicit access on selected/Dock URLs and
requires a matching stop after use; it does not authorize reconstructed path
strings to manufacture permission.
Source: https://developer.apple.com/documentation/security/accessing-files-from-the-macos-app-sandbox?changes=_4


#### Owned opening implementation and verification

Implemented original-NSURL retention for Apple Store Open With/Dock events,
worker-side capture, and exact-once discard through process and viewer queues.
Scoped scans use a tracked worker and drainable UI delivery; ordinary loose-file
opening retains its synchronous fast path. Cancellation cannot retire an actively
capturing native object or publish a partial collection. Shutdown discards native
inputs even when their scheduled UI callback never runs. Existing collection
replay and superseded-scan tests now explicitly drain retired scan delivery.

Native serialization exposed an escaped-URL identity defect: Foundation emits
percent escapes while Fyne's persisted URI strings contain decoded paths. Native
picker and Open With boundaries now decode once; persistent records remain
unchanged, including literal percent sequences. MIME fallback inspection stays
inside the acquired operation scope. No new dependency or shipped payload.

Evidence (base `03a2eaf`, this slice's working-tree changes; local logs retained
under `.scratch/apple-app-store-opening/`):

- Ordinary focused root opening/collection regressions pass. Apple-tagged root
  opening/collection race regressions pass (94.286s); after the final MIME-scope
  adjustment, the same Apple-tagged non-race regression set passes (7.812s).
- Final Apple-tagged fileaccess/openwith/filepicker race suites pass. The native
  serializer fixture captures a real bookmark for a filename containing a space
  and accent; picker cases also cover literal percent, hash and question marks.
  This fixture runs outside a signed production sandbox and does not qualify
  actual LaunchServices/Dock authorization or restored sandbox access.
- Translation parity and English identity tests pass. Shard inventory validates
  748 root UI tests across three shards. Exact Qodana exclusion added for the new
  fileaccess test file; existing exclusions cover modified test files.
- Negative Go overlays fail behaviorally when capture releases early, active
  capture loses ownership, failure omits batch discard, process/viewer shutdown retains
  queued input, or either native boundary keeps escaped path identities. The
  initial batch-discard test masked the defect by explicitly releasing before
  asserting; it now asserts release count at capture return and detects the
  deliberately broken implementation. Actual sources stayed intact throughout.
- `make verify-build` and the complete Apple-tagged application build pass.
  Apple-tagged vet of changed packages and Windows
  amd64 no-cgo vet of their shared code pass. GoLand inspections cover every
  changed Go file plus the Objective-C/header bridge, including weak warnings;
  only existing intentional filepicker test duplication is reported, covered by
  its exact Qodana exclusion. Reinspection after fixes is clear. IDE fallback
  evidence is not a fresh Qodana SARIF result.
- `make verify` stops at the native Linux/amd64 prerequisite: the local Docker
  daemon reports Linux/aarch64. Complete race/golden CI verification remains
  unverified, with no relaxed isolation or excluded worker test.

Remaining T2 work: native save/window-drop ownership, stale/missing grant
reselection and renewal, write/metadata/version/OS-handoff lifetimes, and worker
source transport. Signed production opening, runtime/package validation and
Intel/Apple Silicon qualification remain required before release. T2 remains
open; this is a completed opening-ownership slice, not App Store readiness.

Cost ledger: 0 spawns (budget 0); 2 inline review rounds (budget 2), with guard
strengthening inside final review; one full gate attempted and blocked before
suite execution. Build-only checks and focused native/race checks recorded above.

### Next slice — native save ownership and scoped image writes

Base `93c3040`; Deep route, T0 inline, zero delegates. Continue the agreed T2
fileaccess/picker/imaging/export boundaries with regression tests at those seams.
Task graph: destination lifetime -> native picker/export owners -> scoped writes
and native staging -> integrated verification. No dependency or payload change.

Contract: `NewDestination(uri, release)` owns the original selected save URL,
including a destination that does not exist. `ReleaseDestination(uri)` closes
admission, retires unborrowed access exactly once, and leaves active `Acquire`
borrows alive until their synchronous operation returns. Native save and mosaic
export workers retire the destination on every exit; imaging holds a bounded
borrow across serialization, encoding, commit and cleanup. Source metadata reads
hold independent access. Save Changes and metadata stripping acquire restored
source authority before path resolution and serialization.

Apple Store atomic writes use NSItemReplacementDirectory on the destination's
volume; ordinary builds retain sibling staging. Preserve cancellation before
commit, exact destination naming, symlink policy, permissions and Committed facts.
No direct truncate/write fallback. Missing destination bookmarks must not be
manufactured by expanding permission to the parent directory.

Native recon: ad-hoc signed WriteProbe.app with a genuine implicit grant for a
generated existing file denied a sibling write, allowed item-replacement staging
and succeeded in POSIX atomic rename (sibling=0, staged=1, rename=0). Foundation
bookmark creation for a nonexistent destination failed with Cocoa code 260.
The initial unbundled sandbox helper could not launch (-5); bundle identity fixed
that fixture setup. Evidence: `.scratch/apple-store-write-probe/result.txt`.
This is arm64 fixture qualification, not signed NSSavePanel production validation.
Sources: Apple's Accessing files from the macOS App Sandbox and FileManager
SearchPathDirectory.itemReplacementDirectory documentation.

Tests/commands: fileaccess destination active close/idempotence/cancellation;
native picker transport for missing and escaped names; imaging write refusal
with closed authority, cancellation/atomicity and source metadata scope;
root/mosaic export success/error/cancel ownership through existing OS stubs.
`go test -tags no_emoji,nodynamic[,appleappstore]` on those focused packages/tests,
focused race checks, negative guard overlays, shard/exclusion checks, GoLand,
`make verify-build`, Apple build/vet and Windows shared-package vet.
Final `make verify` retains the native amd64 prerequisite. Budget: 0 spawns,
2 inline review rounds; full suite once at final gate. Remaining scope excludes
window drops, stale-grant recovery, post-write source-version reconciliation,
OS handoffs, worker transfer and signed Store packaging/Intel qualification.


#### Autonomous continuation instruction

Ronin requested continued implementation and testing across all independently
completable App Store work, with a running protocol of results and human-only
steps collected last. Finish and checkpoint coherent slices; do not stop after a
slice merely because other implementation/testing remains. Preserve the full
feature set and existing commit authorization. No upload/submission/release is
implied. Continue T2/T3/T4 tooling and qualification before asking for credentials,
account setup, external hardware or product decisions that actually block work.

#### Save/export implementation evidence

The save-panel destination now retains its original NSURL through export worker
exit, without trying to create a bookmark for a file that does not yet exist.
`ReleaseDestination` is idempotent and nonblocking; active Acquire borrows keep
native access until the complete synchronous write has returned. Root and mosaic
exports discard destinations after chooser errors, cancellation, close, write
failure and success. Source metadata reads acquire their own scope. Save Changes,
Export and metadata stripping acquire authority before resolving/serializing the
write and retain it through staging cleanup and atomic commit.

Apple builds stage in Foundation's private same-volume replacement directory,
then use the existing sync/close/cancellation/rename transaction. Ordinary builds
retain sibling staging. Original symlink, mode and committed-result behavior is
covered by existing mutation regressions. No direct-write fallback was added.

Evidence, base `93c3040` plus this slice's working tree:

- Destination ownership tests failed before implementation (closed admission and
  omitted retirement), then passed. Mutation tests demonstrated unauthorized
  writes and premature encoding release before the wrapper was added. Both UI
  export flows failed all retirement cases before worker defers were added.
- Native save transport tests cover a nonexistent filename with percent/hash
  characters and decomposed accents, matching Foundation's filesystem spelling.
  The initial fixture used a precomposed accent and failed on normalization;
  after correcting that fixture, a deliberate plain-URI overlay fails the actual
  ownership assertion. No filesystem object is created by save transport.
- Focused ordinary tests pass in fileaccess, filepicker, imaging, root UI and
  mosaicwin. Apple-tagged race checks pass in all five (imaging 12.380s, root UI
  36.052s, mosaic 4.035s), including existing atomicity/cancel/symlink/metadata tests.
- Native ad-hoc signed Go fixture executes the actual imaging.ExportContext and
  fileaccess destination code under a file-only implicit sandbox grant. Results:
  sibling denied; existing-file replacement committed; newly absent destination
  created; cancelled export did not commit and preserved bytes. This is local
  arm64 filesystem qualification, not NSSavePanel UI or Intel qualification.
  A separately signed ordinary-staging build fails with EPERM on its temporary
  sibling and leaves the original fixture bytes unchanged.
- Negative overlays fail for early active-destination release, skipped mutation
  acquisition, metadata reads outside source authority, both export retirements,
  and flattened native save destinations. Actual sources remain unchanged by
  the overlays. Restored ordinary and Apple race tests pass.
- GoLand inspected all 16 changed Go files, including both platform staging files
  and tests. Its mutations_test WriteResult/error warning is the existing
  intentional Committed-on-error contract already covered by the exact
  GoDfaErrorMayBeNotNil exclusion. Existing mosaic test duplicate fragments are
  covered by that file's exact DuplicatedCode exclusion. No new suppression.
  IDE fallback is not a fresh Qodana SARIF result.
- `make verify-build` and full Apple-tagged application build passed.
  Apple-tagged and Windows/amd64 no-cgo vet passed.
  A native macOS shard inventory invocation included Darwin-only tests; the
  manifest is Linux-specific; the prescribed Docker target passed with 749
  runnable tests across three shards.
  Full native Linux/amd64 race verification remains subject to the recorded
  ARM-daemon blocker; no isolation policy is bypassed.

Logs and native fixture evidence are retained in `.scratch/apple-store-write-probe`
and `.scratch/apple-app-store-save/`. No third-party payload or dependency changed.
Remaining: stale grant renewal/reselection, post-write authority/source-version
reconciliation, window drop and OS handoffs, worker source transport, signed
runtime validation and packaging, full artifact/platform qualification. T2 remains
open. Budget: 0 spawns, 2 inline review rounds; focused tests only for this slice.

### Next slice — signed bundled runtime validation

Base `cae44c9`; T4, T0 inline, Deep route. Native dylib bytes change when signed,
but runtime admission still compares them with upstream archive hashes. The
worker executable also lives in an XPC bundle, so its executable directory does
not identify a shared app runtime. Fix these before building the full package.

Contract: `internal/macbundle` locates only the main app or the known nested XPC
image-helper layout and validates a regular, architecture-matching dylib in that
app's Contents/Frameworks. Native Security checks validate the library signature
and the outer app's complete resource/nested-code seal under PicFetch's signing
identifier. Apple runtime loading revalidates before dlopen. Models/cache cannot
redirect native code. Ordinary/MS Store checksums and directory policy remain.
Pre-sign staging always checks upstream payload hashes independent of build tags.
Native libraries go in Frameworks; license/privacy notices go in Resources,
following Apple's standard nested-code locations rather than mixing data/code.

Files: new macbundle locator/native verifier plus tests; similarity asset
admission/encoder/staging and focused tests; architecture/exclusions/evidence.
Tests: accepted main/helper layout, refused unrelated executable, wrong library
location/architecture, unsigned/modified library and modified outer resources;
valid ad-hoc signed temporary native bundle; pre-sign checks still reject bad
archives under Apple tags. Tests use generated C fixtures, no user application.
Verification: focused ordinary/Apple tests, native signing fixture, negative
guards, package build/vet, GoLand. Full Store packaging is the next dependent
slice. No new dependency; preserve pinned ONNX versions and notice obligations.
Budget: zero spawns, two inline review rounds, no extra complete race suite.
References: Apple Code Signing Tasks; TN2206 standard code locations and nested
resource seals; current code-signature format guidance. Ad-hoc qualification is
separate from Apple Distribution signing, provisioning and App Store processing.


#### Signed runtime admission evidence

Added `internal/macbundle`: known main/XPC-helper layout only, shared Frameworks
location, no escaping symlinks, exact dylib architecture/type, native library
signature and complete outer PicFetch resource/nested-code seal. Kernel
`proc_pidpath` identifies the running executable independently of argv and the
sandbox's changed working directory. Apple loading checks these seals again
immediately before dlopen. Pre-sign staging retains upstream checksum validation;
ordinary and Microsoft Store policies remain unchanged. Native system Security
and CoreFoundation APIs add no redistributed dependency.

- Generated native fixtures pass valid signing and reject modified/re-signed/
  unsigned libraries, altered outer resources, wrong architecture and escaping
  Frameworks/library links. Deliberately bypassing native validity or architecture
  causes the regression tests to fail behaviorally.
- Apple-tagged full affected-package race tests pass (macbundle 1.417s, similarity
  38.848s). Ordinary and Microsoft Store focused policy/staging tests pass.
  One existing checksum-cancellation test initially hit unbundled-executable
  refusal first; model checks now retain their cancellation behavior before the
  native runtime lookup, while Store installation still refuses missing runtime
  before any download.
- The real pinned arm64 ONNX 1.29.0 archive was staged with original hashes and
  all three notices. Its dylib was then signed inside a sandboxed app without
  network entitlement. Real CPU inference through CheckAssets/NewEncoder/Encode
  returned 768 finite values. The first relative-launch fixture used os.Executable
  after the sandbox changed cwd and failed to find models; the corrected native
  executable lookup passes both absolute and relative launches. One intermediate
  scratch fixture rebuild failed on an unused import; only the subsequent
  fail-fast rebuilt execution counts as qualification.
- The existing pinned Intel ONNX 1.23.2 archive was fetched from the exact upstream
  release and passed repository size/SHA-256 admission and complete notice staging
  under Apple tags. Rosetta execution is available for further Intel testing.
  ARM runtime load commands declare macOS 14.0; Intel's existing policy is 13.4.
- GoLand inspected every changed Go file, including warnings. Wrapped-error
  comparison and a fixture section nil check were corrected; reinspection is
  clean. `make verify-build`, Apple build/vet and Windows no-cgo vet passed. An
  initial format gate caught an unformatted scratch qualification helper; the
  corrected complete build gate passed. No full Linux race gate is implied.

Evidence: `.scratch/apple-app-store-runtime/` and
`.scratch/apple-store-runtime-probe/`; base `cae44c9` plus this slice. Ad-hoc code
validity is not Apple Distribution trust, provisioning or App Store approval.
The full packaged app/XPC workers and Intel execution remain next qualification
steps. Cost: 0 spawns, 2 inline review rounds; no repeated complete suite.

### Next slice — reproducible local Store bundle

Base `beb8491`; T4, T0 inline, Deep route; zero spawns/two reviews. Build an
ad-hoc sandboxed universal app using pinned Fyne v1.7.2, both pinned runtime
archives, current app metadata and native XPC components. Keep local qualification
explicitly separate from Apple Distribution signing and submission. No new
runtime dependencies or model-distribution decision; models remain optional data.

Files: scripts/macstorestage Go archive adapter; scripts/macstorepackage Python
orchestrator/validator and tests; Make entry point; packaging guide, architecture,
requirements, todos and evidence. Contracts: fresh output only, explicit metadata
(no source Build mutation), fixed app/XPC identifiers and sandbox entitlement
sets, architecture-specific minimum OS, standard nested-code paths, upstream
hashes before signing, complete upstream notices in Resources, deepest-first
signing and strict final validation. Missing prerequisites fail before builds.

Acceptance commands: Python packaging guard tests (bad entitlements, unexpected
code/dependencies/architecture, wrong minima, edited resources); actual universal
local build and strict code-signature readback; real HEIC worker probes through
the packaged broker on arm64 and Rosetta where available; all existing archive
staging guards, GoLand changed code inspections, make verify-build. Final package
hash/manifest and build log retained in .scratch. No credential requirement for
this local route; no distribution-ready or App Store acceptance claim.

Apple TN3125 distinguishes unrestricted macOS entitlements from restricted
profile-backed capabilities: TestFlight always requires a profile, but an
ordinary Store Mac app without restricted entitlements may not. Correct the
preflight/docs accordingly; do not assume every inherited helper needs a profile.

Ronin's overnight instruction: continue autonomous implementation/testing, but
make **no further commits** while Ronin is away because commits require Ronin's
signature. Leave reviewed changes and evidence in the working tree for return.

#### Packaging progress and dependent worker finding

The universal arm64/x86_64 ad-hoc bundle builds and passes nested/outer signing,
exact sandbox entitlements, architecture, dependency-path and deployment-minimum
validation. Pinned Fyne requires a Go source in its working directory even with
--executable; a staging-only stub satisfies it without changing FyneApp.toml.
The first minimum-OS parser incorrectly included LC_BUILD_VERSION's linker tool
version; the corrected parser distinguishes minos from the linker version.

A disposable thin arm64 derivative retaining production worker code passes real
HEIC 8/10-bit pixel probes through XPC. ONNX then fails EPERM reading model data
in the outer app Resources. The service has its own bundle/container; shared
bundle location alone does not confer authority. This is useful failing native
acceptance evidence, not an ONNX pass. Continue the dependent T3 grant transport
before declaring worker/package qualification complete.

### Dependent slice — transient worker authority

T3/T2, T0 inline, zero spawns/two review rounds. Preserve captured URI authority
in Explorer/search request contexts. Main exports fresh implicit bookmarks only
while source access is active; transfer exact source files, internal model/cache
roots, and the immutable outer app needed for signature validation. Do not widen
an external source to its parent or pass persistent app-scoped bookmarks to a
different process. Main retains acquisitions until worker return; worker resolves
and releases transferred scopes within its existing tracked process lifetime.
Invalid/cancelled transfer refuses launch. No new goroutines or global test seams.

Files: fileaccess transfer/context API with native build pair and tests; similarity
request capture/import build pair and clients; Explorer/search captured request
wiring; existing feature tests, architecture, qodana exclusions and evidence.
Verification: scoped capture/release/cancellation guards, request serialization
limits, existing client/search regressions, native packaged workers with real
models/source/cache permissions, no-network checks, focused race/build/vet/IDE.
Apple's Accessing files from the macOS App Sandbox documents options=0 implicit
bookmark transfer across processes. Such grants convey the rights the OS permits;
do not claim a read-only attenuation that the public implicit API does not offer.

#### Local bundle / worker evidence (overnight, no commits)

The local universal bundle passes full nested/outer strict signatures, exact
sandbox entitlements, architecture/dependency/minimum-OS checks. Seven Python
policy/artifact tests pass, including resource tampering, a validly re-signed
network-enabled service, a hidden non-executable-bit Mach-O resource and a
re-signed wrong deployment minimum. Both ONNX archives retain original checksums
and notices before signing. Make target `apple-store-package-local` produces the
same qualification layout and an external source/input/payload hash manifest.

Production HEIC/analysis/search clients now transfer fresh implicit authority
while retaining original source access until worker return. Disposable thin
arm64 and x86_64/Rosetta derivatives of the bundle pass real 8/10-bit HEIC pixels,
ONNX inference, private-container source/cache grants, cache reuse, retained
search, actual TCP/UDP denial and worker retirement. The GUI is replaced only by
a qualification driver; production worker/broker/service code is retained.
This is not physical Intel/minimum-OS or complete GUI qualification.

Affected Apple-tagged race suites pass: fileaccess 1.280s, similarity 38.031s,
Explorer 4.309s and visualsearch 1.292s. Four deliberate Go overlays fail for
premature release, flattened worker source, lost Explorer authority and lost
search authority. GoLand inspected all new transfer code and changed feature
code including warnings; shared client capture removed a confirmed duplication,
and goimports corrected one import grouping. Seven signed artifact guards pass.

Broader root UI tests exposed five legacy sort-admission waits that did not drain
scanUI after the earlier scan-queue change. The diagnosing-bugs workflow narrowed
the failure to scan completion queued before sort admission (other candidates:
source wrapper loss, changed admission). Adding the existing settleScan barrier
only to sort-stage fixtures makes the whole affected replacement/cancellation
subgroup pass in 1.030s, without a production behavior change or sleeps.

One qualification-driver build tried to replace an existing universal binary,
which Go refused; disposable copies now unlink that main executable first.
Explicit minimum-OS cgo flags also avoid reusing cache objects compiled for a
newer deployment target. A format gate caught a deliberately faulty scratch
overlay; formatting that fixture fixed the gate prerequisite. Final build/vet
and complete root-group output remain tracked in /tmp/picfetch-store-* and
/tmp/picfetch-worker-access-* until archived. No commit has been attempted since
Ronin withdrew overnight signing authorization.

### Next slice — OS handoff and source metadata scopes

T2, T0 inline, zero spawns/two reviews. Acquire captured authority on existing
workers before Trash serialization/native completion, clipboard file-reference
publication and Finder handoff. Keep original confirmed Trash paths to preserve
symlink semantics; never retarget a destructive action from a refreshed bookmark.
Scope favorite-preview source stat calls and write reconciliation before alias
resolution/version reads. Do not add desktop effects to unit tests: existing
uitest OS stubs prove active scope and exact release; closed authority must
refuse the native operation. Existing mutation/symlink and completion tests remain.
Files: fileaccess metadata helper/test; root batch/reveal/filework and tests;
delete feature and tests; favthumbs source-version helper/tests. Focused Apple
and ordinary tests/race, negative guards, GoLand and final Make build checks.
Wallpaper already writes an app-owned persistent PNG; native wallpaper and
Finder/pasteboard integration still require separate signed GUI qualification.

#### Handoff test evidence and GUI limitation

Four new assertions failed before fixes: clipboard/Finder/Trash released native
ownership during the OS call, and Favorite preview versioning stat-ed a closed
source. After scoped acquisition, ordinary focused tests pass (root UI 5.300s,
delete 0.451s, favthumbs 0.348s). Apple race checks pass in fileaccess, favthumbs
and deletion. The broad root regex also selected the direct-distribution updater
`TestApplyStagedUpdate_SavesNotesAndCallsApply`, which correctly cannot apply an
update in an Apple Store build; rerun with the actual file-action test prefixes.
Do not relax Store update policy. GoLand found a loop-defer warning in clipboard
scope collection; one explicit reverse-release list replaces it, preserving the
required simultaneous grants through pasteboard publication.

Computer-use skill was read for full GUI/picker qualification. Its node_repl
get_app_state call did not return after more than a minute and was terminated;
no generated PicFetch app process was observed afterward. Native picker/GUI
interaction is therefore unverified, not passed. Continue independent native
fixtures and implementation; leave the inaccessible GUI check in the final list.

### Next slice — native window-drop authority

T2, T0 inline, zero spawns/two reviews. Fyne/GLFW converts dropped NSURLs to plain
paths before the viewer sees them. Install a per-instance Cocoa content-view
subclass only on the shown main window in Apple Store builds. Override its drop
method to read native file URLs and deliver the existing retained-selection queue;
other views and ordinary builds retain the original method. Preserve ordering,
command admission, worker capture/discard and shutdown through the existing
Open With path; no new queues/goroutines or permissions. The root callback ignores
drop coordinates already. Native tests use synthetic objects/private pasteboards,
not desktop input, to pin one delivery, ownership, repeated install and unaffected
other views. Full real GLFW/picker/drop UI qualification stays unverified until
the GUI tool is available. Files: openwith native bridge/build pair/test, root Run,
architecture/qodana/evidence. Verify focused Darwin/Apple tests, builds, inspections.

#### Window-drop and handoff completion

Native drop ownership/repeated-install/isolation guard failed before the hook,
then Apple openwith race passed (1.291s); ordinary build passed. Corrected Apple
file-action root race passed (44.450s). GoLand inspected the new Go bridge/tests,
root Run, clipboard release fix and qualifier code. One ordinary-build constant
condition in Run is replaced by an unconditional platform dispatcher whose
ordinary implementation is a successful no-op. Native Objective-C bridge remains
covered by cgo compilation and native tests, not a GoLand Objective-C scan.

### Next slice — refresh restored bookmark authority

T2, T0 inline, zero spawns/two reviews. Resolution returns an operation-owned
path plus optional renewed bookmark bytes. Native resolution checks Apple's
stale flag and regenerates explicit app-scope data while access is active.
Acquire returns new immutable metadata; it never mutates a captured source.
The existing opening worker refreshes restored sources once per shared URI,
releases access before delivery, preserves duplicate occurrences, and leaves
unavailable sources intact for current per-item/offline handling. Cancellation
still aborts the opening. Refreshed records follow existing session/Favorite
serialization. No UI prompt or global mutable bookmark cache. Tests pin changed
paths/bytes, directory-relative ownership, exact release, cancellation and
unavailable fallback. Verify fileaccess race plus existing collection tests,
Apple native compilation, vet and GoLand. Real moved-folder relaunch still needs
signed native/GUI qualification.
Apple API contract: https://developer.apple.com/documentation/foundation/nsurl/urlbyresolvingbookmarkdata:options:relativetourl:bookmarkdataisstale:error:

#### Bookmark refresh verification

The new opening guards first failed on stale authority and missing cancellation.
After implementation, ordinary fileaccess race passed (1.365s); Apple
fileaccess/openwith race passed (1.272s/1.275s). GoLand inspected source, selection,
both native build pairs, source/transfer tests, Run and the ordinary drop bridge,
including warnings; no findings remain. Persistence roundtrip coverage added
before the final run. Scope renewal is operation-owned and copied before release;
original records remain immutable. Offline failure leaves the entry unchanged,
so it cannot silently replace a saved collection with only currently online files.
The collection/opening regression group is running separately.

### Next slice — submission facts and dependency privacy audit

T5, T0 inline, zero spawns/two reviews. Record exact dependency evidence, service
endpoints, draft listing/review notes and export-encryption facts. Do not publish
or select legal questionnaire answers. Apple lists Abseil and Protobuf, including
SDKs that repackage them; both ONNX notice files contain both components. The
pinned runtime tarballs include Privacy.md but no xcprivacy manifest. Investigate
exact upstream revisions and binary evidence before claiming SDK compliance.
Any unresolved upstream manifests/signatures stay explicit release blockers.

#### Submission records and remaining SDK artifact gap

Prepared submission-draft.md and privacy-dependency-audit.md with explicit
technical/account gates, not legal answers or a submission claim. Both runtime
binaries confirm Abseil/Protobuf via symbols; exact upstream dependency versions
match. Pinned tarballs lack xcprivacy files. Abseil's exact ARM-tag tree lists one,
but content retrieval and recursive GitHub API requests failed/time out. Do not
substitute default-branch or Objective-C Protobuf declarations without evidence.
PRIVACY.md now covers Apple bundled-runtime delivery and optional note artwork.
Collection/opening/Favorite Apple race regressions passed in 99.087s.

### Next slice — production worker negative source grant

T4, T0 inline, zero spawns/two reviews. Extend only the disposable qualification
main/script. A Go build overlay removes source export from the parent, retaining
app/model/cache grants and the unchanged production broker/service/image worker.
The same private PNG must fail per-item with no successful inference. Without
that overlay the refusal assertion must fail because the positive path succeeds.
Run the final overlay on both native ARM and Intel under Rosetta, alongside the
normal worker qualification. Do not ship the diagnostic flag or overlay.

#### Final local checks and inspection disposition

`make verify-build` passes after all production changes: format, pinned/generated
assets, notice checks, vet and build. Full Apple-tagged vet and Windows/amd64
no-cgo internal vet are clean. Apple race checks: fileaccess 1.269s, openwith
cached from its passing current-code run, Help 28.138s. Collection/opening root
race passed 99.087s. The latest universal artifact is
`.scratch/apple-store-package/overnight-final/PicFetch.app`; its external manifest
records the build source and every bundled file hash. It includes all production
changes, unchanged version/build 1.1.11/477, both architectures and strict nested/
outer code seals. Subsequent qualifier-only changes do not modify that app.

GoLand inspection requested every one of 46 changed Go/Python/shell/native bridge
code files with warnings enabled. Four production files showed duplicate-code
weak warnings caused by deliberately faulty `.scratch` Go overlay copies.
Renaming retained fixture payloads to `.fixture` and updating their overlay JSON
removed those false comparisons; all four affected inspections and the final
qualifier Go file are clean on reinspection. No production duplication suppression
was added. Native code is additionally compiled into both architectures; an
empty GoLand report is not a complete Objective-C/Qodana profile guarantee.
The JSON reports are archived in `.scratch/apple-store-package/`.

The first negative source fixture correctly hit the existing all-images-failed
terminal error before a Complete event. It now uses an allowed bundle image as
a control: the worker must initialize/infer successfully for that image and
report an OS permission error for the private ungranted image. Removing the
overlay makes the new assertion fail with two successful images, zero failures
and two inferences, confirming the guard. No production policy was weakened.

#### Final production worker and artifact results

Latest universal app positive qualification passes native ARM and Intel under
Rosetta: real HEIC pixels, ONNX inference, private source/cache access, cache reuse,
retained search, TCP/UDP denial and orderly worker exit. Negative source-grant
qualification passes both architectures: OS-denied private source plus successful
allowed control image, with exactly one inference. JSON results:
`overnight-workers.json` and `overnight-source-denial.json` in the scratch package
directory. Signed-artifact/policy suite: seven tests pass (1.381s), including
modified resource, re-signed network-enabled service, hidden Mach-O and minimum
OS tampering. The required full-suite platform check still refuses this host's
linux/aarch64 Docker daemon; no isolation test was skipped or policy relaxed.

SDK manifests/signature provenance, native GUI interaction (unresponsive tool),
physical Intel/minimum-OS runs and full native-amd64 CI remain unverified. The
Store distribution signing/installer route cannot be qualified without matching
identities/profile; the ad-hoc route remains deliberately non-submittable.
GoLand fallback evidence is not fresh Qodana SARIF, CodeQL or a GitHub review.
These gates precede the account/legal/submission checklist in submission-draft.md.

Cost ledger for overnight continuation: all slices T0 inline, 0 spawns, two
review rounds per implemented slice. Focused tests and native fixtures ran as
recorded; final broad local build/vet passed. Full race suite unavailable on the
selected Docker platform. No commits or pushes; HEAD remains beb8491 and Ronin's
signing authorization is still withheld. Remaining work stays in todos.md.

Final formatting and qualifier vet pass; Docker shard inventory passes with
749 runnable root UI tests assigned across three shards. Shell syntax and Python
compile checks pass. Evidence is archived under `.scratch/apple-store-package/logs/`;
`overnight-source.json` fingerprints the uncommitted source and
`overnight-worktree.patch` preserves its tracked diff. Human-only tasks remain
last; no credentials, questionnaire answers, uploads, commits or release actions
were attempted.

### Commit and push authorization

Ronin subsequently authorized committing and pushing the prepared work. Before
staging, all 57 changed files matched the archived overnight source hashes, with
no additional paths. Existing runtime test/inspection evidence therefore carries
forward unchanged. Only this authorization record, the TODO entry and submission
checklist were updated. Use the configured SSH commit signing; do not disable
signing. Push the existing feature branch, without merging or releasing.

### Continuation — exact SDK privacy inputs and retained Abseil resource

T5/T4, T0 inline, zero spawns/two reviews. GitHub's authenticated read API now
retrieves both exact Abseil manifests (20250512.0 and 20250814.0): same git blob
3ff4a9d98b13eafdc813fb5394796afd6cb8486b. Protobuf v21.12's complete non-truncated
source tree has no privacy manifest. Preserve exact upstream bytes, license and
source/version/hash provenance; do not substitute a current Protobuf manifest.
Package the identical Abseil declaration once in a macOS resource bundle, retaining
flat ONNX libraries and existing runtime admission. Validate its exact content
before signing and when inspecting a final app; signed-but-altered/missing resource
must fail. This supplies the known declaration, not an App Store acceptance claim:
Protobuf/ONNX declarations and upstream binary provenance still need resolution.
Files: packaging/apple-app-store/privacy/abseil inputs; package.py/test_package.py;
notices/README/audit/todos/evidence. Test existing packaging boundary first with
missing/changed manifest, then real signed artifact tampering. Verify Python suite,
new universal package, strict seals, worker regression and GoLand. The user's
f0565d1 E2E bundle remains untouched.

Pinned 1.29.0 Privacy.md was checked for its initialization-event warning.
Encoder already sets ORT_DISABLE_TELEMETRY=1 before dlopen and also invokes the
API opt-out before creating a session; no telemetry code change is needed.

#### Upstream runtime signature finding

Original versioned runtime dylibs extracted from the pinned archives both carry
Developer ID Application signatures from Microsoft Corporation, Team UBF8T346G9,
identifier libonnxruntime.1 (ARM timestamp 2026-08-12; Intel 2025-10-22). Debug and
test dylibs in the archive are not the shipped payload and are excluded. Add a
pre-sign guard after existing archive verification: native codesign strict/all-
architecture validation with Apple anchor, exact Team and identifier. Only then
may the local packager replace that signature. A valid ad-hoc signature must fail
this upstream provenance check. Keep post-sign app/nested seal validation separate.
Existing packaging tests are the agreed seam; add one native negative guard and
verify original libraries as positive fixtures. Preserve both signature records
in the build manifest. This establishes input provenance, not final Store trust.

#### Privacy/provenance implementation evidence

Missing-resource and staged-byte tests failed before implementation. The native
signed-artifact guard also failed when an altered Abseil declaration was re-signed
but final validation omitted the privacy check; it now rejects that artifact.
The upstream identity guard failed before implementation on a valid ad-hoc dylib;
strict Apple-anchor/Team/identifier checks now reject it and accept both original
Microsoft-signed libraries. codesign inline requirements require the leading '=';
positive fixtures exposed and corrected an initial command syntax error.

Eleven packaging tests pass (1.884s). A fresh universal package with the retained
resources and original signature records built successfully at
`.scratch/apple-store-package/privacy-provenance/PicFetch.app`. The preceding
resource-identical app passed production HEIC/ONNX/cache/retained-search/network
qualification on native ARM and Intel under Rosetta. The later change only adds
pre-sign identity verification/manifest records; it changes no worker code or
resource content. GoLand inspected both Python code files including weak warnings;
no findings. Reinspection follows the small provenance addition before commit.
The older f0565d1 E2E app/ZIP given to Ronin is untouched.

The final Make notice check caught normalized CRLF inside unrelated retained
updater license blocks. Restored those original bytes and limited the notice
change to the new ten-line Abseil paragraph. `make verify-build` then passed,
including the reviewed updater-source notice test. No license text was removed.
Both changed Python files were reinspected after the provenance guard; no findings.

Add the existing portable packaging policy/resource suite to CI's Ubuntu
validation job (five runnable tests; six macOS signed-artifact tests explicitly
skip without an artifact). Native eleven-test evidence remains a separate local
gate. This small workflow addition is within T4/T5, zero spawns/two reviews.
A draft PR will activate full native Linux/amd64 tests, CodeQL and Qodana; it does
not mark the unfinished Store release ready or authorize a merge/submission.

Final artifact at `.scratch/apple-store-package/privacy-final/PicFetch.app`
includes preserved upstream notice bytes and both provenance records. All eleven
native packaging tests pass (1.753s). Portable invocation passes five tests with
six explicit native skips. The CI workflow inspection is clean. All current
local gates passed before staging; full native-amd64 CI is pending the draft PR.

### Continuation — submission metadata and quarantine guards

Draft PR #75 now runs CI, CodeQL and Qodana against cd063e9. T4 Standard slice,
T0 inline, zero spawns/two reviews. Inspecting the built app found the required
NSHumanReadableCopyright key absent. Final validation also omitted quarantine
attributes. Apple preparation/upload requirements were rechecked 2026-09-30.
Use the existing LICENSE copyright line and photography category; validate both
in the final plist. Reject quarantine on the app root, nested directories or files
without mutating the inspected app or clearing downloaded user files.
Files: existing package.py/test_package.py, packaging README, todos and this record.
Seam: existing final signed-artifact validator. AC: re-signed missing/changed
metadata and quarantined copies fail; a fresh universal package passes all native
packaging tests. Run the Python suite, Make package target and GoLand inspections.
The user-facing f0565d1 E2E artifact remains unchanged. Distribution signing and
submission remain separate gates; this slice does not need account credentials.

### User regression — single-image Open dialog cannot navigate siblings

Ronin reproduced in the ad-hoc Store app using PicFetch's Open dialog. A temporary
Go overlay on the existing Left/Right UI test changes only the input to a
file-scoped record: `go test -tags no_emoji,nodynamic -overlay
.scratch/apple-store-package/sibling-repro-overlay.json ./internal/ui -run
'^TestHandleKeyEvent_LeftRightWalkFolderSiblings$' -count=1` fails: Right still
shows a.jpg, expected b.jpg. Changing only that record to captured folder authority
passes. The picker captures a single file grant; Parent correctly refuses to
broaden it. Keyboard handling and retained directory propagation work.

T3 Deep continuation, T0 inline, zero spawns/two reviews. Add an explicit native
folder grant after single-file selection in the Store Open dialog, preserving the
selected image as the initial image. Cancellation keeps the original file; no
parent access is manufactured from a path. Multiple selections/directories and
non-Store channels retain current behavior. Native UI work stays in the existing
tracked chooser and capture remains on its worker. Filepicker behavior tests cover
accept/cancel/wrong-folder and multi-file cases; existing Left/Right test adds a
scoped-folder chooser case. Test native transport, focused Go race suites, locale
parity, Make package and signed guards, then GoLand. Build a fresh E2E app without
overwriting the original. Finder/Drop entry-point folder consent remains a
separate open workflow; do not silently claim this Open-dialog fix covers it.

### CI follow-up at cd063e9

PR #75's first run passed CodeQL, validation, Linux native/non-UI/ui-2, ARM native
and FOSSA. Three failures have concrete causes: Windows worker-access expected
backslashes even though Fyne URI paths use slashes; the HEIC progress test observed
worker traversal before draining scanUI; Intel's unsigned Go test binary could
not retrieve Foundation's app-scope key. The latter reproduced locally under
Rosetta, then passed with the identical binary ad-hoc signed. T0 inline fixes
only these test assumptions: URI-form expected paths, drain already-submitted
progress, and run the bookmark test in a signed disposable self-copy (no skip).
Verify focused race tests plus unsigned Intel parent invoking its signed child;
CI must still rerun on actual Windows/Linux/Intel. No production sandbox changes.

Qodana post-suppression `/qodana.sarif.json` reports eight unused-function warnings.
Each has live callers: Stat (favthumbs/name.go and ui/filework.go), NeedsCapture
(ui/drop.go), HasScope (favstore and ui/drop.go), Export (worker_access_darwin.go),
Import (worker_access.go), CurrentRuntimeDirectory (similarity/assets.go and native
worker launch), InstallWindowDrop and Stop (ui/run.go). These are cross-package/
platform analysis false positives, not removable code. The `/start/` SARIF is
baseline output and must not overwrite the actual result. Keep the warnings and
these exact dispositions visible; no broad unused-code exclusion is introduced.

Folder-consent policy, cancellation, wrong/unscoped directory, native picker,
Apple-tagged picker and ordinary/authorized-folder Left/Right tests pass. English/
German locale parity passes. Universal test app is `bin/apple-store-e2e-folder-navigation/PicFetch.app`
and its matching ZIP/hash are beside the output directory. All 13 signed artifact
guards pass (3.231s). The app's manifest records the exact working tree used;
subsequent changes are inspection comments/test fixes/docs, not runtime behavior.
The native live permission panel still needs Ronin's E2E confirmation; original
f0565d1 app is untouched. GoLand inspected all seven changed packaging/picker/UI
files. Existing test duplication remains covered by exact qodana.yaml test
exclusions; immutable distribution-channel condition has a narrow documented
GoBoolExpressions suppression. No other findings in that scope.

Focused race verification passed: similarity 1.427s, openwith 2.665s, root UI
7.655s. Unsigned Intel parent now runs the real signed child guard successfully
(0.64s), preserving native capture assertions. GoLand reinspection of all three
CI test fixes and the picker condition suppression is clean. `make verify-build`
passes; native package suite is 13/13. No top-level root UI test was added, so
existing shard assignments remain exact. Source-local test duplication exclusions
are unchanged. Final review: permission scope remains explicit, original image
identity survives consent, cancellation does not widen access, and no production
sandbox policy is relaxed. CI results for the next pushed commit remain pending.

### User regression — dropped single files need the same folder consent

Ronin confirmed the Open-dialog fix, then reproduced the same symptom with one
file dropped on the dropzone. Extend consent at the shared collection-discovery
boundary, after native URL capture and before sibling scan. This also covers OS
Open With. Move consent out of Choose so there is no duplicate prompt. Ask only
for a single protected image being expanded, excluding merge/replay/multiple/
directory inputs. Preserve native permission code and the folder policy guards.

T3 Deep continuation, T0 inline, zero spawns/two reviews. A per-viewer consent
function is captured on UI; the existing scan worker owns capture/permission/scan.
Reserve its native-prefix membership in openChooserWorkers on UI before launch;
retire that prefix before scanning, including every early error. Context is checked
before native presentation and after its return; the scan token still gates result
delivery. No extra goroutine or blocking shutdown wait. Tests stub per-viewer
native consent, never the real desktop. The root harness installs that stub and
already cancels scans before draining native and scan workers. Verify the original
Left/Right test through picker/drop/OS delivery with native selected-file records,
plus cancellation/replacement and no prompts for merge/replay. Rebuild a new E2E
artifact once focused race tests, Store native compile and GoLand pass.

Shared-consent verification: picker/drop/OS selected-file tests initially failed
with Right still showing a.jpg. All now pass with consent in the shared path.
Focused race suite covering opening, discovery and lifetimes passed (UI 16.466s,
filepicker 1.290s); Store-tagged picker passed (0.279s). Native-prefix worker counts
are reserved on UI and retired before scanning; harness cancellation precedes
both native and scan waits. Existing folder authority bypasses the prompt.
GoLand inspected all eight changed production/harness/test files; only existing
exact-excluded test duplication remains. `make verify-build` passes. The direct
host shard check correctly exposed macOS-only tests absent from the Linux manifest;
required `make check-test-shards` via Docker then passed all 750 tests/three shards.

New universal E2E app: `bin/apple-store-e2e-drop-navigation/PicFetch.app`, with
TESTING.md and manifest; ZIP and SHA-256 are beside its output directory. All 13
artifact guards pass (3.228s), and ZIP extraction preserves all 24 file hashes and
strict nested signatures. Earlier E2E artifacts remain untouched. User-facing
Open-dialog navigation was confirmed by Ronin; live drop/Open With confirmation
is pending for this newer bundle.

The 38ec88e CI round passed both macOS architectures, the repaired Linux ui-3
shard, CodeQL and validation. Windows showed the path fixtures still lacked a
Windows volume; changing separators alone was insufficient. Both affected tests
now derive absolute fixture roots from t.TempDir and compare fully qualified URI
paths. Focused local race tests pass (filepicker 1.469s, similarity 1.520s); actual
Windows confirmation remains pending the next CI round. Fresh Qodana has exactly
the same eight documented unused-function false positives and no new findings.
Final review covered consent ownership/currentness, no duplicate picker prompt,
merge/replay exclusions, native cancellation and artifact provenance. Two reviews,
zero agents. No Store submission or distribution-signing readiness is claimed.

### Follow-up — native guard inventory and multiple-selection report

75356d9 CI passed every Linux race partition, Linux native guards, both macOS
architectures, validation, CodeQL and FOSSA. Windows passed the repaired path
fixtures, then stopped because the Store inventory requires
TestStoreManaged_MicrosoftStoreBuildIsTrue but the test was renamed to
TestStoreManaged_StoreBuildIsTrue. Thin T0 fix: restore the explicit Microsoft
name in the existing test; no production change. Store-tagged test discovery now
selects that exact required name and scripts/nativeguards tests pass (0.455s).
GoLand inspected the changed distribution test with no findings. Actual Windows
confirmation awaits the next CI run. Zero agents, one review.

Fresh 75356d9 Qodana post-suppression /qodana.sarif.json has nine unused-exported-
function warnings: the eight previously recorded false positives plus
filepicker.AuthorizeSiblingFolder, assigned in internal/ui/build.go:115. Rechecked
all nine callers; none is unused. No new actionable finding or broad suppression.

Ronin subsequently reported multiple-file Open-dialog/drop navigation still
failing. A new focused diagnostic subcase in the existing Left/Right test opens
two distinct selected-file bookmarks through picker/drop/OS; all three retain
two images and navigate Right to b.jpg then Left to a.jpg (UI 0.793s). This does
not reproduce the packaged-app report. Ronin clarified that the reported behavior
was reaching the selection boundaries and confirmed it is intentional: multiple
inputs must browse only the selected images, while a single-file open/drop may
discover folder siblings. Close this report as expected behavior; preserve the
selection-size and Left/Right coverage. No production change or rebuilt artifact
is needed for this report. Focused navigation race tests also passed (UI 6.128s).
GoLand inspected the diagnostic test: only the existing exact-excluded duplicate
fixture remains. This clarification supersedes the earlier pending diagnosis.

### Persistent sibling-folder approval — implementation contract

Request: reuse explicit folder approvals across fresh single-image opens and app
launches. Preserve multiple-selection/merge/replay behavior. Deep continuation,
T0 inline, zero spawns, two reviews; no dependency, entitlement or UI string change.
Apple's Accessing files from the macOS App Sandbox documentation confirms stored
security-scoped bookmarks require explicit resolution/start/stop on later use.
Existing fileaccess.Acquire already renews stale bookmarks and balances scopes.

Task: add per-viewer filepicker.FolderAuthorizer backed by app preferences;
resolve saved directory bookmarks on the existing scan worker before prompting,
require the resolved directory to match the requested parent, save renewed data,
and persist only validated explicit folder consent. Invalid/missing/offline grants
fall back to the existing panel; cancellation must not publish a new grant.
Never infer permission from a matching pathname alone. Keep native scopes bounded
and no native work under the preference lock. Filepicker owns the grant preference
as part of native input authorization, independently of transient session data.

Files: filepicker folder authorizer + tests, UI build wiring, architecture index,
exact Qodana test exclusion, evidence/todos/docs. Tests: a new authorizer instance
with persisted preferences suppresses the second prompt; unrelated/failed/moved
bookmarks, renewal, cancelled consent and multi-selection retain their contracts.
Verify focused picker/UI race tests, Store compile, GoLand changed files,
make verify-build and fresh universal Store test package/13 artifact guards.
Physical relaunch confirmation remains a user E2E check. Full native-amd64 race
suite remains CI-owned because the local Docker daemon is ARM.

Persistent approval verification: the new-authorizer test first failed with two
prompts instead of one, then passed after persistence/reuse. Tests cover distinct
images across authorizer lifetimes, explicit selection identity, balanced scopes,
moved and renewed bookmarks, wrong resolved folders, revoked/offline/malformed
records, file-only grants, cancellation during lookup/panel, and multi-selection.
Renewal tests exposed duplicate trailing-slash records from deriving a folder
back from its child; fixed by persisting the validated native folder directly.
Isolated Go overlay mutations disabling folder matching and renewal persistence
both fail their targeted guards. Overlays are scratch-only .fixture files.

Focused race results: filepicker 1.546s; root navigation/consent lifecycle 8.518s.
Store-tagged filepicker 0.364s. make verify-build passed formatting, notice checks,
vet and build. GoLand inspected all four changed code files including weak
warnings; corrected import ordering and reinspected the two new files cleanly.
The exact folders_test.go Qodana exclusion is added; no root top-level tests were
added, so the existing shard manifest remains unchanged. Two inline reviews,
zero agents. No new runtime/model dependency or entitlement; existing notices
and SDK/privacy release gates remain unchanged.

Fresh universal bundle: bin/apple-store-e2e-persistent-folders/PicFetch.app;
ZIP/SHA256 beside the output directory, TESTING.md and source manifest inside it.
All 13 signed-artifact guards pass (3.179s); all 24 ZIP payload hashes match the
manifest. Existing E2E bundles preserved. Native relaunch/moved-folder UI evidence
remains pending Ronin; unit tests simulate native resolution. Previous builds do
not have a separate stored approval history, so one initial approval per folder
is expected in this build. Current runtime change remains uncommitted; previous
Secretive documentation signing was refused, and the earlier b98d848 push ended
with an SSH disconnect. Latest remote remains 75356d9; no fresh CI pass is claimed
for this change. The completed local package includes the uncommitted sources
and records their hashes explicitly.

User E2E confirmation (2026-09-30): Ronin tested the persistent-folder build
and reported "Awesome yes it works perfectly!" in response to the requested
quit/relaunch check. Persistent approval reuse across an actual app relaunch is
confirmed. This does not establish moved-folder behavior, which remains a
separate live qualification item. No further runtime changes were needed.

### Distribution packaging tooling — contract and limits

Deep continuation, T0 inline, zero spawns/two reviews. Add a separate
make apple-store-package-signed route consuming an already-qualified local app
and writing only a fresh output directory. Never change the E2E bundle, upload,
install credentials, or install the output. Reuse existing native/entitlement/
privacy verification and retain the input manifest/hash evidence.

Select explicit valid application/installer identities for the requested Team;
reject wrong-role, absent or ambiguous identities. Decode supplied profiles with
security cms and diagnose macOS platform, exact app ID/prefix, Team, expiry,
distribution-only scope and allowed signing certificate. Profile plist inspection
is a local diagnostic, not proof of Apple's CMS/DER acceptance. Require app and
XPC profiles for this route's TestFlight option; keep inherited helpers unprofiled
with their exact sandbox/inherit entitlements. Sign inside-out, verify every code
item against the Apple anchor, selected leaf fingerprint and Team, then create a
productbuild --component /Applications installer. Validate installer signature
and expand it to check that its payload matches the signed app exactly.

Primary references (2026-09-30): Apple TN3125 (profiles, macOS exceptions, profile
location) and Creating distribution-signed code for macOS (manual signing and
nested code), plus installed productbuild/pkgutil/codesign manuals. These are
version-sensitive build diagnostics. Existing SDK privacy gaps remain release
gates regardless of signing success.

Files: scripts/macstorepackage distribution module/tests, narrow package.py
verification parameter, Make target, CI portable-test discovery, architecture,
README/todos/evidence. Verify portable policy tests, real native unsigned installer
payload roundtrip and negative guards, existing 13 artifact guards, GoLand
inspections and Make checks. Missing certificates/profiles prevent a true signed
artifact test; record that boundary rather than substitute ad-hoc evidence.

Distribution-tooling verification (2026-09-30): policy tests initially failed with
13 missing-validation assertions. Completed policy/assembly tests cover wrong
identity role/Team/certificate, wildcard/wrong app IDs, legacy App ID prefixes,
expired/future/development profiles, exact helper/service claims, immutable
profile snapshots, TestFlight profile requirements, installer leaf fingerprint,
wrong destinations/scripts/extra or changed payloads and signing order. New
portable tests are included in CI discovery. No Apple credentials are read by
those tests; command assembly uses explicit mocks and is not signing evidence.

Actual native productbuild --component/pkgutil --expand-full roundtrip of the
persistent-folder test app passed; extracted files and executable permissions
match and codesign --verify --deep --strict still passes. The disposable installer
was unsigned and never installed. A real ad-hoc artifact fails the new
Apple-anchored distribution identity requirement as expected. Full native/portable
packaging suite: 24 tests, 10.002s, all passed. Python scripts parse with the 3.11
grammar; make fmt-check and Make target dry-run pass. Prior unchanged Go runtime
build/vet/race evidence carries forward from the persistent-approval section.

GoLand inspected distribution.py, test_distribution.py, package.py, Makefile and
CI YAML with no findings; final changed Python files were reinspected cleanly.
Review added exact installer certificate SHA-256 comparison after pkgutil trust
validation, immutable profile decoding/embedding, and input-to-staged file
comparison before signing. Two reviews, zero agents. Preflight's TestFlight
check was aligned to require separate app/worker profiles; shell syntax passes.
Xcode 27.0 is available, but this invocation has no configured Team ID or either
TestFlight profile. Certificate availability is not established without the Team
input. Therefore real distribution signing, profile CMS/DER acceptance, installer
trust output against actual certificates, TestFlight and Apple SDK/privacy review
remain unverified. No distribution artifact, upload or installation was performed.
The user-confirmed local E2E app is unchanged by this tooling work.

### Lunch-break handoff and latest CI — 2026-09-30

Signed commit 62661a19291e5f936dc02751b88914158c50beaa was pushed before
Ronin requested the lunch-break commit pause. This supersedes earlier local-only
status in this chronological record. No further commits are authorized until
Ronin returns and authorizes them. The existing E2E app remains unchanged.

All three runs completed successfully on exactly that revision:

- [CI 36697931823](https://github.com/frathe/picfetch/actions/runs/36697931823):
  validation, all four native Linux/amd64 race shards, Linux native guards,
  Windows tests, and both macOS architecture native guards passed.
- [CodeQL 36697931680](https://github.com/frathe/picfetch/actions/runs/36697931680):
  Go and Actions analysis passed.
- [Qodana 36697931732](https://github.com/frathe/picfetch/actions/runs/36697931732):
  scan completed; post-suppression /qodana.sarif.json contains eight findings.
  Reviewed the actual report in .scratch/apple-store-package/qodana-62661a1/,
  not the pre-suppression CSV or start baseline. All are known
  GoUnusedExportedFunction false positives: fileaccess.Stat, NeedsCapture,
  HasScope, Export, Import; macbundle.CurrentRuntimeDirectory;
  openwith.InstallWindowDrop and Stop. Rechecked current callers in ui,
  favthumbs, favstore and similarity. The former global folder-authorizer warning
  disappeared after the per-instance change. No actionable finding or broad
  suppression added. The Qodana for Go annotation check is neutral, not a report
  with zero findings.

This is platform/native-guard evidence, not physical minimum-OS GUI or Apple
submission acceptance. No fresh GitHub Codex code review was requested, so no
new clean bot-review claim is made. Local changed-code GoLand evidence above
still applies; this continuation changes documentation only.

A separate read-only security find-identity inventory found zero valid Store
application and zero valid Store installer identities with private keys in the
accessible Keychain. Only aggregate counts were retained. This establishes the
local credential gap, not the state of Ronin's Apple membership or identities
held elsewhere. Xcode installation/license work is already complete.

Created packaging/apple-app-store/ronin-checklist.md with ordered account,
identifier, certificate and two-profile setup; a handoff back to Pico for actual
signing; remaining live E2E/platform checks; owner listing decisions; and separate
upload/review/release decisions. Checked Apple primary guidance for current
portal terminology and linked it in the checklist. Updated README, submission
draft and todos to point to it and reflect the commit pause. SDK privacy closure
remains an explicitly technical responsibility; no declaration or Apple approval
was invented to make the handoff appear complete.

Ronin returned from lunch and explicitly authorized committing and pushing the
current state. Updated the checklist/submission/todo status to end the pause.
This handoff changes documentation only; local Markdown targets resolve and
git diff --check passes. Runtime inspection/CI evidence remains at 62661a1.
