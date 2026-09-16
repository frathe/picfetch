# Experimental HEIC opt-in implementation

Date: 2026-09-16. Route: Deep SDD/TDD. Status: implementation integrated;
remaining checklist evidence, installed-MSIX qualification and an external AI
scan keep final acceptance open.

Resuming agent: start with the [handoff](../.scratch/experimental-heic-opt-in/handoff.md)
and reconciled ticket checklists. On 2026-09-16, 48/57 items are checked and
tickets 01/02/03/04/06/07 are resolved. Later handoff notes distinguish completed native
constructor tests from compound application scenarios still needing proof.

Deliver the accepted [spec](../.scratch/experimental-heic-opt-in/spec.md) through
the [nine approved tickets](../.scratch/experimental-heic-opt-in/issues/README.md).
Ronin invoked `/implement use SDD and TDD`; the preceding breakdown is accepted.
The prior restoration plan remains the historical helper qualification record.
Initial HEAD: `fad62e69d896171aa3dd59508fb1c4cf536e7fb5`; existing plan/todos edits
are the specification pointers prepared in this conversation.

## Contracts and limits

- `imaging.Reader.IsSupportedImage(fyne.URI) bool` and
  `SupportedExtensions() []string` describe the reader's immutable capability.
  Zero reader and package-level queries remain unchanged.
- `filescan.Option` and `WithAdmission(func(fyne.URI) bool) Option` extend
  `Images`/`Siblings` with optional arguments; omitted/nil admission uses the
  existing default. All viewer opens capture its foreground reader predicate.
- `preferences.State.ExperimentalHEIC` persists as `experimentalHEIC`, false by
  default. Settings edits affect the saved intent only. Startup loads preferences
  before constructing one shared HEIC owner; readers never change mid-session.
- Startup package discovery derives the installation from the running executable.
  Package/launch failures retain intent and supply localized unavailable status.
  Fixed helper identity and all existing readiness/admission/limits remain intact.
- Windows preparation publishes verified content-addressed copies in private
  app storage; its exact lifetime contract is pinned before T04 code. Safe
  cleanup may defer, but it cannot remove a copy in use or publish partial bytes.
- No decoder source/dependency/OS association/CLI override changes. Mac total
  native-memory limitation remains accepted. Production signing, distribution
  clearance and broad color qualification remain release gates.

## Task graph and file map

`01 -> 02 -> {03, 04, 06, 07}`; `04 -> {05, 08}`;
`{03, 05, 06, 07, 08} -> 09`.

| Ticket | Files / modules | Test and verification | Owner / budget |
| --- | --- | --- | --- |
| 01 | imaging source, filescan, UI drop/images tests; architecture | Real admission through injected reader; T01 execution gate | Lead; 0 implementation spawns; 1 review; focused only |
| 02 | preferences, UI startup/images/settings/run/features, settingswin, translations | Actual Settings surface and saved choice across viewer lifetimes; T02 gate | Lead; 0 implementation spawns; 2 reviews; focused only |
| 03 | UI consumers, Explorer preset choices, similarity integration tests | Admitted uncached inputs reach shared readers/owner; T03 gate | Lead; 0 implementation spawns; 1 review; focused only |
| 04 | HEIC client package/preparation platform files, winisolation, startup | Verified copied launch, concurrent publishers, damaged cache, lifecycle; T04 gate | Lead; 0 implementation spawns; 2 reviews; native evidence required |
| 05 | msixstage/Store workflow, nativeguards and package activation fixtures | Installed test-MSIX under standard user on both architectures; T05 gate | Lead; 0 implementation spawns; 1 review; native evidence required |
| 06 | macOS packaged activation fixtures/nativeguards/CI | Complete native app on amd64/arm64; T06 gate | Lead; 0 implementation spawns; 1 review; native evidence required |
| 07 | Linux packaged activation fixtures/nativeguards/CI | Complete native app on amd64/arm64; T07 gate | Lead; 0 implementation spawns; 1 review; native evidence required |
| 08 | Windows native account/package fixture/nativeguards/CI | Standard-user package on amd64/arm64; T08 gate | Lead; 0 implementation spawns; 1 review; native evidence required |
| 09 | all touched code inspections, plan/todos/architecture/threat/qualification | Parent AC1–12, complete CI and fresh code/security reviews | Lead; 0 implementation spawns; final review; full suite once |

