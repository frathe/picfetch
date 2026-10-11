# PR 75 cancellation review fixes

Base: `6146a9e8ca28b5ffc0ff8cb5b8fbfa6546c3299e`. Its security review,
CodeQL, Qodana (eight confirmed unused-export false positives) and full CI passed.
The code review raised two confirmed cancellation defects; a fresh clean round
is required after their disposition.

Route: Deep for the native broker boundary, implemented and reviewed by the
lead. Zero delegates; focused local tests and native qualification, full CI in
GitHub. No dependency, package, UI string or public API change.

## Contracts and tasks

1. Broker SIGTERM received during handler installation must survive until the
   XPC cancel path can retire and reap the child. Preserve normal mode/pipe,
   network isolation, bookmark and crash behavior. Add deterministic early
   SIGTERM injection to the existing signed qualification fixture; demonstrate
   failure before fixing and pass afterward with
   `sh scripts/macworkerqualify/run.sh`. Files: native/client.m, qualification
   run.sh, driver.m and check.py. Guard both HEIC and similarity.
2. Animated terminal help must restore cursor/screen on both SIGINT and SIGTERM;
   only SIGINT means successful skip. Existing launchArgs error handling must
   return nonzero for SIGTERM. Add signal cases to consolehelp/help_test.go,
   then fix help.go. Run `go test -race ./internal/consolehelp` and a real PTY
   subprocess probe; preserve redirected/small-terminal/resize/output-error paths.
3. Inspect every changed code file including weak warnings, run focused native
   and Go regressions plus `make verify-build`, update this record and todos,
   sign/push, reply/resolve both threads and request one fresh review round.

Graph: each regression -> its fix -> shared qualification/review. One candidate
review per fix. No broad local race suite. Existing test files/exclusions stay.

## Investigation

- Broker changes SIGTERM to SIG_IGN before dispatch registration. A signal in
  that gap is lost. A native Darwin probe also proved SIG_BLOCK plus SIG_IGN
  discards it (`sigpending` returned zero), so the suggested mask alone is
  insufficient. The broker now installs a minimal handler that records a
  `volatile sig_atomic_t` flag before Foundation setup, explicitly unblocks
  inherited SIGTERM, and replays a recorded cancellation in dispatch's
  registration handler. Later signals use the ordinary event handler. Resume
  the XPC connection before activating the source; both routes preserve the
  service's existing cleanup acknowledgment. No XPC work runs in a signal handler.
- consolehelp.Write subscribes both signals to a context whose cancellation
  play treats as success. launchArgs already returns 1 for Write errors; no
  root startup change is required. Pass notifications into the existing finite
  animation loop, avoiding another goroutine or mutable test seam.

## Verification

- Native red: injecting SIGTERM immediately after the production signal
  disposition call made the signed broker's early-HEIC case time out while its
  worker remained active. A mask-only candidate also failed and was replaced.
- Native green: `sh scripts/macworkerqualify/run.sh` passes all 12 cases: both
  worker modes with blocked/unblocked inherited masks, normal mode/pipe/reaping,
  invalid mode, held/descendant cancellation, broker crash, normal descendant
  cleanup and private-source bookmark transfer. The injection is compile-time
  only; production has no test environment or argument seam. Clang uses
  `-Wall -Wextra -Werror`. The app/helper/service fixtures are ad-hoc signed.
- Console red: a real PTY sent SIGTERM after the first rendered frame; original
  help restored the terminal but returned 0. The same probe now observes SIGINT
  exit 0 and SIGTERM exit 1, both with cursor/alternate-screen restoration.
- `TestAnimationSignalsRestoreTerminal` covers both signals. A deliberately
  restored SIGTERM-as-success branch failed its exact expected diagnostic;
  the intended source was restored immediately afterward.
- `go test -race -tags no_emoji,nodynamic ./internal/consolehelp
  ./internal/macworker` passes, including existing resize/plain/output-error
  help checks. `make verify-build` completes formatting, asset/TUF/notice and
  exact-exclusion checks, full host vet and build without errors; only the
  existing native duplicate `-lobjc` linker warning appears.
- GoLand `get_file_problems(errorsOnly=false)` reports no findings in all six
  changed Go/native/Python/shell code files. Reinspected the Python fixture
  after adding both inherited signal masks. This is the IDE-local fallback,
  not a replacement for fresh hosted Qodana SARIF assessment.
- Root `go test -tags no_emoji,nodynamic -run
  'TestLaunchArgs|TestLaunchStartupContract' .` passes (0.571s). Console-help
  tests cross-compile for Windows/amd64 with CGO disabled (exit 0).
- Candidate review (lead): early and later signal paths converge on the same
  idempotent XPC cancel operation; normal completion still waits for the reply
  after service reaping. Terminal cleanup remains deferred on every return,
  and existing launchArgs/preview error handling returns nonzero. No new Go
  worker or public interface, dependency, package, test file or shard entry.

Final hosted outcomes will be recorded in the PR after the fresh review;
no release or merge is authorized. Native qualification here covers Apple
Silicon, not a physical Intel host. Full native amd64 suites run in hosted CI.

Ledger: zero spawns; one final candidate review of each fix; targeted race,
native qualification, real PTY and one local build/static gate.
