# Mascot-circle hint implementation

Status: completed and archived at Ronin's request on September 14, including the bubble-width and manual-search Escape fixes. Verification limits are preserved below. Base: `84650fbccdc89517775955d69775f79c1e5c9de8`.
Accepted contract: [spec](../.scratch/spiral-mascot-hint/spec.md) and
[three tickets](../.scratch/spiral-mascot-hint/issues/README.md).

Deliver the welcome Trane -> Finis -> localized clue -> empty manual search ->
existing Spiral discovery path, using SDD and TDD at Ronin's already accepted
viewer/canvas and timestamped recognizer seams. Route: Deep for the three UI
packages and multi-file qualification; no new subsystem or dependency.
No new artwork, workers, global clock, persistence, or Spiral behavior.

## Decisions and proof

The accepted product and testing decisions stand. Recognition uses independent
UI-thread instances, actual movement only, inclusive 20-second attempts,
ten net full turns, no outer radius, and quiet lifecycle/geometry reset.
Host geometry supplies head-relative positions normalized by its gaze dead zone.
Final tolerances and boundary evidence are recorded below.
Every AC1-AC6 remains gated by `make verify`; focused commands below provide
iterative evidence. Native desktop qualification and GoLand inspections remain
separate evidence and will be reported unverified if unavailable.

## Task graph and file map

01 -> 02 -> 03 -> final gate/review/commit. All implementation, architecture,
reviews and fixes belong to T0; hot context stays with the lead.

### Task 01 — Summon Finis
Owner: T0 inline.
Files: shared `internal/ui/widgets/circlegesture.go` and its new test;
`internal/ui/trane.go`, `components.go`, `features.go`, `help/finis.go`, new
`internal/ui/mascot_hint_test.go`; architecture, shard and exclusion records.
Depends: none.
Contract: `widgets.CircleGesture.Move(fyne.Position, time.Time) bool`,
`Reset()`; zero value ready; positions normalized to one inner radius.
`Help.ShowFinis()` is the existing singleton entry point made available to UI.
Test: actual welcome canvas movement opens one Finis only after ten circles;
timestamped I/O traces cover direction, inclusive deadline, wobble, resets,
ovals, radius variation, center exclusion, discontinuities and retry.
Verify: `go test ./internal/ui/widgets ./internal/ui/help ./internal/ui -run 'Test(CircleGesture|MascotHint|Trane|Finis)'`.
Budget: 0 implementation spawns; 2 review rounds; no full suite.

### Task 02 — Reveal the clue
Owner: T0 inline.
Files: `internal/ui/help/finis.go`, new `finis_clue.go`, existing `finis_test.go`,
`internal/ui/mascot_hint_test.go`, both translation catalogues.
Depends: 01.
Contract: each Finis view owns one recognizer and one persistent bubble.
Test: actual canvas discovery by either opening route, independent deadlines,
single visible bubble, geometry/exit/reveal persistence, close/reopen reset,
wrapped layout at supported sizes and themes, catalogue parity.
Verify: `go test ./internal/ui/help ./internal/ui -run 'Test(Finis|MascotHint|Trane)'` and `go test . -run 'TestTranslations'`.
Budget: 0 spawns; 2 review rounds; no full suite.

### Task 03 — Follow the clue
Owner: T0 inline.
Files: `internal/ui/help/finis.go`, `manual.go`, existing manual-search tests,
`internal/ui/mascot_hint_test.go`.
Depends: 02.
Contract: Help owns show/reuse, clear query/highlights, focus empty search;
click does not submit, close Finis, or activate Spiral.
Test: primary full real-viewer/canvas journey plus focused clearing/reuse;
existing manual, gaze, welcome, and Spiral entrances regressions.
Verify: `go test ./internal/ui/help ./internal/ui -run 'Test(MascotHint|Finis|Trane|Manual|HypnoTunnel)'`.
Budget: 0 spawns; 2 review rounds; no full suite.

### Final gate and qualification
Owner: T0 inline. Depends: 01-03.
Files: changed-file GoLand inspections; this evidence, todos and local tickets.
Verify: `make verify`; `make check-test-shards`; GoLand every changed code file,
including weak warnings; native small/large light/dark discovery and focus.
Review standards and spec axes from the code-review skill locally: repository
working agreement explicitly reserves all review for T0. Baseline is the recorded
starting HEAD plus our working changes. Commit only feature-owned edits on the
current branch; user invoked implement with explicit commit authorization.
Budget: one full suite; one final review per axis plus any justified fixes.

## Delegation and ledger

One read-only Scout gathers current native verification/desktop tooling while
T0 implements. G1 bounded environment question; G2 returned commands can be run
by T0; G3 no writes; G4 tooling breadth kept out of implementation context;
G5 current alternatives not yet understood. Shell identified the platform gate
but cannot alone establish available native automation. S/W: reconnaissance,
not a scripted transform or an already-written implementation.

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| Recon | 1/1 Scout | n/a | no | tooling inventory; no edits |
| 01 | 0/0 | 2 | no | canvas, traces, mutation guards |
| 02 | 0/0 | 2 | no | independent discovery, layout, locale, lifecycle |
| 03 | 0/0 | 1 | no | full viewer journey, manual clearing |
| Final | 0/0 | 3 | attempted once | platform blocked; focused race and local static/build pass |

