# MA-033: captured launch side-effect policy

Status: ready-for-agent
Date: 2026-09-28
Source: `/to-spec ma-033`, synthesizing accepted decisions Q1-Q10.
Inspected revision: `a5caf73a9031723b42b0bfd13b1fbd25ac893be9`;
its tracked tree matches the interview baseline `5070d6d`.
Implementation status: not started; this publication authorizes no code changes,
commits, pushes, pull-request activity or releases.
Design: [accepted launch-policy design](../../docs/launch-policy.md)
Decision: [captured launch policy ADR](../../docs/adr/0007-captured-launch-policy.md)
Interview: [all ten accepted decisions](interview.md)
Language: [project glossary](../../CONTEXT.md)

## Problem Statement

An isolated trial should use its own settings and working data without touching
the ordinary installation's updater. Microsoft Store users should receive updates
through the Store. Ordinary portable users should retain their existing update
choices, installed-update recovery and saved preferences. These expectations must
hold during startup, manual actions and shutdown, including when a feature closes
or startup fails partway through.

The application currently repeats distribution/trial checks in the entry point,
viewer startup, update actions and shutdown. Several checks derive permissions
from live trial objects. The updater itself has no captured launch permission,
and viewer construction initially supplies ordinary storage configuration before
trial options replace it. Its constructor currently performs no stage I/O, so
that temporary configuration is not evidence of an existing storage leak.

Trial evidence-directory validation happens after Fyne app creation, and Location
Map validates after preferences/session loading and viewer construction. Settings
also offers update controls in trial sessions even though their actions are
refused. Existing helper tests do not prove the production entry point's full
ordering. This is an architectural correction with explicitly accepted startup
and presentation changes, not a claim that every current route is defective.

## Solution

Capture one validated, immutable launch policy before covered startup effects.
It selects application identity and ordinary/trial storage and determines whether
GitHub self-update effects are permitted. Root composition, the updater and
Settings consume the relevant decisions without reconstructing permissions from
feature objects. A missing policy refuses effects; an explicitly constructed
ordinary policy preserves normal behavior.

Separate passive policy from effectful preparation. Preparation validates trial
prerequisites and reserves a fresh evidence directory before Fyne storage or
features are opened. One owner retains trial resources around the UI run,
finalizing them after the UI joins its trial-evidence producers and also on
ordinary startup failures. Failed evidence remains available and retry requires
a new directory.

Distribution and trial purpose compose their restrictions. Settings keeps the
Updates tab and installed version, displays every applicable refusal reason,
and offers update controls only when permitted. Existing storage identities,
layouts, update verification, ordinary update behavior and trial network
prerequisites remain the compatibility contract.

## User Stories

1. As an ordinary user, I want my existing application identity and saved settings
   preserved, so that this refactoring does not make the app appear newly installed.
2. As an ordinary user, I want my saved session and launch-option overrides
   preserved, so that reopening and scripted launches retain their existing behavior.
3. As a user of an update relaunch, I want predecessor waiting and cleanup before
   preferences open, so that the previous process can finish its own persistence.
4. As a portable user who enables automatic checks, I want the existing daily,
   version and platform conditions respected, so that checks happen as expected.
5. As a portable user who leaves automatic checks off, I want no automatic
   network check, so that my saved preference continues to control that activity.
6. As a portable user choosing Check now, I want it to bypass automatic opt-in
   and the daily gate, so that I can explicitly request a check whenever needed.
7. As a portable user disabling automatic checks, I want active work cancelled
   while a completed stage survives, so that a finished download is not lost.
8. As a portable user closing normally, I want a valid staged update applied
   without relaunch, so that ordinary exit retains its existing meaning.
9. As a portable user choosing Perform update, I want validation before quit
   and relaunch intent, so that an unusable stage does not close the application.
10. As a user recovering from a failed installation, I want the last working
    backup protected before the failure record is consumed, so that recovery remains possible.
11. As a Microsoft Store user, I want automatic and manual GitHub updates refused,
    so that PicFetch does not replace a Store-managed installation itself.
12. As a Microsoft Store user, I want GitHub stage and predecessor artifacts
    left untouched, so that merely launching or quitting has no updater side effects.
13. As a Microsoft Store user, I want Settings to explain Store-managed updates,
    so that unavailable controls do not invite actions the application will reject.
14. As a Location Map trial operator, I want separate preferences and session
    storage, so that qualification does not change my ordinary browsing setup.
15. As an Explorer trial operator, I want the existing isolated identity retained,
    so that the application and its native launcher agree on the trial identity.
16. As a trial operator, I want Favorites routed to the trial's storage,
    so that trial collection work does not use my ordinary saved collections.
