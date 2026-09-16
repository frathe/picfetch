# Experimental HEIC opt-in implementation

Date: 2026-09-16. Route: Deep SDD/TDD. Status: in progress.

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

| Slice | Red / green / inspection evidence | Status |
| --- | --- | --- |
| Publication | Nine tickets published, ticket 01 claimed; prior spec preserved | Complete |
| 01 | Real-admission consumer red: unsupported scan never began sort; green imaging/filescan and direct/folder/sibling/session/Favorite UI cases | Implemented; final gate pending |
| 02 | Red: missing Experimental tab/default activation; green preference and real-checkbox separate-lifetime tests. Later red: stale open Settings; green live status update | Implemented; final gate pending |
| 03 | Red: active HEIC choices absent and saved rule rejected; green Explorer real-dialog save/rename and source consumers; native successful analysis and two retained preview queries pass, missing-owner negative control fails as expected | Implemented; final gate pending |
| 04 | Red: no staging publication; green owned bounded copy/reuse/repair/cancel/preparation-refusal cases. Windows cache/concurrent-process/lease tests pass natively on both architectures at 085d185 | Fresh final-commit qualification pending |
| 05, 08 | Standard-user standalone guards pass on both architectures at 085d185. Disposable signed MSIX installs, but activation fails with 0x80070520 in the alternate-user session | Installed-MSIX qualification blocked; wrapper exit correction pending |
| 06 | Both native macOS architectures pass signed application-constructor and analysis guards at 085d185; real HEIC decode, restart, cancellation and shutdown | Fresh final-commit qualification and production GUI smoke pending |
| 07 | Linux native target now requires application fixture in addition to helper/seccomp/resource suite | Native execution unverified |
| 09 | Focused package race suite and make verify-build pass. Initial IDE batch: 47 files; only three weak warnings (one redundant type fixed; two test duplication reports already excluded by exact Qodana paths). English/German light/dark, Cache/Store layout matrix rendered; German overflow seen red and default widened | Native matrix, complete CI and fresh reviews pending |

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
