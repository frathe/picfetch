# MA-029 — explicit browsing visits

Status: tickets 01-03 complete; implementing ticket 04 of 09.
Date: 2026-09-27. Base: `11e8c4c` on `feature/ma-029-browsing-visits`.
Route: Deep SDD, vertical TDD slices. Lead owns design, review and fixes.

Deliver one private root-UI owner for Explorer, ranked search and Location Map
visits, scopes and returns. Accepted D1-D10 remain authoritative in
[the design](../docs/browsing-visits.md). The user authorized implementation,
commits after every ticket, draft PR/CI, then ready-for-review and the GitHub
Codex review loop. Merge and release are outside scope.

The nine [local tickets](../.scratch/ma-029/issues/README.md) are published under
that instruction. They and raw evidence remain gitignored. This tracked plan
retains completed criteria, commands, inspection dispositions and commit evidence.
The accepted spec already agrees the real viewer harness and private module
interface as test seams; no new seam approval is needed.

## Decisions and limits

- Keep collection identities, command admission, feature workers, cameras,
  ranked production/history and Grid interaction with their existing owners.
- Restricted empty scopes cannot select unrelated collection members. Empty
  filters and incomplete discovery remain distinct from exhausted membership.
- Frozen search-image order, path-based cohorts and exact occurrence clusters
  differ from live direct-map visits. Preserve these policies explicitly.
- Preserve duplicate inspection's live groups, collection Home/End and
  singleton fallback; preserve Picture-frame scheduling/shuffle and failed-load
  recovery's separate successor policy and single display retry chain.
- No dependency, native input/window glue, new user string, persistence,
  collection redesign, worker or generic deferred-command queue is planned.
  Existing shipped dependency/notice obligations remain unchanged.
- Full test execution belongs to native amd64 GitHub CI in the authorized
  review workflow. Run focused local tests plus local verification-build;
  do not duplicate the complete local race suite. Native behavior is not
  newly qualified by harness tests; changed native glue would need evidence.

## Contracts and migration map

Ticket 01 introduces `browsingScope`, captured by
`(*viewer).captureBrowsingScope() browsingScope`, with explicit restriction,
discovery completion, captured collection generation/source binding and ordered
indexes. `Next(from, delta)`, `First()` and `Last()` return `(int, bool)` from
that immutable value; absence is separate from ordinary negative wraparound
indexes, so callers cannot pass it through ShowImage's modulo wrapping. Baseline navigation
remains an explicit duplicate-model adapter. Both preloads consume one capture.
Capability observations do not capture payloads or perform source I/O.

`browsing.go` is the sole temporary feature-scope adapter. `visibility.go`,
`load.go` and action consumers must not reproduce its precedence chain.
Bindings initially describe the legacy feature lifetime; ticket 03 establishes
the authoritative visit token used to authorize transitions, and 04-06 migrate
the remaining source adapters. Captured values never authorize a future return.

Ticket 03 establishes a value-only `browsingVisits` module in root UI:
explicit entry/open-image/return/leave operations, immutable observations and
visit/collection tokens. Return planning and commit validate the same token;
root executes admitted effects. Exact transition signatures will be recorded
before ticket 03 tests, after 01 establishes the snapshot contract. Search
retains an origin independently of its active query; map/Explorer origins and
foreground surface are distinct. Covers stay MA-028 observations. Grid bookmarks
carry interaction, never independent authorization to restore callbacks.

03 transition contract: `browsingVisits` retains a stack of value visits: an
Explorer map parent and a cohort foreground, each with a monotonic identity and
collection binding. `enterExplorer`, `openExplorerCohort` and `openImage` own
entry; `planReturn(binding, generation, destination)` captures an immutable
return and `commitReturn(plan, generation)` rejects retired/rebound sources.
Grid and parent are explicit destinations; feature visibility never authorizes
one. `leaveExplorer` retires the family. Source reconciliation rebinds surviving
visits and bookmarks, making previously captured returns obsolete.
The module keeps only Grid's captured interaction value, not cohort analysis or
camera state. Root restores through one boundary: install current subset and
callbacks, then `Grid.RestoreInteraction`. A pre-close callback now also captures
path-based Explorer subsets. Comparison/input covers remain current MA-028 facts,
independent of retained visits. Search and Location Map keep the documented
legacy adapters until 04-06; they may temporarily cover a retained Explorer.

