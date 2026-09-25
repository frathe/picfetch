# Location Map implementation

Status: implementing the approved MVP; native qualification and Ronin's verdict pending.
Route: Deep. Authority: `.scratch/location-map/spec.md`, approved tickets and
`.scratch/location-map/model-routing.md`. No commits or publication authorized.

## Frame and decisions

Deliver the loaded collection's geographic browsing workflow, preserving the
accepted duplicate, navigation, cache ownership and qualification contracts.
The parent spec's non-goals and honest limits apply unchanged. The agreed test
seams are normal viewer actions, real imaging metadata, Favorite storage, HTTP,
and public geographic input/output. No further seam approval is required.

- Root owns composition, collection occurrences, duplicate preparation and
  map/Grid/image transitions. `internal/ui/locationmap` owns the surface,
  metadata/preview work, camera and per-instance UI queue.
- Source identity uses `fileidentity.Occurrence`; source versions remain separate.
- Metadata uses `imaging.ReadMetadataURIContext` with captured
  HEIC capability. Previews use the existing canonical thumbnail pipeline/cache.
- Initial tiles are a checkerboard. Tile transport/cache follows ticket 02;
  the existing unbounded decoded EXIF widget is unsuitable for this surface.
- Keep existing dependencies. No new runtime/model/license closure is proposed.
  Recheck any dependency decision before changing it.
- Each acceptance case uses the parent's named `TestLocationMap` case and
  retains exact passing JSON events. A leaf does not complete its parent.

## Task graph and ownership

`01 -> {02,03,06}; 03 -> {04,05,07,11}; 07 -> 08;
{04,05,08} -> 09 -> 10; {02,06,10,11} -> 12 -> 13`.

All tickets are lead-owned. Each implementation slice is red, green, lead
review and evidence before its dependent work. Fixes remain with the lead.

### 01 — Real GPS browsing

Owner: T0 inline. Depends: none. Budget: 0 implementation spawns, 1 scout,
2 planned review rounds; full suite at final integration gate.
Files: new `internal/ui/locationmap/{feature,surface,work}.go`, root
`locationmap.go`/`locationmap_test.go`; composition in viewer/features/build,
menus/keys/windowmenu/browsing, navigation, startup/shutdown and test harness;
translations, manuals, architecture, Qodana exclusions and shard manifest.
Contract: `New(Host, Options) *Feature`, `Open([]Source)`, `Close`, `Stop`,
`Settle`, `SetUIQueue`, `Overlay`, `Visible`, `State`, and camera/surface input.
`Source` captures URI plus occurrence. Host opens an occurrence, leaves the map
and notifies root state changes; no appState passes into the feature.
Test/verify: parent AC01/AC06 and ticket 01's named leaves for AC07/08/09/17/18,
using normal menu/key actions, real JPEG GPS and mounted surface assertions.

### Remaining slices

Ticket 03 concrete contract: root `locationInput` owns duplicate preparation
with a request token; `syncDuplicateState` admits scanning only after the
existing Grid barrier. Feature `Preparing` displays cancellable preparation;
`Open` receives representative Sources after grouping. Scanning uses one
coalesced pending UI publication with incremental points and cumulative Counts,
so slow UI cannot accumulate one callback/snapshot per image. Manual camera
input disables background fit. Tests use held reads and real queued delivery.
Files: root locationmap/actionmenu/harness plus feature/work and existing UI
acceptance file. Owner T0, 0 spawns, 2 planned reviews, no full suite.
Verify: parent preparation_and_counts, camera_retention/progressive_discoveries,
lifecycle/progressive_direct_visit and lifecycle/preparation_and_scan_retirement.