Each ticket's exact commands and acceptance cases are in the linked execution
record. Run one behavioral red/green slice at a time, using the accepted seams.
Missing native environments and permission-query failures remain unverified
blockers. No helper-only or cross-build result substitutes for packaged runtime.
Update this file with exact new native invocation contracts before qualification.

## Delegation

One read-only Windows API/lifetime scout runs alongside lead-owned ticket 01.
G1: bounded current API question; G2: reported call sites checked with source;
G3: no writes; G4: three platform files rather than full feature context;
G5: lead has not traced this platform lifetime. No design, review, platform
implementation or user-visible strings are delegated. Additional scouts require
a recorded bounded question; no concurrent code ownership overlaps.

## Evidence and cost ledger

Resume 2026-09-16: retain Deep routing and approved seams. One read-only scout
locates existing consumer evidence and completion observables in the UI tests,
Spiral and Grid while the lead traces native activation fixtures. G1: bounded
consumer/lifetime question; G2: source-location claims checked with `rg`/reads;
G3: no writes; G4: consumer sweep is smaller than the complete feature context;
G5: lead has not traced these consumers in this resumed session. No design,
review or fixes delegated. Lead adds missing behavioral coverage one slice at
a time, negatively verifies new guards, and keeps native environment blockers
explicit. Budget: one scout, focused tests, complete suite through CI.

Resumed slice A (lead): extend `internal/ui/images_native_test.go` through the
existing packaged constructor to prove mixed-directory HEIC/HEIF/PNG navigation,
saved disable during an admitted native request, cancellation of a retired
foreground load on collection replacement, ordinary replacement pixels, and
continued HEIC use before restart. Existing package construction and execution
commands below remain unchanged. Verify with
`go test -tags no_emoji,nodynamic,heicnative -count=1 -run '^TestNativePackagedHEICActivation$' -v ./internal/ui`;
observe a negative control before recording green. No new runtime seam or
decoder behavior is planned. Other architectures require fresh native CI.

| Slice | Red / green / inspection evidence | Status |
| --- | --- | --- |
| Publication | Nine tickets published, ticket 01 claimed; prior spec preserved | Complete |
| 01 | Real-admission consumer red: unsupported scan never began sort; green imaging/filescan and direct/folder/sibling/session/Favorite UI cases | Ticket resolved; integrated final gate separate |
| 02 | Red: missing Experimental tab/default activation; green preference and real-checkbox separate-lifetime tests. Later red: stale open Settings; green live status update | Ticket resolved; platform/final gates separate |
| 03 | Red: active HEIC choices absent and saved rule rejected; green Explorer/source consumers and native analysis/retained queries. Resumed uncached duplicate/Spiral and active-analysis/source-replacement guards pass with negative controls | 6/6 items complete; final integrated native gate remains separate |
| 04 | Red: no staging publication; green owned bounded copy/reuse/repair/cancel/preparation-refusal cases. Windows cache/concurrent-process/lease tests pass natively on both architectures at 69fef1a | Ticket resolved for staging; installed-MSIX context separate |
| 05, 08 | Standard-user standalone guards pass on both architectures at 69fef1a. Disposable signed MSIX installs, but both COM and direct activation fail before the test process starts | Installed-MSIX qualification blocked on a suitable interactive standard-user environment |
| 06 | Both native macOS architectures pass expanded package failure, sandbox-readiness refusal, navigation/cancellation and analysis guards at c1b6890 | 6/6 items complete; production GUI smoke and release clearance remain separate |
| 07 | Both native Linux architectures pass mixed-directory, package-failure recovery, navigation/cancellation, analysis and helper/seccomp/resource guards at c1b6890 | 6/6 items complete |
| 09 | Complete Linux race partitions and validation pass at 69fef1a. Fresh code/security reviews have no findings; Qodana has zero final results, CodeQL only its two existing dismissed false positives. IDE and sixteen Settings layouts verified | Installed-MSIX and the external GitHub AI scanner remain blocked |

