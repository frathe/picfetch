# Windows Update Apply Without cmd.exe — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use `superpowers:subagent-driven-development` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Windows `cmd.exe` apply script with an in-process binary swap performed by `picfetch.exe` itself, and when the swap is still refused (Controlled Folder Access, antivirus), record why and explain it to the user on the next launch with a link to the download page.

**Architecture:** `applyWindows` stops writing a `.cmd` file and stops spawning `cmd /C`. Instead it renames the running executable to `<dest>.old` (Windows permits renaming a running image), copies the staged binary into `<dest>`, verifies the copy byte-for-byte, and rolls the rename back on any failure. Relaunch starts the new executable directly with `PICFETCH_UPDATE_AWAIT_PID` in its environment; the new process waits for the old PID to exit before `app.NewWithID` runs, then deletes the leftover `<dest>.old`. Failures are classified from the underlying `syscall.Errno` and stored in the Fyne app cache exactly like the existing What's-New payload, and `SetOnStarted` reports them.

**Tech Stack:** Go 1.26.7, Fyne v2, stdlib `syscall` (no new module dependency), GitHub Actions.

**Spec:** this file — decisions were taken in the 2026-08-30 brainstorming session recorded under "Decisions" below.

---

## Problem statement

Windows Defender's **Controlled Folder Access** (CFA, German UI: *Überwachter Ordnerzugriff*) blocked the update on 2026-08-30 01:45. The protection-history entry names:

- Blocked app or process: `cmd.exe`
- Protected folder: `%userprofile%\Music\`
- Blocked by: *Überwachter Ordnerzugriff*
- "Ihr Administrator hat diese Aktion blockiert" — the rule is enforced by policy

The update did **not** apply. The affected installation lives inside a CFA-protected user folder.

Two independent reasons the current design triggers this:

1. **`cmd.exe` is never trusted by CFA.** CFA maintains its own trust list; script hosts (`cmd.exe`, `powershell.exe`, `wscript.exe`) are excluded from it regardless of their Microsoft signature. Any write they attempt into a protected folder is blocked by design, so no amount of quoting or flag tuning in `windowsApplyScript` can fix this.
2. **The pattern itself is the malware template.** Drop a script beside a binary, wait for the parent process to die, swap the binary, self-delete. Behavioural heuristics and ASR rules score exactly that shape.

Removing `cmd.exe` moves the write from a permanently untrusted LOLBIN to `picfetch.exe`, which CFA judges on signature and reputation. **That is an improvement, not a guarantee** — releases are not Authenticode-signed (`.github/workflows/release.yml` runs no `signtool`), so an unsigned `picfetch.exe` writing into a protected folder may still be denied. Hence the second half of this plan: detect the denial, classify it, and tell the user what to do.

## Decisions (from brainstorming, do not relitigate)

| Question | Decision |
|---|---|
| Fix shape | Both: perform the in-process swap, verify it worked, and report a precise reason when it did not. |
| Where the error is shown | **Next launch only.** No pre-flight probe. `Apply` runs from `SetOnStopped` with no window left, so the failure is cached and reported by `SetOnStarted`. |
| What the dialog offers | Explanation naming Controlled Folder Access + a button opening the GitHub releases page for a manual install. |
| CI | Cross-compile + vet for `GOOS=windows` in the existing ubuntu job, **plus** a real `windows-latest` test job. |
| Code signing | Out of scope for this plan. Recorded as a follow-up. |
| Await mechanism | Environment variable `PICFETCH_UPDATE_AWAIT_PID`, not a CLI flag — `main.go` has no flag parsing and every bare argument is treated as a file path by `argsToURIs`. |

## Global constraints

- No new module dependencies. `golang.org/x/sys` is an **indirect** requirement; use stdlib `syscall` on Windows instead of promoting it.
- Windows-only code must compile on the ubuntu CI job via `GOOS=windows` — never assume the runner is Windows.
- Platform-independent orchestration lives in files with no build tag so it is testable on macOS/Linux; only syscalls and process handling live behind `//go:build windows`.
- Every user-visible string goes through `lang.L` and is added to both `translations/en.json` and `translations/de.json`.
- Follow the repo's comment style: comments explain *why*, and doc comments open with the documented identifier (`GoCommentStart`; see `finished_refactorings/2026-08-29-gocommentstart-doc-comments.md`).
- Commit after each task with a conventional-commit subject (`feat:`, `fix:`, `test:`, `ci:`, `docs:`).
- `make fmt-check`, `go vet ./...` and `go test ./...` must pass before each commit.

## File map

