# Use system-provided HEIC decoders

Status: accepted design; implementation pending

Restore HEIC through Apple ImageIO, Microsoft WIC with official codec extensions,
and system-installed libheif with an HEVC decoder on Linux, returning pixels to
PicFetch's shared image path. Accept machine-dependent availability and explicit
installation guidance instead of shipping another HEIC decoder after the
previous decoder's unresolved distribution qualification. Cache refreshable
capability observations and provide Settings actions to recheck support and
open the current OS's installation guide. This choice does not qualify the
native integration or establish legal clearance for the adapter and its inputs.

The [accepted design and qualification requirements](../heic-system-decoding.md)
define the behavior and release evidence. The earlier
[HEIC removal](../../finished_refactorings/2026-09-14-remove-heic-decoder.md)
remains the current runtime state until those requirements are met.
