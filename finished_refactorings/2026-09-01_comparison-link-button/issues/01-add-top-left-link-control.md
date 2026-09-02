# 01 - Add the top-left comparison link control

Status: resolved

## Contract

Add a ready-gated Unlink/Link button and move the existing target-aware
Unlinked status into the same top-left translucent card. Keep the top-right
action card intact. Button and physical `Ctrl+L` must share
`compare.Feature.ToggleLink` and its readiness boundary.

## Acceptance

- [x] The top-left card, loading gate, and unchanged top-right card pass the
      feature-level comparison test.
- [x] Button and physical `Ctrl+L` pass the assembled-viewer equivalence test.
- [x] Open and Swap restore the linked button/status state.
- [x] English and German strings and manuals describe the completed behavior.
- [x] Focused comparison suites and `make verify` pass.

## Comments

- 2026-09-02: Specification approved with `compare.Feature` and assembled
  viewer as the confirmed TDD seams. Two mechanical sub-agents are limited to
  transcribing one pre-designed red test each; production, strings, review,
  fixes, and the final gate remain with the primary agent.
- 2026-09-02: Feature tracer first failed because the comparison overlay had
  no Unlink button. The assembled-viewer tracer first failed because pre-ready
  `Ctrl+L` entered the unlinked state and no Link button existed.
- 2026-09-02: Negative verification proved both new boundaries: removing the
  readiness guard failed the shortcut tracer, and placing the status before
  the button failed its geometry assertion.
- 2026-09-02: Focused compare, viewer, translation, and manual tests passed.
  `make verify` then passed formatting/TUF checks, vet, build, and the complete
  Linux/amd64 race suite.