| File | Status | Responsibility |
|---|---|---|
| `internal/update/applyerr.go` | create | `ApplyError`, `FailureReason`, `ClassifyApplyError`; platform-independent half. |
| `internal/update/applyerr_windows.go` | create | Errno → reason mapping for Windows. |
| `internal/update/applyerr_other.go` | create | `fs.ErrPermission` fallback for every other platform. |
| `internal/update/swap.go` | create | Platform-independent swap orchestration (`swapBinary`) with injectable file operations. |
| `internal/update/apply_windows.go` | rewrite | Wires real file ops + relaunch into `swapBinary`; no `cmd.exe`. |
| `internal/update/apply.go` | modify | Delete `windowsApplyScript` and `windowsCommandPath`. |
| `internal/update/apply_script_test.go` | delete | Tests the deleted script generator. |
| `internal/update/await.go` | create | `CleanupPredecessor`, `SweepOldBinary`, `AwaitPIDEnv`. |
| `internal/update/await_windows.go` | create | `OpenProcess` + `WaitForSingleObject` wait. |
| `internal/update/await_other.go` | create | Signal-0 poll wait. |
| `internal/update/update.go` | modify | Add `ReleasesPageURL`. |
| `internal/ui/autoupdate/applyfailure.go` | create | Fyne-cache store for the failure record, mirroring `whatsnew.go`. |
| `internal/ui/autoupdate/updater.go:495-531` | modify | Save the failure record when `update.Apply` fails; drop the Windows stage-removal exception. |
| `internal/ui/autoupdate.go` | modify | `maybeShowUpdateFailure`. |
| `internal/ui/run.go:46-48` | modify | Call it from `SetOnStarted`. |
| `main.go:66-91` | modify | `update.CleanupPredecessor()` before `app.NewWithID`. |
| `translations/{en,de}.json` | modify | New strings. |
| `.github/workflows/ci.yml` | modify | Windows cross-build/vet step + `windows-latest` test job. |
| `ARCHITECTURE.md`, `internal/ui/help/manual{,_de}.md`, `todos.md` | modify | Docs and close-out. |

## Task graph

```
T1 (classify) ─┐
               ├─> T3 (windows apply) ─> T4 (await + main) ─┐
T2 (swap)     ─┘                                            ├─> T6 (translations) ─> T7 (UI report) ─> T9 (docs)
T5 (failure record store) ──────────────────────────────────┘
T8 (CI) — independent, run any time
```

T1, T2, T5 and T8 are independent and may run in parallel. Everything else is sequential. **Review after every task before dispatching the next.**

---

### Task 1 — Apply-error classification

**Agent:** `go-expert` · **Model:** Sonnet

**Files:**
- Create: `internal/update/applyerr.go`, `internal/update/applyerr_windows.go`, `internal/update/applyerr_other.go`
- Test: `internal/update/applyerr_test.go`, `internal/update/applyerr_windows_test.go`

**Interfaces produced** (later tasks depend on these exact names):

```go
type FailureReason string

const (
	ReasonAccessDenied     FailureReason = "access-denied"
	ReasonVirusBlocked     FailureReason = "virus-blocked"
	ReasonSharingViolation FailureReason = "sharing-violation"
	ReasonUnknown          FailureReason = "unknown"
)

type ApplyError struct {
	Op   string // "rename", "copy", "verify", "restore", "relaunch"
	Path string
	Err  error
}

func (e *ApplyError) Error() string
func (e *ApplyError) Unwrap() error
func ClassifyApplyError(err error) FailureReason
```

- [ ] **Step 1: Write the failing tests** in `internal/update/applyerr_test.go`

```go
func TestApplyError_UnwrapsAndFormats(t *testing.T) {
	inner := errors.New("boom")
	err := &ApplyError{Op: "copy", Path: `C:\App\picfetch.exe`, Err: inner}
	if !errors.Is(err, inner) {
		t.Errorf("ApplyError does not unwrap to its cause")
	}
	for _, want := range []string{"copy", `C:\App\picfetch.exe`, "boom"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Error() = %q, missing %q", err.Error(), want)
		}
	}
}

func TestClassifyApplyError_PermissionIsAccessDenied(t *testing.T) {
	err := &ApplyError{Op: "copy", Path: "p", Err: &fs.PathError{Op: "open", Path: "p", Err: fs.ErrPermission}}
	if got := ClassifyApplyError(err); got != ReasonAccessDenied {
		t.Errorf("ClassifyApplyError = %q, want %q", got, ReasonAccessDenied)
	}
}

func TestClassifyApplyError_NilAndUnknown(t *testing.T) {
	if got := ClassifyApplyError(nil); got != ReasonUnknown {
		t.Errorf("nil = %q, want %q", got, ReasonUnknown)
	}
	if got := ClassifyApplyError(errors.New("plain")); got != ReasonUnknown {
		t.Errorf("plain = %q, want %q", got, ReasonUnknown)
	}
}
```

  And in `internal/update/applyerr_windows_test.go` (build tag `//go:build windows`):

```go
func TestClassifyApplyError_WindowsErrno(t *testing.T) {
	cases := map[syscall.Errno]FailureReason{
		syscall.Errno(5):   ReasonAccessDenied,     // ERROR_ACCESS_DENIED
		syscall.Errno(225): ReasonVirusBlocked,     // ERROR_VIRUS_INFECTED
		syscall.Errno(226): ReasonVirusBlocked,     // ERROR_VIRUS_DELETED
		syscall.Errno(32):  ReasonSharingViolation, // ERROR_SHARING_VIOLATION
		syscall.Errno(33):  ReasonSharingViolation, // ERROR_LOCK_VIOLATION
	}
	for errno, want := range cases {
		err := &ApplyError{Op: "copy", Path: "p", Err: &os.LinkError{Op: "rename", Err: errno}}
		if got := ClassifyApplyError(err); got != want {
			t.Errorf("errno %d = %q, want %q", uintptr(errno), got, want)
		}
	}
}
```

