# Transparent PNG generation

Investigated on 2026-09-13 to prevent future Trane illustrations from requiring
repeated generation and manual background removal.

**Verification status:** direct no-reference generation and a neutral Trane
illustration using one inspected reference are verified. B6's untouched PNG
passed raw-alpha inspection and light/dark visual QA, completing the scoped
**reference-based native-alpha verification**. The earlier B3/B4 service errors
remain part of the evidence; B6 does not establish their cause.

**Verified result:** the built-in `image_gen` tool returned genuine RGBA PNGs
directly for a simple sphere, a furry retriever with a cyan beam/glow, three
chess pieces, and neutral Trane with an identity/style reference. All four
untouched outputs passed alpha inspection and light/dark compositing. Use the
concise request below for the requested asset's first generation, then validate
the raw file immediately. There is no need to repeat this investigation before
each asset.

This is evidence that the available tool can deliver useful transparency without
local cleanup. It does **not** prove which prompt difference caused success,
guarantee the next generation, or expose the service's internal rendering or
postprocessing. The reference-based success covers this neutral pose with one
reference; more complex Trane scenes and first-attempt reliability remain
unverified.

## Workflow for the next request

1. Read the active imagegen skill and inspect the live tool schema. Use its
   supported controls. In the tested session, built-in `image_gen` accepted only
   `prompt`, `referenced_image_paths`, and `num_last_images_to_include`.
2. Describe the requested subject, action and composition concisely. End with:

   ```text
   Genuinely transparent PNG with an alpha channel; all empty pixels have alpha 0.
   ```

   When needed, add: `Preserve softly translucent fur edges and localized cyan
   glow.` Keep effects localized and leave clear edge padding. These are prompt
   words, not a hidden `background` API parameter.
