# MA-033: captured launch policy implementation

Status: active, tickets 01 (`ccf4702`), 02 (`5ea960f`), 03 (`de5cb5c`), 04 (`8528f7b`), 05 (`7a5629b`), 06 (`d8f84c0`) and 07 complete; 08 active.
Baseline: `9549e3b` (approved specification), runtime baseline `a5caf73`.
Route: Deep SDD with vertical TDD slices.
Authorization: 2026-09-28 `/implement MA-033`: implementation, per-ticket commits,
pushes, draft PR and CI, then ready status and GitHub Codex review loop after
all tickets complete. No merge or release. This supersedes publication-only
authorization notes in the specification and ticket index.

Deliverable: one explicit immutable launch decision governs existing storage
selection and all application self-update effects, with one prepared trial
resource owner spanning startup and UI completion.

## Specification and limits

The [specification](../.scratch/ma-033/spec.md),
[ticket graph](../.scratch/ma-033/issues/README.md) and
[accepted design](../docs/launch-policy.md) are authoritative. Q1-Q10 and the
five testing seams are already approved. Each ticket's V commands and the
parent's AC1-AC22 commands are the acceptance oracles; required child execution
must be retained, with uncached output and actual revisions. A build or empty
selection is not behavioral evidence.

No new dependencies, disk formats, launch modes, general sandbox, update trust
changes or unrelated shutdown joins. Path routing does not protect against
external tree replacement; reservation cannot promise future writes. Native
platform qualification remains distinct from simulated policy and fixture tests.

## Contracts and task graph

`01 -> 02 -> 03 -> 04 -> 08 -> 09`
`          -> 05 -> 06 -> 08`
`          -> 07 ------> 08`

Lead owns interfaces, production composition, all review, post-review fixes,
user-visible text and final verification. Separate branches may proceed only
with stable contracts and disjoint file ownership. At most two active subagents.
This harness maps T1 to GPT-6 Sol and T2/T3 to GPT-6 Luna; the working agreement's
model names describe roles rather than unavailable literal models.

### 01 — Observable production startup

Owner: T0 inline.
Files: `main.go`, new `main_startup.go`, `main_startup_test.go`, Qodana exclusion,
architecture locator, ticket/evidence/todos.
Depends: none.
Contract: `runStartup(args []string, stdout, stderr io.Writer, ops startupOps)
(int, error)` is shared with `main`; per-call operations dispatch workers,
install native handlers, resolve identity, create app and run UI. Existing
`launchArgs`, cleanup permission and trial reservation timing remain unchanged.
Test: help/bad flags, each worker short circuit, native-install/app order,
predecessor completion before preferences, ordinary options/path forwarding,
identity/app/run errors. Existing launch option regressions remain green.
Verify: ticket 01 V1-V3; enumerate root and UI selections first.
Budget: 0 implementation spawns, 1 lead review, no complete local race suite.

### 02 — Immutable captured decisions

Owner: T0 for startup/composition; eligible T1 for a bounded pure policy slice.
Files: `internal/launch` policy/tests; root startup/tests; UI build/startup,
viewer and harness/tests; distribution guards and bookkeeping.
Depends: 01.
Contract: explicit `launch.NewPolicy(opts, normalID, storeManaged)` returns
`(Policy, error)`; private fields distinguish absent from explicit ordinary.
Value accessors provide identity, purpose, captured trial root and update reasons.
Storage selection is passive; prerequisite probing stays outside the value.
Root production captures `distribution.StoreManaged` once. Composition requires
Policy before persisted state access; tests use the same explicit constructor.
Test/Verify: ticket 02 V1-V5 (six literal matrix cases, identity/storage,
missing/invalid policy, immutability/instance independence, compiled distribution).
Budget: <=1 bounded spawn, 1 lead review, no full suite.

### 03 — Prepared trial lifetime

Owner: T0 (cross-package lifetime).
Files: launch preparation/tests, trial recorders as needed, startup, UI run and
trial wiring, root tests and harness bookkeeping.
Depends: 02.
Contract: `launch.Prepare(ctx, policy)` returns one `*Prepared` owner; passive
Policy contains no handles. Prepared lends existing recorder types and exposes
idempotent `Close() error`. Per-call environmental seams remain at acquisition
boundaries. Production returns original and cleanup errors using `errors.Join`.
Test/Verify: ticket 03 V1-V4 (exclusive reservation, prerequisite distinction,
partial acquisition, held producer completion, flush failures, retained evidence).
Budget: 1 read-only recon spawn (shared with 04), 1 lead review, no full suite.