17. As a trial operator, I want Explorer presets routed to trial storage,
    so that qualification does not change my ordinary preset library.
18. As a trial operator, I want general analysis data routed to trial storage,
    so that the trial does not populate or clear the ordinary analysis cache.
19. As a trial operator, I want the isolated updater path selected from the start
    while update effects remain refused, so that construction never borrows ordinary staging.
20. As a trial operator, I want update checks, downloads, staging and apply refused
    even when requested directly, so that isolation survives every update entry point.
21. As a trial operator closing or replacing a view, I want the process to remain
    isolated, so that feature lifetime cannot silently enable normal-install effects.
22. As a Store trial operator, I want trial storage and both update restrictions
    honored, so that one launch characteristic cannot cancel the other.
23. As a caller supplying both trial flags, I want a launch error before app
    storage opens, so that an ambiguous request cannot select a mode silently.
24. As an Explorer trial operator, I want actual OS network denial verified before
    startup, so that unsupported or unisolated runs cannot masquerade as offline trials.
25. As a Location Map trial operator, I want its existing network behavior retained,
    so that trial storage isolation does not newly prevent map-tile use.
26. As a command-line user, I want help and rejected flags to exit without desktop
    startup or update effects, so that inspecting usage cannot change installation state.
27. As a worker-process caller, I want private worker dispatch to precede desktop
    preparation, so that a worker never opens app storage or reserves trial evidence.
28. As a macOS user, I want Open With installed before driver initialization,
    so that early file-open events continue to reach the viewer.
29. As a trial operator supplying an invalid or used evidence directory, I want
    refusal before Fyne storage or feature construction, so that failure happens at admission.
30. As an operator starting competing trials at the same path, I want exclusive
    reservation, so that only one attempt can claim the evidence directory.
31. As an operator whose trial setup partially fails, I want partial evidence kept,
    so that I can inspect the failure and retry at a fresh path.
32. As an operator whose startup fails before the UI runs, I want acquired files
    and recorder workers released, so that partial construction does not leak resources.
33. As an operator closing a trial normally, I want evidence producers joined
    before evidence closes, so that final publication is not lost or written after closure.
34. As an operator waiting for trial completion, I want recorder flush and worker
    completion observed explicitly, so that an empty work counter cannot imply completion.
35. As an operator encountering startup and cleanup errors, I want both observable
    with idempotent cleanup, so that the original failure is not hidden or cleanup repeated.
36. As a qualification analyst, I want incomplete attempts reported as incomplete,
    so that directory existence or partial records cannot be mistaken for success.
37. As a native-runner maintainer, I want existing trial paths, identities and
    evidence formats preserved, so that earlier preparation does not break collection tooling.
38. As a caller omitting launch policy, I want composition and covered effects
    refused, so that an initialization mistake cannot grant ordinary permissions.
39. As a caller requesting an ordinary launch with zero options, I want explicit
    construction to succeed, so that zero options remain distinct from missing policy.
40. As a caller supplying invalid launch input, I want no usable policy or covered
    effects, so that invalid input cannot yield a permissive partial result.
41. As a maintainer retaining captured policy, I want later option changes unable
    to alter it, so that startup facts remain stable for the process lifetime.
42. As a maintainer supporting multiple distributions, I want the build's
    distribution captured once, so that executable paths or UI objects cannot redefine it.
43. As a maintainer constructing multiple test viewers, I want independent launch
    decisions, so that one instance's trial settings cannot affect another.
44. As a maintainer composing features, I want their selected storage supplied
    before construction, so that a later retargeting step is not an isolation requirement.
45. As an ordinary user on a system with fallback storage, I want existing ordinary
    fallback behavior preserved, while failed trial setup never falls back to my normal data.
46. As a user in a restricted session, I want the installed version still visible
    and unavailable update controls absent, so that Settings accurately describes the session.
47. As a Store trial user, I want both applicable explanations visible,
    so that the UI does not hide one of the reasons self-update is unavailable.
48. As a user of any supported language, I want update explanations localized
    consistently, so that the new presentation has complete, renderable translations.
49. As an updater caller in a restricted launch, I want verifier preparation refused
    even with preconfigured dependencies, so that test or setup seams cannot bypass policy.
50. As an updater caller in a restricted launch, I want stage reads and cleanup
    refused, so that a no-network result cannot conceal prohibited filesystem access.
51. As a caller requesting a prohibited manual check, I want a terminal refusal
    without worker/progress activity, so that the action neither hangs nor appears successful.
52. As a caller requesting prohibited apply or relaunch, I want no stage access,
    binary mutation or quit, so that a direct call respects the same launch decision.
