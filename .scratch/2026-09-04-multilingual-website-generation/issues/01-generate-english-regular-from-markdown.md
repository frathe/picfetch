# 01: Generate the English regular page from Markdown

Type: task
Status: resolved
Blocked by: none

**What to build:** Give maintainers one readable English Markdown content source and an offline Go-based build that deterministically reproduces the existing regular website without changing what visitors see or how it behaves.

- [x] One root-level English Markdown document is the only authored source for page metadata, assets, links, repeated content, accessibility labels, and prose.
- [x] The document uses YAML front matter for structured values and Markdown sections for prose, with stable identities for repeated or translatable content.
- [x] The source schema explicitly distinguishes translatable text from opaque URLs, commands, filenames, keyboard labels, IDs, dimensions, product names, and technical terms.
- [x] A Go generator with pinned dependencies parses and validates the source and reports malformed or missing data using actionable content-field or section names.
- [x] `make build` works offline and generates the English regular page from the source rather than copied template prose.
- [x] The regular-page template contains presentation and structure but no independently maintained user-facing copy.
- [x] Generated output preserves the current sections, wording, metadata, artwork, screenshots, videos, download links, warning content, footer links, responsive layout, dark mode, and screenshot lightbox behavior.
- [x] Generated writes are atomic and byte-for-byte deterministic for identical inputs.
- [x] High-level tests exercise the build command, representative source validation failures, current links/media, and visitor-visible regular-page behavior without coupling to private helper functions.
- [x] The generated English regular deployment artifact is committed and agrees with a fresh build.

## Answer

Implemented `website.md`, a strict Go content parser/validator, a presentation-only
regular template, and the public `make build` contract. The generated root page
preserves the existing content, media, downloads, styling, dark mode, Vimeo embeds,
and lightbox behavior. Verified with `go test ./sitecontract`, `go vet ./...`, a
fresh committed-artifact generation, the original link/media inventory, and
`git diff --check`.
