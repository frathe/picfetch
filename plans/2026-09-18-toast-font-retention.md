# Toast font retention during repeated image errors

Route: Standard. Lead owns the diagnosis, tests, implementation and review.
One read-only scout traced local Fyne cache ownership. No dependency changes.

## Evidence and scope

The user's bounded capture reached the 4 GiB process-group limit in 51.589s.
The last heap snapshot attributed 94.68% of retained allocations to
`painter.CachedFontFace`, almost entirely the bold theme font. Recorded stacks
show repeated image-load failures; the user confirms many invalid JPEGs.
Fyne 2.8 creates a new scope on each `ThemeOverride.Refresh`, retaining parsed
fonts under every scope. PicFetch's root repaint refreshes the toast override.
This path predates PR #45. The separate Favorite preview-reuse regression is
tracked in `2026-09-18-favorite-preview-reuse.md`; fixing the toast alone does
not establish restored Grid responsiveness.

## Acceptance and work

1. Repainting and replacing a visible toast reuses already parsed fonts.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui -run '^TestToast_'`.
   Count actual font-resource reads through the real canvas painter, including
   root refreshes. Establish failure before applying the fix.
2. Toast text remains bold, centered, wrapped and readable on its dark card
   across message replacement and application theme changes.
   Verify through the same focused tests and existing Explorer toast wrapping
   regression (`TestVisualSimilarityExplorer/resource_limit_toast`).
3. The changed files pass formatting, GoLand inspections and UI shard/Qodana
   manifest checks. Full race verification remains on GitHub CI.
4. Rebuild the temporary profiler after the fix and compare the same bounded
   interaction when available; do not claim the original crash verified fixed
   from the synthetic test alone.

Files: `internal/ui/toast.go`, `internal/ui/toast_test.go`, test shard manifest,
`qodana.yaml`, `todos.md`, this record. No new background workers or UI strings.
Budget: one scout (actual one), lead-only fixes, focused local tests, full CI.
Local profiles and user source paths stay outside committed evidence.

## Local verification

- Before the fix, five toast replacements/root repaints caused five additional
  font-resource reads through the real canvas painter. The regression failed
  for that reason. With a persistent toast theme scope, the same test passes
  with no additional reads.
- Focused race tests passed (8.108s). Wrapped text stays bold and readable under
  Light/Dark appearance changes and short/long message replacement. The existing
  Explorer resource-limit toast regression passed (0.129s).
- GoLand reported no findings, including warnings, in both changed Go files.
- Formatting, Qodana test exclusions, the 691-entry UI shard manifest, vet and
  build passed. No dependencies changed; no source instrumentation ships.
- The original profiling binary is preserved locally. The fixed diagnostic
  was rebuilt; the user's equivalent bounded capture is pending.
- Full CI, Qodana, CodeQL and the security review passed on `4eb5300`.
  Its code review identified two preview-cache edge cases, handled in the
  linked plan. No new toast finding was reported.
