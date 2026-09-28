# MA-033 implementation tickets

Status: ready-for-agent
Approved: 2026-09-28, after the user accepted the nine-ticket breakdown.
Parent: [captured launch side-effect policy specification](../spec.md)
Design: [accepted decisions](../../../docs/launch-policy.md)
ADR: [captured permissions and prepared resources](../../../docs/adr/0007-captured-launch-policy.md)
Baseline: `a5caf73a9031723b42b0bfd13b1fbd25ac893be9`.

Implementation began on 2026-09-28 under `/implement MA-033`, authorizing
per-ticket commits, pushes, a draft PR/CI and a review loop after all tickets.
The [active Deep SDD plan](../../../plans/2026-09-28-ma-033-launch-policy.md)
records interfaces, routing and evidence. Tickets 01-05 are complete; 06-09 remain
pending. All ten design decisions remain settled. No merge/release is authorized.

## Ticket graph and frontier

| Ticket | Blocked by | Delivers |
| --- | --- | --- |
| [01: Make production startup testable](01-production-startup.md) | None | Observable production ordering with existing behavior preserved. |
| [02: Capture immutable launch decisions](02-captured-launch-decisions.md) | 01 | Fixed identity/permission decisions and policy-based predecessor cleanup. |
| [03: Prepare and finalize isolated trials](03-prepared-trial-lifetime.md) | 02 | Early exclusive admission and one resource owner across the UI run. |
| [04: Select storage before constructing consumers](04-storage-at-construction.md) | 03 | Correct consumer roots from construction, without trial fallback. |
| [05: Enforce policy for checks and staging](05-checks-and-staging.md) | 02 | Root/direct check admission, staging and last-check persistence. |
| [06: Enforce policy for recovery and installation](06-recovery-and-installation.md) | 05 | Guarded records, backup recovery, apply and relaunch. |
| [07: Explain update restrictions in Settings](07-settings-explanations.md) | 02 | Mounted version/reasons and localized, accurately available controls. |
| [08: Complete policy adoption and native guard coverage](08-convergence-and-native-guards.md) | 04, 06, 07 | No remaining application bypasses and a strict native guard inventory. |
| [09: Qualify the integrated implementation](09-integrated-qualification.md) | 08 | Native and repository evidence for the integrated result. |

Initial frontier: **01 only**. After 02, the preparation, updater and Settings
branches are logically independent. That does not authorize concurrent edits
to shared root-UI files or contracts still in flux. Apply the repository's
delegation limits; the lead owns cross-package design, review and fixes.

## Acceptance-criterion ownership

The numbers below reference the parent's exact behavior and command map.
An owner completes its stated portion; 08 checks convergence and 09 verifies the
integrated result. No partial ticket result is a claim that a whole parent
criterion passed.

| Parent criterion | Implementation owner(s) | Boundary |
| --- | --- | --- |
| AC1 | 02 | Six policy combinations and dual-trial rejection. |
| AC2 | 02, 05, 06, 08 | Capture/composition validity; direct effects and feature-lifetime proof. |
| AC3 | 02, 04 | Identity/routing decisions, then real consumers. |
| AC4 | 03 | Early prerequisites and exclusive evidence reservation. |
| AC5 | 01, 02, 03 | Existing order, captured cleanup permission, then early trial admission. |
| AC6 | 03 | Partial acquisition and startup/run/finalization errors. |
| AC7 | 02, 04 | Missing-policy refusal and construction-time storage. |
| AC8 | 05, 06 | Check/stage admission; apply/relaunch admission. |
| AC9 | 05, 06 | Last-check restoration/persistence; update-record admission. |
| AC10 | 05, 06, 08 | Root entry points, fixed lifetime and final caller convergence. |
| AC11 | 05 | Ordinary checks, cancellation and callback compatibility. |
| AC12 | 05, 06 | Serialized staging, authentication and installation compatibility. |
| AC13 | 02, 06 | Predecessor cleanup; startup backup and record ordering. |
| AC14 | 03, 06 | Evidence-producer/resource completion; shutdown apply. |
| AC15 | 07 | Mounted Settings, actions and localization. |
| AC16 | 01, 02, 04 | Existing launch overrides and ordinary storage behavior. |
| AC17 | 02 | Actual compiled distribution captured in production. |
| AC18 | 03, 04, 09 | Trial/resource/identity compatibility and actual native runner execution. |
| AC19 | 08 | Focused runner inventory and missing/skipped-child refusal. |
| AC20 | 09 | Required native platforms, architectures and Store build. |
| AC21 | 08 | Lead's complete production-caller assessment. |
| AC22 | 09 | Final repository gates and complete inspection evidence. |

## Execution and evidence rules

- Plan the implementation using the parent's Deep SDD route. These tickets
  define outcomes, not a replacement for the implementation plan's exact
  interfaces, file ownership, routing and budget. Keep each completed slice green.
- Prefactor first. Do not introduce a temporary implicit ordinary policy or
  weaken existing refusal paths to stage the migration. Unmigrated operations
  retain their existing guards until their owning ticket replaces them.
- Every ticket includes focused behavioral tests and verification commands.
  Commands marked **new** name future guards, not tests that already exist.
  Each V-number below a ticket's checklist identifies an executable verification
  command for those criteria. Package names locate tests, not required code layout.
- Establish a behavioral red before green. List selected tests, retain uncached
  output proving required children ran, and never count an empty selection,
  skipped case or build-only package as behavioral evidence. Record exact
  tested revisions and any later invalidation of evidence.
- Use per-call/per-instance environmental seams, temporary files and offline
  clients. Observe forbidden reads as well as writes. Never exercise real user
  storage, executable replacement or desktop integrations in automated tests.
- Keep exact test exclusions, root-UI shard assignments, translations and
  package-map changes current in the ticket that introduces them. Inspect changed
  code including weak warnings; 09 consolidates evidence rather than deferring
  all testing and inspection until the end.
- Parent commands remain authoritative. If a proposed suite name changes during
  planning, preserve all cases and update the command mapping through an explicit
  follow-up; do not call an unmatched old command a pass.
- Native prerequisites and supported-host execution are separate from fake
  policy matrices. Missing hosts, unavailable inspections and incomplete scans
  remain unverified. Do not weaken isolation or broaden trial platform support.

## Publication record

Nine individual tickets were published with explicit blocking edges and
`ready-for-agent` status. The index maps all 22 parent acceptance criteria;
the parent maps all 60 user stories. No runtime tests or implementation gates
were run for ticket publication. Documentation validation passed: nine tickets
with the approved acyclic graph, 41 criteria linked to verification steps,
all 22 parent criteria assigned, 53 local links/anchors resolved, and 46 concrete
commands parsed by both bash and zsh. All referenced Go packages and Make targets
exist; 17 existing filtered test selections match declared tests. These are
static checks, not implementation evidence.

Only this effort's Markdown tracker artifacts are included in the authorized
documentation commit. The repository-wide scratch ignore rule is unchanged;
future raw evidence remains local unless separately selected for publication.
