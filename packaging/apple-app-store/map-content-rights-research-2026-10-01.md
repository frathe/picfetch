# Map content rights check — 2026-10-01

## Conclusion

Apple's live Content Rights form covers content an app contains, shows **or
accesses**. PicFetch downloads and displays OpenStreetMap tiles, so absence from
the bundle does not support answering No. Open licensing can supply permission;
it does not mean the content is first-party. No declaration was saved during
this check. Ronin reports that the app's artwork/images were generated with
OpenAI; this report does not independently audit those assets or their inputs.

## Primary sources

- [OSM copyright](https://www.openstreetmap.org/copyright): ODbL map-data reuse
  requires credit and making the license clear. The copyright-page link is an
  accepted way to identify that license.
- [OSM tile policy](https://operations.osmfoundation.org/policies/tiles/): permits
  interactive viewing subject to attribution, app identification and caching.
  Requests must honor freshness headers or, without that support, retain tiles
  for at least seven days. Bulk/offline downloads are prohibited; modest local
  look-ahead is permitted. Service availability is not guaranteed.
- [OSMF terms](https://osmfoundation.org/wiki/Terms_of_Use): service usage policies
  apply in addition to the data license.

## Implementation evidence and limitations

- Both `internal/ui/locationmap/tiles.go` and `internal/ui/exifwin/tiles.go`
  use the prescribed HTTPS tile endpoint and identify PicFetch in User-Agent.
- Location Map parses freshness headers, retains validators and sends conditional
  requests. Its cache is memory-only; this check does not certify retention across
  restarts or eviction patterns against the service's sufficient-cache requirement.
- EXIF's `tileFetcher.get` discards response freshness headers and returns only
  PNG bytes. `tilework.go` stores these in a byte-bounded in-memory cache without
  age/expiry metadata. There is neither header-based freshness handling nor a
  seven-day retention guarantee. This is a concrete gap against the tile policy,
  not evidence that PicFetch has no license to OSM data.
- EXIF warms a fixed 5-by-5 neighborhood before painting. Whether that exceeds
  permitted modest look-ahead depends on the displayed viewport; do not label
  this bulk scraping without further assessment. Prefer viewport-derived demand.
- Location Map shows plain contributor credit. THIRD-PARTY-NOTICES includes the
  copyright/license link. Attribution placement is being checked separately in
  `map-attribution-research-2026-10-01.md`.

Recommendation: correct/verify map-service compliance, then use the rights-held
Yes declaration rather than No. This scoped technical review is not a blanket
legal certification. No app code or App Store rights answers changed here.
