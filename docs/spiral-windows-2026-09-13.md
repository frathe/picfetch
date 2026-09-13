# Spiral v1.1.1 on Intel Windows, September 13, 2026

## Finding

The v1.1.1 image-tunnel shader fails to compile on this ThinkPad P16s Gen 2's
Intel Iris Xe driver, 32.0.101.7088. The new helper parameter `active` causes:

```text
ERROR: 0:120: 'active' : reserved word
ERROR: 0:120: 'active' : syntax error syntax error
```

The running PicFetch process loads Intel's `igxelpicd64.dll`. Windows reports
both Intel and the NVIDIA RTX A500 (driver 596.86) healthy. No PicFetch-specific
Windows GPU preference was found. No driver, GPU preference, or installed
executable was changed during the investigation.

A hidden WGL context on the Intel adapter compiles these fragment sources:

| Source | Result |
| --- | --- |
| Unchanged v1.1.0 desktop shader | Pass |
| Unchanged v1.1.1 desktop shader | Fail: `active` |
| v1.1.1 with only the parameter and its use renamed to `enabled` | Pass |

The context reports OpenGL 4.6.0 and GLSL 4.60, both build 32.0.101.7088.
Fyne caches failed shader compilation and skips that shader's drawing, so the
failure prevents the entire Spiral background from rendering, even without
any source images. This establishes a release-introduced driver compatibility
regression; it does not require contention between the two adapters.

`active` is reserved in [GLSL 1.40 section 3.6](https://registry.khronos.org/OpenGL/specs/gl/GLSLangSpec.1.40.pdf),
but is absent from the reserved list in the shader's declared
[GLSL 1.10](https://registry.khronos.org/OpenGL/specs/gl/GLSLangSpec.1.10.pdf).
The evidence therefore supports avoiding a compiler-sensitive identifier,
not claiming that the shader violates the declared 1.10 specification.
The NVIDIA backend and other Intel driver versions were not tested.

Existing Go UI tests use Fyne's test driver and do not compile these strings
with Intel's native GLSL compiler. The recorded pre-release native Spiral
qualification was on macOS. Those checks therefore did not cover this failure.

## Release identity

The desktop executable's SHA-256 is byte-identical to `picfetch.exe` inside
the official v1.1.1 Windows amd64 ZIP:

```text
Executable: 71e24978f64a7223cb7274acbaa714c46a535521051242194ac13db14e790ded
ZIP:        07220fe45abf46f3ea26853837391f70d252732d9ea0456e2c1505c4c1a78cfc
```

The ZIP digest also matches GitHub's release metadata. Build information names
revision `5fe09a950d63bcbec65824c85718bc6a3c306967`, tags
`no_emoji,release,migrated_fynedo`, and Fyne 2.8.0. The build's `vcs.modified=true`
is present in the official artifact too; it is not evidence of local corruption.

The shader change arrived in `a0023ab` (Feature/tunnelview, PR #20), between
v1.1.0 and v1.1.1. Fyne, go-gl, GLFW, and the Windows packaging-tool pins did
not change. `no_emoji` was added to all application build routes, rather than
only the release route. This is a source compatibility regression present in
the published release, not a damaged download or demonstrated packaging defect.

## Correction and verification

`internal/ui/spiral/shader.go` renames the helper parameter to `enabled` in
the shared desktop/ES shader body. No uniform names or rendering calculations
change. `TestShaderAvoidsNewerReservedIdentifiers` rejects the newer
`common`/`partition`/`active` group as standalone tokens in either source,
ignoring comments and preserving identifiers such as `traveller0Active`.
The regression failed on both original variants before the correction.

Commands run on Windows:

```powershell
go test -tags no_emoji ./internal/ui/spiral -run '^Test(Shader|NewShader)' -count=1
go vet -tags no_emoji ./internal/ui/spiral
go tool goimports -local github.com/frathe/picfetch -l internal/ui/spiral/shader.go internal/ui/spiral/shader_test.go
powershell.exe -NoProfile -File .scratch/spiral-windows-20260913/probe.ps1
```

Focused tests pass (0.398s); vet and formatting checks pass. The native probe
again rejects the release shader and accepts the corrected source on Intel.
This observes native shader compilation, not linking, full-window rendering,
or motion. The host has no configured native C compiler (`CGO_ENABLED=0`),
so the diagnostic application was cross-built in Docker as described below.

The full Spiral package runs but fails the existing FPS allocation test:
16 allocations per update versus a maximum of 9. Its German-locale run logs
missing translations, including the FPS text. Repeating that test using an
overlay with the unchanged v1.1.1 shader and tests reproduces the same failure;
the allocation difference is independent of this fix. Other Spiral tests pass.
The first `make verify` stops in `check-test-platform` because Git Bash cannot
fork; Docker was initially unavailable too.

After Docker started, its native Linux/x86_64 daemon reported 16,592,297,984
bytes (15.45 GiB), below the Makefile's required 16 GiB for the full race suite.
An isolated Linux checkout with the current patch, LF line endings and the
usual Ubuntu 24.04 build dependencies ran these checks in an 8 GiB container:

```text
make verify-build                                              exit 0
go test -race -tags no_emoji -count=1 ./internal/ui/spiral       ok 1.611s
```

Formatting, TUF/Qodana metadata, generated assets, updater notices, full vet
and full package builds pass. All Spiral tests pass under the standard
English Linux locale. The full repository race gate remains unverified;
its memory requirement was not lowered. The snapshot/container runner and
complete logs are retained alongside `linux-status.txt` in the evidence folder.

GoLand inspected both changed Go files with weak warnings included. The
existing duplicate test fragments are covered by the exact shader-test path
already listed in `qodana.yaml`; the correction adds no new duplicate fragment.

## Windows diagnostic executable

The corrected executable is
`.scratch/spiral-windows-20260913/windows-build/picfetch-fixed.exe`.
It is an unsigned, unpackaged Windows amd64 GUI executable (PE32+, subsystem 2),
44,104,192 bytes, built with the existing Makefile `build` recipe and the pinned
Windows fyne-cross image/Zig compiler flags. The recipe uses `no_emoji`,
`-trimpath`, and `-ldflags="-s -w"`; it does not reproduce the release's
packaging, signing, or additional release tags. The ordinary application
identity is retained. No installed executable was replaced.

```text
SHA-256: 7c2a0eea7eed64695395ea286a71e572752f994639bdd0b2433295b0b5b38f6e
```

The lead independently checked the output digest, Go build information and
source digest against the build records. Shader and embedded-vector hashes
were unchanged during the build, which mounted source read-only. The executable
runs `--help` on this Windows host, prints the expected usage, and exits 0.
Its native Spiral window has not yet been visually tested. Exact build commands,
toolchain/PE metadata and the startup smoke log are beside the executable.

Raw probe source, extracted shaders, compiler output, baseline overlay,
test logs, and the downloaded release ZIP remain in
`.scratch/spiral-windows-20260913/`. Work used one read-only release-history
scout (spawn budget/actual: 1/1), followed by a Windows artifact build assigned
to the same helper; the lead owned native diagnosis, correction and review.
One full `make verify` attempt was made, followed by the Docker checks above.
No commit or push was made.