- [ ] **Step 2: Run and confirm failure**

`go test ./internal/update/ -run 'ApplyError|ClassifyApply' -v` → build failure, undefined symbols.

- [ ] **Step 3: Implement**

`applyerr.go` holds the type, the constants, `Error`/`Unwrap`, and `ClassifyApplyError`, which first consults `classifyPlatform(err)` and falls back to `errors.Is(err, fs.ErrPermission) → ReasonAccessDenied`, else `ReasonUnknown`.

```go
// ClassifyApplyError maps a failed Apply to the reason PicFetch reports on
// the next launch. Windows errno values win over the portable
// fs.ErrPermission check because Defender's virus and sharing denials are
// not permission errors.
func ClassifyApplyError(err error) FailureReason {
	if err == nil {
		return ReasonUnknown
	}
	if reason, ok := classifyPlatform(err); ok {
		return reason
	}
	if errors.Is(err, fs.ErrPermission) {
		return ReasonAccessDenied
	}
	return ReasonUnknown
}
```

`applyerr_windows.go` (`//go:build windows`) defines the errno constants that stdlib `syscall` does not export (`errorVirusInfected = 225`, `errorVirusDeleted = 226`) and implements `classifyPlatform` with `errors.As(err, &errno)`. `applyerr_other.go` (`//go:build !windows`) returns `("", false)`.

- [ ] **Step 4: Verify**

`go test ./internal/update/ -run 'ApplyError|ClassifyApply' -v` → PASS.
`GOOS=windows go build ./... && GOOS=windows go vet ./internal/update/` → clean.

- [ ] **Step 5: Commit** — `feat(update): classify apply failures by cause`

**Review gate:** the Windows errno list is complete and correct; no `golang.org/x/sys` import appeared; `ClassifyApplyError` never panics on wrapped nil.

---

### Task 2 — Platform-independent swap orchestration

**Agent:** `go-expert` · **Model:** Opus (this is the core correctness of the feature: rollback ordering and rollback-of-rollback)

**Files:**
- Create: `internal/update/swap.go`
- Test: `internal/update/swap_test.go`
- Modify: `internal/update/apply.go` (delete `windowsApplyScript`, `windowsCommandPath`)
- Delete: `internal/update/apply_script_test.go`

**Interfaces consumed:** `ApplyError` from Task 1.

**Interfaces produced:**

```go
type binaryOps struct {
	Rename  func(oldPath, newPath string) error
	Copy    func(src, dst string) error
	Remove  func(path string) error
	Same    func(a, b string) (bool, error) // SHA-256 equality of two files
	Relaunch func(dest string) error
}

func swapBinary(stagedPath, dest string, options ApplyOptions, ops binaryOps) error
```

Required ordering, which the tests must pin:

1. `Remove(dest + ".old")` — best effort, a stale leftover must not block the rename. Errors ignored.
2. `Rename(dest, dest+".old")` — fails ⇒ `&ApplyError{Op: "rename", Path: dest}`; nothing else has happened, no rollback needed.
3. `Copy(stagedPath, dest)` — fails ⇒ rename `.old` back, then `&ApplyError{Op: "copy", Path: dest}`. If the restore *also* fails, return `errors.Join` of both wrapped in `&ApplyError{Op: "restore", Path: dest}`.
4. `Same(stagedPath, dest)` — false or error ⇒ `Remove(dest)`, rename `.old` back, return `&ApplyError{Op: "verify", Path: dest}`. This is what makes "check if it worked" real rather than assumed: a filter driver may accept the write and discard or alter the bytes.
5. `options.Relaunch` ⇒ `Relaunch(dest)`, failure ⇒ `&ApplyError{Op: "relaunch", Path: dest}`.
6. `dest + ".old"` is **deliberately left on disk** — it is the running image and cannot be deleted by this process. Task 4 sweeps it.

- [ ] **Step 1: Write the failing tests** in `swap_test.go` (no build tag — these run on every platform)

Use a recording fake, e.g.:

```go
type fakeOps struct {
	calls    []string
	renameErr map[string]error
	copyErr  error
	same     bool
	sameErr  error
	relaunched bool
	relaunchErr error
}
```

Cover, one test each:

- `TestSwapBinary_HappyPath`: calls in order `remove old`, `rename dest→old`, `copy staged→dest`, `same`, no relaunch when `Relaunch` is false; `.old` is **not** removed at the end.
- `TestSwapBinary_RenameFailureIsNotRolledBack`: returns `*ApplyError` with `Op == "rename"`, and `copy` was never called.
- `TestSwapBinary_CopyFailureRestoresOriginal`: `rename old→dest` happened, error `Op == "copy"`, unwraps to the copy cause.
- `TestSwapBinary_RestoreFailureReportsBoth`: copy fails *and* the restoring rename fails; error `Op == "restore"` and `errors.Is` finds both causes.
- `TestSwapBinary_VerifyMismatchRollsBack`: `Same` returns false; calls include `remove dest` then `rename old→dest`; error `Op == "verify"`.
- `TestSwapBinary_RelaunchOnlyAfterSuccessfulVerify`: with `Relaunch: true` and a failing `Same`, `relaunched` stays false.
- `TestSwapBinary_RelaunchFailureIsReported`: `Op == "relaunch"`.

