# Trane ASCII turntable

Deliverable: `picfetch --help` plays one three-second, eased 360-degree ASCII
Trane turn above stationary help, reveals an ASCII PicFetch wordmark beneath
the final front portrait, then exits successfully. `make trane` previews the
same code. Ronin selected one turn then exit, requested the wordmark, and set a
four-second maximum. The underlying angle renderer is seamlessly loopable.

Scope expanded from a standalone art demo to application startup integration;
use the Deep route for the Windows console boundary. All work remains inline
with Pico: no delegated tasks or reviews. Ronin accepted the work and authorized
committing and pushing the Trane changes after verification.

## Decisions and limits

- Native Go, original analytic ellipsoid geometry inspired by
  `assets/trane/trane_security_superhero.png`, monochrome ASCII with ANSI cursor
  control. No images, downloaded mesh, Python runtime or Fyne startup.
- Synchronous bounded rendering at 30 FPS; elapsed-time easing; a local signal
  context and stopped ticker own cancellation. Ctrl+C/SIGTERM restore the
  terminal and leave plain help. Normal completion leaves portrait/wordmark/help
  in scrollback. No renderer goroutines or mutable global seams.
- All help remains available. Wrapping and adaptive artwork preserve spare
  terminal margins. Pipes, `TERM=dumb`, unavailable console support and small
  terminals get immediate plain help. Shrinking below the minimum exits cleanly.
- Windows enables/restores VT output with the existing `x/sys/windows` API.
- Existing `golang.org/x/term` v0.46.0 becomes direct (unchanged version), source
  `https://proxy.golang.org/golang.org/x/term/@v/v0.46.0.zip`, BSD-3-Clause. Its
  license is already shipped in `THIRD-PARTY-NOTICES.md` and the reviewed notice
  manifest; existing x/sys notices remain unchanged. No new shipped dependency.
- Honest limits: stylized sculpture, not a recovered Trane mesh; a generous
  terminal (recommended 100x55+) is needed to show art and full help together.
  Native Windows/Linux console behavior remains unverified on this Mac.

## Tasks and verification

1. Renderer and presentation, Pico inline: `internal/consolehelp/{render,help}.go`,
   build-tagged terminal files and `help_test.go`. Contract: `Write(io.Writer,
   usage string) error` keeps nonterminal help unchanged. Verify periodic ASCII
   frames, different viewpoints, cancellation, deterministic three-second
   completion, wordmark placement, wrapping, resizing and partial-write cleanup
   with `go test -race ./internal/consolehelp`.
2. Integration, depends on 1, Pico inline: `main.go`, existing `main_test.go`,
   `internal/launch/launch.go`, preview command/README, Makefile, go.mod, exact
   Qodana exclusion, architecture and todos. Verify early startup exit using
   `go test -tags no_emoji,nodynamic -run 'TestLaunchArgs|TestLaunchStartupContract' .`;
   test parser separately and run the built binary in a real PTY.
3. Gate, depends on 1/2, Pico inline: GoLand per-file inspections including weak
   warnings, `make verify`, Windows/Linux helper cross-compilation and actual
   terminal capture. Budget: zero spawns; three visual passes; one complete gate.

## Evidence

- Renderer test first failed with `front view has no sculpture`; implementing
  the renderer made it pass. Cancellation first failed because no animation was
  emitted, then passed after terminal lifecycle implementation.
- `go test -race -tags no_emoji,nodynamic ./internal/consolehelp ./internal/launch`
  passed. Added resize and partial-write regressions also pass under `-race`.
- Root CLI tests passed, including the final strengthened plain-output and
  `TestLaunchStartupContract` run.
- `make verify-build` passed: formatting, TUF, Qodana exact test exclusions,
  generated vectors/assets, both notice checks, full tagged vet and build.
  `make build` produced `bin/picfetch`. Native linker emitted only its existing
  duplicate `-lobjc` warning.
- `make verify` stopped at its platform guard: daemon is `linux/aarch64`, not
  native Linux/amd64. Full Docker race suite remains unverified; no isolation
  policy or tests were weakened.
- Console-help test binaries cross-compiled for Windows/amd64 and Linux/amd64.
- Actual built `bin/picfetch --help` in a 120x65 PTY: 90 frames, exit 0 in
  **3.436 seconds**, including process startup. SIGINT: exit 0 in 0.026 seconds
  with cursor/screen restored. 80x24 and `TERM=dumb`: plain help, exit 0 in
  0.022/0.021 seconds. Piped help has no ANSI and exits 0.
- Full front portrait, wordmark and every help flag inspected in retained
  `.scratch/trane-ascii/final.txt`; raw PTY streams are beside it. No image
  preview was generated (Pillow unavailable).
- GoLand `get_file_problems(errorsOnly=false)` returned no findings for renderer,
  help, tests, both platform adapters, main, launch and preview command. This is
  the IDE inspection fallback, not a fresh Qodana SARIF pass. Final reruns on
  help, renderer, expanded tests and root tests also returned no findings.
  Analyzed scope is these working-tree changes atop `4a80484`, using the IDE's
  active profile (not an asserted equivalent of `qodana.starter`). Final
  `make fmt-check` and `git diff --check` passed.

Cost: zero spawns, three visual passes (front/profile/back; front bridge;
integrated portrait/wordmark), one full gate attempt.