| Ticket | Files/package responsibility | Contract and proof | Owner/budget |
| --- | --- | --- | --- |
| 02 | locationmap tiles/HTTP/cache, surface delivery | Identified viewport HTTP, freshness/revalidation, bounded bytes, retry and hide; AC15/16 | T0; up to 1 Sol/high slice |
| 03 | root duplicate preparation, locationmap scan work | Completed groups, progressive delivery, cancellation; AC03/08/17 | T0; 0 |
| 04 | locationmap geographic donor resolution | Representative priority, all-pairs <=100 m, provenance; AC02/03/07/18 | T0; up to 1 Sol/high slice |
| 05 | locationmap geography/surface, root/Grid visits | Display-space clusters, exact frozen occurrences, camera; AC04/05/07/08/15/17 | T0; up to 1 Sol/high slice |
| 06 | imaging shared EXIF, fixtures, root acceptance | PNG/WebP plus established formats/failures; AC14/18 | T0; up to 1 Sol/high slice |
| 07 | locationmap facts, source versions/previews | Live-only memory, stale producer refusal; AC09/10/17 | T0; up to 1 Sol/high slice |
| 08 | Favorite GPS storage and root save/remove hooks | Owner-bound atomic publication, corruption/recovery; AC10/11/18 | T0; 0 |
| 09 | sourcechange/filework/sort, feature rebuild | Aliases/donors, committed effects, frozen survivors; AC08/09/12/17/18 | T0; 0 |
| 10 | root entry/return and feature version validation | External refresh cannot supersede newer sources; AC13/17 | T0; 0 |
| 11 | native qualification runner/checker, launch/Make | Real input-to-visible measurements and strict evidence; tooling tests | T0; up to 1 Sol/high slice |
| 12 | acceptance/evidence, inspections, repository gates | Complete AC01-19/21, native 10k evidence | T0; 0; full suite once |
| 13 | local qualification evidence | Ronin's 30k run/verdict; AC20 | Ronin; 0 |

Before each candidate delegation, fix exact signatures, <=3 files, tests and
one verification command here. No candidate is admitted merely by this table.

## Delegation admission

Recon scout: locate existing source-versioned preview/metadata fixtures and
public reuse boundaries across imaging/favthumbs/Grid/uitest. Shell found the
owning packages but not the cross-package contracts. G1: bounded <=25-line
read-only prompt; G2: verify returned locators with `rg -n`/targeted reads;
G3: zero writes; G4: returns contracts/locators rather than broad source dumps;
G5: lead has not read those implementations. S: tracing reuse and fixture
semantics requires reading, not a mechanical transform. W: no code supplied.
Model: gpt-6-luna, medium, fresh context. Lead implements integration meanwhile.

## Evidence and limits

- Initial worktree clean on `feature/image-map`, HEAD `94c685f`.
- Approved seams and ticket dependencies read; implementation not yet verified.
- Partial implementation verification is recorded below; no complete-MVP or
  native-performance acceptance is claimed from those focused checks.
- Qualification requires caller-chosen datasets; no private collection inferred.
- Native Linux/amd64 full verification, GoLand inspections, 10k measurements
  and Ronin's 30k verdict remain required, not inferred from headless tests.

## Ledger

| Task | Spawns budget/actual | Reviews | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| Recon | 2/2 | lead | no | source/version/cache and native capture API scouts complete |
| 01 | 0/0 | 2 | no | seven acceptance cases, race regression, native render and inspections |
| 02-11 | 6/6 total | lead review in progress | no | all six bounded Sol/high slices delivered; root integration and review remain lead-owned |
| 12 | 0/0 | lead, native acceptance pending | preflight blocked | all named acceptance tests pass; full gate needs native amd64 Docker |

### Ticket 01 evidence

- Named acceptance cases pass through `TestLocationMap`; exact JSON evidence
  is retained under `.scratch/location-map/evidence/01-tests.json`.
- Observed red: missing menu, missing mounted GPS thumbnails, lost Grid
  selection, unavailable Shift+L, absent preview, missing pan input,
  absent displayed-image highlight, and retired visit intercepting Escape.
- Negative stale-delivery check: disabled the scan callback's cancellation /
  generation guard; `lifecycle/base_round_trip` failed with `queued retired
  scan replaced current points`; restored the guard before further work.
- Focused native race tests (`TestLocationMap`, Window/Actions menus and
  Close Files) passed. Imaging package and locale parity passed. Menu
  composition/pair inventory was updated for the new Window item and passed.
- Inspected native-host test render `evidence/01-map.png`; added an opaque
  full-surface backdrop after detecting the underlying viewer in toolbar gaps.