- [ ] **Step 2: Run and confirm failure** — `go test ./internal/update/ -run SwapBinary -v`

- [ ] **Step 3: Implement `swapBinary`** exactly to the ordering above, plus `defaultBinaryOps(relaunch func(string) error) binaryOps` wiring `os.Rename`, the existing `copyFile` helper from `apply_unix.go` (**move it to `swap.go`** so it compiles on Windows too; keep the `!windows` file free of it), `os.Remove`, and a `sameContents(a, b string) (bool, error)` using the package's existing `fileSHA256` from `download.go`.

- [ ] **Step 4: Delete the dead script generator**

Remove `windowsApplyScript` and `windowsCommandPath` from `apply.go` and delete `apply_script_test.go`. `go build ./...` and `GOOS=windows go build ./...` must both still pass — `apply_windows.go` is rewritten in Task 3, so this task leaves it calling the old removed function *only if* Task 3 is not yet done; to avoid a broken tree, temporarily make `applyWindows` return `errors.New("update: windows apply not yet reimplemented")` and let Task 3 replace it.

- [ ] **Step 5: Verify** — `go test ./internal/update/ -v`, `make fmt-check`, `GOOS=windows go build ./...`

- [ ] **Step 6: Commit** — `refactor(update): replace the windows apply script with a swap orchestrator`

**Review gate:** rollback paths verified by reading the call log, not by trusting the agent's summary; `copyFile` still used by `applyUnix` and not duplicated; no leftover references to `.apply.cmd` anywhere (`grep -rn "apply.cmd" .`).

---

### Task 3 — Windows apply implementation

**Agent:** `go-expert` · **Model:** Opus

**Files:**
- Rewrite: `internal/update/apply_windows.go`
- Test: `internal/update/apply_windows_test.go` (`//go:build windows`)

**Interfaces consumed:** `swapBinary`, `binaryOps`, `defaultBinaryOps` (Task 2); `ApplyError` (Task 1); `AwaitPIDEnv` (Task 4 — declare it in Task 4 and let this task import it; if Task 4 has not run yet, define the constant here and let Task 4 move it, whichever order the executor takes, but the final tree must have exactly one definition, in `await.go`).

```go
func applyWindows(stage Stage, dest string, options ApplyOptions) error {
	return swapBinary(stage.BinaryPath, dest, options, defaultBinaryOps(relaunchWindows))
}

// relaunchWindows starts the freshly installed executable and tells it which
// process to wait for. The new instance must not touch preferences before the
// old one is gone: Apply runs inside Fyne's stopped callback, and Fyne saves
// preferences immediately after that callback returns.
func relaunchWindows(dest string) error {
	cmd := exec.Command(dest)
	cmd.Env = append(os.Environ(), AwaitPIDEnv+"="+strconv.Itoa(os.Getpid()))
	return cmd.Start()
}
```

The `applyUnix` stub at the bottom of the file stays as it is.

- [ ] **Step 1: Write the failing Windows test**

```go
//go:build windows

func TestApplyWindows_ReplacesDestAndKeepsOld(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch.exe")
	if err := os.WriteFile(dest, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged.exe")
	if err := os.WriteFile(staged, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := applyWindows(Stage{BinaryPath: staged}, dest, ApplyOptions{}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "new" {
		t.Fatalf("dest = %q, %v; want %q", got, err, "new")
	}
	if _, err := os.Stat(dest + ".old"); err != nil {
		t.Errorf("dest.old must survive for the next launch to sweep: %v", err)
	}
	if _, err := os.Stat(staged); err != nil {
		t.Errorf("staged binary must remain (copy, not rename): %v", err)
	}
	if _, err := os.Stat(dest + ".apply.cmd"); !os.IsNotExist(err) {
		t.Errorf("apply must not write a cmd script any more")
	}
}

func TestApplyWindows_MissingStagedBinaryRestoresDest(t *testing.T) { /* dest content is "old" again, error Op == "copy" */ }
```

- [ ] **Step 2: Confirm it fails** — on the Windows CI job, or locally via `GOOS=windows go vet ./internal/update/` if no Windows machine is available. State plainly in the task report which of the two was actually run.

- [ ] **Step 3: Implement** as sketched above.

- [ ] **Step 4: Verify** — `GOOS=windows go build ./... && GOOS=windows go vet ./...` locally; `go test ./internal/update/` on macOS still passes (the non-tagged swap tests carry the logic).

- [ ] **Step 5: Commit** — `fix(update): apply windows updates in-process instead of via cmd.exe`

**Review gate:** no `exec.Command("cmd", ...)` remains in the repository (`grep -rn '"cmd"' internal/`); `CreationFlags`/`createNoWindow` is gone with the script or still justified; the relaunch inherits the environment rather than replacing it.

---

### Task 4 — Predecessor wait and leftover sweep

**Agent:** `go-expert` · **Model:** Opus (touches process startup ordering in `main.go`)

