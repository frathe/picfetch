# 02: Generate the English AMP page from the shared source

Type: task
Status: resolved
Blocked by: 01

**What to build:** Generate the English AMP experience from the same content model as the regular page so maintainers no longer duplicate English content while AMP visitors retain the existing accelerated experience.

- [x] The generator renders an English AMP page from the content model introduced by Ticket 01 without adding a second content source.
- [x] Regular and AMP renderers receive the same section ordering, text, links, assets, metadata, and accessibility values.
- [x] The AMP renderer uses valid AMP-native image, Vimeo, and screenshot-lightbox elements with required component declarations, dimensions, boilerplate, and custom styling.
- [x] The generated AMP page preserves the current content, visual design, responsive behavior, dark mode, video behavior, screenshot gallery, downloads, warning, and footer.
- [x] The English AMP page declares English as its document language and canonicalizes to the English regular route.
- [x] The English regular page advertises the English AMP route.
- [x] `make build` generates both English variants offline and deterministically.
- [x] A pinned authoritative AMP validator accepts the generated English AMP page.
- [x] High-level regression tests compare representative content and links across the English regular and AMP outputs while allowing their required markup to differ.
- [x] Both generated English deployment artifacts are committed and agree with a fresh build.

## Answer

Added a separate AMP template backed by the same content model and section order.
The generated page uses AMP-native responsive images, Vimeo embeds, screenshot
lightbox markup, required component declarations, and boilerplate. The official
`amphtml-validator` package and validator payload are pinned locally, checksummed,
and invoked offline by `make validate-amp`. Both English artifacts are deterministic
and agree with a fresh build.
