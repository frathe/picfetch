# Windows Defender Controlled Folder Access — update-apply rework

Status: implemented on 2026-08-30 (branch `windows-cfa-update-apply`, 9 tasks,
commits `6455432..8dfa258`); not yet executed on real Windows and not yet
pushed. Full verification remains the final review gate.

Source todo: formerly under `todos.md` → `## TODO` ("the two step file
replacement in windows triggers windows defender every single time"); moved
to `## Done` → `#### Bugfix` on 2026-08-30, alongside three new `## TODO`
entries for what is still open.

## The bug, and how it was known

A user reported that PicFetch's in-app update never applied on Windows. The
reported Windows Security protection-history entry named `cmd.exe` and
`%userprofile%\Music\` as the blocked action. The pre-branch apply path
(`internal/update/apply_windows.go`, dispatched from `apply.go`'s
`runtime.GOOS` switch) wrote a generated `<exe>.apply.cmd` beside the
executable — one that waited for the old PID, copied the staged binary over
the destination, deleted the staged copy, optionally started the new
executable, and deleted itself — and ran it through `cmd.exe`.

That script host is the actual cause, not incidental to it. Controlled
Folder Access (CFA) is Microsoft Defender's ransomware-protection feature:
writes into a fixed list of user folders (Documents, Pictures, Music,
Videos, Desktop, and a few more) are denied unless the writing application
is on CFA's allow list. `cmd.exe`, `powershell.exe`, `wscript.exe`, and the
other built-in interpreters and script hosts are permanently excluded from
that allow list regardless of their own Microsoft signature — trusting the
host would mean trusting whatever script it happens to be handed, which
defeats the point of the feature. So the block reported by the user was not
a false positive or a one-off misconfiguration: every CFA-protected install
of PicFetch was going to hit it, every time, on every update.

## What changed

The swap now happens **in-process** — no script host, no `cmd.exe`, no
second process at all past the relaunch.

- `internal/update/swap.go` — `swapBinary` renames the running executable to
  `<dest>.old` (the one replacement Windows allows on a running image),
  copies the staged binary over `dest`, SHA-256-verifies the copy against
  the stage (`sameContents`, guarding against a filter driver that accepts a
  write and silently rewrites or truncates it), and restores `<dest>.old` on
  any failure past the rename (`restoreBinary`). `<dest>.old` deliberately
  survives a successful swap: it is still this process's own running image
  and cannot be deleted from here.
- `internal/update/apply_windows.go` — `applyWindows` is now a thin call
  into `swapBinary`. The relaunch (`relaunchWindows` /
  `windowsRelaunchCommand`) starts the freshly installed executable directly
  — no arguments, since `main.go` treats every bare argument as a file to
  open — carrying the installing process's PID in
  `PICFETCH_UPDATE_AWAIT_PID` in the child's inherited environment.
  `CREATE_NO_WINDOW` is deliberately not set here, unlike the console
  helpers in `trash`, `wallpaper`, `filepicker`, and `clipboard`: this
  starts `picfetch.exe`, which release builds link as GUI-subsystem
  (`fyne-cross` without `-console`) and therefore has no console for the
  flag to suppress.
- `internal/update/await.go` — `update.CleanupPredecessor`, called from
  `main.go` before `app.NewWithID`, waits for the PID named in
  `PICFETCH_UPDATE_AWAIT_PID` (bounded at 15s — `awaitPredecessorTimeout`),
  because a relaunched PicFetch must not touch preferences before the old
  process has finished flushing its own (`Apply` runs inside Fyne's
  `OnStopped`, and Fyne saves preferences immediately after that callback
  returns). It then unsets the variable — so nothing PicFetch spawns later
  inherits and re-waits on a recycled PID — and sweeps `<dest>.new` /
  `<dest>.apply.cmd`, the two files a pre-2026-08-30 update could have left
  beside the executable. `SweepBackup`, in the same file but called
  separately from `internal/ui` startup once the Fyne app cache exists,
  removes `<dest>.old`. It is skipped when the last recorded apply failure
  has `Op == "restore"`: in that state `dest` may be missing or truncated
  and the backup is the user's only intact executable, so sweeping it would
  turn a recoverable failure into data loss.
- `internal/update/applyerr.go` (+ `applyerr_windows.go` /
  `applyerr_other.go`) — `ApplyError` wraps a failed step with its `Op` and
  `Path`; `ClassifyApplyError` turns the underlying error into a
  `FailureReason` (`ReasonAccessDenied`, `ReasonVirusBlocked`,
  `ReasonSharingViolation`, `ReasonUnknown`) for the next-launch report,
  preferring a Windows errno match (`ERROR_ACCESS_DENIED`,
  `ERROR_VIRUS_INFECTED`/`_DELETED`, `ERROR_SHARING_VIOLATION`/
  `ERROR_LOCK_VIOLATION` — the last two are declared locally because they
  exist in the standard library only under the non-importable
  `internal/syscall/windows`) over the portable `fs.ErrPermission` fallback
  used off Windows.
- `internal/ui/autoupdate/applyfailure.go` — `ApplyFailure` is cached into
  the Fyne app cache under `updatefailure.json`, because `ApplyStagedUpdate`
  runs from the stopped callback where there is no window left to report
  into. `SaveApplyFailure` / `LoadApplyFailure` / `ClearApplyFailure` carry
  it forward; the raw OS error text lives only in `Detail`, kept out of the
  dialog because an errno-flavoured sentence explains nothing and cannot be
  translated.
- `internal/ui/autoupdate.go` — `sweepUpdateBackup` (called from
  `run.go`'s `SetOnStarted`, before `maybeShowWhatsNew`) loads the cached
  failure, sweeps `<dest>.old` unless the failure says `Op == "restore"`,
  and hands the loaded record to `maybeShowUpdateFailure` — which explains
  the failure with `dialog.NewCustomConfirm` (version, reason-specific
  sentence, path, an "Open download page" button) and clears the cache
  entry. The ordering is deliberate and load-bearing: reporting clears the
  record, so a reporter that ran before the sweep would let the sweep take
  the user's last working binary out from under a state it hadn't reported
  yet.
- `.github/workflows/ci.yml` — a new `windows-test` job (`windows-latest`)
  runs `go test ./internal/update/... ./internal/ui/autoupdate/...` without
  `-race` (no C toolchain on that runner). The existing `ubuntu-latest` job
  gained a `GOOS=windows` cross-build and `go vet` step over `./internal/...`
  (the root package still fails to cross-compile for an unrelated,
  pre-existing Fyne/glfw reason).

## Decisions and corrections made along the way

Recorded here because they shaped the final shape and are easy to
re-relitigate by accident:

- **The `.apply.cmd` string was not fully eradicated, and that is correct.**
  `apply_windows_test.go` keeps one reference to it as a deliberate
  regression guard, and `await.go`'s `sweepLeftovers` still names it, because
  a user upgrading from a pre-2026-08-30 build can have that file sitting
  beside their executable. The actual invariant is "no code writes or
  executes a `.cmd` file", not "the string never appears in the tree" —
  `grep -rn 'apply\.cmd' .` was checked against that narrower invariant, not
  a bare string count.
- **`AwaitPIDEnv` needed an explicit unset**, not just a wait: without it,
  every process PicFetch spawns after a relaunch (helper commands, the next
  update's own relaunch) inherits the variable and would wait again on
  whatever process now happens to hold that recycled PID.
- **The `.old` sweep could not live in `CleanupPredecessor`.** The
  restore-in-progress guard needs the cached `ApplyFailure` record, which
  lives in `app.Cache()` — and `CleanupPredecessor` runs before
  `app.NewWithID` even exists. The sweep was split out into `SweepBackup`
  and moved to `internal/ui` startup, where the app object is available.
- **A successful apply must clear a stale failure record itself**, not rely
  solely on the next-launch dialog to do it. `maybeShowUpdateFailure` is
  deliberately not version-gated the way `maybeShowWhatsNew` is (a
  version-gated dialog that never becomes reachable again would strand an
  `Op == "restore"` record forever, permanently vetoing the backup sweep) —
  so `ApplyStagedUpdate` clears the record on its own success path instead
  of leaving that job to a dialog that might not fire.
- **`CREATE_NO_WINDOW` was deliberately dropped**, not carried over from the
  script-based relaunch. The four other console-helper call sites in this
  codebase (`trash`, `wallpaper`, `filepicker`, `clipboard`) spawn
  console-subsystem tools and need the flag to suppress a flashing window;
  this call spawns `picfetch.exe` itself, which is GUI-subsystem in release
  builds and has no console for the flag to suppress. On
  `package-windows-debug` (console-subsystem), keeping the flag would have
  suppressed the very console that build exists to show.

## Verification performed, and its limits

Every task in the plan (`plans/2026-08-30-windows-cfa-update-apply.md`) went
through an implement → controller-verify → independent-review → fix-round
loop; the full ledger is in this session's `progress.md`. What was actually
run:

- `go build ./...`, `go vet ./internal/update ./internal/ui/autoupdate`, and
  the full `internal/update` / `internal/ui/autoupdate` suites, on macOS.
- `GOOS=windows GOARCH=amd64 go build|vet ./internal/...` and
  `GOOS=windows GOARCH=arm64 go build ./internal/...` — cross-compilation
  only.
- Mutation checks on the reviewer's own terms for the swap ordering, the
  timeout clamp arithmetic (`waitMilliseconds`; a raw conversion of a
  negative duration would have produced roughly 49 days, and `0xFFFFFFFF`ms
  converts back to exactly `syscall.INFINITE`), the restore-branch failure
  path, and the sweep's `Op`-based guard.

**Nothing on this branch has been executed on real Windows.** There is no
Windows machine available in this environment, and the new `windows-test` CI
job has never run — the branch is deliberately unpushed at the user's
request, pending their own review. `await_windows.go`'s
`syscall.OpenProcess` / `WaitForSingleObject` path in particular has zero
runtime evidence behind it beyond the cross-compiler accepting it.

## The honest limit

Windows releases are **not Authenticode-signed** —
`.github/workflows/release.yml` runs no `signtool` step. Controlled Folder
Access does not only care what kind of program is writing (script host
versus ordinary executable, which is what this branch fixes); it also
weighs the writing program's own signature and reputation. An unsigned
`picfetch.exe` writing into a CFA-protected folder can therefore still be
denied — and if that happens, the block would name `picfetch.exe` itself
rather than `cmd.exe`.

That is precisely why this branch reports the failure to the user on the
next launch instead of the old behavior of failing silently into
`%TEMP%\picfetch-update.log`. The reporting half of this work is useful
regardless of whether the signature problem is ever fixed: it turns a silent
failure into a visible one with a manual way out (the releases page button).
The remaining real fix — Authenticode signing the release build — is out of
this branch's scope and is recorded in `todos.md` → `## TODO`.

## Outstanding follow-ups (see `todos.md` → `## TODO`)

1. Authenticode-sign the Windows release build (Azure Trusted Signing or a
   purchased certificate) — the actual fix for reputation-based CFA and
   SmartScreen blocks, as opposed to the script-host block this branch
   fixes.
2. Manual Windows verification: install a build into a CFA-protected folder,
   trigger Perform update, and record which of the two outcomes happens —
   the swap now succeeds, or Defender still blocks it and the new dialog
   correctly names `picfetch.exe`.
3. Native-German review of two sentences in the CFA failure message
   (`translations/de.json`) that a code reviewer flagged as reading like
   translation rather than native composition; suggested rewordings are
   recorded in `todos.md` for the file's owner, a native speaker, to decide.
