# Keeping the previous HEIC WASM decoder

Research date: 2026-09-14. This records options, not an implementation or a
release qualification. The inspected checkout was `2d9a7cb`, with
`github.com/gen2brain/heic v0.7.1` replaced by
`github.com/frathe/heic v0.0.0-20260820164529-0ac0a39f8206`.

Subsequent decision: Ronin requested removal of HEIC support pending
distribution qualification. These options remain research only; see the
[removal record](../../plans/2026-09-14-remove-heic-decoder.md).

## What the previous dependency actually contains

The pinned wrapper's `lib/Cargo.lock` selects Rust `heic 0.1.6`, checksum
`4922930855bf6756c7a9caf637c6bca37f851927d4c89a0117beb254614f79a3`.
Its corresponding [upstream manifest](https://github.com/imazen/heic/blob/b863223c05406785f1c82e670eb495873fb9af4d/Cargo.toml)
declares `AGPL-3.0-only OR LicenseRef-Imazen-Commercial`.
The wrapper's MIT license does not grant MIT rights to that payload.
See the existing [dependency qualification](dependency-qualification.md).

The package is not unconditionally WASM: it preferentially loads installed
native `libheif`. Upstream documents `nodynamic` to disable that route and
`wasm2go` to transpile the guest into Go. A sandboxed configuration must use
wazero execution and disable native loading, without the `wasm2go` tag.
[Wrapper documentation](https://github.com/gen2brain/heic/tree/v0.7.1)

## Licensing routes

### Keep this Rust payload and obtain suitable additional permission

The exact crate's [commercial license file](https://github.com/imazen/heic/blob/b863223c05406785f1c82e670eb495873fb9af4d/LICENSE-COMMERCIAL)
advertises a $1 open-source key after evidence of published AGPL-compatible
source. It incorporates the Site-wide Subscription License v1.1 or later;
the advertised key is not itself an unrestricted redistribution grant.

The [site-wide terms](https://www.imazen.io/legal/subscription/site) distinguish
packaged/OEM redistribution from bespoke client software. The separate
[redistributor agreement](https://www.imazen.io/legal/subscription/redist)
requires a Scope Agreement identifying the work and permitted products, and
is subscription-bound. The [recipient terms](https://www.imazen.io/legal/subscription/recipient)
restrict modification and redistribution and depend on the provider's rights.
Their application to this crate's discounted open-source offer remains unclear.

Before relying on this route, obtain written terms covering the exact crate
and WASM build, PicFetch's public binaries and distribution channels, source
and binary redistribution by downstream maintainers, security modifications,
offline operation, and continued use of already distributed releases.
Confirm all compiled dependencies and required notice delivery. PicFetch's
own source can remain MIT under a suitable grant; the decoder would retain
its separate license. No contact or purchase was made in this research.

### Use the AGPL grant

AGPL permits distribution subject to its conditions. Sections 4–6 require
notices, license delivery and a compliant way to provide Corresponding Source,
including the required build scripts. Section 5 applies AGPL to a covered
combined work as a whole; it also distinguishes independent aggregation.
MIT source can remain available separately under MIT, but an AGPL-covered
combined distribution cannot be represented as MIT-only. A public GitHub
repository alone does not establish compliance for the actual shipped binary.
The desktop/offline setting does not remove ordinary distribution obligations.
[AGPL text, especially sections 1 and 4–6](https://spdx.org/licenses/AGPL-3.0-only.html)

Whether a proposed arrangement is an independent aggregate needs a licensing
assessment. A WASM sandbox or subprocess is a security boundary, not automatic
proof of legal separation. This route requires review of the complete shipped
dependency set and distribution terms, not just changing the root LICENSE.

### Retain WASM using the earlier LGPL decoder family

There is also a different historical backend: the
[v0.4.0 recipe](https://github.com/gen2brain/heic/blob/v0.4.0/lib/Makefile)
builds `libheif v1.18.2` and `libde265 v1.0.15` into WASM. Their library license
files identify LGPL terms:
[libheif](https://github.com/strukturag/libheif/blob/v1.18.2/COPYING),
[libde265](https://github.com/strukturag/libde265/blob/v1.0.15/COPYING).

This is a possible backend project, not a recommendation to ship those old
versions. It needs supported, security-reviewed pins, a rebuilt guest and
complete license inventory, including source/build delivery and the applicable
ability to modify and relink the LGPL components. It replaces the Rust payload
while preserving the WASM approach. No such build was qualified here.

## Security work if the previous Rust decoder is retained

The inspected fork's `decode_wazero.go` creates `wazero.NewRuntime` at line 204
without an explicit memory cap and uses `context.Background` for calls at
lines 54 and 123. Its `heic.go` parses sequences in Go before calling WASM,
and still-image `Decode` calls `DecodeAll` for sequences. Metadata and Go
container parsing are therefore outside the guest's protection. These are
source observations, not claims of a newly reproduced exploit.

The proposed engineering work is:

1. Adapt the disposable worker from commit `fc127b4` to the Rust guest's ABI.
   Keep all HEIC probing, metadata, container handling and pixel decoding in
   the worker, including generic image-registration and thumbnail paths.
2. Enforce WASM-only execution. Give the guest no filesystem, environment or
   network capabilities. Configure a measured hard guest-memory ceiling and
   interruptible calls; keep the parent deadline, worker kill/join and bounded
   input, output, diagnostics and concurrency. Wazero exposes the required
   [runtime controls](https://pkg.go.dev/github.com/tetratelabs/wazero@v1.12.0#RuntimeConfig).
3. Bound Go-side counts, allocations, dimension products and sequences before
   allocating. Decode only the requested still frame. Validate every worker
   result in the parent and reject malformed HEIC without a parser fallback.
4. Complete OS restrictions and total-process memory controls for the worker.
   Guest memory limits alone do not cap Go-side allocations or the host runtime.
5. Pin and reproduce the guest build, scan both Go and Rust dependencies, and
   run malformed-input, cancellation, memory/CPU, concurrency and capability
   tests. Fuzz the wrapper and guest and exercise packaged applications on
   Windows, macOS and native Linux/amd64. Retest image fidelity and metadata.

The worker design and useful tests can be reused; the different guest ABI and
parser placement need implementation and renewed validation. Choosing Rust or
WASM does not by itself establish that a particular decoder build is safe.

## Recommendation and evidence scope

First seek clear redistribution terms for the exact Rust decoder if retaining
it is the preference. In parallel, the worker adaptation can be designed without
committing to a license purchase. If acceptable terms are unavailable, compare
AGPL distribution with a maintained LGPL WASM backend.

This research changed no application code and ran no decoder qualification
tests. One bounded, read-only scout gathered independent upstream licensing
and historical source facts; the lead inspected local integration and runtime
code, fetched the decisive license sources and authored this record. No
implementation task or review was delegated.
