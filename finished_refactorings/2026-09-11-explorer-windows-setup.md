# Windows Explorer setup

Route: Deep (native Windows behavior). Deliverable: verified Windows 11 x64
Explorer setup and analysis on `feature/similarity-explorer`, plus a local test
EXE. Implementation, review, and fixes belong to the lead. No commits authorized.

## Decisions

- Keep the shared SigLIP 2 revision and official ONNX Runtime 1.29.0 CPU runtime.
- Windows uses a hidden ordinary subprocess with inherited process pipes. It
  opens no model service port and installs no OS network block. This follows
  the user's choice to disable telemetry without firewall/AppContainer setup,
  after clarifying the difference between pipes, localhost, and outbound access.
- Disable ONNX Runtime telemetry through the native API before creating sessions;
  stop initialization on opt-out failure. Keep the pre-load environment opt-out.
- Explain the Windows policy on first use, in both translations/manuals, and in
  PRIVACY.md. Keep Linux seccomp and macOS sandbox network denial unchanged.
  Windows must report `OfflineVerified: false`.
- Download the pinned Windows ZIP and verify both DLLs. Extract only required
  libraries and license notices; skip headers, import libraries and debug symbols.
- Windows ARM64, native library-trial tooling, Store packaging/publishing and
  diagnosis of the earlier system crashes are outside this implementation.
- Store follow-up: bundle runtime DLLs in MSIX and qualify model-only downloads,
  or obtain Microsoft's confirmation for downloaded DLLs. See `todos.md` and
  `docs/microsoft-store.md`; a standalone EXE is not Store certification evidence.

## Host constraint

The user reported two Windows crashes during prior work. Event 41 records had
BugcheckCode=0; the cause is not established. Do not repeat AppContainer/bfscfg/BFS
probes from the scratch directory. Do not run the broad Docker/race gate on this
host in this task. Use serial bounded native tests, GOMAXPROCS=2 and Go `-p 1`.
No firewall, global installation, source ACL or system-policy changes are needed.
The initial automatic-review block on enabling a network-capable worker was
resolved after the user's policy choice; the enabling patch was approved/applied.

## Acceptance criteria and task graph

Toolchain/assets research -> installer -> Windows worker/telemetry -> setup/docs
-> native integration -> lead review/build. Store research is independent.

| Criterion | Owner / files | Verification |
|---|---|---|
| AC1: pinned Windows x64 ZIP, exact bounded allowlist, required shared DLL, integrity/cancellation | Lead; `internal/similarity/assets*.go`, `assets.sha256` | `go test -p 1 ./internal/similarity ./scripts/explorereval -count=1` |
| AC2: real downloads, retained notices, reuse without HTTP | Lead; installer and `scripts/explorereval/assets_install*_test.go` | `go test -p 1 -tags explorerinstall ./scripts/explorereval -run '^TestRealAssetDownload$' -count=1 -v -timeout 25m`; run `go run ./scripts/explorereval -install -assets .scratch/visual-similarity-explorer/assets` twice |
| AC3: ordinary hidden Windows worker, API telemetry opt-out, honest network policy and clean exit | Lead; worker/control platform files, encoder, analyze/offline, test assertions | AC1 plus `go test -p 1 -tags explorerinstall ./scripts/explorereval -run '^TestInstalledAssetAnalysis$' -count=1 -v -timeout 2m` |
| AC4: translated first-use notice precedes analysis; native setup, source limit and cancellation/restart | Lead; UI setup/state, existing test files, translations/manuals | `go test -p 1 -tags explorertrial ./internal/ui -run '^TestVisualSimilarityExplorer(Local)?$/(setup_|configured_file_size_limit|cancel_ui|cancel_worker)' -count=1 -timeout 3m`; inspect captures through `PICFETCH_EXPLORER_SETUP_QA` |
| AC5: translations/manual guards and local static/build gate; test EXE | Lead; docs, Makefile and final artifact | `go test -p 1 . -run '^TestTranslations' -count=1`; `go test -p 1 ./internal/ui/help -run '^TestManual' -count=1`; `mingw32-make verify-build`; `mingw32-make build BIN_NAME=picfetch.exe` |

