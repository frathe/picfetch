# Multilingual Markdown-Generated Website

Status: resolved

## Problem Statement

PicFetch currently publishes an English website in two separately maintained forms: a regular HTML page and an AMP page. The same content is copied between them by hand. This duplication makes ordinary edits error-prone, and adding German would expand the problem to four independent pages that can drift in content, links, metadata, and behavior.

German-speaking visitors cannot currently read localized page content or select German. Search engines also receive no language-alternate metadata. The desired publishing workflow must continue to work with the existing static GitHub Pages deployment, which publishes committed output from the documentation directory after a push.

The maintainer needs one English Markdown document at the repository root to be the only authored content source. A local Makefile-driven workflow must translate changed content, generate regular and AMP pages in English and German, validate the result, and update the committed deployment artifacts.

## Solution

Introduce a small Go-based static-site generator driven by one root-level English Markdown document. The document will use YAML front matter for structured metadata, assets, links, and repeated page elements, with Markdown sections for prose. Separate regular and AMP templates will consume one validated content model so both formats stay aligned without forcing layout code into the content document.

Generate four independently addressable pages:

| Experience | Public route |
| --- | --- |
| English regular | `/` |
| German regular | `/de/` |
| English AMP | `/amp/` |
| German AMP | `/de/amp/` |

English content will render directly from the source document. German content will come from a committed, derived, hash-keyed translation cache populated through DeepL. Translation refreshes will be explicit and networked; ordinary builds will be deterministic and offline.

Regular pages will offer a top-right English/German selector and remember an explicit choice. A first visit to the English root will redirect browsers whose preferred language begins with `de` to the German regular page. AMP pages will provide the same visible selector but will not attempt unsupported automatic page-load redirection.

All four pages will expose correct language, canonical, AMP, alternate-language, social, accessibility, and search metadata while preserving the current design, responsive behavior, dark mode, media, downloads, lightbox behavior, and AMP validity.

## User Stories