- GoLand inspected all 20 initially changed code files including weak warnings.
  Formatting findings fixed with `make fmt`. The platform-selected Cmd/Control
  constant warning in `shortcuts.go` is a false positive on macOS; scoped
  `GoBoolExpressions` suppression documents the other-platform branch.
- Final suite preflight: `make check-test-platform` refuses the available
  `linux/aarch64` Docker daemon. Native Linux/amd64 full gate remains pending;
  worker isolation has not been disabled. Native render is not 10k qualification.

### Ticket 06 bounded metadata task admission

Owner: T1 `gpt-6-sol`/high, lead retains viewer integration and all review.
Files: `internal/imaging/exif.go`, `internal/imaging/loader.go`, new
`internal/imaging/exif_containers_test.go` only. Depends: ticket 01 base path.
Contract: existing `ReadMetadata`/`ReadMetadataContext` read embedded PNG eXIf
and WebP EXIF using existing TIFF interpretation. Add
`ReadMetadataURIContext(context.Context, fyne.URI) (Metadata, error)` to share
bounded raw reads without pixel/config decoding, preserving operational errors.
Test: authored real PNG/WebP containers with GPS/date, explicit zero, missing,
malformed/truncated chunks; existing JPEG/TIFF behavior; URI failure/input bound.
Verify: `go test -tags no_emoji,nodynamic -count=1 ./internal/imaging`.
Budget: one spawn, two review rounds, no full suite.
G1: standalone <=25-line prompt. G2: focused package command. G3: three files
in one package, no concurrent edits. G4: local container parsing context is
smaller than feature integration. G5: lead has not read the extraction helpers.
S: parsing/fixture comprehension is not a text transform. W: contracts only.

### Ticket 02 bounded tile-store task admission

Owner: T1 `gpt-6-sol`/high; lead owns rendering, timers, notifications and tests
through the viewer. Files: new `internal/ui/locationmap/tiles.go`,
`tiles_test.go`, optionally `tilecache.go` only; no shared feature/surface edits.
Contract: `TileKey{Z,X,Y int}`; `TileOptions{Client *http.Client, Now func()
time.Time, URL string, EncodedBytes, DecodedBytes int64}`;
`NewTileStore(TileOptions) *TileStore`;
`Fetch(context.Context, TileKey) (image.Image, time.Time, error)` where time is
the earliest retry after failure, zero after success;
`Usage() (encodedBytes, decodedBytes int64)`.
No owned goroutines: feature runs viewport requests with its own lifecycle.
Defaults: OSM HTTPS URL template, app-identifying User-Agent with project URL,
32 MiB encoded and 64 MiB decoded LRU. HTTP freshness includes max-age/Expires,
Age, ETag/Last-Modified revalidation, no-store/no-cache; fallback TTL seven days.
Responses limited to 1 MiB and PNG dimensions 256x256 before decoding.
Failure backoff starts at 1 second, doubles to one minute; respect Retry-After
seconds/date (later server deadlines win). Bound failure bookkeeping to 1024
recent keys; canceled requests must not install cache/failure entries.
Test: fake HTTP/time, real tiny authored PNG fixture expanded to tile size,
fresh reuse, stale 304, server retry guidance, cancellation and cache eviction.
Verify: `go test -tags no_emoji,nodynamic -count=1 ./internal/ui/locationmap`.
Budget: one spawn, two review rounds, no full suite.
G1: <=25-line standalone prompt. G2: focused public-store tests. G3: <=3
explicit files, no overlap with lead or metadata implementer. G4: isolated HTTP
semantics need less context than viewer state. G5: lead has not inspected tile
internals. S: expiry/error state requires comprehension; W: contracts only.
Current policy checked 2026-09-25 at
https://operations.osmfoundation.org/policies/tiles/; attribution remains on
the mounted surface, and only human-visible viewport tiles may be requested.

