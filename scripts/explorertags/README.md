# Regenerate local semantic tag vectors

Use the runtime installed for the host architecture. The example below uses
Apple Silicon; on Intel macOS the runtime path ends in
`onnxruntime-osx-x86_64-1.23.2/lib/libonnxruntime.1.23.2.dylib`.
The tool selects the matching API binding through `internal/ort`.

The application embeds 75 fixed subject/scene text vectors as readable JSON
(about 997 KiB of text, representing 230,400 bytes of float32 values).
The text model is only used for regeneration. Every fresh or reused image
representation is compared locally with all vectors. Sigmoid scores are normalized to shares of the fixed
catalogue's total score. A strongest share below 0.35, or strongest raw score
below 0.00001, leaves the source Untagged. Otherwise every share of at least
0.10 qualifies. This preserves multiple substantial matches without forcing a
label onto an ambiguous score distribution. These are trial heuristics, not
calibrated confidence; changing the catalogue requires reevaluation. Model output
does not add labels outside this catalogue or infer traits for saved presets.

`internal/similarity/tag-catalog.json` owns the prompt order, model revision,
threshold, scoring constants and vector digest. `tag-vectors.json` maps each
tag ID to its L2-normalized array of 768 decimal float32 values. The decimal
values round-trip to the exact original float32 bits. `NewTagger` decodes the
embedded JSON directly; ordinary builds need no generator or binary asset.
The digest identifies the numeric values encoded as little-endian float32 in
catalogue order, so whitespace and JSON key order do not change vector identity.
`internal/ui/explorer/tags.go` owns localized labels; update both translations
when changing the catalogue. Image caches retain representations, not labels, so catalogue changes reuse
image analysis and recompute tags. Preset catalogue identity changes require
the existing review-and-save compatibility flow before applying old definitions.

The expanded catalogue includes clothing, transport, landmarks, nature, objects
and activities. Food uses a food-or-beverages prompt so it retains coffee
alongside the more specific Drink tag. The 0.10 secondary-share threshold
retains overlapping subjects such as Costume and Festival; the strongest-share
threshold remains 0.35. Neither threshold is a calibrated probability.

## Reproduce on Apple Silicon macOS

Run `make explorer-setup` for the existing vision/runtime assets first.
Python and the text model are development dependencies only. The fixed
catalogue contains no user images, filenames or text.

```sh
set -e
tag_work=.scratch/visual-similarity-explorer/tag-generation
mkdir -p "$tag_work"
python3 -m venv "$tag_work/venv"
"$tag_work/venv/bin/pip" install sentencepiece==0.2.1
model_base=https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/resolve/ba1f3b0843f24bc5417d38e19c37b287d719b2f4
curl -fL "$model_base/onnx/text_model.onnx" -o "$tag_work/text_model.onnx"
curl -fL "$model_base/tokenizer.model" -o "$tag_work/tokenizer.model"
"$tag_work/venv/bin/python" scripts/explorertags/prepare_tokens.py \
  "$tag_work/tokenizer.model" internal/similarity/tag-catalog.json > "$tag_work/tokens.json"
go run ./scripts/explorertags \
  -model "$tag_work/text_model.onnx" \
  -runtime .scratch/visual-similarity-explorer/assets/onnxruntime-osx-arm64-1.29.0/lib/libonnxruntime.1.29.0.dylib \
  -tokens "$tag_work/tokens.json" \
  -catalog internal/similarity/tag-catalog.json -out "$tag_work/tag-vectors.json"
cmp "$tag_work/tag-vectors.json" internal/similarity/tag-vectors.json
```

Both tools verify their pinned model/tokenizer checksums before processing.
When intentionally changing prompts, copy the newly generated vectors into
`internal/similarity`, update `vectorsSHA256` using the generator's reported
numeric digest and the catalogue version, and run
`make explorer-ui-test`, `make explorer-test`, then `make verify`.

## Primary provenance

- [Pinned ONNX export](https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/tree/ba1f3b0843f24bc5417d38e19c37b287d719b2f4): text tower SHA256
  `baf12d941beabafafb14f7b4adb38dc15be18681b964a84410ec53d9d65e6293`,
  tokenizer SHA256 `61a7b147390c64585d6c3543dd6fc636906c9af3865a5548f27f31aee1d4c8e2`.
- [SigLIP2 preprocessing and scoring](https://huggingface.co/docs/transformers/model_doc/siglip2#text-embeddings-and-retrieval): lowercase prompts and a fixed 64-token input.
- [Pinned Gemma settings](https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/resolve/ba1f3b0843f24bc5417d38e19c37b287d719b2f4/tokenizer_config.json): no BOS, one EOS (1), right PAD (0). The SentencePiece model performs identity normalization without adding a prefix space.
- [Tokenizer source](https://github.com/huggingface/transformers/blob/v4.50.0/src/transformers/models/gemma/tokenization_gemma.py): SentencePiece encoding and EOS handling.
- The pinned combined `onnx/model.onnx` contains `exp(logit_scale)` =
  `112.66889953613281`, bias = `-16.771724700927734`. These were verified from
  bytes 1501100918–1501100972: initializer float32 bytes `7a56e142` and `7e2c86c1`.
  Scores are `sigmoid(scale * cosine + bias)`. The standalone text tower omits
  these constants; it returns a projected `pooler_output` of shape `[1,768]`.

The export is distributed under Apache-2.0; these generated vectors derive from
that model. Public fixtures have separate attribution in
`internal/ui/testdata/explorer/README.md`. Known cat, portrait, coffee, carnival
costume, train and three blank cases guard specific labels, overlapping labels,
ambiguity and cache reuse. They do not qualify library-wide semantic accuracy.