Budget: one initial read-only scout; lead implementation and fixes; focused
tests during iteration; complete suite once at final gate. No qualification,
commit, push, review or release is claimed by this plan's creation.

Second read-only scout: locate packaged-activation runners and evidence hooks
in `scripts/nativeguards/main.go`, `.github/workflows/ci.yml`, and
`.github/workflows/microsoft-store.yml`. G1–G5: bounded three-file read,
source-location oracle, no writes, independent native context, not yet traced
by lead. Lead continues restart/Explorer implementation. No design delegated.

### Native application fixture contract

`TestNativePackagedHEICActivation` is a `heicnative` application-package test.
The native guard runner must require it separately from the broad helper suites.
It copies the application test executable into the fixed installation layout,
runs `go run ./scripts/heicpackage -os <host> -arch <host> -out <package>`,
and on macOS independently verifies the helper and ad-hoc signs/verifies the
enclosing application. The relocated executable runs the existing shared
startup/viewer harness, actual Settings checkbox, normal admission and native
helper. This is application-constructor evidence using Fyne's test driver;
rendered layout and production GUI/package qualification remain separate checks.
Windows execution additionally requires a real non-administrator token. Installed
MSIX uses a disposable signed test package and requires real package identity;
its immutable source helper bytes and ACLs are compared around activation.
No production CLI flag or activation environment variable is introduced.

### Local verification checkpoint

- `go test -race -tags no_emoji,nodynamic -count=1 ./internal/heicdecode/... ./internal/filescan ./internal/imaging ./internal/preferences ./internal/ui/settingswin ./internal/ui/explorer`: all pass (client 8.654 s, worker 14.357 s, imaging 24.185 s; remaining packages 1.199–5.924 s).
- `make verify-build`: pass, including vet/build, all existing notice/asset checks,
  native import guard and reproduced WASI artifact.
- `make generate-heic-wasm`: refreshed only the build-tool status-message digest
  in decoder.json; guest hash remains `cdaf9af71d8a8624c620a2ca8f90865a00a2d2dc228bb8bdb058564726d769ed`.
- `make check-qodana-test-exclusions`: pass. Root UI tests are assigned in ui-2.
- `make verify` and `make check-test-shards`: blocked by absent Docker daemon
  socket. A direct host shard check correctly cannot qualify the Linux/amd64
  manifest (its macOS-only runnable differs); no assignment is added for that
  platform-only test. Do not bypass the canonical platform requirement.
- `gh pr view`: existing PR #28 remains at initial `fad62e69`; all current work
  is local and uncommitted. No fresh CI/review is claimed for these changes.
- Layout PNGs: `.scratch/experimental-heic-opt-in/evidence/layout/` (16 locale,
  theme, optional Cache and Store combinations). Existing small saved geometry
  retains Fyne's overflow behavior; a new default Settings window is 640x520.

Lead review found a pre-input native-probe storage failure that was not classified
unavailable. Its owned-file regression failed first, then passed after setup
errors were classified without changing any policy or limit. Request cancellation
continues to take precedence over availability diagnostics. The lead owns all
review/fixes; the code-review skill's separate standards/spec axes are applied
inline because the repository agreement prohibits delegated reviews.

### Final local review checkpoint

The lead applied both standards and spec review axes and fixed the evidence gap
in successful analysis subprocess coverage. `TestNativeHEICAnalysisPixels`
uses a complete native helper package, admits the owned fixture through filescan,
and verifies pixels through the shared owner into analysis and two uncached
preview queries in one retained search subprocess. It also checks cancellation
and process joining. Removing the attached owner produced the expected failure
before restoration. No model inference or color-fidelity claim follows from it.
All three platform suites now require this guard; the registry test was observed
failing before registration and passing afterward.

- Focused UI/Settings race regressions: pass (`internal/ui` 14.941 s,
  `settingswin` 4.076 s); translation/manual guards pass.