53. As a user in a restricted launch, I want update notification/failure records
    neither consumed nor modified, so that launch denial covers update metadata as well.
54. As a user whose last-check day is restored, I want that restoration distinct
    from recording a successful permitted check, so that startup does not manufacture update effects.
55. As a portable user, I want existing staging authentication and serialized
    transactions preserved, so that policy refactoring does not weaken update verification.
56. As a user cancelling an update or closing Settings, I want obsolete callbacks
    rejected while completed disk effects keep their meaning, so that late work cannot revive UI.
57. As a maintainer, I want tests to exercise the production startup and updater
    interfaces, so that passing helper tests cannot conceal incorrect composition.
58. As a maintainer qualifying releases, I want native evidence identify the actual
    platform and distribution, so that simulated policies are not mistaken for native support.
59. As a maintainer handing off implementation, I want complete test and inspection
    evidence, so that missing tests, skipped cases or incomplete scans cannot count as a pass.
60. As a maintainer extending launch modes, I want one policy decision and an
    explicit isolation limit, so that the change does not require scattered trial predicates
    or imply a general filesystem/network sandbox.

## Implementation Decisions

### Settled decisions: do not relitigate

| Decision | Contract |
| --- | --- |
| Q1 | Cover self-update effects and existing launch storage selection; other feature/network contracts retain their owners. |
| Q2 | Root composition and updater effect admission consume the same captured decision. |
| Q3 | Reject invalid/used trial evidence directories before Fyne preferences/session access and features. |
| Q4 | Require valid explicit policy; missing policy refuses effects; decisions remain immutable. |
| Q5 | Distribution and trial purpose are separate inputs whose restrictions compose. |
| Q6 | Retain failed evidence; preparation cleans up acquired resources and retry uses a fresh directory. |
| Q7 | Restricted Settings shows the version and accurate explanations in place of update controls. |
| Q8 | One preparation owner spans construction and the UI run; the UI borrows recorders. |
| Q9 | Preserve path-routing isolation and existing identities/layouts; external directory replacement is outside the guarantee. |
| Q10 | Store trial presentation shows both applicable refusal explanations. |

### Policy and consumer interfaces

- Extend the existing launch module with validated, read-only launch decisions
  and a distinct effectful preparation owner. Keep the entry point thin and
  reuse the existing identity validation and trial recorders.
- Capture distribution, trial purpose, identity and storage selection from one
  launch input. Expose update permission and applicable refusal reasons as
  value observations. The policy contains no live features, resource handles,
  effect callbacks, setters or caller-owned mutable option aliases.
- Valid construction from zero launch options produces an ordinary launch.
  An absent or invalid policy cannot be interpreted as that ordinary value.
  Composition rejects it before opening consumer storage or starting features;
  updater operations independently refuse covered effects.
- Pass the required value to root composition and updater construction, and a
  presentation decision to Settings. Tests use the same construction interface.
  Keep injection per call or per instance; no mutable package-level test switches.
- Root retains pre-app cleanup execution, feature composition, UI feedback,
  ordinary update preferences and shutdown order. Updater admission owns its
  covered effects. Low-level binary replacement, stage authentication and raw
  serialization do not become a global permission framework.

### Mode, identity and storage contract

After existing prerequisites succeed, ordinary portable launches permit GitHub
self-update subject to each operation's existing conditions. Store launches and
either trial refuse those effects. Store plus a trial retains both restrictions
and the trial's storage. Both trial flags together are invalid.

Preserve the normal application identity and each trial's existing identity
format, including Explorer's established namespace and the identity embedded by
its native runner. Resolve the trial root consistently with that identity from
captured input; do not introduce a new symlink or case-canonicalization policy.

Trial preferences/session data use the isolated app identity. Favorites, presets,
general analysis data and updater storage use their existing trial locations.
Supply this selection before feature construction, including analysis-cache and
Explorer consumers, rather than building against ordinary roots and retargeting
them. Ordinary app-cache locations may resolve from the correctly identified
Fyne app. Existing ordinary fallback paths remain valid; trial failures never
select those fallbacks. Shared model/runtime assets remain under their own policy.

Exclusive fresh-directory reservation establishes startup ownership; a prior
existence check alone is insufficient under competing launches. Preserve existing
directory-newness and error semantics. This is a storage-routing guarantee, not
ongoing protection from external replacement of the directory tree. Existing
Favorite ownership still governs its operations independently.

### Preparation, startup and completion

- Help and malformed flags return before desktop effects. Private workers
  dispatch before desktop preparation. Keep native Open With installation in its
  existing early desktop position, before Fyne/GLFW initialization.
