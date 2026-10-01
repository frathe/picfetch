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
- Historical pre-fix finding: EXIF discarded response freshness headers and
  cached PNG bytes without expiry metadata. The implementation at `3eafe3d`
  retains bounded freshness/validator metadata, conditionally validates expired
  entries, merges 304 responses and uses an owned renderer without the upstream
  global cache bypass. The review follow-up also clears expired displayed pixels
  after failed validation and binds one-shot delivery to the current view.
  These fixes address that concrete technical gap; memory eviction/restart still
  does not guarantee seven-day retention or certify sufficient cache capacity.
- EXIF warms a fixed 5-by-5 neighborhood before painting. Whether that exceeds
  permitted modest look-ahead depends on the displayed viewport; do not label
  this bulk scraping without further assessment. Prefer viewport-derived demand.
- Historical pre-fix finding: Location Map showed plain contributor credit.
  Both map surfaces now display attribution linked directly to the copyright/
  license page, and THIRD-PARTY-NOTICES retains that link. The attribution research
  records the primary-source reasoning for this change. A new signed candidate
  and native visual/Store qualification remain pending.

Recommendation: correct/verify map-service compliance, then use the rights-held
Yes declaration rather than No. This scoped technical review is not a blanket
legal certification. This record incorporates the implementation and review
follow-up; no App Store rights answer was saved and no replacement was uploaded.
