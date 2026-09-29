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
are complete, Apple builds are experimental and must not be submitted. No
signed sandbox test, Apple validation or App Review result exists yet.

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
