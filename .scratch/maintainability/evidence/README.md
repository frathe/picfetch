# Ticket 06 native Linux path transport evidence

Executed on 2026-09-06 in Ubuntu 24.04 Linux/amd64 Docker on the macOS host. Installed backend versions: Zenity 4.0.1-1build3, GTK 4.14.5+ds-0ubuntu0.10, GLib 2.80.0-6ubuntu3.8.

`zenity-transport-probe.py` runs the real Zenity executable, GTK chooser transport and GFile path conversion. Its private session D-Bus portal returns controlled selections of real temporary filenames. It replaces the selection UI; it does not test manual panel interaction or overwrite prompts. It never touches the user's desktop or files. Xvfb supplies the isolated display.

Six successful cases preserve exact UTF-8 bytes: five single-save selections (embedded LF, trailing CR, trailing LF, surrounding spaces and accented/CJK/emoji text), then all five in multi-open selection order. Two cancel cases produce empty output and exit 1. See `zenity-native-results.json` and `zenity-native-run.log` for captured outputs.

The successful probe used:

```sh
docker exec -e DISPLAY=:99 -e XAUTHORITY=/tmp/xvfb-run.6gvm8I/Xauthority 9b5df2da50fc dbus-run-session -- python3 /probe/portal_probe.py
```

That disposable container mounted only a temporary probe directory at `/probe`. To reproduce in a new Ubuntu 24.04 Linux container, install `zenity xvfb dbus-x11 python3 python3-dbus python3-gi`, place the probe at `/probe/portal_probe.py`, and run:

```sh
xvfb-run -a dbus-run-session -- python3 /probe/portal_probe.py
```

The script writes `/probe/native-results.json`; output can be copied to this evidence directory. Under the host's Rosetta Docker emulation, the xvfb-run startup shell stalled despite Xvfb being ready, so the actual recorded command used the already running X server explicitly. Initial keyboard automation and broadcast portal-signal attempts did not pass; directed portal Response signals fixed the harness.

The captured native output was also replayed through the production `zenityResult` and `decodePickedPaths` functions using a temporary Go overlay. Cancellation replays use the existing real exit-status fixture. Result:

```text
=== RUN   TestRecordedNativeZenityTransport
--- PASS: TestRecordedNativeZenityTransport (0.03s)
PASS
ok  github.com/frathe/picfetch/internal/filepicker 0.420s
```

Reproduce that replay from the repository root:

```sh
python3 - <<'PY'
from pathlib import Path
import json, tempfile
root = Path.cwd()
probe = Path(tempfile.mkdtemp(prefix='picfetch-native-replay-'))
source = root / 'internal/filepicker/filepicker_test.go'
target = probe / 'filepicker_test.go'
target.write_text(source.read_text() + '\n' + (root / '.scratch/maintainability/evidence/zenity-decoder-test.go.txt').read_text())
overlay = probe / 'overlay.json'
overlay.write_text(json.dumps({'Replace': {str(source): str(target)}}))
print(overlay)
PY
# Use the overlay path printed above.
PICFETCH_NATIVE_RESULTS="$PWD/.scratch/maintainability/evidence/zenity-native-results.json" go test -overlay /printed/overlay.json ./internal/filepicker -run '^TestRecordedNativeZenityTransport$' -count=1 -v
```

Darwin's actual NSURL serializer separately passes `TestDarwinPathTransport_RoundTripsNativeURLPaths`; deliberately trimming its serialized paths makes that guard fail. The Windows transport guard has cross-compiled but still requires a Windows runtime. These Linux and macOS results do not substitute for it.
