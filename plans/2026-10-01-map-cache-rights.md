# Map caching and license access

Route: Standard. Owner: Pico inline; no implementation delegation (hot context).

Deliverable: EXIF tile fetches honor server freshness/validators without blocking
paint, and both map surfaces expose the OSM copyright/license page directly.

Decisions: preserve bounded in-memory storage and cancellation; do not introduce
disk persistence of viewed geographic locations. Header-aware caching avoids the
seven-day fallback requirement; memory eviction/restart still loses cache entries,
as in Location Map. Do not claim a permanent retention guarantee or blanket legal
certification. No upload, signing, App Store declaration or submission in scope.
Fixed EXIF neighborhood warming remains unchanged; its modest-look-ahead assessment
is separate, not a confirmed bulk-download violation.

## Tasks and acceptance

1. EXIF caching: retain bounded response metadata, freshness and validators;
   reuse fresh entries, conditionally validate stale entries, merge 304 updates,
   deliver no-cache/no-store responses once without an endless redraw loop.
   Cancelled jobs cannot publish; failures cannot serve stale bytes as fresh.
   Tests in existing `internal/ui/exifwin/tiles_test.go` use fake time and local
   HTTP fixtures. Verify `go test -tags no_emoji,nodynamic ./internal/ui/exifwin`.
2. Attribution: configure EXIF's existing link and replace Location Map's label
   with a direct copyright hyperlink, using the existing translated credit.
   Regression tests inspect renderer/content trees, not merely visibility.
   Verify focused tests in exifwin/locationmap; native visual check remains a
   separate qualification if desktop execution is unavailable.
3. Review focused races/vet, formatting and changed-file GoLand inspections;
   record unavailable broader gates honestly. Full Docker suite requires native
   amd64; do not run ARM emulation and claim isolation qualification.

Budget: zero spawns, at most three local repair/review rounds; one final full
verification attempt subject to environment prerequisites. Ronin subsequently
authorized committing and pushing the completed fix on the existing feature branch.

## Evidence

Implemented download-layer freshness/validators, 304 merging, bounded metadata,
one-shot delivery for no-cache/no-store, cancellation cleanup and direct license
links. Focused initial tests failed on expired reuse, no-cache/no-store reuse and
missing direct attribution before implementation, then passed.

Validation: both feature package suites pass with race detection; root
`TestLocationMap/tile_policy_and_bounds` passes with race detection. Focused vet,
full native build, fmt-check pass. The initial scope count of eight was stale;
the review loop inspected all nine changed Go files at `3eafe3d` with GoLand's
weak-inclusive inspection tools: `internal/ui/exifwin/exifwin.go`,
`exifwin_test.go`, `map.go`, `tilecache.go`, `tiles.go`, `tiles_test.go`,
`tilework.go` (all under that same package), `internal/ui/locationmap/feature.go`
and `internal/ui/locationmap_test.go`. No new actionable findings; six
duplicate-fragment weak warnings in unchanged
EXIF test fixture sections are covered by the existing exact Qodana exclusion.
Native Linux/amd64 full suite unavailable: daemon reports linux/aarch64.

## Renderer-cache discovery and authorized scope expansion

The pinned Fyne-X `widget/mapcache.go` has a package-global decoded `tileMap`
keyed by URL, with no expiry or invalidation API. Its getTile returns pixels
without calling the supplied HTTP client on a cache hit. Fixing our fetcher does
not fix that layer. A temporary diagnostic in TestLocationMapTheme rendered the
same 256x256 raster four times, waiting for tile workers after each; then advanced
the fetcher clock eight days and rendered again. Request count did not increase:
the diagnostic failed with "map renderer bypasses expired tile validation".
The diagnostic was initially removed pending scope approval, then reinstated as
a permanent regression once Ronin authorized continuing the broader fix.

Ronin authorized continuing. Replaced the Fyne-X map renderer with an owned
viewport renderer, preserving drag/zoom, a tappable photo marker and theme
filtering. No new dependency, version upgrade or copied upstream implementation;
the existing Fyne-X marker-value API and its notice remain. Decoded tiles are
normalized on workers and charged to the existing total 16 MiB cache/delivery
budget, not retained in a global decoded cache. Current display pixels are
viewport-scoped; no-store/no-cache can finish an in-progress view without a
request loop, but Hide, session replacement or camera change prevents reuse.
The formerly failing renderer expiry regression now passes; view-lifecycle
tests cover repeated repaint, cancellation/reopen, zoom center/limits and panning.
Synthetic dark-mode screenshot inspected: map area, centered marker, zoom
buttons and direct attribution link are visible. Test-only capture code removed.