1. As an English-speaking visitor, I want the existing root URL to remain English, so that current bookmarks and search results continue to work.
2. As a German-speaking visitor, I want a complete German regular page, so that I can understand PicFetch without reading English.
3. As a German-speaking AMP visitor, I want a complete German AMP page, so that the accelerated experience is also localized.
4. As a visitor using either language, I want the regular and AMP pages to present equivalent content, so that page format does not change the product information I receive.
5. As a first-time visitor whose browser prefers German, I want the English root entry point to take me to the German regular page, so that I receive an appropriate language automatically.
6. As a German speaker in Germany, Austria, or Switzerland, I want any browser preference beginning with `de` to be recognized, so that detection does not depend on one regional locale.
7. As a visitor following a direct German URL, I want that explicit route to remain open regardless of my browser language, so that shared links are respected.
8. As a visitor following an AMP URL, I want the selected AMP page to remain open, so that the static AMP implementation does not perform an unsupported redirect.
9. As a visitor, I want a visible English/German control at the top right, so that I can override automatic selection at any time.
10. As a visitor, I want the selector to include UK and German flag indicators, so that the available languages are quickly recognizable.
11. As a screen-reader user, I want textual accessible names for both language choices, so that the flags are not the only source of meaning.
12. As a visitor who manually chooses English, I want that preference remembered before returning to the English root, so that browser detection does not immediately send me back to German.
13. As a visitor who manually chooses German, I want that preference remembered, so that later visits honor my explicit decision.
14. As a privacy-conscious visitor, I want language preference handling to remain local to my browser, so that it does not require an account or analytics service.
15. As a search-engine crawler, I want every regular page to identify its own canonical URL, so that English and German are indexed as distinct localized pages.
16. As a search-engine crawler, I want every AMP page to identify the same-language regular page as canonical, so that AMP and canonical content are paired correctly.
17. As a search-engine crawler, I want each regular page to advertise its same-language AMP counterpart, so that AMP discovery works for both languages.
18. As a search-engine crawler, I want English, German, and default language alternates declared, so that users can be sent to the appropriate localized result.
19. As a search-engine crawler, I want regular pages linked to regular language peers and AMP pages linked to AMP language peers, so that format and locale relationships are unambiguous.
20. As someone sharing the website socially, I want localized title, description, and locale metadata, so that link previews match the selected language.
21. As a visitor using assistive technology, I want alternative text, ARIA labels, warnings, and controls localized, so that German support is not limited to visible paragraphs.
22. As a German visitor, I want a clear DeepL translation disclosure, so that the origin of unedited machine-translated copy is transparent.
23. As a maintainer, I want to edit one English content document, so that regular, AMP, English, and German output cannot acquire separate hand-maintained copy.
24. As a maintainer, I want metadata, images, videos, links, downloads, feature cards, and prose represented in that source, so that template prose does not become a hidden second content source.
25. As a maintainer, I want YAML front matter for structured values and Markdown for prose, so that the source remains readable while representing the existing page accurately.
26. As a maintainer, I want commands, URLs, filenames, keyboard shortcuts, product names, and technical identifiers excluded from translation, so that machine translation cannot corrupt them.
27. As a maintainer, I want only changed or missing English text sent for translation, so that the limited free translation allowance is conserved.
28. As a maintainer, I want translated entries tied to stable content identities and source hashes, so that an English edit invalidates exactly the affected German text.
29. As a maintainer, I want obsolete derived translation entries cleaned up predictably, so that the cache does not accumulate abandoned content indefinitely.
30. As a maintainer, I want translation failures to leave the prior cache and generated pages intact, so that a network problem cannot partially corrupt the published site.
31. As a maintainer, I want missing German translations to fail a build, so that mixed-language pages are never published silently.
32. As a maintainer, I want ordinary page generation to work without network access or credentials, so that builds are fast and reproducible.
33. As a maintainer, I want the translation API key supplied only through local environment configuration, so that it cannot be committed or printed accidentally.
34. As a maintainer, I want a single translation command, so that refreshing German copy is deliberate.
35. As a maintainer, I want a single offline build command, so that all four pages can be regenerated deterministically from checked-in inputs.
36. As a maintainer, I want a single update command, so that translation, generation, and validation can be performed in the correct order.
37. As a maintainer, I want a single check command, so that I can tell whether tests pass and committed website artifacts are current.
38. As a maintainer reviewing an update, I want translation-cache and generated-page changes visible in the Git diff, so that I can inspect them before pushing.
39. As a maintainer, I want generated files treated as deployment artifacts rather than editing surfaces, so that manual changes are not lost or allowed to drift.
40. As a maintainer, I want the current download URLs, external links, images, videos, and platform details preserved, so that localization does not change product behavior.
41. As a regular-page visitor, I want the existing screenshot lightbox and Vimeo embeds to keep working, so that generation does not degrade the interactive experience.
42. As an AMP visitor, I want AMP-native images, Vimeo embeds, and screenshot lightbox behavior to remain valid, so that the AMP page retains its current capabilities.
43. As a visitor using a narrow screen or dark color scheme, I want the current responsive layout and theme behavior preserved, so that localization does not introduce a visual regression.
44. As a deployment maintainer, I want all generated pages committed beneath the existing static publishing root, so that the current push-to-publish behavior continues without a new hosting service.
45. As a deployment maintainer, I want validation to catch stale output, broken internal navigation, missing locale relationships, and invalid AMP before a push, so that GitHub Pages receives deployable artifacts.
46. As a developer without a live DeepL key, I want automated tests to use a controlled fake translation service, so that routine verification is reliable and free.
47. As a developer diagnosing a content error, I want failures to identify the affected content field or section, so that malformed source or missing translation data is easy to repair.
48. As a future maintainer, I want Go module versions and external validation tooling pinned, so that the local workflow does not change unexpectedly over time.
49. As a repository maintainer, I want branches whose names contain `website` or `webpage` kept separate from `main`, so that independently published website history can never enter the application branch.

## Implementation Decisions

