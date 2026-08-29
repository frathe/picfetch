# Plan: Cover the manual-opened menu observer

## Goal

Close the first open item in `todos.md`: protect the registration of
`help.SetOnManualOpened(view.syncMenus)` in `buildMainMenu` with a
viewer-boundary regression test.

The finished change should fail if that one registration is removed, while
leaving production behavior and package boundaries unchanged.

## Current behavior and test gap

- `internal/ui/menu.go:buildMainMenu` registers `view.syncMenus` as both the
  manual-opened and manual-closed observer.
- `internal/ui/help/manual.go:ShowManual` changes the singleton window state,
  then invokes the opened observer after either creating or raising the manual.
- `internal/ui/menu.go:menuState` reads `view.help.ManualOpen()`, and
  `internal/ui/menus.Apply` disables `Window -> Help` while the manual is open.
- F1 (`internal/ui/keys.go`) and `Window -> Help`
  (`internal/ui/windowmenu.go`) deliberately call only `ShowManual`; they no
  longer call `syncMenus` themselves.
- `internal/ui/help` already proves that the opened observer fires. It cannot
  prove that `internal/ui` subscribes to it.
- Existing viewer tests avoid the real manual because Fyne's test theme lacks
  the combined bold/monospace font style produced by the embedded Markdown.
  Removing the registration therefore passes the current suite.

## Chosen implementation

Add one focused viewer-level test. The test will temporarily install
`theme.DefaultTheme()` on the shared Fyne test app, build a normal viewer
through `newTestViewer`, open the real manual through `v.help.ShowManual()`, and
assert that `Window -> Help` changes from enabled to disabled.

This directly tests the composition seam in `buildMainMenu`. It does not add a
test-only production API, a mutable package-level seam, or a broad interface
around `help.Help`.

A local experiment already confirmed this approach under Fyne 2.8.0:

```text
go test ./internal/ui -run '^TestExperimentManualObserverWithDefaultTheme$' -count=1
ok github.com/frathe/picfetch/internal/ui
```

The experimental test file was removed; only this plan remains.

## Scope

Expected implementation files:

- `internal/ui/menu_test.go`
  - Add the viewer-boundary regression test.
  - Import `fyne.io/fyne/v2/theme`.
- `internal/ui/e2e_test.go`
  - Refine the trailing manual-test note: screenshot coverage still uses the
    limited test theme, while the new state/wiring test opts into the default
    theme for this one case.
- `todos.md`
  - Remove the item from `## TODO` after the regression test and mutation check
    pass.
  - Record the completed work under `Done -> Internal`, including the scoped
    theme swap and mutation result.

No `ARCHITECTURE.md` update is expected because no package, file, ownership, or
data-flow boundary changes.

## Delegation sequence

Subagents run sequentially. The primary agent reviews and, when needed, fixes
each completed chunk before starting the next one. No agents edit overlapping
files concurrently.

### Chunk 1: Add the regression test

**Owner:** implementation subagent

**Model:** `gpt-5.6-sol`, reasoning `high`

**Why this model:** the edit is small, but cleanup ordering around Fyne's shared
test app, secondary windows, theme cache, and `t.Cleanup` needs careful coding.

Tasks:

1. Add `TestBuildMainMenu_ManualOpenedObserverSyncsWindowHelp` to
   `internal/ui/menu_test.go`.
2. Before changing the theme, ensure the shared `testApp` is current if needed.
3. Save the existing theme, install `theme.DefaultTheme()`, and register theme
   restoration before constructing the viewer. Cleanup order must restore the
   theme only after the manual and main windows are closed.
4. Build through `newTestViewer(t)` so production startup and `buildMainMenu`
   wiring are exercised.
5. Assert the Window Help item starts enabled.
6. Capture the test driver's window set/count, call `v.help.ShowManual()`
   directly, identify the newly created manual window, and register exactly-once
   cleanup for it. Do not close all application windows or rely on a guessed
   delay.
7. Assert the manual is open and the Window Help item is now disabled.
8. Run formatting and the focused test:

   ```sh
   make fmt
   go test ./internal/ui -run '^TestBuildMainMenu_ManualOpenedObserverSyncsWindowHelp$' -count=1
   ```

Deliverable: test-only change in `internal/ui/menu_test.go`, with command output
summary and no documentation edits.

### Primary-agent review gate 1

The primary agent will inspect the full diff before accepting Chunk 1:

- Confirm the test drives the real `buildMainMenu` registration rather than
  duplicating the callback in test code.
- Confirm it invokes `ShowManual` directly, so an unrelated call-site
  `syncMenus` cannot make the test pass.