No App Store rights declaration, signing or upload is authorized by this code fix.
Memory-only caches also do not promise retention after restart. No persistent
location cache was introduced.

Ledger: zero implementation spawns; scope expansion added two review/repair rounds
(recorded rather than silently exceeding the original budget). Full suite
attempt stopped at platform prerequisite. Final focused race regressions pass;
the final cache-helper GoLand inspection reports no findings. No signing or
uploads. Ronin invoked the GitHub review loop after this push; Pico owns its
assessment and fixes inline, with zero implementation delegates.

## GitHub review follow-up

The fresh review of `3eafe3d` reported five findings. Two runtime defects were
confirmed: expired displayed pixels survived failed revalidation, and neighboring
one-shot warm responses survived camera changes. Both regressions failed before
the fixes. The renderer now drops expired pixels immediately. Tile demand and
delivery carry a view version; a camera/size change or Hide purges one-shot
delivery, retires queued work, and prevents late active results or an obsolete
warm pass from publishing into the next view. Fresh cache entries are retained;
active native/HTTP work remains tracked and is joined off UI by existing barriers.

The rights research now distinguishes pre-fix observations from the implementation,
and the inspection scope above accounts for all nine files. The scroll-wheel
report is rejected: the pinned Fyne-X `Map` at
`v0.0.0-20260712112324-6989f2f174fb` has no Scrolled method, and the old wrapper
provided none. The existing zoom buttons and drag interaction remain available.

`TestThemedMapExpiredPixelsOnFailedValidation`,
`TestThemedMapCameraChangeRetiresWarmDelivery` and
`TestThemedMapCameraChangeRetiresActiveDelivery` cover failed validation/backoff,
neighbor warming, retained fresh cache and a late response after view replacement.
Both feature package race suites and focused vet pass after the fixes. GoLand
reinspected all five changed Go files with weak warnings included; only the same
six excluded EXIF fixture-duplication warnings remain. Warm demand captures its
view version on UI before launching the tracked worker; the delayed-admission
regression confirms it cannot adopt a later view. Hosted CI and CodeQL passed
for the source head; its root Qodana SARIF
has exact `3eafe3d` provenance and the same eleven assessed unused-export false
positives as `4b238da`. A fresh code/security-focused review, latest-head CI and
post-suppression Qodana assessment remain required for the follow-up commit.
The fresh code review of `c8fd4df` reports no findings:
https://github.com/frathe/picfetch/pull/75#issuecomment-5935121517.
Its root post-suppression Qodana SARIF (artifact 11175007949) has exact c8fd4df
provenance and the same eleven assessed unused-export false positives; real
production callers were rechecked. CodeQL passes with no open PR alerts. Full
hosted tests and the separate security-focused round were still finishing when
this documentation update was prepared; final latest-head review/CI evidence is
tracked on PR #75. Documentation-only updates carry the unchanged-code inspection
and focused-test evidence at c8fd4df; fresh latest-head hosted gates remain required.

Ronin supplied a FOSSA CSV export after dashboard access required onboarding.
It identifies denied issue 21375863 for the root PicFetch package at exact 3eafe3d
under Standard Bundle Distribution. The export has no paths; Ronin confirmed the
sole match is `packaging/apple-app-store/map-attribution-research-2026-10-01.md`.
This is our own research citing the map-data license, without an OSM database or
third-party code. No dependency/version or database asset was added. Ronin added
this documentation-reference false positive to FOSSA's ignore list. Record and
scope: `docs/fossa-license-ci-2026-09-26.md`. The earlier GitHub status predates
that disposition; verify a fresh scan before calling the license gate passed.
This does not waive OSM attribution, tile-service obligations or Store qualification.

## Foreground pixel delivery review follow-up

The separate security-focused review of c8fd4df reported two confirmed P2
findings (discussion_r4157592372 and discussion_r4157592390). The visible frame
pinned whole cached responses after eviction, and eviction could discard completed
foreground results before the batch repaint consumed them. A valid 256x256 PNG
padded to the existing 4 MiB body limit reproduced both: sixteen visible tiles
caused 58 downloads with only three displayed after four paints; a completed
viewport retained 71,347,776 extra heap bytes after GC.

The frame now owns only decoded pixels, copied freshness timestamps and the
no-store bit. Foreground workers deliver this same minimal value through EXIF's
existing UIQueue independently of the two evicting LRUs. UI claims prevent a
repaint from requesting a completed-but-not-yet-consumed tile again. Queued
results recheck cancellation, session, view version and the current claim before
installing pixels. Hide/renderer retirement clear claims; Close/Stop remove the
callback. Workers never wait for UI acknowledgement: existing Settle joins
tracked workers and then drains delivery, preserving shutdown and warm barriers.

