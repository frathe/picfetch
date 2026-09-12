# Embedded artwork

The original illustrations and full Codex pet atlases remain under `assets/`.
`main.go` derives only the artwork that PicFetch embeds. It is development
tooling, not part of application startup or the shipped dependency closure.

The generator uses Go's existing `golang.org/x/image` dependency for decoding
and Catmull-Rom resizing, then local `cwebp` for lossless WebP output. PNG output
uses the standard library. The local encoder used for qualification is libwebp
1.6.0 (`cwebp -version`); it is BSD-3-Clause development tooling and is not
redistributed with PicFetch. No image-generation service is involved.

```sh
go run ./scripts/appassets
go run ./scripts/appassets -check
```

Checking requires no `cwebp`: it compares dimensions, every alpha value, and
exact RGB values wherever alpha is nonzero against the retained source
transformation. Ordinary illustrations may differ in RGB beneath fully
transparent pixels, which lossless optimizers such as ImageOptim can rewrite
without changing their appearance. Gaze atlases still require every decoded
RGBA byte to match, including RGB beneath transparent pixels. Encoder byte
differences do not invalidate equivalent pixels. Regeneration requires `cwebp`
on PATH. The generated files are checked in; ordinary application builds do
not re-encode artwork. Vector generation is a separate offline Go-only build step.

The atlas cells are pixel-identical to the originals. Resized illustrations are
also encoded losslessly after resampling to avoid another lossy encoding pass.
Some formerly lossy WebPs therefore grow individually despite having fewer
pixels; the combined embedded artwork is substantially smaller. Existing alpha
edges are retained rather than retouched.

## Dimensions

Ordinary illustrations target two physical pixels per logical display unit.
This covers standard 2x Retina rendering; larger user-selected Fyne scaling
still scales the same artwork. Dimensions preserve aspect ratios to the nearest
pixel. No cropping removes artwork from these illustrations.

| Embed | Output pixels | Display constraint |
| --- | --- | --- |
| `assets/trane.webp`, `help/finis.webp` | 3264 × 208 | 17 original 192 × 208 gaze cells in one row; 16 directions clockwise from up, then neutral. |
| `assets/welcome.webp` | 360 × 434 | Fallback in the 180-wide welcome slot. |
| `assets/placeholder.webp` | 360 × 394 | 180-wide error-art slot. |
| `assets/digging.webp` | 360 × 240 | Scan VBox fixes logical height at 120. |
| `assets/comparingImages.webp` | 520 × 475 | About border fixes logical width at 260. |
| `assets/explorer-intro.png` | 640 × 480 | Explorer VBox fixes logical height at 240. |
| `help/TaneWithFrame.webp` | 629 × 440 | Manual RichText image row has logical height 220. |
| `help/trane_digging.webp` | 660 × 440 | Same manual image-row constraint. |
| `help/trane_wags.webp` | 365 × 440 | Same manual image-row constraint. |

Embed paths above are relative to `internal/ui/`. Gaze frames are never resized;
the runtime still applies Trane's existing fringe correction after decoding.
The packaged 1024 × 1024 app icon retains its source resolution because macOS
and Store packaging generate their own platform-specific icon sizes.

## Sources

- Trane: unchanged `assets/trane/codex-pet/spritesheet.webp` and its existing
  [provenance](../../assets/trane/codex-pet/README.md).
- Finis: the user-supplied original previously embedded directly as
  `internal/ui/help/finis.webp`, retained as `assets/finis/spritesheet.webp`.
- Manual illustrations: unchanged corresponding originals under `assets/trane/`.
- Viewer illustrations: exact pre-resize embedded files retained under
  `assets/ui-originals/`; Explorer generation/background-removal provenance is
  in [explorer-intro.md](../../assets/trane/explorer-intro.md).

These transformations introduce no new artwork or model assets. Preserve
existing source attribution and shipped license/notice obligations.
