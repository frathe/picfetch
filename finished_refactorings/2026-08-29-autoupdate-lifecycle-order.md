# Autoupdate Lifecycle Ordering — Implementation Plan

> Requested execution style: subagent-driven development with review after every
> chunk. The `superpowers` skill is not available in this session's skill
> allowlist, so this plan spells out the equivalent delegation, review, mutation,
> and verification gates directly.

## Goal

Close the first open item in `todos.md`: restore the pre-`9fb2f79` ordering in
`maybeStartUpdateCheck` so construction of the production Sigstore verifier and
update client succeeds before `updateOp.begin()` supersedes the current
lifecycle token.

No user-visible updater policy changes are intended. Opt-in gating, daily
due-ness, asset selection, cancellation, check/download behavior, stage
retention, logging text, and apply-on-stop behavior must remain unchanged.

## Current behavior and exact regression

Before the `internal/ui/autoupdate` extraction, `maybeStartUpdateCheck` did this:

```text
gates -> build verifier/client -> begin lifecycle token -> start goroutine
```

Current code does this:

```text
gates -> begin lifecycle token -> Updater.Start -> build verifier/client
```

If `update.NewSigstoreVerifier()` fails, `Updater.Start` returns without
starting its completion signal or goroutine, but the viewer has already
advanced `updateOp` and cancelled/superseded its previous token. Today the
client is nil only on the first production attempt, so no real update-check
goroutine is normally displaced. The ordering difference is nevertheless real
and makes the lifecycle contract weaker than the pre-extraction implementation.

Desired sequence:

```text
gates -> EnsureClient
             failure: log and return; lifecycle unchanged
             success: begin lifecycle token -> Start check/download
```

## Relevant code and constraints

- `internal/ui/autoupdate.go:maybeStartUpdateCheck` owns viewer-side gates and
  `requestLifecycle.begin()`.
- `internal/ui/autoupdate/updater.go:Updater.Start` currently lazily constructs
  the verifier/client, begins `completion.Signal`, and starts the goroutine.
- `internal/ui/lifecycle.go` makes revisions monotonic. There is no valid
  "rollback" after `begin`; cancelling the new context would not restore the
  previous token.
- The locked architecture decision remains: cancellation stays in
  `internal/ui`; `requestLifecycle` must not move into `internal/ui/autoupdate`.
- Runtime/test-configurable seams belong on `viewer` or the owning feature, not
  in mutable package-level variables. Any verifier factory seam therefore
  belongs on `autoupdate.Updater`.
- `internal/ui/grid`'s `UIQueue` exception is unrelated and must not be touched.
- No new goroutine is needed. Existing `Updater.Done()` and harness draining
  remain the only completion path.
- No translation key or user-visible string should change.
- Do not add `TODO` or `FIXME` comments. Do not run `git commit`.

## Recommended design

Add an idempotent `Updater.EnsureClient() error` preparation step.

`EnsureClient` should:

1. Return nil immediately when a test or earlier successful call already set
   `u.client`.
2. Call a per-`Updater` verifier factory, defaulted by `New` to
   `update.NewSigstoreVerifier`.
3. On failure, return the original error, leave `u.client` nil, and remain
   retryable.
4. On success, build the same `update.Client` configuration used today:
   `http.Client{Timeout: 30 * time.Second}`, `time.Now`, returned verifier, and
   `u.dir` as `StageDir`.
5. Cache the client exactly once under the existing UI-goroutine ownership
   assumption. Do not add a mutex or `sync.Once`; a failed `sync.Once` would
   prevent retry, and no concurrent caller exists.

Add an `Updater.SetVerifierFactory` seam for the viewer-boundary regression
test. The seam is an exported method only because package `ui` cannot access an
unexported field in its `autoupdate` child package; the package remains under
Go's `internal` boundary. `New` installs the production default. Avoid a mutable
package variable.

`maybeStartUpdateCheck` will call `EnsureClient`, log the same
`"update verifier unavailable"` message on error, and return before
`updateOp.begin`. `Updater.Start` then handles only an already-prepared client,
the completion signal, and the existing goroutine.

### Alternatives rejected

- **Invalidate or cancel the just-created token on verifier failure.** This
  cannot restore the previous monotonic revision or uncancel its context.
- **Pass a `begin` callback into `Updater.Start`.** This keeps one public method
  but gives the child package control over when the viewer creates lifecycle
  state, obscuring the locked ownership boundary.
- **Move `requestLifecycle` into `autoupdate`.** Explicitly rejected during the
  viewer field-cluster extraction and unnecessary for this fix.
- **Build the verifier eagerly in `autoupdate.New`.** This would do TUF/cache
  work when update checks are disabled, violating the current opt-in/lazy
  behavior.
