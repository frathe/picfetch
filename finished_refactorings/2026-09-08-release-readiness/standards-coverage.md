# PicFetch Standards release-review coverage

Comparison: `git diff 73a5f6381ab8e643de47a6c2038ed1d929f11670...d306654e49cec03432070d6219cc920cac9c2b72`; all 30 commits read from `commits.txt`. Scope stays pinned despite integrator edits after review started.

Sources read: AGENTS.md, ARCHITECTURE.md, CONTEXT.md, .github/CONTRIBUTING.md, .agents/skills/improved_sdd_tdd_cycle.md, docs/agents/issue-tracker.md, code-review/SKILL.md and full smell baseline. User explicitly authorized independent review agents and bounded fix agents, overriding the local process delegation restrictions.

## Inventory and review depth

539 changed paths: 104 Go files excluding `_test.go` (including three test-fixture files under internal/uitest), 85 Go test files, 6 workflow/packaging files, 306 archived plans/evidence/pet assets, 38 other docs/config/assets. Generated evidence was classified as historical evidence, not exact-head proof; bulk pet images and JSON are not executable application behavior. The embedded Trane atlas loader and pointer/layout code were reviewed.

Inspected changed executable behavior across:

- Launch parser/main/UI application and nonpersistent overrides; native open/save transport and UI admission.
- TIFF/EXIF/RAW bounds; JPEG geometry correction and metadata omission; atomic Save/Strip/Export transactions; saved-rotation adoption and post-write reconciliation.
- Complete full-image cache records, revision-bound cache writes, thumbnail cancellation and versioned favorites, prewarm budget.
- Duplicate facts/grouping/index acceleration, grouping admission, close/reopen sessions, progressive hide, browse identity, selection/filter/sort lifecycle.
- Clipboard workers and shared admission, region-copy integration, deletion identity reconciliation, native integrations changed for UTF-8/XDG/errors.
- EXIF metadata/removal/map lifetimes, bounded tile queue, warm/foreground completion, shutdown.
- Animated frames and picture-frame worker acknowledgement; poller Stop/Done and singleton retention; menu/shutdown interaction.
- Mosaic bounded/tiled preparation, separable filtering, canvas coverage, immutable transient preview, window cancellation/export.
- Choice-card extra rows/focus, native menu wiring, Trane startup/pointer behavior.
- Native test runner, pinned packaging tools/images and changed release/Store workflows.
- Store approval identity, release/artifact/tag validation, notes/listing preservation, receipt journal, recoverable lifecycle and credential-safe request errors. Root separately owns CodeQL archive finding and Qodana adjudication.

Test review was risk directed (new concurrency/regression tests and relevant fixtures), not line-by-line reading of all 85 test files. Runtime test run by this reviewer before fixes: deterministic read-only overlay reproduction of queued slideshow Kick race (failed as intended). Root owns final broad tests/CI and native packaging evidence.

## Initial Standards result

Two accepted P2 findings; worst P2. One independently reproduced here: queued picture-frame advances can execute after manual navigation because Kick invalidation happens only when the worker observes the channel. Original test waited after Kick and masked the race. A read-only Go overlay using the existing heldSubmissionQueue failed deterministically. This violates AGENTS.md Concurrency and Fyne stale-result/queued-callback discipline and ARCHITECTURE.md's explicit Kick-discard contract.

The second, source-case aliases bypassing serialized file writes and UI reconciliation, was forwarded by the coordinator and accepted against the same-source transaction/current resolved source invariant. Root owns its native reproduction and fix; this reviewer did not duplicate the experiment.

No additional material baseline smell findings. Narrow Host adapters, State snapshots, per-feature UI queues and independent lifecycles are explicitly repo-endorsed, so their superficial resemblance to Middle Man/Data Clumps/Duplicated Code was not flagged. Tool-enforced style/parity/Qodana checks are left to the integrator.

## Limits

No blanket guarantee. This static review and focused native test do not prove Windows behavior, macOS packaged GL behavior, exact final-head CI, real Store environment permissions, or a Developer-role submission. Store must remain protected by a fresh user approval and ordinary release, with no real submission used as a test. Root tracks all of those independently. No repository files were edited during independent review; a subsequent bounded slideshow fix is separately authorized.

## Bounded fix result

Authorized edit scope: only internal/ui/slideshow/slideshow.go and existing slideshow_test.go. `Kick` now increments an atomic countdown revision synchronously; timed callbacks compare the captured revision before calling Advance, independently of the worker receiving its reset channel. Existing stop/acknowledgement behavior is retained. The existing Kick regression now drains immediately after Kick, exercises both queued and held-submission workers, and proves the next countdown still advances once. No test-file or top-level root-UI test additions, so no Qodana path or UI shard additions are required for this fix.

Red: `go test ./internal/ui/slideshow -run '^TestKickDiscardsAlreadyQueuedTimedAdvance$' -count=1` failed both `queued` and `worker_still_submitting` cases with the stated unwanted advance.
Green after final test strengthening: `go test -race ./internal/ui/slideshow -count=1` -> `ok github.com/frathe/picfetch/internal/ui/slideshow 1.384s` on native macOS. `git diff --check` for both owned files and `gofmt -l` are clean. Root integrator will independently verify and own final-head validation. Suggested commit: `fix: invalidate queued picture-frame advances on manual navigation`.