- Confirm theme restoration and secondary-window cleanup are registered in
  safe LIFO order and cannot double-close a test window.
- Confirm no mutable package-level seam, sleep, production API, or unrelated
  formatting change was introduced.
- Fix any issue found, then rerun the focused test.

The primary agent will then perform a mutation check:

1. Temporarily remove only
   `view.help.SetOnManualOpened(view.syncMenus)` from `buildMainMenu`.
2. Run the focused test and require a failure showing that Window Help stayed
   enabled after the manual opened.
3. Restore the registration immediately.
4. Rerun the focused test and require success.
5. Inspect `git diff` to confirm the mutation left no residue.

### Chunk 2: Reconcile comments and work tracking

**Owner:** documentation subagent

**Model:** `gpt-5.6-luna`, reasoning `medium`

**Why this model:** changes are narrow editorial updates after behavior is
already settled.

Tasks:

1. Update the manual note at the end of `internal/ui/e2e_test.go` without
   claiming screenshot coverage. Point to the focused menu-wiring test and
   explain that it temporarily uses the complete default theme.
2. Move the completed `todos.md` item into `Done -> Internal`. Record:
   - the exact observer registration now covered;
   - why the default theme is scoped to one test;
   - how the new manual window and theme are cleaned up;
   - that deleting the registration fails the new test.
3. Do not change `ARCHITECTURE.md`, translations, production code, or other
   TODO entries.

Deliverable: only `internal/ui/e2e_test.go` comments and `todos.md` changes.

### Primary-agent review gate 2

The primary agent will:

- Review wording against actual test behavior and mutation evidence.
- Ensure the old open TODO is fully removed and the completed note is under the
  correct release-note category.
- Ensure no user-visible strings or translation bundles changed.
- Fix inaccuracies before continuing.
- Run:

  ```sh
  make fmt-check
  go test ./internal/ui -run 'ManualOpenedObserver|SetOnManualOpened' -count=1
  ```

### Chunk 3: Independent verification

**Owner:** verification subagent, read-only unless explicitly sent back for a
specific fix

**Model:** `gpt-5.6-terra`, reasoning `high`

**Why this model:** suited to a fresh diff audit and broad Go verification
without spending frontier-model capacity on mechanical execution.

Tasks:

1. Review the final diff for test isolation, cleanup ordering, observer
   coverage, and compliance with `AGENTS.md`.
2. Confirm the test would fail for deletion of the opened-observer registration
   and is not merely re-testing `help.ShowManual`.
3. Run the repository handoff checks from the root:

   ```sh
   make fmt-check
   go vet ./...
   go build ./...
   go test -timeout 20m -race ./...
   ```

4. Report failures with the shortest decisive output. Do not edit files during
   this chunk.

### Primary-agent review gate 3 and handoff

The primary agent will review the verifier's findings, inspect the final diff
and `git status`, fix any problem personally, and rerun affected checks. Handoff
will summarize the regression guard, mutation result, complete verification,
changed files, and a suggested commit message. No commit will be created.

Suggested commit message:

```text
test: cover manual-opened menu observer wiring
```

## Acceptance criteria

- Removing `SetOnManualOpened(view.syncMenus)` makes the new focused test fail.
- With the registration present, opening the real manual disables
  `Window -> Help` without a hand-written sync at the opener.
- The test does not panic on the embedded manual Markdown.
- The manual window closes exactly once during cleanup.
- The original shared test theme is restored after the test.
- No production seam or package interface is added.
- Existing F1, Window menu, and help-package tests still pass.
- `todos.md` records completion only after the mutation check succeeds.
- Full race-enabled repository checks pass.

## Open decisions

1. **Coverage granularity.** Recommended: call `v.help.ShowManual()` directly
   and test the observer once. Existing tests pin the displayed F1 accelerator,
   Window menu callback dispatch, and the help package's observer firing; both
   production openers call this same `ShowManual` method. Alternative: drive F1
   or `Window -> Help`, which covers an opener too but could pass if a future
   hand-written `syncMenus` at that opener masked a missing registration.
2. **Theme strategy.** Recommended: use a scoped `theme.DefaultTheme()` swap in
   the test, proven locally and requiring no production changes. Alternative:
   introduce a manual-content injection or help abstraction; this creates a
   larger API/test seam solely for one regression test.
3. **Work-log detail.** Recommended: move the item to `Done -> Internal` and
   update the e2e note in the same change. Alternative: remove the TODO with a
   shorter completion note and leave the e2e explanation unchanged.
