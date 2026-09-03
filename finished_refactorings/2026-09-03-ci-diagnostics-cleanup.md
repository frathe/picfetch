# CI diagnostics cleanup

## Frame

Remove actionable warning/error-looking output from the v1.0.0 CI and close
the four current Qodana findings without changing PicFetch behavior. Preserve
the user's in-progress screenshot edits. Do not downgrade GitHub Actions or
hide upstream diagnostics globally.

Route: **Deep** because the feedback came from release and Microsoft Store
Windows packaging, even though each code edit is small.

## Evidence

- Release run `33791853835` and Store run `33791853716` both log Fyne's
  `Failed to inject metadata init file` on the first fyne-cross architecture.
- A fresh `fyneio/fyne-cross-images:windows` container runs Go 1.25.10 while
  `go.mod` requires Go 1.27.1. Its first `GOTOOLCHAIN=auto go mod edit -json`
  emits `go: downloading go1.27.1` before the JSON, reproducing Fyne's
  `invalid character 'g'` parse failure. The second architecture is clean
  because fyne-cross shares `/go` as its cache.
- Release macOS jobs install the deprecated
  `fyne.io/fyne/v2/cmd/fyne` wrapper.
- Qodana run `33790763030` reports exactly four results: one build-tag false
  positive (`GoBoolExpressions`) and three actionable Go style/error results.
- WinGet run `33793741644` failed in `komac sync-fork` with a GitHub query
  incident identifier after validating the release and assets.
- `actions/download-artifact@v8` resolves to the latest official v8.0.1 and
  still has an open upstream `Buffer()` deprecation issue; changing versions
  is not an evidence-backed project fix.

## Decisions

| Area | Decision |
|---|---|
| Fyne metadata | Warm the exact Go toolchain in the same fyne-cross image/cache before the first Windows package build. |
| Fyne CLI | Install the current `fyne.io/tools/cmd/fyne` command. |
| Store marker | Add the narrow source-local `GoBoolExpressions` suppression at the build-tag-dependent branch. |
| Other Qodana findings | Fix the two error strings and explicitly ignore the impossible `strings.Builder` write error. |
| Download action warning | Leave it visible until upstream fixes v8; do not downgrade. |
| WinGet | Retry the unchanged run once; change the workflow only if the same sync failure reproduces. |

## Acceptance criteria

1. Windows packaging warms Go 1.27.1 before Fyne parses `go mod edit -json`,
   using the same image and cache as fyne-cross.

   ```sh
   go test ./scripts/msixstage -run 'TestMicrosoftStoreWorkflowAndBuildTarget|TestPackagingToolsUseCurrentFyneCLI'
   ```

2. The real container feedback loop produces JSON without a leading `go:`
   diagnostic after the warm target runs.

   ```sh
   make warm-fyne-cross-windows
   docker run --rm -v "$PWD:/app:ro" -v "$(dirname "$(go env GOCACHE)")/fyne-cross:/go" -w /app -e GOTOOLCHAIN=auto fyneio/fyne-cross-images:windows go mod edit -json
   ```

3. The deprecated Fyne command path is absent and the current path is used.

   ```sh
   ! rg 'fyne.io/fyne/v2/cmd/fyne' Makefile .github/workflows/release.yml
   rg 'fyne.io/tools/cmd/fyne' Makefile .github/workflows/release.yml
   ```

4. The four Qodana sites are either fixed or narrowly suppressed, with no
   behavior change.

   ```sh
   rg -n 'noinspection GoBoolExpressions' main.go
   ! rg 'errors.New\("usage: .*\.\.\."\)|fmt.Errorf\("Go build context|^[[:space:]]*fmt.Fprintf\(&fileTypes' scripts/testshards/main.go scripts/msixstage/stage.go
   ```

5. Focused package tests and the repository gate pass.

   ```sh
   go test ./scripts/msixstage ./scripts/testshards
   make verify
   ```

## Non-goals

- Suppressing or downgrading `actions/download-artifact` for its upstream
  Node deprecation warning.
- Refactoring Store update ownership or changing runtime behavior.
- Changing release assets, the published v1.0.0 tag, or the submitted Store
  package.
- Automatically committing or pushing these cleanup changes.

## Honest limit

The local container loop proves the Fyne JSON contamination and cache warmup.
Only a later GitHub Actions run can prove GitHub-hosted runner logs are clean.
Qodana's exact post-change result set likewise requires a new CI scan.

## Task graph

```text
T1 Makefile/workflow contract test -> T2 packaging cleanup
T3 Qodana source cleanup --------------------------\
T4 WinGet retry (external, unchanged workflow) ----+-> T5 final gate
```

### Task 1 - Packaging contract test

Owner: T0 inline. Files: `scripts/msixstage/msixstage_test.go`.
Test: Require the warm target/cache coupling and current Fyne CLI path.
Verify: AC1. Budget: 0 spawns; 1 review round; full suite no.

### Task 2 - Packaging diagnostics cleanup

Owner: T0 inline. Files: `Makefile`, `.github/workflows/release.yml`.
Depends: T1. Contract: `warm-fyne-cross-windows` prepares the toolchain in
the same cache used by all Windows fyne-cross calls.
Verify: AC1-AC3. Budget: 0 spawns; 1 review round; full suite no.

### Task 3 - Qodana cleanup

Owner: T0 inline. Files: `main.go`, `scripts/testshards/main.go`,
`scripts/msixstage/stage.go`.
Test: Existing behavior tests; analyzer findings are semantics-preserving and
have no legitimate new behavior seam.
Verify: AC4 plus focused package tests. Budget: 0 spawns; 1 review round;
full suite no.

### Task 4 - WinGet retry

Owner: T0 inline. Files: none unless the unchanged retry reproduces.
Verify: `gh run view 33793741644 --json conclusion`.
Budget: 0 spawns; 1 retry; full suite no.

### Task 5 - Final verification

Owner: T0 inline. Files: all changed files.
Verify: AC5, diff review, preservation of the user's screenshot changes.
Budget: 0 spawns; 1 review round; full suite once.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|---|---:|---:|---:|---|
| T1 | 0 / 0 | 1 | no | existing Makefile contract seam |
| T2 | 0 / 0 | 1 | no | Windows-specific; kept inline |
| T3 | 0 / 0 | 1 | no | hot context |
| T4 | 0 / 0 | 1 | no | one external retry |
| T5 | 0 / 0 | 1 | yes | final gate only |
