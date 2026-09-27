# MA-028 ticket and execution plan

Status: Windows prompt-focus and Help-owned navigation fixes locally verified; fresh PR 66 gates required; physical/native desktop acceptance open.
Date: 2026-09-27.
Planning revision: `7db3faf75ea704055b72baa98b2eca20b8bad268`.
Route: Deep, because the change crosses feature adapters and native input on
all shipped desktop platforms. Implementation base: `e440685` (clean worktree).
The implementation request authorizes the draft breakdown and local commits.

## Implementation record

The [implementation verification record](../docs/command-admission-verification-2026-09-27.md)
contains the completed command/route ledger, ownership review, observed
red/green failures, final test/inspection scope and native limitations.
The [Linux qualification record](../docs/command-admission-linux-qualification-2026-09-27.md)
adds passing scoped native scenarios at `6db8d73` using OS-injected XTEST input
on GNOME/XWayland, followed by diagnosis and repair of a native Escape/maximize
reset defect. The repair now passes native controls, TDD, inspections and a
fresh full Make gate. The [Windows qualification](../docs/command-admission-windows-qualification-2026-09-27.md)
adds scoped native evidence and a verified prompt-focus repair. Physical input
and full macOS desktop gates remain open;
the PR review-loop section records completed remote verification separately.
The historical ticketing evidence at the end describes the earlier planning
turn, not the current implementation state.

Linux qualification correction (user report, 2026-09-27): a real maximized-window
Escape reset failure is now reproduced with a fresh profile and one image.
Native geometry stays 1920 x 1131 with both maximization flags after Escape,
instead of the initial 624 x 409 welcome target. The earlier small rendered
surface was not merely a capture limitation. At the diagnostic handoff it was
unfixed; the subsequently authorized repair below is now locally verified.
The outstanding physical/native desktop gates still prevent overall acceptance.

One further read-only scout assignment traced pinned Fyne resize and native
geometry callbacks while T0 investigates application reset/maximize ownership.
G1-G5: under 25 lines; verify returned source citations with targeted reads;
zero files changed; bounded pinned-driver scope; separate from T0's app path.
No implementation, review or native desktop interaction is delegated.

### Remaining acceptance follow-up (2026-09-27)

#### Windows 11 qualification follow-up

PR review of `d662160` identified a Help-owned continuation blocked by fresh
main-window admission: Finis's clue can dereference a missing manual beneath a
main-window modal. T0 owns the fix and related About/Manual/Spiral routing.
Files: Help manual/about/finis and existing tests, root feature wiring/tunnel
and its existing test. Preserve fresh command guards; owned secondary-window
links must open/reuse their destinations without dismissing the main modal.
Verify red/green with `go test -tags no_emoji,nodynamic -count=1 ./internal/ui/help
./internal/ui -run '^(TestFinisClue.*|TestShowAbout.*|TestHelp_Finis.*|TestHypnoTunnel)$'`,
then focused race/vet and changed-file IDE inspections. Full suite: CI only.
One additional bounded read-only route inventory uses the existing scout;
G1-G5: cited callback ownership, no writes/review, independent remaining routes.
Budget: no new spawn; T0 fixes and reviews until a fresh clean bot round.

Result: the Finis panic and five owned routes have red/green coverage; separate
About/Spiral-F1 omission overlays fail as expected. Focused race/vet and nine-file
GoLand inspections pass with one unchanged, already-excluded test duplicate.
Native Windows repeats pass Finis/Manual/Spiral and About navigation beneath
Manage Favorites, preserving that modal; all owned windows/processes closed.
The [PR evidence](../docs/command-admission-pr-66-review-2026-09-27.md) records
the exact findings, commands, native scope and previous-head remote results.

The user authorizes completing feasible ticket 10 checks on Windows 11, then
committing, pushing and running the PR review loop. T0 retains native execution,
assessment and fixes. Use the existing AC10 procedure with an isolated ordinary
profile and copied fixtures; record OS-injected input separately from physical
keyboard evidence. No merge or release. Retain evidence under
`.scratch/ma-028/windows-2026-09-27` and summarize outcomes in a tracked Windows
qualification record. The full suite runs in CI under the review-loop rule.

One read-only scout located Windows profile isolation and the clipboard launcher
hold. G1-G5: bounded prompt, cited paths checked by T0, no writes, small independent
launch/clipboard scope, no duplicated native execution/review context. Budget
one spawn / actual one; no implementation or review delegated. T0 verifies
native payloads/screenshots/process exits, runs focused native guards, updates
ticket/spec/todos evidence, and inspects any changed code before publication.

Windows found a D4 prompt-control defect: native Tab focuses Export's metadata
checkbox, which swallows Escape; Delete's focused panel can remain focused after
dismissal and swallow the next viewer Escape. A new subtest in the existing
`TestCommandAdmissionModalOwnership` reproduces both failures (red log retained).
T0 repairs focus ownership in the existing ChoiceCard/ChoicePanel composition
and root prompt adapter, preserving its established arrow/Return/Escape controls.
Verify with that regression, focused command/prompt/widget suites, native repeats,
changed-file IDE inspections and latest-head CI. No new top-level test, package,
dependency or user-visible string is planned. This is D4's existing requirement
that the prompt's own controls remain usable, not a new command-policy choice.

