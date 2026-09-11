# PR 18: Explorer admission and Unassigned return

Route: Standard. Validate and address the three Codex findings on `574ee6e`.
The lead owns assessment, regressions and fixes; no delegation. No new packages,
interfaces or UI strings. Full race verification runs in GitHub CI under the
authorized review loop; local verification stays focused.

## Acceptance criteria and tasks

1. Window -> Grid View and G reopen the captured Unassigned membership with a
   working Analyze action after opening an image. Ordinary cohorts retain their
   existing behavior. Files: `internal/ui/windowmenu.go`, existing
   `internal/ui/explorer_test.go`.
   Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/unassigned_grid_return$' -count=1`.
2. Explorer entry through the menu, Shift+S and its direct handler refuses a
   pending replacement scan or sort. After replacement finishes, entry analyzes
   only the committed sources. Files: `internal/ui/explorer.go`, existing
   `internal/ui/explorer_test.go`.
   Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/replacement_admission$' -count=1`.
3. README lists Apple Silicon macOS and x86-64 Linux with glibc/seccomp, including
   their respective download sizes. File: `README.md`.
   Verify: compare `sed -n '29,38p' README.md` against the supported-platform and
   download sections in `internal/ui/help/manual.md`.

Tasks are independent and implemented inline. Budget: zero spawns; one initial
fix round followed by fresh external review rounds as required. No new test files
or top-level UI tests; existing Qodana exclusions and shard assignment apply.

## Evidence

- Initial head: all CI checks pass; Codex security reports no issues.
- Qodana run `34566204631`: full post-suppression SARIF has zero results.
- CodeQL: no open alerts for `refs/pull/18/merge`.
- Before the fixes, all six replacement stage/entry combinations failed because
  Explorer admitted the old collection; the Window-menu return failed because
  Analyze was missing. The equivalent keyboard return passed.
- After the fixes, the new regressions and focused entry, cohort, cancellation,
  startup, stale-map and window-command checks passed in 6.012s, then passed under
  the race detector in 82.604s:

  ```sh
  go test -race ./internal/ui -run '^TestVisualSimilarityExplorer$/(replacement_admission|unassigned_grid_return|keyboard_entry|active_menu|cohort_viewer_menu|create_cohort_selection|create_cohort_review|favorite_identity_cancel|trial_launch|replacement_discards_late_map)$|^TestWindowCommandAdmissionMatrix$' -count=1 -timeout 3m
  ```

- README platform requirements and download sizes match the maintained manual.
- `make verify-build` passed (format/TUF/Qodana configuration, vet and build).
  `git diff --check` passed. The existing top-level test remains on `ui-1` and
  its existing Qodana exclusion applies. Commit-linked thread dispositions,
  fresh reviews and complete CI/shard/SARIF results are recorded on PR 18.

Review findings:
- https://github.com/frathe/picfetch/pull/18#discussion_r3986368996
- https://github.com/frathe/picfetch/pull/18#discussion_r3986369009
- https://github.com/frathe/picfetch/pull/18#discussion_r3986369015

Cost record: zero spawns; one implementation round; focused local race tests;
the complete suite runs only in GitHub CI.
