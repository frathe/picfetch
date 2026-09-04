# Orchestrate cancellable mosaic generation

Type: task
Status: resolved
Priority: P0
Blocked by: 05, 07

## Goal

Run `mosaic.Generate` without blocking Fyne, reject every stale completion, and
make all generations observable to deterministic tests and shutdown cleanup.

## Existing code anchors

- `internal/ui/requestLifecycle` is unexported and intentionally belongs to the
  viewer package; `mosaicwin` must not export or import it.
- `internal/ui/compare` is the relevant feature-package pattern: a package-local
  cancellable revision, `completion.Signal`, a worker tracker that includes
  superseded generations, a per-instance `UIQueue`, and `Settle(ctx)`.
- Fyne's test driver runs `fyne.Do` inline. `newTestUI` therefore installs
  drainable queues on Grid and Compare; Mosaic needs the same treatment.

## Scope

- Give `mosaicwin.Window` a package-local context/revision lifecycle. Generate
  and Regenerate cancel and permanently supersede the previous token.
- Capture a validated `mosaic.Request` on the UI goroutine, then run
  `mosaic.Generate` on a tracked worker. No widget or mutable window state may
  be read from that worker.
- Check token currency before expensive work where possible, before queueing a
  completion, and again inside the queued completion. Always finish the stale
  worker's own completion handle.
- Add a `UIQueue` interface and `SetUIQueue` to `mosaicwin`, defaulting to
  `fyne.Do`; tests install `*uitest.UIQueue` per window. Do not add a mutable
  package-level UI seam.
- Implement `Settle(context.Context) error` as a causal barrier: wait all worker
  generations, drain queued completions, and repeat until both are empty. A
  latest-only `completion.Signal.Wait` is insufficient after supersession.
- Close/Cancel invalidates work before clearing widgets. A cancelled or stale
  result may finish internally but must not change activity, error, preview, or
  action state.
- In `internal/ui`, install the test queue in `newTestUI`; add Mosaic Close and
  `Settle` to `drain`, and invalidate/close it from `registerShutdown` before
  the event loop stops.
- Localize any new activity, cancellation, or failure text in this ticket and
  update Qodana for every new test path.

## Acceptance Criteria

- A blocking fake generator proves window controls and the UI goroutine remain
  responsive while work is active.
- Two rapid generations can complete in reverse order; only the second may
  publish a preview or status.
- Close and Cancel propagate `context.Canceled`, wait cleanly in `Settle`, and
  allow no later queued UI mutation.
- `Settle` waits both a superseded worker and work started by a drained
  completion, with no sleep-based assertions.
- Generator source errors are presented once; source-skipping policy remains in
  `internal/mosaic`, not duplicated in the window.

```sh
go test ./internal/ui/mosaicwin -run 'TestMosaic(Generate|Supersede|Cancel|Settle)' &&
go test ./internal/ui -run 'TestMosaic(Drain|Shutdown)' &&
make check-qodana-test-exclusions
```

## Non-Goals

- A second copy of viewer `requestLifecycle`
- UI-side source decoding or retry policy
- Sleep-based synchronization

## Comments

Ticket 07 now owns the window shell. This ticket owns only its asynchronous
generation state, removing the original overlapping file ownership.

Implemented and verified on 2026-09-04: tracked asynchronous work, cancellation,
reverse-order stale rejection, causal Settle, drain, and shutdown tests are green.
