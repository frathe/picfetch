# Native subpixel pan search regression

`before.luma` and `after.luma` are the observer's unmodified 300x200 luminance
arrays from the synthetic 24-image macOS diagnostic run at
`.scratch/location-map-native-after-linux-frames/`. Each byte is the integer
average of one top-down 4x4 BGRA block using `29*B + 150*G + 77*R`, divided by
`16*256`; there is no header. Sources: `gesture-000-before.luma` and
`debug-pan-0.luma`. Whole-window PNGs remain in that Mac-local run.

The map background is a stationary checkerboard in both frames; the distributed
synthetic photo cards move horizontally. The requested Shift+Left pan is
independently identified at scale 1, dx 58, dy 0 with 33/48 patches after
refinement. The two-sample coarse grid had only 14/48 matches at dx 56 and
discarded this candidate before refinement. Preserve strict final coverage,
spatial spread and ambiguity checks.

These are matcher fixtures, not latency evidence. The instrumented run failed
and remains retained; no native qualification is inferred from replay.