The native shell needs the ignored portable toolchain activation script at
`.scratch/windows-explorer/toolchain.ps1`. Preserve the installed Go module/build
caches when sourcing it, set GOFLAGS=-p=1, and pass the Git Bash executable as
Make's SHELL. No global compiler installation is required.

The full `make verify` Linux/Docker race gate is deferred to CI because of the
host constraint. Native tests use the Fyne software test driver; they do not
qualify the actual Windows desktop's graphics behavior. User EXE acceptance is
separate. No new test files or top-level ordinary UI tests were introduced, so
Qodana exclusions and the UI shard manifest need no entries.

## Evidence

- RED: runtime selection failed for missing Windows support; valid Windows ZIP
  failed while only tarballs were implemented. Full similarity tests then passed.
- RED: Windows worker asset-check/exit test failed at unsupported-platform
  admission. It passed with the Windows launcher and ordinary control pipes.
- RED: the first-use notice test found no notice before the UI change. The setup
  tests passed after adding it and still assert no analysis before Continue.
- Actual download qualification passed: 451,453,666 bytes, runtime licenses,
  archive and extracted DLL hashes, and reuse with an HTTP-failing transport.
  Persistent CLI install and second-run verification/reuse also passed.
- Native cgo similarity and evaluator package tests passed. Real synthetic-image
  inference passed in 1.34 seconds, exercising the DLL, telemetry opt-out, private
  pipes, complete worker exit, and `OfflineVerified: false`.
- Native UI setup/source-size/cancellation/restart tests passed in 15.301 seconds.
  Captured setup at 720x660 and 520x400; the Windows notice and action button are
  readable and remain inside the small canvas. Logs/captures are in ignored
  `.scratch/windows-explorer`.
- Final static/build checks and executable evidence are recorded below.

## Routing and cost ledger

| Task | Delegation and oracle | Actual |
|---|---|---|
| Portable toolchain restoration | One tooling scout; ignored tool paths only; publisher checksum and tiny cgo arithmetic probe | Completed; no native isolation/runtime probe |
| Runtime dependency research | One read-only scout; official release/source plus bounded ZIP DLL import/hash inspection | Completed; no DLL execution |
| Store policy research | One read-only scout; current Microsoft policy and AI Dev Gallery sources | Completed; no publishing |
| Implementation/review/fixes | Lead; AC commands and diff review | Inline throughout |
| Final gate | Lead; serial native tests/static checks/build | Full Docker/race suite intentionally deferred |

Delegation gate for each scout: bounded standalone question, external evidence
oracle, no tracked production files, independent search context, no duplicated
lead implementation/review. No user-visible strings or review fixes delegated.
The toolchain scout was resumed after prior lost/interrupted bootstrap work;
three independent research/tooling tasks were used, at most two concurrently.
The original 238 added `go.sum` lines belong to the user's pre-existing changes
and remain preserved. Keep this plan active until branch acceptance; then move
it to `finished_refactorings/`.

### Final native gate

`go vet ./...` and `go build ./...` passed on Windows with cgo and serial package
compilation. Focused similarity/evaluator tests passed again (0.984s/0.208s),
including the Windows policy guard. Translation guards passed (0.238s), and
manual guards passed (0.384s). Qodana's exact exclusion inventory matches all
234 test files. `git diff --check` passed.

The Make wrapper could not fork Git Bash children (0xC0000142), so the lead ran
its underlying checks directly in PowerShell. Raw `goimports -l .` also reports
the checkout's existing CRLF files and ignored scratch probes. A read-only helper
checked all 543 repository Go files using the same goimports options with CRLF
normalized in memory; all passed. Only changed Go files were formatted on disk.
The TUF root check passed. No global shell/tooling changes were made.

### Local executable

Built `bin/picfetch.exe` using the Makefile build recipe with `-H windowsgui`
added: `go build -trimpath -ldflags="-s -w -H windowsgui" -o bin/picfetch.exe .`.
Artifact: Windows x64 PE32+ GUI, 53,587,968 bytes. SHA-256:
`689ebd88c4a53b406b02b32b9b49829b7a2e66806d4d11d35d087ab893ba8987`.

The exact EXE analyzed one generated 96x64 PNG through its private worker mode,
reported one successful source, no failures, `OfflineVerified: false`, and clean
process exit. The helper was `go run ./.scratch/windows-explorer/exesmoke.go`;
it did not open the desktop app or use the user's images.