### 04 — Construction-time storage

Owner: T0 (shared root composition).
Files: launch storage/tests, UI construction/features/run/trial/harness tests.
Depends: 03.
Contract: resolve ordinary fallback roots only for ordinary policy; supply one
complete selected storage value before constructing updater, Explorer and
analysis-cache; Favorites receives its selected root before runtime admission.
Test/Verify: ticket 04 V1-V4, observed consumer construction and denied storage
access, ordinary saved state and launch overrides.
Budget: 0 additional spawns, 1 lead review, no full suite.

### 05 — Check and stage admission

Owner: T0 root wiring; eligible T1 bounded updater implementation.
Files: autoupdate updater/tests, root update handlers/preferences/build/harness.
Depends: 02.
Contract: `autoupdate.New` requires captured Policy; missing/restricted policy
refuses before external effects even with configured dependencies. Restore day
is distinct from persist-day effects. Existing request/callback protocols remain.
Test/Verify: ticket 05 V1-V4 including forbidden reads and configured stages.
Budget: <=1 bounded spawn if G1-G5 pass, 1 lead review, no full suite.

### 06 — Recovery, records and installation

Owner: T0 (hot updater context and root lifecycle).
Files: updater/record interfaces and tests, root startup/update/shutdown tests.
Depends: 05.
Contract: application records become policy-admitted updater operations; raw
serialization cannot remain an application bypass. Backup decision precedes
report consumption. Apply/relaunch intent and shutdown obey captured policy.
Test/Verify: ticket 06 V1-V4, real registered hooks and observable forbidden I/O.
Budget: 0 spawns, 1 lead review, no full suite.

### 07 — Settings explanation

Owner: T0 (all app-visible strings).
Files: Settings state/Updates/tests, root snapshot/wiring, every translation.
Depends: 02.
Contract: supplied update decision carries permission and all refusal reasons;
Settings renders it without feature-based permission inference.
Test/Verify: ticket 07 V1-V3, six mounted-content/action cases and catalog parity.
Budget: 0 spawns, 1 lead review, no full suite.

### 08 — Caller convergence and native guards

Owner: T0 caller assessment; eligible T1 runner-only change once inventory fixed.
Files: nativeguards runner/tests, native workflow wiring, remaining covered
callers/tests and architecture documentation.
Depends: 04, 06, 07.
Contract: `launch-policy` and `launch-policy-store` suites require exact named
parents/children and record host/build/revision; missing/skipped cases fail.
Test/Verify: ticket 08 V1-V3 and full production-caller inventory.
Budget: <=1 bounded spawn, 1 lead review, no full suite.

### 09 — Integrated qualification

Owner: T0 only.
Files: evidence/plan/todos/tickets, qualification wiring if necessary.
Depends: 08.
Contract: reconcile all 22 ACs against actual final code and native evidence.
Test/Verify: ticket 09 V1-V5; final Make gates, native Linux/Windows/both macOS
architectures and Windows Store, IDE inspections plus fresh Qodana SARIF.
User-authorized GitHub CI supplies full race suite (do not duplicate locally
during review workflow). Keep PR draft until every ticket is complete. Then
request fresh code/security review, disposition every finding, and repeat until
latest commit reviews and required CI are clean. Do not merge.
Budget: 0 spawns, complete suite once via CI and rerun only affected gates.

## Delegation gate and evidence

