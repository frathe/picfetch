//go:build windows && (amd64 || arm64)

package client

import (
	"os"
	"os/exec"

	"github.com/frathe/picfetch/internal/heicdecode"
	"github.com/frathe/picfetch/internal/heicdecode/winisolation"
)

func startProcess(cmd *exec.Cmd, limits heicdecode.Limits) (helperProcess, error) {
	input, inOK := cmd.Stdin.(*os.File)
	output, outOK := cmd.Stdout.(*os.File)
	diagnostic, errOK := cmd.Stderr.(*os.File)
	if !inOK || !outOK || !errOK {
		return nil, ErrUnavailable
	}
	return winisolation.Start(cmd.Path, cmd.Args[1:], [3]*os.File{input, output, diagnostic}, limits)
}
