//go:build !windows

package filemanager

import "os/exec"

// applyExplorerCommandLine's real implementation (windows.go) sets a
// Windows-only SysProcAttr field, so it needs this no-op twin for every
// other platform: revealWindows is compiled - and unit-tested - everywhere,
// even though Reveal only ever dispatches to it on Windows. explorerCmdLine
// itself stays portable, so the command line this would set is still
// asserted on from here.
func applyExplorerCommandLine(_ *exec.Cmd, _ string) {}
