# Microsoft Store update tickets

Source: [Approved Microsoft Store updates](spec.md).
Tickets 02–05 are implemented and locally verified. Ticket 01 has verified environment metadata but needs the approved CI read check; ticket 06 has the workflow code, with activation and
first live publication still pending.

These are complete, verifiable slices of the Store release flow. The parent
spec includes the final user requirement for a reviewer on every rollout. No standalone prefactoring
ticket is needed: the first code slice introduces the publishing command at the
existing repository-tooling boundary.

## Breakdown and dependencies

| Ticket | Blocked by | What it delivers |
| --- | --- | --- |
| [01: Configure Store access and prove read-only connectivity](issues/01-configure-store-access.md) | None | Configured automation identity and a read-only product check from CI. |
| [02: Preview a validated release with generated Store notes](issues/02-preview-validated-store-release.md) | None | Validated candidate, generated notes and metadata preview without Store mutation. |
| [03: Submit one validated update and report its Store state](issues/03-submit-validated-update.md) | 02 | One complete authenticated upload and certification submission through the command. |
| [04: Recover interrupted submissions without creating duplicates](issues/04-recover-interrupted-submissions.md) | 03 | Recovery across interruptions and reruns without duplicate or guessed mutations. |
| [05: Track publication and approve the next waiting release](issues/05-reconcile-publication-and-waiting-releases.md) | 04 | Approved receipt reconciliation, then separate approval for the newest waiting release. |
| [06: Enable approved CI updates and verify the first publication](issues/06-enable-unattended-store-release-workflow.md) | 01, 05 | Prepared, approved tag-to-Store CI flow, final verification and first live publication evidence. |

**Remaining frontier:** 01 and live activation/evidence in 06. The code path is 02 -> 03 -> 04 -> 05;
06 needs both completed account setup (01) and completed reconciliation (05).
Ticket 03 deliberately does not depend on live credentials: fake services make
its complete command path testable. Account provisioning must not hold up the
offline implementation.

Dependency independence is not permission for conflicting edits. The lead records
the command contract, client/authentication choices, receipt/snapshot shape and
durable-storage/retention decisions in the required SDD plan before parallel
implementation would rely on them. The lead owns architecture, review, fixes
and the final gate under the repository's working agreement.

## Scope and activation

- Carry forward the spec's proposed stable-tag trigger, existing-note source,
  English notes across current listing locales and newest-waiting-version policy.
  The final user amendment requires approval of each prepared release.
- Account setup is an operational slice; preview and submission are complete
  command paths; recovery and reconciliation each add a demonstrated release
  scenario. Tests and diagnostics are included with the behavior they establish.
- Automatic production triggers are connected only in 06 after the complete
  flow is implemented. Each rollout requires REDACTED_REVIEWER approval of its exact artifact and frozen notes.
- The next normal stable release supplies live publication evidence. No ticket
  authorizes manufacturing a release or resubmitting already-live 1.0.2 to test.

## Coverage

| Spec criterion | Owning tickets |
| --- | --- |
| AC1: candidate admission | 02; workflow enforcement in 06 |
| AC2: generated notes | 02; skipped-release integration in 05 |
| AC3: bundle submission and metadata | 03 |
| AC4: interrupted-run recovery | 04 |
| AC5: competing and waiting releases | 05; manual approved workflow in 06 |
| AC6: accurate states, failures and retries | 03, 04, 05 |
| AC7: read-only preview and connectivity | 02, 03; actual account evidence in 01 and 06 |
| AC8: workflow boundaries | 06 |
| AC9: regression coverage | 02, 03 and final integration in 06 |
| AC10: complete repository gate | 06 |
| AC11: immutable approval and no replacement selection | 02, 03, 04, 06 |
| Live access and eventual Published state | 01 and 06 |

User stories US1-US23 are mapped in each ticket. Later tickets extend the same
command-level test groups rather than creating parallel testing seams.

## Verification and completion

Each acceptance criterion carries its verification command or live-evidence
procedure. Placeholder run identifiers refer to actual future CI runs. Named
publisher commands and tests are implementation requirements, not existing or
already-passing checks. A command with no matching test scenarios does not prove
acceptance; retain evidence that each required scenario executed.

Use focused tests while implementing each slice, and negatively verify guards.
Register new test files with Qodana and new packages in the architecture map.
Run the complete `make verify` gate once after integration in 06. Remote
environment rules, CI credential access and Store publication require actual
remote evidence; fake-service tests do not establish those facts.

Append progress and evidence under each ticket's Comments. Keep unresolved
external setup or live-release criteria open and report the specific missing
evidence. Do not close or change the parent spec as a side effect of ticket work.

## Creation checks

Ticket creation changes documentation only. Validate local links, dependency
ordering and acyclicity, story/acceptance coverage, ticket status, unchecked
criteria and unchanged parent-spec content. No application tests, live API
calls, release command or git commit are required to publish these local tickets.

## Implementation evidence (2026-09-08)

- Focused release-tooling tests and publisher race tests pass. The Windows command
  cross-build succeeds. Both workflows parse as YAML.
- Eight deliberately broken guards failed their assertions and were restored;
  focused green reruns passed. The combined `make verify` run recorded a Docker OOM event; all UI race shards and all non-UI race-test packages passed across the isolated reruns. The original combined command did not exit successfully.
- The actual accepted 1.0.2 bundle from run `33983485123` passes the new recorder;
  its outer timestamp version is recorded separately from inner app versions.
- [Setup wizard](configure-access.sh), [record of the approved environment settings](environment.json),
  [main-only branch rule](main-branch.json). The wizard does not create environments.
- Current environment metadata is verified: main only, REDACTED_REVIEWER reviewer, self-review
  allowed, bypass/timer/custom rules disabled; all three secret names exist. The
  user confirmed the linked Developer application and rotated key.
- The new approval workflow is not on main. Microsoft read access and submission
  permission remain unverified. Landing the reviewed workflow and approving its
  read-only check is the next live validation step. No credential values were read.
- Cron was removed. Microsoft completes certification independently; manual
  approved check/reconcile observes it. A waiting release needs a separate submit
  dispatch and approval. No release, tag, Store draft, live protection change or
  git commit was created by this local amendment.
