# PicFetch maintainability implementation plan — 2026-09-06

Status: proposed; no refactor implemented. Canonical findings and severity are in [needs_refactoring.md](../needs_refactoring.md). This plan references its stable MA IDs and does not maintain a competing findings list. [Audit validation](2026-09-06-maintainability-validation.md) records the baseline and probes.

## Objective and constraints

Prevent input-driven crashes/resource blowups, restore identity/metadata correctness, and make asynchronous work bounded and observable. Improve existing package contracts where the audit demonstrates a gap. Preserve public behavior except the identified defects; record deliberate behavior changes in each implementation PR.

This is a sequence of reviewable work packages, not one bulk refactor. Effort estimates are focused engineer-days including meaningful tests and review, not calendar promises. S = up to 1 day, M = 1–3 days, L = 3–5 days. Native setup can extend elapsed time. Risk describes implementation/regression risk, not audit severity.

Follow the current agent working agreement when executing a package; re-read it rather than treating historical plan instructions as current. Use existing instance-owned lifecycle/queue seams. Add tests that prove the behavior, not mechanical tests of new helpers. New `_test.go` files require exact Qodana exclusions; new internal/ui top-level tests require shard assignments/header counts. Update ARCHITECTURE only if package/file placement changes. All open implementation work stays linked from todos.md. Move accepted/completed plans to the established archive with links updated according to repository process. This audit's explicitly authorized documentation commit does not grant general permission to commit future implementation work.

## Phase 1 — Contain the high-impact input failures

Recommended first implementation phase. Each package can be reviewed independently; neither should wait for a UI redesign.

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| A: Checked TIFF spans | MA-001 | None | M / medium | Arbitrary offsets/lengths cannot panic orientation, metadata or RAW IFD walking. Both endian formats, MaxUint32, near-end entry addresses and nested/truncated IFDs return tolerant results/errors. Existing valid metadata survives. |
| B: Bounded mosaic preparation | MA-002 | None | L / high | Wide/tall source ratios through 10000 cannot request scratch beyond an explicit budget. Check planned allocations before executing them. Ordinary seeded layouts and rotated edge fidelity remain correct; cancellation prevents further preparation. |

A should deepen the existing metadata boundary by sharing checked arithmetic, not merge read/write semantics. B should preserve crop/placement mathematics: clipping or tiled sampling changes allocation strategy, not photo geometry. Do not use a real OOM test as the regression test.

Commands during iteration:

```sh
go test ./internal/imaging -run 'Test.*(EXIF|Exif|Metadata|RAW|Raw|Orientation)' -count=1
go test ./internal/mosaic ./internal/ui/mosaicwin -count=1
```

Add a bounded fuzz run for the new TIFF entry-point target after deterministic seeds pass. Use its actual added name in the implementation record. Existing golden output must remain stable; if an intentional visible change is necessary, use `make golden` in Linux/amd64 Docker and inspect differences. Finish each mergeable change with the common gate below.

## Phase 2 — Correct identity, file transport and complete image records

This phase is separable into small changes. A platform path fix should not wait for a new async framework.

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| C: Deletion target identity | MA-004 | None | M / medium | Prompt A, reorder, confirm: only A leaves disk and current list; B stays. Cover fresh drop, partial trash failure and generation change during worker completion. All OS mutations stubbed. |
| D: Complete shared image records | MA-006 | None | S–M / low | Foreground, preload and comparison preserve identical size/EXIF facts for the same bytes; comparison-first navigation displays correct info. Preserve Add/AddIfFits and stale-request rules. |
| E: Rotated JPEG metadata | MA-007 | A useful but not required | S–M / medium | Save Changes writes actual numeric dimensions for both quarter turns/orientations 5–8; unchanged geometry and unrelated metadata remain intact. Replace the old presence-only assertion. |
| F: Lossless native path transport | MA-005, MA-012 | None | M / medium | Save result is exactly the confirmed path. Open lists round-trip newline/trailing-CR/space/Unicode filenames. Windows PowerShell decodes accented/CJK/emoji UTF-8 paths correctly. Cancellation remains distinct from error. |
| G: Correct local error/environment behavior | MA-011, MA-013 | None | S each / low for clipboard, medium for trash | A failed temp write/close returns its primary error even if removal succeeds; normal XDG overrides survive while demonstrated sandbox redirection is normalized. |
| H: Small freshness/test fixes | MA-018, MA-019 | None | S total / low | Same-size subsecond changes invalidate previews; increasing/equal/decreasing RSS arithmetic cannot wrap. Preserve optional leak-test gating. |