Completed delegates: `location_tiles` (02), `location_resolution` (04),
`location_geography` (05), `location_metadata` (06), `location_facts` (07), and
`location_evidence` (11), all T1 Sol/high with fresh bounded contexts. The
separate `location_boundaries` read-only scout used Luna/medium. Tickets 08-10,
native observations, cross-feature integration and all review/fixes remain
inline with the lead, as the routing plan requires. Delivery is not acceptance.

### Ticket 03 current evidence

Duplicate preparation now precedes plotting; a held hash proves no provisional
point is published. A patterned same-shot fixture verifies the highest native
resolution representative. A held metadata read exposes honest partial counts
and usable thumbnails. Direct visits keep scanning; manual camera survives new
discoveries. Focused `TestLocationMap` passed. Root preparation invalidates on
exit/shutdown; queued metadata deliveries retain captured generation/queue.
Switched scans to the shared metadata-only URI reader after its package tests
passed. Remaining cancellation variants and negative checks stay lead-owned.

### Tickets 04 and 05 bounded pure task admission

04 owner T1 Sol/high, files only new `internal/ui/locationmap/resolution.go`
and `resolution_test.go`. Contract: `LocationCandidate{Index int, PixelCount
int64, Metadata imaging.Metadata, ReadError bool}`, `LocationResolution{Metadata
imaging.Metadata, DonorIndex int, Outcome LocationOutcome}`; constants
`LocationLocated`, `LocationUnlocated`, `LocationUnreadable`, `LocationConflict`;
`ResolveLocation(representative LocationCandidate, others []LocationCandidate)`.
Direct valid GPS wins. Otherwise all required reads must succeed (unknown
candidate locations cannot prove agreement). All valid donors must agree
pairwise within 100 m using mean Earth radius 6371008.8 m, with <=0.000001 m
roundoff tolerance. Choose highest PixelCount then lowest Index. Preserve all
representative metadata except borrowed latitude/longitude/HasGPS. Never mutate
inputs. Absent donor uses -1. Validate finite lat [-90,90], lon [-180,180].
Tests: direct priority, no/single/multiple donors, read failure, all-pairs chain,
exact/over boundary, world wrap, invalid coordinates, deterministic ties.

05 owner T1 Sol/high, files only new `internal/ui/locationmap/geography.go`
and `geography_test.go`. Contract: `WorldPoint{X,Y float64}`;
`Project(lat,lon float64)(WorldPoint,bool)`; `Camera{X,Y,Scale float64}`;
`FitCamera(points []WorldPoint,width,height,padding float64) Camera` uses the
smallest circular longitude arc, Mercator latitude clamped after validation,
scale 256 through 256*2^14. `Cluster{Members []int,X,Y float64}` with screen
coordinates; `Clusters(points []WorldPoint,camera Camera,width,height,diameter
float64) []Cluster`. Each valid visible point belongs once, including margin
diameter around viewport, using nearest wrapped world copy. Stable input-order
greedy leaders: assign within Chebyshev distance <diameter to the earliest
existing leader; otherwise new leader. Leaders retain their point position.
Spatial hash of leaders bounds work even for coincident 30k inputs. Members
are exact original input indexes; invalid inputs are excluded. Tests cover
dateline fit/wrap, poles/boundaries/invalids, coincident 30k, splitting by zoom,
exact membership and deterministic order. No surface/root/cache edits.

Both tasks: fresh <=25-line prompts, <=2 files in one package, disjoint owners;
verify `go test -tags no_emoji,nodynamic -count=1 ./internal/ui/locationmap`.
One spawn/ticket, two planned lead reviews, no full suite. G4/G5: pure geometry
and donor math do not require the cross-feature state the lead is integrating;
lead has not implemented these helpers. S: spatial/distance rules require
reasoning rather than a mechanical script. W: contracts/tests, no code solution.

### Native measurement API scout admission

T3 Luna/medium, fresh bounded context, second read-only scout. Native capture
availability, pixel-buffer timestamp semantics and permission APIs require
official API/header reading beyond shell locators. No architecture, code,
review, UI actions or file writes delegated. Research skill provides primary-
source discipline; the user-requested routing's read-only restriction takes
precedence, so the lead records returned facts in the implementation evidence.
G1 <=25-line factual prompt; G2 check official Apple links against installed SDK
declarations; G3 zero writes; G4/G5 native API details are not yet held by the
lead and independent acceptance work continues. S: semantic timing/permission
contracts, not grep-only enumeration. W: no implementation supplied.

