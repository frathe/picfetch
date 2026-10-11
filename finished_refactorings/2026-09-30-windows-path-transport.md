# Windows image paths remain data in PowerShell

## Contract and scope

Fix the first medium finding in `.scratch/sec_todo.md`: Export suggestions and
Trash targets must never enter PowerShell source. Preserve exact Unicode paths,
initial save directory/name, picker cancellation/JSON, file/directory recycling,
error propagation, inherited environment and hidden-console behavior.

Route: Deep (Windows security boundary). Lead implements and reviews inline.
Repository process sections 7/8 prohibit delegated review and hot-context fixes;
those instructions override the skill's fresh-agent investigation/review defaults.
Separate pre-patch and candidate-review passes remain required. The initial
implementation made no commits; Ronin subsequently authorized commit, push and
the GitHub review loop on 2026-09-30. Other security findings stay separate.

## Investigation

Export: `suggestedExportPath` -> `buildPowerShellSaveCmd` -> `powershell -Command`.
Trash: `Move` -> `moveWindows` -> the same script parser. Current escaping misses
PowerShell's smart-quote delimiters. The narrow boundary is command construction:
use per-child environment data, following `clipboard.copyImageWindows`, rather
than extend an incomplete character escape list. No dependency/API changes.

Sibling pass: clipboard image/file transports already use environment data.
Wallpaper uses a separate cache-generated PNG path (`ui.writeWallpaperFile`),
not the selected image filename; its legacy string helper is outside this fix.
Picker title interpolation takes shipped translations, not source paths.

## Tasks, file map and acceptance gates

1. Lead: change existing `internal/filepicker/filepicker_test.go` and
   `internal/trash/trash_test.go` to guard constant script text and exact path
   transport, including ordinary names, smart quotes, shell punctuation,
   non-ASCII text and inherited-variable collisions. Observe the old code fail.
   Command: `go test -tags no_emoji,nodynamic ./internal/filepicker ./internal/trash`.
2. Lead, depends on 1: update command construction in
   `internal/filepicker/filepicker.go` and `internal/trash/trash.go`.
   Run the same package tests; verify no loss of native error/cancellation behavior.
3. Lead, depends on 2: add native Windows regression coverage without opening
   panels or touching the recycle bin, using existing test files; wire required
   guards into `scripts/nativeguards/main.go` as needed. Cross-compile tests on
   macOS; actual Windows execution is a separate gate if unavailable locally.
4. Lead: inspect final candidate for bypasses/regressions, run focused race tests,
   vet and Windows cross-builds, GoLand inspections including weak warnings, and
   the required final `make verify` once. Record unavailable gates honestly.
5. Lead: update this evidence record, `todos.md` and the scratch finding status.

Task graph: 1 -> 2 -> 3 -> 4 -> 5. All inline: delegation gate G5 fails (existing
context), and repository section 8 reserves review for the lead. Budget: zero
spawns, one candidate review round, one full-suite attempt at final verification.
No new packages or moved files; no architecture map change needed.

## Evidence

Analyzed revision: `4a8048405bef8a1eff0a00190a2452f8b12742df` plus this
working-tree patch on 2026-09-30.

- Red: package tests failed on the original implementation because path bytes
  changed PowerShell source and no exact child-environment value existed.
- Green: `go test -tags no_emoji,nodynamic ./internal/filepicker ./internal/trash
  ./scripts/nativeguards` passed after the transport changes. This is the
  strongest local substitute for the original Windows exploit: script arguments
  stay identical across malicious/ordinary paths and child values round-trip
  unchanged. Both file/directory dispatch, missing-path errors and child-process
  failures retain existing passing coverage.
- Focused race: `go test -race -tags no_emoji,nodynamic ./internal/filepicker
  ./internal/trash ./scripts/nativeguards` passed.
- Native Windows coverage: `TestWindowsSaveTransport_PathIsData` exercises actual
  WinForms properties/JSON and synthetic OK/Cancel results; the modal call is
  replaced, so no panel opens. `TestWindowsTrashTransport_PathIsData` replaces
  only the native deletion sink with a recorder and checks exact path/method/
  recycle options; nothing is moved. Both are required by `nativeguards`' Windows
  suite. Native Windows execution is unavailable on this macOS host and remains
  unverified; the Trash native test skips here and the picker native test is
  Windows-build-selected.
