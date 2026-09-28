# MA-032 implementation and evidence

Status: ticket 01 complete; ticket 02 is the frontier. Base: `04cb74c`.
Authorization: 2026-09-28 `/implement MA-032`, TDD/SDD, a commit after each
ticket, draft PR/CI, then ready-for-review and the GitHub Codex review loop.
No merge or release is authorized.

## Frame and accepted specification

Deep route: pilot shared request mechanics in sorting and SVG delivery, assess
both callers, then complete either root/display convergence or local fallback.
The accepted [design](../docs/worker-lifetimes.md), local
[specification](../.scratch/ma-032/spec.md) and
[tickets](../.scratch/ma-032/issues/README.md) fix D1-D9 and AC1-AC20.
Test seams were already accepted: the public request API, real viewer sort and
display SVG behavior, existing consumer contracts and actual production shutdown.
No new behavior, dependencies, native integrations, queues, worker managers or
feature-local lifecycle migrations. Cancellation does not interrupt native I/O;
delivery, operation completion and worker exit remain distinct.

## Contract and task graph

Candidate: `internal/requestlife.Owner` (zero value, never copied after use),
`Begin(context.Context) Token`, `Invalidate() uint64`, `Revision() uint64`.
Immutable `Token`: `Context() context.Context`, `Current() bool`,
`Revision() uint64`, `Release()`; release never advances owner revision.
`Token.FinalDelivery() *Delivery`: a worker defers `Abandon()` then calls
`Dispatch(dispatch func(func()), apply func(), finish func())` for one disposable
terminal result. Transfer precedes dispatch, callbacks run outside locks, stale
delivery still finishes, early abandonment releases, and cancellation never
needs queue draining. A nil finisher is allowed. No runtime goroutine is added.

Graph: 01 -> 02 -> {03,04,05,06,07} -> 09 on acceptance;
01 -> 02 -> 08 -> 09 on rejection. Root composition edits serialize.
Each ticket's V commands and parent AC map are the verification contract;
enumerate selections and retain uncached verbose race output in `.scratch/ma-032`.
The authorized GitHub full suite supplies the final broad race gate; use
`make verify-build` locally and record this review-workflow substitution.

| Ticket | Files / contract / test | Owner and budget | State |
| --- | --- | --- | --- |
| 01 | New requestlife API/tests; asyncop/sort/viewer, affected harness and tests; real sort handoff and production shutdown; local scan retained | T0; at most 1 bounded T1 implementation; focused V1-V6, no full suite | Complete |
| 02 | display/vector.go, display contract tests, root shutdown tests; same delivery API; D2 verdict before adoption | T0; 1 T3 scout, focused V1-V5 | Pending |
| 03 | display feature/load/preload/animation/lifecycle and affected tests; retained-load and playback protocols | T0; optional 1 T1 task <=3 files, focused V1-V4 | Blocked by verdict |
| 04 | root asyncop/drop/openfiles/explorer/locationmap and composition/tests; four basic owners | T0; optional 1 T1 task <=3 files, focused V1-V5 | Blocked by verdict |
| 05 | filework/save/export/search reconciliation and tests; committed callbacks stay explicit | T0; no spawn, focused V1-V4 | Blocked by verdict |
| 06 | clipboardwork/clipboard/copyfiles/copyselection, viewer/harness and tests; retain capture and delivery completion | T0; optional 1 T1 task <=3 files, focused V1-V4 | Blocked by verdict |
| 07 | favthumbs/autoupdate/viewer/harness and affected tests; basic tokens only | T0; optional 1 T1 task <=3 files, focused V1-V4 | Blocked by verdict |
| 08 | Restore only pilot runtime changes; test-only shared local contract | T0; no spawn, focused V1-V4 | Conditional fallback |
| 09 | Remove remaining duplicate request mechanics; retain independent revisions; docs/metadata/evidence and full qualification | T0; no spawn, final gate | Blocked by selected branch |

Every ticket updates exact Qodana exclusions, root test shards, architecture when
needed, local issue checkboxes and this record before its commit. GoLand changed
file inspections include weak warnings; unavailable/incomplete checks remain
unverified. Reviews and fixes belong to the lead. Dependency closure unchanged.

## Delegation gates and ledger

T01 shared API: T1 implements only `internal/requestlife/request.go`,
`request_test.go`, `delivery_test.go` under the fixed interface above. G1 <=25
line prompt; G2 the ticket's V1 race command; G3 three exclusively owned files;
G4 pure context/dispatcher contract independent of UI; G5 lead has fixed the
interface but has not developed implementation or tests. S: behavioral TDD
requires reasoning; W: no implementation supplied. Lead owns caller adapters,
review, metadata and all post-review fixes. Agent works vertical red/green slices.