A follow-up read-only tooling scout inventoried PE imports while the lead ran
that smoke test; the lead repeated objdump inspection. The binary imports only
Windows system/UCRT APIs, with no GCC/libstdc++/winpthread DLL to distribute.
ONNX Runtime is still the separate verified setup download, and its Visual C++
x64 prerequisite remains as documented. No Store package or signature was made.
This follow-up reused the existing tooling scout and changed no files.

Implementation and bounded native qualification are complete. Actual desktop
GUI acceptance belongs to the user's local test; complete Linux/race verification
remains for CI. The existing 238-line go.sum addition is still untouched. No
commits or pushes were performed.

## Store runtime bundling follow-up (authorized 2026-09-11)

The user selected bundled ONNX DLLs for the Microsoft Store version and requested
updated third-party notices. Route remains Deep. Preserve the completed direct
Windows installer and the user's staged/unstaged changes; no commits or publishing.

Task graph: asset/notices reconnaissance -> runtime location and model-only
installer -> MSIX staging/workflow -> notices/docs -> serial native verification.

| Task | Owner / files | Acceptance command |
|---|---|---|
| Runtime/license inventory | One read-only scout; current module/notice inventory and pinned archive notices | Compare pinned archive/license metadata with notice entries; no executable loading |
| Store runtime selection and model-only setup | Lead; `internal/similarity/assets*.go`, encoder, existing tests | Ordinary and `-tags microsoftstore` focused similarity tests; Store never downloads native code or loads DLLs from the model cache |
| MSIX staging | Lead; `scripts/msixstage`, Store workflow and existing tests | Staging tests reject absent/mismatched runtime; actual pinned staging includes both DLLs and licenses for x64 and ARM64 |
| Notice/docs | Lead; `THIRD-PARTY-NOTICES.md`, privacy/manual/setup strings as needed, Store listing/docs | Exact upstream license text retained in staged package; translation/manual guards |
| Native qualification | Lead | `go vet`/build, runtime/model setup tests, staged Store EXE one-image worker smoke; no AppContainer, Docker/race, desktop launch or OS changes |

Scout gate: one bounded independent dependency/license sweep, a reproducible
inventory comparison, no modified files, no lead implementation or review context.
All design, UI text, review and fixes remain with the lead. The final package may
be staged/packed locally if tools are available; Store certification/publication
is not implied by this request.

### Windows ARM64 extension and Store decisions

The user accepted the x64 desktop build, then requested an ARM64 build and
authorized implementation for a later hardware test. Both Windows architectures
now use ONNX Runtime 1.29.0 with independently verified archive and DLL pins.
Direct setup transfers 451,453,666 bytes on x64 or 453,487,179 on ARM64; Store
setup transfers 371,808,146 model/configuration bytes on either architecture.

The Store runtime root is the executable directory, independent of model/cache
overrides. Missing or corrupt packaged DLLs fail before HTTP and the UI explains
Store repair/update. Packaging requires the matching pinned runtime archive and
an empty staging directory, retains both DLLs and all three upstream notice
files, and excludes debug symbols. The manifest declares the Desktop C++
framework at 14.0.33728.0. Read-only x64 PE-import/export comparison found every
ONNX C++ symbol in that installed framework; this is a known sufficient floor,
not a claim about the oldest usable version or native ARM qualification.

The existing tooling scout obtained the official LLVM-MinGW portable x86_64
host / ARM64 target toolchain (20260908 UCRT). Its verified archive SHA-256 is
`1bcf74d06b724aeecaa6412ca85f5b26fb1da770e7cdcefa9263c9c5c3ad34b6`.
Activation is process-local through the ignored
`.scratch/windows-explorer/arm64-toolchain.ps1`; package compilation is serial,
with GOMAXPROCS=2. An ARM64 cgo arithmetic executable cross-compiled with PE
machine AA64; it was not executed. No Docker, AppContainer/BFS, firewall change,
SDK installation, certificate trust change or package installation ran locally.

Acceptance extends to an ordinary ARM64 GUI EXE, Store x64/ARM64 EXEs with matching
runtime staging, focused native tests and exact x64 Store worker analysis.
ARM64 execution belongs to the user's hardware test; signed MSIX/WACK and the
complete Linux race suite remain CI work. The Windows SDK packaging tools are
absent on this host, so staging does not claim an actual certified MSIX bundle.

