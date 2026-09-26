# Native Location Map qualification

These tools collect native macOS observations and validate retained evidence.
Unit-test timings and small synthetic collections do **not** qualify 10k or 30k.
The screen helper is compiled from this directory using Apple's system SDK;
no new third-party runtime or distributable asset is introduced.

## Human-controlled session

For a session where the user loads images and performs all clicks:

```sh
make build
go run ./scripts/locationmapqualify manual \
  -binary ./bin/picfetch -evidence /explicit/new-evidence-directory -timeout 30m
```

This opens an empty, isolated client without a screen/input helper or extra OS
permissions. Load the chosen collection, open Location Map, browse, return from
images/clusters, and close/reopen the map. Quit the client to finish. The tool
records the binary SHA-256, PID, source-free app state and sampled RSS; live
observations are in `observations.jsonl`, with a final `manual-report.json`.
The console and isolated app storage may contain private library paths. Keep the
whole evidence directory local. Interrupted/failed sessions are retained; never
reuse an evidence directory.

This is not input-to-visible latency measurement and cannot pass the formal
evidence checker. It generates no human verdict. A user can supply an assessment
of this exact build separately, but the agreed measured latency gate remains
open until independently measured or explicitly changed by the maintainer.

## Run

Choose the image directory explicitly. Use a fresh evidence directory whose
parent already exists. The run never writes image files. The launched app uses
isolated preferences, Favorites, analysis storage and update settings; it does
not change the normal PicFetch session. Evidence includes visible photo
thumbnails, filenames and map locations: keep it local unless you intend to
share those details. Visible map tiles use the normal OSM network path.

```sh
make location-map-qualify \
  LOCATION_MAP_IMAGES=/explicit/collection \
  LOCATION_MAP_EVIDENCE=/explicit/new-evidence-directory
```

This launches a real 1200x800 PicFetch window and sends it native keyboard
input. Keep that window foreground and do not interact during the run. The
helper requires existing Screen Recording and Accessibility/post-event
permissions. It checks permissions without requesting or granting them; a
missing permission fails the run and requires the human to configure it.
Agent-driven execution of the CGEvent helper requires explicit user approval
under the Computer Use workflow. Default deadline is 30 minutes; override with
`LOCATION_MAP_TIMEOUT=45m`. Interruptions and failures retain `report.json`,
`native-console.log`, app state and any observations/artifacts already collected.
Re-runs require a new evidence directory, never overwrite a failed run.

The collection count comes from the app's admitted file set, not an argument.
Cold means the first map entry in this isolated process, not a flushed OS disk
cache. Warm means a second complete entry in the same process. Stage timings
record duplicate preparation and metadata scanning on their actual UI path;
they are never substituted for paint observations.

## Visible-response protocol

The independent helper captures complete ScreenCaptureKit window frames,
requesting 120 fps (actual cadence depends on the display/system). It timestamps
keyboard submission using Mach absolute time and uses WindowServer's frame
`displayTime` in the same timebase, not media PTS or worker completion. Before
each pan/zoom or map entry it requires 250 ms of stable body pixels; failure to become
stable is retained as a failed sample. It samples RGB pixels every third pixel
inside the central 80% by 60% of the window, excluding pointer/bars/toasts, and
retains whole-window before/after PNGs for each measured gesture.

Formal gesture qualification is currently unavailable: the capture helper refuses
pan/zoom measurements because its body hash cannot identify the requested
transform independently of background tile delivery. A native run therefore
stops with an explicit error at its first gesture and cannot produce a qualifying
report. Manual browsing and stage/RSS observation remain available. Reliable
visual transform correlation is tracked in `todos.md`; the release's existing
maintainer performance acceptance is separate from measured latency evidence.

The protocol requires at least 40 alternating horizontal Shift+arrow pans and
in/out zooms after complete cold and warm scans. Every submitted measurement is retained, including slow
or failed measurements. Another entry followed immediately by Escape measures
visible cancellation/exit feedback. Exit frames must match the stable closed-viewer
body captured before that entry; unrelated scan/tile changes cannot complete the
sample. Missing or changed baselines (for example, an animated background photo)
fail closed rather than producing a successful latency measurement. App state
independently confirms retirement; it never supplies or corrects the pixel timestamp.
This does not claim an uninterruptible filesystem read has already returned.
For 30k, browsing continues for at least one further minute after the first
40 gestures, with RSS observation throughout that interval,
and at least three complete open/close cycles are recorded.

Capture cadence quantizes latency. Changed body pixels alone do not identify
a gesture. Both collection and report validation require explicit gesture
identification; older reports without it no longer qualify. The current helper
does not set that evidence. Capture and PNG
encoding run in a separate process but consume system resources. `/bin/ps`
samples application RSS every 250 ms, so reported peak RSS is a sampled peak.
Hardware, storage description, format counts, binary SHA-256, helper SHA-256 and
protocol description are retained with results. The checker validates evidence
consistency, not cryptographic attestation of its provenance.

## Check

```sh
make location-map-qualification-test
make location-map-check-evidence \
  LOCATION_MAP_EVIDENCE=/explicit/evidence-directory \
  LOCATION_MAP_EXPECTED_IMAGES=10000
```

The checker hashes the current built application and rejects wrong-build,
wrong-count, partial, skipped, missing-screen-artifact or inconsistent evidence.
Use the **actual** admitted count when checking smoke data; also demonstrate that
the same small report fails with expected count 10000 and 30000. Exactly 10k
requires at least 95% of all pan/zoom samples within 100 ms and every observed
cancellation within 250 ms. Scan durations have no universal deadline.

30k requires sustained memory evidence, repeated open/close and Ronin's own
verdict; it does not apply the 10k latency promise. After running and reviewing
the app/evidence, Ronin records `"verdict_by": "Ronin"` and `"verdict": "pass"`
or `"fail"` in `report.json`. The tool never supplies that judgment. Only a
passing verdict satisfies the 30k gate; a failure stays a failure.

`runner_test.go` and `evidence_test.go` author synthetic protocol fixtures only.
They establish tool contracts, never native observations or feature acceptance.
