# Unreleased build antivirus triage, 2026-09-11

Assessment: **likely false positives; not yet confirmed by the vendors**.
No evidence of compromise was found in the bounded checks below. Detection
counts alone do not establish safety. No application or packaging change is
justified by the reports at this stage.

## Samples and observed results

These are local, unreleased `main` builds, not the public v1.0.3 release.
All four VirusTotal hashes exactly match their corresponding files in `bin/`.

| File | Bytes | VirusTotal result | Detection |
|---|---:|---|---|
| `picfetch-linux-amd64` | 53,538,624 | [1/62](https://www.virustotal.com/gui/file/4bd607911cda715905194b0b914750e9deb3bc0ca895bfa6bcaf4fb26af84de0) | Microsoft: `Trojan:Script/Wacatac.C!ml` |
| `picfetch-linux-arm64` | 50,175,936 | [1/60](https://www.virustotal.com/gui/file/b737f1bb1f4184ddab5787388cb0ddce8342ad0b62eac8fb48f0f3ab7c9ec40a) | Microsoft: `Trojan:Script/Wacatac.C!ml` |
| `picfetch-windows-amd64.exe` | 55,008,768 | [1/68](https://www.virustotal.com/gui/file/acf1f250de03f5b703afb85d04cafcd00ec9a5abf7dc52ace7dda179254bbd76) | Trapmine: `Malicious.high.ml.score` |
| `picfetch-windows-arm64.exe` | 51,394,560 | [0/67](https://www.virustotal.com/gui/file/9f89249e8e911c2ba716561f970b9a6ec73c2393869d114bf83347d299d5d323) | None |

The reports' analysis times were 15:18:11, 15:18:27, 15:18:44 and 15:18:54 UTC,
respectively. Results can change after this record.

Microsoft reports both Windows samples as **undetected**. Trapmine reports
Windows ARM64 as **Unable to process file type**, so its absence there is not
an independent clean verdict on the same code. Unsupported engines and failures
must not be counted as successful scans.

## Integrity and provenance checks

- `shasum -a 256 bin/picfetch-linux-* bin/picfetch-windows-*.exe` reproduced all
  four supplied hashes. Both Linux hashes also match the original outputs in
  `fyne-cross/bin/linux-{amd64,arm64}/picfetch`.
- `go version -m` identifies Go 1.27.1, cgo enabled, `release,migrated_fynedo`,
  `-trimpath`, and source revision
  `12249a6720689b91609e6872f9844282defa1dcf` in every sample.
  Each carries `vcs.modified=true`. Fyne's packaging tool temporarily injects
  `fyne_metadata_init.go` and removes it afterwards; that is a normal possible
  explanation for this marker, not proof of the exact historical working tree.
  The checkout inspected here was clean at `84a4b01`, a subsequent todos update.
- Every embedded dependency version/checksum matches the repository's `go.sum`.
  There are 99 dependencies in each Linux binary and 98 in each Windows binary,
  with 99 distinct modules across the four builds.
- Independently recomputed Go `h1:` hashes for **all 99 cached module source
  directories and ZIP archives** in the fyne-cross cache. All match both the
  embedded build checksums and `go.sum`: **99/99 passed, zero failures**.
  This checks cached content, not just its accompanying `.ziphash` file.
  Raw build metadata, per-module results and the verification script are retained
  locally in ignored `.scratch/antivirus-20260911/`.
- The local Linux Docker image identity matches the repository pin:
  `sha256:7502500e2224dbbc207df49b13c98b9116a6f6967ff3f8ceab6798be75918706`.
  Packaging uses fyne-cross v1.6.3; the pinned packaging CLI is Fyne v1.7.2.
- Static ELF/PE header inspection shows ordinary Go/native executable sections
  and no bytes appended beyond the declared sections/section table in any sample.
  This excludes a simple appended overlay, not arbitrary malicious code.

The initial offline `go mod verify` attempt could not load an uncached module
metadata file for the Intel-macOS-only ONNX binding. The targeted content-hash
check above covers the exact dependency union of the four supplied binaries and
does not require fetching that unrelated binding. It follows the `Hash1`,
`HashDir` and `HashZip` algorithms in `golang.org/x/mod@v0.40.0/sumdb/dirhash`.

## Behavior report

The visible Windows AMD64 CAPE report showed font-cache and PicFetch preference
writes, Fyne application storage reads, theme/OpenGL registry reads, and cleanup
attempts for `.new`, `.old` and `.apply.cmd` update leftovers. The dropped files
were `font_index_v6.cache` and `preferences.json`; the font cache uses gzip in
`github.com/go-text/typesetting@v0.3.4/fontscan/serialize.go`.
Updater cleanup is implemented in `internal/update/await.go`.

The visible summary had no malicious detections, MITRE signatures, IDS/Sigma
matches or network communications. It also showed an `obfuscated` behavior tag;
the report does not establish which content caused that tag or whether it is
related to Trapmine's verdict. Some sandboxes were still analysing the sample.
The Linux AMD64 behavior summary likewise showed no detections and incomplete
sandbox analysis. ARM reports did not expose a Behavior tab.

## Interpretation and remaining work

The isolated, differing scanner verdicts, matching dependency contents and
expected visible startup behavior support the false-positive hypothesis.
The Go project explicitly documents antivirus false positives involving Go
binaries, but that general observation is not a verdict on these samples:
[Go FAQ](https://go.dev/doc/faq#virus).

The proprietary detection trigger remains unknown. These checks do not include
an independent rebuild, a fresh local Defender scan, a complete native/toolchain
audit, or completed sandbox coverage of every feature. Build metadata itself is
not an attestation binding all executable bytes to source.

Next actions:

1. Submit both exact Linux samples to [Microsoft Security Intelligence](https://www.microsoft.com/en-us/wdsi/filesubmission)
   as a **Software developer**, reporting suspected incorrect detections.
   [Microsoft's dispute procedure](https://learn.microsoft.com/en-us/unified-secops/submission-guide#how-do-i-dispute-the-detection-of-my-program)
   calls for waiting for a final determination.
2. Request review of the Windows AMD64 sample from Trapmine. Its
   [official site](https://www.trapmine.com/) lists `fp@trapmine.com` for false-positive reports.
3. Record vendor dispositions and scan the actual final release artifacts after
   the normal CI build and Windows signing. Different bytes need their own scan.

Vendor-review drafts are in `.scratch/antivirus-20260911/vendor-review-drafts.md`.
They have not been submitted. No application binaries were executed during this
investigation, and antivirus settings were not changed.
