# Manual update check and restart plan

Status: implemented on 2026-08-30; full verification remains the final review gate.

Source todo: formerly the first item under `todos.md` → `## TODO`; moved to
`## Done` after implementation on 2026-08-30.

## Goal

Add a manual update flow below the existing automatic-update checkbox in the
Settings window:

1. The user starts a check explicitly, regardless of the automatic-update
   preference or the once-per-day throttle.
2. If the installed version is current, an OK-only dialog says so.
3. If a newer release exists, a modal dialog names the version and shows the
   archive download progress.
4. When the verified archive is staged, the dialog changes to a success state
   with a **Perform update** action.
5. Choosing that action closes PicFetch, installs the staged build, relaunches
   it, and shows the already-supported What's New window on the new launch.

The existing automatic flow remains opt-in, checks at most once per local day,
downloads in the background, verifies GitHub's immutable release attestation,
and stages the update for shutdown.

## Repository findings that shape the work

- `internal/ui/settingswin.Window` owns the Settings widgets and is backed by a
  consumer-side `Host`. The new button and its dialogs belong there so the
  dialogs can be parented to the Settings window rather than appearing behind
  it on the main viewer window.
- `internal/ui/autoupdate.go` owns viewer-side gates and the viewer's
  `requestLifecycle`. `internal/ui/autoupdate.Updater` owns the worker,
  `completion.Signal`, client, staging policy, and What's New cache. Manual
  work should reuse this path, not create a second update implementation.
- `update.Client.Check` already separates checking from downloading.
  `update.Client.Download` currently reads the complete archive without a
  progress callback, then verifies the digest and attestation, extracts the
  archive, and writes `stage.json`.
- The existing security boundary is immutable for this feature: a downloaded
  archive must pass its advertised SHA-256 digest check (when GitHub supplies
  one) and the GitHub Sigstore release-attestation verification before it can
  be extracted, staged, reported as successful, applied, or relaunched. A
  missing verifier and every invalid signature/attestation remain hard errors.
- `Updater.ApplyStagedUpdate` is called from `Run`'s `SetOnStopped` callback.
  It stores release notes before replacing the executable, which is why the
  next matching build can show What's New.
- Contrary to the todo's parenthetical note, automatic restart is not currently
  implemented. `internal/update/apply.go` explicitly says Windows does not
  relaunch, `windowsApplyScript` contains no launch command, Unix only replaces
  files, and `ARCHITECTURE.md` says “Apply is OnStopped, not a relaunch.”
- A manual request can overlap an automatic request started at launch or when
  the checkbox is enabled. Both must be serialized before either can remove or
  rewrite the shared stage directory.
- Every background callback that changes Fyne state must be marshalled through
  `fyne.Do`; the request token must be checked again inside the queued closure.
  The test harness must be able to drain every update worker without sleeps.
- All new strings must use `lang.L` and be added to both `translations/en.json`
  and `translations/de.json`. The English bundle must remain an identity map.
- No new package is needed. `ARCHITECTURE.md` still needs wording updates
  because the responsibilities and restart behavior of existing packages will
  change.

## Locked product decisions

### Decision 1 — what **Perform update** means

Decision: set an “apply and relaunch” intent, call `app.Quit`, apply from the
existing shutdown hook, and relaunch the newly installed executable. On Unix,
start the replaced executable after a successful copy. On Windows, add the
launch command to the existing post-exit `.cmd` script after its copy succeeds.

### Decision 2 — success-dialog choices and deferred installation

Decision: show **Later** first/default and **Perform update** second. Later
closes the dialog and leaves the stage in place. Preserve current policy that a
staged update is still installed without relaunch on the next normal shutdown.

### Decision 3 — visible checking and failures

Decision: show an indeterminate **Checking for updates…** dialog immediately;
replace it with the current-version, downloading, success, or error state. Show
network, verification, extraction, and staging failures in an OK-only dialog as
well as logging them with `fyne.LogError`.

### Decision 4 — manual behavior relative to automatic updates

