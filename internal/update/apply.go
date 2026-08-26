package update

import (
	"fmt"
	"runtime"
)

// Apply replaces the running executable at dest with stage.BinaryPath.
// dest is os.Executable() from UI glue. A var so tests stub it.
var Apply = func(stage Stage, dest string) error {
	switch runtime.GOOS {
	case "windows":
		return applyWindows(stage, dest)
	default:
		return applyUnix(stage, dest)
	}
}

// windowsApplyScript returns a .cmd that waits for pid to exit, copies
// staged onto dest, deletes the staged file, then self-deletes. It does
// not relaunch the app — the user starts PicFetch again themselves.
func windowsApplyScript(dest, staged string, pid int) string {
	return fmt.Sprintf(`@echo off
:wait
tasklist /FI "PID eq %d" 2>NUL | find "%d" >NUL
if not errorlevel 1 (
	timeout /t 1 /nobreak >NUL
	goto wait
)
copy /Y "%s" "%s"
del "%s"
del "%%~f0"
`, pid, pid, staged, dest, staged)
}
