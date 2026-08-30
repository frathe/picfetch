# Settings Tabs

## Frame

Deliverable: split the existing Settings window into General, Appearance, and
Updates tabs without changing any setting's live behavior.

Route: Standard. This changes one feature package, its tests, localized UI
strings, and the todo/release-note record.

Non-goals: persist the selected tab, redesign controls, or implement the next
static-window-size todo.

## Decisions

| Decision | Choice |
|----------|--------|
| Default tab | General |
| General contents | Every existing control except appearance and update controls |
| Appearance contents | Appearance selector |
| Updates contents | Automatic-update checkbox and manual Check now button |
| Overflow | Each tab scrolls independently inside the existing window size |
| Labels | All tab labels use `lang.L`; English and German bundles stay in parity |

These decisions are fixed for this task.

## Acceptance criteria

AC1. Settings shows General, Appearance, and Updates tabs in that order, with
General selected by default, and every control belongs to the matching tab.

```sh
go test ./internal/ui/settingswin -run TestSettingsTabs
```

AC2. Update controls remain adjacent and manual-update activation remains
single-flight inside the Updates tab.

```sh
go test ./internal/ui/settingswin -run TestUpdateNow
```

AC3. Existing setting seeding, live setters, update flow, geometry, and layout
behavior remain green.

```sh
go test ./internal/ui/settingswin
```

AC4. New labels exist in every locale and English remains an identity map.

```sh
go test . -run '^TestTranslations_'
```

Honest limit: selected tab resets to General whenever Settings is reopened.

## Task 1 — Tabbed Settings layout

Owner: T0 inline

Files: `internal/ui/settingswin/settingswin.go`,
`internal/ui/settingswin/settingswin_test.go`, `translations/en.json`,
`translations/de.json`, `todos.md`, `ARCHITECTURE.md`

Depends: none

Contract: `Window.build` returns three ordered localized tabs; current widget
fields and Host callbacks remain unchanged.

Test: AC1-AC4.

Verify: `go test ./internal/ui/settingswin && go test . -run '^TestTranslations_'`

Budget: 0 spawns, 1 review round, final full suite once.

Delegation: none. User-visible strings and review stay with T0; hot context and
shared files make delegation fail G3-G5. Rule S and Rule W do not apply.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|------|------------------------|---------------|------------|-------|
| T1 | 0 / 0 | 1 | no | T0 inline |
| gate | - | - | yes | Passed; race retry ran outside sandbox because `httptest` needs local ports |
