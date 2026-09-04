# 04: Publish the German regular experience

**What to build:** Publish a complete German regular page and give regular-page visitors accessible manual language controls plus respectful first-visit browser-language selection.

**Blocked by:** 03: Refresh German translations through DeepL

**Status:** resolved

- [x] The German regular route is generated entirely from the English source model and the current German translation cache.
- [x] The German document declares `de` and localizes visible copy, titles, descriptions, social metadata, screenshot captions, alternative text, ARIA labels, warnings, footer text, and selector labels.
- [x] Opaque commands, URLs, filenames, keyboard shortcuts, product names, media identifiers, and technical values remain intact.
- [x] The German page includes the required disclosure for unedited public DeepL output without disrupting the current design.
- [x] Both regular pages display a top-right selector with UK and German flag indicators, ordinary language links, and textual accessible names.
- [x] Selecting a regular-page language records that explicit preference locally before navigation and requires no account, analytics, geolocation, or server-side state.
- [x] On a first visit to the English root with no stored preference, ordered browser language preferences beginning with `de` redirect to the German regular route; other language preferences stay in English.
- [x] A stored explicit English choice prevents German browser detection from redirecting the visitor away from English.
- [x] The explicit German route is never automatically redirected based on browser language.
- [x] The two regular pages self-canonicalize and expose English, German, and English `x-default` regular-page alternates with localized Open Graph locale metadata.
- [x] Existing regular-page media, downloads, responsive styling, dark mode, and lightbox behavior work in both languages.
- [x] Browser-level behavior tests cover German regional preferences, non-German fallback, stored-choice precedence, manual switching, and absence of redirects from the explicit German route.
- [x] High-level output tests prove that every required German unit is present and that the generated English and German regular artifacts are current.

## Answer

Generated the complete `/de/` page from the shared model and production cache,
including localized metadata and accessibility text, protected technical values,
the unedited-DeepL disclosure, and an accessible two-language selector. Browser
contracts cover regional German preferences, explicit-choice precedence, direct
German navigation, manual selection, and storage-denied behavior; visual smoke
confirmed responsive, dark-mode, media, download, and lightbox presentation.
