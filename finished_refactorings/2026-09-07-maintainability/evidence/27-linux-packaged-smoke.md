# Linux packaged-executable smoke, 2026-09-07

Both refreshed Linux packages rendered the generated Alpha/Beta fixtures and
quit with exit code 0 and empty application logs. The ARM64 package also passed
production comparison side-by-side and swipe rendering, then quit while
comparison was open. No application or renderer source was modified for these
runs.

## Environments and limits

| Property | ARM64 run | amd64 run |
|---|---|---|
| Execution | Native ARM64 Linux process in Docker Desktop's ARM64 Linux VM | Linux amd64 process through Docker CPU emulation on the same ARM64 host |
| Userspace | Debian 12 bookworm | Debian 12 bookworm |
| Display | Xvfb 1280x960, 96 DPI; Openbox | Xvfb 1280x960, 96 DPI; Openbox |
| Renderer | Mesa 22.3.6, llvmpipe / LLVM 15.0.6, 128-bit vectors | Mesa 22.3.6, llvmpipe / LLVM 15.0.6, 256-bit vectors |
| OpenGL / GLSL | 4.5 / 4.50 | 4.5 / 4.50 |
| Acceleration | Software | Software |
| Executable SHA256 | `8e268fe7ca23ef76b1b09d08805bb86e88e43e181f67f57164fd49fb81e9abe0` | `efa8fd527289465f58fe841b44515e544994f74c7c0d27cedb18188728b25127` |
| Terminal result | Exit 0, empty log | Exit 0, empty log |

[Runtime image identities](27-linux-runtime-images.txt),
[ARM64 graphics/linked-library/fixture records](27-linux-arm64-results/environment.txt),
[amd64 records](27-linux-amd64-results/environment.txt).
Both hashes match the [reviewed refreshed artifacts](27-artifact-followup-inspection.txt).
These are actual production executables and GL rendering on the target OS,
viewed through a local noVNC connection. They do not use Fyne's test driver or
canvas-reference renderer. This is bounded Linux software-GL coverage, not a
hardware-accelerated GPU or physical x64 hardware claim. The separate Ubuntu
UTM desktop was not used; its prepared smoke ISO remains available.

## Setup and repetition

The two disposable containers used localhost-only ports 6081 and 6082. The
packaged binaries and generated fixtures were mounted read-only; settings,
cache, session bus and runtime directories were isolated within the test
containers/result folders. The actual app identity was preserved.

For each architecture, start its recorded base image with `--platform
linux/arm64` or `--platform linux/amd64`, mount the disposable checkout's `bin/`
at `/input:ro`, the generated fixture directory at `/fixtures:ro`, and a fresh
host result directory at `/results`. Publish only `127.0.0.1:6081:6080` (ARM64)
or `127.0.0.1:6082:6080` (amd64). The container command is `sleep infinity` and
`--rm` removes its writable runtime after stopping.

Install desktop prerequisites inside that disposable container:

```sh
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends \
  xvfb x11-utils openbox x11vnc novnc websockify mesa-utils libgl1-mesa-dri \
  libxcursor1 libxrandr2 libxinerama1 libxi6 libxxf86vm1 libxkbcommon0 \
  libwayland-client0 fonts-dejavu-core dbus-x11 locales
localedef -i en_US -f UTF-8 en_US.UTF-8
```

Copy the [desktop launcher](27-linux-container-desktop.sh.txt) into `/results`
and run it with `docker exec -d <container> sh /results/start-desktop.sh`.
For amd64 change only the executable name in that launcher's metadata commands
from `picfetch-linux-arm64` to `picfetch-linux-amd64`. It waits for a successful
`xdpyinfo` response before starting the window manager and local viewer, then
writes `desktop-ready.txt` after collecting environment records.

Copy and run the corresponding [ARM64 app launcher](27-linux-container-arm64.sh.txt)
or [amd64 app launcher](27-linux-container-amd64.sh.txt). Each invokes the
unchanged package with both 8192x6144 fixtures under `dbus-run-session`, with a
valid locale and existing isolated config/cache directories. A completed
`exit.txt` records the actual application exit; no delay guesses completion.

Open `http://127.0.0.1:6081/vnc.html` or port 6082 and click Connect. If the
viewport clips the remote desktop, choose noVNC Settings, Scaling Mode,
Local Scaling. This scales the viewing surface; it does not change Linux's
reported display DPI. Use the Browser skill for screenshots and input.

## Observed actions

1. ARM64 opened [Alpha](27-linux-arm64-alpha.png), including all four corner
   labels and the 8192x6144, 1/2 title. Right switched to
   [Beta](27-linux-arm64-beta.png) with the magenta marker and 2/2 title.
2. G opened Grid. Ctrl+A selected both fixtures; Ctrl+D opened
   [comparison](27-linux-arm64-comparison.png). Both source images fitted
   correctly with their corner geometry, labels and Beta marker.
3. Click Swipe. The [actual shader output](27-linux-arm64-swipe.png) shows
   aligned full-viewport sources and the midpoint reveal edge. Close the native
   window while comparison remains open: [exit 0](27-linux-arm64-results/arm64-r2/exit.txt),
   [empty app log](27-linux-arm64-results/arm64-r2/process.txt).
4. amd64 opened [Alpha](27-linux-amd64-alpha.png), and Right switched to
   [Beta](27-linux-amd64-beta.png), preserving all four landmarks, image
   dimensions and the second-image marker. Close the native window:
   [exit 0](27-linux-amd64-results/amd64/exit.txt),
   [empty app log](27-linux-amd64-results/amd64/process.txt).

Browser key input used the focused noVNC canvas. For compound shortcuts,
`canvas.press('Control+a')` / `canvas.press('Control+d')` delivered the expected
native actions; the initial generic Control+A attempt produced no selection
and was not counted as a pass. Screenshots were inspected after the remote
frame arrived, rather than treating an immediate stale frame as the result.

## Setup corrections retained

The first ARM64 launch rendered Alpha and quit 0, but logged missing temporary
cache parents, a C-locale parsing error and missing `dbus-launch`. Those were
incomplete disposable-desktop prerequisites. The original
[log](27-linux-arm64-results/arm64/process.txt),
[exit](27-linux-arm64-results/arm64/exit.txt) and
[screenshot](27-linux-arm64-initial-startup.png) are retained. Precreating the
isolated directories, generating a valid locale and starting a session bus
produced the accepted repeat with the identical executable and empty log.
The amd64 display setup separately needed `x11-utils` for the readiness probe;
no PicFetch process was launched until that succeeded.

No golden image was regenerated, system desktop changed, artifact published or
production source edited. The prior full common gate therefore remains the
source verification result. Temporary browser tabs and both containers were
closed after retaining the completed process records.
