# Native zoom response diagnostic (unresolved)

These unmodified 300x200 raw luminance arrays come from
`.scratch/location-map-native-after-linux-zoom-frames/gesture-001-before.luma`
and `debug-zoom-1.luma`, the first changed zoom frame captured in that diagnostic
run. The format is the same headerless 4x4 block average documented by the sibling
native pan fixture. The synthetic 24-image corpus contains no private photos.
Whole-window PNGs remain in the Mac-local diagnostic evidence.

The background retains the previous map tiles scaled 2x; photo cards move but
stay the same size. The original matcher selected 125 high-contrast patches,
only 31 of which overlap the scaled interior. Fifteen match the correct transform
at dx -574, dy -394, but fixed-size photo detail dominates the remaining samples.
This fixture precedes replacement tiles, unlike the original timeout PNG pair.

The current matcher still rejects this mixed-scale scene. Lowering the source
contrast floor from 45 to 16 retained more map detail but still matched only
24/40 patches (60%); reversing registration matched 34/84. Neither meets the
unchanged 65% acceptance rule. These experiments are not shipped. Supporting
the scene requires distinguishing scalable background from fixed-size cards,
not just lowering the acceptance threshold.

This is an unresolved diagnostic fixture, not native performance qualification
or a passing regression case. Diagnostics change observer workload; the failed
run and original timestamps remain retained.