The two LRUs still share the 16 MiB budget, including encoded bytes, headers and
decoded pixels. That budget excludes the displayed viewport and queued foreground
pixels: each visible/requested URL retains one normalized 256x256 NRGBA tile
(256 KiB), plus minimal metadata. This is bounded by viewport demand, independent
of encoded-body padding; no raw response bodies or header references cross the
foreground UI callback. Warm-only responses retain the existing bounded cache
policy. No new dependency, goroutine, global test seam or persistent cache.

Acceptance evidence: TestThemedMapPaddedForegroundDelivery verifies exactly one
download per visible tile for padded fresh and no-store responses and a 128-tile
viewport exceeding the work-queue capacity. TestThemedMapPaddedViewportMemory
verifies retained heap stays below a generous 40 MiB increase for sixteen padded
tiles; it failed at 68 MiB before the fix. TestThemedMapQueuedDeliveryRetiresHiddenView
checks queued delivery cannot revive a hidden frame and reopen admits fresh work.
make verify-build passes formatting/configuration/notice checks, full vet and
build. Both feature package race suites and the focused root TestLocationMap/
tile_policy_and_bounds regression pass. GoLand inspected all five changed Go
files (exifwin.go, exifwin_test.go, map.go, tiles.go, tilework.go) with weak
warnings included, and reinspected the final test edits. No actionable findings;
the same six existing fixture-duplication warnings remain covered by the exact
exifwin_test.go Qodana exclusion. Lead fixes inline; zero delegates. Latest-head
fresh code/security review, hosted CI, CodeQL, Qodana SARIF and the FOSSA ignore
rescan remain the final review-loop gates.

## No-store single-use review follow-up

The fresh c89c74d code round is clean (issuecomment-5935816898). Full CI run
36891892382 and CodeQL run 36891892683 passed; open PR CodeQL alerts are empty.
Qodana run 36891892521/artifact 11177162384 has exact c89c74d provenance and the
same eleven assessed unused-export false positives as c8fd4df, with no new
results. FOSSA's fresh c89c74d license compliance, security and dependency-quality
statuses all passed, confirming Ronin's research-reference ignore took effect.

Its separate security-focused round reported one confirmed finding,
discussion_r4157961192: direct decoded delivery left foreground no-store bodies
in the one-shot LRU after adoption. Frame paints no longer consumed that entry,
so renderer recreation could reuse an already delivered response. Direct
foreground results now bypass the one-shot LRU; pending warm/cache-only consumers
keep their existing bounded one-shot delivery. Validator cache policy is unchanged.
TestThemedMapNoStoreDirectDeliveryIsSingleUse failed before the fix on retained
no-store entries and second delivery after renderer recreation. It now verifies
four initial downloads, no stored response, stable repaint and four fresh downloads
when the renderer is recreated. Both map feature race suites and focused vet
pass. GoLand inspected the final two changed Go files (tilework.go and
exifwin_test.go), weak warnings included: no actionable findings, only the same
six existing excluded fixture duplicates. Zero delegates. Fresh code/security
rounds and all exact-head hosted gates are required again after this fix; final
review-loop evidence is tracked on PR #75.

## Renderer retirement ownership follow-up

The fresh 68d89d6 code review reported confirmed discussion_r4158045732: a
queued result from a destroyed renderer could satisfy a new boolean claim for
the same URL and discard the replacement result. TestThemedMapQueuedDeliveryRetiresRenderer
reproduces this without draining old UI callbacks first: red retired tiles win
over newer blue tiles before the fix. Renderer Destroy now advances the delivery
view version before clearing claims. Retirement checks its captured session under
the fetcher mutex, so a late Destroy from an old map cannot invalidate a replacement
session using the same fetcher. TestThemedMapRetiredRendererCannotInvalidateReplacement
failed for unconditional retirement and passes with that ownership check.

Both map feature race suites and focused vet pass after the final correction.
GoLand inspected all three changed Go files (map.go, tilework.go,
exifwin_test.go), including weak warnings: no actionable findings. Eight
fixture-duplication warnings are assessed as intentional independent test setup
(the previous six and two new setup fragments in replacement/hide tests); all
are covered by the existing exact exifwin_test.go Qodana duplication exclusion.
No production suppression or broad exclusion was introduced. Zero delegates.
Fresh latest-head code/security rounds and hosted CI/CodeQL/Qodana/FOSSA checks
remain required; final outcomes are tracked on PR #75.

## Invalid freshness metadata follow-up

