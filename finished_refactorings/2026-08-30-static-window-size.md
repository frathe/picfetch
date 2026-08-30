# Static window size toggle

## Frame

Deliverable: Settings toggle that freezes auto window resize; manual
resizes still persist (existing tracker).

Route: Standard.

## Decisions

| Decision | Choice |
|----------|--------|
| Pref key | `staticWindowSize`, default false |
| Settings tab | General, after separator before merge check |
| Label | `Keep a fixed window size` |
| Guard | `viewer.autoResizeToImage` wraps all production `resizeToImage` call sites |
| Manual size | Unchanged: `windowSizeTracker` + `currentPreferences` |

## Acceptance criteria

AC1. Pref round-trips; unset loads false.

```sh
go test ./internal/preferences -run 'TestLoadPreferences_NothingSaved|TestSavePreferences_RoundTrip'
```

AC2. Settings seeds and sets the toggle without seed round-trip.

```sh
go test ./internal/ui/settingswin -run 'TestShow_Seeds|TestStaticSizeCheck|TestSettingsTabs'
```

AC3. With toggle on, load / zoom / rotate leave window size alone; wiring
restores and `currentPreferences` reflect it; size still tracks manual resize.

```sh
go test ./internal/ui -run 'TestStaticWindowSize|TestWindowSizeTracker|TestSyncWindowToZoom_Static|TestRotate.*Static|TestSetStaticWindowSize|TestCheckForUpdates_Defaults'
```

AC4. Locale parity + no Unicode arrows.

```sh
go test . -run '^TestTranslations_'
```

## Task 1 — Implement

Owner: T0 inline. Files: preferences, settingswin, load/rotate/features/run/
memlimits, translations, help/manual.md, todos.md, ARCHITECTURE.md.