**Files:**
- Create: `internal/update/await.go`, `internal/update/await_windows.go`, `internal/update/await_other.go`
- Test: `internal/update/await_test.go`
- Modify: `main.go` (after `openwith.Install()`, before `app.NewWithID`)

**Interfaces produced:**

```go
const AwaitPIDEnv = "PICFETCH_UPDATE_AWAIT_PID"

// CleanupPredecessor waits for the process that installed this executable to
// exit, then deletes the files an update leaves behind. Both halves are
// no-ops on a normal launch.
func CleanupPredecessor()

// SweepOldBinary removes the backup and any leftovers from the pre-2026-08-30
// cmd.exe apply script. Errors are ignored: a file still locked by an exiting
// process is swept by the launch after this one.
func SweepOldBinary(dest string)

func awaitProcessExit(pid int, timeout time.Duration) // platform-specific
```

`CleanupPredecessor` reads `AwaitPIDEnv`, and when it parses to a positive int, calls `awaitProcessExit(pid, 15*time.Second)`. Then it resolves `os.Executable()` and calls `SweepOldBinary(dest)`, which removes `dest+".old"`, `dest+".new"` and `dest+".apply.cmd"` — the last two clean up after installations that ran the old script.

**Hazard raised by Task 2's review — do not skip.** `swapBinary` can fail with `Op: "restore"`, which means the copy failed *and* the restoring rename failed: `dest` is then missing or partial and `<dest>.old` is the user's only intact executable. The sweep must not delete `.old` in that state. Task 5 stores the failed apply's `Op` in the cache record, so `CleanupPredecessor` checks for a recorded failure with `Op == "restore"` and skips the `.old` removal when it finds one (still sweeping `.new` and `.apply.cmd`). Because Task 5 runs after Task 4 in the graph, implement the guard against the record's shape and pin it with a test that fakes the record; if the record type is not available yet, gate on `dest` being absent instead — a missing `dest` alone is sufficient reason never to delete `.old`. A test must pin the guard both ways.

`await_windows.go`: `syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))`; an error means the process is already gone, so return immediately. Otherwise `syscall.WaitForSingleObject(h, uint32(timeout/time.Millisecond))` and `syscall.CloseHandle(h)` in a defer.
**Two more requirements raised by Task 3's review.** First, `AwaitPIDEnv` otherwise stays in the environment for the life of the process and is inherited by everything PicFetch later spawns — unset it with `os.Unsetenv(AwaitPIDEnv)` once the wait is done. Second, Windows recycles PIDs aggressively, so a bare PID is a weak handle: a recycled PID could make the new instance wait on an unrelated process. The bounded timeout already caps the damage, so treat this as a documented limit rather than a mechanism to build — but say so in `CleanupPredecessor`'s doc comment rather than leaving it implied.

`await_other.go`: poll `syscall.Kill(pid, 0)` every 100ms until it errors or the timeout expires. Unix never sets the variable; the implementation exists so the package builds and the behaviour is testable off Windows.

- [ ] **Step 1: Write the failing tests** (no build tag)

```go
func TestCleanupPredecessor_NoEnvReturnsImmediately(t *testing.T) {
	t.Setenv(AwaitPIDEnv, "")
	done := make(chan struct{})
	go func() { CleanupPredecessor(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("CleanupPredecessor blocked with no pid to await")
	}
}

func TestSweepOldBinary_RemovesLeftovers(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	for _, suffix := range []string{"", ".old", ".new", ".apply.cmd"} {
		if err := os.WriteFile(dest+suffix, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	SweepOldBinary(dest)
	for _, suffix := range []string{".old", ".new", ".apply.cmd"} {
		if _, err := os.Stat(dest + suffix); !os.IsNotExist(err) {
			t.Errorf("%s survived the sweep: %v", suffix, err)
		}
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("sweep deleted the executable itself: %v", err)
	}
}

func TestAwaitProcessExit_DeadPidReturnsBeforeTimeout(t *testing.T) {
	// A pid that has certainly exited: start `true`/a helper and Wait for it first.
	// Assert awaitProcessExit returns in well under the timeout.
}
```

- [ ] **Step 2: Run and confirm failure**

- [ ] **Step 3: Implement the three files.**

- [ ] **Step 4: Wire `main.go`**

```go
	openwith.Install()

	// Before app.NewWithID: an update relaunch must not read or write
	// preferences while the process it replaced is still flushing its own.
	update.CleanupPredecessor()

	application := app.NewWithID(appID)
```

- [ ] **Step 5: Verify** — `go test ./internal/update/ -run 'Cleanup|Sweep|Await' -v`, `go build ./...`, `GOOS=windows go build ./...`

- [ ] **Step 6: Commit** — `feat(update): wait for the replaced process and sweep its leftovers`

**Review gate:** the wait is bounded (a hung predecessor must not hang the new launch); `main.go`'s ordering comment explains *why* it precedes `app.NewWithID`; sweep never touches `dest` itself.

---

### Task 5 — Failure-record store

**Agent:** `go-expert` · **Model:** Sonnet

**Files:**
- Create: `internal/ui/autoupdate/applyfailure.go`
- Test: `internal/ui/autoupdate/applyfailure_test.go`
- Modify: `internal/ui/autoupdate/updater.go:495-531` (`ApplyStagedUpdate`)