## Execution evidence

- Starting worktree has only the preceding ticket-publication update to
  `todos.md`; this belongs to the same feature and will be retained.
- No dependency changes: shipped dependency and artwork closure unchanged.


### SDD/TDD results and calibration

- Initial Trane canvas test failed because ten circles did not open Finis,
  then passed with the shared recognizer and singleton entry point.
- Timestamped traces first exposed late/tiny orbit acceptance and missing
  reversal/center/expiry resets. They pass after implementing those rules.
- Full-surface tests exposed the restore link interrupting Trane hover and
  Finis's default four-pixel window padding excluding edge movement. The
  welcome hover overlay now tracks the whole dropzone and forwards the
  restore link's own hover logic; Finis uses an unpadded canvas.
- Independent Finis discovery and the complete click/search journey were
  each observed red before implementation, including the existing-query case.
- Bubble space is reserved before reveal. Text wraps in two lines at minimum
  width, with width/height measured for the active font and localized hint.
  Minimum-size queries do not resize the live label. Both catalogues and
  light/dark themes fit at supported minimum, default and large sizes.
- Trane's inner radius is its existing 18 * portrait-scale logical pixels;
  Finis's is its existing 24 logical pixels. There is no outer radius.
  A chord entering the inner disc also resets, even with endpoints outside it.
- Backward wobble up to 15 degrees from the furthest angle is tolerated and
  deducted from signed progress; a larger reversal resets. Tests distinguish
  14.9 from 15.1 degrees and repeated incomplete arcs from complete turns.
- Sample gaps above 90 degrees reset instead of guessing their direction.
  A 1e-6-radian numerical allowance handles float32 axis/end-point rounding
  for quarter turns and the ten-turn completion comparison. Tests distinguish
  90 from 91 degree gaps. There is no polling or expiry worker.
- Time starts at first qualifying angular movement, includes pauses, and
  accepts exactly 20 seconds; 20 seconds plus one nanosecond fails. Stationary
  time before the attempt does not count. Retry needs no cooldown.
- The Fyne test driver implements CanvasForObject by returning its last-created
  window. Its default absolute-coordinate lookup therefore silently loses
  Trane geometry once Finis/manual opens. The mascot fixture overrides only
  AbsolutePositionForObject to walk the actual owning content tree; all
  application composition, canvases and pointer/tap dispatch remain real.
  An initial broader driver adapter recursed while constructing renderers;
  it was removed. No Help mock or clock seam was added.
- Eight deliberate mutations were caught for their intended behavioral
  failures: exclusive deadline, unsigned wobble, ignored clear reversal,
  Trane lifecycle resets, Finis lifecycle resets, retained Finis on close,
  uncleared manual highlights and wrong catalogue hint. Sources were restored
  after each. Logs and the runner live in
  `.scratch/spiral-mascot-hint/evidence/`.
- Review also caught the hover overlay suppressing the restore link's
  underline/cursor. Its canvas regression was observed failing, then passed
  after forwarding the original link's hover behavior. The extra final review
  round and focused reruns exceed the original estimate for this concrete
  regression and the multi-window fixture correction.

### Verification

- Focused race command (no_emoji) covered CircleGesture, MascotHint, Trane,
  Finis, manual search, HypnoTunnel and existing welcome tap routes:
  widgets 1.489s, help 27.311s, UI 28.258s, all passed.
- After the final hover/geometry-fixture changes, the full MascotHint plus
  Trane and welcome tap regression selection passed under race in 21.399s.
- `go test -tags no_emoji . -run '^TestTranslations' -count=1`: passed (0.537s),
  covering locale parity, English identity and prohibited Unicode arrows.
- `make check-test-shards`: passed, 688 runnable root UI tests across three
  shards; all six new root tests are assigned to ui-2. Both new test files
  have exact Qodana duplicate exclusions.
- `make verify-build`: passed (format, TUF root, generated assets/notices,
  Qodana exclusions, vet and build). Repeated only after review changes.
- `make verify`: attempted once; stopped at check-test-platform because the
  selected daemon is linux/aarch64. The complete native Linux/amd64 race suite
  did not run. No worker policy or isolation test was relaxed or skipped.
- GoLand get_file_problems (errorsOnly=false), lint_files for every changed Go
  file and get_project_modules all returned tool errors. No inspection results,
  including weak warnings, are claimed. This remains pending.
- Ronin explicitly requested on September 14: "Let me do the E2E testing later."
  He later reported the E2E flow looks good apart from the bubble-width
  feedback; see acceptance below. A native accessibility probe was
  rejected/interrupted before any Pico desktop interaction; Pico did not
  execute native E2E.
- No dependency, artwork, worker, persistence schema or platform-specific
  runtime change. No application launch, push, PR, release or merge.

### Lead review and handoff

Used the code-review skill's two axes against starting HEAD and the working
feature diff. The repository process reserves all review and fixes for T0,
so both axes were reviewed locally. Standards review found and fixed the
restore-link hover regression; spec review found and fixed full-window edge
routing and corrected the multi-window test evidence. No confirmed code issue
remains. Pending external checks are listed above, rather than treated as passes.

