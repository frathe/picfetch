# Explorer on Intel macOS

Route: Deep (platform support and native dependency selection). Enable the
existing Intel Mac release build to install and run Explorer using Microsoft's
last official Intel runtime. No commit, publication or global tool installation.
The release matrix and updater already support macOS x86_64.

## Decisions and dependency closure

- Microsoft removed Intel macOS binaries in
  [ONNX Runtime 1.24.1](https://github.com/microsoft/onnxruntime/releases/tag/v1.24.1). Both the
  GitHub and NuGet 1.29.0 distributions lack macOS x64. Use the official
  `onnxruntime-osx-x86_64-1.23.2.tgz` from
  https://github.com/microsoft/onnxruntime/releases/tag/v1.23.2 for Intel only.
  Other platforms retain 1.29.0. This is a legacy runtime without current
  upstream Intel binary updates, not qualification of a current runtime.
- The existing Go wrapper requests C API 29, which 1.23.2 cannot supply.
  Select unmodified `github.com/yalue/onnxruntime_go` v1.25.0 (API 23,
  commit `7db3ea8b1c748950b3a587ce074a8be139217a71`) through a distinct module
  alias on darwin/amd64 only. A narrow `internal/ort` package selects the
  binding at build time for both the encoder and tag-generation tool.
  Never link both bindings into one executable. Other targets keep v1.36.0.
- Both wrapper versions and ONNX Runtime are MIT licensed. Preserve their
  notices in THIRD-PARTY-NOTICES.md and the runtime's original LICENSE and
  ThirdPartyNotices.txt beside the extracted library. Review the original
  native dependency closure and Privacy.md before handoff. The existing
  SigLIP 2 model/revision and Apache-2.0 obligations remain unchanged.
- Preserve explicit downloads, byte/hash checks, CPU inference, telemetry
  opt-out, sandbox network denial and closeable worker control input.
- Preserve the recorded host constraint: serial bounded checks, no broad
  Docker/race run or Windows isolation probes. Intel native execution and
  the complete Linux race gate belong in CI/on hardware; cross-compilation
  cannot establish either.

## Tasks, acceptance and routing

| Task | Files / contract | Verification | Owner / spawn budget |
|---|---|---|---|
| Runtime provenance | ignored `.scratch/macos-x64-runtime`; publisher archive and library hashes, Mach-O dependencies, license closure | `Get-FileHash`; archive/Mach-O inspection retained in scratch | one read-only scout / 1 |
| Runtime and binding | assets.go, assets_install.go, assets.sha256, assets_test.go, go.mod/sum, new internal/ort pair; existing encoder and explorertags imports | platform/download/extraction tests; build-selected dependencies and API/header checks; cgo build when toolchain permits | Lead / 0 |
| Worker and evaluator | control build tags, platform errors, evaluator admission and existing tests | focused similarity/evaluator tests; darwin amd64/arm64 test compilation; native CI worker exit guard | Lead / 0 |
| Integration and docs | CI Intel matrix; setup string, both translations/manuals, README, notices, Makefile, architecture, todos | nativeguards workflow contract; translation/manual checks; formatting/vet/build | Lead / 0 |

Graph: provenance scout alongside lead binding/worker work, then integration,
focused verification, lead review and evidence update. No reviews or fixes
delegated. Scout gate: G1 bounded artifact question; G2 retained publisher/hash
and binary-inspection output; G3 no tracked edits; G4 external artifact context
is smaller; G5 parent did not already inspect that artifact. Rule S cannot
replace interpreting the upstream support and binary closure; Rule W does not
apply to a read-only evidence task.

## Evidence and cost

Official archive: 11,676,322 bytes, SHA-256
`d10359e16347b57d9959f7e80a225a5b4a66ed7d7e007274a15cae86836485a6`.
The lead recomputed the hash and compared it to Microsoft's release metadata.
The library is 39,742,608 bytes, SHA-256
`8c9c78de65ea3786f987c0d980e9c1b13a3a5fbc6b3e2965ba05b450e6e4c054`.
Mach-O inspection identifies x86_64, macOS minimum 13.4 and only Apple frameworks
or /usr/lib dependencies. Runtime source commit is
`a83fc4d58cb48eb68890dd689f94f28288cf2278`; no extra native dylib is bundled.

The installer preserves LICENSE, ThirdPartyNotices.txt and Privacy.md. The
upstream Eigen component is MPL-2.0: retain the full native notice and the
unmodified source pointer at revision
`1d8b82b0740839c0de7f1242a3585e3390ff5f33`. Both Go wrapper LICENSE files were
compared by SHA-256 and are identical:
`b7177fa196f86e5102a5d2702c1b28ea62eb02fd2d633a37f2dbd6ffc0ba8b4f`.
The notices document the explicit module alias and both original source URLs.
Read-only provenance, downloaded archive, Mach-O inspector/output and notice
records remain in ignored `.scratch/macos-x64-runtime/EVIDENCE.md`.

Verification observed on Windows, with GOMAXPROCS=2 and GOFLAGS=-p=1:

- Platform and extraction tests failed first because darwin/amd64 was unsupported,
  then passed. Binding-selection tests failed first without a binding, then
  passed for all six target combinations. Dependency inspection confirms
  darwin/amd64 selects v1.25.0 only and Windows amd64 selects v1.36.0 only.
- `go test ./internal/ort ./internal/similarity ./scripts/explorereval
  ./scripts/nativeguards -count=1` passes with cgo enabled. Root translation,
  UI translation and manual guards pass. The actual setup subtests pass via
  `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^setup_' -count=1`.
- A temporary Go test overlay exercised the actual downloaded Intel archive
  through the production extractor and library verifier: four files extracted,
  the dylib hash matched, all three notices were present. No native code was
  loaded. A separate overlay deliberately changed the Intel download version
  to 1.29.0; TestRuntimeDownloadSelection failed on the wrong URL. Unmodified
  production platform/download/extraction tests passed afterward.
- `go vet ./...` and `go build ./...` pass with the native Windows cgo compiler.
  All 556 tracked/new Go files pass goimports in an ignored LF-normalized copy
  matching Linux checkout line endings. The raw Windows checkout's CRLF files
  make `goimports -l .` noisy; no unrelated source files were rewritten.
  Qodana exclusions cover all 240 test files; no top-level UI tests were added.
  `scripts/synctuf --check` and `git diff --check` pass.
  Module tidy normalizes the existing direct/indirect declarations for
  testify and x/sys and removes stale checksums alongside the new binding pin;
  existing dependency versions are unchanged. `go mod tidy -diff` is clean.
- `GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build ./internal/ort` passes with
  the existing portable Zig compiler, proving the API 23 binding compiles.
  Linking a Mac test binary needs the missing SDK's libresolv; broader cgo
  compilation needs AppKit headers. No-cgo Mac test compilation also hits the
  existing trash package's missing moveDarwin implementation. Those failed
  cross-build attempts do not count as native qualification.

CI now runs native Mac guards and ordinary similarity/evaluator tests on both
Mac architectures, plus `make explorer-install-test` on Intel for a real
download, reuse, CPU inference and OS network-denial check. Architecture-specific
artifact names prevent the parallel Mac jobs from colliding. The existing Intel
release bundle/updater route is retained. No CI execution or Mac app packaging
was performed here; no commit or publication was made.

Actual cost: one read-only provenance scout, one lead review plus inline fixes,
no delegated implementation/review. The complete Linux race gate remains CI work
under the recorded host constraint. Keep this plan active until Intel native
inference, sandbox/cancellation, UI and packaged-app acceptance are observed.