- Native analysis/attachment/preview race regressions: pass (14.379 s).
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet -tags no_emoji,nodynamic ./internal/...`: pass.
- Final installed Store UI and native analysis test binaries compile for Windows
  amd64 and arm64; execution remains unverified.
- `make security-govulncheck`: no reachable vulnerabilities in the application;
  one required-module finding outside imported packages/called symbols. Separate
  WASI guest scan reports no vulnerabilities. Dependency versions are unchanged.
- Packaging/registry tests pass: nativeguards, heicpackage, plistdoctypes,
  msixstage (3.987 s); Explorer preset validation exercised by UI tests.
- Final GoLand inspection covers 49 changed code/configuration files, including
  weak warnings, with no timeouts. Two intentional duplicate-test-setup reports
  are covered by existing exact DuplicatedCode exclusions. Machine-readable
  evidence: `.scratch/experimental-heic-opt-in/evidence/goland-inspections.json`.
- Sixteen Settings renders pass. English dark and German light with Cache were
  visually inspected at 640x520; all tabs and full explanatory text fit.

A third bounded read-only scout task inventoried analysis/retained-search test
names and source-entry paths while the lead finished verification. It reused
the existing scout (one spawned agent total; three bounded tasks across recon
and final evidence phases). Its source citations exposed the positive-pixel
coverage gap; the lead designed, wrote and verified the additional tests.
No review or fix was delegated. Review budget for ticket 03 increased from one
to two rounds to close that gap; native gate rerun is justified by its new
required guard. No complete broad race suite was repeated.

Outstanding before acceptance: native Linux and Windows on both architectures,
installed standard-user test-MSIX on both architectures, macOS Intel, complete
native Linux/amd64 suite and canonical shard check, fresh latest-commit CI,
post-suppression Qodana/CodeQL and Codex code/security reviews. Docker is stopped
locally. The branch still requires explicit authorization to commit/push under
AGENTS.md; no publication or review-loop invocation has occurred. Production GUI
smoke and release licensing/signing/camera/color gates remain separately open.

Final native capture: `make heic-native-macos` passes all six required guards,
including signed helper/runtime, signed update/reinstall, packaged application
and successful analysis pixels. The final application test took 5.120 s and the
analysis test 8.640 s; package totals: client 77.488 s, update 4.209 s,
similarity 10.627 s, UI 5.626 s. No mandatory guard skipped; the unrelated
Store-only policy test correctly skips in this non-Store suite. Raw events are
`.scratch/heic-qualification/native-macos.json`; command output is
`.scratch/experimental-heic-opt-in/evidence/native-macos-run.log`.
Final formatting, Qodana exclusion, provenance and native import checks pass.
The import check needed ordinary Go module-cache access after a sandboxed cache
write failed; rerun with that access passed without policy/source changes.

## PR review loop — f41fe63

Ronin committed/pushed the implementation as `f41fe630158baa1c65f8dc75122c332dcddd4a1d`
and resumed the requested CI/Codex loop on September 16. The working tree was
clean on entry. Fix commits, pushes, review requests/replies and dispositions
follow the repository's invoked review-loop authorization; no merge/release is
included. One read-only scout task located existing Linux CGo setup while the
lead investigated provenance. All fixes and reviews remain lead-owned.

CI 35103830559 passes both macOS native architectures, ordinary Windows, the
Store executable build, the non-UI race partition and UI shards 1/3. The remaining
failures are addressed in this first correction, pending a fresh run:

- Regenerate provenance after committed `SplitSeq` build-tool cleanup. The guest
  artifact is unchanged and both provenance/import checks pass locally.
- Keep Linux helper/analysis guards no-CGo, but enable CGo for the separately
  executed GUI inventory/fixture and install the existing X11/C build inputs.
  The runner regression failed for both GUI command modes before correction.
- The restart test selected another viewer's Settings surface. An owned prior
  window reproduces the exact failure; selecting only the window opened by the
  tested viewer fixes it. Focused UI race tests pass (2.947 s).
- The disposable standard account passes SID/nonadministrator checks, but
  known-folder lookup is denied. The fixture now imports its loaded user's
  environment with CreateEnvironmentBlock before native activation, replacing
  inherited privileged-runner profile values. No application fallback or
  permission-query relaxation is added; native CI must qualify the correction.
- Shorten the disposable MSIX identity to fit the manifest's 50-character limit.
- Qodana's final post-suppression SARIF has one confirmed unused test function;
  remove both platform variants now that OpenInstalled owns helper preparation.
  Start-report warnings are not the final result set.

Focused nativeguards/client tests pass, along with changed-file GoLand checks.
The three mixed-receiver IDE warnings in the runner are fixed. Two intentional
local test-setup duplication notices remain documented under their existing exact
Qodana exclusions; no production warning is suppressed. GoLand build reports
success but cannot collect detailed build messages in this project.

Docker is available again and its canonical shard check passes: 691 runnables,
three shards. The daemon is Linux/aarch64; the complete native AMD64 gate correctly
refuses emulation. Full race verification stays in hosted native CI.

## PR review loop — 085d185

CI 35105325751 passes validation, every Linux race partition (including the
previously failing Settings shard), ordinary Windows, the Store executable
build and both native macOS architectures. Both standard-user Windows native
event streams contain all required passes and no failed events, including the
permission query, AppContainer/job checks, concurrent staging, native application
activation and analysis. Their parent jobs report failure only because GitHub's
PowerShell wrapper propagates Robocopy's successful copy status of 1. An explicit
success exit after all checks and cleanup corrects that wrapper; exceptions still
fail before reaching it.

Both Linux native inventories expose another CGo consumer: similarity's test
fixtures import the desktop stubs. The runner regression first failed for its
inventory and execution commands, then passed after separating its complete
suite with CGo enabled. The helper/client/worker suites retain CGO_ENABLED=0.
The focused nativeguards package passes; Linux Docker compilation and the
analysis-owner regression pass with heicnative (0.403 s). This emulated check
does not qualify native seccomp or resource controls.

Both installed-MSIX jobs now pack, sign and install successfully for the actual
standard account, but IApplicationActivationManager returns 0x80070520 (no logon
session). Microsoft documents that a [packaged application runs as an interactive
user](https://learn.microsoft.com/en-us/windows/msix/desktop/desktop-to-uwp-debug).
The loaded alternate-user profile is not proof of an interactive Windows logon.
Installed activation remains a failed gate; no elevated/debug-token substitute
or successful skip is accepted. A fifth bounded read-only scout task checks
Microsoft's documented direct executable launch semantics while the lead owns
the fixture decision and other fixes; it reuses the existing scout and changes
no files.

Qodana's final post-suppression SARIF at 085d185 contains zero findings. Codex's
security report for that commit says no security issues; its code review is
still running. CodeQL's previous two results are existing dismissed false
positives, revalidated against the current exact-name archive admission and
checked integer conversion; no open alerts were returned. Current-head CodeQL
results and all final-head reviews must still be inspected before acceptance.

### Installed executable launch follow-up

The native installed-MSIX activation failure is the red evidence for a bounded
fixture correction. A [Microsoft Terminal maintainer's explanation](https://github.com/microsoft/terminal/discussions/20060)
establishes that ordinary CreateProcess on an installed WindowsApps executable
can resolve its package identity. It does not guarantee success in our
alternate-user session. The fixture now uses that ordinary launch path, leaving
the declared test application and every runtime package identity, exact user,
nonadministrator, Store policy, installed-byte/ACL and native-helper assertion
intact. No debug activation, access-control changes or fake identity is used.
Both native architectures must pass; a launch or permission failure still blocks
qualification. The existing child-script registration guard now names the test
entry point instead of the removed local COM wrapper class. No production code
changes.

The 085d185 Go CodeQL analysis could not be processed (HTTP 422, analysis
1786545678, rules_count 0, Unknown Error), despite its successful workflow. This
is unverified, not an empty clean analysis; a fresh run must produce a readable
result set. No repository self-hosted runners are configured.

### Code review disposition and native progress

Codex's code review of 085d185 reports one confirmed P2: startup permanently
captured a low file-size preference in the immutable HEIC owner. The native
application regression starts at 1 MiB and uses the owned image with a valid
free-space box to exceed that limit. Raising to 2 MiB failed with the original
1 MiB error before the fix. Startup now uses the fixed hard HEIC limits;
imaging.Reader already applies the current user limit on each new read.
Foreground and background reads pass both increase/decrease cases through the
same owner, and a 128 MiB setting still grants at most 64 MiB to input.
The signed native macOS application regression passes (6.913 s). Evidence:
`native-live-input-{red,green}.log` in the opt-in evidence directory. No new
top-level test, Qodana scope or shard entry is introduced. Changed-code GoLand
inspection is clear, including weak warnings.

At 9ea4dde, both Linux and standard-user Windows native jobs pass, as do all
Linux race partitions, ordinary Windows and validation. Qodana's final SARIF
again has zero findings. CodeQL Go processing recovered (analysis 1786683225):
34 rules and the same two existing dismissed false positives; Actions has 17
rules and zero results. Installed-MSIX qualification remains pending the
direct-launch fixture. The fresh code/security review must cover the final
fix commit after the P2 thread is answered and resolved.

The final 9ea4dde CI run (35107489388) completes with every job passing except
installed-MSIX activation on amd64/arm64, both still reporting 0x80070520 before
the direct-launch correction. Focused UI race regressions for the new fix pass
(5.872 s); formatting is clean. The correction and direct-launch fixture are
staged, but the configured SSH signer refused the commit on September 16 at
14:26 UTC (`commit.gpgsign=true`, agent refused operation). Signing remains
enabled. Publication, the P2 reply/resolution and fresh final-head reviews await
the user's signing-agent unlock; no new commit or native MSIX success is claimed.

## PR review loop — 4aca7be

Ronin unlocked the signer and authorized retry. The signed fix commit is pushed;
the confirmed P2 thread was answered with red/green evidence and resolved. Fresh
Codex code review reports no findings on 4aca7be; security review is running.
All six standalone native OS/architecture jobs pass, as do all four Linux race
partitions, ordinary Windows, validation and Store executable construction.
Final Qodana SARIF has zero findings. CodeQL Go analysis 1786844866 processes
successfully with the same two dismissed false positives; Actions has zero
results. Raw native artifacts are retained under the opt-in evidence directory.

Direct installed-MSIX launch still fails before the test process starts:
Access denied. The fixture had selected the protected WindowsApps package as
its current directory. One bounded follow-up starts from its already writable
owned workspace, retaining the absolute installed executable, real identity
and every other guard. This tests whether directory access is the remaining
launcher issue; it is not proof that the alternate-user session is sufficient.
Failed launch or permission checks remain blockers.

GitHub's separate dynamic AI code-scanning job fails before analysis with a
service error: HTTP 400, "The requested model is not supported." Its log is
`ai-scanning-4aca7be.log`; this is distinct from the passing CodeQL workflow
and requested Codex reviews. No scanner setting or gate is disabled.

## Qualification checkpoint — 69fef1a

CI run 35111412255 is complete. Every job passes except installed-MSIX activation
on both architectures. The direct launcher also returns Access denied when
started from the owned writable workspace, so the extra working-directory
permission demand was not the blocker. The installed test process never starts;
no package-context helper or sandbox success is claimed. The next qualification
requires native x64/ARM64 environments with interactive standard-user sessions.
All identity/permission checks and the failing gates remain intact.

Fresh Codex code review (comment 5699569800) and security review (5699579439)
report no findings on 69fef1a. Both prior review threads are resolved. Final
Qodana SARIF has zero findings. CodeQL Go analysis 1786920301 has the same two
previously assessed dismissed false positives; Actions analysis 1786878185 has
zero findings, and the open-alert query is empty. The separate dynamic AI scan
35111413724 fails before analysis with the same unsupported-model service error.

Evidence is under `.scratch/experimental-heic-opt-in/evidence/`: native artifacts
for all six platform/architecture combinations at 4aca7be, corresponding final
CI runs, `ci-msix-{amd64,arm64}-69fef1a.log`, `qodana-69fef1a.sarif.json`,
`codeql-69fef1a-go.sarif.json`, and `ai-scanning-69fef1a.log`. No broad local race
suite was duplicated. Local Docker remains aarch64; its canonical shard inventory
passes, while full AMD64 verification stays in native CI. Production GUI smoke,
licensing/distribution, production signing and broader camera/color gates remain
separate. This plan stays active because required installed-MSIX qualification
is incomplete; the draft PR is not ready for acceptance.

### Online investigation of alternate-user MSIX activation

At Ronin's request, stop speculative launcher changes and examine primary
sources. Microsoft's [WindowsAppSDK issue 2555](https://github.com/microsoft/WindowsAppSDK/issues/2555)
records the same `0x80070520` during packaged-component activation as another
user inside the desktop session owner's session. The maintainer specifically
identifies the [CreateProcessWithLogonW/MSIX interaction](https://github.com/microsoft/WindowsAppSDK/issues/2555#issuecomment-1190815856)
as an OS issue. Its closed-not-planned status and later OS escalation do not
establish a fix. This strongly matches our COM failure; the exact failed OS
check behind our direct-launch Access denied remains untraced.

The next qualification needs the standard test user to own the desktop session
and run the ordinary activation launcher from that session. Record both process
SID and desktop-session ownership before launching. A credential process with
a loaded profile does not establish this arrangement. Do not label GitHub ARM
runners categorically headless: the [runner maintainer's investigation](https://github.com/actions/runner-images/issues/14049#issuecomment-5217338266)
found an interactive desktop for the runner account. No supported workaround
for our secondary-account launch was found, and no further CI retry is justified
without changing that prerequisite. Native installed-MSIX proof remains open.

The detailed source assessment and next qualification procedure are recorded in
`.scratch/experimental-heic-opt-in/evidence/msix-activation-research.md`.
The current hosted-only provisioner must be split from session-local execution
before offering a manual or interactive-runner path. No implementation or gate
was changed by this research. Reused the existing read-only scout for one bounded
primary-source issue search (zero new spawns); lead verified cited comments with
`gh api`, retained all assessment/design ownership and wrote the record.

The unchanged Windows ARM64 standalone retry at `8aa317a` passes all five native
packages with no failed event records. Its original analysis-pixel failure is
retained as unexplained intermittent evidence, not reported as fixed. Fresh
Codex code/security reviews of that head are clean (comments 5699763980 and
5699899622); Qodana has zero final results and CodeQL only the two existing
dismissed false positives. Both installed-MSIX jobs and the separate unsupported-
model AI scanner remain unsuccessful.

### Resumed consumer and packaged-application evidence — 2026-09-16

Lead-owned slice B extends `images_test.go` at the approved startup/admission,
Grid, Spiral shader and Explorer Analyze seams. The two new root tests are
assigned to ui-2 (253 entries). `TestExperimentalHEICPreviewConsumers` establishes
uncached duplicate previews and actual Spiral shader pixels after ordinary
admission. `TestExperimentalHEICActiveAnalysis` establishes positive HEIC pixels,
saved disable while analysis remains current, source replacement/cancellation,
late-map refusal and continued HEIC use until restart. Native inherited analysis
and retained-search proof remains in the existing native suite.

Slice A now holds an actual foreground source read after native readiness,
changes the checkbox while that load is active, replaces its collection and
observes source closure plus retired-load completion. The fresh ordinary surface
survives, and the same owner decodes HEIC again before restart. It also opens a
mixed HEIC/HEIF/PNG directory and navigates each source. Each owned standalone
package is then relaunched with a missing helper, missing/invalid manifest,
wrong architecture or changed helper identity. All cases preserve intent and
the localized explanation and display the ordinary member of a mixed directory.
These fixture mutations never touch installed MSIX files.

On macOS a final owned negative helper is ad-hoc signed with an empty entitlement
plist using `codesign --force --sign - --options runtime --entitlements <empty.plist>
<owned HEICWorker.app>`. Its manifest pins those exact bytes; the enclosing app
is re-signed and checked with `codesign --verify --deep --strict --verbose=2`.
Actual readiness refusal must precede any bulk source read; the configured owner
and saved intent remain stable and ordinary PNG viewing recovers. Production
entitlements, code, dependencies and decoder bytes are unchanged.

Meaningful negative controls (all temporary edits restored): removing Grid and
Spiral reader injection fails both preview cases; closing Explorer on the saved
choice fails the active-session assertion; discarding startup errors fails every
invalid-package case; giving the negative helper its normal sandbox entitlements
fails the readiness-refusal assertion. Logs are `evidence/resume-*-negative.log`.
These are new coverage for already implemented behavior; no production defect
or runtime fix is claimed.

Local green: the T03 HEIC/injected-reader race gate, Explorer/preset package gate,
and client race gate pass. The client gate initially failed because the tool
sandbox prohibited its owned loopback binds; the identical unsandboxed command
passes (8.643 s), with both logs retained. The expanded real macOS arm64 packaged
fixture passes with race detection (25.637 s), including all six refusal cases.
Other native architectures and the complete suite require the fresh CI round.
Installed-MSIX/session prerequisites, Windows combined ACL/query/concurrent-app
failure evidence and the external AI scanner remain open. One read-only consumer
scout was used; all tests, negative controls, assessment and fixes stayed with
the lead. Final gates/inspection and new-head CI/review results follow below.

Final local gates: `make verify-build` passes, including formatting, exact Qodana
exclusions, reproducible guest/provenance, native import guard, vet and build.
Its first sandboxed attempt was blocked by Go-cache writes; the unchanged
unsandboxed command passed. `make check-test-shards` passes the canonical
Linux/amd64 inventory with 693 tests. Package/native-runner/plist tests pass.
`make security-govulncheck` reports zero reachable native vulnerabilities and
no guest vulnerabilities (one module-only native advisory remains unreachable).
GoLand inspections of both changed code files are clean, including warnings;
`resume-goland-inspections.json` retains the results. New test inventory is
retained in `resume-test-inventory.log`. The full race suite remains a CI gate
under the authorized review-loop procedure.

### Integrated checkpoint — c1b6890

[CI 35120395881](https://github.com/frathe/picfetch/actions/runs/35120395881)
passes validation, all four Linux race partitions, ordinary Windows, Store input
construction and all six standalone native platform/architecture targets on the
first attempt. Retained `resume-{linux,macos,windows}-{amd64,arm64}` artifacts
contain the mandatory package guards, with no skipped or failed application
scenarios. The prior unexplained ARM64 analysis failure did not recur; no fix is
claimed. Both installed-MSIX jobs still install successfully and then fail
`Process.Start` with Access denied before the test app produces evidence.

Fresh [code review](https://github.com/frathe/picfetch/pull/28#issuecomment-5700753505)
and [security review](https://github.com/frathe/picfetch/pull/28#issuecomment-5700821584)
report no findings for c1b6890. Both earlier review threads remain resolved.
Qodana run 35120395935 has zero post-suppression SARIF results. CodeQL run
35120395948 passes: Actions analysis 1787413575 has zero results, Go analysis
1787457184 has exactly the two existing dismissed false positives (fixed runtime
archive allowlist and bounded integer conversion), validated against unchanged
source; no open alerts. GitHub AI scan 35120401536 still fails before analysis
with HTTP 400, requested model unsupported. Its failed log and both new MSIX
logs are retained as `resume-*-failure.log`; neither failure counts as success.

48/57 ticket items are complete: 01/02/03/04/06/07 are resolved. Ticket 08 retains
combined application-level private-storage/ACL and failed-query recovery plus
concurrent application lifetime evidence; native navigation/source replacement
is now demonstrated. Ticket 05 needs the provisioner/executor split and native
x64/ARM64 desktops owned by the standard test account. No environment access was
supplied during this resume, and no such access was inferred. Ticket 09 remains
open for those Windows cases and complete required CI/external scan results.
Release, production signing and distribution/color qualification remain separate.

The documentation follow-up records this completed implementation checkpoint;
it changes no executable or test inputs. Its fresh PR checks/reviews remain
visible on PR #28. The local handoff and evidence directory preserve current
operational state without rewriting the historical native results above.