Recon 03/04: G1 yes (one bounded inventory); G2 yes (named `rg` inventory
cross-check); G3 yes (read-only); G4 yes (broad trial ownership exploration kept
out of lead's startup context); G5 yes (lead has not traced those lifetimes).
S: cross-file lifetime conclusions require comprehension. W: no implementation
provided. T3 Luna scout only; lead retains design/review. Future delegated tasks
must record G1-G5 before spawning.

## Working checklist

- [x] Frame: deliverable, Deep route and non-goals.
- [x] Spec: accepted decisions and executable criterion map.
- [ ] Recon: close each ticket's concrete implementation questions before edits.
- [x] Plan: task graph, contracts, owners and budgets.
- [x] Delegation gate recorded for initial scout.
- [ ] Red/green: record behavioral failure and passing command per slice.
- [ ] Lead review: formatting, vet, criteria, negative guards, diff; fixes inline.
- [ ] Final gate: generated inputs/notices, vet/build, shards, full race and native.
- [ ] Land: per-ticket completion, commits, inspections and final review evidence.

## Execution ledger

| Ticket | Spawns budget/actual | Review rounds | Full suite | Status |
| --- | --- | --- | --- | --- |
| 01 | 0/0 | 1 | no | done |
| 02 | 1/1 | 1 | no | done |
| 03/04 recon | 2/2 | n/a | no | complete; acquisition/producer/storage inventory |
| 03 | 1/1 | 1 | no | done |
| 04 | 0/0 | 1 | no | done |
| 05 | 1/1 | 1 | no | done |
| 06 | 0/0 | 1 | no | done |
| 07 | 0/0 | 1 | no | done |
| 08 | 1/1 | 0 | no | runner slice active; convergence pending 07 |
| 09 | 0/0 | 0 | CI | pending |

## Verification evidence

Initial checkout is clean on `feature/ma-033-launch-policy` at `9549e3b`.
GitHub reports no open PR on this branch. Local inspections use the available
GoLand per-file engine including weak warnings, following the documented
fallback; no automated IDE-local Qodana action is exposed. CI Qodana remains a
separate required post-suppression SARIF gate.

### Ticket 01

Analyzed/tested revision: `9549e3b` plus this ticket's complete patch, recorded
by the ticket-01 commit. New startup parent has eight required leaf cases:
help, malformed flags, private HEIC, private similarity, native/predecessor/run
ordering, identity failure, app failure and run failure. All ran and passed;
existing artifact/argument tests and all ten launch-option tests also passed.
Uncached root/UI/Store outputs: `.scratch/ma-033/evidence/01-*-green.log`.
Initial red missed usage/errors and desktop work. A deliberate native-install
reordering then failed the order assertion in ordinary and Store builds. The
Store check overlapped that mutation; it was rerun after restoration and passed.

Formatting, Qodana exact test exclusion and root tagged vet passed. GoLand
`get_file_problems(errorsOnly=false)` completed on all three changed Go files
with no remaining findings or timeouts. The test's compiled distribution branch
has a narrow `GoBoolExpressions` suppression because both build tags must run
the same guard. Reinspection passed. No feature storage/lifetime behavior changed.

### Ticket 02 precise policy contract and delegation

`NewPolicy(Options, normalID string, storeManaged bool) (Policy, error)` captures
only value facts. Accessors: `Valid`, `ApplicationID`, `Purpose`, `TrialDir`,
`StoreManaged`, `Updates`; purposes `Ordinary`, `ExplorerTrial`, `LocationMapTrial`.
`UpdatePermission.Allowed()` and `Reasons() []UpdateReason` return independent
values; reasons `MissingPolicy`, `StoreManagedUpdates`, `TrialUpdates` in that
order when applicable. No policy contains caller option pointers or resources.
`ResolveStorage(ordinary Storage) (Storage, error)` returns ordinary fallbacks
unchanged, or captured-root children favorites/presets/image-analysis/updates.
Storage fields: `FavoritesDir`, `PresetsDir`, `AnalysisDir`, `UpdatesDir`.
`ErrInvalidPolicy` identifies absent/invalid policy. Identity retains existing
Explorer namespace and Location digest; absolute/clean path resolution introduces
no symlink/case canonicalization or offline probe.

T1 Sol owns only new `internal/launch/policy.go` and `policy_test.go`.
G1 yes (bounded contract); G2 yes (ticket 02 V1); G3 yes (two disjoint files);
G4 yes (pure policy context is smaller than root composition); G5 yes (lead has
defined interface but has not implemented it). S requires behavior/test judgment;
W contract only. Lead owns all cross-package integration, negative verification,
inspection, review and fixes. No subagent commit.

### Ticket 02 evidence

Analyzed/tested revision: `ccf4702` plus the complete ticket-02 patch recorded
by its commit. Draft PR: https://github.com/frathe/picfetch/pull/72.
All V1-V5 selections ran and passed, independently executed by the lead, with
raw local output in `02-policy-green.log`, `02-ui-green.log`,
`02-startup-green.log`, `02-store-green.log`. Root/UI tests add absent-policy
refusal before storage/acquisition, immutable independent capture, compiled
distribution/identity propagation and restricted predecessor admission. Existing
launch options, startup defaults/geometry/session restoration stayed green.

Behavioral reds retained in tool transcript: UI cache opened before missing
policy refusal; Run tried trial acquisition; startup forwarded Policy{}; removal
of the viewer policy initializer failed four new capture tests, then restoration
passed. Delegated policy reds (transcript, not raw files): matrix saw invalid
Policy{}; validity accepted empty identity and dual trials; identity returned
an empty app ID; storage returned ErrInvalidPolicy for explicit ordinary roots.
Lead inspected implementation and independently verified all green results.

Inspection scope: all 25 changed Go files, GoLand per-file errorsOnly=false;
complete, no timeouts. Detailed JSON is retained under local evidence. Fixture
duplicates and mosaic explicit types match existing exact exclusions. Two
unchanged viewer duplicate fragments match the ignored local
`.scratch/ma-028/linux-reset-repair-2026-09-27/mutant-viewer.go`, not another
production implementation; no source suppression or unrelated refactor needed.
No new actionable findings. Scoped vet passed. Shard check reports 740 runnable
tests across three shards. Format and exact exclusion checks passed after
formatting the final test callback. Actual native qualification remains open.

### Ticket 03 acquisition contract and routing refinement

The recon closed the ownership questions: Explorer reserves via exclusive
Mkdir/OpenFile and closes idempotently; Location reserves via Mkdir and owns one
Stop/Wait recorder worker. Existing `waitForShutdown` already joins Explorer and
Location Map producers after their Stop calls. Preparation will reuse both.

Precise API: `Prepare(ctx context.Context, policy Policy, options PreparationOptions)
(*Prepared, error)`. Per-call options expose only external acquisition operations:
`VerifyOffline func(context.Context) error`,
`NewExplorer func(string) (*explorertrial.Session,error)`,
`NewLocation func(string) (*locationtrial.Recorder,error)`; nil uses production
operations. Default options are not a default policy. Prepared exposes `Policy`,
`ExplorerTrial`, `LocationMapTrial`, and idempotent `Close() error`, with nil-safe
observations. Failed partial acquisition closes nonnil returned resources and
joins acquisition/finalization errors. Context cancellation also closes acquired
resources. Retain directories and original evidence formats.

Lead owns main/UI borrowing and production shutdown proof. One T1 Sol task owns
only new preparation.go/preparation_test.go within internal/launch; G1 exact
bounded contract, G2 ticket03 V1, G3 two disjoint files, G4 acquisition-only
context, G5 unimplemented acquisition loop. S/W: comprehension and resource
tests, contract only. This raises ticket03's implementation-spawn budget from
zero to one to overlap independent module work with lead-owned lifecycle wiring;
the earlier T3 scout remains separately recorded. No review or fixes delegated.

### Ticket 03 completion evidence

Analyzed/tested tree: `5ea960f` plus this ticket's named changes (recorded by the
ticket 03 commit). Launch acquisition, main owner lifetime and UI borrowing are
implemented. Only the existing evidence-producer waits are retained; no unrelated
shutdown join was added. The production hook now leaves Location's recorder open
until the post-run worker join and entry-point owner close.

