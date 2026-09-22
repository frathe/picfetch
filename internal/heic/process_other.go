//go:build !linux && !darwin && !windows

package heic

import "os/exec"

func prepareWorker(_ *exec.Cmd)        {}
func retireWorker(cmd *exec.Cmd) error { return cmd.Process.Kill() }
