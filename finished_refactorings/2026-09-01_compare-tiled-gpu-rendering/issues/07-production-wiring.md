# 07 - Wire production rendering and migrate comparison tests

Status: resolved

## Contract

Use tiled shader renderers in production side-by-side and swipe composition.
Migrate package and assembled UI assertions to the scene seam or shader state.
Preserve all existing comparison behavior while removing interaction-path owner
repaints and full-image canvas scaling.

## Acceptance

The complete compare package and affected assembled UI suites pass, including
fidelity, vectors, link/unlink, divider, swap, resize, cancellation, and 100-event
interaction guards.

## Comments

Production now constructs two tiled shader renderers while package tests retain
the private canvas reference adapter. Comparison and assembled UI assertions
were migrated to scenes or visible shader state. The complete compare package,
recursive comparison integration suite, physical Ctrl+D invocation, shutdown
guard, manual guards, and focused race suite pass.

