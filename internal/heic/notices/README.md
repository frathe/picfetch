# Optional Linux HEIC binding provenance

`../libheif_abi.h` adapts only public C ABI declarations from
[libheif v1.17.6, `libheif/heif.h`](https://github.com/strukturag/libheif/blob/v1.17.6/libheif/heif.h),
copyright 2017–2023 Dirk Farin, licensed LGPL-3.0-or-later. It contains no decoder
implementation. The exact source attribution is retained in that header;
`LGPL-3.txt` and its incorporated `GPL-3.txt` are retained here. These license
texts came from Ubuntu's `/usr/share/common-licenses/{LGPL-3,GPL-3}`. They must be
included by PicFetch's shipped third-party notice delivery, with this attribution.

The adapter is authored in this repository. It uses opaque pointers and the
selected public layouts rather than requiring development headers or a mandatory
`libheif` link. The child dynamically opens the installed `libheif.so.1` and
resolves every required function. No libheif, libde265, FFmpeg, x265 or other codec
binary is copied into PicFetch by this change. Dynamic use alone is not a claim
that license obligations disappear: the release plan must retain this source
record and audit the actual package payload and notice delivery.

The native qualification environment and runtime dependency closure are recorded
in `.scratch/os-heic/linux-native-recon.md`. The initial qualified runtime is
Ubuntu libheif `1.17.6-1ubuntu4.8` plus libde265 `1.0.15-1ubuntu0.1` on Linux
x86_64. The runtime accepts symbol-compatible libheif 1.x from 1.17.6; the test
record does not qualify every later library or architecture.

The fixtures are independently authored repository content, covered by the root
MIT license. Their reproducible source and exact generation tools are in
`../testdata/README.md`; they contain no third-party photographs.