Authority inventory before migration:

| Current authority | Retirement ticket |
| --- | --- |
| `cohortIndexes` feature precedence and nonempty inference in navigation | 01 adapter centralization; 09 final convergence |
| Explorer `HasCohort` plus surface visibility as browsing ownership | 03 |
| `searchPresentation.imageOrder`, search active/history-origin authorization | 04 |
| `locationInput.image`, direct-map order/return flags | 05 |
| `locationInput.cluster`, cluster order and bookmark return authorization | 06 |
| Distributed source/remapping and origin restore decisions | 07 |
| `imageLoadFailed` feature-scope inference | 08 |
| Remaining input/menu/lifecycle projections | 09 |

## Ticket graph and routing

`01 -> 02`; `01 -> 03 -> 04`; `03 -> 05 -> 06`;
`04 + 06 -> 07 -> 08`; `02 + 08 -> 09`.
Run numerical order to avoid shared root-file edits. Every ticket has focused
red/green, guard failure evidence, lead review, inventory checks, changed-file
GoLand inspection, documentation updates and a separate commit.

| Ticket | Owner and files | Contract/test and verification | Budget |
| --- | --- | --- | --- |
| 01 scope | T0; browsing.go, visibility.go, load.go, explorer.go, viewer.go, new browsing scope tests | Scope contract above; `TestBrowsingScope`, baseline `TestBrowsingCompatibility`; existing search/Explorer/map families | 1 scout, 2 reviews, no full suite |
| 02 targets | T0; browsing.go, visualsearch.go, batch/clipboard adapters only as necessary, action tests | Ranked vs persistable Favorite capture; batch targets and displayed pixels; `TestBrowsingActionTargets`, existing ranked-target/admission families | 0 implementation spawns, 2 reviews |
| 03 Explorer | T0; new browsing visit module/tests, explorer.go, Explorer feature Host, Grid bookmark bindings, command observations | Authoritative transitions, cohort/Unassigned round trips, exhaustion and obsolete delivery; transition/round-trip/empty/deferred/lifecycle tests and Explorer anchor | 1 bounded scout if needed, 2 reviews |
| 04 search | T0; visualsearch.go, visualsearch Feature origin Host, browsing module, search tests | Frozen image/live Grid, history vs Exit, deferred delivery and valid origin; transition/round-trip/progressive/deferred/lifecycle plus existing search anchors | 1 bounded package implementer only after contract fixed, 2 reviews |
| 05 direct map | T0; locationmap.go, browsing module, locationmap tests | Live mapped scope, validation and admission, camera/exit/exhaustion; progressive/round-trip/empty/deferred/lifecycle and Location Map anchor | 0 spawns, 2 reviews |
| 06 clusters | T0; locationmap.go, browsing module, Grid bookmark bindings, cluster tests | Frozen exact occurrences, remapping, cluster Grid/image/map stages; identity/progressive/round-trip/empty/lifecycle and Location Map anchor | 0 spawns, 2 reviews |
| 07 collection changes | T0; sourcechange.go, sort.go, browsing module, relevant filework/drop seams, reconciliation tests | Detach, commit, reconcile, restore; exact/same-source/eligible fallback; collection/identity/deferred/lifecycle plus source/sort anchor | 1 bounded scout if needed, 2 reviews |
| 08 recovery | T0; load.go, browsing module, sourcechange.go, recovery tests | Restored origin then recovery successor, same request revision, HEIC stop; load-recovery/empty/lifecycle plus HEIC/search/Explorer anchors | 0 spawns, 2 reviews |
| 09 convergence | T0; remaining keys/window/menu/lifecycle adapters, all changed scope files/docs | Complete authority inventory, mixed routes, all acceptance families, verification-build, CI/review/security/SARIF | 0 review spawns; final full CI suite |

All Go test commands use `go test -tags no_emoji,nodynamic -count=1 -v
./internal/ui -run '<anchored families>'`. The local tickets spell out the exact
expressions; finalize each new subcase inventory in evidence as it is added.
Extend existing integration tests where they already cover the criterion, rather
than duplicate assertions merely to populate a proposed family name. Any mapping
to existing tests must be explicit. A skipped or unmatched case is unverified.

### Delegation gate