### Ticket 07 bounded raw memory-fact admission

Owner T1 Sol/high, only new `internal/ui/locationmap/facts.go` and
`facts_test.go`. One spawn, two lead reviews, no full suite. Contract:
`Fact{Version string, Metadata imaging.Metadata}`, `NewFactCache() *FactCache`,
`Keep(keys []string)` replaces allowed live source membership, retaining matching
facts; `Get(key,version string)(imaging.Metadata,bool)`; `Capture(key,version
string) FactWriter`; `FactWriter.Store(context.Context,imaging.Metadata) bool`;
`Invalidate(keys []string)` retires facts and outstanding writers for those keys;
`Snapshot() map[string]Fact` returns independent completed raw records.
Thread-safe, memory-only, no I/O/goroutines. Only current Keep members admitted;
empty versions never cached. Changed-version capture supersedes previous writers
for the same source. Removing/readding membership cannot revive an old writer.
Invalidation preserves allowed membership but retires publication authority.
Successful absent GPS is a fact; operational failures are not Store calls.
Tests: reuse/absence, duplicate membership, version reversal, invalidation,
cancelled Store, remove/readd, snapshot mutation, concurrent operations/race.
Verify `go test -race -tags no_emoji,nodynamic -count=1 ./internal/ui/locationmap`.
G1 bounded <=25-line prompt; G2 exact command; G3 two new files; G4/G5 isolated
admission locks need no viewer state, implementation not held by lead. S/W:
ownership semantics require comprehension; no solution code is supplied.

### Current integration and review evidence

- Tiles render under local results, retry without input, throttle notifications
  across reopen, and cancel through the normal direct-image route. Controlled
  transport/time only. Lead red regressions fixed 304 losing no-cache and
  undercounted 16-bit PNG residency; decoded pixels normalize to four bytes.
- Fyne 2.8 Clip needed BaseWidget initialization before Border's first Resize;
  a zero-sized mounted surface exposed this, then tile scenarios passed.
- Duplicate fallback now reads real donor metadata, keeps the representative
  target/date and exposes provenance. Core borrowed/direct cases pass.
- Exact occurrence subsets extend Grid without changing path-only Explorer
  subsets. Cluster -> Grid -> image -> Grid -> map acceptance passes.
- Raw memory facts are source-version checked before and after metadata reads;
  an unchanged no-GPS source was observed read twice (red), then once (green).
- Focused integrated `TestLocationMap` race run passed (11.441s).
- GoLand: 14 additional files inspected. Three tile LRU nilness warnings were
  mitigated with explicit non-empty loop conditions; retry-result warnings are
  narrow false positives because deadlines accompany errors by API contract.
  Existing LoadedImage padding warning fixed by field grouping. Imaging build
  guard diagnostic reflects missing IDE nodynamic tags, not a failing tagged
  build; IDE configuration remains unverified. Reinspection still due.

### Ticket 08 storage design

Lead only. Raw Favorite records use versioned source keys inside a directory
bound to the captured saved membership. Open directory handles prevent a moved
Favorite from being recreated under its old name; membership-specific namespaces
prevent retired producers overwriting replacement membership. Validate directory,
file-list identity and source version around admission/publication. Cache records
are disposable, schema-versioned, bounded JSON; no derived donor locations.
Saving promotes the current memory snapshot, and scan completion promotes facts
completed after that save. Unsaved members are excluded by actual disk membership.
Removal leaves memory useful. Cancellation/retirement suppresses stale warnings;
other cache failures warn once and do not fail geographic browsing.

### Ticket 11 checker admission

