# Native position-poller smoke

Observed on 2026-09-06, macOS 26.6.2 (25G83), arm64, Go 1.27.1,
Fyne v2.8.0. Source: the current maintainability working tree, including
the Poller Stop/Done/Wait implementation and all main/secondary bindings.

`make run` started successfully, but the GUI tool could not target its raw
`go run` executable among the installed apps sharing the bundle identifier.
That attempt is not counted as interaction evidence. To identify the source
build unambiguously, `go build -o '/private/tmp/PicFetch Maintainability.app/Contents/MacOS/PicFetch' .`
created a temporary app with a minimal Info.plist and the unchanged production
bundle identifier. Its distinct display name and full app path were used for
every GUI action. No packaging build number or installation was changed.

Using the computer-use tool on that exact app:

1. Opened the main window and dragged its title bar. The app remained responsive.
2. Opened Settings through the native app menu, dragged its title bar, and
   closed it with its native close button. The main window remained available.
3. Reopened Settings, leaving both main and secondary position pollers active.
   The [captured Settings window](20-native-settings.png) records this state.
4. Sent Cmd+Q. Process 73955, whose command was the exact temporary executable,
   disappeared: `ps -p 73955 -o pid=,stat=,command=` returned no rows, status 1.
   A subsequent app inventory confirmed it was no longer running.

After exit, the persisted main geometry had `windowPosSet=true`, x=0, y=63;
the Settings geometry had `settingsWinPosSet=true`, x=391, y=66. These values
record the saved geometry alongside the interaction evidence. The procedure changed no controls
in Settings. GUI quit and observed process disappearance are the shutdown
evidence; this is not a native race test or the comparison GL smoke in ticket 26.

Deterministic gated worker tests separately cover queued cancellation without
UI drain, cancellation during dispatch/native read, suppressed publication,
actual Wait completion, and retained closed-window workers. Their commands and
negative-guard results are recorded in the active implementation plan.
