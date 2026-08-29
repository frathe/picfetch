package update

import (
	"fmt"
	"runtime"
	"strings"
)

// ApplyOptions controls behavior after a staged update has been installed.
// The zero value preserves the normal-shutdown policy: install without
// relaunching PicFetch.
type ApplyOptions struct {
	Relaunch bool
}

// Apply replaces the running executable at dest with stage.BinaryPath.
// dest is os.Executable() from UI glue. A var so tests stub it.
var Apply = func(stage Stage, dest string, options ApplyOptions) error {
	switch runtime.GOOS {
	case "windows":
		return applyWindows(stage, dest, options)
	default:
		return applyUnix(stage, dest, options)
	}
}

// windowsApplyScript returns a .cmd that waits for pid to exit, copies
// staged onto dest, deletes the staged file, optionally starts the installed
// executable, then self-deletes. The launch is only reachable after a
// successful copy.
func windowsApplyScript(dest, staged string, pid int, options ApplyOptions) string {
	newDest := windowsCommandPath(dest + ".new")
	oldDest := windowsCommandPath(dest + ".old")
	dest = windowsCommandPath(dest)
	staged = windowsCommandPath(staged)
	relaunch := ""
	if options.Relaunch {
		relaunch = fmt.Sprintf("start \"\" %s\nif errorlevel 1 >>\"%%TEMP%%\\picfetch-update.log\" echo PicFetch update relaunch failed.\n", dest)
	}
	return fmt.Sprintf(`@echo off
setlocal DisableDelayedExpansion
:wait
tasklist /FI "PID eq %d" 2>NUL | find "%d" >NUL
if not errorlevel 1 (
	timeout /t 1 /nobreak >NUL
	goto wait
)
copy /Y %s %s >NUL
if errorlevel 1 (
	>>"%%TEMP%%\picfetch-update.log" echo PicFetch update copy failed.
	goto cleanup
)
move /Y %s %s >NUL
if errorlevel 1 (
	>>"%%TEMP%%\picfetch-update.log" echo PicFetch update backup failed.
	goto cleanup
)
move /Y %s %s >NUL
if errorlevel 1 (
	move /Y %s %s >NUL
	if errorlevel 1 >>"%%TEMP%%\picfetch-update.log" echo PicFetch update rollback failed.
	>>"%%TEMP%%\picfetch-update.log" echo PicFetch update install failed.
	goto cleanup
)
del %s
del %s
%s:cleanup
del %s 2>NUL
del "%%~f0"
`, pid, pid, staged, newDest, dest, oldDest, newDest, dest, oldDest, dest, oldDest, staged, relaunch, newDest)
}

// windowsCommandPath quotes a path for a batch file. Quoting protects cmd
// metacharacters such as spaces and ampersands; doubling percent signs stops
// environment-variable expansion. Delayed expansion is disabled by the
// generated script so exclamation marks remain literal as well.
func windowsCommandPath(path string) string {
	return `"` + strings.ReplaceAll(path, "%", "%%") + `"`
}
