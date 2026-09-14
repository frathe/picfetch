# Single-image presentation owns its image surface

For [MA-027](../../needs_refactoring.md#ma-027), deepen the existing display
module to own frame state, pixel installation, refresh and fades together.
Keeping canvas publication in the viewer would preserve ordering dependencies
across the extraction, so the viewer instead composes the module's surface
with zoom and retains collection navigation, window policy and cross-feature
decisions. This is the accepted design direction; migration proceeds in
verified stages, with loading and preloading included in MA-027's final scope.
