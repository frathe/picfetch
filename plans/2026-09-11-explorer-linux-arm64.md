# Linux ARM64 Explorer

Route: Deep (platform behavior and worker isolation). The user requested Linux
ARM support and will test the result on hardware later. ARM means ARM64/AArch64;
32-bit ARM, Intel macOS Explorer and the separate GPL release decision are outside
this increment. Preserve existing branch commits and the open Windows-plan/todo
edits. No commits or publication are authorized.

## Design and acceptance

Keep the common model, setup/cache paths, worker protocol and CPU inference.
Pin the official ONNX Runtime 1.29.0 Linux AArch64 archive and main shared library.
Reuse the Linux worker with ARM64 audit-architecture/syscall constants, the same
thread-synchronized seccomp denial, and pollable control input. Test filter
decisions in a userspace BPF VM on Windows; never install a filter on this host.

| Task | Owner / files | Acceptance command |
|---|---|---|
| Runtime pins | Lead; assets.go, assets.sha256, existing asset tests | `go test ./internal/similarity -run 'TestRuntimePlatforms|TestUnpackRuntime'`; actual archive hash/extraction check |
| Linux worker ABI/control | Lead; worker_linux.go (rename), worker_other.go, control pair, offline.go and offline_test.go | `go test ./internal/similarity -run TestLinuxNetworkFilter`; compile similarity tests for linux/amd64 and linux/arm64 |
| Download/qualification/docs | Lead; evaluator tests/README, setup string/translations, manuals, README, Makefile, architecture, todos | focused evaluator tests; translation/manual guards; exact Qodana exclusion for new test file |
| Portable test artifact | Lead; existing Makefile release architecture already includes ARM64; ignored native-host toolchain/sysroot | cgo ARM64 GUI cross-build; inspect ELF machine, dynamic dependencies and required symbol versions; provide binary/archive |

Task graph: archive/toolchain reconnaissance in parallel with lead-owned policy
and setup work -> focused tests -> cross-build -> lead review and handoff.
Two existing scouts handle bounded official archive/ABI and compiler/sysroot
research independently, with no tracked production edits. Lead owns all design,
implementation, review and fixes. Tool downloads/extraction stay under ignored
workspace directories, verified against publisher metadata. Compilation is serial
with at most two runtime workers. No Docker/WSL/VM startup, AppContainer/BFS probe,
firewall change, global installation or target-binary execution on Windows.

Native ARM64 inference, kernel enforcement and GUI behavior remain for the user's
hardware test. The complete Linux race suite remains CI work because previous
heavy/probing work destabilized this Windows host. Cross-compilation is not native
qualification. Windows ARM64 testing and the license/notice-release follow-ups
remain open independently.

## Evidence

Initial branch: `d05de9f` (Windows ARM64/Store increment committed by the user).
Official runtime archive: 10,027,600 bytes; SHA-256
`e1799098ebc054b370f6176a450f158720f297818c613e5dc99b92e2ec82346f`.
Main library: 24,538,024 bytes; SHA-256
`a27d21126db312aa8f02f3d5eaebe466e991f51f469882e6d0407d5a8b64afda`.
Static inspection identifies ELF64 little-endian AArch64, GLIBC_2.28,
GLIBCXX_3.4.22 and CXXABI_1.3.11. Archive/ELF evidence is in ignored
`.scratch/linux-arm-explorer`. No native code was executed by that inspection.

### Build and bounded verification

The portable native Windows cross-compiler uses Zig 0.15.2, pinned archive
SHA-256 `3a0ed1e8799a2f8ce2a6e6290a9ff22e6906f8227865911fb7ddedc3cc14cb0c`.
Debian Bookworm ARM64 graphics headers/libraries were checked against the signed
InRelease/Packages index: 47 package pins, including the EGL/libffi development
dependencies required by GLFW. No system installation, Docker or WSL was used.
The initial missing EGL header was fixed in the ignored portable sysroot only.
Both X11 and Wayland remain enabled. Commands and authenticated pins are retained
under `.scratch/linux-arm-explorer`.

The final GUI build passes with cgo enabled. Read-only ELF inspection confirms
EM_AARCH64 and required GLIBC versions no higher than 2.34, despite the compiler's
2.36 target. Dynamic dependencies are libGL, libX11, libwayland-client, libm and
libc. The test archive recommends Debian 12 or Ubuntu 24.04 desktop systems and
retains additional compiler/startup licenses; those system libraries are not
redistributed. No ARM64 executable was run here.

| Artifact | Bytes | SHA-256 |
|---|---:|---|
| `bin/picfetch-linux-arm64` | 50,073,480 | `c55d3a64d01d4037361ba7c5cd33a5411a71fd7eb40dc158a9fe95365350a501` |
| `bin/picfetch-linux-arm64.tar.gz` | 28,694,562 | `f3c0a29e11c073430a523d3fff42d7374800ef1214fb86ae5ead2427dce8a552` |

The archive's 11 entries were read back and compared by SHA-256 with their source
files; the executable has mode 0755. It includes the license, third-party notice,
privacy policy, launch/requirements README and compiler/startup attribution.
The final binary includes the permissive clustering replacement described in
[its plan](2026-09-11-permissive-clustering.md).

Focused runtime extraction tests cover both Linux architectures, including
missing/duplicate entries, links, traversal and cancellation. The BPF VM tests
pass for both syscall/audit ABIs; a deliberate allow-instead-of-deny mutation was
observed failing, restored, and the focused tests pass (0.357s). The actual Linux
isolation test binaries compile for amd64 and arm64. Installing/enforcing the
filter on Linux ARM64 remains a hardware acceptance task, not a Windows result.

Final gate uses native serial Make-equivalent commands after the earlier Make
shell failure. Full Linux/race and canonical shard validation remain CI work:
the shard tool explicitly rejects a Windows host. No top-level UI tests were
added. Qodana covers all 239 test files, and goimports covers all 553 Go files.
See the clustering plan for shared native tests, vet and executable smoke checks.

Routing/cost record: the lead owned all code, integration, review and fixes;
two existing bounded scouts checked archive/ABI and portable toolchain sources.
The toolchain scout also located exact runtime attribution. No implementation
or review was delegated. Compilers ran serially with GOMAXPROCS=2 and -p=1.
User hardware testing and CI acceptance keep this plan active. No commit or
publication was performed.
