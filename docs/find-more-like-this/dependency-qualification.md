# Dependency qualification for Find more like this

September 14, 2026. No module, native runtime, model asset or pin changes are
introduced by this feature. Search reuses the existing offline image pipeline
and assets; advisory locks use the already pinned `golang.org/x/sys v0.48.0`.

## Search assets and notice delivery

- SigLIP2 ONNX revision `ba1f3b0843f24bc5417d38e19c37b287d719b2f4` is the
  existing Apache-2.0 model; attribution is in the root third-party notices.
- ONNX Runtime remains 1.29.0, with the existing Intel Mac exception at 1.23.2.
  The installer retains MIT and bundled third-party notices. Intel's Eigen
  source reference remains `1d8b82b0740839c0de7f1242a3585e3390ff5f33` for the
  MPL component. `onnxruntime_go` remains 1.36.0 / Intel 1.25.0, MIT.
- Existing packaging places notices in macOS Resources, standalone archives
  and MSIX payloads. Signed final Windows payload qualification remains in the
  existing release work. The updater notice gate covers x/sys 0.48.0; the older
  dependency inventory still names 0.47.0 and requires reconciliation.

## Existing closure gaps found during the required audit

These are release qualification issues, not new dependencies selected here.
The feature and PR do not establish that the existing distribution is ready
for release under every bundled license.

The HEIC wrapper remains nominal `github.com/gen2brain/heic v0.7.1`, replaced
by `github.com/frathe/heic v0.0.0-20260820164529-0ac0a39f8206` for the native
memory-leak fix. Its MIT wrapper includes a Rust `heic 0.1.6` WASM payload.
The locked crate checksum is
`4922930855bf6756c7a9caf637c6bca37f851927d4c89a0117beb254614f79a3`; the fetched
crate matched that checksum. Its [upstream manifest](https://github.com/imazen/heic/blob/b863223c05406785f1c82e670eb495873fb9af4d/Cargo.toml)
declares `AGPL-3.0-only OR LicenseRef-Imazen-Commercial`. The repository audit
found neither a commercial grant nor payload-specific AGPL notice/source
delivery evidence. The wrapper's MIT notice does not resolve that evidence gap.
Confirm the applicable grant and distribution obligations before release;
this change does not replace the decoder or claim a legal conclusion.

The unchanged AVIF v0.6.0 WASM recipe includes libavif 1.4.2, AOM 3.14.1,
dav1d 1.5.3 and a libyuv `stable` reference without an exact source pin in the
recipe. Component-level notice delivery needs completion. Fyne v2.8.0 also
bundles NotoSans and InterSymbols under SIL OFL 1.1 and DejaVuSansMono-Powerline
under Bitstream/Arev terms; the root inventory does not yet deliver all of
those font notices. The default `no_emoji` build excludes EmojiOne.

Track remediation in the standing todo list, retaining this exact source and
version evidence. Do not silently substitute assets or weaken decoder support
to close the record.