Behavioral reds: root app/run/normal-return tests lost Explorer cleanup errors or
returned before Location flush; held Location metadata demonstrated recorder stop
before producer join. Green V1 runs reservation/resources/evidence including
exclusive competing paths, cancellation, partial-acquisition joined errors,
stable cleanup, flush failures and actual default prerequisite refusal. V2 runs
all four required startup parents. V3 runs named held Explorer and Location Map
production-hook/post-run children, also under `-race`. V4 runs recorder tests and
all seven manual-launcher fixtures; Explorer recorder is build-only there, with
its resource behavior supplied by V1/V3. Root/launch/UI contracts also pass with
`microsoft_store`; native Windows execution is still ticket 09, not inferred.
Logs: `.scratch/ma-033/evidence/03-*-{red,green}.log` (local raw evidence).

Lead assessment: acquisition fails closed before app construction, resource-free
policy stays separate, error joining preserves original failures, evidence
formats/incomplete meaning remain unchanged, signals still perform orderly stop.
All ten changed Go files inspected through GoLand with weak warnings and no
timeouts. Two intentional error-identity comparisons now have narrow
`GoDirectComparisonOfErrors` suppressions explaining that memoization, not
wrapping equivalence, is under test; affected files reinspected clean. Scoped vet
and formatting/exclusion/shard gates run before landing. No dependency changes.