Completed: scoped Windows scenarios, red/green/omission regression, focused
Windows race tests, native case-alias guard, Make build, package vet and
eight-file GoLand inspection. Two intentional test-setup duplicate fragments
retain the existing exact Qodana exclusion. Whole-tree local formatting and a
broader export selection exposed host limitations retained in the Windows
record; changed-file formatting passes. Native busy/modal close exit 0; all
owned app/hold processes were released. Fresh remote gates must pass on the
pushed head and are pinned in the PR's final disposition. Actual delegation:
one read-only scout; all implementation/review/fixes T0.

#### PR 66 review loop

The user invoked the repository review loop after publishing `1bfca4a` in
[PR 66](https://github.com/frathe/picfetch/pull/66). This authorizes scoped fix
commits/pushes, review replies/resolution and fresh code/security review requests;
no merge or release. Review base is `6c0db304e18f8591ae340630a693b0d3e5919635`.
T0 owns both standards/spec assessment and every fix. Initial CI/CodeQL and
native jobs passed; Codex completed code/security reviews without findings and
left a thumbs-up. There are no review threads, including unresolved older ones.
Qodana completed but reported eight findings; its post-suppression SARIF must be
assessed, not waived based on the job's successful exit.

Scope: inspect the eight static-analysis findings, use the existing accepted
policy/real-viewer test seams for any behavior repair, inspect changed files
locally, and run focused regressions. The full suite runs in GitHub CI under
the review-loop rule; do not repeat the broad local race suite. Record findings,
dispositions and evidence in `docs/command-admission-pr-66-review-2026-09-27.md`.
After any fixes/dispositions, obtain fresh code/security review and passing
checks for the newest pushed commit. Physical-input acceptance stays separate.

Budget: one bounded read-only scout assignment can extract the existing native
artifact test outcomes while T0 assesses Qodana and source changes. G1-G5:
under 25 lines; verify cited run/test events with `jq`; zero edits to sources;
bounded cross-platform artifact facts; distinct from T0's review/fix context.
No delegated review or fixes. T0 retains final acceptance and SARIF review.

Completed source round at `4674cca`: focused race regressions, package vet,
shard/exclusion checks and four-file GoLand inspections pass. The initial eight
SARIF findings are addressed with behavior-preserving cleanup and narrow,
justified suppressions. Fresh Codex code/security reviews have no findings,
all CI/CodeQL/FOSSA checks pass, and T0 inspected the fresh Qodana SARIF with
zero results and a successful invocation. Exact run/artifact IDs and source
hashes are in the [review record](../docs/command-admission-pr-66-review-2026-09-27.md).
The documentation-only evidence follow-up carries forward local inspection
evidence from `4674cca`; final-head remote checks and reviews must be repeated
and retained in the PR's final disposition comment. No merge/release; ticket 10
and overall physical/native desktop acceptance remain open.

#### Portable tracker and PR publication

The user approved moving the reviewed planning Markdown, committing, pushing
this branch and opening a PR. T0 will move exactly the spec, interview and twenty
current/draft tickets to `docs/ma-028`, redact the interview's home path, update
links and add an index. Raw artifacts remain ignored under `.scratch/ma-028`.
No product behavior or acceptance requirement changes; ticket 10 stays open.
Verify the exact 22-document move, redaction, tracked-only Markdown link targets,
unchanged source/workflow hashes and `git diff --check`. Carry forward the full
Make/inspection evidence from `ba43d1b` for this documentation-only change.
Then push the existing branch and open a PR against the repository default.
This is not authorization to merge, release or claim pending CI/native passes.

Budget: one bounded read-only scout can inventory PR-triggered workflow/check
names while T0 owns the document move and privacy/link verification. G1-G5:
under 25 lines; verify returned citations with one targeted `rg`; zero edits;
bounded cross-workflow trigger sweep; distinct from T0's document work. No
delegated review or implementation. No repeat broad local race run for docs.

Local migration verification passed: all 22 documents compared with their
originals after only the approved substitutions, then current ticket status
was updated separately. The 23-file Markdown-only directory includes its new
index; all 117 scoped relative links resolve through tracked files. Credential/
home-path scans, raw-evidence checksums and staged whitespace checks passed.
Source/workflow hashes match `ba43d1b`. T0 verified the scout's PR-trigger
citations: opening against `main` starts CI, CodeQL and Qodana, but is not proof
that those checks have completed successfully.

#### Native CI selection implementation

The user subsequently requested continuing ticket 10 until human interaction
is needed. This accepts the proposed local nativeguards command-boundary tests;
it does not authorize publishing the branch. Continue within this existing
plan. T0 owns all design, TDD, implementation and review.

1. Add a focused `command-admission` native suite: case-alias export on the
   current supported host, plus the Copy key-equivalent guard on macOS. Preserve
   the existing full-package suites and fail on missing/skipped required guards.
   Files: `scripts/nativeguards/main.go`, existing `main_test.go`.
   Verify: `go test -race -count=1 ./scripts/nativeguards`, and execute the real
   new suite on temporary case-insensitive Linux storage; reject a real ext4 skip.
2. Wire that suite into Windows/macOS CI, retaining JSON evidence even on failure
   and excluding Linux golden tests. File: `.github/workflows/ci.yml`; guard the
   job-specific wiring in the existing command-boundary tests before editing it.
3. Run negative controls, inspect changed code/workflow files with GoLand and
   run one final `make verify`. Record native-platform/physical-input and PR/CI
   limits, update tracker/evidence/todos, and commit locally. No new dependency,
   source package, production viewer behavior or platform waiver.

Budget: one bounded read-only scout assignment may inspect native UI test
startup/linkage and existing CI toolchain prerequisites while T0 implements
the runner. G1-G5: under 25 lines; verify source citations; zero edits; bounded
cross-file platform prerequisite sweep; separate from T0's command runner.
No delegated review or implementation. Full suite: once after local iterations.

Implemented locally on base `574cf82`. Command-boundary tests were observed red
for the absent suite, forbidden HEIC exemption and missing per-job CI steps,
then green. Overlay omissions of the required inventory and focused filter
also fail. The real new runner rejects an ext4 skip and passes the existing
export regression on actual FAT16. GoLand inspected both changed Go files and
the workflow with weak warnings included: no findings. Windows runner cross-vet
passes. Final `make verify` exited 0, including every Docker race partition;
the
[Linux qualification record](../docs/command-admission-linux-qualification-2026-09-27.md)
records the exact evidence and remaining platform/publication gates.

The workflow test reuses already-pinned `go.yaml.in/yaml/v3` v3.0.5, test-only.
Its existing module LICENSE/NOTICE were reviewed (MIT and Apache-2.0); no new
module/version, shipped dependency closure or notice-delivery change is made.
The user's cross-desktop portability request is assessed in the
[privacy and migration audit](../docs/ma-028-portability-audit-2026-09-27.md).
The subsequently approved curated move is complete; raw artifacts stay local.

Final gate artifacts: `.scratch/race-runs/20260927T113110Z-p0dfVe` and
`.scratch/ma-028/native-ci-2026-09-27`. The ordinary suite's existing environment
skips, including ext4 case aliasing, are retained rather than treated as native
passes; the focused FAT runner provides that case's executed observation.
No production viewer code changed during this follow-up. Inspection hashes
still match the final source/workflow, and current Markdown links resolve.

From `75fd69e`, T0 continues ticket 10's open gates. First run the existing
case-alias export regression against an isolated temporary case-insensitive
filesystem; accept only an executed PASS, not a skip. The first NTFS attempt
could not supply that environment; actual FAT16/vfat subsequently did.
No existing disk, system configuration or production code changes.
Verify with the compiled UI test binary, `TMPDIR` on that filesystem, and
`-test.run '^TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset$' -test.v`.
Record the driver/options, test output and normal unmount. Source inspection
and full-suite evidence carry forward only while code is unchanged.

One read-only scout mapped CI/nativeguard selection for the outstanding
case-alias and macOS menu assertions. G1-G5: prompt under 25 lines; verify
file:line conclusions with targeted reads; zero edits; bounded cross-file
workflow/test-selection search; separate from T0's filesystem execution.
T0 retains acceptance and review. Budget: one scout, no new broad suite for
evidence-only work. At that handoff GitHub had no PR/runs for this branch;
publication still awaited permission. Native physical-input and
unavailable OS checks stay open.

Completed: the existing case-alias guard passed once normally and five times
with race instrumentation on FAT16, without skipping; the owned image was
unmounted and its loop device detached. Source is unchanged at `75fd69e`.
T0 verified the scout's selection finding: neither native CI package list
includes root UI, so the case-alias and Darwin Copy-menu assertions are absent.
A focused suite and command-boundary guard tests were proposed at that handoff
and subsequently implemented above; publication permission followed with the
portable tracker request. That evidence-only follow-up used one
read-only scout spawn, zero code edits, focused tests only, T0 review. Evidence:
[Linux qualification](../docs/command-admission-linux-qualification-2026-09-27.md).

### Authorized Linux reset repair (2026-09-27)

The user's "continue the work" authorizes repairing the confirmed defect.
T0 owns the fix and review within ticket 10. Retain the existing accepted
real-viewer input seam plus the diagnosed native geometry/state loop. Add an
instance-owned OS-unmaximize seam, initialized to the existing winpos operation,
so the real Escape path can be tested against a window-manager boundary that
rejects Resize while maximized; canvas size alone is explicitly not the oracle.

- [x] Red: real Escape/reset leaves the simulated native window maximized even
  though its logical canvas changes; retain the already-red native desktop loop.
- [x] Green: dynamic reset restores ordinary WM maximization before requesting
  the compact welcome size. Preserve Grid/Explorer-owned restoration, fixed-size
  behavior and ordinary load/zoom policy; no new worker or dependency.
- [x] Verify: focused reset/static-size/command-admission tests; native ordinary
  and Grid maximize/Escape, fixed-size control and original multi-image sequence;
  changed-file GoLand inspections; one new final `make verify` for changed code.
- [x] Land: update ticket/spec/evidence/todos, commit locally, retain external
  platform/CI limitations. No push, PR workflow, merge or release is authorized.

Expected source scope: `viewer.go`, `build.go`, existing `reset_test.go`, shard
manifest; extend only if the native feedback loop demonstrates a further need.
No new test file or package is planned. One read-only scout assignment can
inventory existing static-size and owned-maximize regressions while T0 builds
the new boundary test. G1-G5: bounded prompt under 25 lines, source/test citations
verified with `rg`, zero edits, small test-inventory scope distinct from T0's
native-window fake and fix. No implementation or review is delegated.

Final-gate budget exception: the first complete Docker run exposed clipboard
surface-pixel fixtures rejected by the new frame-count fact and a direct
Close Files startup-reset regression. Both received focused fixes and green
regressions. The ui-2 panic prevented later tests in that partition, so a
second complete Make verification is required rather than treating the
partial run as coverage. T0 owns both runs and their findings.
Before that rerun, an additional red completion assertion caught Copy/Copy Path
menus remaining disabled after region copying. The existing clipboard result
queue now restores availability before completion; success, cancellation and
terminal no-UI shutdown regressions passed. No new worker was introduced.

Use the accepted pure-decision and real-viewer test seams from the specification.
Work through vertical red/green slices in ticket order; T0 owns integration,
review and fixes. Native platforms without runnable desktop access remain
unverified, with ticket 10 open rather than inferred from unit tests.

Read-only reconnaissance support: one bounded scout (two assignments) maps accelerator/editor
dispatch and existing tests while T0 implements Save admission. G1-G5: bounded
question under 25 lines; verify citations with `rg`/targeted reads; zero edited
files; small input-routing scope; no duplicate of T0's Save context. No review
or implementation delegated. Budget: one scout spawn, two read-only assignments, no support writes. The second
assignment audited main-canvas modal creators/notifications across features:
G1 <=25 lines, G2 citations verified with targeted reads, G3 zero writes,
G4 bounded notification sweep, G5 separate from T0's navigation implementation.

Linux qualification follow-up: one additional bounded read-only assignment to
the same scout checked isolated launch storage and clipboard hold seams. G1-G5:
under 25 prompt lines; source citations verified; no edited files; bounded
launch/clipboard scope; distinct from T0's desktop execution. T0 owns all native
input, review and evidence updates. No implementation/review was delegated.

Source of truth: the [accepted specification](../docs/ma-028/spec.md),
[design](../docs/command-admission.md) and
[ownership ADR](../docs/adr/0003-shared-command-admission.md). D1-D7 remain settled.
The parent specification and backlog issue are not modified by ticketing.

## Published tickets and blocking edges

The implementation request accepted the breakdown. Tickets are published under
`docs/ma-028/issues/`; the original drafts are retained as planning history.
Tickets 01-09 are **done/resolved** at commit `9dc3a81`, with all 46 implementation
checklist items checked against the recorded local evidence. Ticket 10 is
unblocked and `ready-for-human` after repair of the confirmed Linux reset defect:
its completed preparation, scoped Linux checks and repair are checked separately.
Case-insensitive export now passes on FAT16, including five race repetitions.
Focused native guards now pass in Windows amd64 and both macOS CI architectures.
Fresh remote review/CI/analysis gates pass at `4674cca` as recorded above.
Physical-input and full Windows/macOS desktop execution remain open;
the partial evidence does not resolve ticket 10.

| Ticket | Status | Delivers | Depends on | Owner / model / effort |
| --- | --- | --- | --- | --- |
| [01](../docs/ma-028/issues/01-policy-and-save.md) | Done / resolved | Shared admission, proven through Save Changes without losing an unavailable selection | None | T0 Codex Lead / GPT-6 Astra / extra high |
| [02](../docs/ma-028/issues/02-clipboard-and-editing.md) | Done / resolved | Correct text, region, Grid and image copy routing, including accelerator delivery | 01 | T0 Codex Lead / GPT-6 Astra / extra high |
| [03](../docs/ma-028/issues/03-open-close-favorites.md) | Done / resolved | Consistent Open, Close Files and Favorites admission, including external delivery | 01 | T0 Codex Lead / GPT-6 Astra / high |
| [04](../docs/ma-028/issues/04-file-actions.md) | Done / resolved | Export, Trash, reveal and wallpaper with prompt ownership and captured subjects | 01 | T0 Codex Lead / GPT-6 Astra / high |
| [05](../docs/ma-028/issues/05-window-and-help-entry.md) | Done / resolved | Consistent ordinary window, comparison, Settings and Help entry | 01 | T0 Codex Lead / GPT-6 Astra / high |
| [06](../docs/ma-028/issues/06-map-and-mosaic-entry.md) | Done / resolved | Explorer, Location Map and mosaic entry with safe preparation and retained visits | 01 | T0 Codex Lead / GPT-6 Astra / extra high |
| [07](../docs/ma-028/issues/07-sort-duplicates-search.md) | Done / resolved | Sort, duplicates and Find more like this across restricted visits | 01 | T0 Codex Lead / GPT-6 Astra / extra high |
| [08](../docs/ma-028/issues/08-navigation-and-presentation.md) | Done / resolved | Navigation, rotation, zoom and presentation commands with local input preserved | 01 | T0 Codex Lead / GPT-6 Astra / high |
| [09](../docs/ma-028/issues/09-converge-and-verify.md) | Done / resolved | Complete migration, removal of superseded policy, deterministic verification | 02, 03, 04, 05, 06, 07, 08 | T0 Codex Lead / GPT-6 Astra / extra high |
| [10](../docs/ma-028/issues/10-native-qualification.md) | Open / ready-for-human | Linux reset and Windows prompt-focus repaired; scoped native checks passed; complete physical/native desktop acceptance | 09 | T0 Codex Lead + native desktop operators / GPT-6 Astra / high |

The graph is `01 -> {02,03,04,05,06,07,08} -> 09 -> 10`.
Those middle tickets share a prerequisite, not permission to edit concurrently.
They touch common root input/menu code. Work serially in the displayed order,
one vertical behavior slice per ticket. Shared-file ownership is a scheduling
constraint, not an invented dependency edge. Recheck each ticket against the
then-current tree and retain its predecessor evidence.

01 includes the small prefactor and a real Save Changes path; there is no
standalone infrastructure-only milestone. 02-08 are vertical behavior slices.
09 is the contract/removal step after the migration batches; 10 is the platform
acceptance gate. All intermediate batches must remain buildable and green for
their applicable tests; keep existing admission for not-yet-migrated commands.
The first slice is not completion of MA-028.

## Agent and model selection

The recommendation is task-specific judgment, not a benchmark claim. The live
Codex tools expose `gpt-6-astra`, `gpt-6-sol` and `gpt-6-luna`. Official
[model selection guidance](https://developers.openai.com/api/docs/guides/model-selection)
positions Astra for demanding analysis, Sol for everyday coding and Luna for
scoped work; the [model catalog](https://developers.openai.com/api/docs/models)
confirms the model IDs. These pages were consulted on 2026-09-27. No API access,
pricing assumption or model switch is needed to use this plan.

Best primary agent: a Codex Lead with Go/Fyne context, using GPT-6 Astra.
Each migration ticket crosses admission, actual invocation and menu presentation;
several also cross asynchronous or native dispatch. The repository assigns
architecture, cross-cutting work, review and fixes to T0. Calling a whole ticket
an ordinary `go-expert` subtask would evade that ownership rule.

| Role | Codex recommendation | Repository's Claude harness equivalent | Use here |
| --- | --- | --- | --- |
| T0 Lead | `gpt-6-astra`, high; extra high for the rows marked above | Opus 5 Lead, per working agreement | All design, family integration, review, fixes and acceptance |
| T1 Implementer | `gpt-6-sol`, medium | `go-expert`, Sonnet 5 | Optional bounded Go implementation after T0 fixes its interface and tests |
| T2 Mechanical | `gpt-6-luna`, medium | exact-spec general-purpose, Haiku 4.5 | Optional table expansion needing comprehension; use scripts for deterministic manifest/catalog edits |
| T3 Scout | read-only agent, `gpt-6-luna`, medium when selectable | `Explore` | Bounded factual route/test search, returning citations |
| Native operator | Human with the actual desktop; T0 interprets evidence | Same | Physical accelerator, native menu and close behavior on each platform |

The Claude names are the repository's recorded mapping, not a fresh claim about
provider availability. The named `.claude/agents` are not live Codex agent types.
In particular, do not assign this plan to `refactor-planner` wholesale: its local
definition refers to an older Phase-2 plan and it remains a T1 role under the
working agreement. No standing tier map or agent definition is changed here.

Implementation defaults to zero subagent spawns per ticket. An optional support
task must first record G1-G5: <=25-line cold-start prompt, one verification
command, <=3 non-overlapping files, smaller necessary context, and no duplicate
of the Lead's hot context. Scripts precede spawns. At most one implementation
support spawn per ticket and two concurrent scouts/support tasks overall;
never concurrent edits to shared root files. After two failed delegated gates,
T0 resumes inline. No delegated review or post-review fixer.

## Shared contracts established by 01

01 pins these private root-UI boundaries before another ticket starts:

- `decideCommand(commandRequest, commandContext) commandDecision`: pure,
  deterministic, value-only admission, refusal reason, required yield and target
  kind. The request distinguishes command, intent and route.
- `(*viewer).commandContext() commandContext`: UI-thread observation of existing
  owners, without payload capture, cancellation, I/O or worker admission.
- `(*viewer).queryCommand(commandRequest) commandDecision`: fresh observation
  and pure decision only, suitable for menu availability.
- `(*viewer).admitCommand(commandRequest) (commandDecision, bool)`: fresh
  observation and decision, then existing refusal feedback or the required
  yield. The boolean means the caller may proceed; it never executes arbitrary
  callbacks or captures file/image payloads.

Use distinct facts for visible surface, retained visit, modal/editor/menu input
owner, busy region copy, ordinary clipboard work, operation admission, concrete
capabilities and prospective subject. Do not store widgets, callbacks, files,
pixels, mutable feature state or a generic active-mode flag in the policy.
01 records the concrete value/enumeration fields and outcome precedence after
its failing tests demonstrate the distinctions. Later tickets extend cases
without replacing these boundaries or introducing a runtime command catalogue.

Menu availability remains data in the existing `menus.State` snapshot, built at
one root site. Introduce a `menus.Availability` value with named availability
fields for migrated actions; retain presentation facts for labels/checked states.
Do not export root command IDs to features. Favorites/Help keep their own
construction and accept narrow, intent-specific root admission/presentation
adapters. Their precise adapter signatures are owned by 03/05 and consumed
within those tickets, not speculative prerequisites for another ticket.

An invocation must pass admission before destructive yielding or operation-starting
capture. An admitted action captures its established subject exactly once.
An asynchronous admission point repeats policy plus its operation's token and
identity checks. Prompt-owned controls, already admitted feature-local controls,
committed writes and housekeeping are classified separately; do not route them
indiscriminately through the fresh-command gate.

## File map and verification by ticket

This is a locator map, not permission for adjacent rewrites. Tickets describe
behavior; implementation updates this map if the current tree requires a move.
Tests extend the existing root harness and affected feature suites. New root
tests also require exact shard entries/counts and exact Qodana test exclusions.

| Ticket | Main implementation locations | Test / contract focus | Spawn budget; review; full suite |
| --- | --- | --- | --- |
| 01 | New private policy/adapter files in `internal/ui`; `save.go`, `menu.go`, `shortcuts.go`, `menus/menus.go` | Shared contracts above; Save capability before yielding; pure queries and fresh invocation | 0; T0 task gate; no |
| 02 | `clipboard.go`, `clipboardwork.go`, `copyfiles.go`, `copyselection.go`, `batch.go`, `shortcuts.go`, `actionmenu.go`, `menu.go`, relevant native menu adapter | Copy target and editor intent, positive accelerator delivery, busy transitions | 0; T0 task gate; no |
| 03 | `openfiles.go`, `openwith.go`, `drop.go`, `viewer.go`, `session.go`, `shortcuts.go`, `favorites`, `menu.go` | Open/Close/Favorite routes; chooser identity and modal refusal | 0; T0 task gate; no |
| 04 | `save.go`, `export.go`, `exportoptions.go`, `batch.go`, `reveal.go`, `wallpaper.go`, `filework.go`, input/menu adapters | Owner controls versus new commands; committed writes survive changing admission | 0; T0 task gate; no |
| 05 | `windowmenu.go`, `compare.go`, `keys.go`, `shortcuts.go`, `menu.go`, `features.go`, `info.go`, `tunnel.go`, `help`, Settings entry adapter | Show/toggle difference, comparison Help, actual feature links/menu entries | 0; T0 task gate; no |
| 06 | `explorer.go`, `locationmap.go`, `mosaic.go`, `keys.go`, menu/Host adapters | Setup/preparation revalidation; captured sources; retained map/cohort visits | 0; T0 task gate; no |
| 07 | `actionmenu.go`, `sort.go`, `browsing.go`, `visualsearch.go`, `keys.go`, `shortcuts.go`, menu/Host adapters | Restricted browsing, duplicate preparation, captured search reference/generation | 0; T0 task gate; no |
| 08 | `keys.go`, `viewer.go`, `rotate.go`, `slideshow.go`, `info.go`, `actionmenu.go`, relevant EXIF/Grid/display adapters | Typed/local versus EXIF navigation; reset/zoom/rotation; Escape and automatic advance | 0; T0 task gate; no |
| 09 | All migrated adapters; superseded wrappers/state predicates; tests; `ARCHITECTURE.md`, `todos.md`, this evidence record | Complete inventory and structural review; all deterministic gates | 0; T0 final deterministic review; once |
| 10 | Native evidence/runbook and this record; any discovered fixes remain T0-owned | Actual desktop input on Linux/Windows/macOS and latest-revision reconciliation | 0; T0 final acceptance; only repeat affected gates if code changes |

For every migrated command, record actual menu, accelerator, registered shortcut,
plain/modified key, bare user entry, Host/link, drop/OS and async admission routes.
Each row states observation owner, effect owner, intent/target, applicable test
case, and a reason for every non-applicable route. Complete a family's rows
before accepting that family; a `RunCommand` wrapper call is not bare-entry
coverage. The inventory is evidence, not runtime registration.

The following concrete command groups seed that ledger. Split grouped verbs
into individual evidence rows when their routes or outcomes differ:

| Ticket | Commands / actual route anchors | Observation and effects stay with |
| --- | --- | --- |
| 01 | Save Changes: File callback, Cmd/Ctrl+S registration/accelerator, bare `saveRotation` | display capability/capture; existing save/file-work lane |
| 02 | Copy; explicit Copy image; Copy Path; activate/repeat Copy Selection; region confirm/cancel; Select All | focused editor, region feature, Grid targets, display and clipboard owner |
| 03 | Open chooser/dropzone; dropped URIs; OS-open/startup deliveries; restore previous session link; Close Files; Favorite open/index shortcut, add/overwrite/manage/remove and direct Host entry | chooser/scan lifecycle, root collection and Favorite feature/storage; dialog controls stay owned |
| 04 | Export prompt; admitted format/option confirmation; Trash request and owned confirmation; reveal menu/shortcut/info link; ordinary and mosaic wallpaper entry; Save completion regression | export/deletion prompt owners; captured targets; file/wallpaper work; EXIF/mosaic committed notifications |
| 05 | Viewer; Grid Show/G toggle; Picture-frame Show/P toggle; compare selected; EXIF E/menu/info link; Settings; Manual F1/menus/About/Spiral links; About; Release Notes; Licenses; Discussions; Finis; Hypno Spiral secret/gesture entry | existing window/feature state and explicit root composition; separate-window local controls remain local |
| 06 | Explorer Shift+S/menu/bare/retry; Location Map Shift+L/menu/bare/return; map image/cluster and Explorer cohort/unassigned Host visits; mosaic Shift+M/menu/bare | feature-owned setup/work/camera; root preparation/collection transitions and captured mosaic source pool |
| 07 | Sort selected order / S cycle; hide duplicates D/menu; browse variants Shift+D/menu; Find more like this Cmd/Ctrl+Shift+L/menu/bare; search Back/Exit/save-results callbacks and Explorer selection-analysis callbacks classified at the ownership seam | file sort, duplicate model/Grid, search generation/reference, existing local feature controls |
| 08 | Next/previous/first/last image; EXIF StepImage and Grid image-open Host routes; rotate both ways; reset+fit 0; actual size 1; zoom +/- and pan; merge M; info I; shuffle Shift+P; interval Up/Down; settings setters; Escape and native close | display/zoom, Grid, EXIF, slideshow and existing ordered dispatch; background advances/loads are not new user commands |

No new shortcuts/menu items are implied by this ledger. Mark absent bindings
N/A with that reason. Startup option application, automatic update UI, render
notifications, settings-owned controls and feature-owned confirmation are not
automatically reclassified as unrelated main-window commands.

## Acceptance commands

These implementation gates supersede the earlier ticketing-only status. Keep
the spec's top-level `TestCommandAdmission*` names, adding family subtests under
`TestCommandAdmissionRoutes` (`save`, `clipboard`, `open`, `files`, `windows`,
`maps`, `search`, `navigation`). Other top-level matrices may use those same
family subtest names as needed. A command exit of zero with absent cases/skips
is not evidence: retain verbose case names and compare against the inventory.

| Gate | Command and required evidence |
| --- | --- |
| V01 | Run both V01 commands in ticket 01: pure policy/queries/yield/Save regressions, then route subtest `save`; include actual menu/shortcut/bare cases and observed D7 red before green |
| V02 | Run both V02 commands in ticket 02: editing/busy/region/clipboard regressions, then route subtest `clipboard`; prove positive editor delivery and region/ordinary clipboard distinction |
| V03 | Run both V03 commands in ticket 03: modal/async/Open/Favorites/comparison-refusal regressions, then route subtest `open`; replace obsolete drop-cancels-delete assertions with accepted D4 behavior |
| V04 | Run both V04 commands in ticket 04: modal/async/file-action regressions, then route subtest `files`; include committed writes after modal entry |
| V05 | Run both V05 commands in ticket 05: window/comparison/modal/busy regressions, then route subtest `windows`; exercise actual feature-created Help items/links and bare entries |
| V06 | Run both V06 commands in ticket 06: visits/async/Explorer/Location Map/mosaic regressions, then route subtest `maps`; hold preparation then change admission |
| V07 | Run both V07 commands in ticket 07: visits/async/search/sort/duplicate regressions, then route subtest `search`; assert generation/reference and retained-visit cases |
| V08 | Run both V08 commands in ticket 08: yield/visits/step/region/Escape regressions, then route subtest `navigation`; assert EXIF versus typed navigation and local key ownership |
| V09 | List guards with `go test -tags no_emoji,nodynamic ./internal/ui -list '^TestCommandAdmission'`; run `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmission'` and every spec AC regression command; inspect `git diff <recorded-implementation-base> -- internal/ui`; run `make check-test-shards`, `make check-qodana-test-exclusions`, `make verify` |
| V10 | `make run` on each native Linux/Windows/macOS desktop, followed by all four native scenarios in the spec; retain OS/arch, Fyne version, revision, input route and visible/payload outcomes. Launching alone does not pass |

Go splits `-run` patterns on slashes. The tickets therefore use two invocations:
one for top-level regressions and one for the exact family subtest. Retain that
form when extending their gates; do not insert slash-qualified names into the
top-level alternation.

AC ownership: AC1/AC7 start in 01; AC4 in 02; AC3/AC5/AC6 and AC2 are extended
by every applicable family; AC8 in 05-08; AC9 in 03/04/06/07; AC11/AC12 in 09;
AC10 and the final acceptance check in 10. All ACs are mandatory at completion.

At every task gate: observe intended red then green for changed behavior, keep
OS seams stubbed, settle actual queues/signals, inspect all changed code files
including weak warnings, and record actual command output and findings. Follow
the local inspection guide before configuring Qodana/GoLand. Re-run affected
inspections after fixes. Keep existing justified suppressions. Current Qodana CI
is enabled; incomplete/licensing-error scans remain unverified and only a fresh
post-suppression SARIF can establish its result. `make verify` uses native
Linux/amd64 Docker; do not substitute emulation or weaken isolation tests.

Start a native accelerator feasibility check during 02 on any available desktop;
do not wait until 10 to discover menu interception. The final three-platform run
still targets the integrated revision. If a platform is unavailable, record it
as unverified and leave 10 open. If qualification requires a new dependency or
native decoding change, record the concrete obstacle under D1 before expanding
scope. An instance-owned bounded qualification seam is permissible; global
mutable seams and timing-based guesses are not.

Record the actual implementation base at the start of 01. At 09 update the
architecture map for any package/file ownership movement, the backlog and this
record. At 10 reconcile all evidence with the tested revision; retain earlier
unchanged-code evidence explicitly, and repeat affected tests/inspections/native
scenarios after fixes. The later `/implement MA-028` request explicitly authorizes local commits.
The later portability request also authorizes pushing and opening a PR, not
merge or release. Move the accepted implementation plan to
`finished_refactorings/` only after its gates pass and the work is accepted.

## Ticketing evidence and cost ledger

This section records the earlier ticket-preparation turn. Current execution
evidence and the implementation ledger follow it.

Read the working agreement, architecture, full spec/interview, design/ADR and
domain vocabulary; checked the real menu, shortcut, key, preparation and Help
entry paths. A read-only scout enumerated Open/Favorite/clipboard routes and
test anchors while T0 examined the other adapters. G1-G5: bounded factual prompt,
source-citation oracle, zero edited files, smaller route/test scope, no duplicate
of T0's source-reading scope. The scout inherited this session's model; that is
an actual execution fact, separate from the cheaper T3 recommendation above.
T0 retains all ticket design and review.

Source checks confirmed chooser admission/delivery are separate
(`internal/ui/openfiles.go:18`, `:33`); Favorite menus use the generic runner
while numbered opens and Manage rows have distinct entry paths
(`internal/ui/favorites/favorites.go:176`, `:193`, `manage.go:274`). The pinned
Fyne driver tries menu accelerators before focused widgets and invokes matching
actions without checking Disabled (`internal/driver/glfw/window.go:862`, `:886`
in the local v2.8.0 module). These are source facts, not native test results.

Documentation verification passed: ten numbered draft files, required fields,
all relative links, the expected acyclic blocking graph, acceptance checklists,
test-selector regex syntax and whitespace. Confirmed that no published `issues/`
directory exists. `git diff --check` passed; only this plan and `todos.md` are
visible Git changes, with the ten local drafts retained under ignored `.scratch/`.

| Phase | Spawn budget / actual | Review | Full suite | Result |
| --- | --- | --- | --- | --- |
| Ticket preparation | 1 / 1 read-only scout | T0 consistency check | No | Drafts prepared for breakdown approval |
| Implementation 01-08 | 0 each / not started | T0 per task | No | Pending |
| Convergence 09 | 0 / not started | T0 | Once | Pending |
| Native acceptance 10 | 0 / not started | T0 + native operators | Conditional on fixes | Pending |

No application test, native run or code inspection is claimed for this
documentation-only preparation. At that stage the tracker was ignored under
`.scratch/`. The approved portability follow-up now tracks its planning Markdown
under `docs/ma-028`; raw artifacts remain ignored.

## Implementation execution ledger (2026-09-27)

| Phase | Actual delegation | Review and verification | Result |
| --- | --- | --- | --- |
| Implementation 01-08 | One read-only scout, two bounded assignments; no implementation support | T0 owned vertical red/green slices, integration and all fixes | All six families migrated; superseded wrappers/policy removed |
| Convergence 09 | None | T0 structural review; 124 focused top-level passes, nine new guard groups; all 68 changed Go files inspected without findings | Final `make verify` passed, including every Docker race partition; existing platform skips are explicitly retained in the evidence |
| Native acceptance 10 | Five further read-only scout assignments (launch isolation; pinned driver resize; neighboring tests; native CI selection; native test startup/toolchain), the last two on one new scout; no implementation support | T0: scoped native checks, Escape/maximize TDD repair, repeated native/static controls, inspections/Make gates, case-alias FAT16 pass plus five race repeats, focused CI suite with red/green and mutation controls | Linux reset, case-alias and local CI-selection gaps closed; physical operator, Windows/macOS execution and external CI gates remain open |
| Portable tracker / publication | One bounded read-only assignment to the existing scout for PR workflow triggers; no implementation support | T0: exact 22-file move/content comparison, redaction, authoritative index/tracker pointers, tracked-only link and scope checks | Documentation-only; Make/inspection evidence carried forward from unchanged code at `ba43d1b`; push and PR creation authorized |
| PR 66 review loop | One bounded read-only assignment to the existing scout for native artifact events; no implementation support | T0: standards/spec review, eight SARIF dispositions, focused race/vet, four-file IDE inspection, fresh reviews and full remote gates | Source round `4674cca` clean; final documentation-head checks retained on PR 66; physical/native desktop acceptance still open |

The original implementation used two complete Make runs under the budget
exception above; no broad local race run was used during individual red/green
loops. The subsequent repair adds the single final gate below. The original
[evidence record](../docs/command-admission-verification-2026-09-27.md) names both
retained artifact directories, the exact inspection scope/hash and every
remaining gate. At that initial handoff no push or PR review loop had been
performed; both were subsequently authorized and are recorded above. No merge
or release was performed.
The initial Linux follow-up changed evidence/tracker files only. The authorized
reset repair then changed three Go files plus the shard manifest, with one
additional complete `make verify` (exit 0) and fresh GoLand inspection of all
three files, without findings. Race artifacts are
`.scratch/race-runs/20260927T103730Z-3Tubvu`; native/review evidence is in the
Linux record. Both original clipboard holds and every subsequent native test
window were closed. No new worker, dependency, package or translation was added.