### Notice inventory and release follow-up

The notices now include ONNX Runtime, its Go binding, UMAP, Gonum (including
its Go/Cephes notices), and the existing latent v0.1.4 GPLv3 license. Store
staging also copies PicFetch's license/privacy/third-party notice documents and
the runtime's unmodified upstream ThirdPartyNotices.txt and Privacy.md. The
latter documents initialization telemetry that can precede the API opt-out;
the application promises the opt-out before session creation.

The audit found that `github.com/alDuncanson/latent/projection` has a GPLv3
license with no separate projection exception. This predates the Windows
increment. The lead asked about replacing it with a permissive implementation;
no explicit license/algorithm decision was received. The ARM build instruction
was applied to ARM support, not treated as permission to relicense PicFetch.
Record the distribution decision in todos; updating notices alone does not
close it. A bounded read-only scout found PhotoPrism's `pkg/vector/alg` at
`c48d23f6b03c25fc19d376d789fac56c32a26fdb` with its own MIT license and an HDBSCAN
implementation accepting both minimum points and minimum cluster size. Its
unmodified candidate files are in ignored scratch space only. No new dependency
or clustering algorithm was introduced. Future replacement needs behavior and
performance qualification, including the minimum-neighbor convention.

### Store verification record

RED tests observed the previous Store runtime download size, absent-runtime
staging acceptance, unsupported Windows ARM runtime, and missing C++ manifest
dependency. All pass after implementation. Packaging tests also cover archive
checksum/size rejection, cancelled staging, empty-output admission and runtime
failure propagation. The UI test exposed the generic connection message for a
missing bundled runtime; the final translated repair/update message fixes it.

The x64 evaluator was compiled with `microsoftstore explorerinstall`, staged
with the actual pinned x64 archive, and run with quoted Go test flags. Its real
model-only install completed in 20.23 seconds, verified 371,808,146 bytes,
retained notices, reused installation without HTTP, ignored fake cached DLLs
and successfully analyzed one generated image with clean worker exit and
OfflineVerified=false. Evidence: `.scratch/windows-explorer/store-install-test.log`.
An initial PowerShell launch split an unquoted dotted test flag and printed
usage; quoting the arguments fixed the invocation without a code change.

### Final artifacts and bounded native gate

| Artifact | Bytes | SHA-256 |
|---|---:|---|
| `bin/picfetch-windows-arm64.exe` | 49,997,312 | `029f1ed19c9784a939e4d3174b6fd06ec36eddf4169197252a3e30d40246237f` |
| `bin/picfetch-microsoft-store-arm64.exe` | 49,984,000 | `a2ff066504da4b9df1398a209d30afd0e6e6a0404c357ee7924c8dc3d088ab89` |
| `bin/picfetch-microsoft-store-amd64.exe` | 53,579,776 | `0211157d29ed879900b9848bac2e11364e9ca887839207d9f1221577c7227ae8` |

Both ARM64 files have PE machine AA64 and GUI subsystem 2. Read-only import
inspection found Windows/UCRT dependencies, without a separate LLVM/MinGW
runtime DLL. The previously user-tested `bin/picfetch.exe` remains available.

Actual Store packages were staged into
`.scratch/windows-store-explorer/stage-x64` and `stage-arm64`, each with its
architecture-specific runtime and the final notices. The exact staged x64 GUI
EXE passed private-worker analysis of one synthetic image and clean exit
(`.scratch/windows-explorer/store-exe-smoke.log`). ARM64 was not executed.

Ordinary native similarity/evaluator/packaging tests pass. The native Explorer
setup, configured file-size limit and cancellation subset passes (11.106s), as
does the unstaged Store setup/repair subset (1.448s). A Store-tagged UI test EXE
was then staged with real DLLs and passed the same setup/lifecycle subset,
including native file-size and cancellation guards; evidence is
`.scratch/windows-explorer/staged-store-ui-test.log`. These use the Fyne test
driver, not the user's desktop. Root translation and manual guards pass
(0.207s/0.420s), and `go vet -tags microsoftstore ./...` passes.

