# MA-032 implementation and evidence

Status: tickets 01-07 complete; ticket 09 local qualification complete, CI pending.
Fallback 08 inapplicable.
Base: `04cb74c`. Draft PR: https://github.com/frathe/picfetch/pull/71.
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
| 02 | display/vector.go, display contract tests, root shutdown tests; same delivery API; D2 verdict before adoption | T0; 1 T3 scout, focused V1-V5 | Complete |
| 03 | display feature/load/preload/animation/lifecycle and affected tests; retained-load and playback protocols | T0; 0 spawns (mechanical rule S), focused V1-V4 | Complete |
| 04 | root asyncop/drop/openfiles/explorer/locationmap and composition/tests; four basic owners | T0; 0 spawns (mechanical rule S), focused V1-V5 | Complete |
| 05 | filework/save/export/search reconciliation and tests; committed callbacks stay explicit | T0; no spawn, focused V1-V4 | Complete |
| 06 | clipboardwork/clipboard/batch/copyselection, viewer/harness and tests; retain capture and delivery completion | T0; 0 spawns (mechanical rule S), focused V1-V4 | Complete |
| 07 | favthumbs/autoupdate/viewer/harness and affected tests; basic tokens only | T0; 0 spawns (mechanical rule S), focused V1-V4 | Complete |
| 08 | Restore only pilot runtime changes; test-only shared local contract | T0; no spawn, focused V1-V4 | Inapplicable: extraction accepted |
| 09 | Remove remaining duplicate request mechanics; retain independent revisions; docs/metadata/evidence and full qualification | T0; no spawn, final gate | Local gates complete; full CI pending |

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
- [x] 02 SVG pilot and recorded verdict.
- [x] Selected migration/fallback tickets committed separately.
- [ ] 09 convergence, final CI and inspections.
- [ ] PR ready, latest-commit clean Codex/security/CodeQL/Qodana/CI round.

## Evidence

Initial sandbox checks: snap wrapper cannot run Go; direct installed
`/snap/go/current/bin/go` works (Go 1.27.1, linux/amd64). Authorized test runs use
that toolchain in PATH. GitHub/Docker and build-cache access checked through the
normal escalation mechanism. Docker reports native linux/x86_64 with 33.3 GB RAM.

### Ticket 01 — candidate tree on parent `04cb74c`

Committed as `c415c7e`, pushed to draft PR 71.

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

### Ticket 02 — D2 verdict (candidate tree on parent `c415c7e`)

Committed as `6a3f477`, pushed to draft PR 71.

**Extraction accepted.** Recorded before any owner beyond sorting/SVG migrates.

| Caller | Previous caller obligation | Shared guarantee now used | Retained feature policy |
| --- | --- | --- | --- |
| Sorting | Check currentness in finishSort; defer token release; finish captured operation even when stale | FinalDelivery transfers release before dispatch, gates application and runs the captured finisher once on delivery | Mode rollback, progress/menu state, collection commit and existing fyne dispatch |
| SVG | Maintain handedOff flag; conditional worker cleanup; deferred callback release; repeat request-currentness check at delivery | The same FinalDelivery owns early abandonment and queued/inline handoff, currentness and request-local release | Debounce, pre-raster cancellation, vector/display identity checks, raster worker tracking, pixel publication and repaint |

Neither adapter recreates a handoff flag, wraps a feature switch, or delegates
worker admission/settlement. Basic tokens retain parent context at creation.
Callbacks run outside locks; cleanup affects only the captured request.
This is responsibility reduction in both actual callers, not a code-size verdict.
Tickets 03-07 are eligible; 08 and AC19 are inapplicable, not passed.

V1/V2: both enumerated display suites passed uncached under `-race` (3.722s),
including existing vector debounce/failure/identity checks and new superseded
queued delivery, inline repaint reentry, clear/reopen and terminal Stop cases.
The complete existing presentation contract was also exercised. Shared contract
tests passed again. V2/V3: seven enumerated root top-level tests passed uncached
under `-race` (49.190s), including real queued sort and held/queued SVG shutdown,
actual post-event-loop waits, CollectionLifecycle, HEIC backend lifecycle,
HypnoTunnel's blocked-preview exception, and all three capture-sort guards.
Logs and enumerations: `.scratch/ma-032/ticket02-{display,display-list,contract,ui,ui-list}.log`.

