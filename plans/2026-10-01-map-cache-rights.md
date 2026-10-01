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
Ronin supplied a FOSSA CSV export after dashboard access required onboarding.
It identifies active denied issue 21375863, ODbL-1.0, for the root PicFetch package
at exact 3eafe3d under Standard Bundle Distribution; matching paths are absent.
No dependency/version or database asset was added. The new license-reference
documentation is the likely cause, pending matched-file confirmation. Do not
count the license gate as passed or silently broaden its policy.
