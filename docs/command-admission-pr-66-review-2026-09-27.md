# MA-028: PR 66 review-loop evidence

PR: [66](https://github.com/frathe/picfetch/pull/66).
Review base: `6c0db304e18f8591ae340630a693b0d3e5919635`.
Initial head: `1bfca4aa7677c5097d53aebc3c1c8413d102adfb`.
The user explicitly invoked the review loop on 2026-09-27, authorizing scoped
fixes, commits, pushes, review replies/resolution and fresh review requests.
Merge/release and physical-input acceptance remain outside this loop.

Current status: MA-028 is accepted and all ten tickets are resolved after the
clean `e2d3b30` round below. All three native platforms are qualified. Current
acceptance supersedes historical pending-platform statements in earlier
sections. The documentation-only closure receives fresh latest-head remote
gates, recorded in PR 66's final disposition rather than another evidence commit.

## Windows follow-up from `bae3c55`

The user requested all feasible Windows ticket 10 checks, then commit/push and
another review loop. The [Windows qualification record](command-admission-windows-qualification-2026-09-27.md)
retains native outcomes, the discovered prompt-focus defect, red/green and
omission evidence, focused Windows race results, eight-file GoLand inspections,
source hashes and honest host limits. Initial PR inspection found no unresolved
threads; prior code/security reviews and CI were clean on `bae3c55`.

The repair gives in-tree Export/Delete cards one keyboard owner, preserves
their controls and releases focus on dismissal. Native repeats verify editor
copy, prompt control, maximized-window reset and native busy/modal close. T0
owns every assessment/fix. No broad local race suite was repeated. Required
fresh code/security reviews, CI/CodeQL and inspected Qodana SARIF must pass on
the new pushed head; the final PR disposition pins that commit and evidence.
At that Windows handoff, physical keyboard and full macOS desktop acceptance
remained open in ticket 10; subsequent completion is recorded below.

### Windows review round on `d662160`

[CI 36324473600](https://github.com/frathe/picfetch/actions/runs/36324473600)
passed all nine jobs, including every Linux race partition and native Windows
and both macOS architectures. Downloaded command-admission native event logs
show the required case-alias test passing without skips on all three native
jobs and the modifier-mask guard passing on both macOS jobs.
[CodeQL 36324473598](https://github.com/frathe/picfetch/actions/runs/36324473598)
passed Go and Actions analysis, with no open branch alerts. FOSSA checks passed.
[Qodana 36324473602](https://github.com/frathe/picfetch/actions/runs/36324473602)
completed successfully; artifact `10933756513`'s **end/qodana.sarif.json** has
zero results, successful invocation/exit 0 and no execution notifications.
Security review completed at 14:08:13 UTC without actionable findings.

Code review completed at 14:11:29 UTC with one confirmed P2:
[Help-owned navigation](https://github.com/frathe/picfetch/pull/66#discussion_r4115647190).
Finis's clue called gated ShowManual, then dereferenced an unconstructed manual
when the main window owned a modal. The same fresh-command gate blocked About's
manual link, Manual's Finis/Spiral actions and Spiral's F1. T0 reproduced the
nil-pointer panic and blocked routes before repair. Owned links now use their
window-local continuation; fresh main-window commands remain gated and terminal
shutdown still refuses new windows. No menu policy, worker or dependency changed.

Verification of the follow-up tree based on `d662160`:

- Focused Help/root command and prompt regressions pass; the focused race run
  (`TestFinis.*`, `TestShow.*`, `TestHelp.*`, `TestHypnoTunnel`,
  `TestCommandAdmission.*`, `TestWindowCommandAdmissionMatrix`,
  `TestExportPrompt.*`) passes Help in 35.829s and root UI in 76.176s.
  Focused vet, changed-file goimports and whitespace checks pass.
- Tests exercise the actual clue/link/search/F1 bindings, absent and existing
  manuals, shutdown and preservation of the main modal. Separate Go overlays
  restoring the old About binding and Spiral F1 binding each fail for the
  expected blocked-navigation reason. The shutdown red run also exposed a test
  cleanup double-close; cleanup now closes only a live singleton.
- Native Windows `make run` repeats use the same isolated profile and OS-injected
  input as the Windows record: open Finis, close Manual, open Manage Favorites,
  reveal/tap the clue, open Spiral from Manual and reopen Manual using Spiral F1.
  About's actual manual link also opens beneath Manage Favorites. Captures after
  both chains retain the main modal unchanged. Owned app PID 8288 and secondary
  windows were closed; the launcher exited 0.
- GoLand inspection fallback covers all nine changed Go files, including weak
  warnings. Eight are clear; `help/finis_test.go` reports one unchanged 8-line
  circle-loop duplicate, covered by its existing exact `qodana.yaml` exclusion.
  No suppression or exclusion was added. Raw reports, native captures, red/green,
  race and mutation logs are in `.scratch/ma-028/windows-2026-09-27`.

This finding requires a fix push, thread disposition/resolution and another
fresh code/security review and complete CI/CodeQL/Qodana evidence on the new
head. The final PR disposition records that exact head and its results.

### Windows review round on `bad9b82`

[CI 36325879551](https://github.com/frathe/picfetch/actions/runs/36325879551),
[CodeQL 36325879554](https://github.com/frathe/picfetch/actions/runs/36325879554)
and all FOSSA checks passed. Native artifacts `10934445136` (Windows),
`10934087321` (macOS amd64) and `10933534978` (macOS arm64) match this exact
revision; the required case-alias/Copy-menu events each have one run, one pass,
zero skips and zero failures. T0 independently counted the scout's extracted
events. [Qodana 36325879543](https://github.com/frathe/picfetch/actions/runs/36325879543)
artifact `10933822903` contains a successful final SARIF with zero results and
the exact `bad9b822c076d8e5fbf083498ac0607f152c1139` provenance. Security review
completed cleanly at 14:31:11 UTC. The previous Help thread was resolved.

Fresh code review completed at 14:34:34 UTC with one new confirmed P2:
[minimized-window cleanup](https://github.com/frathe/picfetch/pull/66#discussion_r4115719860).
Windows Unmaximize called SW_RESTORE regardless of state, so asynchronous
last-image cleanup could restore a minimized window and take focus.
[Microsoft's ShowWindow contract](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-showwindow)
documents restoration/activation;
[IsZoomed](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-iszoomed)
identifies the maximized state. The Windows adapter now checks IsZoomed before
restoring; no root command or other platform policy changes.

T0's disposable native probe calls the production Unmaximize function against
the isolated PicFetch HWND. Before the repair, both minimized and
maximized-then-minimized windows reopened and changed foreground ownership.
After repair, all four cases pass: normal unchanged, maximized restored,
minimized unchanged, and maximized-then-minimized unchanged. IsZoomed was false
in both minimized states. The actual rebuilt app still resets a maximized image
to the 536 x 379 welcome window on Escape.

The end-to-end native check uses an isolated PATH launcher to hold a successful
Recycle Bin operation on a copied fixture. After minimizing PicFetch, releasing
the held child lets last-image cleanup finish: title becomes PicFetch, the child
exits, and the file is gone, while IsIconic stays true and foreground ownership
is unchanged. App PID 30704 and held child 3760 exited; the launcher returned 0.
Native input remains OS-injected. Raw probe source, red/green state matrices,
screenshots and `minimized-trash-result.json` remain in the Windows evidence
directory. No native calls were added to ordinary desktop-free tests.

Focused race tests pass winpos in 2.880s and root UI reset/ownership regressions
in 15.310s; focused vet and changed-file goimports/whitespace pass. GoLand
inspected both changed Go files including weak warnings. Two existing unchecked
ShowWindow results were made explicitly ignored: the return reports previous
visibility, not failure. Reinspection is clear. Fresh latest-head remote gates
and a clean code/security review are still required after this fix push.

T0 owns the standards and specification assessments and all fixes. One bounded
read-only scout extracted existing native test events; T0 verified its results
with `jq`. No review or fix was delegated. Full tests run in GitHub CI; local
verification is focused, as required by the repository review-loop procedure.

## Round 1: initial head

- Codex's summary marks code review complete at 12:13:06 UTC and security review
  complete at 12:09:19 UTC for `1bfca4a`. The bot left a thumbs-up; there are no
  review threads or review findings, including older unresolved threads.
- [CI run 36317733211](https://github.com/frathe/picfetch/actions/runs/36317733211)
  passed validation, all four Linux race partitions and Linux/Windows/macOS
  native jobs. [CodeQL run 36317733263](https://github.com/frathe/picfetch/actions/runs/36317733263)
  passed Actions and Go analysis; FOSSA's three status checks passed.
- [Qodana run 36317733259](https://github.com/frathe/picfetch/actions/runs/36317733259)
  completed successfully, but its **post-suppression** `qodana.sarif.json`
  contains eight findings. Successful job status is not a clean findings gate.
  Artifact `10930824973` retains that report; CSV totals were not used.

### Standards assessment and dispositions

| SARIF finding | Assessment and disposition |
| --- | --- |
| Three `GoUnusedConst` findings: `routeDirect`, `inputViewer`, `refusalNone` in `commandpolicy.go` | The names have no references; behavior uses zero-valued fields. Replace only those names with blank enum slots and document their zero semantics. Preserve every numeric value and all admission behavior. |
| `GoImportUsedAsName`: cohort-handoff test's `context` local | Rename to `facts`, preserving the fixture and both assertions; the file also imports the real `context` package. |
| `GoSwitchMissingCasesForIotaConsts`: inner clipboard switch | Intentional subset: the enclosing case restricts entry to exactly the four handled clipboard commands. Add a statement-local suppression with that reason; do not invent unreachable command handling. |
| Same rule: policy test's surface override switch | Intentional subset: other commands retain the independent ready-viewer fixture. Add a statement-local suppression explaining the complementary case. |
| `DuplicatedCode`: `menuStateFor` cluster, original lines 83/90 | Different named command/intent-to-field mappings, not duplicated policy. Keep explicit composition required by the accepted design; a function-local suppression avoids replacing it with a runtime registry. |
| `DuplicatedCode`: `Menus.Apply` cluster, original lines 450/459 | Different named widget/decision assignments, not repeated business decisions. Preserve the readable one-to-one presentation mapping and existing change-detection tests; use a function-local suppression. |

The duplicate entries are **two clusters/four fragments**, not four independent
findings. No file-wide/global suppression or Qodana profile change was made.
Existing exact test-file duplicate exclusions remain unchanged.

### Specification assessment

No behavior defect was established by these eight findings. The cleanup retains
the existing request/context/decision contract, zero defaults, every command
and explicit root-policy/menu-presentation split. No worker, package, dependency,
translation, native bridge or acceptance requirement changed. Existing accepted
pure-policy and real-viewer/menu test seams suffice; no new test seam or
implementation-shaped assertion was introduced for static-only cleanup.

AC10 remains partial: native isolated menu/filesystem tests do not establish
physical keyboard/menu operation on all supported desktops. This pre-existing
acceptance boundary remains open in [ticket 10](ma-028/issues/10-native-qualification.md).

### Native artifacts now executed

T0 verified the raw `native-guards-command-admission.json` streams from CI run
36317733211 against head `1bfca4a`. Each required guard has one run, one pass,
zero skips and zero failures:

| Platform | Artifact ID | Case-alias export | Darwin Copy-menu assertion |
| --- | --- | --- | --- |
| Windows amd64 | `10930309923` | PASS | Not applicable |
| macOS arm64 | `10931407106` | PASS | PASS |
| macOS amd64 | `10931008783` | PASS | PASS |

The guard names are `TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset` and
`TestSetMenuItemModifierMask_ClearsDefaultCommand`. The export test uses the real
test viewer and a stub save chooser; the Darwin test constructs native menu
objects. This closes the previously missing native compilation/isolated-test
observation, **not** the physical-input requirement or full desktop acceptance.

### Focused verification and local inspection

Before cleanup, the policy/query/visit/route/window and menu-Apply regressions
passed: root UI 3.266 s, menus 0.006 s. The full `TestCommandAdmission.*` group,
window admission matrix and menu Apply tests then passed with race instrumentation
after cleanup (the broad `TestApply.*` filter also includes four updater guards).
Root UI passed in 54.043 s and menus in 1.025 s. Focused package vet,
`make check-test-shards` (717 runnables / three shards),
`make check-qodana-test-exclusions`, formatting and `git diff --check` passed.
Exact command:

```sh
go test -tags no_emoji,nodynamic -race -count=1 -v ./internal/ui ./internal/ui/menus \
  -run '^(TestCommandAdmission.*|TestWindowCommandAdmissionMatrix|TestApply.*)$'
```

GoLand's `get_file_problems(errorsOnly=false, timeout=60000)` inspected every
changed Go file before and after cleanup, returning no findings or timeouts,
including weak warnings. The active IDE profile name is not exposed by the
tool. No callable IDE-local Qodana scan interface was available; this is the
documented IDE-inspection fallback, not equivalent to Qodana's CI profile.
The difference is observable: CI found the eight results while IDE inspections
returned none even before cleanup. Fresh SARIF inspection remains mandatory.

| Analyzed file | Post-cleanup SHA-256 |
| --- | --- |
| `internal/ui/commandpolicy.go` | `97df7eefac276c409645876c292316c14bba93a30b2f5849624ee6a8250bd188` |
| `internal/ui/commandadmission_test.go` | `ab2ad9c46601a6f1a8e91f53143bcf8f4b7afac42aca3592694c78b69ed6d76e` |
| `internal/ui/menu.go` | `d4846befd0ac22095512835641e176139bbd0bef40a2f72cbc6dbf5f8ae951cd` |
| `internal/ui/menus/menus.go` | `8388ddf8a54d6ea4c6e31a7cf85ae32540c74f005e43f58f31e50897f03b5d26` |

Raw local artifacts live under `.scratch/ma-028/pr-66-review-2026-09-27/round-1`;
they remain ignored. T0 reviewed the complete cleanup diff. The required fresh
cycle on the pushed cleanup revision is recorded below.

## Round 2: cleanup revision `4674cca`

Analyzed revision: `4674cca5bab6e7fdcf7a08b9f92f00ec3433f654`.

- Codex completed a fresh code review at 12:47:17 UTC and security review at
  12:45:18 UTC. Its summary names this revision, and the bot's 12:47:20 UTC
  thumbs-up confirms no findings. T0 inspected all review comments/reviews and
  the complete thread inventory: no findings or unresolved threads.
- [CI run 36319791447](https://github.com/frathe/picfetch/actions/runs/36319791447)
  passed validation, all four Linux race partitions, Linux native guards,
  Windows tests and both macOS native jobs. No broad local race suite was
  duplicated for this review loop.
- [CodeQL run 36319791440](https://github.com/frathe/picfetch/actions/runs/36319791440)
  passed Actions and Go analysis; the branch's open code-scanning alert query
  returned no alerts. FOSSA dependency, license and security checks passed.
- [Qodana run 36319791431](https://github.com/frathe/picfetch/actions/runs/36319791431)
  passed. T0 downloaded artifact `10932610243` and inspected
  `/end/qodana.sarif.json` inside `qodana-report.zip`: **zero results**,
  `executionSuccessful: true`, and no tool-execution notifications. The eight
  original findings are therefore resolved by the documented fixes and narrow
  suppressions, not merely hidden by a successful job status. Raw evidence is
  retained under `.scratch/ma-028/pr-66-review-2026-09-27/round-2`.

Standards assessment: no outstanding actionable findings. Specification
assessment: no new implementation defect; the pre-existing physical/native
desktop acceptance boundary remains open. Ticket 10 is not resolved and the
implementation plan must not be archived yet.

The follow-up evidence commit changes Markdown only. Local source inspections
carry forward from `4674cca`, with the four source hashes above unchanged.
The review loop must still obtain fresh code/security reviews, CI, CodeQL and
an inspected Qodana SARIF for that final documentation-only head. The final
disposition comment and check runs on [PR 66](https://github.com/frathe/picfetch/pull/66)
record that head and its results; this source-revision record is not a substitute
for those latest-commit gates. No merge or release is authorized or performed.

## Windows physical-input completion

The user subsequently completed physical Ctrl+A/C in both image and Grid
naming fields on source revision `10f16a0126dde8f852a058682dd0681c61111122`.
T0 verified independent clipboard sentinels were replaced by the exact field
text, inspected full-field selection, compared unchanged image/Grid pixels and
closed the isolated literal `make run` session with exit 0. The
[Windows record](command-admission-windows-qualification-2026-09-27.md#final-physical-keyboard-qualification)
retains source/binary identity, operator/input distinctions, times and artifacts.
Windows V10 is complete. Linux was still recorded as open at that handoff;
the retrospective user confirmation below then left only macOS qualification.

The preceding source revision already passed all 17 PR checks, CI run
`36327036028`, CodeQL run `36327036051` and Qodana run `36327036045`.
Artifact `10934490868`'s post-suppression `end/qodana.sarif.json` had zero
results and successful execution, as recorded in the
[source-head disposition](https://github.com/frathe/picfetch/pull/66#issuecomment-5856936989).
Further fresh code/security reviews on that same source completed at
16:47:00/16:46:22 UTC with no findings. Both earlier review threads are resolved.

This completion changes Markdown only. Local tests and GoLand inspections
carry forward at their original recorded revisions: the initial Windows focus
repair, the Help/Spiral repair at `bad9b82`, and the final Windows adapter scope
at `10f16a0`; no changed code requires a new inspection. T0 checks the evidence,
current-status consistency, tracked local links and whitespace. No broad local
race suite is repeated. A fresh Codex code/security review, full CI, CodeQL
and inspected final Qodana SARIF remain required on the pushed documentation
head; PR 66's final disposition records their exact results without creating
another evidence-only commit and resetting those gates.

## Linux retrospective operator confirmation

On 2026-09-27 the user confirmed that Linux physical-input checks had already
been tested successfully but had not been documented, and that only macOS is
missing. Linux V10 is complete on that operator confirmation and the retained
Make-built native evidence. The original OS-injected sessions keep their actual
input/launch provenance; no new Linux run or literal `make run` is asserted.
The [Linux record](command-admission-linux-qualification-2026-09-27.md) and
active status/checklists record this acceptance explicitly.

The preceding documentation head `62f7e6f` passed all 17 checks and fresh Codex
code/security reviews with no findings. Qodana artifact `10938051613` contained
zero final SARIF results with successful execution; the
[final disposition](https://github.com/frathe/picfetch/pull/66#issuecomment-5858217739)
pins all run and review details. This follow-up changes Markdown only, so local
source inspections carry forward at their recorded revisions. Fresh final-head
remote gates remain mandatory and are recorded on PR 66. At that Linux handoff,
the plan remained active and all-platform acceptance awaited macOS.

## macOS qualification and review round on `6e63d75`

The [macOS qualification record](command-admission-macos-qualification-2026-09-27.md)
completes the final native platform on product source `41cdaa9`. Its literal
`make run` uses documented qualification-only identity/storage overlays, not
changes to input, policy or clipboard behavior. The operator physically performed
Cmd+A/C in image and Grid fields and drew the region selections. Exact clipboard
payload/pixel checks, unchanged image/Grid pixels, real menus, modal/prompt
controls, held-copy refusal/recovery and native busy/modal close passed. Both
final Make sessions exited 0. Native guards and focused root UI/widgets/menu
race tests passed. Evidence-only harness failures and launcher limitations remain
explicit in the macOS record; they are not counted as application defects.

Fresh code review at 18:56:10 UTC reported one confirmed documentation P2:
[stale macOS status in this record](https://github.com/frathe/picfetch/pull/66#discussion_r4116531909).
T0 verified that the chronological record stopped at Linux completion and still
described macOS as pending. This follow-up adds the missing handoff and current
status while retaining earlier sections as historical evidence. No product code
changes, suppression changes or new source inspections are required. Existing
GoLand inspection evidence carries forward at its recorded source revisions.

[Qodana 36342192111](https://github.com/frathe/picfetch/actions/runs/36342192111)
completed successfully. T0 inspected artifact `10939043672`,
`/end/qodana.sarif.json`: zero post-suppression results, successful execution,
exit code 0, no execution notifications, and provenance matching
`6e63d75604fbb9459c945984500bffbd63b2a00e`. This is the configured incremental
PR scope, not a replacement for the recorded changed-file IDE inspections.

At this disposition, CI/CodeQL and security review were still completing. The
documentation finding means this is not a clean final review round regardless
of their outcome. After the fix is pushed and the thread is resolved, obtain
another fresh code/security review and required CI/CodeQL plus inspected final
Qodana SARIF on the new head. Ticket 10 and the plan remain open until that
round passes. PR 66 records exact latest-head outcomes; no merge or release.

## Acceptance round on `e2d3b30`

Accepted revision: `e2d3b30c123752f976fd5f2d3515a62170bfa9d6`.
The stale-status finding is fixed and resolved with evidence in its thread.
All three review threads, including both earlier product defects, are resolved.

- Fresh [code review](https://github.com/frathe/picfetch/pull/66#issuecomment-5858834677)
  completed at 19:04:41 UTC without findings; fresh
  [security review](https://github.com/frathe/picfetch/pull/66#issuecomment-5858836929)
  completed at 19:04:58 UTC without security issues. Both name this exact head.
- [CI 36342698887](https://github.com/frathe/picfetch/actions/runs/36342698887)
  passed validation, all four Linux race partitions, Linux native guards,
  Windows tests and both macOS native jobs.
- [CodeQL 36342698907](https://github.com/frathe/picfetch/actions/runs/36342698907)
  passed Go and Actions analysis; the branch has no open code-scanning alerts.
  All FOSSA dependency, license and security checks passed. All 17 PR checks
  succeeded.
- [Qodana 36342698891](https://github.com/frathe/picfetch/actions/runs/36342698891)
  passed. T0 inspected artifact `10939079228`, `/end/qodana.sarif.json`: zero
  post-suppression results, successful execution, exit code 0, no execution
  notifications and exact `e2d3b30` provenance. Scope is configured incremental
  PR analysis. Original source-revision GoLand inspection evidence remains
  valid for unchanged code; no broad local suite was repeated.

Standards assessment: no outstanding actionable findings. Specification
assessment: all implementation and native acceptance criteria are met, with
the platform records retaining operator/automation and launch distinctions.
Ticket 10 is resolved, the plan is archived and todos/backlog status updated.
The closure changes Markdown only. Local validation checks current status,
all ticket checkboxes, tracked relative links, archive references and whitespace.
All ten tickets have resolved status and no unchecked criteria; 154 relative
links across the 32 closure files resolve, with the backlog check scoped to
its changed sections and todos checked through the Open section. No reference
to the old active-plan path remains. Whitespace checks pass.
Fresh code/security reviews, full CI/CodeQL and inspected Qodana SARIF are still
required on the closure commit itself; PR 66's final disposition pins that
latest head and the actual results without recursively resetting the gates.
No merge or release is authorized or performed.
