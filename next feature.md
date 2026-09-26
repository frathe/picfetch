# Feature Design: Location Map & Future Location Suggestions

## Goal

Add a map-based view to PicFetch that allows users to explore the geographic
distribution of images in the current collection.

PicFetch already has OpenStreetMap integration and reads geolocation metadata.
Reuse suitable parts of that infrastructure. Collection scanning, interactive
image previews and geographic clustering need additional work; the existing
map widget's marker interaction only toggles a title.

## Status

Design decisions agreed through Q1-Q25 of the interview with Ronin on
2026-09-25. This document defines the MVP behavior; implementation and
qualification have not started. Location Suggestions remains a separate future
idea, not part of the accepted MVP. Domain terms are in [CONTEXT.md](CONTEXT.md).

The published [Location Map specification](.scratch/location-map/spec.md) is the
implementation source of truth. This document retains the design interview and
the separate future-feature ideas.

---

# Phase 1: Location Map MVP

## Map View

Add a new view displaying images from the current collection on an
OpenStreetMap map.

An image needs valid GPS coordinates to appear. With duplicate hiding active,
the group's highest-resolution representative can instead be located using
recorded coordinates from another member of that group.

### Features

- Add a main-window **Location Map** for the complete loaded collection,
  independent of Grid search and Grid selection.
- Open through Window -> Location Map or Shift+L, following existing view-entry
  and focused-text-input conventions.
- Reuse the existing OpenStreetMap integration.
- Display individual mapped images as location-anchored thumbnails, reusing the
  existing thumbnail cache and loading visible thumbnails on demand.
- Honor the active duplicate filter, using GPS from another group member when
  the highest-resolution representative lacks it.
- Omit unlocated images and duplicate groups with no usable GPS coordinates.
- Count images after duplicate hiding. For 10,000 loaded files reduced to 7,000
  representatives, of which 3,000 have usable direct or fallback GPS, show
  `3,000 of 7,000 images located`. Cluster counts use the same representatives
  and match the membership their Grid View opens.

## GPS Scanning

- Start scanning automatically when Location Map opens.
- If duplicate hiding is active and grouping is unfinished, complete duplicate
  preparation first with cancellable progress. Reuse current grouping results
  when available, then begin progressive GPS discovery.
- Show discoveries progressively and keep map interaction available during the
  scan.
- Continue scanning while visiting a cluster Grid or a photo from the map.
  Exiting Location Map cancels unfinished scanning. Completed results remain
  reusable under the GPS cache's Favorite-disk/live-memory policy.
- Report checked/total progress separately from located images. With duplicate
  hiding active, progress counts completed representatives/groups rather than
  individual metadata reads needed to resolve each group.
- When duplicate hiding is active, first use the highest-resolution
  representative's GPS. If it has none, inspect all other group members for
  recorded coordinates. Use the highest-resolution geotagged member's recorded
  coordinates only when all valid fallback positions are within 100 metres of
  each other. A single geotagged member is sufficient. If fallback positions
  conflict, leave the group unmapped and report the conflict.
- Keep the highest-resolution representative as the displayed/opened image.
  When its location comes from another member, name the location source in the
  preview. This is especially relevant because duplicate groups are based on
  perceptual similarity, not guaranteed identical file content.
- Map fallback does not write GPS metadata to any image.

## Metadata Coverage and Reporting

- Reuse the existing JPEG, TIFF/RAW, AVIF and system-supported HEIC metadata
  paths. Extend the shared metadata reader to extract embedded EXIF GPS from
  PNG and WebP, which currently expose EXIF orientation only.
- XMP-only locations and sidecar files are outside the MVP.
- Keep separate summary counts for no usable GPS, source-read failures, and
  conflicting duplicate locations. Do not raise a toast for every failed file.
- No usable GPS covers absent, incomplete, malformed and unsupported metadata;
  it does not assert that the source file contains no location information.
- Existing format limitations must remain explicit: RAW uses supported TIFF
  tags or embedded JPEG metadata, and HEIC depends on the captured system
  capability. The current AVIF adapter cannot distinguish absent GPS from an
  explicit `(0, 0)` position; qualification must document that limitation unless
  presence-aware extraction resolves it. Explicit zero coordinates in formats
  whose tags establish their presence remain valid.

## GPS Cache

- Retain scanned GPS results separately from the existing thumbnail cache.
- Each Favorite owns filesystem GPS records for its saved members. Saving a
  live collection also persists its already-known GPS results for those members.
- Newly merged images that have not been saved in the Favorite remain
  memory-only until saved.
- Removing a Favorite removes its disk GPS cache along with that Favorite,
  while keeping currently loaded images and their in-memory facts usable.
- For live collections, retain GPS results in memory only while the collection
  remains loaded. No general on-disk GPS cache is introduced.
- Reuse valid results when reopening or rebuilding the map and invalidate
  affected results after source changes.
- Reread the source when a GPS cache entry is corrupt. If a Favorite's GPS cache
  is unwritable, warn once and continue with in-memory results. Operational read
  failures must not become persistent claims that a source has no GPS.

## Source Changes and Automatic Rebuilding