01 scout: read-only causal fixture inventory across visualsearch_test.go,
explorer_test.go and locationmap_test.go. G1: 8-line bounded prompt; G2: verify
cited helpers/selectors with `rg -n`; G3: three files, no writes; G4: needs only
fixtures, not architecture decisions; G5: lead has not read those large test
sequences. Shell identified files first. Rule S does not answer causal sequencing;
Rule W does not apply. Route: T3 / gpt-6-luna. Lead works on scope contracts.
Future delegation requires its own recorded G1-G5; hot context and all review
remain lead-owned. Maximum two concurrent subagents. Requested model suitability
is interpreted by task tier: gpt-6-sol for isolated Go implementation,
gpt-6-luna for bounded read-only recon; no peer review delegation.

02 stays T0: the lead already holds action-capture context (G5 fails), and
consumer target policy is cross-feature. No independent implementation spawn.
03 reuses the T3 scout for Grid callback ordering in nav.go/grid.go/ranked.go.
G1: bounded 8-line question; G2: verify cited capture/close/notify call sites
with `rg -n`; G3: three read-only files, no edit overlap; G4: only Grid sequencing;
G5: lead has not reconstructed these callback paths. Shell located the relevant
Host/bookmark entry points first; causal ordering requires reading beyond grep.
Rules S/W: no mechanical transform or prewritten implementation. Lead develops
the owner interface while the scout returns facts, never review decisions.

## Acceptance and progress

- [x] Frame: accepted behavior, route, non-goals and authority inventory.
- [x] Spec: D1-D10 and accepted seams; publish nine local tickets.
- [x] Recon: locate navigation, preload, source/return and test entry points.
- [x] Plan: files, dependency graph, routing and focused verification.
- [x] 01: explicit immutable scopes, integrated navigation/preloads, baseline.
- [x] 02: separate captured action targets.
- [x] 03: Explorer authority, round trips, exhaustion and lifecycle.
- [ ] 04: ranked search authority, history and origin restoration.
- [ ] 05: direct-map authority, live navigation and valid return.
- [ ] 06: exact frozen cluster authority and return stages.
- [ ] 07: ordered reconciliation and identity/fallback matrix.
- [ ] 08: single-chain failed-load recovery and HEIC guidance.
- [ ] 09: route/authority convergence and final verification.
- [ ] PR ready, fresh clean latest-head Codex code/security reviews, CI,
  CodeQL and post-suppression Qodana SARIF clear of actionable findings.

## Evidence ledger

Initial tree clean at `11e8c4c`; no PR existed for this branch. GitHub CLI needs
network escalation in this sandbox. The Snap launcher refuses sandbox execution;
the installed `/snap/go/current/bin/go` runs normally (Go 1.27.1 linux/amd64).
Use that PATH and a writable temporary Go cache for local commands.

| Ticket | Spawns budget/actual | Review rounds | Full suite | Result |
| --- | --- | --- | --- | --- |
| 01 | 1 / 1 | 1 | no | complete; evidence below |
| 02 | 0 / 0 | 1 | no | complete; evidence below |
| 03 | 1 / 1 | 1 | no | complete; evidence below |
| 04-08 | per routing above / 0 | 0 | no | pending |
| 09 / review | 0 / 0 | 0 | CI | pending |

Compatibility mapping: the proposed `TestBrowsingCompatibility` family reuses
the existing `TestStepImage_*`, `TestHandleKeyEvent_*HomeEnd*`,
`TestHandleKeyEvent_*Inspect*` and `TestAdvance_*` behavior tests. Do not create
duplicate wrappers merely to reproduce those assertions under a new name.

Each completion appends executed red/green selectors and actual results,
negative verification, inspected files/profile/revision and dispositions.
GoLand tool inspections are the available IDE fallback; they are not claimed
equivalent to qodana.starter. CI SARIF is a separate final gate.

### Ticket 01 evidence (working diff over `11e8c4c`)

New executed subcases: `TestBrowsingScope/snapshot_binding_and_discovery` and
`TestBrowsingScope/restricted_without_current_members`. The first checks
ordinary/pending/completed observations, frozen ranking and source identity,
collection binding and a new binding after search reopen. The second drives
actual Right/Left/Home/End through the viewer and asserts no load request and
no speculative neighbor outside an exhausted restriction.