- Validate policy prerequisites and reserve trial evidence before Fyne storage
  access or feature construction. Explorer still requires actual verified OS
  network denial; Location Map does not acquire that requirement. Windows
  Explorer refusal remains a platform limit, including for Store builds.
- Use validated policy for predecessor cleanup/wait before creating the Fyne
  app. Bind the app identity and selected consumer storage to that same policy.
  Preserve explicit construction/runtime order and ordinary launch overrides.
- Preparation retains resource ownership throughout startup and the UI run.
  The UI borrows recorders and stops/joins trial-evidence producers before
  returning; preparation then closes evidence and stops/joins recorder workers.
  Unrelated feature shutdown policies remain local, including the existing
  uninterruptible-read exception.
- Every ordinary error return after acquisition releases acquired resources.
  Cleanup is idempotent and finishes before process exit; a deferred cleanup
  skipped by immediate process exit does not satisfy this contract. Preserve
  the original error while also reporting finalization failures.
- Retain claimed directories and partial evidence. Failed preparation never
  recursively removes or silently reuses them. Reused evidence is refused
  without changing it. An incomplete prepared session remains incomplete under
  the existing evidence format; no new success marker or evidence schema is added.
- Reservation proves admission, not that all later writes will succeed. Retain
  existing recorder flush/error reporting and distinguish worker completion from
  UI application and from native trial qualification.

### Complete updater effect admission

The permission applies before verifier/client preparation, automatic/manual
worker admission, stale-stage inspection/removal, release requests, download,
staging, apply-intent validation, binary apply, relaunch and updater persistence.
A preconfigured client or stage never supplies launch authorization.

Include update-record access exposed to root: reading, saving or clearing the
What's New and apply-failure records must use a policy-bearing operation.
Raw serialization can remain an implementation detail, but cannot remain an
alternative application-facing route around admission. Cover last-check-day
persistence as well. Restoring an existing day may seed updater memory without
recording a new check or invoking prohibited update persistence; a successful
permitted check retains its existing persisted-day behavior. This distinction is
necessary because the current restore path calls a persistence-capable setter.
Normal settings/session persistence uses the selected namespace and may carry an
unchanged restored day; update-specific persistence callbacks require permission.

Prohibited automatic/startup/shutdown paths perform no covered work. Explicit
manual/apply requests report refusal through their established error/callback
protocols, with no progress, ready event, quit or relaunch. They do not leave
busy state, admitted workers or completion waiters stranded. Root continues to
marshal applicable asynchronous callbacks and reject obsolete delivery.

Preserve ordinary portable behavior: opt-in/daily/version/platform admission,
manual bypass, one serialized transaction, lazy verifier preparation, matching
same-process stage reuse and re-verification of unauthenticated persisted stages.
Turning automatic checks off cancels current work without discarding a completed
stage. Normal shutdown applies without relaunch; explicit Perform update records
relaunch intent only after usable-stage validation and then requests quit.
Keep existing failure/relaunch classification and stage retention behavior.

Startup must read the apply-failure record for the backup decision before any
reporter clears it. A failed restore or unreadable failure record retains the
backup. Restricted launches do not read those records, sweep the backup, consume
notification markers or apply a stage even when ordinary artifacts already exist.
Retain the same fixed policy after feature close/replacement and during shutdown.

### Settings and compatibility

Settings consumes a supplied permission/reasons value. Allowed portable sessions
retain their update controls and flow. Restricted sessions retain the Updates tab
and installed version, replace check/apply controls with explanations, and render
both reasons for a Store trial. Check actual mounted content, not an unattached
widget's visibility flag. Stale/direct actions still encounter effect admission.

Preserve existing Store copy and use the trial-session explanation. Every new
or changed app-visible string uses the existing localization mechanism and all
catalogues; existing font-safe text rules apply. CLI preparation errors retain
the launch module's established shell-facing reporting conventions.

No dependency or disk-format migration is needed. Keep existing update trust,
licensing/notice delivery, trial evidence formats and native-launcher behavior.
Exact code identifiers, signatures and task ordering belong to implementation
planning; this specification fixes behavior and ownership rather than file layout.

## Testing Decisions

### Agreed seams and observable behavior

The interview confirmed production startup, root/updater effects, trial resource
lifetime and Settings presentation as the testing interfaces, and the user then
invoked `/to-spec`. This specification carries those choices forward without
another interview.

1. **Launch policy/preparation interface:** test returned identity, routing,
   refusal reasons and explicit validity through the construction interface
   consumers use. Observe resource completion and retained evidence through
   existing recorder interfaces and temporary filesystem fixtures.
