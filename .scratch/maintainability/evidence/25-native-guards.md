# Native guard execution, 2026-09-07

Host: macOS 26.6.2 (25G83), Apple M5 Max, darwin/arm64, Go 1.27.1. Native C/GL toolchain available. Desktop mutations are stubbed by the named platform tests.

`go run ./scripts/nativeguards -suite macos -capture /private/tmp/picfetch-maintainability-25-macos.json` — exit 0. The runner first listed each package's build-selected tests, then executed the full packages with uncached verbose JSON output. Root main's actual Cocoa-linked `TestInstall_GraftsOntoGLFWsDelegate` ran and passed. Actual open-with Unicode delivery, native chooser path transport, display snapshot and poller cancellation guards also ran and passed; no test skipped. Package times: main 0.698s, openwith 0.366s, displays 0.816s, winpos 1.053s, filepicker 0.257s. [Raw events](25-macos-events.json).

`go run ./scripts/nativeguards -suite store -capture /private/tmp/picfetch-maintainability-25-store.json` — exit 0. Both inventory and full execution explicitly selected `-tags=microsoftstore`. `TestStoreManaged_MicrosoftStoreBuildIsTrue` and the updater transaction guard ran and passed; no test skipped. Package times: distribution 0.233s, autoupdate 0.539s. [Raw events](25-store-events.json).

The runner contract suite passes under race instrumentation (1.473s). Twelve negative overlays reproduce wrong-platform admission, wrong Store selection, substituting the package-level open-with test for the Cocoa-linked root test, omitted inventory checks, missing run/pass evidence, skipped required subcases, malformed events, package/process failure, discarded raw output and mismatched execution tags. CI now invokes the Windows, Store and macOS suites and uploads raw events even on failure. Existing Linux partitions and reusable release gates remain in place.

Equivalent online reproduction commands on Windows from the working tree:

```powershell
go run ./scripts/nativeguards -suite windows -capture native-guards-windows.json
go run ./scripts/nativeguards -suite store -capture native-guards-store.json
```

The Windows suite includes the ticket 06 picker serializer and ticket 08 clipboard decoder guards. No workflow or artifact was published during this local work.

## Offline Windows execution

Windows 11 ARM64 (10.0.26100), Windows PowerShell 5.1.26100.9168. The VM has no
network or Go installation. The retained [build script](25-windows-offline-build.sh.txt)
cross-compiled full package test executables and standard Go `cmd/test2json` with
Go 1.27.1, GOOS=windows, GOARCH=arm64, CGO_ENABLED=0. The user executed those
binaries on Windows through [the harness](25-windows-offline-run.ps1.txt).
Executables use distinct Desktop filenames beside the existing libraries.
These are native OS/PowerShell tests with desktop mutations stubbed; they do
not supply graphical application startup or WACK evidence.

The first run, 20260907-115026, was red in
`TestHostSchemaEnv_DropsSandboxInjectedSchemaSources`: the Linux XDG parser
used the execution host's path-list separator, so Windows treated the colon
list as one sandbox path. [Original events](25-windows-red/wallpaper.json).
`hostDataDirs` now uses the XDG colon delimiter explicitly. The existing guard
and full wallpaper package then passed on both macOS (0.391s) and Windows.
No assertion was weakened or skipped.

Run 20260907-121327 passed seven full packages. Its updater run aborted when
Go's `testing.Chdir` cleanup could not restore a directory handle into the
WebDAV share. The unchanged updater binary passed from a new local Desktop
working directory in run 20260907-122203. It creates all of its own fixtures;
no repository testdata was omitted. The failure and partial output remain in
[the second run](25-windows-r2/update.json).

| Windows package run | Top-level passes | Seconds | Events |
| --- | ---: | ---: | --- |
| wallpaper | 23 | 0.175 | [JSON](25-windows-r2/wallpaper.json) |
| clipboard | 16 | 1.551 | [JSON](25-windows-r2/clipboard.json) |
| filepicker | 14 | 0.685 | [JSON](25-windows-r2/filepicker.json) |
| update, local working directory | 99 | 2.116 | [JSON](25-windows-update-local/update.json) |
| ui/autoupdate | 50 | 0.505 | [JSON](25-windows-r2/ui-autoupdate.json) |
| distribution | 1 | 0.054 | [JSON](25-windows-r2/distribution.json) |
| microsoftstore distribution | 1 | 0.049 | [JSON](25-windows-r2/store-distribution.json) |
| microsoftstore ui/autoupdate | 50 | 0.458 | [JSON](25-windows-r2/store-ui-autoupdate.json) |

All 254 top-level test executions passed; no test or subtest skipped in these
eight accepted package runs. Inventories and raw stderr are retained beside
each JSON file. A [temporary Go overlay](25-windows-replay-test.go.txt) replays
the returned inventories/events through the production `suiteFor` and
`validateEvents` functions, additionally requiring exactly one terminal package
pass. Both ordinary Windows and Store suites passed (0.286s), with all 14
required guards running and passing once without skipped descendants:
[validator output](25-windows-replay.txt).

The [common gate](25-windows-followup-verify.txt) for the XDG parser follow-up
passed (exit 0): formatting/TUF/Qodana, native vet/build, all 665 Linux UI tests
and remaining race packages. UI shard times: 347.975s / 333.033s / 267.607s.
The temporary Docker wrapper uses GOGC=25 with unchanged test inventory,
concurrency and race instrumentation, as in the prior successful ticket 27 gate.
`GOOS=windows GOARCH=amd64 go vet ./internal/...` also passed (exit 0).
Tickets 06, 08 and 25 are resolved. MA-017 still needs ticket 26's remaining
native renderer evidence.