Red: the empty cohort preloaded b/c and Right advanced to index 1/revision 4
(`FAIL .../internal/ui 0.219s`). The observation test failed with ordinary
completion=false and generation=0 (`FAIL .../internal/ui 0.091s`). Green:
both pass (`ok github.com/frathe/picfetch/internal/ui 0.193s`). Deliberately
restoring nonempty-only restriction reproduced both unrelated preloads and
Right navigation (`FAIL 0.173s`); restoring explicit restriction passed both
subcases (`ok .../internal/ui 0.715s`).

Focused uncached verbose gate passed (`ok .../internal/ui 25.989s`):
`^(TestBrowsingScope|TestFindMoreLikeThisInitialRoundTrip|TestFindMoreLikeThisProgressiveForegroundIdentity|TestFindMoreLikeThisActionsCaptureRankedSources|TestVisualSimilarityExplorer|TestLocationMap|TestStepImage_.*|TestHandleKeyEvent_.*(HomeEnd|Inspect).*|TestAdvance_.*)$`.
This covers existing frozen preloads, one-scan payload capture, map/cluster and
Explorer restrictions plus ordinary/duplicate/Picture-frame compatibility.
The sandbox attempt stopped at an existing httptest socket prohibition;
the complete focused retry used approved local socket access and passed.

`make check-test-shards check-qodana-test-exclusions` passed through Docker:
718 runnables, three shards. No complete local race suite was run.
`git diff --check`, `make fmt-check vet` and the live `-list '^TestBrowsing'`
inventory passed; `TestBrowsingScope` is present and both required cases ran.

GoLand `get_file_problems(errorsOnly=false)` completed for browsing.go,
browsing_test.go, explorer.go, keys.go, load.go, visibility.go and viewer.go.
Six files have zero findings. viewer.go has four weak duplicate-fragment
warnings at 501, 553, 577 and 834, all outside the changed navigation methods
at 925+. Lead assessment: unchanged explicit title/reset/source-reconciliation
composition, no defect introduced by this slice; preserve existing separation,
no broad suppression or unrelated extraction. Fresh CI SARIF remains required.

Lead review: source precedence now appears once in the explicit temporary
adapter; navigation tests restriction independently of membership and preloads
capture it once. `cohortIndexes` remains only the documented load-recovery
bridge for 08. Feature visit ownership and final last-member parent transitions
remain 03-08 obligations, not completed by this prefactor.

Ticket 01 commit: `e2cb9e1`. Draft PR: https://github.com/frathe/picfetch/pull/68.

### Ticket 02 evidence (working diff over `e2cb9e1`)

Ranked Favorite capture now consumes `captureBrowsingScope` after its read-only
ranked observation; ordinary Favorites retain `persistedFiles`. No batch or
pixel-capture target policies needed implementation changes.

New `TestBrowsingActionTargets` inventory: frozen image Favorite through newer
rankings and actual naming/save, filtered ranked Grid Favorite, ordinary
Favorite during Explorer/direct-map/cluster visits, and delayed highlighted-file
copy through ranking publication and explicit search exit. All six pass.

This is behavior-preserving integration. The frozen-image test first passed
existing behavior; deliberate removal of frozen image order then produced the
intended failure (latest d/c saved instead of a/c/b, `FAIL .../internal/ui 0.165s`).
Restoring frozen order and integrating the shared scope passed the family
(`ok .../internal/ui 0.526s`). An earlier test setup mistakenly used an unset
Favorite directory; its temporary local fixture was removed and the test now
uses `t.TempDir`. That setup failure is not counted as behavioral red evidence.

The full focused command passed (`ok github.com/frathe/picfetch/internal/ui
3.630s`), with exact expression:
`^(TestBrowsingActionTargets|TestFindMoreLikeThisActionsCaptureRankedSources|TestCommandAdmissionVisits|TestCommandAdmissionQueries|TestWindowCommandAdmissionMatrix|TestCopy_.*|TestClipboardCapturesPixelsBeforeAViewRotation|TestClipboardNavigationCancelsHeldEncoding|TestCopySelectionSuccess|TestCopySelectionPixels|TestCopySelectionBusy)$`.
Existing tests provide the selection/highlight, displayed-pixel/region,
cancellation, refusal, immutable availability and progressive batch coverage;
these are explicit AC mappings, not duplicate TestBrowsing wrappers.