- **Force a real Sigstore failure through filesystem or network state in a
  test.** Brittle, environment-dependent, and slower than an instance-owned
  factory seam.

## Baseline

The focused race-enabled baseline passed while this plan was written:

```text
go test -timeout 3m -race -count=1 ./internal/ui/autoupdate ./internal/ui \
  -run 'TestUpdater_|TestUpdateCheck_|TestSetCheckForUpdates'

ok github.com/frathe/picfetch/internal/ui/autoupdate
ok github.com/frathe/picfetch/internal/ui
```

The sandboxed first attempt could not bind `httptest`'s loopback port; the same
command passed with local-port permission. This is an environment restriction,
not a repository failure.

## Planned file map

- `internal/ui/autoupdate/updater.go`
  - Own the verifier factory and `EnsureClient`.
  - Narrow `Start` to prepared-client execution.
- `internal/ui/autoupdate/updater_test.go`
  - Unit-test idempotent preparation, failure/retry, and the prepared-client
    precondition.
- `internal/ui/autoupdate.go`
  - Prepare before `updateOp.begin` and preserve the current log boundary.
- `internal/ui/autoupdate_test.go`
  - Pin lifecycle revision/token behavior when verifier construction fails.
- `internal/ui/viewer.go`
  - Update the updater/updateOp ownership comment.
- `ARCHITECTURE.md`
  - Update the two updater flow locators to include client preparation before
    lifecycle begin.
- `todos.md`
  - Move the open ordering item into `Done -> Internal` only after tests and
    mutation verification pass.

No changes are expected in `internal/update`, `internal/ui/lifecycle.go`,
translations, the test harness drain, or golden screenshots.

## Delegation strategy

Chunks run sequentially because Chunks 1 and 2 edit the same API. Each
`spawn_agent` call should use `fork_turns: "none"`, the model and effort below,
and a prompt containing `AGENTS.md` requirements plus only that chunk's scope.
The primary agent reviews the complete diff, fixes problems personally, and
runs the stated gate before starting the next subagent.

| Chunk | Model | Effort | Reason |
|-------|-------|--------|--------|
| 1 — package preparation API and unit tests | `gpt-5.6-sol` | `high` | API boundary, retry semantics, and test seam need strongest coding judgment. |
| 2 — viewer ordering and regression test | `gpt-5.6-sol` | `high` | Lifecycle revision/cancellation behavior is subtle despite the small diff. |
| 3 — architecture and backlog reconciliation | `gpt-5.6-terra` | `medium` | Bounded documentation work requiring accurate cross-file wording. |
| 4 — independent read-only audit and full verification | `gpt-5.6-terra` | `high` | Fresh review plus broad race-enabled Go checks. |

No chunk needs an unsplittable Opus task. Opus is also not in this session's
subagent model allowlist; the hardest separable work is assigned to
`gpt-5.6-sol`.

---

## Chunk 1 — Extract idempotent client preparation

**Subagent:** `gpt-5.6-sol`, reasoning `high`

**Files:**

- Modify `internal/ui/autoupdate/updater.go`
- Modify `internal/ui/autoupdate/updater_test.go`
- Do not modify viewer package files or documentation

This chunk must remain behavior-preserving and green. `Updater.Start` may call
the newly extracted preparation method internally until Chunk 2 moves that
call across the viewer lifecycle boundary.

Tasks:

- [ ] Add an instance field with signature
      `func() (update.Verifier, error)` to `Updater`; document that it exists to
      keep verifier construction lazy and testable without a package-level
      mutable seam.
- [ ] Initialize the field to `update.NewSigstoreVerifier` in `New`.
- [ ] Add `SetVerifierFactory` with a narrow contract. Decide and document nil
      handling according to Open Decision D2 before implementation.
- [ ] Add `EnsureClient() error` with the idempotence, retry, and exact client
      configuration listed in Recommended design.
- [ ] Refactor the top of `Start` to use `EnsureClient` while preserving its
      current log text and all behavior. Do not move `done.Begin()` or create a
      goroutine on preparation failure.
- [ ] Update `Updater`, `Client`/`SetClient`, and `Start` comments so none still
      claim that raw construction is embedded only in `Start`.
- [ ] Add unit tests proving:
  - a pre-set client bypasses the verifier factory and remains unchanged;
  - successful preparation creates a non-nil client and invokes the factory
    once across repeated calls;
  - failed preparation returns the sentinel error via `errors.Is`, leaves the
    client nil, and retries the factory on the next call;
  - `Start` still leaves `Done().Begun()` false when preparation fails during
    this compatibility chunk.