Negative evidence: omitted SVG request-currentness failed superseded publication;
owner-wide cleanup failed inline repaint reentry; omitted production display Stop
failed both held and queued SVG shutdown. All mutations restored; no skipped
cases or race reports. Focused root/display vet, formatting, exact test exclusions
and canonical Docker shard check passed (743 runnables). GoLand inspected all three
changed Go files (vector.go and both requestlife integration test files), including
weak warnings, with no findings/timeouts using the same fallback profile as 01.

### Remaining-ticket routing

After the fixed public contract is proven, basic-token renames and call-site
rewrites are deterministic. The lead performs these mechanical batches under
rule S, captures parent contexts before Begin, and owns protocol review. No
implementer is spawned to rediscover the already-understood token transformation.
Existing behavior suites are run before/after their refactor; no inventory-mirror
tests are added. New behavior gaps, if found, get a vertical failing test first.

### Ticket 03 — display convergence (parent `6a3f477`)

Committed as `0add587`, pushed to draft PR 71.

All three display owners now use requestlife. Load/retry/preload signatures use
immutable shared tokens; load captures HEIC before Begin. Successful loading
keeps its token beyond LoadDone, while failed terminal loading releases it.
Animation uses basic tokens and preserves its per-frame acknowledgement, pause
acquisition, completion and worker barrier. SVG retains FinalDelivery from 02.
Removed `display/lifecycle.go`: its revision helper had no independent consumers;
the existing `Feature.revision` for requested/displayed identity is unchanged.
Root and other feature-local lifecycle implementations remain untouched.

Baseline: ticket 02's complete display contract. Negative proof: deliberately
releasing the load token in finishLoad failed the existing preload contract with
`neighbor preparation was not admitted` (ticket03-retained-load-red.log), then
the mutation was removed. Final uncached race output: display 3.272s (both
enumerated suites, load/preloads/animation/captures and all pilot cases); root
39.669s (CollectionLifecycle and all pilot/shutdown cases). Logs/enumerations:
`.scratch/ma-032/ticket03-{display,display-list,ui,ui-list}.log`.
Focused display vet passed. Docker shard inventory remains 743/3; exact exclusions
unchanged. Four surviving changed Go files inspected with GoLand including weak
warnings: feature.go/load.go/preload.go/animation.go, all clean without timeout.
Lead diff/inventory review: no new join, queue, admission rule or release protocol;
all retired generations remain under their existing barriers. Spawns 0, one lead
review, no broad local suite. Formatting and whitespace clean.

### Ticket 04 — preparation and chooser (parent `0add587`)

Committed as `9bda248`, pushed to draft PR 71.

Scan, native chooser, root Explorer preparation and root Location Map preparation
now use the shared Owner/Token API. Includes invalidation-only callers in source
reconciliation and analysis-cache quiescence. Scan captures its parent HEIC context
before Begin; the separate existing admission snapshot/check result still decides
HEIC discovery/replay availability. Native calls keep their original worker and
queued command-admission checks. Feature-local map/Explorer lifecycles unchanged.
The temporary sortOpUI/asyncProgressUI split is removed: scan and sort use the
same asyncOpUI shape, with independent instance owners and completion signals.

Baseline map/chooser tests passed before migration (21.986s); the previous ticket
supplies the unchanged scan/collection baseline. The existing stale-scan test
failed under an omitted-currentness mutation, reporting stale.jpg replacing
current.jpg and changed collection identity (the invalid application also induced
race warnings); restored before migration. Final uncached race outputs: root
V1/V3 and focused scan regressions 67.577s; V2 map preparation/lifecycle 21.872s;
affected collection-admission, launch, menu, initial-open and Escape scan-cancel
test adapters 13.562s. All selections enumerated; no skips or race findings.
Logs: `.scratch/ma-032/ticket04-{ui,maps,adapters}{,-list}.log`.

Root vet, formatting, whitespace, exact Qodana exclusions and canonical Docker
shard inventory passed (743 runnables). All 19 changed Go files inspected with
GoLand, errorsOnly=false, no timeout. Unchanged duplication suggestions in
filestate_test.go (39/72) and openfiles_test.go (258/295) are intentional fixture
repetition covered by existing exact test exclusions; viewer.go title/dropzone
suggestions retain ticket 01's separate-policy disposition. All other files clean.
Lead reviewed all four owners and inferred callers; no new background work or
completion policy. Spawns 0, one review, no broad local suite.

