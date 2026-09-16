# PicFetch's maintained HEIC source

## Source selection

This is the exact production source previously maintained by PicFetch at
commit `fc127b44e1c99447b8d150256563c6d4f99b8ad3`, subtree
`df5936bb22bd65eae70be1f6a50237644c5fa999`. `PICFETCH-SOURCE.json` records every
copied production file. The baseline is upstream `github.com/gen2brain/h265`
v0.2.2, commit `665fd95984177afef4a7efca7d50638e4b695c7a`, with the local
hardening retained unchanged. Root and guest modules explicitly replace h265
with this directory. Only the separate WASI guest imports its decoder.

The restoration's earlier unmodified v0.2.3 guest was a development candidate.
That release does not contain all the maintained source's work-budget,
metadata-accounting, bounded NAL traversal, sequence-retention and transformed
configuration changes. Selecting a newer version alone would discard those
changes. This restoration instead preserves the complete maintained production
copy; it does not claim complete equivalence between two different sequence
implementations.

## Retained local changes

- Finite movie sample/track counts, table payload and duplicate checks, deferred
  timing expansion and checked linear sample layout.
- Missing-metadata handling, checked extent arithmetic and aggregate selected
  bytes, finite parsed metadata entries and HEVC configuration units.
- Coded-frame limits for primary images, grids, alpha and tracks, plus one
  shared atomic `DecodeBudget` for aggregate coded work. Finite decoded-picture
  buffering and reordering counts.
- The 32-sample transform limit, originally taken from upstream commit
  `5138267a04330b8f0e36ddeabbcf6f49b664fa88`, and checked NAL length arithmetic.
- Streaming NAL traversal with finite counts, one-picture still items, bounded
  first-frame traversal and finite aggregate retained sequence output.
- `DecodeConfigWithOptions` for container-transformed bounds, explicit
  `AutoRotate` and one decoder thread in the application guest.
- Existing parameter-name annotations for the project's inspections.

The restoration guest additionally rejects movie containers, admits at most
64 MiB input, and applies its independently enforced WASM/IPC/output/deadline
ceilings. Library limits are defense in depth and do not replace that boundary.
The historical negative-limit API remains in the source; PicFetch never uses it.

## Distribution and verification

MIT; retain `LICENSE`, including the roticv and Karpeles Lab notices. Upstream
credits rust_h265 and oxideav-h265; exact translated-source revisions remain a
traceability gap. No HEVC patent clearance is asserted. The source has no module
dependencies, embedded runtime or native codec library. Native assembly is
retained for exact source provenance, but `wasip1/wasm -tags=noasm` selects the
scalar decoder used by PicFetch.

`make generate-heic-wasm` rebuilds the fixed guest and provenance.
`make check-heic-wasm` checks the production copy, replacements, complete guest
inputs, independent reproduction and absence of native decoder imports.
`make test-h265` exercises this nested source through the bounded WASI guest
using the reviewed ordinary fixtures; root `go test ./...` alone does not
traverse a nested module. It is scoped compatibility coverage, not the upstream
or historical full test suite. Historical malformed-input/fuzz seeds and test
files are not copied or executed by this restoration. Their original record
remains in the historical checkout, which is not modified here.

Before upgrades, review every local change against the proposed upstream
source, preserve checks without verified equivalents, update the exact source
record deliberately, reproduce the guest, and repeat bounded ordinary-image and
owned-boundary qualification. Never add an in-process or native codec fallback.
Current native platform, memory and compatibility status is recorded in
`../../docs/heic/qualification.md`; a source copy does not establish activation.

## Inspection disposition for this restoration

All 57 Go and 47 assembly files were inspected with GoLand, including weak
warnings. The 47 assembly reports reject Go's assembler dialect in the generic
assembly parser; those unchanged files are retained for provenance and excluded
from the WASI build. Their exact paths are excluded from Qodana. Native assembly
execution/compilation is not newly qualified by this restoration.

Exact-file inspection exclusions preserve assembly-backed parameter names,
build-specific helpers, public API/format constants and the unused retained
reference helpers/type. Exact duplication exclusions preserve reviewed codec
variants instead of rewriting the maintained algorithms. Remaining suggestions
are redundant conversions and scoped local `any`/`max` names; they do not change
behavior. The partial NAL switch is followed by explicit EOS/non-VCL handling.
Direct EOF comparisons are fed by the bounded in-memory guest reader; other
errors return a refusal. No confirmed new defect was found in these reports.
The IDE does not apply Qodana's scopes, so this is a reviewed disposition, not a
claim that its generic assembly reports disappeared. Fresh Qodana verification
of the configured scopes remains pending.
