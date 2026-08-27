# macOS "Open With" and Dock-icon drops — retrospective

> Written after all five stages of `plans/2026-08-27-macos-open-with.md`
> landed and the full verification gate passed. That plan carries the
> day-one diagnosis, the stage-by-stage task list, and the manual-QA
> checklist; it is left as-is. This file records the parts of the finished
> design that would not survive a read of the diff alone — mainly *why*,
> not *what*.

**Fixed:** `todos.md`'s *"on MacOS the basic functionality open with does
not work also drag&drop of images onto the (not running) App does not
work"*. macOS never puts files in `argv` for a bundled `.app` — "Open With",
a drag onto the Dock icon, `open -a`, and double-clicking an associated file
all arrive as a `kAEOpenDocuments` Apple Event that AppKit turns into a
delegate call, and GLFW's delegate implemented no such method, so every one
of them was silently ignored.

**Shipped, in five parts:**

```
internal/imaging      SupportedExtensions() — extracted from IsSupportedImage
scripts/plistdoctypes  derives Info.plist CFBundleTypeExtensions from that list
                        (replaces scripts/patch_macos_document_types.py;
                        7 declared extensions -> 28, plus public.folder)
internal/openwith      viewer-independent queue + file:// URL parsing
  openwith_darwin.*     the Objective-C graft (cross-platform core is Go)
internal/ui/openwith.go + run.go   wiring: openwith.Install() first in main(),
                        SetOnStarted installs the handler and opens argv files
```

## Why `class_addMethod` on GLFW's class, not our own delegate or `NSAppleEventManager`

