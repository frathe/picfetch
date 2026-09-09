# PicFetch — Open Refactoring Backlog

Updated 2026-09-09 following user acceptance of the Windows validation closeout.

This file contains remaining work and accepted dependency watches. Completed
findings have been removed; their history remains in Git and the
[maintainability implementation plan](finished_refactorings/2026-09-06-maintainability-plan.md).
Existing MA identifiers are preserved. MA-025 carries forward the separate
decoded map-cache follow-up previously recorded under MA-015 and in
[todos.md](todos.md).

| ID | Priority | Remaining work | Status |
| --- | --- | --- | --- |
| [MA-023](#ma-023) | P3 | Retire the HEIC fork when an official release contains its fix | Accepted dependency watch |
| [MA-024](#ma-024) | P3 | Measure updater verification dependency cost at its next major upgrade | Accepted dependency watch |
| [MA-025](#ma-025) | P3 | Bound the upstream map widget's retained decoded tiles | Open follow-up; workload impact unmeasured |

<a id="ma-020"></a>

MA-020 closed on 2026-09-09: the user reported successful Windows 11 ARM and
x64 testing and accepted the remaining detailed checks as edge cases. See the
[Windows acceptance record](finished_refactorings/2026-09-07-maintainability/windows-test-todo.md).
This records user acceptance and waived checks, without claiming a WACK pass.

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

## Related verification record

The local race-test investigation is closed under the accepted memory
mitigation and evidence-retention contract. See the
[September 9 completion record](finished_refactorings/2026-09-09-local-race-evidence.md).
The original September 7 package-only failure remains unexplained; later OOM
observations do not establish its cause. It is no longer open audit work.