A second bounded T3 storage scout mapped current constructor/default accesses and
the test harness's late overrides for ticket 04 while the lead verified 03. This
expands the shared recon budget to two; it wrote no files and made no review or
design decisions. The lead retains the resulting cross-package storage work.

### Ticket 04 construction contract

Keep `buildStartupViewer` as the shared boundary, adding a per-call ordinary-root
resolver `func(fyne.App) (launch.Storage, error)`. Only explicit ordinary policy
calls it; trials select captured roots without consulting ordinary fallbacks.
Production supplies existing defaults, with Favorites' fallible fallback first
and general analysis derived from the already identified app's cache. Tests supply
temporary ordinary roots before construction instead of retargeting afterward.
The selected value enters `startupState` and is supplied directly to updater,
Explorer presets, Favorites and analysis-cache. Favorites' constructor captures
the root without starting reads; runtime refresh starts only after queues and
complete composition are installed. Runtime no longer accepts another root.
Remove path retargeting from launch overrides. Capture pending trial behavior from
policy, preserving all unrelated flags. `analysiscache.Options()` mirrors Explorer's
existing value accessor, letting its existing per-instance provider/queue seam
preserve and observe constructor inputs rather than replace roots in the harness.
Lead implements/reviews; no further delegation is useful with this hot context.

### Ticket 04 completion evidence

Analyzed/tested tree: `de5cb5c` plus this ticket's named changes. Construction now
selects roots once before persistence/consumers; ordinary resolution failure
returns before app Cache/Preferences, and trials never call the ordinary resolver.
Favorites captures without I/O; its runtime refresh starts after full composition.
The harness supplies temporary ordinary roots before construction and preserves
analysis-cache/Explorer constructor inputs while replacing only external adapters.
Existing Explorer launch fixtures now construct their trial policy and resource
first, rather than trying to create a new launch purpose through mutable flags.

Red: all six construction combinations exposed empty/default consumer roots;
denied ordinary storage still reached app cache; later trial flags retargeted
storage. Green V1 identity/storage; V2 all construction/lifetime children including
ordinary fallback and the correctly identified app's cache, no fallback calls for
trials, all four consumers and actual analysis-cache worker roots; V3 startup
validation/cleanup; V4 ten launch-option tests and seven manual-runner fixtures.
Additional saved-settings, runtime-start, trial-recording regressions and complete
Favorites/analysis-cache package tests pass. Store-tagged construction/lifetime and
Explorer trial-launch/truncation pass. Raw logs are `04-*-green.log`; initial
construction red is `04-construction-red.log` in local evidence.

Lead assessment: no production late retarget remains, cached launch roots survive
feature close and altered options, ordinary defaults and Favorite ownership are
unchanged. All 16 changed Go files inspected with weak warnings, no timeouts.
Only nine existing duplicate fragments in autoupdate_test.go and
favorites/favorites_test.go remain, covered by their existing exact Qodana paths;
no source suppression or test refactor needed. Scoped vet, format, exact exclusions
and shard check pass (740 runnable UI tests). No new dependencies or test files.

### Ticket 05 updater contract and delegation gate

`autoupdate.New(app, dir, policy launch.Policy, persist)` requires an explicit
policy, retaining its immutable `UpdatePermission` privately. A zero policy
creates a refusing updater, never ordinary permission. `EnsureClient`, `Start`,
`StartManual`, `RemoveStaleStage` and `SetLastCheckDay` refuse before effects or
worker/completion admission. Restricted manual requests deliver only Failed for
a current request, synchronously without a worker. A wrapped exported
`ErrUpdatesUnavailable` carries the existing localized trial-session message.
`RestoreLastCheckDay` seeds memory without persistence for any policy; existing
`SetLastCheckDay` remains the permitted check-result write/persist operation.
Per-instance private loadStage/removeStage adapters default to existing update
functions and let tests observe forbidden stage reads as well as mutations.
They do not change low-level authentication or formats. Apply/record operations
retain existing guards until ticket 06 completes their policy adoption.