- Source deletion or a committed source write invalidates the affected GPS
  facts and automatically rebuilds the map without a confirmation or manual
  rebuild action. This includes changes to a hidden member that supplied GPS.
- Changes made inside PicFetch trigger rebuilding immediately. Check for
  external changes on map entry or return and rebuild automatically as needed;
  continuous external-file monitoring is outside this design.
- Preserve surviving images in an already-open cluster Grid. Rebuilding does
  not add new members to that frozen visit; deleted members are removed.
- Sorting alone preserves the map.

## Navigation and Camera

- From a cluster, navigate Map -> Grid -> image. Image arrow keys browse the
  cluster's captured membership in collection order.
- An individual map thumbnail opens the preview; its Open Image action goes
  directly to the image. Image arrow keys then browse mapped images in
  collection order.
- Escape retraces the entry route and retains map position and zoom. Grid keeps
  its existing marquee/selection/search-clearing Escape stages before leaving
  the cluster.
- Back to Viewer exits Location Map.
- Fit discovered locations automatically until the user first pans or zooms.
  Subsequent discoveries and automatic rebuilds preserve the user's camera.
- Fit All explicitly frames the complete currently discovered map result.

## Marker Clustering

Images close to each other should be grouped depending on the current zoom
level.

Example:

```text
        ┌──────┐
        │  37  │
        └──────┘
```

- Cluster nearby images.
- Recalculate clusters when zooming.
- Show a counted pin such as `34 Images` when images cannot be displayed
  separately at the current zoom.
- Clicking a cluster opens Grid View for exactly its counted members, not all
  images inside a surrounding bounding rectangle. Membership stays fixed for
  that visit as scanning discovers more images. Users can separately zoom the
  map further to reveal individual images where there is enough space.
- Returning from Grid View or an image visit restores map position and zoom.
- Identical coordinates can remain clustered at every zoom; Grid View provides
  access to the members.

## Map Tiles and Loading Failures

- Replace failed tiles with a checkerboard background.
- Raise a map-loading error toast at most once per minute, rather than once per
  failed tile. Preserve this cooldown across map close/reopen. Wording:
  `Could not load map tiles. An internet connection is required.`
- Automatically retry failed tiles needed by the visible map with bounded
  backoff and respect for server retry instructions. Stop tile requests while
  the map is hidden; resume needed requests on return. Replace checkerboards
  as tiles arrive. Local GPS scanning can continue during image/Grid visits.