Prefer separate commits/PRs for unrelated entries in G/H during eventual implementation, subject to that session's commit authorization. They are grouped here only for scheduling.

Commands:

```sh
go test ./internal/ui/deletion -count=1
go test ./internal/ui -run 'Test.*(Compare|Cache|Info)' -count=1
go test ./internal/imaging -run 'Test.*(SaveRotated|Export|JPEG|Exif|EXIF)' -count=1
go test ./internal/clipboard ./internal/filepicker ./internal/trash ./internal/favthumbs -count=1
```

Run the added RSS arithmetic test by its actual name; `heicleak` files are excluded from ordinary commands. F additionally requires native Windows PowerShell decoding execution without invoking the real clipboard, and native macOS chooser transport validation without overwriting user files. Unit tests inspecting a generated script alone do not close MA-012.

## Phase 3 — Make worker, cancellation and publication contracts explicit

Use narrow per-feature changes, starting with deterministic failing tests. Do not introduce a universal worker registry or hold the UI thread while waiting for a queued callback.

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| I: Queue-aware animation and admission | MA-003 | None; can start beside phases 1/2 | M / high | A queued driver proves correct heterogeneous frame delays and no worker/UI shared mutable locals. Slideshow stale callbacks cannot advance a closed session. Chooser guard/results run on the right goroutine. Cancel while a callback is pending without deadlock. |
| J: Context-bearing ancillary reads | MA-010 | None | M / medium | Controlled chunked read stops on sort/preview cancellation; a cancelled semaphore waiter never decodes. Worker completion remains observable. Decoder interruption limits are documented honestly. |
| K: Generation-conditional fact publication | MA-009 | J useful; identity tests independent | M / high | Pause old same-URI success/failure, advance generation/reset, then release: no old hash/failure/native dimensions enter the new model. Test reorder/adoption separately from source replacement. |
| L: Bounded tile lifetime | MA-015 | I/J patterns useful, no shared abstraction required | M–L / medium | Concurrent map warms and foreground tiles share an enforced limit; superseded/closed work cancels; failed metadata has an expiry/cap. Completion covers callback effects, not only pending counters. |
| M: Native position-poller lifecycle | MA-016 | I queue test pattern | M / high | Stop before/after enqueue or during native read has defined cancellation and completion behavior. No queued read updates a closed target. Event-loop shutdown cannot wait cyclically on itself. |
| N: Background image action work | MA-014 | I/J; preserve D record contract | M–L / high | Clipboard encoding, metadata read/removal and Save Changes leave UI responsive with a gated slow source. Capture source identity; serialize same-file mutations, preserve atomic writes and report errors once. Close/navigation cannot install old UI state. |

While changing shutdown ownership, compare production shutdown against harness drain for favorite preview, toast, picture-frame, grid and secondary-window workers. Fix proven omissions in the owning feature; do not silently convert every completion.Signal into a broader promise. Grid/compare/mosaic queue/wait ordering in AGENTS.md remains load-bearing.

Commands:

```sh
go test ./internal/ui/slideshow ./internal/ui/grid ./internal/ui/exifwin ./internal/winpos -count=1
go test ./internal/imaging ./internal/filesort ./internal/favthumbs ./internal/dupes -count=1
go test ./internal/ui -run 'Test.*(Anim|Slideshow|Chooser|Clipboard|Save|Exif|Shutdown)' -count=1
make check-test-shards
```

These focused selections are examples over existing names, not a substitute for verifying that each new regression actually ran. Record new exact test names in the PR. Use the canonical Docker race gate at completion; a passing inline-driver race suite alone does not close queue-related findings.

## Phase 4 — Remove avoidable large-folder work and reduce policy drift

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| O: Reusable cancellable group snapshots | MA-008 | K; I/J patterns | L / high | Search changes reuse unchanged groups; distance/hash changes schedule cancellable computation. Superseded snapshots never publish. Benchmarks at 10k/50k/200k cover unrelated and dense hashes without running a quadratic 200k baseline on UI. Golden membership/representative fixtures preserve greedy complete linkage. |
| P: Foreground-aware preview scheduling | MA-021 | J | M / medium; optional after measurement | Measure cold-favorite/interactive latency first. Adopt a bounded yield/share policy only if needed; foreground gets capacity, disk cache still converges when idle, and cancellation does not sweep unseen valid previews. |
| Q: Small shared command admission policy | MA-022 | I; no broad mode refactor | M / medium; opportunistic | Document and test menu/keyboard parity for comparison, Copy Selection, grid, inspect and picture-frame mode, including Escape priority. Centralize repeated decisions only; retain intentional exceptions. |