The final format check covers all 544 repository Go files with CRLF normalized
in memory. Qodana exclusions cover all 234 test files; no new test file or
top-level UI test was added. Working-tree and staged `git diff --check` pass.
The user's original 238 added go.sum lines remain unchanged, and the staged
worker/plan entries remain staged. No commit, push, package installation or
Store publication was performed. ARM64 hardware acceptance, CI's broad suite,
MSIX/WACK, and the pre-existing GPL release decision remain explicitly open.

The final staged evaluator also passed `TestAssetInstall`: cancelled admission,
missing assets/worker exit, rejected untrusted redirects, failed-transfer file
preservation, and modified-download rejection (0.09s). The Store similarity and
packaging packages passed with cgo (2.422s/1.881s). Evidence:
`.scratch/windows-explorer/store-install-fixtures.log`. Both staged packages'
PicFetch license/privacy/third-party documents match the final source hashes.

### Release-status clarification

The ordinary release matrix builds macOS, Linux and Windows for ARM64 and x64;
Explorer's current runtime matrix is narrower: macOS ARM64, Linux x64, and
Windows x64/ARM64. Intel macOS and Linux ARM64 Explorer are not implemented.

A follow-up release-status inspection found a notice-delivery gap outside Store
packaging: `.github/workflows/release.yml` archives only the Windows/Linux
executables, and the Windows signing step repacks only the executable. Track
shipping license/notice documents and source-access information in all standalone
artifacts (including inspection of the macOS bundle) before release. No workflow
or license decision was changed during this status answer.

The scoped license recheck confirmed latent is directly used by production
grouping and GPLv3, rather than LGPL. ONNX Runtime also retains Eigen's MPL-2.0
notice; the root notice already describes golang-lru's MPL-2.0 obligations.
Unchanged code still carries distribution obligations. This check is not a full
transitive legal audit, and the GPL combined-work decision remains open.

### Subsequent authorized remediation

The user subsequently requested Linux ARM64 support and replacement of the GPL
clustering dependency. Both are implemented in the working branch; see
[Linux ARM64](2026-09-11-explorer-linux-arm64.md) and
[permissive clustering](2026-09-11-permissive-clustering.md) for current evidence.
The MIT-licensed HDBSCAN subset replaces latent, and standalone notice delivery
is fixed, including signing repacks and macOS Resources. Earlier artifact hashes
above describe the preceding Windows increment; replacement binaries and current
hardware/CI limitations are recorded in the follow-up plans.

## PR 18 CI follow-up (2026-09-11)