- [ ] Use a local fake `update.Verifier`; do not contact TUF, GitHub, or HTTP.
- [ ] Run:

  ```sh
  make fmt
  go test -timeout 2m -race -count=1 ./internal/ui/autoupdate \
    -run 'TestUpdater_(EnsureClient|Start)'
  ```

### Primary-agent review gate 1

The primary agent will inspect the entire diff and fix any issue before Chunk
2. Review must confirm:

- factory state is per `Updater`, not package-global;
- existing `SetClient` tests still bypass production verifier creation;
- failure never assigns a partial client or begins completion;
- repeated failure remains retryable;
- success uses the same timeout, clock, verifier, and stage directory as the
  pre-change code;
- no goroutine, lock, user string, or unrelated formatting change appeared.

Then run:

```sh
make fmt-check
go test -timeout 2m -race -count=1 ./internal/ui/autoupdate
```

## Chunk 2 — Move preparation before lifecycle begin

**Subagent:** `gpt-5.6-sol`, reasoning `high`

**Files:**

- Modify `internal/ui/autoupdate.go`
- Modify `internal/ui/autoupdate/updater.go`
- Modify `internal/ui/autoupdate_test.go`
- Modify `internal/ui/autoupdate/updater_test.go` only when the final `Start`
  precondition requires an adjusted unit test
- Do not modify documentation or `todos.md`

Tasks:

- [ ] In `maybeStartUpdateCheck`, preserve the existing gate order through
      asset availability.
- [ ] Call `v.updater.EnsureClient()` after those gates and before
      `v.updateOp.begin()`.
- [ ] On preparation failure, call
      `fyne.LogError("update verifier unavailable", err)` and return. Preserve
      exact user-visible/log behavior.
- [ ] Only after successful preparation, begin the lifecycle token and hand
      its context/staleness closure to `Updater.Start`.
- [ ] Remove lazy construction from `Start`. Apply the resolved D1 behavior for
      an invalid direct call with no prepared client; never nil-dereference and
      never begin `Done` in that state.
- [ ] Leave the check/download goroutine body byte-for-byte equivalent where
      practical. In particular, preserve the stale check after `Check`, the day
      persistence point, and the deliberate retention of a completed stage
      after cancellation during download.
- [ ] Add a focused viewer-boundary regression test. Setup:
  - build with `newTestViewer(t)`;
  - set a valid current version and set `v.settings.checkForUpdates = true`
    directly, avoiding the setter's immediate check;
  - skip through the existing asset helper on unsupported GOOS/GOARCH;
  - install a verifier factory that returns a sentinel error;
  - create a synthetic existing `prior := v.updateOp.begin()` and record
    `currentRevision()` before calling `maybeStartUpdateCheck`.
- [ ] After the call, assert all of these:
  - verifier factory was called exactly once;
  - `currentRevision()` did not change;
  - `prior.current()` remains true and its context is not cancelled;
  - `v.updater.Client()` remains nil;
  - `v.updater.Done().Begun()` remains false.
- [ ] Do not use sleep. No goroutine should start on this path.
- [ ] Run:

  ```sh
  make fmt
  go test -timeout 2m -race -count=1 ./internal/ui/autoupdate ./internal/ui \
    -run 'TestUpdater_(EnsureClient|Start)|TestUpdateCheck_VerifierFailurePreservesLifecycle'
  ```

### Primary-agent review gate 2

The primary agent will review and fix the diff before any documentation work.
Review must confirm:

- `EnsureClient` is after all cheap gates but before `begin`;
- no real verifier/TUF work runs while opt-in is off, version is invalid, the
  check is not due, or the platform has no asset;
- test failure cannot be masked by `Done.Wait()`'s never-begun behavior;
- synthetic prior token directly proves both revision and context preservation;
- `SetCheckForUpdates(false)` still invalidates a real in-flight check;
- existing staged-download behavior remains unchanged.

The primary agent then performs a mutation check:

1. Temporarily move `v.updateOp.begin()` above `EnsureClient`.
2. Run only `TestUpdateCheck_VerifierFailurePreservesLifecycle` and require it
   to fail on revision/token preservation.
3. Restore the intended ordering immediately.
4. Rerun the focused test and require success.
5. Inspect `git diff` to ensure no mutation residue remains.

Then run the broader updater subset:

```sh
make fmt-check
go test -timeout 3m -race -count=1 ./internal/ui/autoupdate ./internal/ui \
  -run 'TestUpdater_|TestUpdateCheck_|TestSetCheckForUpdates|TestDrain_WaitsUpdateDone'
```

Local-port permission may be required for `httptest.Server` cases.

## Chunk 3 — Reconcile architecture and work tracking

