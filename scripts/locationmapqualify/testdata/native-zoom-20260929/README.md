# Native zoom rejection: retained screen pair

These unmodified PNGs are diagnostic fixtures, not golden renders or accepted
latency evidence. Both are 1200x800 ScreenCaptureKit window captures from the
2026-09-29 synthetic 24-image macOS smoke run. Synthetic photo names and public,
fabricated Berlin-area coordinates are used; no private photo library is shown.
The rendered map's OpenStreetMap contributor attribution is retained in each PNG.

- `gesture-001-before.png`: stable baseline before keypad-plus zoom.
- `gesture-001-after.png`: latest captured frame when the three-second observer
  deadline expired; visibly zoomed, but no qualifying transform was identified.

Source: `.scratch/location-map-native-20260929-complete-local/` on the Mac.
Application source: `33cd7af`; the binaries were built from the equivalent
pre-commit working tree. SHA-256:

- Application: `5d0bb063fa88bf58a4b2a7c792b43de9956ea3a022f23bb2f51d0eda54f7d2a7`
- Helper: `b6be126d47a5bc8cf2a5a90d642dce4b8827103e4fb0ac1343887c5d9f877c66`

Protocol: `screen-v4`, matcher `patch-grid-v1`, zoom key 0x45, Shift false.
Recorded input: 52459275203375 ns (Mach timebase); visible response: 0;
identified: false; skipped: false. Error:
`no identified native response within 3s`.

The frame pair can support offline diagnosis but does not contain the intermediate
frames or their display timestamps. A final retiled frame can differ from the
first response; do not invent latency from this pair or assume it must satisfy
the existing global-scale matcher. See the
[Linux handoff](../../../../docs/location-map-linux-handoff-2026-09-29.md)
for replay instructions and remaining native gates.