For O, compare algorithm results against the current implementation on bounded randomized and adversarial inputs. Do not substitute transitive grouping. First eliminate recomputation; choose an index only when profiling establishes the remaining bottleneck. Record hardware, non-race/race status, inputs and wall time with benchmarks.

Commands:

```sh
go test ./internal/imaging ./internal/dupes ./internal/ui/grid ./internal/favthumbs -count=1
go test ./internal/ui ./internal/ui/menus -run 'Test.*(Key|Menu|Compare|Selection|PictureFrame|Inspect)' -count=1
```

Add benchmark names to the work package when implemented and run those exact benchmarks with `-benchmem`. Compare foreground latency as well as throughput for P.

## Phase 5 — Exercise shipped platform contracts and stabilize packaging

| Work package | IDs | Dependency | Effort / risk | Concrete acceptance |
| --- | --- | --- | --- | --- |
| R: Native/tag/GL coverage | MA-017 | Can start early; validates F/I/M | M–L / low production risk | CI logs enumerate wallpaper Windows guards, actual macOS class-graft test, and Store-tagged distribution test. A maintained native GL smoke verifies comparison pan/zoom/rotation/swipe at representative DPI and large-source detail. Keep Linux golden baseline. |
| S: Reviewed packaging inputs | MA-020 | None | S–M / low | Local/release/Store tool versions come from reviewed constants; cross images are pinned or explicitly versioned with recorded resolution. Build logs identify inputs; packaging smoke runs without publishing. |

Native runner examples (run on their named OS with required C/GL toolchain):

```sh
# Windows: include the existing platform-only guards, not only updater tests.
go test ./internal/wallpaper ./internal/clipboard ./internal/filepicker ./internal/update ./internal/ui/autoupdate -count=1
# macOS: main package includes the Objective-C graft guard.
go test . ./internal/openwith ./internal/displays ./internal/winpos -count=1
# Store behavior is selected explicitly.
go test -tags=microsoftstore ./internal/distribution ./internal/ui/autoupdate -count=1
```

Do not present cross-compilation as execution of these tests. Record the actual native GL environment and checks; headless reference images cannot validate shader compilation/output. Packaging acceptance is artifact inspection and a smoke run, not a release or Store submission.

## Accepted watch items

MA-023 remains mitigated: inspect upstream HEIC fix inclusion at the next dependency update, then run imaging regressions and the optional Linux `PICFETCH_HEIC_LEAK_TEST=1 go test -tags=heicleak -run TestHEICDecode_DoesNotGrowRSSUnbounded ./internal/imaging/...` after MA-019. Keep the fork until fix inclusion is established.

MA-024 requires no immediate implementation. At the next major verifier upgrade, measure reachable dependencies/build time/binary size and review supported footprint reductions without weakening trust or provenance. Do not replace Sigstore verification with hand-written cryptography to reduce a module count.

## Quick wins, ordering and completion

Small independent changes are G's clipboard error propagation, D's record completeness, H's timestamp/RSS arithmetic and S's tool input pinning. They can accompany the first phase as separate reviews; they must not postpone MA-001/002. E is also small but changes an intentional metadata policy and needs value-based tests.

Dependency sketch: I/J enable safe K/L/M/N; K enables O; J enables P. R can start at any time and provides the native evidence needed to close F/I/M. All other work packages are independent unless the current code changes that assessment. Do not batch all UI concurrency changes into one PR.

For each implemented work package:

1. Recheck the pinned finding against current HEAD and preserve unrelated changes.
2. Prove the regression with controlled inputs/queues; assert behavior and file identity, not only widget Visible flags or counters.
3. Implement the smallest owning-boundary change, check targeted tests, and update exact shard/Qodana entries where required.
4. Run `make verify` for each mergeable change. Use `make golden` only for intended render changes and inspect generated differences; never commit failed images.
5. Add native/benchmark evidence required by that package. A passing common gate does not waive an unmet platform acceptance criterion.
6. Mark the MA entry resolved with commit/test evidence, update todos/plan status, and retain the historical audit reference. Keep unresolved watch items open.

The documentation-only audit already passed the baseline gate; no code has been changed by this plan. The existing architecture's Host/snapshot seams, cache semantics, deterministic render references, explicit feature composition and update defenses are explicit keep-as-is constraints.