Two AppKit facts rule out the more obvious designs, and both are recorded
right at the point of use (`internal/openwith/openwith_darwin.m`'s file
banner and `Install`'s doc comment in `openwith_darwin.go`) so nobody has to
rediscover them by trial and error:

- **`-[NSApplication setDelegate:]` caches which selectors the delegate
  answers, at the moment it is called.** GLFW calls `setDelegate:` inside
  `glfw.Init()`. A delegate method added afterwards — including a whole
  second delegate object swapped in later — is never consulted, no matter
  that `respondsToSelector:`/the Objective-C runtime can now find it on the
  class. The graft therefore has to land on the *class GLFW's instance
  already is*, and it has to happen before `glfw.Init()` runs. That is why
  `openwith.Install()` is the first statement of `main()`, before
  `app.NewWithID` — which itself doesn't create a window and so doesn't
  reach `initGLFW` — rather than an `init()` func or a call from inside
  `internal/ui`.
- **AppKit installs its own `kAEOpenDocuments` handler during
  `-finishLaunching`.** Registering an `NSAppleEventManager` handler ahead
  of that point doesn't survive it — AppKit's own installation clobbers it.
  `class_addMethod` sidesteps the question entirely: it modifies the class
  object itself rather than registering anything with an event manager that
  AppKit is going to rewrite anyway.

`class_addMethod` costs about 30 lines of Objective-C
(`openwith_darwin.m`) against either alternative, and the target class name
is a string looked up with `objc_getClass` rather than referenced at
compile time — the same file has to build cleanly in a binary that doesn't
link the Cocoa driver at all (`internal/openwith`'s own non-darwin test
binary), where the graft is just a no-op.

## Why the queue exists at all

The Apple Event fires **inside `glfw.Init()`**, which Fyne reaches from
`ui.Run` → `buildStartupViewer` → first window creation — well before
`application.Lifecycle().SetOnStarted` in `run.go` ever runs. A cold-start
"Open With" or Dock drop therefore has nowhere to go at the moment AppKit
delivers it: no viewer, no `handleDrop`, not even a `*viewer` value yet.

`internal/openwith`'s `queue` (`openwith.go`) is what bridges that gap. Its
whole contract is one property: `Deliver` and `SetHandler` both take the
same mutex, and `SetHandler` drains whatever is pending *inside* that
critical section, handing it to the handler it just installed before
releasing the lock. A separate "drain, then install" pair would lose
anything `Deliver`-ed in the gap between the two steps; doing both under one
lock acquisition is the actual fix, not an incidental cleanup. The trade-off
recorded in the same file: ordering is only guaranteed per caller, not
across concurrent ones — fine for the one production caller (the Apple
Event callback and the `SetOnStarted` install both run on AppKit's main
thread), and flagged in the doc comment as a real constraint for any future
caller that isn't.

## `fyne.Do` is safe from the AppKit main thread — which inverts this package's own rule

`AGENTS.md`'s standing rule is "marshal background UI updates through
`fyne.Do`" — the implicit assumption everywhere else in the codebase being
that the call originates *off* the UI goroutine. `installOpenWithHandler`
(`internal/ui/openwith.go`) breaks that assumption on purpose: its handler
runs on whichever thread the Apple Event delivered on, and for the
production caller that thread *is* AppKit's main thread, which is also
Fyne's UI goroutine.

That call is still correct, because of an asymmetry in
`gLDriver.DoFromGoroutine` (fyne v2.8.0, `internal/driver/glfw/driver.go`):

```go
func (d *gLDriver) DoFromGoroutine(f func(), wait bool) {
	if wait {
		async.EnsureNotMain(func() {
			runOnMainWithWait(f, true)
		})
	} else {
		runOnMainWithWait(f, false)
	}
}
```

`async.EnsureNotMain` — the panic-if-called-from-main-goroutine guard — only
wraps the `wait == true` branch. `runOnMainWithWait(f, false)`
(`internal/driver/glfw/loop.go`) just enqueues `f` onto `funcQueue` with no
thread check at all. `fyne.Do` calls `DoFromGoroutine(f, false)`;
`fyne.DoAndWait` calls it with `true`. So `fyne.Do` — never `DoAndWait` — is
safe to call from the main thread as well as a background goroutine, and
`installOpenWithHandler`'s doc comment calls this out explicitly as the
reason it must stay `Do` and never be "corrected" to `DoAndWait`, which
would panic exactly where this handler is used.

## The known coverage limit

Fyne's *test* driver (`fyne.io/fyne/v2@v2.8.0/test/driver.go`) implements
`DoFromGoroutine` as:

```go
func (d *driver) DoFromGoroutine(f func(), _ bool) {
	// Tests all run on a single (but potentially different per-test) thread
	f()
}
```

It ignores the `wait` flag entirely and just calls `f()` inline, always. That
collapses the exact distinction the production path depends on: under the
test driver, a "queued `Do`" and a direct synchronous call are
indistinguishable, so no test can tell them apart. Concretely, this means no
test can *fail* if `installOpenWithHandler`'s callback were changed to call
`v.openFilesFromOS` directly instead of through `fyne.Do` — the test suite
would still pass, because the test driver already runs everything inline.

`internal/ui/openwith_test.go`'s
`TestOpenInitialFiles_ArgvAndADeliveryBecomeOneScanWithArgvFirst` verifies
the single-scan, argv-first *outcome* (one scan started, argv file ahead of
the delivered one in `unsortedFiles`) by driving the real
`openwith.Deliver` → queue → handler path the way `Run`'s `SetOnStarted`
wires it. That is a genuine regression guard for the ordering contract
itself. What it cannot do is prove that the `fyne.Do` wrapping is what makes
the ordering hold under the real GLFW driver, where the handler executes on
AppKit's callback thread rather than call-site-inline — that guarantee is
established by the reasoning above (both callers marshal through `fyne.Do`,
so they land on one goroutine in call order) and by the manual verification
checklist in the plan (§4), not by anything the automated suite can
distinguish.

## Verification

Run from the repository root after the documentation-only stage (todos.md,
AGENTS.md, README.md edits; no `.go` file touched):

```
$ make fmt-check           # exit 0
$ go vet ./...             # exit 0
$ go build ./...           # exit 0 (harmless "ld: warning: ignoring duplicate libraries: '-lobjc'")
$ go test -timeout 20m -race ./...
ok  	github.com/frathe/picfetch	1.709s
ok  	github.com/frathe/picfetch/internal/... (34 more packages)   # all ok, none FAIL
```

All 35 test binaries pass; every non-root package came back from the build
cache since nothing at the Go level changed in this stage — the real
`-race` run (the one that actually recompiles and re-executes) is the one
the earlier code stages already gated on before landing.

## Self-review

- The four points above are exactly the ones that don't show up in a plain
  `git diff`: they're either negative results ("this alternative doesn't
  work because...") or properties of fyne's driver internals that have to
  be read out of its source, not this repo's.
- Nothing here duplicates `ARCHITECTURE.md`, which already documents
  `internal/openwith`, the `openwith.go` file-table row, `SupportedExtensions`,
  `openwith.Install` in the package-`main` paragraph, and two "Where to look
  for X" entries — checked against the tree, all present, none touched.
- `plans/2026-08-27-macos-open-with.md` is the authoritative stage-by-stage
  record and manual-QA checklist; this file does not restate it.

## Suggested commit message

```
docs: land the macOS Open With retrospective and close out the todo

Moves the "Open With / Dock-drop doesn't work" item from todos.md's TODO
into Done, extends AGENTS.md's input-flow line and package-mac note,
adds the Open With / Dock-icon / double-click bullet to README's
Features, and records the non-obvious design decisions (the
setDelegate: selector cache, why the queue exists, the fyne.Do safety
inversion, and the test-coverage limit it leaves) in
finished_refactorings/2026-08-28-macos-open-with.md.
```
