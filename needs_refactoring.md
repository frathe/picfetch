# PicFetch — Open Refactoring Backlog

Updated 2026-09-07 against branch head `9f12593`.

This file contains remaining work and accepted dependency watches. Completed
findings have been removed; their history remains in Git and the
[maintainability implementation plan](plans/2026-09-06-maintainability-plan.md).
Existing MA identifiers are preserved. MA-025 carries forward the separate
decoded map-cache follow-up previously recorded under MA-015 and in
[todos.md](todos.md).

| ID | Priority | Remaining work | Status |
| --- | --- | --- | --- |
| [MA-020](#ma-020) | P3 | Reconcile native Windows package and WACK acceptance | Verification record unresolved |
| [MA-023](#ma-023) | P3 | Retire the HEIC fork when an official release contains its fix | Accepted dependency watch |
| [MA-024](#ma-024) | P3 | Measure updater verification dependency cost at its next major upgrade | Accepted dependency watch |
| [MA-025](#ma-025) | P3 | Bound the upstream map widget's retained decoded tiles | Open follow-up; workload impact unmeasured |

<a id="ma-020"></a>

## MA-020 — Reconcile native Windows package acceptance

**Remaining issue.** Packaging tools and container images are already pinned in
[packaging/tools.mk](packaging/tools.mk), and local/release/Store routes share
those inputs. The outstanding question is whether native x64 graphical startup
and Windows SDK/WACK acceptance have been completed for the identified artifacts.

The local `.scratch/maintainability/windows-test-todo.md` has every box checked,
but its introduction, ticket 27 and the tracked implementation plan still say
these checks are deferred. The checklist refers to Phase 6 executables, so it
also cannot establish validation of later branch-head binaries. `.scratch/` is
ignored and its checklist/evidence is unavailable in a fresh clone. Treat this
as an unresolved evidence record, rather than inferring either a pass or an
unperformed test from the checkboxes alone.

**Work to finish:**

- Recover the results behind the checked boxes. Record the source revision,
  executable SHA256 hashes, test date, Windows version, CPU/GPU, graphics driver
  and display scaling in a tracked completion record. Distinguish reported
  manual observations from retained logs and screenshots.
- If native x64 evidence is missing, test both ordinary and Store-tagged x64
  executables on native x64 Windows with a working matching graphics driver.
  Check Alpha/Beta fixture rendering and navigation, Grid, comparison pan/zoom,
  swipe/divider and full-source detail, progressive duplicate hiding and
  navigation during favorite prewarming. Retain clean-exit results and failures
  against the exact artifact hashes.
- If Store acceptance evidence is missing, refresh x64/ARM64 staging from the
  identified executables and run MakeAppx pack/bundle, temporary SignTool signing
  and verification, and WACK using the
  [Store workflow](.github/workflows/microsoft-store.yml). Retain package
  identity/architecture metadata, command results and the certification report;
  clean up only package/certificate/trust entries created for that test.
- Reconcile the completion status with the tracked implementation plan. Preserve
  the artifact scope of existing results; rerun only checks whose evidence is
  missing or whose relevant inputs changed.

The earlier x64 attempts in an ARM64 VM failed graphics initialization or drawing
and do not establish native x64 acceptance. Further VM experiments were deferred
by the user. Existing macOS/Linux package and Windows ARM64 results need not be
repeated merely to reconcile this record. Follow the maintained
[packaging procedure](docs/packaging-inputs.md) for any missing checks.

**Done when:** a consistent tracked record identifies successful native x64
ordinary/Store smoke results and Windows SDK/WACK results for the tested
artifacts. Any remaining failure keeps this entry open. Release or Store
submission is not required.

<a id="ma-023"></a>

## MA-023 — Retire the HEIC fork after an upstream release includes its fix

**Accepted watch; trigger: the next separately scoped HEIC dependency update.**

[go.mod](go.mod) requests `github.com/gen2brain/heic v0.7.1` and replaces it with
`github.com/frathe/heic v0.0.0-20260820164529-0ac0a39f8206`, which carries the
native memory-leak fix documented in [AGENTS.md](AGENTS.md). The remaining debt
is maintaining this fork. Upstream release status was not rechecked for this
backlog rewrite.

**Work at the trigger:**

- Inspect the proposed official release/tag and verify that it contains the
  actual fix before removing the replacement. Keep the fork if inclusion cannot
  be established.
- Verify native decoding and the WASM/no-cgo fallback, including existing format,
  orientation and metadata regressions.
- Run imaging tests and the [opt-in RSS check](internal/imaging/heic_leak_test.go)
  below. The latter needs Linux RSS
  support; establish that the native decoder is actually exercised before
  using the result as evidence for the native leak fix. A skipped check or a
  fallback-only run does not establish that result.

```sh
go test ./internal/imaging -count=1
PICFETCH_HEIC_LEAK_TEST=1 go test -tags=heicleak ./internal/imaging -run '^TestHEICDecode_DoesNotGrowRSSUnbounded$' -count=1 -v
```

**Done when:** an official version containing the fix replaces the fork, the pin
notes are updated, and decode/RSS evidence supports the migration.

<a id="ma-024"></a>

## MA-024 — Assess updater verification dependency cost

**Accepted watch; trigger: the next major verifier dependency upgrade.**

The verifier boundary in [internal/update/attest.go](internal/update/attest.go)
uses Sigstore and TUF. [go.mod](go.mod) currently lists Sigstore `v1.10.9`,
sigstore-go `v1.3.0` and go-tuf/v2 `v2.4.2`. Their present contribution to build
time and binary size has not been measured; there is no quantified footprint
regression to fix now.

**Work at the trigger:**

- Capture reachable dependencies, build time and executable bytes before and
  after the proposed upgrade. Record platform, toolchain, build flags and cache
  conditions so the comparison is meaningful.
- Assess supported ways to reduce the verification dependency graph while
  preserving signature/provenance and trust validation. Keep traversal/symlink
  defenses, download bounds and rollback behavior intact.
- Record a measured decision to retain the current approach or implement and
  validate a supported reduction. Do not replace the verifier with ad-hoc crypto
  solely to reduce dependency count.

Useful starting commands:

```sh
go list -deps ./internal/update
time make build
wc -c bin/picfetch
go test ./internal/update ./internal/ui/autoupdate -count=1
```

**Done when:** the upgrade assessment records measured costs and a justified
choice, with regression evidence for any implementation change. Retaining the
existing verifier is a valid outcome.

<a id="ma-025"></a>

## MA-025 — Bound retained decoded map tiles

**Open follow-up.** The pinned `fyne.io/x/fyne` module
`v0.0.0-20260712112324-6989f2f174fb` keeps decoded tiles in a process-global
`map[string]image.Image` in `widget/mapcache.go`, with no eviction. Visiting new
map locations can retain more decoded images across EXIF window close/reopen.
The missing bound is confirmed in source; long-session memory growth and its
user-visible impact have not been measured.

PicFetch constructs that widget in
[internal/ui/exifwin/exifwin.go](internal/ui/exifwin/exifwin.go). Its own
[tiles.go](internal/ui/exifwin/tiles.go) limits cached **encoded** tile bytes to
16 MiB and already bounds request/failure state. Those limits do not bound the
upstream decoded images. This is separate from the completed MA-015 work.

**Work to finish:**

- Measure decoded retention while visiting many unique tiles and closing and
  reopening the map. Use local tile fixtures/fake responses so the experiment
  does not generate load on public map servers.
- Select a supported upstream eviction/ownership fix or a narrowly maintained
  integration that imposes an explicit decoded-byte budget. Clearing PicFetch's
  encoded cache alone cannot solve this retention.
- Preserve nonblocking UI updates, request deduplication and warm passes,
  cancellation, eventual redraw and rejection of stale completions. Reassess
  the existing process-global log-filter workaround if tile ownership changes.
- Add a regression that observes bounded decoded retention and correct redraw
  after eviction and reopen. Run the EXIF window race suite and the required
  common gate for the resulting implementation. Use RSS observations as
  supporting evidence, alongside deterministic cache ownership/accounting.

**Done when:** decoded retention has a demonstrated bound across long navigation
and window lifetimes, with correct redraw and lifecycle behavior. Document
encoded and decoded cache budgets separately.

## Related open verification work

[Investigate the intermittent UI shard 3 failure](todos.md#investigate-the-intermittent-ui-shard-3-package-failure)
remains open. Later concurrent Docker verification runs recorded OOM events,
while the same shard passed alone; a 1 GiB Go memory target did not prevent one
failure. The original Qodana run's cause remains unknown. Capture raw test and
Docker memory/event evidence, determine whether shard scheduling or the test
memory budget needs adjustment, and verify the concurrent gate under the chosen
configuration. See the latest
[live-preview verification record](plans/2026-09-07-mosaic-live-preview.md).
This infrastructure follow-up does not reopen completed application refactors.
