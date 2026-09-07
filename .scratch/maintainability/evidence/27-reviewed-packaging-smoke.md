# Reviewed packaging smoke — 2026-09-07

Partial acceptance: all requested artifact builds and native macOS startup passed.
Windows/Linux graphical startup and Windows SDK/WACK are
tracked separately below; a build alone does not close ticket 27.

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
changes were made: [observed failure](27-windows-sdk-unavailable.jpg).
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

`Z:\smoke.cmd` launches the four refreshed Windows package variants in sequence
from distinct hash-suffixed Desktop filenames. The
[launcher](27-windows-package-smoke.ps1.txt) gives each process its own
USERPROFILE/APPDATA/LOCALAPPDATA directories, supplies the two generated
fixtures, and records window DPI, process exit and stdout/stderr. It preserves
the production app identity and requires separate visual observations; process
launch alone is not a rendering pass. The user was asked to leave the first
window open for inspection. No new graphical result is claimed yet.

Ubuntu boots to its desktop but also reports failed network activation and no
guest agent. A local `/private/tmp/picfetch-maintainability-linux.iso` contains
the actual Linux ARM64 executable, two generated fixtures and the
[launcher](27-linux-offline-run.sh.txt), which isolates XDG config/cache and
records linked libraries, available graphics metadata and exit status. The ISO
was mounted in the existing removable CD drive. Automated guest input repeatedly
triggered UTM capture instead of opening a terminal, so no Linux application
launch is claimed. The VM was shut down; the smoke ISO remains mounted. Its
previous image was `utm-guest-tools-latest.iso` from UTM's GuestSupportTools
application-support directory.