Owner T1 Sol/high. Allowed only new `scripts/locationmapqualify/evidence.go`
and `evidence_test.go`, package main. Lead owns runner, native observations and
main/Make integration. Contract fixed below; one spawn (6th), two lead reviews.
`Report` JSON fields: `Schema int`, `BuildID string` (64 lowercase hex),
`Native bool`, `Observation string` (must be `macos-screen-capture`),
`Images int`, `Formats map[string]int`, `Hardware,Storage string`,
`Complete bool`, `Stages []Stage`, `Gestures []Gesture`,
`Cancellations []Cancellation`, `Memory []MemorySample`, `OpenCloseCycles int`,
`VerdictBy,Verdict string`.
Stage: `Kind string` (cold/warm), `StartNS,EndNS,PreparationNS,ScanNS int64`,
`Complete bool`. Gesture: `Kind string` (pan/zoom), `InputNS,VisibleNS int64`,
`Before,After string` (artifact relative paths), `Skipped bool`.
Cancellation: `InputNS,VisibleNS int64`, `Before,After string`, `Complete bool`.
MemorySample: `AtNS,RSSBytes int64`.
JSON names use snake_case. `CheckReport(report Report,expectedImages int,
expectedBuild string) error`; `CheckEvidence(dir string,expectedImages int,
expectedBuild string)(Report,error)` loads report.json and validates artifact
files. Exact count/build, native complete report, both complete cold/warm stages,
nonnegative stage durations bounded by interval, sum format counts equals count,
hardware/storage required. >=40 pan+zoom samples with both kinds represented,
all positive ordered timestamps and no skipped samples; >=1 valid cancellation;
>=2 chronological positive memory samples. Relative artifact paths must stay
inside evidence dir, exist, contain valid PNGs, and before/after differ in pixels.
Exactly 10k requires >=95% gestures <=100ms and all cancel feedback <=250ms;
report all slow samples, do not filter. 30k requires >=3 open/close cycles,
memory span >=60s and VerdictBy=Ronin, Verdict=pass, without 10k latency gates.
Other explicit smoke counts do not qualify 10k/30k. Checker validates recorded
evidence, not cryptographic proof of screen capture; native runner remains lead.
Tests author synthetic fixtures as checker tests only: absent/partial/wrong size,
wrong build, missing native/PNG artifacts, skipped/interrupted/invalid stages,
threshold boundary and human verdict cases. Verify
`go test -count=1 ./scripts/locationmapqualify`. G1 <=25-line prompt, G2 command,
G3 two files, G4/G5 checker schema is isolated from native app integration. S/W:
validation needs semantic edge coverage; no implementation code supplied.

### Native runner implementation boundary (lead only)

Add an isolated `--location-map-trial DIR` launch mode with a separate Fyne
application identity and Favorite root. A small `internal/locationtrial`
recorder receives root-owned admission/stage snapshots and writes them on one
tracked worker; it is not a paint-latency source. Root starts/stops/joins it.
The native runner builds its report from actual admitted count/format mix,
completed cold/warm stage observations, process RSS samples and captured frames.
No private dataset is inferred. Native screen/input helper uses ScreenCaptureKit
complete frames and their WindowServer display timestamps, with input-side
Mach timestamps in the same timebase. Before/after captures must show changed
map pixels. Keep failed/interrupted observations and fail the checker, rather
than substituting worker timings. A dedicated helper's UI interaction requires
appropriate user authorization and existing OS capture/post-event permissions.
The normal Computer Use capture delay cannot measure the 100ms requirement.

### Latest focused evidence

- Held Favorite removal/replacement, corruption fallback, once-only write-failure
  warning, fresh-instance reuse, saving promotion and unsaved merge ownership pass.
- Deletion, sorting, committed hidden-donor writes through aliases, exact repeated-
  occurrence survivor selection, empty frozen visits, external return and stale
  queued validation pass. Deletion/sort/external-return were observed red first.
- Conflict-only groups remain unlocated and leave source bytes unchanged. Raw
  representative facts never acquire borrowed GPS. Real PNG/WebP/JPEG/RAW and
  captured HEIC metadata reach the map; controlled HEIC failure stays unreadable.
- Viewport movement respects configured encoded/decoded budgets; identified OSM
  endpoint requests and mounted attribution are tested. Native measurements pending.
- Integrated Location Map/source-mutation race run passed (24.319s); pure map/menu
  race packages and focused imaging/Grid race regressions passed. `make vet build`,
  shard manifest and exact Qodana exclusions passed. Later edits require reruns.
