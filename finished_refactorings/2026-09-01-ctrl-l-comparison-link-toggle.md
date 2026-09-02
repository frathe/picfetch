# Ctrl+L comparison link toggle

Status: complete

Route: Standard. This corrects one comparison interaction across
`internal/ui/compare`, viewer input wiring, Favorites shortcut presentation,
the manuals, and architecture notes. It adds no package, dependency,
preference, or external API.

Deliverable: exact physical `Ctrl+L` toggles retained pane-local views until a
second press relinks from the last-hovered pane through the shared no-blank
clamp.

## Locked decisions

| Decision | Contract |
|---|---|
| Shortcut | Physical `Ctrl+L` on every desktop platform, including macOS; extra modifiers do not match. |
| Lifecycle | New comparisons start linked. Control press/release alone has no effect. |
| Target and relink | Local input uses the hovered or last-hovered pane; relinking adopts that pane and applies the shared clamp. |
| Existing transitions | Resize and layout preserve local views. Swap relinks and resets divergence. |
| Feedback | Reuse `Unlinked`, `Unlinked: Left`, and `Unlinked: Right`; add no button or preference. |

## Tasks

### Task 1 - Toggle state and pane-local input

Owner: T0 inline

Test first through `compare.Feature`: persistent local pointer/key input,
status, relinking/clamping, retained caches, layout/resize, Swap, and vector
rendering.

Verify: `go test ./internal/ui/compare -count=1`

### Task 2 - Physical key edge and shortcut cleanup

Owner: T0 inline

Test first through the assembled viewer key hook: exact physical `Ctrl+L`, no
Control-release effect, no repeat flapping, unmodified transform/divider keys,
hook chaining, and unchanged Favorites availability.

Verify: `go test ./internal/ui ./internal/ui/favorites -count=1`

### Task 3 - Documentation and final gate

Owner: T0 inline

Update both manuals, `ARCHITECTURE.md`, and `todos.md`; preserve the completed
hold-to-unlink plan as historical evidence.

Verify: `go test ./... -run 'Translations|Manual|UnicodeArrows' -count=1`

## Budget and gate

Zero spawns; at most three review rounds; one full suite. Negatively verify the
exact-toggle guard before the final `make verify` run.

## Outcome

Comparison now starts linked and exact physical `Ctrl+L` persistently toggles
the retained pane-local views. Control release is inert, unmodified comparison
gestures and transform keys target the last-hovered pane while unlinked, and a
second toggle relinks from that pane through the shared clamp. The obsolete
modified-key shortcuts and Favorites accelerator workaround were removed.

Verification completed with focused comparison/viewer/Favorites tests, manual
and translation guards, Windows-targeted vet, a negative exact-modifier
mutation, and the full Linux/amd64 race-backed `make verify` gate.
