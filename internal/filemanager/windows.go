//go:build windows

package filemanager

import (
	"os/exec"
	"syscall"
)

// applyExplorerCommandLine replaces the command line os/exec would build
// with the exact one explorerCmdLine spells out. SysProcAttr.CmdLine is
// passed to CreateProcess verbatim, program name included, which is the only
// way to keep the quotes around the path alone - see explorerCmdLine for why
// that matters.
func applyExplorerCommandLine(cmd *exec.Cmd, path string) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: explorerCmdLine(path)}
}