Mirror `internal/ui/autoupdate/whatsnew.go` exactly — same `app.Cache()` read/write/exists/remove shape, same doc-comment style.

**Interfaces produced:**

```go
const ApplyFailureCacheKey = "updatefailure.json"

type ApplyFailure struct {
	Version string `json:"version"`
	Reason  string `json:"reason"` // update.FailureReason
	Op      string `json:"op"`     // ApplyError.Op
	Path    string `json:"path"`   // the executable that could not be replaced
	Detail  string `json:"detail"` // err.Error(), for the log, not the dialog
}

func SaveApplyFailure(app fyne.App, f ApplyFailure) error
func LoadApplyFailure(app fyne.App) (*ApplyFailure, error)
func ClearApplyFailure(app fyne.App) error
```

`ApplyStagedUpdate` change:

```go
	if err := update.Apply(st, dest, u.applyOptions); err != nil {
		fyne.LogError("failed to apply update", err)
		var applyErr *update.ApplyError
		op := ""
		if errors.As(err, &applyErr) {
			op = applyErr.Op
		}
		if saveErr := SaveApplyFailure(u.app, ApplyFailure{
			Version: st.Version,
			Reason:  string(update.ClassifyApplyError(err)),
			Op:      op,
			Path:    dest,
			Detail:  err.Error(),
		}); saveErr != nil {
			fyne.LogError("failed to record update failure", saveErr)
		}
		return
	}
	_ = update.RemoveStage(u.dir)
```

Note the last line: the `if runtime.GOOS != "windows"` exception existed only because the `.cmd` script deleted the staged file itself. The in-process apply finishes before `Apply` returns, so Windows now removes the stage like every other platform. Check whether `runtime` is still used elsewhere in the file before deleting the import.

**Wire the record into the sweep — raised by Task 4's review, do not skip.** Task 4 guards `<dest>.old` by refusing to delete it when `dest` is missing, which is a conservative approximation: a truncated `dest` (a copy that failed mid-`io.Copy`, `swap.go:118`) still stats fine, so the guard passes and the user's only intact executable is swept. The real check is the recorded `Op == "restore"`, which exists only after this task.

The record lives in `app.Cache()`, which needs a `fyne.App`, while `CleanupPredecessor` deliberately runs *before* `app.NewWithID` — so the two cannot simply be joined. Split them by what actually needs the early slot: only the **wait** must precede `app.NewWithID`. Therefore:

- `CleanupPredecessor` keeps the wait, the `os.Unsetenv`, and the `.new` / `.apply.cmd` removal, and stops removing `.old`.
- Export a second function `update.SweepBackup(dest string)` holding just the `.old` removal and Task 4's missing-`dest` guard.
- Call it from `internal/ui`'s startup, where `v.app` exists — next to `maybeShowUpdateFailure` in `SetOnStarted` (Task 7 adds that call site; if this task runs first, add the call and let Task 7 slot in beside it). Skip the call when `LoadApplyFailure` returns a record with `Op == "restore"`; that record's presence is the only reliable evidence the backup is load-bearing.
- Tests: `.old` survives when a restore record is present, is swept when the record is absent, and is swept when a record exists with any other `Op`. Update Task 4's `SweepOldBinary` tests to match the split rather than leaving them asserting behaviour that moved.

- [ ] **Step 1: Write the failing tests** — round-trip save/load/clear, `LoadApplyFailure` returns `(nil, nil)` with nothing cached, and an `ApplyStagedUpdate` test with a stubbed `update.Apply` returning `&update.ApplyError{Op: "copy", Err: fs.ErrPermission}` asserting a record with `Reason == "access-denied"` and that the stage was **not** removed (so a later launch can retry). Follow the existing stubbing pattern in `updater_test.go`.
- [ ] **Step 2: Run and confirm failure**
- [ ] **Step 3: Implement**
- [ ] **Step 4: Verify** — `go test ./internal/ui/autoupdate/ -v`
- [ ] **Step 5: Commit** — `feat(autoupdate): record why an update could not be applied`

**Review gate:** stage retention on failure is deliberate and tested; the Windows special case is gone; no `Detail` string reaches the UI unfiltered (Task 7 decides what is shown).

---

### Task 6 — Translations

**Agent:** `general-purpose` · **Model:** Sonnet

**Files:** `translations/en.json`, `translations/de.json`

Add these keys (English value = key, German translated). Keep the existing file ordering convention and the two-space indent.

| Key | German |
|---|---|
| `Update could not be installed` | `Update konnte nicht installiert werden` |
| `Windows blocked PicFetch from replacing itself at %s. This is Controlled Folder Access ("Überwachter Ordnerzugriff") protecting that folder. Allow PicFetch through it in Windows Security → Virus & threat protection → Ransomware protection → Allow an app through Controlled folder access, or move PicFetch to a folder outside Documents, Pictures, Music, Videos and Desktop.` | German equivalent, keeping the German UI path names Windows itself uses: `Windows-Sicherheit → Viren- & Bedrohungsschutz → Ransomware-Schutz → App durch überwachten Ordnerzugriff zulassen` |
| `Your antivirus removed or quarantined the downloaded update.` | `Ihr Virenschutz hat das heruntergeladene Update entfernt oder in Quarantäne verschoben.` |
| `PicFetch could not replace itself at %s because the file was in use.` | `PicFetch konnte sich unter %s nicht ersetzen, weil die Datei in Verwendung war.` |
| `PicFetch could not install the update at %s.` | `PicFetch konnte das Update unter %s nicht installieren.` |
| `The previous version is still installed and running.` | `Die vorherige Version ist weiterhin installiert und wird ausgeführt.` |
| `Open download page` | `Download-Seite öffnen` |
| `Close` | check first — it may already exist |

