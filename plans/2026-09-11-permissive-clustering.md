# Replace GPL clustering

The user explicitly authorized replacing the GPLv3 dependency on 2026-09-11,
while Linux ARM64 support remained active. Route: Deep (algorithm and release
licensing change across grouping, dependencies, notices and test artifacts).
The license should have been checked before the original dependency was adopted;
the earlier packaging-time discovery was too late.

Use the MIT-licensed HDBSCAN subset from PhotoPrism `pkg/vector/alg`, pinned to
`c48d23f6b03c25fc19d376d789fac56c32a26fdb`. Its independent subtree LICENSE was
verified against official source, SHA-256
`2a63bba8b9b3d1c8be384aacc290d0790ce6e97aecfb98930efc1f1cc3042472`.
Keep the exact copyright/license text, source provenance and a record of edits.
Copy only the HDBSCAN dependency closure into `internal/hdbscan`; no import of
the parent PhotoPrism module or its unrelated code. Lead owns all copying,
adaptation, design, review and fixes. One existing read-only scout verified the
license, dependency closure and semantic differences independently.

## Contract and acceptance

- HDBSCAN still clusters the existing deterministic 15D reduction with minimum
  group size four. Preserve all input identities, -1 noise, canonical cohort
  identities and source-versioned embedding caches. Saved explicit cohorts stay
  untouched; newly calculated automatic groups may differ across implementations.
- Characterize old/new behavior using only API calls on deterministic synthetic
  inputs. Resolve whether minimum samples counts the point itself before mapping
  the setting. No GPL source is copied into the replacement.
- Test small input, identical points, separated clusters, varying density,
  noise, malformed coordinates, determinism and context cancellation at the
  owning worker boundary. Reuse useful MIT upstream synthetic tests with notice.
- Run serial bounded synthetic measurements to catch an unacceptable regression;
  do not run the earlier 50k corpus or any heavy VM/container workload here.
- Remove latent from production imports/module requirements; prove with
  `go list -deps` and built-binary module metadata. Remove its shipped notice
  after no distributed code depends on it, replacing it with the MIT attribution.
- Update ARCHITECTURE, Qodana exclusions, notices and release follow-up records.
  Include notices in rebuilt test archives; preserve unrelated source/license
  notices. Rebuild affected local test binaries after the library is replaced.

Verification: focused `internal/hdbscan` and similarity/evaluator tests; ordinary
and Store build/dependency checks; native x64 synthetic worker/setup regression;
Linux ARM64 cross-build/ELF inspection. Native ARM64 hardware tests remain for the
user, full CI/WACK stays separate. No commit, push, publishing, global tool install,
Docker/WSL/VM startup or Windows isolation probe is authorized by this increment.

## Implementation evidence

The copied upstream synthetic tests passed before integration (1.331s), including
separated/variable-density groups, noise, duplicate coordinates, finite scores,
serial/parallel consistency and core-distance convention. Missing implementation
and missing PicFetch adapter were observed as RED before copying/integration.
The focused replacement/adapter/runtime tests pass (1.375s/0.558s/0.508s).

API documentation did not specify whether latent counted self. A targeted read
of only its neighbor-index convention confirmed MinSamples=2 selected index two
after self at index zero. No GPL code was copied. The MIT implementation therefore
uses minPts=3, retaining two other neighbors, with minSize=4 and workers=1.

The API-only synthetic comparison covered 1, 3, 16, 120 and 1,000 points, 15
dimensions, and three fixed seeds. Inputs of size <=16 matched; larger sets
had different memberships. The replacement consistently found the three generated
groups. At 1,000 points it took about 18–19 ms versus the old 50–51 ms in that
bounded run. These are single-run observations, not a full-library performance
claim. Evidence: `.scratch/linux-arm-explorer/clustering-comparison.log`.

Production imports/module requirement and both latent checksum entries are
removed. Its shipped GPL license section is replaced by the exact MIT attribution;
historical plans retain the discovery and remediation history. Standalone release
packaging now carries the license/notice/privacy files, including Windows signing
repacking and the macOS app Resources directory.