- The site will have one authored English content source. German text is generated data, not a second editorial source, and generated HTML must not be edited manually.
- The English source will be a root-level Markdown document with YAML front matter. Front matter will describe structured metadata, media, calls to action, screenshots, features, download groups, notices, footer links, accessibility labels, and other non-prose values. Markdown sections will hold longer prose.
- The source schema will explicitly distinguish translatable strings from opaque values. Translation exclusion will not rely solely on guessing from text shape. URLs, commands, code, filenames, product and technology names, keyboard labels, IDs, and asset dimensions remain unchanged.
- A Go generator with pinned module dependencies will parse and validate the source, extract translation units, load cached translations, create one shared page model per locale, and render the regular and AMP variants.
- Regular and AMP markup will remain separate templates because their valid media and interaction elements differ. Both templates will consume the same locale-specific page model, section ordering, links, and assets to prevent content drift.
- Template files may contain structure and presentation but no independently maintained user-facing prose. Language-sensitive template labels must come through the content and translation pipeline.
- The German translation cache will be committed, derived data keyed by a stable semantic content identity plus a hash of the current English source text. A changed hash is a cache miss. Translation refreshes will request only missing or changed units and will remove obsolete derived entries deterministically.
- The supported translation backend will be DeepL API Developer. The generator will not scrape the consumer Google Translate website and will not silently fall back to another provider.
- Translation requests will send extracted text or protected markup, never unparsed Markdown. The request must target German while preserving non-translatable tokens and structural markup.
- The DeepL API key will be read from `DEEPL_API_KEY` or an explicitly ignored local environment file. It must never appear in committed configuration, generated output, logs, diagnostics, or test fixtures.
- Translation updates will be atomic. Authentication, quota, rate-limit, response-validation, or network failures must return a clear error without partially replacing the prior cache or deployment output.
- An offline build will fail when a required German entry is absent or stale. It must not fall back to English inside a German page.
- The Makefile contract will expose `make build` for offline deterministic generation, `make translate` for refreshing missing or changed German units, `make update` for translate/build/validate, and `make check` for automated verification and stale-output detection.
- The generated deployment matrix will be English regular at `/`, German regular at `/de/`, English AMP at `/amp/`, and German AMP at `/de/amp/`.
- The regular English page will self-canonicalize and advertise the English AMP page. The regular German page will self-canonicalize and advertise the German AMP page. Each AMP page will canonicalize to its same-language regular page.
- Regular pages will advertise English and German regular alternates plus English as `x-default`. AMP pages will advertise English and German AMP alternates plus English AMP as `x-default`.
- Document language will be `en` for English and `de` for German. Open Graph locale metadata will use the corresponding British-English and German locale values, including alternates where appropriate.
- All visible and machine-consumed language-sensitive content will be localized, including titles, descriptions, Open Graph fields, screenshot captions, alternative text, ARIA labels, feature copy, warnings, footer copy, and selector labels.
- German pages will include the disclosure required for unedited public DeepL output. The disclosure itself will be placed consistently in both regular and AMP variants without disrupting the current visual hierarchy.
- Regular pages will include a top-right language selector with UK and German flag indicators and textual accessible names. Each choice links to the equivalent regular-language page.
- AMP pages will include a crawlable top-right selector using ordinary links between equivalent AMP-language pages. Flags will not be the sole accessible label.
- Automatic detection applies only when loading the English regular root. If no explicit preference exists, the client examines the ordered browser-language preferences and redirects to the German regular route when the first applicable preference begins with `de`; otherwise it remains in English.
- An explicit regular-page selector choice is saved locally before navigation and takes precedence over browser language on later root visits. Explicit German and all AMP routes are never automatically redirected.
- Detection is based on language, not IP-derived country. It must not introduce geolocation, analytics, user accounts, or server-side state.
- The existing GitHub Pages arrangement remains the deployment mechanism. The local generator writes committed artifacts beneath the existing publishing root; pushing those artifacts remains the deployment trigger.
- Existing presentation and behavior are preserved: page structure, styling, responsive layout, dark mode, artwork, screenshots, downloads, warnings, footer links, regular-page lightbox/Vimeo behavior, and AMP-native equivalents.
- Generated writes should be deterministic and atomic. Identical source, templates, and translation cache must produce byte-identical output, including stable ordering and formatting.
- Any branch whose name contains `website` or `webpage`, matched case-insensitively, is publish-only relative to `main`: it is pushed and published independently and must never be merged into `main`.

## Testing Decisions