**Subagent:** `gpt-5.6-terra`, reasoning `medium`

**Files:**

- Modify `internal/ui/autoupdate.go` comments only if still stale
- Modify `internal/ui/viewer.go` comments only
- Modify `ARCHITECTURE.md`
- Modify `todos.md`
- Do not modify executable code or tests

Tasks:

- [ ] Update `maybeStartUpdateCheck` documentation to say it prepares the
      verifier/client before beginning the lifecycle token, then calls `Start`.
- [ ] Update `viewer.updateOp` ownership comment without weakening the locked
      decision that cancellation remains in `internal/ui`.
- [ ] Update both `ARCHITECTURE.md` updater rows so the package map accurately
      names `EnsureClient` and `Start` responsibilities.
- [ ] Move the open lifecycle-order item from `## TODO` to `Done -> Internal`.
      Record the restored order, instance-owned verifier seam, focused
      lifecycle regression test, and successful mutation result. Do not claim
      CI/full-suite success until Chunk 4 reports it.
- [ ] Leave the two Qodana TODO entries and the Windows edge-case note intact.
- [ ] Run:

  ```sh
  make fmt-check
  git diff --check
  ```

### Primary-agent review gate 3

The primary agent will compare every documentation statement against the
accepted code and mutation evidence, fix inaccuracies, and confirm:

- architecture ownership did not drift;
- no package/file move occurred;
- only the completed TODO moved;
- no translation or unrelated release-note content changed.

Then run the focused updater tests once more.

## Chunk 4 — Independent audit and repository verification

**Subagent:** `gpt-5.6-terra`, reasoning `high`, read-only unless the primary
agent sends back one specific fix.

Tasks:

- [ ] Read `AGENTS.md`, `ARCHITECTURE.md`, this plan, and the complete diff.
- [ ] Audit lifecycle ordering, retry behavior, cancellation, completion
      signaling, test seam placement, comments, and TODO bookkeeping.
- [ ] Confirm no package-level mutable seam or untracked goroutine was added.
- [ ] Confirm the regression test fails for the ordering mutation described in
      Gate 2 and does not merely assert that no goroutine started.
- [ ] Run CI-equivalent checks from repository root:

  ```sh
  make fmt-check
  go vet ./...
  go build ./...
  go test -timeout 20m -race ./...
  ```

- [ ] Report shortest decisive output for any failure. Do not edit files in
      this chunk.

### Primary-agent final review and handoff

The primary agent will review the auditor's findings, inspect the final diff and
`git status`, fix any problem personally, and rerun affected checks. Final
handoff must include changed files, ordering guarantee, mutation result, full
verification result, any environment caveat, and a suggested commit message.
No commit will be created.

Suggested commit message:

```text
fix: prepare updater before lifecycle begin
```

## Acceptance criteria

- Verifier/client construction happens only after existing update gates pass.
- Construction succeeds before `updateOp.begin()` runs.
- Verifier construction failure logs the existing message and leaves lifecycle
  revision, prior token context, client, completion signal, and goroutine state
  unchanged.
- Successful checks retain existing cancellation, staleness, day persistence,
  download, and stage semantics.
- A pre-set test client bypasses production verifier construction.
- Failed preparation is retryable; successful preparation is idempotent.
- The focused regression test fails when `begin` is moved back above client
  preparation and passes with intended order.
- `ARCHITECTURE.md` describes the new flow accurately.
- The completed item moves out of open TODOs only after code and mutation checks
  pass.
- All CI-equivalent checks pass.
- No unrelated files, translations, golden images, or commits are produced.

## Open decisions

Resolve these before dispatching Chunk 1. Recommended defaults make the plan
executable without redesign.

### D1 — Invalid direct `Start` call

**Recommended:** change `Start` to return an error when no client has been
prepared. `maybeStartUpdateCheck` treats this as an impossible invariant breach
after `EnsureClient`, logs it, and no completion/goroutine begins.

Alternative: keep `Start`'s no-return signature and defensively log/return on a
nil client. Smaller call-site diff, but less explicit and harder to unit-test as
an API contract.

### D2 — Nil verifier factory

**Recommended:** `SetVerifierFactory(nil)` restores
`update.NewSigstoreVerifier`. This mirrors a resettable test seam and prevents a
nil-function panic.

Alternative: reject nil by panic or leave nil unsupported. Smaller method, but
more fragile cleanup behavior in tests.

### D3 — Commit cadence

**Recommended:** execute all chunks sequentially, review/fix after each, then
offer one combined commit message at final handoff.

Alternative: stop after each accepted chunk so you can commit that unit before
the next subagent starts. The primary agent still never runs `git commit`.