- GoLand inspected the additional feature/root/cache/Grid/checker files; confirmed
  padding/import issues fixed, partial-inventory warning narrowly documented.
  Final all-file reinspection remains pending, as does native Linux/amd64 full CI.
- `make location-map-qualification-test` passes. Checker CLI hashes the actual
  current binary; synthetic checker fixtures are not native qualification evidence.

### Current handoff evidence (2026-09-25)

- All six planned T1 Sol/high slices delivered; two T3 Luna/medium scouts
  completed. The routing file's stale pre-dispatch status is corrected. No
  additional implementation delegates or peer reviewers were spawned.
- `go test -race -tags no_emoji,nodynamic -count=1 -json -run
  '^TestLocationMap$' ./internal/ui` passed, package duration 30.695s. Retained
  all 292 JSON events in `.scratch/location-map/evidence/12-acceptance-tests.json`,
  including passes for all 18 named parent acceptance cases. This is host-native
  headless test evidence, not native screen/performance qualification.
- Extra coverage verifies frozen cluster membership during continuing discovery,
  staged selection/search Escape, malformed/incomplete GPS, unavailable HEIC
  admission, AVIF no-location behavior and standalone TIFF metadata (including
  explicit zero). No source images were changed by map browsing.
- Lead review found that inactive committed writes left Favorite disk records.
  The regression failed with `inactive committed write retained persisted GPS
  fact`, then passed. Each member now owns one version-validated raw record;
  tracked cleanup removes affected disposable records even after map close,
  serializing against publication that rechecks live raw authority. Ordinary
  source-write regressions and Favorite ownership scenarios pass under race.
- Focused race packages passed: locationmap, menus, launch, locationtrial and
  locationmapqualify. Metadata/source-write regressions, locale parity and the
  manual Unicode-arrow guard passed. `make fmt vet build`, `git diff --check`,
  shard assignment (705 runnables, three shards) and Qodana exclusions passed.
- GoLand inspected all 58 changed Go files with warnings included. Final results
  and dispositions are in `.scratch/location-map/evidence/12-goland.json`.
  All returned findings are fixed/mitigated except the existing required-tag
  configuration diagnostic on imaging's avifpolicy import. Its tagged build is
  green; a clean IDE inspection for that file remains unverified. GoLand's build
  endpoint reports success but limited build diagnostics, not a substitute gate.
- The macOS screen/input helper compiles with Swift warnings as errors. Runner
  contract tests retain failed gestures and observed counts, isolate trial
  storage, enforce finite deadlines and reject substituting slow startup for
  sustained browsing. Run/check instructions and measurement limitations live
  in `scripts/locationmapqualify/README.md`. No native capture has been executed.
- `make verify` was attempted and stopped at `check-test-platform`: selected
  daemon is `linux/aarch64`, not required native `linux/amd64`. No worker
  isolation test was skipped and no seccomp policy was relaxed.

### Remaining acceptance / required input

2026-09-25 follow-up: Ronin has now reported an informal 30k run that "went
relatively well", with rendering/interaction findings. See the
[polish plan](2026-09-25-location-map-polish.md) and ticket 13. Its checklist audit
also found proposed child scenarios absent from the retained test inventory:
all 18 parent names passing is not complete acceptance of those composite
contracts. Ticket 12 remains open; 37/64 ticket acceptance points are verified.

- Native smoke and 10k collection runs, artifact review and any resulting fixes
  remain outstanding. The Computer Use skill requires explicit authorization
  for running the dedicated ScreenCaptureKit/CGEvent helper. Requested that
  authorization and an explicitly chosen image directory; neither is inferred.
  Existing OS permissions may also need a human to grant them. Native evidence
  may contain photo thumbnails/filenames/locations and stays local.
- Run the full repository gate on native Linux/amd64 (or authorized CI), resolve
  the IDE build-tag inspection limitation, then reassess complete acceptance.
- Ronin's 30k run/verdict remains human-owned; do not fill in a passing verdict.
  No MVP-complete/release-ready assertion, commit, push, merge or release made.
