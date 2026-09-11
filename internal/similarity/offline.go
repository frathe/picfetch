package similarity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"syscall"
	"time"

	"golang.org/x/net/bpf"
)

// Build the Linux seccomp program independently of kernel installation, so its
// decisions can be checked on every development host. Values are supplied from
// the target architecture's unix constants by the Linux worker.
func linuxNetworkFilter(architecture uint32, socketCalls ...uint32) ([]bpf.RawInstruction, error) {
	const deny = 0x00050001  // Linux SECCOMP_RET_ERRNO | EPERM
	const allow = 0x7fff0000 // Linux SECCOMP_RET_ALLOW
	instructions := []bpf.Instruction{
		bpf.LoadAbsolute{Off: 4, Size: 4},
		bpf.JumpIf{Cond: bpf.JumpEqual, Val: architecture, SkipTrue: 1},
		bpf.RetConstant{Val: deny},
		bpf.LoadAbsolute{Off: 0, Size: 4},
		// Reject alternate syscall numbering, including the x32 ABI.
		bpf.JumpIf{Cond: bpf.JumpGreaterOrEqual, Val: 0x40000000, SkipFalse: 1},
		bpf.RetConstant{Val: deny},
	}
	for _, call := range socketCalls {
		instructions = append(instructions,
			bpf.JumpIf{Cond: bpf.JumpEqual, Val: call, SkipFalse: 1},
			bpf.RetConstant{Val: deny})
	}
	instructions = append(instructions, bpf.RetConstant{Val: allow})
	return bpf.Assemble(instructions)
}

// EnforcesNetworkIsolation describes the worker policy. Windows uses a normal
// local subprocess; its events must never claim verified OS network denial.
func EnforcesNetworkIsolation() bool {
	return runtime.GOOS != "windows"
}

// VerifyOffline checks OS-enforced network denial without sending an image,
// path, or payload. A timeout, refused port, offline network, or environment flag
// cannot substitute for OS denial.
func VerifyOffline(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !EnforcesNetworkIsolation() {
		// Windows is a proper name.
		//goland:noinspection GoErrorStringFormat
		return fmt.Errorf("Windows analysis does not enforce OS network isolation")
	}
	for _, network := range []string{"tcp4", "udp4"} {
		dialer := net.Dialer{Timeout: time.Second}
		connection, err := dialer.DialContext(ctx, network, "192.0.2.1:443")
		if err == nil {
			if network == "udp4" {
				_, err = connection.Write(nil)
			}
			_ = connection.Close()
		}
		if !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.EACCES) {
			return fmt.Errorf("offline denial not verified for %s: %v", network, err)
		}
	}
	return ctx.Err()
}
