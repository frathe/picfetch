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

Bugfix 2026-09-05: native testing on a single-monitor GNOME/Wayland (XWayland)
desktop found that mosaic wallpaper could never succeed at all, even though
there was no second display it could have disturbed. mosaicwin always selects
a real target (it requires one to enable Generate), so every mosaic wallpaper
request on Linux hit this ticket's blanket rejection unconditionally. Global
and single-display application are the same operation when only one display
is attached, so this is not the "per-monitor promise" or "silent global
fallback" this ticket's Non-Goals rule out - the user's selection of the one
available display *is* the explicit request. `wallpaper.Request` gained a
`Solo bool` set by the caller (`mosaicwin.Window.SetWallpaper`, from its own
topology snapshot) confirming the target is currently the only attached
display; `setLinuxRequest` honors it and applies globally instead of
returning `TargetUnsupportedError`. Multi-display topologies are unaffected
and continue to refuse a Linux target. See `internal/wallpaper/wallpaper.go`,
`internal/ui/wallpaper.go`, and `internal/ui/mosaicwin/window.go`.

Bugfix 2026-09-05 (second finding): the `Solo` change above was necessary but
not sufficient — the wallpaper still did not visibly change on the reporter's
GNOME 46 desktop. Two further defects, both in the pre-existing GNOME path
rather than in mosaic code:

1. **The key GNOME actually displays was never written.** GNOME 42+ keeps a
   light/dark background pair, and a user with
   `org.gnome.desktop.interface color-scheme = 'prefer-dark'` sees *only*
   `picture-uri-dark`. `setLinux` treated the `picture-uri` write as the whole
   operation and appended a best-effort dark write whose error it discarded,
   documented as "the one thing that makes it fail is running on an older
   GNOME". That premise is false, so the failure mode was the worst possible
   one: picfetch reported success while the desktop kept the old picture.
   The schema is now asked whether it defines `picture-uri-dark`; the key is
   skipped outright on a pre-42 GNOME and every other failure is reported.
   Only an unreadable schema listing stays forgiving.
2. **Sandbox-injected GSettings schemas shadowed the host's.** A snap- or
   flatpak-wrapped launcher redirects schema lookup through
   `GSETTINGS_SCHEMA_DIR`, `XDG_DATA_HOME`, *and* `XDG_DATA_DIRS` at its own
   bundled copies of the host's desktop schemas. The VS Code snap still ships
   a pre-GNOME-42 `org.gnome.desktop.background` with no `picture-uri-dark`,
   so picfetch launched from that session concluded the running GNOME 46
   could not do dark wallpapers at all. Because the wallpaper is a host
   setting, `hostSchemaEnv` now scrubs those three variables for the gsettings
   child while preserving the rest of the environment — the session bus
   address above all, without which nothing reaches dconf.

Verified natively on GNOME 46 / Wayland in dark mode from the affected snap
environment: `picture-uri-dark` now lands on the generated mosaic.