### Ticket 05 — committed file work (parent `9bda248`)

Committed as `fe49175`, pushed to draft PR 71.

Save, Export and search-origin reconciliation now use shared basic tokens,
including search invalidation in sourcechange/visualsearch. No FinalDelivery:
stale-but-committed writes still purge caches and queue current-source/alias
reconciliation. Its independent fileWork.ctx, worker retries and delivery-bound
operation completion are unchanged. Search retains its own owner/session checks;
feature-local search lifecycles remain outside scope.

Baseline three request-family suites passed (30.509s). Negative proof: deliberately
discarding a stale committed export failed the existing navigate=true alias case
with stale 8x16 pixels, thumbnail and native dimensions. Restored before migration.
Final five enumerated top-level suites passed uncached under race (69.907s),
including V1/V2 plus unrelated-commit reconciliation; no skips or race reports.
Logs: `.scratch/ma-032/ticket05-{baseline,committed-red,ui,ui-list}.log`.
Root vet, formatting, whitespace, exact exclusions and canonical Docker inventory
(743/3) passed. GoLand inspected filework/save/export/sourcechange/visualsearch,
including weak warnings, clean without timeout. One lead protocol review; no
implementation spawns and no broad local race suite.

CI recon routing: reuse T3 scout for the single failed Windows job on `9bda248`
(run 36461891670, job 109062417787). G1 bounded read-only prompt; G2 exact log
diagnostic and job URL; G3 no writes, one log plus <=2 implicated files;
G4 independent platform evidence; G5 lead has not read the log. S/W: interpreting
the failure location is not a supplied mechanical rewrite. Lead owns disposition
and fixes. One recon turn budget, not delegated code review.

### Ticket 06 — clipboard and region copy (parent `fe49175`)

Committed as `cbf4279`, pushed to draft PR 71.

Both clipboard owners now use shared basic tokens. Whole-image/Grid dispatch
retains busy admission, encoding cancellation and the queued result/menu-bound
completion; shutdown can finish cancelled returned work without UI delivery.
Region copying retains CaptureStable's acquisition-bound pause release and its
existing DoAndWait callback before shared clipboard finishing. Native work stays
under the existing clipboard worker barrier. No universal delivery wrapper.

All 12 enumerated V1/V2 top-level tests passed before (8.696s) and after (8.687s)
under uncached race testing, with no skips/race findings. Deliberate early operation
completion failed the existing queued-results case, then was restored. Logs:
`.scratch/ma-032/ticket06-{baseline,delivery-red,ui,ui-list}.log`.
Root vet, formatting, whitespace, exact exclusions and Docker inventory (743/3)
passed. Seven files inspected with GoLand including weak warnings: clipboardwork,
clipboard, batch, copyselection, viewer, run and harness_test. No new findings;
viewer's unchanged duplication suggestions retain the prior disposition.
One lead review, zero implementation spawns, no broad local suite.

Windows recon returned an existing native clipboard multiple-path test failure,
but its connector could not expose the artifact payload. One bounded follow-up
also hit download DNS/cache limits. Lead will retrieve raw events through gh;
no defect or clean platform gate inferred from the summary alone.

### Ticket 07 — previews and updates (parent `cbf4279`)

Committed as `573d3d6`, pushed to draft PR 71.

The final two root owners use shared basic tokens. Favorite preview capability
capture now precedes Begin; captured Favorite owner, full membership, source
versions and cache writers are unchanged. All-generation preview workers still
outlive superseded operation signals. Updater retains its serialized staging,
manual/automatic admission, UI event checks and worker barrier. No shared manager,
new release point or FinalDelivery wrapping. All inferred preview test adapters
and shutdown/source invalidations migrated; root-local types now have no runtime
consumers and remain for ticket 09 removal only.

