# MA-032: request lifetimes and disposable result delivery

Status: extraction accepted after the sorting/SVG pilot; convergence in progress.
Date: 2026-09-28
Source: `/grill-with-docs ma-032`
Implementation inspected at `4c759b2`; unchanged in the relevant paths at `68477a9`.

This is the consolidated design for [MA-032](../needs_refactoring.md#ma-032).
The user accepted the recommended answers in three rounds and invoked
`/to-spec` to proceed with the consolidated design. The
[local interview](../.scratch/ma-032/interview.md) retains the questions and
source observations. The [local specification](../.scratch/ma-032/spec.md) is
published as `ready-for-agent`, with 60 user stories and 20 acceptance criteria
covering the conditional extraction and fallback. The user approved
[nine implementation tickets](../.scratch/ma-032/issues/README.md) and requested
a feature branch and documentation commit. On 2026-09-28 the user authorized
implementation, per-ticket commits, a draft PR/CI and the subsequent review
loop. The [Deep SDD record](../plans/2026-09-28-ma-032-request-lifetimes.md)
tracks selected tickets and revision-bound evidence. Local specification and
tickets remain gitignored.

## Purpose and completion

Pilot one instance-owned request module with root sorting and display SVG
rendering. It must remove repeated caller responsibilities for request validity
and release during disposable final-result delivery. Moving identical code
alone does not meet the agreed acceptance bar.

If the pilot succeeds, migrate the remaining basic request-token uses in
`internal/ui` and `internal/ui/display` and remove their duplicated request
lifecycle implementations. Keep feature-specific completion rules. If it
fails, retain local implementations and deliver shared contract tests and the
documented lifetime rules, with evidence explaining why extraction was rejected.
Both are legitimate outcomes of MA-032.

The proposed implementation location is a small Fyne-independent
`internal/requestlife` package using standard-library contexts and a
caller-supplied dispatcher. The specification fixes interface behavior; exact
exported names belong to implementation planning. No new dependency, native
integration or disk format is needed.

## Resolved decisions

| Decision | Contract |
| --- | --- |
| D1 / Q1: scope | Two consumers, root and display. Shared worker scheduling, admission tracking and joining require a separate decision. |
| D2 / Q2: acceptance | Show a concrete reduction in caller responsibilities in both pilot paths, with one common tested contract and no feature-specific switches. Otherwise use the local-implementation fallback. |
| D3 / Q3: compatibility | Preserve existing completion and production-shutdown policies. Assess any confirmed defect separately before expanding scope. |
| D4 / Q4: delivery | Evaluate a narrow final-delivery operation for disposable results using existing UI dispatchers. Features retain additional identity checks and completion effects. Retained loads, retries and committed-write reconciliation keep their own protocols. |
| D5 / Q5: parent | Begin captures the caller's context once, preserving its values, deadline and cancellation. Release affects only that request. Features capture HEIC capability before Begin; no new application-wide cancellation hierarchy is introduced. |
| D6 / Q6: admission | Features own terminal Stop and admission. Request invalidation permits later Begin for close/reopen. The shared module does not own visibility, feature stopping or shutdown order. |
| D7 / Q7: queued cancellation | Cancel the context without waiting for UI. A queued callback later skips a stale result and still performs its own final-delivery finishing once on UI. Cancellation alone does not report delivery. Transfer release ownership before dispatch, including inline dispatch; old finalization cannot affect a reentrant newer request. |
| D8 / Q8: cancelled parent | An admitted Begin always supersedes its predecessor, even when its parent is already cancelled. The new token is immediately non-current. Parent cancellation and token-local release do not advance the revision. |
| D9 / Q9: migration | Prove the sort/SVG pilot first. On success, converge basic tokens within root/display only. Apply final-delivery handling only to matching disposable final results; preserve standalone revision uses and all feature-specific protocols. Explorer and other features remain outside scope. |

## Lifetime observations

These terms describe implementation contracts, so they belong here rather than
in the domain-only `CONTEXT.md` glossary.

| Observation | What it establishes |
| --- | --- |
| Request currentness | The captured revision still matches and its context is not cancelled. This is an observation, not a lock or permission to commit a disk write. |
| Worker exit | The tracked worker returned. Its queued UI callback may still be pending. |
| Result delivery | The UI callback ran and either applied or discarded its result, then performed that delivery's finishing. |
| Operation completion | The feature's named operation reached its own completion point. For example, a load may be complete while preloads remain active. |
| Finite settlement | The feature joined and delivered its finite work using its existing protocol; this need not stop continuous producers. |
| Continuous-producer stop | Playback or another retained producer actually exited, observed separately from finite operation completion. |
| Committed disk effect | A write or move took effect. Superseding its initiating request does not erase the effect or its required reconciliation. |

Features choose the lifetime: view-bound work can retire on close; retained
session work can outlive a particular result; work reconciling a committed
effect has its own owner. These are documented distinctions, not configurable
modes in a universal manager.

## Request and delivery contract

The request owner is instance-local and zero-value usable. Begin and invalidate
supersede and cancel the previous request. An invalidation before first use is
safe. Tokens capture request identity and context; callers no longer replace
token context fields after creation. Explicit release is idempotent, cancels
only that token and makes it non-current without advancing the owner's revision.
Old release cannot cancel a newer token or the supplied parent. Basic revision
and cancellation operations remain safe for concurrent observation/use.

The final-delivery candidate owns the release handoff for one disposable
terminal result. Before handoff, early worker exit releases its request.
Handoff transfers responsibility before invoking the dispatcher: a dispatcher
may execute immediately or queue the callback. Returning from the worker after
handoff must not cancel a valid queued result. At delivery, the helper checks
request currentness before application and finalizes that request once even
when application is skipped. Feature-owned finishing must still observe the
same operation generation. Do not move UI-bound finishing to a cancellation
goroutine when the event loop retires.

Application and finishing callbacks run without module locks. They may start
another request. Finalizing the old one affects only its own context and
completion. A currentness check does not make a whole callback atomic against
later cancellation or reentry; features retain their required checks after
callbacks and before feature-specific effects. The helper cannot roll back an
effect already performed.

The helper owns neither the UI queue nor its draining and has no general Wait
or Settle operation. Prefer normal context propagation; do not create detached
cancellation observers, per-request workers or a new callback scheduler to make
the pilot fit. The callback finisher is observed when the callback runs; an
event loop that has stopped is not assumed to deliver pending callbacks.

## Consumer boundaries

- Root sorting retains progress/menu state, collection reconciliation and its
  per-generation completion signal. An obsolete sort still finishes its own
  delivery without clearing a newer sort's progress or applying old ordering.
- Display SVG rendering retains vector/display identity checks, debounce,
  raster workers and publication. The candidate replaces its manual
  worker-to-callback release handoff and request-currentness boilerplate.
- A successful load retains its request token for neighbor preloads after
  `LoadDone`. Retry chains retain their existing token and completion. Load
  callbacks can reenter loading; an old callback must not terminate the newer
  image's playback. These paths use basic tokens without terminal-delivery
  shortcuts that would change their lifetimes.
- Save and Export continue reconciling committed writes after request
  cancellation. Their stale callbacks can carry necessary effects and cannot
  be discarded wholesale. The separate reconciliation lifetime and completion
  remain in `filework.go`.
- Feature Stop remains the admission barrier. Close/reopen, stale callbacks,
  tracking of retired generations and feature-specific queues remain explicit.
- Production keeps ordered stop calls on UI and existing joins off UI. Test
  cleanup remains stronger where intended. In particular, Spiral's blocked
  preview-read exception remains; cancellation does not interrupt native I/O.

## Verification required for implementation

The [specification's testing decisions](../.scratch/ma-032/spec.md#testing-decisions)
are the authoritative command map: AC1-AC20 cover request/final-delivery
contracts, the two real pilot consumers, every migrated root/display lifetime,
production shutdown, the lead's pilot assessment, convergence or fallback, and
the repository verification gates. Proposed tests must exist and actually run;
an empty selection cannot establish acceptance.

Verification follows the complete inventory of 12 root and three display
request owners. Current production-hook display tests prove cancellation and
stale-result rejection, not a production join of all display workers.
`waitForShutdown` omits that join; the stronger harness cleanup does not change
the production contract. The pilot's responsibility reduction requires recorded
lead assessment as well as passing behavioral tests.

No runtime tests, native qualification or changed-code inspections were run
for this documentation-only interview, specification and ticket publication.

## Limits and follow-on work

This is a conditional refactoring, not a claim of a currently reproduced bug.
It cannot interrupt blocked native I/O, make stale-result checks transactional,
or determine what feature completion means. It does not consolidate Explorer,
Grid, search, analysis-cache or other feature workers; change durable-effect
policy; or introduce shared Stop/Wait/Settle choreography.

Implementation begins with ticket 01; ticket 02's pilot verdict gates either
the migration batches or fallback. Final qualification depends only on the
selected branch. Sorting and SVG use the accepted Owner/Token/FinalDelivery
contract; scan retains its local lifetime through a temporary progress seam.
Ticket 04 reunifies that seam after acceptance, or ticket 08 removes it on
fallback. The recorded ticket 02 verdict accepts extraction: both real callers
delegate currentness and release handoff through one contract, and sorting also
delegates its captured final-delivery finisher. SVG retains feature identities,
debounce and raster workers. Passing real-consumer and production shutdown tests
support the verdict. Tickets 03-07 may migrate; fallback 08 is inapplicable.
No ADR is needed for this small reversible extraction decision. General
concurrency vocabulary stays in this document, leaving the domain glossary
unchanged. Dependencies and native distribution inputs remain unchanged.
