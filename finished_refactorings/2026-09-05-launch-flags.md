# Startup flags for scripting and picture-frame use

**Route:** Standard. One feature, one new leaf package plus `main.go` glue and
`internal/ui` wiring. One existing contract changes: `ui.Run` gains a third
parameter. No cross-package refactor.

Closes `todos.md` item 9.

## Problem

`picfetch` already accepts paths on the command line — `argsToURIs` resolves
them and `ui.Run` opens them — but nothing else. Every standing preference the
Settings window drives (sort order, merge mode, picture-frame interval and
shuffle, scan cap) can only be set by clicking, and picture-frame mode can only
be entered by pressing `P` on a running app.

That rules out the two deployment stories the setters otherwise already
support: an autostarted photo-frame appliance (a Pi launching
`picfetch --slideshow --shuffle --interval=20s /srv/photos`), and shell
workflows that want one-off ordering without disturbing saved defaults.

## Decisions

Resolved with the user before implementation; do not relitigate.

| Question | Decision |
|---|---|
| Persistence | **Session-only.** A flag applies to that launch; at shutdown the overridden fields save their *pre-flag* values. A scripted `picfetch --shuffle` must not silently leave shuffle on forever, which is what calling the setter and letting `currentPreferences` save the live value would do. |
| Unknown argument | Print the error and usage to stderr, exit 2. A typo in a Pi autostart unit should fail loudly, not launch with the flag ignored. |
| `-psn_*` | Ignored, never an error. macOS LaunchServices can pass `-psn_0_12345` to a GUI app; under a strict parser that would abort the launch. Today it becomes a bogus URI and is harmlessly skipped, and that must not regress. |
| Flag name for the scan cap | `--max-files=n`, not `todos.md`'s `--recursive-limit=n`. The cap it drives (`viewer.SetMaxScan`) also bounds `filescan.Siblings`, the *non-recursive* sibling expansion of a single opened image, so "recursive" claims more than the concept holds. |
| Parser | Hand-rolled in `internal/launch`, not `flag.FlagSet`. `flag` stops parsing at the first non-flag argument, so `picfetch ~/photos --slideshow` would take `--slideshow` as a path. Flags must work anywhere among paths. |
| `Options` field shape | Pointers. "Unset" has to be distinguishable from "set to the zero value" — `--merge=false` and no `--merge` at all mean different things, and the session-only restore is driven entirely by which fields are non-nil. |
| Package placement | New leaf `internal/launch`. `ui.Run` needs the `Options` type, so it cannot live in package `main`; and AGENTS.md pins `main.go` to app setup plus path conversion, which a ~100-line parser plus usage text would break. Matches the existing `winpos`/`wincom`/`distribution` shape. |
| Usage/error language | English, no `lang.L`. The AGENTS.md convention covers what the app draws; these strings go to stderr for a shell or a systemd journal, and a locale-dependent flag error is hostile to scripting. |
| `--sort` vocabulary | `name|date|modified|size|drop` — the `preferences.SortBy*` constants that already persist the setting, mapped by the existing `filesort.FromPref`. `Parse` validates against that set, because `FromPref` silently maps anything unknown to `ByName`, which would turn `--sort=dtae` into a silent wrong answer. |
| Bool flags | `--merge` and `--merge=true|false`. No space-separated form: `--merge false` would have to eat the next argument, which is a path. Matches `flag`'s own rule. |
| Picture-frame timing | Armed at startup, fired from `applyScannedFiles`' reorder callback once files exist. `slideshow.Toggle` no-ops at zero files, so applying `--slideshow` during startup would do nothing at all. |

## Non-goals

- The in-app manual (`internal/ui/help`). Documenting flags there means new
  `lang.L` keys in every `translations/*.json`; the README is the right home
  for a shell surface.
- `--version`, `--fullscreen`, or any flag that is not already a viewer setter.
- Kiosk mode. `CONTEXT.md` leaves that word unclaimed on purpose: nothing here
  suppresses the app's own exits, and `--slideshow` remains leaveable with `P`
  or `Escape`.

## The honest limit

The session-only restore is per *field*, not per *change*. If a run started
with `--sort=date` and the user then picks a different sort order in the
Settings window, that choice is not persisted for that run — the pre-flag value
wins at exit. The other four fields save normally, and any field no flag
touched is unaffected. Tracking "the user changed it after the flag did" would
mean a dirty bit on five settings for a case that costs one relaunch.

## Acceptance criteria