The accepted review round at `408951f` passes the complete Linux race suite,
canonical shard validation, and Windows native guards. This closes the deferred
broad-CI verification above. ARM64 hardware/app acceptance and actual MSIX/WACK
qualification remain separate; no release or Store submission is implied.
See [review evidence](../finished_refactorings/2026-09-11-pr18-review-limits-evidence.md)
and [CI run 34610468241](https://github.com/frathe/picfetch/actions/runs/34610468241).

## Windows ARM64 map crash investigation (2026-09-11)

Ronin reports that indexing finishes and the viewer closes as cohort rendering
begins; he explicitly identified Windows ARM64. Device, image count, reproduction
on a small folder and crash output are pending. No cause or production fix is
established. Native Windows ARM64 rendering remains an open acceptance gate.

Before the platform clarification, the current Apple Silicon host passed
`go test ./internal/ui -run '^TestVisualSimilarityExplorer$' -count=1 -timeout 3m`
(18.327s) and the real-model `granularity`, `small_cohort` and `partial_map`
subtests (6.741s). The latter required running outside the agent sandbox so
PicFetch could install its own macOS sandbox. Native synthetic replays with 300
and 3,000 sources each exited 0 after 109 observed paint/interaction steps.
These checks do not reproduce or qualify the reported Windows failure.

`make package-windows-debug WIN_ARCHES=arm64` passes with the existing pinned
cross-toolchain. PE inspection verifies AA64 and retained DWARF sections. The
packaging CLI still sets subsystem 2 (GUI), despite fyne-cross's `-console` flag:
fyne-cross v1.6.3 parses that flag but never uses it; Fyne tools v1.7.2 adds
`-H=windowsgui`. Correcting that debug-target promise is separate work. The
diagnostic CMD launcher supplies the persistent console and redirected logs.
The packaging-only FyneApp.toml build increment was restored.

Local handoff: `bin/picfetch-windows-arm64-diagnostics.zip`, 37,104,598 bytes,
contains the debug EXE, CMD launcher, instructions, build identity and existing
license/privacy/third-party notices. The EXE is 82,466,304 bytes, SHA-256
`d64f9fbbe1f4d065d11863d3eea56d8a617379b7d34741d0ba31c61822486571`.
ZIP CRC and every unpacked entry were compared with their source bytes.
The launcher records application output, exit code, Windows/CPU/GPU versions
and recent matching Application Error/WER records; dragging the original EXE
onto it captures that build instead. Nothing is uploaded automatically.
Windows execution of the executable/launcher remains unverified on this host.
Retained harnesses and build log: `.scratch/arm-map-crash-20260911/`.

Next gate: run the launcher on the affected machine and obtain the report;
establish a failing reproduction before selecting a production change. No broad
verification suite was run for this diagnostic/docs-only handoff. Lead owns the
diagnosis, all changes and review. One read-only scout performed two bounded
evidence tasks (old replay provenance and documented Windows capture commands);
no implementation or review was delegated, and no commits were made.

### First Windows report and direct-file capture v2

The user supplied the matching diagnostic EXE hash and a run from 17:59:35 to
17:59:43 on Windows 11 Pro build 26100, 64-bit ARM, approximately 8 GiB RAM,
CPU `virt-10.0`, and VirtIO GPU DOD 22.7.38.43. Exit code was 2; application
output was empty and the Windows event query found no records. This does not
identify a fault or prove that this particular run reached the original map
handoff. A clarification about startup versus post-indexing exit is pending.

The local UTM Windows 11 VM is running, but `utmctl exec` reports no guest agent.
The desktop matches prior test records with app-local ARM64 GL/EGL/GLES DLLs.
Computer-use keyboard/mouse injection did not reliably drive its guest; no
calibration command or application launch was completed by that route. Input
capture was restored to off. One new read-only scout found the prior guest-access
and graphics-runtime evidence; implementation and review remain lead-owned.

V2 uses temporary Go overlays for direct `log`/stderr output and
`runtime/debug.SetCrashOutput`, plus map grouping/application/fit markers. Worker
stdout remains untouched. A deliberate capture self-test on the host exited 2
and preserved both startup marker and panic stack with stdout/stderr discarded.
This tests file capture, not Windows graphics or the original defect.

The first direct cross-build attempt failed because its default cache resolved
under `/.cache`; the successful rerun specified writable Go/Zig caches. It uses
the same pinned Windows container, an explicit console linker subsystem and
retained symbols. Read-back PE verification requires AA64 and subsystem 3.
`bin/picfetch-diagnostic-v2-windows-arm64.exe`: 80,671,232 bytes, SHA-256
`cbcd359d4cb84b0e6b39c976b1fabe9fddd8ead9a685c3c8f931c7157d0c207d`.
`bin/picfetch-windows-arm64-diagnostics-v2.zip`: 35,490,944 bytes, seven flat
entries, CRC and every entry verified against source bytes. It includes the
launcher, instructions, identity and existing license/privacy/notices.

The v2 launcher runs its deliberate-panic calibration separately, checks that
its direct file contains the expected stack marker, then starts the real app.
The `app` directory and `app-console.txt` contain the actual attempted
reproduction. In this VM the EXE/CMD belong beside the existing Desktop GL DLLs;
no DLLs or system settings are changed. Native v2 execution remains pending.
Normal application source is unchanged; all instrumentation is retained under
`.scratch/arm-map-crash-20260911/v2/` and must never enter a release build.

### Confirmed SVG-source map crash and fix

The returned `picfetch-diagnostics-v2-21585-19087` report now captures the actual
failure: the worker published 113 items, 112 successful, and four hierarchy
merges. Map fitting panics through `pileRenderer.Layout -> canvas.Image.Resize
-> renderSVG -> svg.Decoder.Draw/Config`. Repeated XML-parser errors precede it.
The user's original report directory is preserved untouched and untracked.

Route for the implementation: Standard, a two-file portable bug fix. The
producer always encodes `Item.Preview` as JPEG, while pile construction gave its
resource the original source path. A `.svg` name forces Fyne v2.8.0 to parse
those JPEG bytes as SVG; the failed decoder is nil when Fit resizes the pile.
Resource names now append `.preview.jpg`; cohort membership retains the exact
original paths. No dependency or native-runtime changes are involved.

| Task | Owner/files | Acceptance command | Budget |
|---|---|---|---|
| Regression and fix | Lead; map.go and existing explorer_test.go | `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/svg_source_previews$' -count=1` | 0 implementation spawns; one review |
| Adjacent naming sweep | One read-only scout, no edits | `rg 'NewStaticResource|NewImageFromResource' internal` plus byte provenance | 1 scout; no review delegation |
| Regressions and final gate | Lead | complete Explorer test, `make verify` | one full suite |
| Windows handoff | Lead; existing Make packaging and local artifact/notice assembly | ARM64 PE/header/hash/archive inspection plus user reproduction | native acceptance required |

Task graph: local regression -> minimal naming fix -> focused regressions ->
Windows package -> full gate; the read-only adjacent sweep runs independently.
The scout task is standalone, has an exact call-site/provenance oracle, changes
no files and does not duplicate lead work. No other generated preview/resource
name mismatch was found. Lead retains all code, review and fixes.

RED observed: the new `svg_source_previews` subtest fails in 0.19s (package
0.933s) with the same nil decoder stack at `map.go:540`. The fixture uses valid
SVG sources and known JPEG previews, excluding corrupt original SVGs or a
Windows-only graphics requirement. The test also checks uppercase `.SVG`, a
JPEG source whose name contains `.svg`, decoded dimensions, viewport release
and restoration, and original cohort navigation identities. It is a subtest
of the already-sharded/excluded Explorer test; no manifest entries are added.
Evidence: `.scratch/arm-map-crash-20260911/svg-source-red.log`.

GREEN: the crash regression passes (0.997s). The first whole-Explorer run exposed
an old test assumption that resource names equal source paths; its membership
assertion now strips the preview suffix. The complete Explorer suite then passes
(17.244s). Native Fyne/GL replay with 30 synthetic `.svg` source paths completed
all 109 paint markers and exited 0; the constructed cohort screenshot was
visually inspected. This directly exercises the corrected preview naming with
the native renderer, but is not Windows qualification.

`make package-windows WIN_ARCHES=arm64 BIN_NAME=picfetch-mapfix` passes. The
packaging-only build-number increment was restored. Read-back verifies AA64,
normal release packaging flags and no `[DEBUG-armmap]` instrumentation.
`bin/picfetch-mapfix-windows-arm64.exe`: 51,394,560 bytes, SHA-256
`817c1c1928fedbc25cdad62e718008ef82480b91b982ada96fa41d04f89707a0`.
`bin/picfetch-windows-arm64-mapfix.zip`: 27,387,374 bytes, six flat entries,
including instructions, identity and existing notices; CRC and every entry
were compared to their source bytes. Ronin has the corrected artifact for the
same VM/image-set test. Existing executables and GL DLLs are preserved.

The first `make verify` stopped at formatting the three temporary diagnostic
harnesses; those files were formatted with the repository's goimports options.
The corrected gate subsequently reached the race suite; its results are below.
The initial formatting failure did not reach or repeat the broad race suite.

Ronin tested the corrected artifact on the affected Windows ARM64 VM and
confirmed, "this version works." This closes the original post-indexing map
crash reproduction. It does not establish MSIX/WACK or Store qualification.

The full local gate passed formatting, TUF/exclusion checks, vet, build and shard
validation, then reported two Linux worker failures: `TestLinuxWorkerIsolation`
and `TestAssetInstall/worker_reaches_asset_check_and_exits`, both with
`offline worker seccomp: invalid argument`. This run uses Linux/amd64 emulation
on Apple Silicon. The same focused tests pass with `-race -count=1` in a native
Linux ARM64 container: similarity 2.022s, explorereval 1.035s. The first native
attempt required enabling the repository's Go 1.27.1 toolchain; the next passed
isolation but lacked X11 headers for the evaluator. Installing those headers
inside the disposable container completed the focused check. No sandbox policy
or test was weakened. All three GUI race partitions pass: ui-1 623.384s,
ui-2 330.853s, ui-3 318.096s (680 top-level passes and one skip). The complete
Explorer test passes in 291.820s, including the new SVG-preview regression in
2.440s. The full `make verify` exits 2 because of the two isolated non-UI failures;
it is not a clean complete gate. Their follow-up remains in `todos.md`.

Evidence: `.scratch/arm-map-crash-20260911/verify.log`,
`.scratch/race-runs/20260911T161936Z-sP2Uoy/`, and
`.scratch/arm-map-crash-20260911/native-linux-isolation-prepared.log`.
The separate Windows static check
`GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/...` also passes.

### User-requested setup wording (2026-09-11)

Ronin accepted the map fix, then requested a simpler English/German first-use
page: call the feature Similarity Explorer (German: Ähnlichkeits-Explorer), name
the SigLIP 2 AI model, and explain its use. Remove the technical Windows notice;
the privacy policy retains the detailed explanation. This supersedes the earlier
requirement to show that notice during setup. Keep the total download size and
offline availability, explicit download/Continue actions and privacy-policy link.

Lead owns all edits and review: setup and its obsolete UI-only isolation flag,
menu/settings names, both translation bundles, existing affected assertions and
current manuals/README/privacy headings. Historical plans and internal identifiers
retain their existing names. No inference or network-policy behavior changes.
One read-only scout maps terminology locations and the pinned model name; its
oracle is the bounded `rg` sweep of current UI/docs and asset definitions. It
changes no files and runs independently of the lead's setup work.

Verify existing Explorer setup/menu tests, locale/manual guards, formatting,
vet/build, and inspect English/German setup captures at ordinary and small sizes.
The already-running full race suite covers the map fix; focused checks cover this
subsequent presentation change. Refresh a separate Windows ARM64 package so the
already-tested map-fix artifact remains available. No commit is authorized.

The setup text now reads "Find and group similar pictures with the SigLIP 2 AI
model." plus the one-time total download size and offline availability. The
German version uses "Finde und gruppiere ähnliche Bilder mit dem KI-Modell
SigLIP 2." The Windows technical paragraph and duplicate privacy explanation
are removed; the privacy-policy link remains. Menu/settings/manual names match,
and the obsolete UI-only isolation flag is removed. The actual worker isolation
and telemetry settings are unchanged.

Focused setup/translation tests pass (ui 1.675s), as do the root package (0.629s),
menus (0.938s), settings (1.811s) and manuals (3.420s). The exact viewer main-menu
tests also pass. English and German setup captures at 720x660 and 520x400 were
visually inspected: ordinary-size copy is readable; at small size the reading
area scrolls while the title, privacy link and actions remain visible. A temporary
Go test overlay loads each catalog and simulates missing model files in the
isolated viewer; no QA hooks are included in production. These host captures
show the host's download size; Windows uses its own total from the same helper.

`make verify-build` passes after packaging finishes. Its first attempt overlapped
Fyne's temporary generated metadata and executable cleanup and stopped at the
format gate; the sequential retry is clean. Evidence is under
`.scratch/arm-map-crash-20260911/`: `setup-copy-tests.log`, `copy-guards.log`,
`copy-root-menu-tests.log`, `copy-verify-build.log` and `setup-copy-qa/`.

`make package-windows WIN_ARCHES=arm64 BIN_NAME=picfetch-similarity` passes;
the packaging-only build-number increment is restored. Read-back verifies AA64,
the GUI subsystem, release flags, Go 1.27.1 and no diagnostic instrumentation.
`bin/picfetch-similarity-windows-arm64.exe`: 51,390,976 bytes, SHA-256
`75822da8ab0a6d0d5bde60320161a64b89e58e4f173bfec8075496738dca8e22`.
`bin/picfetch-windows-arm64-similarity.zip`: 27,318,250 bytes, six flat entries
including instructions, identity and current notices. CRC and every entry were
compared against source bytes. This refreshed package's wording has host visual
verification; Ronin's Windows acceptance applies to the preceding map-fix build.

Lead completed final diff review, all edits and verification. The map-fix phase
used one resource-naming scout; the wording follow-up reused it for one bounded
terminology sweep. No implementation/review was delegated. One full race run,
focused follow-ups only, and no commits or publishing were performed. The parent
Windows plan remains active for its separate outstanding acceptance work.