- [ ] **Step 1:** `grep -n '"Close"' translations/en.json` and reuse existing keys rather than adding duplicates.
- [ ] **Step 2:** Add the keys to both files.
- [ ] **Step 3:** Verify both files parse: `python3 -m json.tool translations/en.json >/dev/null && python3 -m json.tool translations/de.json >/dev/null`, and that the key sets match: compare `jq -r 'keys[]' ` output of both.
- [ ] **Step 4: Commit** — `i18n: add update-failure strings`

**Review gate:** every `%s` count matches between English and German; no key exists in one file only.

---

### Task 7 — Next-launch failure report

**Agent:** `go-expert` · **Model:** Opus (Fyne dialog + `internal/ui` lifecycle wiring)

**Files:**
- Modify: `internal/ui/autoupdate.go` (add `maybeShowUpdateFailure` next to `maybeShowWhatsNew:177`)
- Modify: `internal/ui/run.go:46-48` (call it from `SetOnStarted`, after `maybeShowWhatsNew`)
- Modify: `internal/update/update.go` (add `ReleasesPageURL`)
- Test: `internal/ui/autoupdate_test.go`

**Interfaces consumed:** `autoupdate.LoadApplyFailure` / `ClearApplyFailure` / `ApplyFailure` (Task 5); `update.ReasonAccessDenied` etc. (Task 1); the strings from Task 6.

**Ordering, made structural in Task 5 — honour it.** `run.go`'s `SetOnStarted` calls `view.sweepUpdateBackup()` first, and Task 5 changed it to *return* the loaded `*autoupdate.ApplyFailure`. Take that record as a parameter — `maybeShowUpdateFailure(rec *autoupdate.ApplyFailure)` — instead of loading it again. The data dependency is what keeps the dialog from clearing the record before the sweep has read it; clearing first would make a failed restore look like a clean install and take the user's last working binary. Do not reintroduce a second `LoadApplyFailure` call in this task.

Task 5 also clears the record on a *successful* apply (`ApplyStagedUpdate`), so this task's clear-on-show is a second safety net rather than the only one. Keep it anyway, and do not version-gate the dialog the way `maybeShowWhatsNew` is gated: a suppressed dialog that never clears would strand the record.

```go
// ReleasesPageURL is the human-facing download page PicFetch offers when it
// could not install an update itself.
const ReleasesPageURL = "https://github.com/" + RepoOwner + "/" + RepoName + "/releases/latest"
```

```go
// maybeShowUpdateFailure explains a failed binary replacement on the launch
// after it happened. Apply runs from the stopped callback, where no window
// is left to report into, so the reason is cached and surfaced here. The
// record is cleared before the dialog opens so a later launch stays quiet.
func (v *viewer) maybeShowUpdateFailure() {
	f, err := autoupdate.LoadApplyFailure(v.app)
	if err != nil || f == nil {
		return
	}
	_ = autoupdate.ClearApplyFailure(v.app)
	body := updateFailureMessage(*f)
	d := dialog.NewCustomConfirm(
		lang.L("Update could not be installed"),
		lang.L("Open download page"),
		lang.L("Close"),
		widget.NewRichTextWithText(body),
		func(open bool) {
			if !open {
				return
			}
			u, parseErr := url.Parse(update.ReleasesPageURL)
			if parseErr != nil {
				return
			}
			if openErr := v.app.OpenURL(u); openErr != nil {
				fyne.LogError("failed to open the download page", openErr)
			}
		},
		v.win,
	)
	d.Show()
}

// updateFailureMessage picks the explanation for a recorded reason. Split
// out so the wording is testable without opening a dialog.
func updateFailureMessage(f autoupdate.ApplyFailure) string
```

`updateFailureMessage` maps:
- `ReasonAccessDenied` → the Controlled Folder Access text with `f.Path`
- `ReasonVirusBlocked` → the antivirus text
- `ReasonSharingViolation` → the in-use text with `f.Path`
- anything else → the generic text with `f.Path`

and appends the "previous version is still installed and running" sentence in every case.

- [ ] **Step 1: Write the failing tests** in `internal/ui/autoupdate_test.go`, following the existing harness (`newTestUI`, see `harness_test.go`):
  - `TestMaybeShowUpdateFailure_NoRecordShowsNothing`
  - `TestMaybeShowUpdateFailure_ClearsRecord` — after the call, `LoadApplyFailure` returns nil, so a second launch is silent
  - `TestUpdateFailureMessage_AccessDeniedNamesFolderAccessAndPath` — asserts the path appears and the message differs from the generic one
  - `TestUpdateFailureMessage_UnknownReasonFallsBackToGeneric`
