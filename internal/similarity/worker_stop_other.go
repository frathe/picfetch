//go:build !(darwin && appleappstore)

package similarity

import "os/exec"

func stopWorkerProcess(cmd *exec.Cmd) error { return cmd.Process.Kill() }
