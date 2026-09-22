# AVIF payload notices

```sh
make check-avif-notices
make generate-avif-notices
go test -race ./scripts/avifnotices ./scripts/updaternotices
```

The checker resolves `github.com/gen2brain/avif` and `github.com/tetratelabs/wazero`
from the current module graph, rejects replacements/version drift, and verifies
the reviewed source/payload hashes in `manifest.json`. It compares the generated
AVIF section with `THIRD-PARTY-NOTICES.md`. Generation is offline once the pinned
Go modules are cached; it reads retained notice sources under `licenses/` and
changes only the bounded AVIF section. It never fetches license texts.

The original file is embedded by `main.go`, passed through `ui.Run`, and shown
as Markdown by Help -> Licenses. Both languages localize the window/menu title;
the license texts stay verbatim. All release packages also include the original
Markdown file. The existing [artifact checker](../updaternotices/README.md) checks
the loose document and its complete embedded bytes in each executable.

## Reviewed sources and obligations

The manifest records each retained file's complete-text hash and upstream URL.
License texts and required copyright/attribution statements are reproduced in
full, including the AOM Patent License 1.0 and libyuv's separate patent grant.
The libyuv PATENTS copy adds only a final newline; its upstream byte hash before
that addition is `4da84211f947c6f9f7dd8fbce513e1522e01833407ae732b8c9a2bbb75fed495`.
The five musl/dlmalloc in-source notices are exact excerpts at the linked line
ranges. Their complete applicable terms are retained: musl MIT/Sun and qsort MIT,
the WASI MIT text for Arm's SPDX-MIT declarations, and CC0 for dlmalloc.

The AVIF module's Makefile pins libavif 1.4.2, libaom 3.14.1 and dav1d 1.5.3.
The WASM's strings independently identify AOM 3.14.1 and dav1d 1.5.3. The full
libavif license aggregate is fetched from its pinned source, because the older
`lib/LICENSE.libavif` inside the Go module is incomplete for that revision.
Both encoder and decoder code are in the shipped WASM, so the notices also cover
libaom's embedded fastfeat (BSD) and vector (MIT) helpers. The build does not
enable libaom's Highway component.

WASI source paths identify `wasisdk://v33.0+m`, and the producer section names
LLVM commit `4434dabb69916856b824f68a64b029c67175e532`. The SDK 33 reference pins
wasi-libc `161b3195fc2558d2b1ba3eb9ffae3b2b47407623`; its musl, cloudlibc, dlmalloc
and compiler-rt notices are included, along with every top-level WASI license
alternative. Payload source paths identify the additional Sun/Arm/qsort notices.
The payload contains compiler-rt's `__multi3`. No emmalloc or musl-fts code was
identified. Wazero's Apache-2.0 LICENSE and NOTICE cover the Go runtime/WASI host.

BSD/MIT attribution and disclaimer delivery is provided by the full embedded
and loose document. Apache texts include their applicable NOTICE/LLVM exceptions.
Patent grants are reproduced without changing or independently evaluating their
terms. The full upstream aggregates also retain optional-source notices; that
does not assert that every upstream tool or example is linked into PicFetch.

## Provenance limits

Libyuv is built from a floating `stable` branch, not a commit recorded by the
AVIF module. That branch resolved to `eb6e7bb63738e29efd82ea3cf2a115238a89fa51`
on 2026-09-21. The retained license/patent/author files use that exact reference,
but it does not prove the historical checkout used for the payload. Likewise,
SDK `33.0+m` identifies a modified SDK build; the module supplies no builder
attestation establishing its exact libc checkout or documenting local changes.
Those remain upstream provenance questions before claiming fully qualified
release-source correspondence. The checker binds the exact existing payload;
passing it does not resolve these historical unknowns.

## Upgrade and release checks

1. Review the new Go module, selected WASM/runtime path, build recipe and complete
   linked source closure, including vendored helpers and runtime components.
2. Obtain exact source revisions or document unresolved provenance. Preserve
   original LICENSE, COPYING, NOTICE, PATENTS, attribution and applicable source
   headers in `licenses/`. Update the manifest's versions and hashes after review.
3. Run `make generate-avif-notices`; inspect the complete document diff, then
   `make check-avif-notices`. Update source evidence and limitations here.
4. Run focused notice/viewer tests and `make verify`. The complete suite requires
   a native Linux/amd64 daemon or its native CI runner.
5. On the actual release checkout, run the artifact checker on all six final
   GitHub archives and both Store payloads. The release/Store workflows already
   do this before publishing/uploading. Verify Help -> Licenses offline on native
   platforms; cross-builds and synthetic archive tests are separate evidence.

AVIF, ONNX and other bundled native/WASM dependency changes must carry their
notice updates in the same change. ONNX runtime package/license selection stays
in `internal/similarity/assets.go` and `assets_package.go`; preserve Microsoft's
LICENSE, ThirdPartyNotices and privacy documents when staging those runtimes.