2. **Production startup orchestration:** add one high-level per-call injection
   seam where needed for native install, worker dispatch, preparation, app creation
   and the UI run. The production entry point must use that same orchestration.
   Record observable effect order and inject failures; a parallel test-only
   startup algorithm or assertions on a source-code string cannot prove behavior.
3. **Root and updater operations:** use the normal viewer harness, registered
   production lifecycle hooks and the updater's operation interface. Reuse
   per-instance clients, verifier/persistence/apply injection and temporary stage
   fixtures. Root routing and direct updater admission both need evidence.
4. **Settings content and actions:** use the existing Settings host harness and
   walk mounted window/tab content. Verify version/explanations, absent controls,
   real action outcomes and no unwanted host calls.
5. **Native qualification interface:** extend the existing native-guard runner
   with focused launch-policy coverage on Linux, Windows, macOS and a Windows
   Store-tagged build. Root-UI selection must avoid Linux-only goldens on other
   hosts. Test runner inventory and required child execution as well as the guards.

Prefer existing interfaces and real composition over new fine-grained seams.
Keep pure policy evaluation independent of a real desktop and verified-offline
prerequisites observable through controlled external operations. A simulated
Explorer/Store policy case proves composition of facts, not OS network denial.

Use literal expected matrix results, not expected values recomputed by policy.
Exercise the six distribution/purpose combinations, both-trial rejection, missing
policy, invalid preparation and explicitly constructed ordinary zero options.
Observe multiple independent instances and changes to caller options after capture.

Denied paths must prove absence of reads as well as writes. Record calls at real
external I/O interfaces and use temporary sentinel files for durable outcomes;
unchanged contents alone do not prove that data was never opened. Include stage,
record, verifier, persistence, apply, relaunch and app-construction observations.
Use a configured fake client and pre-existing stage to ensure dependency setup
cannot bypass policy. Tests never replace the running executable or touch the
real user's preferences, updater directories, Favorites or desktop integrations.

Hold producer/recorder work with channels and observe Stop/Wait/Close or existing
completion handles. Exercise partial preparation, failure before the UI run,
failure after handoff, normal shutdown and repeated cleanup. The registered
production lifecycle and post-run path must establish the result independently
of the harness's stronger cleanup. Do not sleep or infer callback completion
from an in-flight counter. Preserve queued callback staleness regressions.

### Existing prior art and known gaps

- `TestLaunchArgs_HelpPrintsUsageAndExits`, `TestLaunchArgs_BadFlagExitsTwo`,
  `TestLaunchArgs_GoodArgsKeepGoing` and `TestTrialLaunchPreservesPredecessorArtifacts`
  provide helper behavior and temporary cleanup artifacts, not full startup ordering.
- `TestApplicationID` and launch-parser trial cases cover identity and rejection;
  add the complete policy matrix and early preparation cases at the agreed interfaces.
- `TestUpdateCheck_StoreManagedBuildNeverTouchesGitHubStage`,
  `TestMicrosoftStoreUpdateActionsAreRefused`, Explorer's `trial_launch` and
  Location Map's `native_trial_update_isolation` provide real root routes.
  Construct their launch decisions explicitly instead of relying on mutable mode flags.
- `TestUpdater_EnsureClient_PreSetClientBypassesVerifierFactory`, verifier
  retry/idempotence cases, `TestUpdater_Start_RequiresPreparedClient` and
  `TestUpdater_AutomaticAndManualShareCompleteTransaction` retain allowed behavior;
  denial must be tested even with those dependencies already configured.
- `TestManualUpdateCheck_BypassesSettingAndDailyGate`,
  `TestUpdateCheck_TurningOffKeepsCompletedStage`,
  `TestPerformUpdate_RecordsRelaunchThenQuits` and
  `TestPerformUpdate_InvalidMissingOrSameStageDoesNotQuit` pin ordinary semantics.
- `TestSweepUpdateBackup_KeepsTheBackupAfterAFailedRestore` and
  `TestSweepUpdateBackup_KeepsTheBackupWhenTheRecordCannotBeRead` pin backup safety.
  Add actual startup sweep-before-record-consumption coverage.
- `TestUpdater_SetLastCheckDayCallsPersist`, update-record round trips and
  `TestApplyStagedUpdate_RecordsWhyTheSwapFailed` identify persistence effects;
  they do not establish launch-policy admission or read-only restoration.
- `TestCurrentUpdateCallback_DropsEventSupersededWhileQueued`,
  `TestManualUpdateCheck_CancelledRequestEmitsNoTerminalCallback` and
  `TestUpdateCallbacksAfterSettingsCloseAreIgnored` separate worker results from UI delivery.