3. For Trane, prefer the successfully tested neutral
   `assets/trane/trane_wags.webp` as the starting reference when appropriate to
   the requested art. Inspect it with `view_image` and verify its actual alpha.
   B6 confirmed RGBA with 718,206 alpha-zero pixels and preserved its blue collar
   and gold tag. Start from the [exact successful B6 prompt](#b6-neutral-trane-reference)
   below, adapting only the requested pose or scene while preserving character
   identity. Label the image explicitly as an **identity and style reference**
   for a new illustration, use only references needed to preserve the character,
   and keep the reference separate from the requested scene. B6 verifies one
   neutral case; this preparation is not a proven transparency switch or a
   guarantee of reliability, and it does not explain the earlier service errors.
4. Preserve the exact returned file. Run the raw-alpha gate below **before**
   resizing, masking, re-encoding, or copying it into app assets. A displayed
   checkerboard and the `.png` extension are insufficient evidence. Converting
   opaque RGB to RGBA in a saved file is not a fix.
5. Inspect the full image and edges over solid white and dark backgrounds. Check
   recognizable foreground, eye/teeth/device details, near-opaque solid regions,
   transparent background holes, and smooth localized effects. A successful
   alpha channel can still contain an unacceptable matte or excessive haze.
6. After acceptance, optimize dimensions and PNG encoding, retain alpha, and
   validate again. Save non-destructively to the user-requested destination.
   Keep raw files, prompts and QA evidence outside application assets.

The successful prompts use positive transparency wording and do not enumerate
unwanted background textures. Prefer that concise form over copying the long
failed Trane prompt or using its painted-checkerboard output as the next input.
This is a practical starting recipe, **not an isolated causal finding** about
negative wording, reference count, or context.

### Retry and method-change boundary

On an alpha failure, inspect the result and current capabilities before another
call. Allow **at most one justified retry** with a named change, such as removing
an unnecessary opaque reference or replacing an overlong prompt with the concise
request. Repeating the same transparency demand is not a new hypothesis. A
second alpha failure ends that route; report the evidence and select an already
authorized alternative or obtain the missing method choice. Handle service
errors according to their error codes rather than treating them as alpha data.

Use existing session authorization. A request for a transparent asset does not
by itself select paid API/CLI generation. Apply the active skill's authorization
boundary before an API/model switch or local cleanup. Local background removal
can deliver a valid final PNG when authorized, but record it as postprocessing,
not direct transparent generation. AGENTS guidance adds neither tool parameters
nor account access.

## Reproducible raw-alpha gate

Run with a Python interpreter that has Pillow. This reads the original file and
converts pixels only in memory for inspection; it never rewrites that file.
The corner requirement assumes the requested padded cutout has empty corners.

```bash
python3 - /absolute/path/to/raw-output.png <<'PY'
from pathlib import Path
from PIL import Image
import hashlib
import json
import sys

path = Path(sys.argv[1])
raw = path.read_bytes()
with Image.open(path) as source:
    source.load()
    encoded_alpha = "A" in source.getbands() or "transparency" in source.info
    alpha = source.convert("RGBA").getchannel("A")
    width, height = source.size
    histogram = alpha.histogram()
    corners = [alpha.getpixel(p) for p in
               [(0, 0), (width-1, 0), (0, height-1), (width-1, height-1)]]
    report = {
        "format": source.format, "mode": source.mode,
        "dimensions": [width, height], "bytes": len(raw),
        "png_color_type": raw[25] if source.format == "PNG" else None,
        "encoded_transparency": encoded_alpha,
        "alpha_extrema": list(alpha.getextrema()),
        "alpha_values": sum(count > 0 for count in histogram),
        "transparent_pixels": histogram[0],
        "opaque_pixels": histogram[255],
        "partial_pixels": sum(histogram[1:255]),
        "near_opaque_pixels": sum(histogram[240:]),
        "corner_alpha": corners,
        "sha256": hashlib.sha256(raw).hexdigest(),
    }
print(json.dumps(report, indent=2))
passed = (report["format"] == "PNG" and encoded_alpha
          and histogram[0] > 0 and sum(histogram[240:]) > 0
          and all(value == 0 for value in corners))
raise SystemExit(0 if passed else 1)
PY
```

This is a preliminary numeric gate, not visual acceptance. Inspect meaningful
solid-subject regions as well as edges; do not accept a nearly invisible subject
just because it has alpha. Do not require an exact alpha-255 pixel: B2's useful
foreground had alpha around 252–253 and its maximum was 254. Generate light/dark
QA composites as separate files, preserving the audited original.

## Measured evidence

All image results below were inspected before local modification. PNG type 2 is
RGB; the failed files had no `tRNS` transparency chunk. Type 6 is RGBA.

| ID | Request | Raw dimensions | PNG type / mode | Alpha range | Alpha-zero pixels | Corner alpha | Result |
|---|---|---|---|---|---|---|---|
| A1 | Original detailed Trane scene, three identity/style references | 1536 × 1024 | 2 / RGB | 255–255 | 0 | 255, 255, 255, 255 | Opaque painted checkerboard |
| A2 | Edit A1, demand background extraction and smaller copy | 1536 × 1024 | 2 / RGB | 255–255 | 0 | 255, 255, 255, 255 | Opaque painted checkerboard |
| A3 | Edit A2, short background-removal request | 1536 × 1024 | 2 / RGB | 255–255 | 0 | 255, 255, 255, 255 | Opaque painted checkerboard |
| B1 | Minimal sphere, no reference | 1261 × 1247 | 6 / RGBA | 0–255 | 1,506,113 | 0, 0, 0, 0 | Direct alpha and visual QA passed |
| B2 | Furry retriever, silver toy, cyan beam/sparkle, no reference | 1536 × 1024 | 6 / RGBA | 0–254 | 837,532 | 0, 0, 0, 0 | Direct alpha and visual QA passed |
| B3 | Compact two-Trane scene with genuine-alpha identity reference | No output | — | — | — | — | Service error; alpha unverified |
| B4 | Simplified single Trane greeting with a tiny gold sparkle and genuine-alpha identity reference | No output | — | — | — | — | Service error; no image to audit |
| B5 | Neutral king, queen and pawn control, no reference | 1536 × 1024 | 6 / RGBA | 0–254 | 1,105,158 | 0, 0, 0, 0 | Direct alpha and visual QA passed |
| B6 | Neutral sitting Trane with one genuine-alpha identity/style reference | 1142 × 1377 | 6 / RGBA | 0–255 | 723,978 | 0, 0, 0, 0 | Direct reference-based alpha and visual QA passed |

B1 was 116,016 bytes, with 252 distinct alpha values, 117 fully opaque pixels and
66,237 partially transparent pixels. Its nonzero-alpha median was 253; 62,989
pixels had alpha at least 240. The sphere remained solid on both backgrounds.

B2 was 2,045,241 bytes, with 255 distinct alpha values and 735,332 partially
transparent pixels. Sampled interior head and torso regions had median alpha
252; their ranges were 251–254 and 250–253 respectively. Fur, facial details and
the silver toy remained visible, and the cyan effect blended into both QA
backgrounds. It is a generic retriever test, not approved replacement Trane art.

B3 returned HTTP 400, `moderation_blocked`, at the output stage, category `other`,
request ID `b3721300-c8f2-489b-8b37-2179ccf9b097`. It was not retried. This yields
no evidence about reference-based alpha behavior. The user then requested that
generation stop and the gathered evidence be documented.

B4 was a later, newly authorized verification of a materially simplified benign
scene: one Trane sitting and raising a paw in greeting, with a tiny gold sparkle.
It used the existing `TaneWithFrame.webp` as an explicitly labelled identity and
style reference. The raw reference was rechecked as RGBA with alpha 0–255 and
539,364 fully transparent pixels, and inspected with `view_image`.

The image service returned HTTP 400, `moderation_blocked`, at the output stage,
category `other`, request ID `df161309-7894-408c-b6e9-6b9b9552e723`. No file was
returned and no retry or local cleanup followed. This is a service failure,
not proof that reference-based alpha is unsupported. At that point the requested
reference-based verification was incomplete. No matching project task-tracking
entry existed; no unrelated task was changed or new ticket created.

Exact B4 prompt, with `referenced_image_paths` set to
`/Users/ronin/Projects/picfetch/assets/trane/TaneWithFrame.webp`:

```text
Identity and illustration-style reference: the attached image is Trane. Create a new storybook illustration of the same cheerful golden retriever sitting with one front paw raised in a friendly greeting, wearing his red collar and plain round gold tag. The foreground consists of the complete Trane figure and a single tiny soft golden sparkle beside the raised paw, with comfortable edge padding. Genuinely transparent PNG with an alpha channel; all empty pixels have alpha 0. Preserve fine fur edges and a solid, near-opaque character.
```

B5 was the user's separately authorized neutral control after B4: exactly one
no-reference request for traditional wooden chess pieces. The untouched returned
PNG was 1,961,474 bytes, with 255 distinct alpha values, 467,706 partially
transparent pixels and 431,577 pixels at alpha 240 or above. Interior samples
from the king, queen and pawn each ranged from 252 to 254, with median 253; every
sampled pixel was at least 240. The documented gate and light/dark visual QA
passed, with solid recognizable pieces, clean edges and clear margins.
No retry, API switch or local background removal was used. This shows that this
neutral no-reference request succeeded at that time; it does not isolate the
Trane reference as the cause of B3/B4 or prove that every service route works.

B6 was a subsequent, separately authorized test of neutral sitting Trane with
exactly one inspected reference, `assets/trane/trane_wags.webp`. The reference
was RGBA, 1142 × 1377, alpha 0–255, with 718,206 alpha-zero pixels and transparent
corners. The request preserved its blue collar and plain gold tag and omitted
props, action and effects. One built-in call returned an untouched 1,547,709-byte
RGBA PNG with all 256 alpha values, 723,978 alpha-zero pixels, 14 fully opaque
pixels, 848,542 partial-alpha pixels and 780,849 pixels at alpha 240 or above.
Interior head, torso and front-paw samples had median alpha 253, ranges 252–254,
252–253 and 252–253, respectively; every sampled pixel was at least 240.

The raw gate passed. Separate white/dark composites showed a recognizable,
solid, full-body Trane and smoothly blending fur edges, without an opaque
background. Top/left padding is tight: the nonzero-alpha bounding box starts at
(0, 0), including faint edge pixels, so this is not proof of ideal padding.
No retry, local background removal, resizing or re-encoding of the raw output
was used. This completes the scoped reference-based native-alpha verification
for this neutral pose; it does not explain B3/B4 or establish reliability for
other scenes. The returned reference dimensions and similar pose do not reveal
how the service generated or processed the image internally.

### Exact successful prompts

B1, `prompt` only; both reference options omitted:

```text
A single small orange sphere centered on a genuinely transparent background. Return a PNG with an alpha channel; all empty pixels have alpha 0.
```

B2, `prompt` only; both reference options omitted:

```text
A cheerful golden retriever puppy with fine fluffy fur, a red collar and plain round gold tag, holding a small silver-and-turquoise toy that emits a narrow cyan beam toward a small four-point sparkle. Complete figure, compact landscape storybook illustration. Genuinely transparent PNG with an alpha channel; all empty pixels have alpha 0. Preserve softly translucent fur edges and localized cyan glow.
```

B5, `prompt` only; both reference options omitted:

```text
Three traditional polished wooden chess pieces: a king, a queen and a pawn, standing side by side, fully visible with clear margins. Genuinely transparent PNG with an alpha channel; all empty pixels have alpha 0. Keep the pieces solid and near-opaque with clean edges.
```

#### B6: Neutral Trane reference

B6, with `referenced_image_paths` set to
`/Users/ronin/Projects/picfetch/assets/trane/trane_wags.webp` and
`num_last_images_to_include` omitted:

```text
Use the attached image as Trane’s identity and illustration-style reference. Draw the same cheerful golden retriever sitting naturally, complete full body with clear margins, wearing his blue collar and plain round gold tag. Trane alone, with no props or effects. Genuinely transparent PNG with an alpha channel; all empty pixels have alpha 0. Keep the character solid and near-opaque with soft fur edges.
```

### Retained raw files and provenance

The raw files remain under
`/Users/ronin/.codex/generated_images/01a09a97-ec95-7200-9f15-d4e939b6dae0/`:

| ID | Raw filename | SHA-256 for successful tests |
|---|---|---|
| A1 | `exec-864b234d-a9de-4ea4-8315-10e7abafe273.png` | — |
| A2 | `exec-16df2415-f45f-4b21-9e83-d8120ec34bd1.png` | — |
| A3 | `exec-ec7e3ae5-95d8-4255-a924-b85ebcc45e65.png` | — |
| B1 | `exec-3111ec81-1231-430c-8679-e5feb36f6079.png` | `b0603c65f44d69727a78b1a7b3e0083fb9b5fbd8fc25c39ad6914cdb4a79c761` |
| B2 | `exec-668409f2-a13b-47fa-a301-ad018168ec89.png` | `129982303ab9af25184bacb085075f1c2eb6ae868f144c4ddd079d0025d95fa0` |
| B5 | `exec-dd490a98-707f-4852-9111-9f433a7407e7.png` | `9f5920089edc34dd814dc04aca328d7ce7ef23cbd0a7aa4a6779c67fdb11d16d` |
| B6 | `exec-61b0683c-5e9d-44b5-a415-10edfcee6734.png` | `e74e3f653368f50849c829f9287e9ce49bb2a699ec6f8010ce491d838c98f27c` |

Original A1–A3 prompts, reference roles and the later authorized local cleanup
are recorded in
`/Users/ronin/Documents/Codex/2026-09-13/realtime-voice-chat/outputs/trane-shrink-ray-generation-details.md`.
The byte-identical user-visible B5 PNG is
`/Users/ronin/Documents/Codex/2026-09-13/realtime-voice-chat/outputs/chess-pieces-transparency-control.png`.
The byte-identical user-visible B6 PNG is
`/Users/ronin/Documents/Codex/2026-09-13/realtime-voice-chat/outputs/trane-neutral-transparency-control.png`.
Byte-identical B1/B2/B5/B6 raw copies, audit JSON (including B6's reference audit),
the B3/B4 request/error records, the observed tool schema and separate QA
composites are retained under
`/Users/ronin/Documents/Codex/2026-09-13/realtime-voice-chat/work/transparent-png/evidence/`.

The original files and B1 contain C2PA metadata naming software agent
`gpt-image`, version `2.0`. This is reported embedded metadata, not independent
verification of the exact API model/snapshot, settings or hidden service
pipeline. The built-in schema exposes no model/background switch. Public
product documentation also identifies built-in generation as GPT Image 2.
[Product image-generation documentation](https://learn.chatgpt.com/docs/image-generation)

## Official API support and the installed-skill mismatch

The official 2026-08-20 changelog introduced transparent output in preview for
`gpt-image-2` and `gpt-image-2-2026-04-21`, through the Images API and Responses
image-generation tool. It requires `background: "transparent"` and PNG or WebP.
The installed skill's blanket claim that GPT Image 2 cannot produce API
transparency is therefore outdated.
[API changelog](https://developers.openai.com/api/docs/changelog)

Current API documentation also supports transparency for GPT Image 2.5 Sunburst
and Flare. A documentation-supported request body is below. It was **not run**
in this investigation; account access and billing authorization were not tested.
[Create image reference](https://developers.openai.com/api/reference/resources/images/methods/generate)

```json
{
  "model": "gpt-image-2.5-sunburst-2026-09-08",
  "prompt": "A single small orange sphere centered on a genuinely transparent background.",
  "background": "transparent",
  "output_format": "png",
  "quality": "medium",
  "size": "1024x1024",
  "n": 1
}
```

For an authorized API route, decode `data[0].b64_json` directly to a raw file and
apply the same validation gate. PNG supports alpha; JPEG does not. In the
Responses API, image background belongs inside the image-generation tool
configuration, not the top-level asynchronous `background` setting.
[Image generation guide](https://developers.openai.com/api/docs/guides/image-generation),
[Responses image-generation guide](https://developers.openai.com/api/docs/guides/tools-image-generation)

The local imagegen skill and `references/image-api.md` repeat the older API
limitation. Its `scripts/image_gen.py` also rejects transparent output for the
exact `gpt-image-2` name in `_validate_model_specific_options`. The script was
inspected, not changed or bypassed. Neither the installed skill nor system
configuration was modified. Consult current official support and actual tool
controls before selecting a future API route; a prompt-only wrapper and the
Images API are different interfaces.

## Scope and conclusion

Four direct built-in outputs proved useful real transparency without local
matting. The concise first-request recipe plus immediate raw-alpha validation
is the practical workflow to reuse. This small investigation establishes
capability, not a first-attempt success rate or a guarantee for reference-based
Trane scenes. B3/B4 returned no image, while the later B5 no-reference control
and B6 neutral reference-based test passed. B6 completes the scoped
reference-based native-alpha verification; more complex scenes remain
unverified and the earlier service errors remain unexplained. The finished
release illustration and existing mascot art were left unchanged during this
investigation. Only this document and the conditional AGENTS.md pointer are
project changes for the follow-up.
