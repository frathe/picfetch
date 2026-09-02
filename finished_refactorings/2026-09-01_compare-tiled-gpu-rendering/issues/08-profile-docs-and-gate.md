# 08 - Profile, document, and run the landing gate

Status: resolved

## Contract

Update architecture/manual/TODO records, perform user-driven macOS side-by-side
and swipe samples on an exact unstripped working-tree build, check memory, and
run the native and final repository gates.

## Acceptance

The profile meets the plan thresholds; `make test-native`, documentation guards,
and one final `make verify` pass. Record that Windows/Linux runtime GPU behavior
was not exercised.

## Comments

The exact unstripped working-tree build was exercised by the user with the same
image pair as the baseline. Physical `Ctrl+D` opened comparison and both modes
were reported visually smooth. Separate 10-second samples measured:

- side-by-side: 1.0 GiB sample footprint and 95.2% main-thread idle;
- swipe: 1.0 GiB sample footprint, 1.1 GiB process peak, and 95.7%
  main-thread idle;
- no Catmull-Rom, `drawNRGBAOver`, or gesture-to-`ForceRepaint` stack in either
  sample;
- Go's live heap stable at 267-271 MiB across repeated collections.

The 1.1 GiB peak is below the 1.2 GiB acceptance ceiling. GPU runtime behavior
on Windows and Linux was not exercised.

The post-fix native suite passed every package except for the existing
Darwin/arm64 antialiasing variance in the Copy Selection golden. That failure
differed at 629 boundary/text pixels, and its generated PNG was byte-for-byte
identical when the same test ran from the parent revision. The authoritative
Linux/amd64 gate first exposed that the shortcut test used duplicate constant
map keys on platforms where the native shortcut modifier is already Control.
After making the expected modifier set portable, the complete `internal/ui`
Linux/amd64 race and golden package passed in 662.436 seconds. The single
`make verify` invocation had already passed formatting, TUF, vet, build, and
every other race package, so the focused rerun completed coverage of the final
tree without spending a second full-gate run.
