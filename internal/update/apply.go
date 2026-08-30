package update

import "runtime"

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
