# SigLIP 2 model mirror: redistribution preflight

Checked 2026-09-26. Scope: mirror the exact existing vision model and processor
configuration as PicFetch-owned GitHub release assets, not change the model,
bundle it in installers, or publish an application release. This is a technical
compliance assessment, not a legal opinion.

## Result

Google explicitly licenses the original model under Apache-2.0. That licence
permits redistribution subject to its conditions. The exact community ONNX
export links to that original model but does not declare its own licence.
That omission is not proof redistribution is prohibited; it prevents an
unqualified statement that the publisher has expressly confirmed terms for
every file in this export. Public mirror publication is pending clarification.
No model assets have been uploaded, download URLs changed, or model tests altered.

## Exact artifacts already used by PicFetch

Export revision: `ba1f3b0843f24bc5417d38e19c37b287d719b2f4`.
The same revision is still the export repository's current head.

| File | Bytes | SHA-256 |
| --- | ---: | --- |
| `onnx/vision_model.onnx` | 371807752 | `c0573e3f4140c3a7c4e9cc5912bd6b26a033b46a6a8e8af26cbea262b163bcad` |
| `preprocessor_config.json` | 394 | `9b36b57ebaf20f09bf4c22100ccc21877ea6bfe5aead0c00c59f8af8ccefacfc` |

These are the existing pins in `internal/similarity/assets_install.go`; this
preflight does not claim to have downloaded/rehashed another copy of the weights.

## Primary-source evidence

- Google's original model card explicitly declares `apache-2.0`. The inspected
  card revision is `75de2d55ec2d0b4efc50b3e9ad70dba96a7b2fa2`; it is evidence of
  Google's licence declaration, not a claim that the converter used that exact
  source revision. [Pinned original card](https://huggingface.co/google/siglip2-base-patch16-224/blob/75de2d55ec2d0b4efc50b3e9ad70dba96a7b2fa2/README.md).
- Apache-2.0 sections 2 and 4 permit distribution of source/object forms and
  derivatives. Recipients must receive the licence; required attribution and
  applicable upstream NOTICE information must be retained, and modifications
  must be identified. Its definition of object form includes mechanical
  conversions, but that alone does not establish whether a particular export
  contains independently contributed material. [Licence text](https://www.apache.org/licenses/LICENSE-2.0).
- The pinned export's complete API inventory has no LICENSE or NOTICE file;
  its card metadata has only `library_name` and `base_model`. Its README
  identifies the original Google model and explains that it supplies ONNX
  weights for Transformers.js. Direct HTTPS reads verified both API and raw
  README because the browser reader could not open that pinned card.
  [Pinned API](https://huggingface.co/api/models/onnx-community/siglip2-base-patch16-224-ONNX/revision/ba1f3b0843f24bc5417d38e19c37b287d719b2f4),
  [pinned README](https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/raw/ba1f3b0843f24bc5417d38e19c37b287d719b2f4/README.md).
- The converter's maintainer has corrected missing licence metadata for other
  named exports and said those licences match their original models. The
  inspected replies concern UAE-Large-V1, whisper-small and bert-base-NER, not
  this SigLIP 2 export; they are evidence of a clarification route, not a
  blanket licensing grant. [December 2025 clarification](https://github.com/huggingface/transformers.js/issues/1487#issuecomment-3661917497),
  [September 2026 clarification](https://github.com/huggingface/transformers.js/issues/1487#issuecomment-5666868185).
- Hugging Face's terms grant public-content rights through its own services
  and preserve applicable open-source terms. This is not treated here as an
  independent, unambiguous off-platform redistribution grant for unlabelled
  conversion contributions. [Terms, Your Content and Intellectual Property](https://huggingface.co/terms-of-service).

## Upstream clarification requested

Ronin authorized posting the clarification request. It was posted in the
maintainer's existing licensing thread on 2026-09-26 at 20:46:23 UTC:
[upstream request](https://github.com/huggingface/transformers.js/issues/1487#issuecomment-5849743595).
The API response confirmed the submitted text and comment ID `5849743595`.
No reply or redistribution clearance is recorded yet. The request asks the
publisher to confirm Apache-2.0 for both pinned files, identify required notices,
and add the missing licence metadata. Its substantive question is:

> PicFetch uses `onnx-community/siglip2-base-patch16-224-ONNX` at revision
> `ba1f3b0843f24bc5417d38e19c37b287d719b2f4`. Google's original model declares
> Apache-2.0, but the export card does not specify a licence. Could you confirm
> that `onnx/vision_model.onnx` and `preprocessor_config.json` at that revision
> may be redistributed unchanged under Apache-2.0, and identify any additional
> required notices? We would like to mirror those exact files in a versioned
> GitHub release, preserving the licence, attribution, provenance and hashes.

An independently produced export of Google's licensed weights is an alternative
if clarification is unavailable. That changes the artifact/provenance and may
change numerical behavior; it needs a separately approved qualification task,
not a silent substitution for the already-tested file.

After clearance: retain the original licence, attribution, exact source/version
and conversion provenance alongside the mirrored data and in the application's
offline notices. Preserve size/SHA-256 validation on every source; fallback must
not turn checksum failures into accepted data. Confirm cancellation, progress,
failed staging cleanup and cache reuse at the existing installation interface.
Keep the model release outside the application updater's latest-release channel
and outside the application's `v*` release/Store triggers. The mirror still
depends on network availability; it is not offline first-use setup.