- `TestUpdatesTab_MicrosoftStoreOwnsUpdates`,
  `TestUpdatesTab_ShowsCurrentVersionAndBuild` and existing tab-content walkers
  demonstrate mounted-content testing; extend them to both trials and combined reasons.
- `TestRecorderKeepsLatestStateAndRequiresNewDirectory`,
  `TestRecorderReportsWriteFailure`,
  `TestManualLaunchFailureAndExistingEvidence` and cancellation/deadline runner
  cases observe retained evidence and completion. The tagged macOS
  `TestNativeLibraryRunner` uses a controlled child; it does not run the viewer's
  production startup or prove real-model qualification.
- Worker packages dispatch through their own test entry points. The native Cocoa
  graft test calls installation directly. Neither proves placement within main.
  Current Windows/Store native suites omit root main, and Store omits root UI;
  their existing green results cannot qualify the new composition by themselves.

### Acceptance criteria and commands

These are future implementation gates, not tests run for this publication.
**New** names are proposed behavioral suites/subcases and native runner suites.
They do not exist yet. Planning may map them to suitable existing interfaces and
tests while preserving every scenario and updating this command map. Paths below
locate verification, not prescribed implementation files.

Before claiming a pass, enumerate selected top-level tests with
`go test -tags no_emoji,nodynamic -list '<anchored selection>' <package>` and compare
the result with the required inventory. Store variants include `microsoftstore`.
Retain uncached verbose/JSON output proving each required parent and child ran
and passed. Zero matches, parent-only success, skips or build failures are not
acceptance. Observe each guard failing for the intended behavioral violation
before counting its green result. Work in focused vertical TDD slices.