Ronin requested marking the todo done after the wider-bubble and focused-search
Escape fixes; this plan is now archived. Commit authorization comes from his
invoked implement skill. His completion decision does not claim a new native
recheck, GoLand result or native Linux/amd64 full-suite pass.

### Native layout feedback, September 14

Ronin's screenshot showed the fixed 320-pixel bubble leaving “it)” alone on a
second line. The bubble now prefers the measured width of the complete localized
text plus label/card padding, capped by the available window width. Smaller
windows retain wrapping. Existing locale/layout and click-through tests passed
(help 0.636s, UI 1.275s); formatting and diff checks passed. GoLand inspections of
both changed code files still returned tool errors. Native recheck stays with Ronin.

### Native E2E acceptance, September 14

Ronin reported: "Else the E2E test looks good!" This accepts the tested discovery
flow apart from the previously reported bubble-width issue. That fix is committed
as `70eed645d9a2ec343e73af353fc5276933350e86`; its post-change visual recheck has
not yet been reported. This is Ronin's direct observation, not a Pico-run desktop
check or a claim that every theme/size combination was separately exercised.
The native Linux/amd64 full gate and GoLand inspection gaps remain unchanged.
His report does not replace automated qualification.

### Manual-search Escape follow-up, September 14

Ronin reported that Escape did not close the manual while search was focused.
Standard follow-up, T0 inline; the existing Help keyboard boundary and root
discovery regressions remain the accepted test seams. No new application API,
strings or background work. The root observers must accept an extended Entry.

- Task: close the manual on Escape from empty or populated/selected search,
  including after raise and reopen; retain typing/Return and leave Finis open.
- Files: Help manual/search tests; root harness, mascot discovery and tunnel
  tests; this evidence record and todos.
- Verify: `go test -tags no_emoji -race ./internal/ui/help ./internal/ui -run
  'TestShowManual|TestNewManualView|TestHelp_Manual|TestMascotHint|TestHypnoTunnel'
  -count=1`.
- Budget: no spawns, one lead review, focused regressions and local build gate;
  the previously attempted full gate still needs native Linux/amd64.
- Red: `TestShowManual_EscapeClosesFromSearch` failed for both empty and
  populated search because Escape did not close the manual. Native Fyne routes
  keys to the focused widget before the canvas callback; ordinary Entry has no
  Escape handling. This direct reproduction needs no speculative instrumentation.
- Green: the manual's extended Entry handles Escape with the owning window's
  Close callback and delegates other keys. The existing singleton close
  notification still runs once. A mouse click also refocuses the extended Entry;
  temporarily removing ExtendBaseWidget made the focus guard fail, then the
  production call was restored and the test passed.
- Focused race selection passed: help 3.035s, UI 23.946s. It includes ordinary
  search, the full mascot discovery sequence and the existing HypnoTunnel route.
  The click/refocus addition also passed its final focused race test (2.255s).
- `make verify-build` and `git diff --check HEAD` passed. No root test inventory or
  file/package ownership changed; shard assignments and exclusions stay valid.
- Lead standards/spec review found no remaining issue. All five changed Go
  files were submitted to GoLand with weak warnings included; every call still
  returned a tool error, including the updated click regression. Inspections
  remain unverified. Native confirmation of this follow-up stays with Ronin.

### Completion, September 14

Ronin requested: "mark the todo as done" after the Escape fix was committed as
`0ebad441fa1804969e7aa4f49046f086a4b0fa4f`. The feature is recorded under Done
and this plan is archived. Earlier native observations and unavailable checks
above remain the evidence; this documentation update adds no test results.

### PR 24 localization review, September 14

Codex review of 8665a2f identified the revealed search phrase bypassing the
translation catalog ([thread](https://github.com/frathe/picfetch/pull/24#discussion_r4006130235)).
This is a convention defect, not a request to translate the canonical trigger:
the accepted spec requires the phrase to remain English in every locale.
The clue's label and width measurement now use lang.L(secretPhrase); every
bundle contains the exact key with an identical value. No visible wording or
search admission changes.

TestFinisClueLayoutAndLocale now requires the catalog entry to match the
canonical search trigger. It failed before the fix with the missing English
entry, then the complete TestFinis family passed under race (24.320s). Root
TestTranslations_ checks passed under race (1.768s), covering locale parity,
English identity and prohibited arrows. No new test file or root runnable was
added, so shard assignments and exact exclusions remain current.

GoLand inspected both changed Go files including weak warnings. finis_clue.go
is clean. finis_test.go intentionally forces light and dark themes independently
of the machine preference; Fyne deprecates these helpers for ignoring that
preference. A narrow GoDeprecation suppression documents this deliberate test
use. Reinspection leaves only the existing duplicate fixture warning, covered
by the file's exact DuplicatedCode exclusion in qodana.yaml. These results cover
this follow-up's files; they do not substitute for earlier unavailable IDE calls.
Fresh native CI and Codex code/security reviews are required for the fix commit.
