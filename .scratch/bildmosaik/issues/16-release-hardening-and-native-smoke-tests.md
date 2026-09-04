# Release hardening and native smoke tests

Type: task
Status: ready-for-human
Priority: P2
Blocked by: 05, 06, 07, 08, 09, 10, 11, 12, 13, 14, 15

## Goal

Run the already-specified automated gates once, then record the native display
and wallpaper behavior that unit tests and Linux CI cannot establish.

## Existing code anchors

- `make verify` is the repository's CI-equivalent final gate: format, offline
  TUF, exact Qodana exclusions, vet, build, and the Linux/amd64 race suite.
  Re-spelling those commands manually omits guards and drifts from CI.
- `make test-native` is the explicit host-platform suite. Windows cross-vet is
  an additional build-selected check, not evidence that a desktop changed.
- Lazy 10,000-source generation, reverse-order regeneration, source checksums,
  and atomic export/cache behavior belong to tickets 05, 06, 09, and 11. This
  ticket runs those tests; it does not postpone creating them.

## Scope

- Run every focused verification command from tickets 01-15 and record actual
  output or failures. Do not substitute the stale aggregate regexes in the
  original feature spec for the refined per-ticket commands.
- Run `make verify` exactly once after focused checks are green, then run
  `make test-native` on each available native host and Windows cross-vet.
- Exercise HiDPI, negative-origin/portrait/unusual-aspect displays, display
  attach/remove while the window is open, rapid Regenerate/Close, corrupt and
  disappearing sources, and both output actions.
- Record OS, desktop environment/display server, versions, display arrangement,
  scaling, target ID label (not raw device path), result, and whether behavior
  is automated, observed, or unverified under `## Comments`.
- Verify source checksums before/after Generate, Regenerate, Save Image, and Set
  as Wallpaper. Verify cancelled/failed export leaves no destination temp file;
  verify wallpaper failure leaves no new cache file and no premature sweep.
- If a smoke test fails, reopen the owning implementation ticket with evidence;
  do not patch production behavior under this release checklist.

## Native smoke matrix

- Windows: one and two displays; select each; detach selected display; legacy
  no-target action; wallpaper survives PicFetch exit.
- macOS: internal plus external display; select each; preserve each screen's
  existing desktop-image options; detach selected display; legacy all-screen
  action; wallpaper survives exit.
- GNOME on X11 and Wayland where available: display inspection is truthful or
  explicitly unsupported; targeted wallpaper is refused before global change;
  legacy global light/dark URI behavior still works.
- KDE Plasma on X11 and Wayland where available: same inspection requirement;
  targeted wallpaper is refused before global change; legacy Plasma-first
  behavior still works.
- Unsupported Linux desktop: generation/export remain usable and both display
  and target limitations are explicit.

## Acceptance Criteria

- Every refined ticket verification command passes, including the 10,000-entry
  lazy-load and reverse-order stale-result tests.
- No cancellation/failure leaves a partial export, new dead wallpaper copy, or
  late UI mutation; all source checksums remain unchanged.
- Native results identify verified and unverified rows explicitly. Cross-build
  success is never reported as native wallpaper success.
- `make verify`, native host tests, and Windows cross-vet are green on the final
  candidate.
- Plan/tracker artifacts remain in place until the implementation branch is
  accepted; no `git commit` is made by the agent.

```sh
make verify &&
make test-native &&
GOOS=windows GOARCH=amd64 go vet ./internal/...
```

## Non-Goals

- Adding missing stress tests at release time
- Treating untested platforms as passing
- Moving tracker/plan artifacts before branch acceptance

## Comments

This ticket remains `ready-for-human` because changing real desktop wallpaper
and attaching/removing physical displays require supervised native hosts. Append
the completed smoke matrix here.

Automated results on 2026-09-04:

| Environment | Result | Evidence |
| --- | --- | --- |
| Linux/amd64 CI container | verified | `make verify` passed formatting, TUF, Qodana, vet, build, 581-test shard validation, and every race partition. |
| Windows/amd64 cross-build | verified for compilation only | Wallpaper test binary compilation and `GOOS=windows GOARCH=amd64 go vet ./internal/...` passed; no desktop was changed. |
| macOS 26.6.2 arm64, Go 1.27.1 | partially verified | Mosaic/display/wallpaper packages and build-selected adapter tests passed. `make test-native` otherwise failed only at the existing Linux-master `TestE2E_CopySelection` rendering comparison; no physical wallpaper change was attempted. |
| Windows physical displays | unverified | Requires a supervised Windows host with the listed arrangements. |
| macOS physical multi-display | unverified | Requires an attached external display and supervised wallpaper/options checks. |
| GNOME/KDE/other Linux desktops | unverified | Requires supervised X11/Wayland desktop sessions; automated tests prove fail-closed target handling only. |

The tracker and plan intentionally remain in place pending those physical checks
and branch acceptance. No agent commit was created.