Decision: manual checks work even when automatic checks are off, bypass the
daily `Due` gate, and record the day after a successful GitHub check so an
automatic check does not repeat later that day. If a matching valid stage
already exists, perform the GitHub check but reuse the stage instead of
downloading the same archive again.

### Decision 5 — unknown download size

Decision: show a determinate percentage when the final HTTP response has a
positive `Content-Length`; otherwise keep an indeterminate bar until the
download ends. Do not estimate against the 200 MiB safety ceiling, because that
would present a false percentage.

## Proposed design

### Low-level download progress

Keep `Client.Download(ctx, release)` as the compatibility wrapper and add a
focused `Client.DownloadWithProgress(ctx, release, callback)` path. Define a
small value such as:

```go
type DownloadProgress struct {
	Downloaded int64
	Total      int64 // <= 0 means unknown
}
```

The archive response reader will:

- emit an initial event;
- count only release-archive bytes, not the later attestation request or
  extraction work;
- preserve the existing `maxArchiveBytes` limit and cancellation behavior;
- emit monotonic values and one exact final event after EOF;
- cap callback frequency to percentage changes for known totals and sensible
  byte steps for unknown totals, avoiding thousands of `fyne.Do` calls;
- never claim 100% before all declared archive bytes have arrived;
- continue to fail closed on digest or attestation errors and remove an
  incomplete stage exactly as today.

Progress reporting wraps archive reads only. It must not reorder, bypass, make
optional, or pre-empt the existing digest and Sigstore release-attestation
checks. Existing wrong-digest, missing-verifier, and verifier-failure tests stay
in place; new manual-flow tests must prove none of those failures emits Ready.

Tests will cover known length, unknown length, zero-byte/short responses,
monotonicity, exact final values, cancellation, oversized input, and the nil
callback compatibility path.

### Shared automatic/manual update worker

Add callback-based request events to `internal/ui/autoupdate`, for example:

```go
type Events struct {
	Downloading func(version string)
	Progress    func(update.DownloadProgress)
	Current     func()
	Ready       func(update.Stage)
	Failed      func(error)
}
```

Callbacks run on the update worker. They contain no Fyne calls. The viewer
adapter is solely responsible for `fyne.Do` and the second staleness check.

Refactor the existing worker body into one internal run function used by:

- the existing automatic `Start`, which continues to require `EnsureClient`
  before `updateOp.begin` and receives zero callbacks; and
- a manual start path, which may prepare the verifier/client inside tracked
  background work so the Settings UI does not freeze.

Add serialization around the complete check/download/stage transaction. A
new manual request may cancel/supersede an automatic token, but it must not
enter `Download` until the earlier worker has left the shared stage directory.
Add an all-worker `Settle`/wait mechanism if the existing latest-generation
`completion.Signal` cannot prove every superseded worker has stopped; wire it
into `internal/ui/harness_test.go` cleanup.

Worker rules:

- check staleness before expensive work, after waiting for serialization,
  after `Check`, before UI events, and before applying UI results;
- a successful check records `LastUpdateCheckDay` before reporting Current or
  starting a download;
- a failed check does not record the day;
- a matching, usable staged release is reported Ready without redownloading;
- reuse requires provenance written only after the original digest and
  Sigstore checks, matching release/platform identity, and unchanged
  extracted binary/plist hashes;
- a canceled request emits no terminal UI event;
- a fully written stage remains usable even if its UI token becomes stale at
  the final boundary, matching the existing completed-stage policy;
- every failure remains logged, while manual callers also receive Failed;
- no package-level mutable test seam is added.

### Viewer glue and consumer-side Settings contract

Extend `settingswin.Host` with two narrow operations:

```go
CheckForUpdatesNow(settingswin.UpdateCallbacks)
PerformUpdate() error
```

`settingswin.UpdateCallbacks` will be a consumer-defined struct with callbacks
for Downloading, Progress, Current, Ready, and Failed. Document that the Host
delivers them on the Fyne UI thread. This keeps `settingswin` independent of
the concrete updater package.

The viewer implementation will:

- remove stale stage metadata first;
- bypass `CheckForUpdates()` and `Due` only for the manual entry point;
- validate the current version and platform asset before starting;
- begin/supersede the viewer's single `updateOp` lifecycle;
- adapt updater events to `settingswin.UpdateCallbacks` with `fyne.Do`;
- re-check `token.current()` inside each `fyne.Do` closure;
- keep automatic setting changes from racing stage writes;
- request restart intent only after confirming a still-newer staged update;
- quit through a per-viewer function initialized from `application.Quit`, so
  tests can observe the request without quitting the shared test app.

### Settings UI state machine

Add a **Check now** button immediately below **Check for updates**. The checkbox
continues to mean “check automatically”; the button means “check once now.”

Store transient widgets/dialogs on `settingswin.Window`, following the existing
testable-field pattern. Keep a single active update dialog and hide/replace it
when the phase changes:

```text
idle
  -> checking (indeterminate)
      -> current (OK)
      -> downloading(version, determinate or indeterminate)
          -> ready (Later / Perform update)
      -> failed (OK)
```

Behavior details:

- disable **Check now** as soon as it is tapped;
- prevent duplicate requests while the current Settings window owns a flow;
- parent all dialogs to `w.win.Window()`;
- use `widgets.ChoicePanel` in terminal dialogs so Return and Escape do not
  leak into the Settings window's unfocused key handler;
- default focus to the non-disruptive button;
- display normalized release version consistently with What's New;
- update the progress bar only on the UI thread;
- switch from indeterminate to determinate when a positive total becomes
  known;
- restore button/check-box enabled state on Current, Failed, Later, dialog
  close, and Settings-window teardown;
- tolerate the Settings window closing while work continues: queued callbacks
  become no-ops against closed/nil widgets, while a completed verified stage
  remains available;
- measure the added row and increase the default `windowH` only if the content
  would otherwise clip; remembered geometry remains authoritative.

Proposed localized English keys (final punctuation/capitalization will be
locked before coding):

- `Check now`
- `Software Update`
- `Checking for updates…`
- `You are on the current version.`
- `Downloading version %s`
- `Update downloaded successfully.`
- `Perform update`
- `Later`
- `OK`
- `Could not check for updates: %v`

### Apply and optional relaunch

Carry explicit relaunch intent into the existing shutdown-time apply path.
Do not replace the running executable before the Fyne lifecycle has stopped.

- `PerformUpdate` validates the staged version, records relaunch intent, and
  calls the viewer's quit function.
- `registerShutdown` continues to save session and preferences before apply.
- `ApplyStagedUpdate` saves What's New before replacement as it does now.
- The `internal/update.Apply` dispatcher gains explicit apply options (or an
  equivalent boolean) so normal shutdown and Perform update cannot be confused.
- Unix applies first, then starts the installed executable only when requested.
- Windows' generated script waits for the old PID, copies, deletes the staged
  binary, starts the installed executable only when requested, and deletes
  itself.
- A helper-start failure is logged through `fyne.LogError`; delayed Unix exec
  failures stay on stderr and delayed Windows copy/start failures are appended
  to `%TEMP%\picfetch-update.log`. The installed binary and What's New cache
  are retained so a manual launch can recover and show the notes.
- Normal shutdown with a staged automatic update preserves today's no-relaunch
  behavior.

## Delegable implementation chunks

Agents share one worktree and must not commit. Chunks run sequentially because
later contracts depend on earlier ones, and the primary reviews/fixes each diff
before starting the next chunk. Opus is not available in this environment;
`gpt-5.6-sol` with `xhigh` reasoning is the selected substitute for the two
cross-platform/concurrency-heavy chunks.

### Chunk 0 — lock test names and baseline

Owner: primary agent.

Completed 2026-08-30. All five product decisions are locked. Baseline passed:

```text
go test ./internal/update ./internal/ui/autoupdate ./internal/ui/settingswin
go test -run 'Test.*Update|TestDrain_WaitsUpdateDone|TestLastUpdateCheckDay' ./internal/ui
```

The tests required permission to bind existing localhost `httptest` listeners;
no external network was used.

Inputs: the five locked decisions above.

