# Reviewed packaging smoke — 2026-09-07

Partial acceptance: all requested artifact builds, native macOS startup, both
Windows ARM64 package startups and both Linux package smoke runs passed. Linux
amd64 ran through CPU emulation; Linux rendering used software OpenGL. Both x64 packages fail in the ARM64 VM: first context creation, then an
illegal instruction during drawing with an isolated matching runtime. Native x64 Windows startup and Windows SDK/WACK remain open; the user deferred
further Windows runs to native x64 systems. [Test checklist](../windows-test-todo.md).

## Inputs and builds

The disposable checkout `/private/tmp/picfetch-maintainability-27-package` was
created at `d608c17` and populated with the complete dirty working source. No
commit, tag, push, release or Store submission was performed. Native host:
macOS 26.6.2, darwin/arm64, Go 1.27.1, Fyne 2.8.0, Apple M5 Max, Retina display.

| Command, run in disposable checkout | Result | Retained output |
|---|---|---|
| `make install-tools` | exit 0; version-specific CLIs installed | [tools](27-install.txt) |
| `make package-mac` | exit 0; ARM64 `.app` | [macOS build](27-mac.txt) |
| `make package-windows package-windows-store package-linux` | exit 0; amd64 and arm64 for all three routes | [cross builds](27-cross.txt) |
| `file`, `go version -m`, macOS plist inspection | architecture, Store-only tag and macOS identity/version/associations passed | [artifact inspection](27-artifact-inspection.txt) |
| `go run ./scripts/msixstage -arch amd64 -exe bin/picfetch-microsoft-store-amd64.exe -out offline-windows/stage-x64`, corresponding arm64 command | exit 0; production Store identity, version 1.0.2.0, x64/arm64 manifests | stages retained in disposable checkout |

The pinned multiarchitecture indexes are in `packaging/tools.mk`. Actual selected
images report linux/arm64; warmup selects Go 1.27.1, container Fyne CLI 1.7.2.
Both tools and image identity are printed by the Make targets. Timestamps and
signatures are not claimed reproducible byte for byte.

## Native macOS package

[Launcher](27-native-mac.py.txt) ran the actual packaged executable with the
existing generated `01-alpha.png` fixture. It temporarily moved aside only the
production app's preferences and cache directories and restored them after
process completion. The production binary/app ID was unchanged. The 8192×6144
fixture rendered with the correct corner labels and checkerboard geometry:
[screenshot](27-macos-package-startup.jpg). Cmd+Q exited 0, the process log was
empty ([log](27-macos-process.txt)), and the launcher confirmed original state
restoration. No source/renderer diagnostic overlay was used for this run.

## Guard evidence

Executed packaging route guards cover both architectures for Linux, Linux debug,
Windows, Store and Windows debug, preserving Store/debug flags and rejecting
container, compiler and first-architecture copy failures. The new copy-failure
case first failed (5.217s), then passed. Negative verification rejected all
17 input/routing/provenance/failure mutations:
[results](27-negative.txt), [temporary-input/Go-overlay script](27-negative.py.txt).
It also exposed and strengthened a whole-log UID assertion to check each actual
container call.

The first common gate exposed a real Make help regression from adding a second
input file: grep printed filename prefixes and hid target names. Existing
`TestMakeCIFailuresFindsLatestCompletedRunOrAcceptsRunID` reproduced it natively;
`grep -hE` fixed it. All affected script suites then passed uncached:
`scripts/testshards` 4.345s, `scripts/msixstage` 6.489s,
`scripts/plistdoctypes` 0.376s. The second gate run printed PASS for every comparison test but the process was
OOM-killed: Docker cgroup `oom_kill=1`, memory peak 7,931,346,944 bytes, VM memory
8,319,213,568 bytes. No code assertion failed in that run. The unchanged full gate
then passed with `GOGC=25` in its Docker test container, supplied by this temporary
[wrapper](27-docker-gc-wrapper.sh.txt):
`env PATH="/private/tmp/picfetch-test-memory-tools:$PATH" make verify`.
All 665 UI tests (212/225/228) and every other race package passed; UI times were
334.273s/324.107s/258.171s. Native format/TUF/Qodana/vet/build passed.
[Gate results](27-verify.txt). No test inventory, concurrency or race checks were
reduced and no persistent Docker settings changed.