One bounded T1 Sol implementer owns only `internal/ui/autoupdate` for this
check/stage slice, including migration of that package's existing New calls and
new TestUpdaterLaunchPolicy admission/preconfigured/persistence children. Lead
owns all root files, UI tests and bookkeeping. G1 fixed API/effects; G2 exact V1/V4;
G3 one package, no shared root edits; G4 bounded operation contract; G5 lead has
not implemented the updater slice. S/W: behavioral tests/admission decisions need
comprehension, not regex; no implementation body is prescribed. Lead alone reviews
and fixes findings. No record/apply redesign or new dependencies delegated.

### Ticket 05 completion evidence

Analyzed/tested tree: `8528f7b` plus this ticket's changes. T1 delivered the bounded
updater slice and ordinary fixture migration; the lead reviewed and fixed all
findings inline, expanded direct manual refusal to all six denied cases, and
integrated root admission/restore behavior. Store check refusal reuses the existing
localized Store message, with no catalogue/key changes. Apply/refailure/What's New
guards remain unchanged until 06. The root harness now creates the temporary
ordinary child directories before construction, retaining its old existing-dir
fixture guarantee for tests that directly write a stage; trials keep selected paths.

Red: direct EnsureClient admitted all six denied cases; configured requests leaked.
Root Explorer/Location checks reached HTTP, removed a seeded stale binary and
reported work/success; restored last-check days invoked persistence on every
construction. Corrected an initial fixture assertion to compare actual archive
bytes, not its release-notes string, and retained the subsequent behavioral red.
Green V1-V4 cover direct/root admission, all restrictions plus absent policy,
observable stage reads/removals, verifier/HTTP/persistence calls, no completion or
busy state on refusal, current/stale/cancelled manual callbacks, pure restore and
successful ordinary persistence. Existing daily/version/opt-in and serialized
manual/automatic transaction, stage authentication/reuse, disable/cancel and queued
callback regressions pass. Store-tagged V1/V2 and focused root/updater race guards
pass. Raw output: local `05-*-green.log`, with updater/root red logs retained.

All 11 changed Go files inspected with GoLand weak warnings and no timeouts. Fixed
the new helper's missing Ordinary/default switch cases; final reinspection is
clean except seven deliberate existing fixture duplicates in three exactly excluded
test files. Scoped vet, formatting, exact Qodana test exclusions and the unchanged
740-test shard manifest pass. Added the new updater test's exact exclusion. No new
dependencies. Complete CI and fresh post-suppression SARIF are still final gates.

### Ticket 06 recovery/installation contract

Lead-only with the existing updater/root context. Convert the six exported record
functions to policy-bearing Updater methods (Save/Load/Clear WhatsNew and
ApplyFailure); retain only private generic JSON serialization. Refusal returns
the same wrapped permission error before app.Cache access, including reads.
RequestApplyAndRelaunch and ApplyStagedUpdate gate before transaction/stage access;
reuse the existing per-instance stage adapters there, preserving validation,
failure retention and installation mechanics. Root PerformUpdate and notifications
consume captured policy, including stale record inputs and closed/replaced trials.

Extract registration of the existing OnStarted callback into `registerStartup`,
used unchanged by Run and tests. A per-viewer executable resolver, initialized to
os.Executable, lets that actual hook exercise temporary backup paths without
touching the test executable. Preserve failure-read -> sweep -> reporter order,
and the registered OnStopped apply point. Explicit ordinary test record fixtures
replace raw public helper use; no application-facing bypass remains. No new
shutdown waits, dependency, disk schema or trust algorithm.

### Ticket 06 completion evidence

Analyzed revision: `7a5629b` plus the ticket 06 changes in this commit.
Lead implemented and reviewed; no delegation or changed dependencies.

- V1/V2 red: denied direct apply read/removed a stage; all six denied/missing
  policies accessed every update record operation. Root startup/direct recovery
  resolved the executable and displayed stale failure records in restricted
  sessions. Each failed for the intended effect, then passed with captured guards.
- V1-V4 green: full updater policy family and existing updater/apply regressions;
  root integration and PerformUpdate/apply/current-callback/backup regressions;
  root startup ordering. Registered hooks retain ordinary false/explicit true
  relaunch and preserve staged binaries across all denied combinations, after
  feature close/replacement. Record formats and installation algorithms unchanged.