- The primary test seam is the public Makefile contract, especially `make check`. Tests should assert observable generated-site behavior rather than internal helper calls or implementation structure.
- `make check` will generate the complete site into an isolated temporary destination and compare it with committed deployment artifacts. It fails when a source, template, or translation-cache change has not been regenerated.
- The generated-site check will assert that exactly the four required route variants exist and that each declares the expected document language.
- Output checks will verify the canonical, AMP, `hreflang`, `x-default`, and Open Graph locale matrix for every page as an external contract.
- Output checks will verify that the language selector is present, points to the correct same-format counterpart, exposes accessible English and German names, and retains the agreed flag indicators.
- A browser-level behavior test will cover first-visit German detection, non-German fallback, recognition of regional German preferences, stored-choice precedence, and the absence of automatic redirects on explicit German and AMP routes.
- Content-level checks will prove that every current translatable unit has a matching German cache entry and that opaque technical values survive unchanged in all outputs.
- Regression assertions will cover representative current product content, media IDs, image metadata, download targets, warning commands, footer destinations, regular-page interactions, and AMP-native element choices without snapshotting irrelevant whitespace or private template structure.
- Both AMP outputs will be checked with a pinned official AMP validator or an equivalently authoritative validator. Validation is required for `make update` and `make check` to succeed.
- Link checks will validate internal anchors, local asset references, alternate/canonical relationships, and URL syntax. Routine tests will not depend on live availability of external sites.
- The translation client will have a contract test against a controlled local HTTP server. It will verify authentication placement, target language, request batching/escaping, protected content, valid response mapping, and clear handling of authentication, quota, malformed-response, and partial-failure cases.
- No routine test will call the live DeepL service, consume translation allowance, require a real credential, or depend on network access.
- Failure-path tests will verify that missing credentials prevent only networked translation, missing or stale cache entries prevent German generation, and failed updates leave prior cache/output untouched.
- Determinism will be tested by generating twice from identical inputs and comparing bytes. Cache updates will also be tested for changed-only requests and deterministic removal of obsolete entries.
- Source-validation tests will use invalid high-level documents to confirm that missing required metadata, duplicate identities, malformed structured elements, or unsupported content shapes produce actionable field/section diagnostics.
- There is no existing generator or website test suite to copy on this branch. The new command-level check is therefore the first and preferred prior-art seam for subsequent website-generation work.
- A final manual smoke check should inspect regular and AMP pages in both languages at desktop and narrow widths, in light and dark modes, before the first deployment. This complements rather than replaces the automated contract.

## Out of Scope

- Adding a server, CDN worker, edge redirect, or other hosting layer for `Accept-Language` negotiation.
- Automatic page-load redirection on AMP pages.
- Adding languages beyond English and German.
- Building a runtime CMS, translation dashboard, or browser-based editor.
- Maintaining an independently authored German source document or a parallel set of manual German overrides.
- Translating the PicFetch application, repository documentation, user manual, contributing guide, security policy, release artifacts, or linked third-party pages.
- Replacing DeepL with a provider abstraction, provider fallback chain, scraped consumer endpoint, or self-hosted translation service.
- Changing download URLs, media assets, video IDs, product claims, external destinations, or application behavior.
- Redesigning the website or performing unrelated accessibility, semantic, or metadata cleanup.
- Fixing the existing manifest application name, introducing a new textual page heading, or addressing other pre-existing issues not required for generation and localization.
- Changing the current GitHub Pages publishing configuration or adding a new remote CI/CD pipeline.
- Modifying or cleaning unrelated staged, deleted, ignored, or untracked worktree files.

## Further Notes

- The current branch contains hand-written regular and AMP HTML with duplicated English content, inline styling, no generator, no dependency manifest, and no automated website tests.
- GitHub Pages is understood to publish committed content from the existing documentation directory on push; generation therefore remains a local pre-push responsibility.
- The maintainer supplied a DeepL credential through ignored local configuration and explicitly authorized the first production translation refresh. The resulting German cache is derived output; the credential was not printed, rendered, or committed.
- DeepL plan allowances and terms can change. At specification time, the Developer offering is the preferred low-cost hosted option, and public unedited output requires disclosure. See the [DeepL API plan documentation](https://support.deepl.com/hc/en-us/articles/360021200939-DeepL-API-plans) and [DeepL terms](https://www.deepl.com/en/pro-license).
- Static AMP can detect browser language inside constrained scripting, but supported AMP APIs do not provide reliable automatic top-level page-load navigation. The manual AMP selector is intentional. See the [`amp-script` documentation](https://amp.dev/documentation/components/amp-script) and [AMP internationalization example](https://amp.dev/documentation/examples/guides/internationalization/).
- Separate URLs and `hreflang` metadata are the durable discovery mechanism; automatic redirects must not make localized URLs inaccessible. See [Google's multilingual-site guidance](https://developers.google.com/search/docs/specialty/international/managing-multi-regional-sites).
- Machine-generated German changes and all four generated pages are reviewed together in the local Git diff before pushing. The existing GitHub Pages publication then remains unchanged.