## Offline VM handoff

Windows 11 is an existing ARM64 UTM guest with no network or guest agent. The
user mounted the disposable checkout as `Z:`. `Z:\go.cmd` runs the retained
[PowerShell harness](25-windows-offline-run.ps1.txt), using the standard Go
`test2json` tool and [eight complete package executables](25-windows-offline-suites.json.txt)
compiled by [this script](25-windows-offline-build.sh.txt), Go 1.27.1,
Windows/arm64, CGO_ENABLED=0. It checks all 14 required native Windows/Store guards
for one run/pass and no skipped or failed cases. All executables copy to distinct
Desktop filenames beside the existing GL libraries. Results go into a timestamped
Desktop folder and copy back if the share permits writes. This is native test
execution only once actual results are returned; compilation is not a pass.

The staged Store files are ready for `Z:\pack.cmd`, which performs the existing
MakeAppx/sign/verify/WACK sequence in a disposable VM with the Windows SDK and
administrator access. Its [script](27-offline-wack.ps1.txt) refuses a pre-existing
PicFetch Store installation, retains output, and cleans its test certificate,
trust entry and test package. The user executed `Z:\pack.cmd`; its prerequisite check failed because the required
SDK/MakeAppx/SignTool/WACK tool set is unavailable. No package or certificate trust
changes were made: [observed failure](27-windows-sdk-unavailable.redacted.md).
The official SDK 10.0.26100.9169 offline ISO was downloaded from Microsoft into the
shared checkout for preparation; no SDK installation has run.
The [installer inspection](27-sdk-installer-inspection.txt) records its source,
hash and WACK payload conditions: executable/native components are conditioned
on x86 or amd64 hosts, so availability on this ARM64 VM remains uncertain.
The host-side read-only ISO mount was ejected after inspection.