19 enumerated V1/V2 top-level suites passed under uncached race before (3.640s)
and after migration (3.631s). Omitting queued event currentness failed the existing
updater callback test, then was restored. Another 15 affected preview/collection/
file-write adapter tests passed (11.055s). No skips/race reports. Logs:
`.scratch/ma-032/ticket07-{baseline,stale-event-red,ui,ui-list,adapters,adapters-list}.log`.
Root vet, formatting, whitespace, exact exclusions and Docker inventory (743/3)
passed. GoLand inspected all 10 changed Go files including weak warnings:
favthumbs/autoupdate/viewer/sourcechange/run/harness_test and favthumbs_test,
autoupdate_test, exportwork_test, collection_effects_test. No actionable findings:
viewer's old suggestions unchanged; autoupdate_test duplicate fixtures at
445/641/776/906 are intentional and covered by its existing exact exclusion.
One lead protocol review, zero implementation spawns, no broad local suite.

Lead retrieved the `9bda248` Windows artifact via gh. Its multiple-path PowerShell
decode exited status 1 at 30.03s with no decoded paths, matching the fixture's
30-second context limit. Neither this package nor that subprocess changed in
MA-032. Root cause remains unproven; do not change unrelated platform behavior
from this one timeout-shaped failure. Latest-head Windows CI must pass.

### Ticket 09 — convergence (candidate on parent `573d3d6`)

Lead inventory and diff assessment: all 15 owners use requestlife, with inferred
tokens and invalidation-only callers included. No aliases, temporary progress
forms, mutable token contexts, or duplicate request implementations remain in
root/display. Root lifecycle.go retains only its independent atomic revision;
its original revision test remains. Four obsolete root-local request tests are
removed because the stronger public requestlife contract covers those behaviors.
The ui-3 manifest count is reduced by four; exact Qodana exclusions remain valid.
Display's unused duplicate was already removed in 03. Explorer/compare and other
feature-local lifecycles, native workers and independent revisions are unchanged.

| Owner inventory | Count | Retained local protocol |
| --- | --- | --- |
| Root sort | 1 | Progress/rollback/collection commit; shared FinalDelivery |
| Display SVG | 1 | Debounce, raster workers, pixel identities; shared FinalDelivery |
| Display load and animation | 2 | Retry/load completion, borrowed retained preloads, playback acknowledgement/capture |
| Root scan, native chooser, Explorer preparation, Location Map preparation | 4 | Admission, collection checks, native/feature workers and queued results |
| Root Save, Export, search-origin reconciliation | 3 | Committed effects, independent reconciliation retries/completion |
| Root whole-image/Grid clipboard and region copy | 2 | Busy admission, captured pixels/pause release, native and queued completion |
| Root Favorite preview and updater | 2 | Captured owner/version/cache policy, durable staging, all-worker barriers |

Final uncached race evidence after duplicate removal: requestlife 2 suites/1.020s;
display 2 suites/3.637s (entire presentation contract plus MA032 delivery);
root 52 suites/98.793s; map preparation/lifecycle 2 suites/20.982s.
All applicable AC1-AC16 selections are covered, including held/queued real shutdown,
Spiral's blocked-read exception, HEIC backend lifetimes and retained preloads.
Every selection enumerated, all required children passed; no empty selections,
skips or races. Logs: `.scratch/ma-032/ticket09-{contract,display,ui,maps}{,-list}.log`.
AC17 uses the recorded 02 verdict; AC18 inventory converged; AC19 inapplicable.

`make check-test-shards verify-build` passed: native Docker inventory 739/3;
formatting, exact exclusions, TUF/generated assets/tag vectors/third-party notices,
full-repo vet and build. Full race suite is delegated to native Linux/amd64 CI
under the user-authorized review workflow instead of duplicating it locally.
Dependencies, native payloads, formats and user-visible strings unchanged.

GoLand re-inspected every one of the 48 surviving changed Go files against this
final code, including weak warnings, with no skipped files, errors or timeouts.
Raw scope/results: `.scratch/ma-032/ticket09-goland.json`. The ten weak duplicate
fragments are the previously assessed fixtures (autoupdate_test 445/641/776/906,
filestate_test 39/72, openfiles_test 258/295), covered by exact test exclusions,
and viewer's separate title/dropzone policies (507/620). No actionable finding
or new suppression. This IDE fallback does not replace CI post-suppression SARIF.
Writing-for-agents informed the concise AGENTS pointer to the lifetime contract;
architecture, lifetime design and open-work status now describe the converged code.

Ticket 09 remains in qualification until fresh full CI, CodeQL and Qodana SARIF
are assessed. The PR stays draft. After all ticket points are complete, mark it
ready and run the separate latest-head code/security review loop; do not merge.
