# 06: Guard the local publishing workflow

**What to build:** Give maintainers one reliable local update-and-check workflow that prevents incomplete translations, stale pages, invalid AMP, broken internal navigation, or nondeterministic output from reaching the existing GitHub Pages deployment.

**Blocked by:** 05: Publish German AMP and complete page discovery

**Status:** resolved

- [x] `make update` performs translation refresh, four-page generation, and validation in the agreed order and stops immediately on failure.
- [x] `make check` runs automated tests, generates into an isolated temporary destination, and fails when committed deployment artifacts differ from a clean generation.
- [x] The check proves that exactly the four expected route variants are generated with the correct document languages and complete translation coverage.
- [x] The check validates canonical, AMP, `hreflang`, `x-default`, Open Graph locale, and selector relationships across the full page matrix.
- [x] Internal anchors, local asset references, alternate relationships, and URL syntax are verified without depending on live third-party availability.
- [x] Both AMP documents pass pinned authoritative validation as part of the update and check workflows.
- [x] Repeated builds from identical source, templates, dependencies, and cache produce byte-identical output.
- [x] Translation and generation failures leave the previously committed cache and deployment output unchanged.
- [x] Routine checks require neither a DeepL credential nor network access; only the explicit translation/update path requires the credential.
- [x] All Go modules and external validation tools used by the workflow are pinned for reproducibility.
- [x] Maintainer documentation explains the content-edit, credential, translate, build, update, check, Git-diff review, and push-to-publish workflow without exposing secrets.
- [x] Maintainer documentation states that branches whose names contain `website` or `webpage`, matched case-insensitively, are published independently and must never be merged into `main`.
- [x] A final smoke check confirms both languages and both formats at desktop and narrow widths, in light and dark modes, including selectors, videos, screenshots, downloads, and regular/AMP interaction differences.
- [x] Existing GitHub Pages publishing behavior is unchanged and unrelated worktree files are not modified or cleaned up.

## Answer

Added transactional `make update` and offline `make check` workflows with exact
four-route enforcement, deterministic/stale comparisons, strict cache and source
validation, safe link and path checks, both AMP validators, and browser contracts.
The final smoke matrix covered four routes, two widths, and two color schemes.
`site/README.md` documents credentials, editing, review, publishing, and the required
website-branch separation; unrelated pre-existing worktree files were preserved.