**AC1 — Composed permissions.** All six distribution/purpose combinations have
the accepted permission and reason set; both trials together are rejected.
Stories 11-12, 20, 22-23, 42.
Command (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunchPolicyContract$/^matrix$'`.

**AC2 — Explicit validity and immutable capture.** Missing/invalid policy refuses
effects, explicit ordinary zero options succeed, and later input/feature changes
or another instance cannot alter a captured decision. Stories 38-43, 60.
Commands (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunchPolicyContract$/^validity$'`;
`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^feature_lifetime$'`.

**AC3 — Identity and storage compatibility.** Preserve normal and both trial
identities, native-launcher identity agreement and every covered storage route,
including ordinary fallback behavior with no trial fallback. Stories 1-2, 14-19, 37, 44-45.
Command (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunchPolicyContract$/^(identity|storage)$'`.

**AC4 — Early exclusive trial admission.** Invalid, used and concurrently claimed
directories and failed Explorer offline prerequisites reject before Fyne
storage/features. Location Map does not invoke the Explorer offline probe.
One competing reservation succeeds; prior evidence remains unchanged. Stories 23-25, 29-31.
Commands (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunchPreparationContract$/^reservation$'`;
`go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$/^validation$'`.

**AC5 — Production entry ordering.** Help/bad flags and private workers avoid
desktop preparation; native install precedes app/driver creation; admitted
predecessor cleanup/wait precedes preferences. Stories 3, 26-28, 57.
Command (new): `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$/^(early_exit|ordering)$'`.

**AC6 — Preparation resource cleanup.** Partial acquisition and startup/run
errors release acquired resources exactly once, retain partial evidence, require
a fresh retry path and preserve original plus cleanup errors. Stories 31-32, 35-36.
Commands (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunchPreparationContract$/^resources$'`;
`go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$/^prepared_cleanup$'`.

**AC7 — Construction consumes selected storage.** Real root composition receives
valid policy and correct identity/roots before consumer construction; no trial
consumer temporarily uses normal roots. Missing policy rejects before effects.
Stories 14-19, 38, 44-45, 57.
Command (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^construction$'`.

**AC8 — Direct updater refusal.** Prohibited verifier preparation, stale-stage
access/removal, automatic/manual admission, apply intent and apply/relaunch cause
no covered I/O, worker admission, quit or false success, even with preconfigured
clients/stages. Stories 20, 49-52.
Command (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/autoupdate -run '^TestUpdaterLaunchPolicy$/^(admission|preconfigured)$'`.

**AC9 — Update metadata admission.** Restricted update-record reads/saves/clears
and update-specific last-check persistence cause no I/O; restoring state does
not record a new check; successful permitted checks retain persistence. Stories 53-54.
Commands (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/autoupdate -run '^TestUpdaterLaunchPolicy$/^(records|persistence)$'`;
`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^update_records$'`.

**AC10 — Root update paths and fixed lifetime.** Automatic/manual actions,
preference toggles, stale/direct actions and feature closure consume the same
restriction; no path re-enables updates or touches ordinary stages. Stories 11-12, 20-22, 51-53.
Command (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^(update_entrypoints|feature_lifetime)$'`.

**AC11 — Ordinary check/cancellation compatibility.** Preserve automatic gating,
manual bypass, successful-day recording, cancellation, completed-stage retention
and stale callback rejection. Stories 4-7, 54, 56.
Command (existing regressions): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestUpdateCheck_|TestManualUpdateCheck_|TestCurrentUpdateCallback_)'`.

**AC12 — Ordinary stage/apply compatibility.** Preserve transaction serialization,
authentication, stage reuse/retention, failure records, ordinary no-relaunch apply
and explicit validated relaunch/quit. Stories 8-10, 55.
Commands (existing regressions): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/autoupdate -run '^(TestUpdater_|TestApplyStagedUpdate_)'`;
`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestPerformUpdate_|TestApplyStagedUpdate_)'`.

**AC13 — Startup cleanup and recovery ordering.** Both cleanup paths obey policy;
ordinary failed-restore/unreadable records protect backups before reporting can
consume them, and restricted launches do not consume markers. Stories 3, 10, 12, 53.
Commands (new): `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestLaunchStartupContract$/^ordering$'`;
`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^backup_order$'`;
existing `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestSweepUpdateBackup_'`.

**AC14 — Production shutdown and evidence completion.** Actual production hooks
and post-run finalization refuse restricted apply, preserve ordinary apply, join
trial-evidence producers before closing records, observe recorder flush/error and
release resources before exit without broadening unrelated joins. Stories 8-9, 20, 33-36.
Commands (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchPolicyIntegration$/^shutdown$'`;
`go test -tags no_emoji,nodynamic -count=1 -v ./internal/launch -run '^TestLaunchPreparationContract$/^evidence$'`.

**AC15 — Mounted Settings and localization.** Allowed controls work; restricted
sessions mount the installed version and exactly the applicable explanation(s),
with update controls absent. Direct/stale actions still refuse. Catalogues remain
complete and renderable. Stories 13, 22, 46-48, 56.
Commands (new): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/settingswin -run '^TestUpdatesTabLaunchPolicy$'`;
existing `go test -tags no_emoji,nodynamic -count=1 -v . -run '^TestTranslations_'`;
`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/settingswin -run '^(TestUpdatesTab_.*|TestUpdateCallbacksAfterSettingsCloseAreIgnored)$'`.

**AC16 — Ordinary launch options.** Preserve zero-option behavior, overridden
preference restoration and one-shot Picture-frame mode after loading/cancellation.
Stories 1-2, 26, 39.
Command (existing): `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLaunchOptions_'`.

**AC17 — Actual distribution selection.** Production captures the compiled
distribution correctly in ordinary and Store-tagged builds, and the production
startup route preserves that decision instead of inferring it from a path.
Stories 11-12, 22, 42, 58.
Commands: `go test -tags no_emoji,nodynamic -count=1 -v ./internal/distribution -run '^TestStoreManaged_'`;
`go test -tags no_emoji,nodynamic,microsoftstore -count=1 -v ./internal/distribution -run '^TestStoreManaged_'`;
new `go test -tags no_emoji,nodynamic,microsoftstore -count=1 -v . -run '^TestLaunchStartupContract$'`.
Tag-selected tests on another OS are not native Store qualification.

**AC18 — Trial evidence/tool compatibility.** Preserve recorder formats,
fresh-directory rejection, partial/cancelled evidence, launcher identity and
normal finalization; earlier reservation does not become a false ready signal.
Stories 15, 25, 30-37.
Commands: `go test -tags no_emoji,nodynamic -count=1 -v ./internal/explorertrial ./internal/locationtrial`;
`go test -tags no_emoji,nodynamic -count=1 -v ./scripts/locationmapqualify -run '^TestManual'`;
on native macOS, `go test -tags no_emoji,nodynamic,explorertrial -count=1 -v ./scripts/explorereval -run '^TestNativeLibraryRunner$'`.
These existing fixture tests complement, and do not replace, AC4-AC7 and AC14.
Explorer's recorder package currently has no tests; its package invocation is
only a build check until behavior coverage is added at the agreed interfaces.

**AC19 — Native coverage inventory.** Focused native suites select production
startup, policy/preparation, root/updater and Settings guards, require named
children, set Store tags when appropriate, and reject missing/skipped guards
without selecting unrelated platform goldens. Stories 57-59.
Command (new): `go test -tags no_emoji,nodynamic -count=1 -v ./scripts/nativeguards -run '^TestLaunchPolicyNativeSuite$'`.

**AC20 — Native execution.** Run the finalized focused suites on native Linux,
Windows and both supported macOS architectures, plus native Windows with the
Store tag. Preserve native Open With and predecessor behavior. Verify Explorer's
real prerequisite/refusal without asserting Windows Explorer support. Stories 3, 24-28, 37, 58-59.
Proposed new runner commands, after creating a dedicated evidence directory:
`go run ./scripts/nativeguards -suite launch-policy -capture .scratch/ma-033/evidence/native-launch-policy.json`;
on Windows, `go run ./scripts/nativeguards -suite launch-policy-store -capture .scratch/ma-033/evidence/native-launch-policy-store.json`.
Retain separate artifacts per platform/revision. These new suite names require
implementation; runner fixtures alone do not satisfy native execution.

**AC21 — Ownership convergence.** All covered production callers consume the
captured decision; policy-free effect routes and feature-derived update predicates
are removed. Preserve feature-local trial behavior, low-level updater mechanics
and storage formats. Stories 41-45, 49-54, 60.
Command for lead assessment: `git diff a5caf73a9031723b42b0bfd13b1fbd25ac893be9 -- main.go internal/launch internal/ui internal/distribution scripts/nativeguards`;
record the covered caller inventory and dispositions alongside AC1-AC20 evidence.
A grep count or green helper test alone cannot establish convergence.

**AC22 — Final repository gates.** Formatting/generated inputs/notices, vet,
build, exact test exclusions, UI shard assignments and the full race suite pass;
all changed code has complete GoLand inspection evidence including weak warnings.
Story 59.
Commands: `make check-test-shards`; `make check-qodana-test-exclusions`; `make verify`.
Retain separate analyzed-revision/file-scope/tool/finding dispositions and native
evidence. Inspect fresh post-suppression Qodana SARIF when CI is run; unavailable,
incomplete or licensing-failed analysis is unverified, not a pass.

Use focused checks while iterating and the full Make gate at handoff on supported
native Linux/amd64 Docker or qualifying CI. Never weaken worker isolation for
emulation. Add exact Qodana exclusions for new test files and assign every new
root-UI top-level test to the shard manifest. Preserve source-local justified
suppressions and update the package map if package ownership changes.

The native suite names above are proposed focused additions, following the
existing command-admission/Favorite qualification pattern; implementation planning
can retain equivalent existing suites if they enforce the same complete inventory.
Keep inherited-PID, native Open With and trial prerequisite checks in scope without
launching model-quality or Location Map latency requalification. Existing full
native platform suites may supply unchanged underlying-operation evidence with
the actual tested revision stated. A Store-tagged test is not a claim about an
installed Store package or new Store/platform trial support.

## Out of Scope

- Reopening Q1-Q10, adding launch modes, migrating application identities or
  changing existing storage/evidence formats.
- A general network/filesystem permission framework, filesystem sandbox or
  adversarial protection against external trial-tree replacement.
- Moving source-image edits, map-tile access, model/runtime asset setup or
  existing Favorite ownership under launch policy.
- Changing update trust, attestation, authenticated-stage reuse, native binary
  replacement/recovery algorithms, distribution publishing or release policy.
- Broadening unrelated feature shutdown waits, including blocked source reads,
  or changing request/completion contracts established by earlier refactorings.
- Reworking command admission, browsing ownership, collection transitions,
  Favorites, feature registration or rendering architecture.
- New dependencies, a package registry, global mutable launch configuration,
  dependency upgrades or unrelated code cleanup.
- Location Map latency, model quality/performance, codec or new-platform
  qualification beyond the changed startup/policy paths.
- Implementation tickets, implementation, commits, pushes, PR review loops,
  merging or release actions during this specification publication.

## Further Notes

This is the local issue tracker's published specification with `ready-for-agent`
status. The accepted design and ADR remain the decision record; this document is
the behavior/story/test contract for subsequent planning. No additional user
interview or testing-seam approval is needed for the already accepted interfaces.

Implementation should take the Deep SDD route because it crosses startup/UI
modules and requires native Windows/Store qualification. The lead owns interface
decisions, review and fixes; any bounded delegation follows the repository's
working agreement. Do not confuse publication readiness with implementation
authorization or completed qualification.

The honest limit is unchanged: storage routing does not secure paths against
external replacement, reservation cannot guarantee later disk writes, and trial
policy cannot prove OS network denial or human/native performance acceptance.
Resource cleanup covers ordinary error returns and orderly termination, not an
uncatchable process kill. Preserve the application's existing evidence/error
conventions for incomplete attempts.

No code or runtime test was changed or executed to publish this specification.
New test/suite names are requirements, not reported results. Documentation
validation passed for structure, story/criterion coverage, links, command syntax
and existing test-name selections, separately from the future implementation gates.