T02 recon: T3 read-only scout of display's SVG contract fixture and root production
shutdown fixture. G1 bounded question <=25 lines; G2 exact existing test-name
enumeration and file:line facts; G3 no writes, <=3 fixture files; G4 independent
fixture context is smaller than the lead's implementation context; G5 lead has
not read those fixtures. Shell located the suites but cannot explain controlled
timer/queue fixture composition. S: not a mechanical transform; W: no supplied
implementation. Scout supplies facts only, never review or acceptance judgment.
Available model mapping: T3 uses `gpt-6-luna`; T1, if justified, `gpt-6-sol`.
The historical model names in the working agreement are interpreted as tiers.

| Task | Spawn budget / actual | Lead reviews | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| 01 | 1 / 1 | 1 | No | Contract and real sort tests, vet, formatting, metadata, inspections passed |
| 02 recon | 1 / 1 | N/A | No | Existing SVG fixture mapped; queued/held production SVG shutdown needs coverage |

## Progress

- [x] Read accepted spec, all nine tickets, working agreement and TDD skill.
- [x] Clean starting branch `feature/ma-032-request-lifetimes`, base `04cb74c`.
- [x] 01 shared request and sorting pilot.
- [ ] 02 SVG pilot and recorded verdict.
- [ ] Selected migration/fallback tickets committed separately.
- [ ] 09 convergence, final CI and inspections.
- [ ] PR ready, latest-commit clean Codex/security/CodeQL/Qodana/CI round.

## Evidence

Initial sandbox checks: snap wrapper cannot run Go; direct installed
`/snap/go/current/bin/go` works (Go 1.27.1, linux/amd64). Authorized test runs use
that toolchain in PATH. GitHub/Docker and build-cache access checked through the
normal escalation mechanism. Docker reports native linux/x86_64 with 33.3 GB RAM.

### Ticket 01 — candidate tree on parent `04cb74c`

V1: both enumerated requestlife suites passed uncached under `-race` (13
subcases, final run 1.014s). V2-V4: 13 enumerated root top-level tests passed
uncached under `-race` (44.967s), including all CollectionLifecycle preparation
scan/replay/sort cancel/replacement/close_reopen/stop children. The new suite
covers queued current/superseded results, inline reentry, and production shutdown
with a queued sort followed by the actual post-event-loop wait and stale drain.
Existing collection preparation tests supply held-worker shutdown coverage.
No skipped cases, race reports or empty selections. Logs and enumerations:
`.scratch/ma-032/ticket01-{contract,contract-list,ui,ui-list}.log`.

Behavioral red/green: compiling request stubs failed first-token identity and
early abandonment; deliberate real-consumer mutations failed queued publication
(early release), stale progress/collection protection (omitted currentness),
shutdown retirement (omitted production invalidation), and inline reentry (old
release invalidated owner). Each mutation was restored before passing the suite.
Corresponding logs are `requestlife-red-*` and `ticket01-*-red.log` in that folder.

Focused `go vet -tags no_emoji,nodynamic ./internal/requestlife ./internal/ui`,
`make fmt-check`, `git diff --check`, `make check-qodana-test-exclusions`, and
canonical Docker `make check-test-shards` passed (743 runnable tests, 3 shards).
The new three test files have exact Qodana exclusions and the root suite is ui-1.

GoLand inspection-tool fallback: IDE-local Qodana's interactive action is not
exposed by the available IDE tools. Inspected all 12 changed Go files using
`get_file_problems(errorsOnly=false)` with the IDE profile, including weak
warnings, without timeout. Scope: requestlife's three files; root asyncop,
sort, viewer, run, harness_test, sort_test, settings_apply_test, filestate_test,
requestlife_test. Replaced 12 direct error comparisons in contract tests with
`errors.Is`; reinspection is clean. The two unchanged filestate test duplicate
fragments (lines 39/72) are covered by its existing exact Qodana exclusion.
Viewer's unchanged title and dropzone blocks (lines 506/619) receive structural
duplication suggestions: retained as distinct presentation policies, no repeated
request protocol or correctness finding; no new suppression. This IDE profile
does not replace the pending CI post-suppression Qodana SARIF gate.

Lead assessment: sort no longer coordinates token-currentness rejection, release
after delivery, or per-generation finalization in its result application. One
FinalDelivery contract owns those obligations; sorting retains progress, rollback,
menus and collection commit. Its existing fyne dispatcher is captured before
worker admission, with an instance seam for held/inline test delivery. No worker,
queue manager or new shutdown join. Scan remains on the local lifecycle; shared
progress presentation is the temporary bridge owned by 04/08. This establishes
the first consumer only; extraction remains undecided until ticket 02.