- Cross-platform: host and Windows/amd64 focused vet passed; both Windows/amd64
  package test binaries compiled; Windows/arm64 package builds passed. Initial
  sandbox cache-access failures were resolved by authorized cache access.
- Local analysis: GoLand MCP `get_file_problems(errorsOnly=false)` inspected all
  six changed Go files with the IDE profile, including weak warnings. No
  actionable findings. The two existing duplicate-code weak warnings in
  `filepicker_test.go` (open/save command assertions) are covered by that file's
  existing exact `qodana.yaml` DuplicatedCode exclusion. No new test file or UI
  shard assignment was introduced. This is the IDE inspection fallback, not a
  fresh Qodana scan; hosted Qodana/CodeQL remain unverified for this candidate.
- Candidate review (lead, one round): all direct callers still feed the fixed
  command builders. Source-controlled filenames reach only environment values;
  no evaluation/interpolation is applied to those values. Method selection uses
  fixed file/directory constants. Inherited collisions are overridden in the
  child, and the parent environment remains unchanged. Ordinary Unicode/UNC
  suggestions and real admitted file/directory paths pass the portable guards.
  No confirmed bypass or compatibility regression found in the changed boundary.
- Full gate: `make verify` stopped at its platform guard: Docker reports
  `linux/aarch64`, while complete isolation/race tests require native Linux/amd64.
  No isolation test was skipped or policy relaxed to claim a pass.
- `make verify-build` passed (exit 0): formatting, TUF, exact Qodana test
  exclusions, generated assets/notices, full host vet and full host build.

## Hosted qualification and review

Signed fix commit `bb6ce5dd86d4adf0f3271fbee50fb68692df15ff` was pushed to
[PR #75](https://github.com/frathe/picfetch/pull/75) with Ronin's authorization.
The earlier Trane commit `8533342` was already on the branch; its changes were
not included in the security-fix commit.

- [CI 36725097510](https://github.com/frathe/picfetch/actions/runs/36725097510)
  passed on the first attempt: validation, all four native Linux/amd64 race
  partitions, Linux native guards, Windows tests, and both macOS architectures.
  The local full-suite platform limitation is covered by this hosted run.
- The downloaded Windows event artifact confirms
  `TestWindowsSaveTransport_PathIsData` passed (2.35 seconds) and
  `TestWindowsTrashTransport_PathIsData` passed (7.03 seconds), with run/pass
  events and no skips. The production PowerShell commands received exact
  ordinary/malicious Unicode paths at the instrumented sinks; picker OK/Cancel
  and file/directory recycle semantics remained intact. This closes the native
  verification gap without opening a dialog or moving a real file.
- [CodeQL 36725097446](https://github.com/frathe/picfetch/actions/runs/36725097446)
  passed Go and Actions analysis.
- [Qodana 36725097549](https://github.com/frathe/picfetch/actions/runs/36725097549)
  completed. Its fresh post-suppression `/qodana.sarif.json` contains eight
  `GoUnusedExportedFunction` false positives, all checked against production
  callers: fileaccess.Stat, NeedsCapture, HasScope, Export, Import;
  macbundle.CurrentRuntimeDirectory; openwith.InstallWindowDrop and Stop.
  No actionable result or new broad suppression. The neutral annotation check
  is not described as an empty report. Downloaded evidence is retained locally
  under `.scratch/windows-path-review/bb6ce5d/`.
- [Fresh code review](https://github.com/frathe/picfetch/pull/75#issuecomment-5912827831)
  and [fresh security review](https://github.com/frathe/picfetch/pull/75#issuecomment-5912966500)
  report no findings on `bb6ce5d`. All three historical threads remain resolved;
  this round introduced none.

The first medium finding is fixed and qualified. This documentation-only
completion record carries forward unchanged-code local inspection/test evidence
from `bb6ce5d`; it does not claim a review of an older commit covers a new head.
Fresh code/security reviews and hosted CI for the final documentation commit
will be retained on PR #75, without another status-only commit. No merge or
release is authorized by this review loop.

Ledger: zero spawns, one inline candidate review, one local full-suite attempt;
hosted full-suite and Windows qualification passed. The GitHub review loop is
additional user-authorized work. Unrelated Trane changes were preserved.
