# MA-033: captured launch policy implementation

Status: active, tickets 01-02 complete; ticket 03 is the frontier.
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
| 03/04 recon | 1/1 | n/a | no | complete; acquisition/producer/storage inventory |
| 03 | 0/0 | 0 | no | pending |
| 04 | 0/0 | 0 | no | pending |
| 05 | 1/0 | 0 | no | pending |
| 06 | 0/0 | 0 | no | pending |
| 07 | 0/0 | 0 | no | pending |
| 08 | 1/0 | 0 | no | pending |
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