GoLand inspected browsing_test.go and visualsearch.go with weak warnings enabled:
zero findings, no timeouts. `make fmt-check vet` passed. Docker shard/exclusion
gate passed: 719 runnables, three shards. `git diff --check` passed.
Lead review confirms scope sharing does not narrow ordinary
Favorites or replace batch selection with visit membership. No new worker,
command replay, user string, dependency or native glue. Ticket 02 uses no
subagent because this is already-hot cross-feature context.

Ticket 02 commit: `90c3aab`, pushed to draft PR 68.

### Ticket 03 evidence (working diff over `90c3aab`)

The private value owner now retains Explorer map/cohort visits, a foreground
surface and collection/visit-bound, single-consumption return plans. Root owns
admission and effects; Explorer still owns cohort membership, analysis and
camera. All root `HasCohort` navigation/admission observations were retired.
Feature return requests no longer unconditionally reveal its map. Refused cohort
delivery rolls back feature membership. Path-subset Grid callbacks capture the
bookmark before Close and authorize Back/Escape/G through the owner; exact map
subsets keep their existing adapter until 06.

New executed families: `TestBrowsingVisitTransitions` (valid image/Grid/parent
transitions; collection, revision and retired identity guards),
`TestBrowsingRoundTrips` (cohort and Unassigned interaction/nonzero scroll/camera,
surviving selection), `TestBrowsingEmptyScope` (editable filter, committed final
removal, final decode failure), `TestBrowsingVisitLifecycle` (close/reopen,
replacement, removal/rebound Back), and `TestBrowsingDeferredReturns` (live modal
refusal, no replay, refused cohort membership unchanged).

Red: both cohort/Unassigned round trips lost query and selection, and final-member
removal failed to return to Explorer (`FAIL 0.297s`). The later final-decode guard
also failed by loading an unrelated successor (`FAIL 0.764s`). Green restored
interaction and stops the existing retry chain when no restricted successor
exists. This narrow recovery bridge is necessary for this slice's exhaustion;
the full successor-policy consolidation remains 08.

Negative verification: removing the collection guard failed with "old collection
return was accepted" (`0.013s`); removing the transition revision guard failed
with "already consumed transition was replayed" (`0.013s`); ignoring visit
identity failed the real close/reopen return (`0.139s`); bypassing return admission
failed the covered-return test (`0.138s`). Each mutation was restored.

The uncached verbose focused gate passed (`ok .../internal/ui 40.558s`):
`^(TestBrowsing.*|TestVisualSimilarityExplorer|TestFindMoreLikeThis.*|TestLocationMap|TestCommandAdmissionVisits|TestWindowCommandAdmissionMatrix)$`.
No skipped cases in this log. Full changed-package suites passed: grid 1.775s,
explorer 0.490s. Existing `TestVisualSimilarityExplorer/progressive_browse_camera`
and `unassigned_grid_return` are the progressive-scope and menu/key AC anchors;
its setup/replacement/source families and command-admission matrix remain the
lifetime/route anchors. The trial recording test now uses the restored filter's
normal extra Escape stage, consistent with preserving the accepted bookmark;
no recording evidence is dropped. Root now records one parent return, not two.

Docker shard/exclusion checks passed: 724 runnables, three shards. No new test
file, so existing exact Qodana test exclusions cover this slice. GoLand completed
all 21 changed code files (including new browsing_visits.go) with weak warnings
enabled and no timeouts. Only viewer.go's same four unchanged duplicate fragments
remain (now lines 502, 554, 578, 835); dispositions are unchanged from 01. This is
the IDE fallback, not a claim of fresh CI Qodana SARIF equivalence.

Lead review found and fixed callback ordering, duplicate return recording and
refused cohort mutation. Source removals and sorts rebind Explorer before fresh
bookmark restoration. Their currently separate Explorer/Location remap capture
adapters are an explicit 07 consolidation obligation. Search retains its legacy
origin adapter until 04, but restores an Explorer Grid through the new validated
boundary instead of reviving saved subset callbacks. No worker, native glue,
user string or dependency was introduced.

Final restored-guard Explorer/new-family check passed (19.991s), final trial
presentation projection check passed (0.534s), and the changed Explorer package
passed again (0.414s). Final `make fmt-check vet` and `git diff --check` passed.
Intermediate PR head `90c3aab` has successful full CI, CodeQL and Qodana jobs;
this is not the final-head review/SARIF gate and no final scan claim is made.