- [ ] **Step 2: Run and confirm failure**
- [ ] **Step 3: Implement, including the `run.go` call site**

```go
		view.maybeShowWhatsNew()
		view.maybeShowUpdateFailure()
```

- [ ] **Step 4: Verify** — `go test ./internal/ui/ -run UpdateFailure -v` then the full `go test ./internal/ui/` (it is slow; budget ~10 min)
- [ ] **Step 5: Commit** — `feat(ui): explain a blocked update on the next launch`

**Review gate:** dialog uses `v.win` and cannot fire before the window exists; the record is cleared exactly once; `OpenURL` failure is logged, not swallowed silently; no raw `Detail` string in the dialog.

---

### Task 8 — CI gates for Windows

**Agent:** `general-purpose` · **Model:** Sonnet · **May run in parallel with Tasks 1–7**

**Files:** `.github/workflows/ci.yml`

- [ ] **Step 1:** In the existing ubuntu job, after the `Build` step, add:

```yaml
      - name: Cross-build Windows
        run: |
          GOOS=windows GOARCH=amd64 go build ./...
          GOOS=windows GOARCH=arm64 go build ./...

      - name: Vet Windows
        run: GOOS=windows GOARCH=amd64 go vet ./...
```

- [ ] **Step 2:** Add a `windows-latest` job running the non-UI packages:

```yaml
  windows-test:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      # Not -race: the race detector needs a C toolchain, and these packages
      # are the file-replacement logic, not the concurrent UI. Not ./...:
      # internal/ui needs a display driver this runner does not have.
      - name: Test
        run: go test ./internal/update/... ./internal/ui/autoupdate/...
```

- [ ] **Step 3: Verify** the YAML parses and the job names do not collide: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/ci.yml'))"`. Push the branch and confirm both jobs actually run and pass — report the run ID.
- [ ] **Step 4: Commit** — `ci: build, vet and test the Windows code paths`

**Review gate:** the Windows job really executed (check the run, do not assume); the cross-build covers both release architectures.

---

### Task 9 — Documentation and close-out

**Agent:** `general-purpose` · **Model:** Sonnet · **Runs last**

**Files:** `ARCHITECTURE.md`, `internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`, `todos.md`, `finished_refactorings/2026-08-30-windows-defender-cfa-update.md`

- [ ] **Step 1:** `ARCHITECTURE.md` — update the `apply.go / apply_unix.go / apply_windows.go` row (line ~160): it currently describes the Windows path as a script. Add rows for `swap.go`, `applyerr.go`, `await.go`, and `internal/ui/autoupdate/applyfailure.go`. Update the "How do in-app updates work?" answer (line ~386) with the next-launch failure report and the `PICFETCH_UPDATE_AWAIT_PID` relaunch handshake.
- [ ] **Step 2:** Both manuals — one short paragraph in the update section: if Windows blocks the update, PicFetch says so on the next start; allow PicFetch through Controlled Folder Access or keep it outside the protected user folders. German manual uses the German Windows UI names.
- [ ] **Step 3:** `todos.md` — move the TODO entry into `## Done → #### Bugfix`, written in the established evidence style: what was blocked, why `cmd.exe` can never be allowed by CFA, what replaced it, and the honest limit — an unsigned binary writing into a protected folder can still be denied, which is why the failure is now reported instead of silent. Add a `## TODO` entry for Authenticode signing of the Windows release as the remaining real fix.
- [ ] **Step 4:** Write `finished_refactorings/2026-08-30-windows-defender-cfa-update.md` with the same evidence discipline the other files in that directory use.
- [ ] **Step 5: Commit** — `docs: record the windows update-apply rework`

**Review gate:** no document still claims a `.cmd` script is written; the signing follow-up is recorded rather than implied.

---

## Verification before declaring done

1. `make fmt-check && go vet ./... && go test ./...` on macOS.
2. `GOOS=windows GOARCH=amd64 go build ./... && GOOS=windows GOARCH=arm64 go build ./... && GOOS=windows go vet ./...`.
3. CI green, including the new `windows-test` job — quote the run ID.
4. `grep -rn 'apply\.cmd\|exec.Command("cmd"' .` returns nothing outside `todos.md`/`finished_refactorings/`.
5. **Manual Windows check by the user** — this cannot be automated here. Build a Windows binary, place it in the same protected folder as the failing installation, trigger *Perform update*, and check: does the swap now succeed? If Defender still blocks it, does the next launch show the Controlled Folder Access dialog with the right path? Both outcomes are useful; the second confirms the reporting half works and escalates the signing follow-up.

## Known limits

- **Unsigned binaries may still be blocked.** CFA trusts by signature and reputation. If the manual check in step 5 still shows a block — now naming `picfetch.exe` rather than `cmd.exe` — the remaining fix is Authenticode signing in the release workflow (Azure Trusted Signing or a purchased certificate), which is deliberately out of this plan's scope.
- **`<dest>.old` lingers until the next launch.** Windows cannot delete a running image. The sweep in Task 4 handles both the relaunch case and the next cold start.
- **No Windows machine in this session.** Every Windows-only assertion in this plan is verified by cross-compilation and by the new CI job, never by a local run. Task reports must say which of the two was used.