The completed first Windows run was retrieved after making the temporary share
writable and restarting Windows. It failed an existing Linux XDG-list parser
test inside the full wallpaper package; the Windows-specific wallpaper guards
passed. The retained [failure events](25-windows-red/wallpaper.json) exposed use
of the execution host's path-list separator for the colon-delimited XDG format.
That parser is corrected and the complete native Mac wallpaper suite passes
(0.391s). The revised offline run collected every package's outcome before
reporting failure; seven packages passed, and the unchanged updater binary then
passed from a local working directory after a WebDAV directory-restoration
failure. All eight accepted package runs and all 14 required guards pass without
skips: [native Windows evidence](25-native-guards.md#offline-windows-execution).
The user handles VM keyboard input because automated input drops keystrokes.
The package artifacts recorded above precede this Linux XDG parser follow-up;
graphical Windows/Linux startup and WACK remain separate open requirements.

After the follow-up common gate passed, `make package-mac package-windows
package-windows-store package-linux` rebuilt all seven artifacts from the
verified source in the same disposable checkout (exit 0). Retained
[build output](27-package-followup.txt) and [fresh metadata/hashes](27-artifact-followup-inspection.txt)
confirm the expected architectures, Store-only tag, CGO setting and production
macOS identity/version. The new Mac bundle build number is 450; the working
repository's FyneApp.toml was not changed by packaging.

## Refreshed Windows graphical package results

`Z:\smoke.cmd` attempted all four refreshed variants in sequence from distinct
hash-suffixed Desktop filenames. The
[launcher](27-windows-package-smoke.ps1.txt) gives each process its own
USERPROFILE/APPDATA/LOCALAPPDATA directories, supplies the generated fixtures,
and records window DPI, process exit and stdout/stderr. It preserves production
application identity. The returned [process records](27-windows-package-results/processes.json)
and [ordered transcript](27-windows-package-results/transcript.txt) identify the
actual outcomes:

| Package | PID | Window DPI | Exit | Native result |
|---|---:|---:|---:|---|
| Ordinary amd64 | 10164 | 0 | 2 | OpenGL context creation failed before a visible window |
| Ordinary arm64 | 6672 | 96 | 0 | Alpha/Beta navigation and comparison interactions passed; clean quit |
| Store amd64 | 7560 | 0 | 2 | OpenGL context creation failed before a visible window |
| Store arm64 | 1900 | 96 | 0 | Alpha rendered; clean quit |

All four recorded executable hashes exactly match the
[fresh artifact inspection](27-artifact-followup-inspection.txt). Both ARM64
processes have empty stdout/stderr. This validates the Store-tagged ARM64
executable, not MSIX installation or WACK.

The first visible window belonged to **ordinary ARM64**. It rendered
[Alpha](27-windows-first-alpha.redacted.md), then [Beta](27-windows-first-beta.redacted.md) after
the user's Right-key navigation, preserving all four landmarks, Beta's magenta
marker and the 8192x6144, 2/2 title. The user then exercised
[comparison pan, zoom, swipe and detail](26-native-comparison-smoke.md#windows-continuation-with-user-operated-input).
The later [empty drop area](27-windows-second-empty.redacted.md) was in this same
process. The earlier visual-order attribution of that image to a new package
was incorrect: the launcher had already skipped failed amd64 startup before
any visible image window. The empty frame does not demonstrate an ARM64
startup or fixture-loading defect. After that process closed, the next visible
window was **Store ARM64**, rendering [Alpha](27-windows-third-alpha.redacted.md) with
all four corner markers and its 8192x6144, 1/2 title.

The ordinary amd64 [failure log](27-windows-package-results/windows-amd64/stderr.txt)
and Store amd64 [failure log](27-windows-package-results/microsoft-store-amd64/stderr.txt)
both report `APIUnavailable: WGL: The driver does not appear to support OpenGL`
from Fyne window creation, followed by a nil-window panic in Fyne's Windows
theme application. These are failed native startup attempts, not passing
coverage. The [read-only environment probe](27-windows-graphics-details.ps1.txt),
run via `Z:\details.cmd`, returned [driver/DLL/monitor records](27-windows-graphics.json):
Windows 11 Pro ARM64, VirtIO GPU DOD driver 22.7.38.43, 1728x1043 at 100% scale,
and **ARM64-only** Desktop OpenGL/EGL/GLES libraries. No graphics compatibility
package or renderer override is reported. This supports the hypothesis that
the x64 applications lack a matching usable GL runtime in this VM. It does
not prove the runtime search path by itself. No existing library was replaced.

The next isolated attempt uses `Z:\x64.cmd` and the
[retained launcher](27-windows-x64-smoke.ps1.txt), copying the unchanged x64
executables and one matching `opengl32.dll` into new per-variant Desktop
subdirectories. The software implementation is the mmozeiko/build-mesa
[26.2.2 llvmpipe release](https://github.com/mmozeiko/build-mesa/releases/tag/26.2.2);
its [maintainer documentation](https://github.com/mmozeiko/build-mesa) describes
static dependencies and app-local DLL use. Archive SHA256 matched the release
API digest; extracted PE machine is amd64. [Provenance and hashes](27-windows-x64-runtime.json).
No installer or system-wide graphics change is involved. The launcher records
package identity before waiting for user close, plus loaded graphics modules
where accessible. Actual rendering/exit results are still pending.

The first isolated x64 attempt (20260907-133956) stopped before process launch:
the WebDAV share refused to read the 58,609,152-byte DLL as exceeding its
permitted file size. [Raw failure](27-windows-x64-share-limit/FAILURE.txt) and
[transcript](27-windows-x64-share-limit/transcript.txt) retain this transfer
failure, which is not a new PicFetch failure. The revised launcher copies a
21,995,876-byte ZIP, verifies its SHA256, expands it locally and verifies the
unchanged DLL SHA256 before launch. A host ZIP round-trip hash check passed.
Only JSON/text results are returned to the share; generated fixtures, DLLs,
executables and profile caches stay in the isolated local result folders.
No registry, share-limit or system-library setting was changed.

The compressed-transfer retry (20260907-134213) passed both hash checks and
loaded each app-local amd64 `OPENGL32.dll`, confirmed in the returned module
records. Both executables created a window at DPI 96, then exited 2 with
`Exception 0xc000001d` during the initial `glDrawArrays` call from Fyne's
rectangle painter. The user observed the windows closing immediately. These
are failed rendering attempts. [Raw records and crashes](27-windows-x64-illegal-instruction/processes.json),
[ordinary loaded module](27-windows-x64-illegal-instruction/windows-amd64/graphics-modules.json),
[ordinary crash](27-windows-x64-illegal-instruction/windows-amd64/stderr.txt),
[Store crash](27-windows-x64-illegal-instruction/microsoft-store-amd64/stderr.txt).
The earlier context-unavailable error is gone, but native x64 acceptance is
still open.

Ranked, falsifiable hypotheses: (1) Mesa's detected CPU features produce an
instruction the ARM64 guest's x64 emulator cannot execute; limiting the same
runtime to SSE2 should avoid the failure. (2) A Mesa build/code-generation bug
would persist with that limit but change with a different renderer build.
(3) An application/Fyne-specific rendering issue would persist in the app while
a minimal draw with the same runtime succeeds. The first bounded probe is
`Z:\x64-sse2.cmd`, selecting the documented
[`GALLIUM_OVERRIDE_CPU_CAPS=sse2`](https://docs.mesa3d.org/drivers/llvmpipe.html)
only in the child process. The app/DLL hashes and profile isolation are
unchanged; `GALLIUM_DUMP_CPU=1` records detected capabilities. No persistent
CPU/VM setting changes. Results are pending.


Ubuntu boots to its desktop but also reports failed network activation and no
guest agent. A local `/private/tmp/picfetch-maintainability-linux.iso` contains
the actual Linux ARM64 executable, two generated fixtures and the
[launcher](27-linux-offline-run.sh.txt), which isolates XDG config/cache and
records linked libraries, available graphics metadata and exit status. The ISO
was mounted in the existing removable CD drive. Automated guest input repeatedly
triggered UTM capture instead of opening a terminal, so no Linux application
launch is claimed. The VM was shut down; the smoke ISO remains mounted. It was then refreshed
while Ubuntu was stopped with the final verified ARM64 executable and a
hash-reporting launcher; [media hashes](27-linux-media-followup.txt) distinguish
it from the earlier ISO. No new native launch is claimed. Its
previous image was `utm-guest-tools-latest.iso` from UTM's GuestSupportTools
application-support directory.


## Completed non-Windows follow-up

The refreshed actual macOS package (build 450, unchanged production ID and
SHA256 matching the fresh inspection) rendered Alpha with all four landmarks
and quit via Cmd+Q with exit 0. Its app log is empty and the launcher confirmed
restoration of the user's original preferences/session. Retained
[screenshot](27-macos-followup-startup.jpg),
[result](27-macos-followup-result.txt), [log](27-macos-followup-process.txt),
[launcher](27-native-mac-followup.py.txt).

Both refreshed Linux executables then passed real production GL rendering,
Alpha/Beta navigation and clean quit with exit 0 and empty logs in isolated
Debian 12 desktops. ARM64 additionally passed comparison side-by-side and
swipe output plus quit while comparison was open. The ARM64 process is native
to Docker's ARM64 Linux VM; amd64 uses CPU emulation. Mesa llvmpipe software
rendering and Xvfb 96 DPI are explicitly recorded. This supplies Linux target-OS
package startup evidence with those limits; no hardware-accelerated GPU claim
is made. [Commands, screenshots, environment and terminal results](27-linux-packaged-smoke.md).
The unused Ubuntu UTM smoke ISO remains available for later physical/display
coverage. Required remaining acceptance is the user-deferred native x64 Windows
startup and Windows packaging/WACK checklist. No full suite was rerun for these
evidence-only additions; the final verified production source is unchanged.
