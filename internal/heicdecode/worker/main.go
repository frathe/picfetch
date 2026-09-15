package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// Main serves one request in the dedicated, disposable helper process. Probe
// paths and loopback endpoints are parent-owned launch controls, never supplied
// by an image. Any missing platform restriction exits before reading stdin.
func Main(args []string) int {
	if len(args) != 6 || args[0] != "--heic-worker-v2" || len(args[1]) > 2048 {
		return 2
	}
	var limits heicdecode.Limits
	if err := json.Unmarshal([]byte(args[1]), &limits); err != nil || limits.Validate() != nil {
		return 2
	}
	for _, arg := range args[2:] {
		if len(arg) == 0 || len(arg) > 4096 {
			return 2
		}
	}
	runtime.GOMAXPROCS(1)
	debug.SetMaxThreads(32)
	// A Go runtime target, explicitly not a hard cap for native process memory.
	debug.SetMemoryLimit(limits.OSProcessBytes - limits.OSProcessBytes/4)
	nativeMemory, err := isolate(limits)
	if err != nil {
		return 3
	}
	ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
	defer cancel()
	if err = verifyDenial(ctx, args[2], args[3], args[4], args[5]); err != nil {
		return 3
	}
	if err = heicdecode.WriteReady(os.Stdout, heicdecode.Ready{WASMMemoryBytes: limits.WASMMemoryBytes, NativeMemoryBytes: nativeMemory}); err != nil {
		return 4
	}
	if err = execute(ctx, decoder, os.Stdin, os.Stdout, os.Stderr, limits); err != nil {
		return 4
	}
	return 0
}

func verifyDenial(ctx context.Context, readPath, writePath, tcpAddress, udpAddress string) error {
	file, err := os.Open(readPath)
	if file != nil {
		_ = file.Close()
	}
	if !permissionDenied(err) {
		return errors.New("HEIC native file-read denial unavailable")
	}
	file, err = os.OpenFile(writePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if file != nil {
		_ = file.Close()
	}
	if !permissionDenied(err) {
		return errors.New("HEIC native file-write denial unavailable")
	}
	for i, address := range []string{tcpAddress, udpAddress} {
		host, _, splitErr := net.SplitHostPort(address)
		if splitErr != nil || host != "127.0.0.1" {
			return errors.New("HEIC capability probe must be loopback")
		}
		network := "tcp4"
		if i == 1 {
			network = "udp4"
		}
		dialer := net.Dialer{Timeout: time.Second}
		connection, dialErr := dialer.DialContext(ctx, network, address)
		if connection != nil {
			if i == 1 {
				_ = connection.SetWriteDeadline(time.Now().Add(time.Second))
				_, dialErr = connection.Write([]byte{0})
			}
			_ = connection.Close()
		}
		if !permissionDenied(dialErr) {
			return errors.New("HEIC native network denial unavailable")
		}
	}
	return ctx.Err()
}

func permissionDenied(err error) bool {
	return errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES)
}