```
AC1  launch.Parse takes flags anywhere among paths, accepts --x=v and --x v
     for value flags, honours the -- terminator, ignores -psn_*, and returns
     ErrHelp for --help/-h.
     go test ./internal/launch/

AC2  launch.Parse rejects an unknown flag, an unparseable duration or int, and
     a --sort value outside name|date|modified|size|drop.
     go test ./internal/launch/

AC3  main's launch helper exits 0 on --help, exits 2 with usage on stderr for
     a bad flag, and otherwise signals "keep going" with the parsed paths.
     go test . -run TestLaunch

AC4  applyLaunchOptions drives the existing setters: sort mode, merge mode,
     slideshow shuffle and interval, and the scan cap. A zero Options changes
     nothing.
     go test ./internal/ui/ -run TestLaunchOptions

AC5  currentPreferences saves the pre-flag value for every flag-overridden
     field and the live value for every other field.
     go test ./internal/ui/ -run TestLaunchOptions

AC6  --slideshow enters picture-frame mode after the launch file set loads,
     and exactly once - a later drop does not re-enter it.
     go test ./internal/ui/ -run TestLaunchOptions

AC7  Every new *_test.go file is excluded from Qodana duplication and every
     new internal/ui top-level test has a shard assignment.
     make check-qodana-test-exclusions && make check-test-shards

AC8  The tree still builds, formats, vets, and passes the full race suite.
     make fmt-check && go vet ./... && go build ./... &&
     go test -timeout 30m -race ./...
```

## Tasks

### Task 1 — `internal/launch`
Owner:   T0 inline
Files:   create `internal/launch/launch.go`, `internal/launch/launch_test.go`
Depends: none
Contract:
```go
type Options struct {
	PictureFrame bool
	Sort         *string
	Merge        *bool
	Shuffle      *bool
	Interval     *time.Duration
	MaxFiles     *int
}
func Parse(args []string) (paths []string, opts Options, err error)
func Usage() string
var ErrHelp = errors.New("launch: help requested")
```
Test:    interleaving, both value forms, `--`, `-psn_*`, help, and one case per
         rejection in AC2.
Verify:  `go test ./internal/launch/`
Budget:  0 spawns · 1 review round · full suite: no

### Task 2 — `main.go` glue
Owner:   T0 inline
Files:   modify `main.go`, `main_test.go`
Depends: 1
Contract: `func launchArgs(args []string, stdout, stderr io.Writer) (paths []string, opts launch.Options, exit int)`, where `exit < 0` means "keep going".
Test:    the three exit paths.
Verify:  `go test . -run TestLaunch`
Budget:  0 spawns · 1 review round · full suite: no

### Task 3 — `internal/ui` application and session-only save
Owner:   T0 inline
Files:   create `internal/ui/launchoptions.go`, `internal/ui/launchoptions_test.go`;
         modify `internal/ui/run.go`, `internal/ui/viewer.go`, `internal/ui/drop.go`
Depends: 1
Contract: `func Run(application fyne.App, initial []fyne.URI, opts launch.Options)`;
          `func (v *viewer) applyLaunchOptions(opts launch.Options)`;
          `func (v *viewer) startPendingPictureFrame()`.
Test:    AC4, AC5, AC6.
Verify:  `go test ./internal/ui/ -run TestLaunchOptions`
Budget:  0 spawns · 1 review round · full suite: no

### Task 4 — Gates and docs
Owner:   T0 inline
Files:   modify `qodana.yaml` (via `make sync-qodana-test-exclusions`),
         `.github/testshards/internal-ui.tsv`, `README.md`, `ARCHITECTURE.md`,
         `AGENTS.md`, `todos.md`
Depends: 1-3
Verify:  `make check-qodana-test-exclusions && make check-test-shards`
Budget:  0 spawns · 1 review round · full suite: yes (final gate)

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|------|------------------------|---------------|------------|-------|
| 1    | 0 / 0                  | 1             | no         | hot context |
| 2    | 0 / 0                  | 1             | no         | hot context |
| 3    | 0 / 0                  | 1             | no         | hot context |
| 4    | 0 / 0                  | 1             | yes        | final gate |

Zero spawns, zero scouts: every recon question died in the shell (§2), and
every finding was the Lead's to fix inline (§7). Four guards were negatively
verified - `-psn_*` tolerance, the usage-documents-every-flag check, the
session-only restore, and the spent-request-on-a-failed-launch case - each
seen failing with its behaviour removed, then restored.

Review was T0 only, per §6 and §8: `gofmt`, `go vet`, `GOOS=windows go vet`,
every AC command, the negative guard pass, `git diff --stat` against the file
map above, then the diff itself. `/implement`'s delegated `/code-review` step
was deliberately not taken - this document forbids delegating review, and
AGENTS.md outranks the skill.