- Negative mutations: reporting before sweep failed restore retention and copy
  ordering; removing the real shutdown apply failed both ordinary hook cases.
  Mutations restored and the selected suites rerun green.
- All new root/updater policy guards also passed with `microsoftstore`, and with
  `-race` on Linux/amd64. Focused vet, imports/format, diff whitespace, exact
  Qodana exclusions and 740-test root shard assignment pass. CI on `7a5629b`
  reported CodeQL, Qodana, Windows/macOS guards and completed race shards green;
  this is not final-head or post-suppression SARIF qualification.
- GoLand `get_file_problems(errorsOnly=false)` completed all 11 changed Go files,
  Project Default fallback per the local inspection guide. Seven weak duplicate
  fragments only: five pre-existing updater/root test setups covered by exact
  exclusions; two unchanged viewer presentation fragments also matching retained
  `.scratch/ma-028` mutation sources. No actionable changed-code findings, no
  timeouts. The viewer fragments were inspected and left outside this scope.
- Local raw captures: `.scratch/ma-033/evidence/06-{updater-red,root-red,
  backup-order-negative,shutdown-negative,updater-green,regressions-green}.log`
  and `06-inspections.json`. No new test files or top-level root tests.

### Ticket 07 contract / ticket 08 runner delegation

Settings Show accepts `launch.UpdatePermission` alongside its preference snapshot,
replacing the Store boolean. It renders Allowed controls or each supplied reason;
Store copy remains unchanged, Trial uses the existing localized "Updates are
unavailable in this session". Missing policy also fails closed with that message.
Mounted tests use literal names `ordinary_portable`, `ordinary_store`,
`explorer_portable`, `explorer_store`, `location_map_portable`, `location_map_store`
under `TestUpdatesTabLaunchPolicy`; additional `missing_policy` is defensive.
Root passes its captured permission. Lead owns UI text, tests and review.

08 runner slice (T1 Sol, <=2 existing files): add launch-policy and
launch-policy-store to scripts/nativeguards main.go/main_test.go. The former is
native Linux/Windows/macOS; the latter Windows-only, tags microsoftstore. Require
the six approved guard families and all their current named children, compiled
distribution guards and existing applicable native prerequisite/Open With/
predecessor guards from the spec. No unrelated full-package goldens or codecs.
Capture host/arch/Go/build tags/revision and outcomes; retain strict missing,
skipped, failed-process and incorrect build-selection refusal.
Oracle: `go test -tags no_emoji,nodynamic -count=1 -v ./scripts/nativeguards -run
'^TestLaunchPolicyNativeSuite$'`, followed by the complete runner package.
G1 yes bounded two-file runner; G2 yes fixture command; G3 yes exclusive files;
G4 yes isolated runner context; G5 yes lead has not designed implementation.
S/W: evidence validation and fixtures need comprehension, not a regex; contract
only, not code. No workflow, UI, docs, review or commits delegated. Lead reviews
and commits with 08 after Settings/caller convergence. Budget 1 spawn, 1 review.

### Ticket 07 completion evidence

Analyzed revision: `d8f84c0` plus ticket 07 changes in this commit. Lead-only UI
implementation and review; ticket 08 runner files are independent, not included.

- V1 red: portable trials and absent policy mounted update controls and sent a
  manual host request; Store trials omitted their second explanation. V1 green:
  all six combinations and missing policy show exact mounted version/reasons,
  denied controls are absent and send no host actions; ordinary toggle/check/
  Perform actions and closed-window stale callback refusal pass.
- V2 full Settings package passed, including existing Updates-tab and close
  regressions. V3 translation parity/English identity and UI Unicode-arrow guards
  passed. Existing Store and trial keys already exist in both shipped bundles.
- Focused root policy integration and vet passed; Settings matrix also passed
  with microsoftstore tags and focused race. All five changed Go files completed
  GoLand Project Default fallback inspection including weak warnings, with zero
  findings/timeouts. No new test files or root top-level tests; existing exact
  exclusions and shard inventory remain valid.
- Raw red/green output and full inspection JSON are retained locally in
  `.scratch/ma-033/evidence/07-settings-{red,green}.log` and `07-inspections.json`.
