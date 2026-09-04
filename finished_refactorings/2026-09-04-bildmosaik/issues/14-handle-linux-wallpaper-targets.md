# Handle Linux wallpaper targets honestly

Type: task
Status: resolved
Priority: P1
Blocked by: 11

## Goal

Preserve the proven global GNOME/KDE setters while refusing a selected-display
request before any global mutation they cannot truthfully honor.

## Existing code anchors

- Current `setLinux` is deliberately global: KDE uses
  `plasma-apply-wallpaperimage`; other sessions set GNOME's
  `org.gnome.desktop.background` `picture-uri` and best-effort
  `picture-uri-dark`.
- `isKDE` checks `XDG_CURRENT_DESKTOP`; KDE falls back to GSettings when the
  Plasma helper is absent. Existing tests pin that exact order.
- `fileURI` already uses `net/url` so spaces, `#`, and `?` are encoded safely.
  None of the current commands accepts ticket 02's opaque display ID.

## Scope

- For every non-empty `Request.Target`, return ticket 11's typed
  target-unsupported error before binary lookup, URI construction, or command
  execution. Do not apply globally and then warn afterward.
- Keep the zero-target code path byte-for-byte compatible where practical:
  KDE-first selection, pre-5.24 GSettings fallback, GNOME light key as required,
  dark key as best effort, and explicit error when neither tool exists.
- Include enough non-sensitive context in the typed limitation for the mosaic
  window to explain that Save Image remains available. Do not expose or parse
  the opaque display ID in user-facing copy.
- Keep display enumeration concerns in `internal/displays`; this ticket changes
  only application scope in `internal/wallpaper`.
- Add tests proving zero command execution for targeted requests on GNOME, KDE,
  unknown desktops, and missing-tool environments. Update Qodana for new test
  paths.

## Acceptance Criteria

- A non-empty target always returns `TargetUnsupportedError` and records zero
  calls to `lookupGsettings`, `lookupPlasmaApply`, or `runWallpaperCommand`.
- No targeted request can report success after a global desktop change.
- Zero-target GNOME still sets `picture-uri` plus best-effort
  `picture-uri-dark`; zero-target KDE still prefers Plasma and keeps its
  existing fallback/error behavior.
- Paths containing spaces, `#`, and `?` retain the existing encoded URI tests.
- UI integration retains the preview and enabled Save action after the typed
  limitation. Ticket 16 records actual GNOME/KDE versions and behavior.

```sh
go test ./internal/wallpaper -run 'Test(SetLinux|LinuxTarget|FileURI)' -count=1 &&
go test ./internal/ui/mosaicwin -run 'TestMosaicWallpaper_TargetUnsupported' -count=1 &&
make check-qodana-test-exclusions
```

## Non-Goals

- Inventing a per-monitor promise for `gsettings` or
  `plasma-apply-wallpaperimage`
- Adding desktop-specific scripting in version 1
- Applying globally without a separate explicit user request

## Comments

This is agent-implementable. Human native checks belong in ticket 16, not in
the implementation ticket's triage status.

Implemented and verified on 2026-09-04: targeted Linux requests return the typed
limitation before lookup/mutation while legacy GNOME/KDE global behavior stays green.