The fresh 2e971e8 code round is clean (issuecomment-5936227565). Its exact-head
Qodana artifact 11179112321 has the same eleven assessed unused-export false
positives; all three FOSSA statuses and CodeQL pass. Hosted tests were still
finishing when the separate security-focused review reported confirmed
discussion_r4158179845: Expires: 0, invalid dates and empty Expires were treated
as missing and received the seven-day fallback. Presence now selects explicit
expiration; parse failure yields zero freshness, while valid max-age still takes
precedence. [RFC 9111 section 5.3](https://www.rfc-editor.org/rfc/rfc9111.html#section-5.3)
supports this distinction. Related parser cases were confirmed against
[section 4.2.1](https://www.rfc-editor.org/rfc/rfc9111.html#section-4.2.1): repeated
max-age, signed values and malformed quote pairs now require revalidation rather
than allowing a later directive or permissive parsing to extend freshness.

Seven new TestTileCacheFreshnessRules cases failed before the fix. Additional
positive cases preserve quoted valid max-age, missing-header fallback and
max-age precedence. TestTileFetcherHTTPFreshness now checks real conditional
requests and 304 delivery for Expires: 0 and its valid max-age override. Both map
feature race suites and focused vet pass. GoLand inspected both changed Go files
(tilecache.go and tiles_test.go), weak warnings included, and reinspected the
final integration test: no findings. Existing test exclusions cover the files;
no new root UI test, package, dependency or worker was added. Zero delegates.
Fresh code/security rounds and exact-head hosted gates remain required after
this fix. Final review-loop outcomes are tracked on PR #75; Store qualification
remains a separate open item.

## Cache directive grammar and Vary follow-up

The fresh 1a7941a code round reported discussion_r4158356565 (quoted commas),
discussion_r4158356579 (repeated Expires) and discussion_r4158356588 (Vary: *).
Quoted extension text could inject a fake max-age into the comma splitter, and
Vary wildcard metadata was discarded: both defects reproduced before fixing.
The repeated Expires report is rejected: RFC 9111 section 4.2.1 permits selecting
the first occurrence, which Header.Get does. The new positive regression keeps
that explicitly allowed policy; source comments record the reason.

New cachecontrol.go validates tokens, quoted strings and quoted-pairs, splitting
only outside quotes. Joined repeated Cache-Control fields preserve quoted state;
malformed grammar is non-storable. Vary metadata is retained within the existing
16 KiB header bound, and wildcard responses use non-storable single-view delivery.
HTTP request-count/validator tests cover Vary: *; freshness cases cover embedded
fake directives, escaped quotes/backslashes, a real following max-age, malformed
extensions and wildcard lists. Both feature map race suites and focused vet pass.
GoLand inspected all three changed Go files, weak warnings included: no findings.
make verify-build passes formatting/configuration/notice checks, full vet and build.
ARCHITECTURE.md now distinguishes the shared 16 MiB LRU budget from viewport
pixels and indexes the new grammar helper and captured-session renderer retirement.

No dependency/version, copied implementation, new test file, UI string or worker
was added. Existing exact tiles_test.go duplication exclusion remains applicable.
Zero delegates. Fresh code review is required after both fixes and the rejection,
then a separate security-focused round and exact-head hosted gates. Final outcomes
are tracked on PR #75. Standards evidence: RFC 9111 sections
[4.2.1](https://www.rfc-editor.org/rfc/rfc9111.html#section-4.2.1),
[4.1](https://www.rfc-editor.org/rfc/rfc9111.html#section-4.1) and
[5.2](https://www.rfc-editor.org/rfc/rfc9111.html#section-5.2).

## Equals-boundary grammar follow-up

The fresh e72ef05 code review reported confirmed discussion_r4158768254:
trimming name/argument boundaries accepted whitespace around the equals sign,
contrary to the strict cache-directive grammar policy. The parser now preserves
those boundaries for token/quoted-value validation, while retaining legal outer
list whitespace. Four malformed boundary cases failed before the fix; five
space/tab and quoted-argument cases pass afterward. Both map feature race suites
and focused vet pass. GoLand inspected the final cachecontrol.go and tiles_test.go
with weak warnings included: no findings. Zero delegates. Latest-head fresh
code/security reviews and hosted gates remain required, with final evidence on
PR #75.

## Hidden Location frame follow-up

Fresh 9ae519c review discussion_r4158984230 confirmed that hiding an ancestor
container does not call the map widget's Hide method. Both collapse and no-GPS
regressions failed with retained viewport pixels. The shared cancelTiles path now
explicitly hides the map, clearing frame/request claims and retiring queued
view delivery; regressions also verify reopening renders a fresh frame.
The EXIF race suite and focused vet pass. GoLand weak-inclusive inspections cover
uiqueue.go and exifwin_test.go: no runtime findings; eight intentional independent
fixture duplicates match the existing exact test exclusion. Zero delegates.
Latest-head hosted gates and fresh reviews remain required on PR #75.