- Follow the [OpenStreetMap tile usage policy](https://operations.osmfoundation.org/policies/tiles/):
  visible attribution, an application-identifying User-Agent, viewport-driven
  requests, and caching that honors HTTP freshness headers (or at least seven
  days where those headers cannot be read). Offline-area downloads remain out
  of scope. The tile cache is separate from the Favorite/live GPS-cache policy.
- Bound encoded and decoded tile residency for sustained map browsing. The
  existing widget's process-global decoded cache has no eviction; the older
  EXIF-only acceptance of that behavior does not qualify a collection map.

## Image Preview

Clicking an individual map thumbnail opens a small preview.

The preview should contain:

- Thumbnail
- Filename
- Capture date/time, if available
- Location source filename when coordinates come from another duplicate member
- Action to navigate to/open the image in PicFetch

Map thumbnails and preview thumbnails reuse the existing thumbnail cache.

Example:

```text
┌─────────────────────────┐
│      [ Thumbnail ]      │
│                         │
│ IMG_3821.JPG            │
│ 2026-06-18 14:32        │
│                         │
│       Open Image        │
└─────────────────────────┘
```

## Selection Feedback

- Highlight the displayed image's map representative or containing cluster when
  it is represented on the map.
- Clicking a map thumbnail opens its preview. Only Open Image requests a change
  to the displayed image.
- Preserve existing Grid selection when entering or interacting with the map.

## Empty States

Handle at least:

- Empty collection
- Collection without geotagged images
- Only one geotagged image
- Invalid or incomplete GPS metadata
- Conflicting duplicate locations
- Source-read failures and unavailable system HEIC support
- Missing map tiles and failed Favorite-cache persistence

## Performance Qualification

These are accepted targets, not measured results:

- Qualify with 10,000 images. At least 95% of pan/zoom responses should be
  visible within 100 ms of input; cancellation feedback should appear within
  250 ms. Feedback does not imply an already-blocked filesystem read has exited.
- Measure cold scanning and warm reopening separately, recording hardware,
  storage, format mix, duplicate-filter state and peak memory. Include duplicate
  preparation in the observed cold-entry experience and report it separately.
- Report full-scan duration without imposing a storage-independent deadline.
- Ronin performs the 30,000-image stress test, checking responsiveness, repeated
  opening/closing and memory growth during sustained browsing.
- Qualification must distinguish actual native UI responsiveness from headless
  tests; performance and the human stress test remain unverified until run.

## Implementation and Verification Handoff

The implementation plan should bind this behavior to concrete tests and commands
under the repository's SDD/TDD agreement. Cover the complete map/Grid/image
return paths, duplicate fallback and conflicts, cache ownership/version changes,
automatic rebuilding, cancellation/stale completions, and offline tile recovery.
Include coincident points, world wrapping, invalid coordinates and empty states
in deterministic geographic tests. Apply the ordinary repository verification
and GoLand inspection requirements before feature completion.

This design does not promise instantaneous cold scanning, internet availability,
or complete metadata coverage for every container. Coordinate fallback is a
display decision based on a perceptual duplicate group, with its source exposed;
it does not prove that the representative was taken at that location.

---

# Suggested Implementation Tickets

These are work areas for the implementation plan, not completed tickets.

## MAP-1 — Location Map Shell

Create the main-window Location Map and its menu/keyboard entry. Reuse suitable
OpenStreetMap infrastructure while meeting tile policy and bounded-memory needs.

## MAP-2 — Geo Image Provider

Provide the map with the minimum required image information:

- Image ID
- Latitude
- Longitude
- Thumbnail
- Filename
- Capture timestamp

Reuse existing thumbnail cache entries and load missing thumbnails on demand.
Avoid image decoding for the GPS scan where only metadata is required. The
existing EXIF path reads/probes whole sources; a collection scan must account
for that cost rather than assuming a reusable GPS index already exists.
Include duplicate GPS fallback/conflict handling, progressive group-based
accounting, PNG/WebP EXIF GPS extraction, and Favorite-disk/live-memory GPS
caching with ownership/version invalidation and memory fallback.

## MAP-3 — Marker Clustering

Implement zoom-dependent geographic marker clustering.

## MAP-4 — Image Preview Popup

Add the marker preview UI and navigation back to the corresponding image.

## MAP-5 — Navigation and Selection Feedback

Integrate frozen cluster Grid visits, mapped-image browsing, Escape/Back routes,
displayed-image highlighting, camera retention, and existing Grid selection.

## MAP-6 — Empty States & Edge Cases

Handle collections with missing, invalid, conflicting or sparse GPS data, tile
loading failures, cache errors, and automatic source-change rebuilding. Collect
the agreed 10k performance evidence and support Ronin's 30k stress test.

---

# Explicitly Out of Scope for MVP

Do **not** include these in the initial Location Map implementation:

- GPS metadata editing
- Reverse geocoding
- Route generation
- Travel animations
- Heatmaps
- Offline map downloads
- Location inference beyond the recorded-coordinate duplicate fallback above
- XMP-only and sidecar location extraction
- Continuous monitoring for external file changes

These can be implemented independently after the basic Location Map has proven
useful.

---

# Future Feature: Location Suggestions

PicFetch could later use its existing visual-similarity system to suggest
locations for images without GPS metadata.

## Concept

If an untagged image is visually similar to multiple geotagged images and
those images form a sufficiently tight geographic cluster, PicFetch can
suggest that location to the user.

Example:

```text
IMG_4827.JPG
No location information

Possible location found:

Aachen, Germany
Approx. coordinates: 50.77, 6.08

Based on:
  8 visually similar images
  7 located within ~350 m

[ Show Similar Images ]
[ Apply Location ]
[ Ignore ]
```

## Important Principle

Location information should **never be copied automatically**.

PicFetch should make a suggestion and require explicit user confirmation.

Existing GPS metadata should never be silently overwritten.

## Confidence Calculation

A future location suggestion could combine several signals:

1. **Visual similarity**
   - Existing similarity embeddings/scores.

2. **Geographic agreement**
   - Similar images should form a geographic cluster.
   - Widely distributed matches should reduce confidence.

3. **Temporal proximity**
   - Images captured close together in time provide additional evidence.

4. **Sequence proximity**
   - Adjacent images from the same camera/import/session may strengthen the
     suggestion.

For example:

```text
Visual similarity      █████████░  91%
Geographic agreement   ████████░░  84%
Temporal proximity     ██████████  97%

Location confidence: HIGH
```

The exact scoring mechanism can be determined experimentally.

## Batch Suggestions

The feature could also identify groups of untagged images that likely belong
to the same location.

Example:

```text
12 images appear to have been taken at the same location.

Location inferred from 6 geotagged similar images.

[ Review Images ]
[ Apply to All ]
[ Ignore ]
```

This could be particularly useful for:

- Imported photo archives
- Cameras without GPS
- Photos where only some devices recorded GPS
- Collections containing photos from multiple cameras taken during the same
  trip or event

## Safety / Data Integrity

- Never overwrite existing GPS coordinates automatically.
- Always require explicit confirmation.
- Allow individual and batch review.
- Show the evidence behind a suggestion.
- Prefer no suggestion over a low-confidence suggestion.
- Future GPS editing should reuse applicable file-mutation transactions, but
  requires its own GPS writer and recovery/undo design: PicFetch currently has
  neither a general GPS-writing API nor a metadata undo mechanism.

---

# Possible Later Map Features

Once the basic Location Map exists, it could become the foundation for:

- Timeline filtering on the map
- Geographic collection filtering
- Heatmaps
- Trip/travel visualisation
- Route reconstruction
- "Photos near this location"
- Geographic filtering in Similarity Explorer
- Location Suggestions
- Batch geotagging
- Finding images with suspicious/outlier GPS coordinates
