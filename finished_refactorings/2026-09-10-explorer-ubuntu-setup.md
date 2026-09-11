# Ubuntu Explorer setup

Route: Deep (platform behavior). Deliverable: install and use the local similarity
Explorer on this Ubuntu 24.04 amd64 host, preserving macOS arm64 support.

Evidence: setup.sh rejects Linux, asset verification and runtime loading name a
Darwin dylib, and Client.Analyze/controlInput reject Linux. Host is Ubuntu 24.04
x86_64; direct /snap/go/current/bin/go works despite the Snap launcher failure.

Decisions: support Linux amd64 with the pinned official CPU runtime; retain model
revision and macOS runtime. Use a worker-only seccomp filter synchronized across
threads before reading source requests, deny sockets and io_uring, and retain
VerifyOffline. Failure to install isolation aborts analysis. No sudo, GPU setup,
new library dependency, Windows/ARM Linux support, or native trial launcher port.

Acceptance commands:
1. Platform selection, worker isolation/cancellation, extraction and download
   rejection: go test ./internal/similarity ./scripts/explorereval.
2. Actual HTTPS install, byte/hash verification, license retention, repeat install
   without HTTP, and synthetic-image offline inference: make explorer-install-test.
3. Developer setup uses the same installer and leaves verified discoverable assets:
   make explorer-setup (repeat to demonstrate reuse).
4. Localized setup surface: go test ./internal/ui -run TestVisualSimilarityExplorer;
   locale/manual parity and complete integration: make verify.
5. Built application: make build.

Task graph: reconnaissance -> platform assets + worker isolation -> setup/docs ->
real qualification -> final review/gate. All implementation and review are T0 inline.

Tasks:
- T1 assets: assets.go, assets_install.go, assets.sha256, encoder.go and asset tests.
  Contract: platform-selected verified runtime; aggregate progress reflects its size.
  Oracle AC1/AC2. Budget 0 spawns, 1 review, no full suite.
- T2 worker: client.go, control platform files, worker platform files and Linux tests.
  Contract: network denial on every thread before source admission; completion and
  cancellation remain observable. Oracle AC1/AC2. Budget 0 spawns, 1 review.
- T3 setup: CLI/setup.sh, Makefile, UI text/catalogues, manuals, architecture, todos.
  Contract: user and developer install paths share verified downloads. Oracle AC3/4.
  Budget 0 spawns, 1 review.
- Scout: read-only sweep of test/CLI assumptions and local build tooling while T0
  investigates runtime. G1 bounded prompt, G2 file:line/command evidence, G3 zero
  writes, G4 independent sweep, G5 no implementation context duplicated. S/W do not
  apply: several test flows require reading comprehension. Budget 1 spawn (actual 1).
- Final gate: T0 review, make verify once, real install, make build. No delegated review.

Portability steering: use distro-independent seccomp, no apt/bwrap dependency;
check official runtime glibc/libstdc++ requirements and qualify the same worker
on another distro container. Produce the existing Linux packaging build where feasible.

Honest limit: Linux support qualified on this Ubuntu amd64 host; other Linux
architectures and the macOS-specific full-library evidence launcher remain out of scope.

## Verification record

- Red: Linux admission and worker tests failed with the original Apple Silicon
  restriction; both passed after implementation.
- Isolation guard was deliberately run with the filter disabled: IPv4, IPv6,
  Unix sockets, socketpair, io_uring and x32 assertions failed on both threads.
  Restored filter passes; the parent retains ordinary loopback socket access.
- `go test ./internal/similarity ./scripts/explorereval`: pass.
- `make explorer-install-test`: pass, 382,891,026 downloaded bytes, verified
  licenses/checksums, HTTP-free reuse, actual offline inference (36.258 seconds).
- `make explorer-setup`: pass, repeated setup reused the verified files.
- Assets also copied to the normal per-user cache and reverified; the tagged
  `TestInstalledAssetAnalysis` passed from that cache.
- `make explorer-ui-test`: pass, including production inference, cancellation,
  caches, cohorts and setup lifecycle (133.679 seconds).
- `make build`: pass; actual `bin/picfetch` worker completed a synthetic image
  with `OfflineVerified=true`. Native Ubuntu binary requires GLIBC_2.38.
- Windows amd64 no-cgo cross-vet: pass.
- Final formatting, Qodana exclusions and whitespace checks: pass.
- `make verify`: formatting/TUF/vet/build pass. All three UI race partitions
  pass (986.676s / 591.852s / 585.379s). The full gate exits 2 because of the
  one comparison timeout detailed below; all other packages pass, and there
  was no OOM. Both isolated reruns of that unchanged comparison test pass.
  Full raw evidence: `.scratch/race-runs/20260910T212110Z-AfsBtB/`.
  This is not recorded as an entirely green `make verify` run.
- Other-distro build: Debian 12, pinned image
  `debian@sha256:88200866dfff7ea7f5cbcb6ec7c8a701889efe6fe859fe64d6990e4b07ea4171`,
  exact Go 1.27.1. Build/test script and logs in `.scratch/linux-portability/`.
  `bin/linux-portable/picfetch` builds with GLIBC_2.34, and its actual worker
  also passes on Ubuntu. Linux isolation and real offline inference pass inside
  Debian 12; repeated with `--network none --user 1000:1000` and read-only
  assets, both tests pass (inference 2.22 seconds).
  Debian 11 qualification was abandoned because retired repositories returned
  expired metadata and missing packages; no host package settings were changed.

Cost ledger: 1 read-only scout (budget 1), all implementation/review/fixes by
T0. One full verification invocation; focused reruns followed actual changes.

The full Docker run hit the unchanged comparison test
`TestCompareSettle_DrainsVectorReplacementBeforeWaitingForObsoleteTiles`'s
one-second Settle deadline during concurrent builds. Its isolated native race
rerun passed (3.923s); the exact Ubuntu Docker rerun also passed (4.345s). No comparison files changed.

Reproduce the additional distro build and its native isolation/inference checks
from the repository root (Go must resolve the go.mod toolchain):

```sh
docker run --rm -v "$PWD:/work" -w /work \
  -v "$(go env GOROOT):/opt/go:ro" \
  -v "$(go env GOMODCACHE):/modules:ro" \
  debian@sha256:88200866dfff7ea7f5cbcb6ec7c8a701889efe6fe859fe64d6990e4b07ea4171 \
  sh .scratch/linux-portability/build.sh
```

The final production executable was additionally fed a synthetic request inside
Debian with `--network none`: completed one successful image with OS denial
verified. The same Debian-built executable's production worker passed on Ubuntu.
Temporary build container/image removed; binaries and evidence retained.