Native cancellation/restart testing exposed the upstream exclusion of a single
root cluster: six synthetic images completed but produced no cohort. The adapter
now preserves Explorer's root cohort when no smaller clusters qualify and at
least four distinct sources remain; it retains noise when child clusters exist.
The added four/six-point regression was observed RED before the adapter fix.
This is a PicFetch adapter policy; the upstream clustering files stay unchanged.
A targeted check of the old selection policy confirmed root selection was
allowed; no old implementation was copied.

### Final validation and test artifacts

- Native cgo tests pass: hdbscan (1.451s), similarity (2.464s), evaluator
  (0.181s), and msixstage (1.767s). Store-tagged similarity/msixstage pass
  (2.285s/1.643s). Native ordinary and Store `go vet ./...` pass.
- The real production profile passes (6.290s): cold/warm embedding reuse,
  grouping, failed sources, cancellation and clean worker completion.
- Native Explorer setup, encoded-file limits, small cohorts, repeated-source
  identities, cancellation and restart pass (16.168s). The real known-content
  granularity/hierarchy test passes (5.388s). Existing tests were retained;
  the root-cohort compatibility fix made their original assertions pass.
- Root translation and both manual guards pass (0.210s/0.411s).
- Ordinary and Store `go list -deps` omit latent and the PhotoPrism parent
  module. The vendored package's production closure is standard-library-only.
  `go version -m` confirms the same exclusions in all five final executables.
- The exact upstream MIT license hash matches; its full text is present in
  THIRD-PARTY-NOTICES.md. Both real Store staging directories contain documents
  matching source hashes and architecture-pinned DLLs.
- The standalone signing-notice guard failed when repacking only the executable,
  then passed after restoring the final workflow. Archive verification compares
  each extracted member with its source; Linux executable mode is also checked.
- No new UI top-level test was introduced. All 239 test files have Qodana
  exclusions; goimports verifies 553 Go files with CRLF normalized in memory.
  The full Linux/race gate, canonical Linux shard check, ARM64 hardware runs and
  MSIX/WACK remain explicitly unverified on this host.

| Current test executable | Bytes | SHA-256 |
|---|---:|---|
| `bin/picfetch.exe` | 53,611,008 | `5483ab37f3c8acf934446c115223adaf1766e0200c00d03609ce6abd32ddfa97` |
| `bin/picfetch-windows-arm64.exe` | 50,011,136 | `6ec9fb1a0dc1549d51452534eae37ce0c1056c1e9307270877efa6721d8e209f` |
| `bin/picfetch-microsoft-store-amd64.exe` | 53,595,136 | `edc6b5fe7ffc66f2a443a5380c6c46042d8b9feda28f43d39bbc590d03f88241` |
| `bin/picfetch-microsoft-store-arm64.exe` | 49,997,312 | `0c446d3523142e4b06e4dca4e80862594ce3e0154d12a07629268d308dd37641` |

All Windows executables have the expected x64/ARM64 PE machine and GUI subsystem.
The exact ordinary x64 EXE and freshly staged Store x64 EXE each analyzed a
synthetic image successfully, reported OfflineVerified=false, and exited cleanly.
Store staging is in `.scratch/linux-arm-explorer/store-x64` and `store-arm64`.
The Linux ARM64 artifact and ABI are recorded in the parallel Linux plan.

`bin/picfetch-windows-amd64.zip` (28,624,833 bytes, SHA-256
`aa74f5019be73ad64bc15cc54e904642a6f3eb7d3444d0037edc3902d7d07d21`) and
`bin/picfetch-windows-arm64.zip` (26,832,845 bytes, SHA-256
`bb7a6c6776654318d6c0e7794e631449eb579ea4a5189d928e332505efcc5b18`) each
contain picfetch.exe and all three license/notice/privacy documents; all four
members were verified against source hashes.

AGENTS.md now requires license and shipped-dependency checks at selection/upgrade
time, with source/version, distribution obligations and notice delivery recorded
before a release-ready claim. This records the missed responsibility explicitly.
The lead owned the adapter, vendoring, tests, review and fixes; one existing scout
checked the permissive source/closure. No review or implementation was delegated.
No commit/push/publication or system policy change was performed. Hardware and CI
acceptance keep the plans active; the former GPL clustering blocker is removed.
