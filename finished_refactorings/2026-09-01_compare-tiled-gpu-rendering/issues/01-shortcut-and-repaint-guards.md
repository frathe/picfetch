# 01 - Pin shortcut parity and repaint-free interaction

Status: resolved

## Contract

Register physical Ctrl+D in addition to the platform-default D shortcut without
duplicating the same chord. Add a sustained pan/zoom regression guard proving
100 interaction events do not invoke the owner repaint callback.

## Red / green

1. Add tests for both shortcut chords and the 100-event interaction boundary.
2. Observe failure against current shortcut wiring and repaint calls.
3. Make only the smallest behavioral change needed; retain lifecycle repaints.

## Comments

Native baseline confirms the forbidden interaction path ends in
`viewer.ForceRepaint` and full-image Catmull-Rom resampling.

The shortcut guard failed with no physical-Control registration on macOS. The
interaction guard reported exactly 100 owner repaints for 100 mixed pan/zoom
events in each layout. Both focused tests pass after conditional shortcut
registration and removal of the six interaction-path repaint calls.