Work:

- lock exact English/German strings and button order;
- list exact test names before production edits;
- record baseline results for `go test ./internal/update`,
  `go test ./internal/ui/autoupdate`, `go test ./internal/ui/settingswin`, and
  focused `internal/ui` update tests.

Exit gate: exact strings/tests are recorded and baseline tests pass.

### Chunk 1 — archive progress API

Delegate: `gpt-5.6-sol`, reasoning `high`.

Allowed primary files:

- `internal/update/download.go`
- `internal/update/download_test.go`
- narrowly related test helpers in `internal/update`

Work:

1. Write failing progress/cancellation/limit tests.
2. Add `DownloadProgress` and `DownloadWithProgress` while preserving
   `Download` behavior.
3. Add deterministic callback coalescing and exact terminal reporting.
4. Run `go test -race ./internal/update`.

Primary review gate:

- inspect the complete scoped diff;
- check that verification/staging semantics did not move or weaken;
- confirm no UI/Fyne dependency entered `internal/update`;
- mutation-check that removing the final progress event fails a test;
- fix findings and rerun the package tests before Chunk 2.

### Chunk 2 — updater events, serialization, and manual lifecycle

Delegate: `gpt-5.6-sol`, reasoning `xhigh`.

Allowed primary files:

- `internal/ui/autoupdate/updater.go`
- `internal/ui/autoupdate/updater_test.go`
- `internal/ui/autoupdate.go`
- `internal/ui/autoupdate_test.go`
- `internal/ui/harness_test.go` if an all-worker drain is required

Work:

1. Write failing event-order and supersession tests.
2. Add callback events and the shared internal worker body.
3. Preserve automatic gates and verifier-before-lifecycle ordering.
4. Add manual bypass of opt-in/Due, background preparation, stage reuse, and
   successful-check day persistence.
5. Serialize stage-directory transactions and make all workers drainable.
6. Marshal viewer callbacks with `fyne.Do` and inner token checks.
7. Run `go test -race ./internal/ui/autoupdate` and focused
   `go test -race -run 'Test.*Update' ./internal/ui`.

Primary review gate:

- trace Current, newer release, check failure, download failure, cancellation,
  stage reuse, and Settings-close paths by hand;
- inspect every goroutine for cancellation and observable completion;
- verify a manual request cannot race an automatic `RemoveStage`/`Download`;
- mutation-check the inner staleness guard and serialization test;
- fix findings before Chunk 3.

### Chunk 3 — Settings button and dialogs

Delegate: `gpt-5.6-terra`, reasoning `high`.

Allowed primary files:

- `internal/ui/settingswin/settingswin.go`
- `internal/ui/settingswin/settingswin_test.go`
- minimal viewer glue/type adjustments already defined by Chunk 2

Work:

1. Extend the consumer-side Host and fake Host.
2. Add **Check now** below the automatic checkbox.
3. Implement checking, current, downloading/progress, ready, and failure UI
   states with one parented dialog at a time.
4. Add keyboard-focus, duplicate-tap, window-close, stale-callback, unknown
   length, and perform-action tests without sleeps.
5. Adjust default Settings height only if layout measurement requires it.
6. Run `go test -race ./internal/ui/settingswin` plus focused menu/feature tests.

Primary review gate:

- inspect object order and all dialog close/focus paths;
- confirm every string uses `lang.L` (translations land in Chunk 5);
- verify callbacks only touch live fields on the UI thread;
- mutation-check that swapping button order or removing focus fails tests;
- fix findings before Chunk 4.

### Chunk 4 — perform-update quit/apply/relaunch path

Delegate: `gpt-5.6-sol`, reasoning `xhigh`.

Allowed primary files:

- `internal/update/apply.go`
- `internal/update/apply_unix.go`
- `internal/update/apply_windows.go`
- `internal/update/apply*_test.go`
- `internal/ui/autoupdate/updater.go`
- `internal/ui/autoupdate.go`
- corresponding focused tests
- `internal/ui/viewer.go` and `internal/ui/build.go` for the per-viewer quit seam

Work:

1. Write failing tests that distinguish normal apply from requested relaunch.
2. Carry relaunch intent through `Updater` and the apply dispatcher.
3. Add Unix launch-after-copy and the requested Windows script launch.
4. Wire Perform update to validate the stage, set intent, and request quit.
5. Preserve release-note caching and recovery on launch/apply errors.
6. Run `go test -race ./internal/update` and focused UI apply tests.

Primary review gate:

- inspect generated Windows quoting/order and Unix replacement/launch order;
- confirm normal shutdown still installs without relaunch;
- mutation-check that deleting the relaunch command fails a test;
- verify tests never launch the real PicFetch binary or mutate the installed app;
- fix findings before Chunk 5.

### Chunk 5 — localization, manuals, architecture, todo, and integration tests

Delegate: `gpt-5.6-terra`, reasoning `high`.

Allowed primary files:

- `translations/en.json`
- `translations/de.json`
- `internal/ui/help/manual.md`
- `internal/ui/help/manual_de.md`
- `ARCHITECTURE.md`
- `todos.md`
- narrowly scoped integration tests in `internal/ui`

Work:

1. Add exact locale-parity entries and natural German translations.
2. Update both manuals: automatic versus manual behavior, progress, Perform
   update, restart, Later, failure handling, and What's New.
3. Update architecture entries that currently say updates never relaunch and
   describe Settings as checkbox-only.
4. Add an end-to-end fake-server test from Settings callback through current,
   progress, ready, and perform intent; no real desktop or executable writes.
5. Move the todo to `## Done` only after all behavior is verified.
6. Run translation parity tests and all touched package tests.

Primary review gate:

- compare English/German key sets and identity-map rules;
- verify docs match actual semantics, especially automatic shutdown;
- inspect the todo move so unrelated user edits remain untouched;
- fix findings before final review.

### Chunk 6 — independent adversarial review

Delegate: `gpt-5.6-sol`, reasoning `xhigh`, read-only review first.

Review targets:

- lifecycle races between automatic and manual checks;
- stale `fyne.Do` callbacks after Settings close or a superseding request;
- stage removal/download races;
- progress correctness through redirects, unknown length, cancellation, and
  verification time after 100%;
- keyboard behavior for every dialog;
- Unix/macOS/Windows relaunch command safety and quoting;
- What's New persistence and version matching;
- translation and architecture drift;
- missing negative tests and mutable package-level seams.

The primary reviews the report, makes all warranted fixes personally, reruns
focused tests, and only then starts final verification.

## Final verification

Run from the repository root, with no test sleeps and no real network/desktop
side effects:

```sh
make fmt
make fmt-check
go vet ./...
go build ./...
go test -timeout 20m -race ./...
```

Also inspect `git diff --check`, `git status --short`, and the complete diff.
Do not regenerate golden screenshots unless a golden test actually covers the
Settings window (none currently does). Do not commit; finish with a suggested
commit message.

## Acceptance checklist

- Manual button is directly below the automatic checkbox.
- Manual check works with automatic checking disabled and when today's check is
  already recorded.
- One request owns the stage directory at a time.
- Current version produces the localized OK dialog.
- New version produces a localized version label and honest progress bar.
- Unknown length stays indeterminate.
- Digest, attestation, extraction, and staging remain fail-closed.
- Existing SHA-256 and Sigstore release-attestation checks execute on the manual
  path exactly as on the automatic path; invalid or missing verification can
  never produce a usable stage, success dialog, apply, or relaunch.
- Success dialog cannot trigger duplicate update actions.
- Perform update saves session/preferences, applies only a still-newer stage,
  relaunches the installed build, and causes matching What's New notes to
  appear on the updated launch.
- Normal shutdown installs an existing stage without relaunch.
- Closing Settings or superseding a request leaves no stale UI mutation or
  untracked goroutine.
- English/German translations, manuals, `ARCHITECTURE.md`, and `todos.md` agree
  with shipped behavior.
- Full CI-equivalent verification passes under `-race`.

## Suggested eventual commit message

`feat: add manual update check and restart flow`
