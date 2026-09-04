# 05: Publish German AMP and complete page discovery

**What to build:** Complete the four-page site with a localized German AMP experience and an unambiguous canonical, AMP, language-alternate, and social-metadata graph across every public route.

**Blocked by:** 02: Generate the English AMP page from the shared source; 04: Publish the German regular experience

**Status:** resolved

- [x] The German AMP route is generated from the shared content model and current German cache with no independently maintained prose.
- [x] The German AMP page declares `de` and contains complete localized visible, metadata, alternative-text, ARIA, warning, footer, selector, and disclosure content.
- [x] The German AMP page preserves AMP-native images, videos, screenshot lightbox behavior, styling, responsive behavior, dark mode, downloads, and external links.
- [x] Both AMP pages provide ordinary crawlable links between equivalent AMP-language routes using UK and German flag indicators plus textual accessible names.
- [x] AMP pages do not attempt automatic page-load language redirection or rely on country-based geolocation.
- [x] Each regular page self-canonicalizes and advertises its same-language AMP page.
- [x] Each AMP page canonicalizes to its same-language regular page.
- [x] Regular pages advertise English and German regular alternates plus English regular as `x-default`.
- [x] AMP pages advertise English and German AMP alternates plus English AMP as `x-default`.
- [x] English and German documents expose the agreed document-language and localized Open Graph locale values.
- [x] Relative local assets and metadata references resolve correctly at every route depth.
- [x] `make build` now generates all four public variants offline, atomically, and deterministically, and fails when any German translation is absent or stale.
- [x] A pinned authoritative AMP validator accepts both AMP documents.
- [x] High-level tests validate the complete route/metadata matrix, same-format selector destinations, localized content completeness, protected values, and regular-versus-AMP content parity.
- [x] All four generated deployment artifacts are committed and agree with a fresh build.

## Answer

Completed `/de/amp/` and the four-route discovery graph from the shared source and
cache. Contract tests verify canonical, same-language AMP, `hreflang`, `x-default`,
Open Graph locale, selector, protected-value, parity, and route-depth behavior.
Both AMP files pass the pinned offline validator and all four artifacts agree with
a clean deterministic build.
