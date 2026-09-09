package similarity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

const workerEnvironment = "PICFETCH_SIMILARITY_WORKER"

// Client runs native inference and batch algorithms outside the viewer process.
// Assets may override the installed assets directory for a local trial.
type Client struct{ Assets string }

type request struct {
	Assets string
	Paths  []string
}

// Analyze streams serialized immutable snapshots and waits for worker exit.
// Cancellation kills even a batch algorithm that does not accept a context.
func (c Client) Analyze(ctx context.Context, paths []string, emit func(Event)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return fmt.Errorf("local similarity currently requires Apple Silicon macOS")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	assets := c.Assets
	if assets == "" {
		assets = defaultAssets(executable)
	}
	payload, err := json.Marshal(request{Assets: assets, Paths: paths})
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1) (allow default) (deny network*)", executable)
	cmd.Env = append(os.Environ(), workerEnvironment+"=1")
	cmd.Stdin = bytes.NewReader(payload)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.WaitDelay = 3 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	decoder := json.NewDecoder(stdout)
	complete := false
	for {
		var event Event
		err = decoder.Decode(&event)
		if err != nil {
			break
		}
		if ctx.Err() != nil {
			break
		}
		if complete {
			err = fmt.Errorf("worker published data after completion")
			break
		}
		complete = event.Complete
		emit(event)
	}
	if err != nil && !errors.Is(err, io.EOF) {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if waitErr != nil {
		return fmt.Errorf("analysis worker: %w: %s", waitErr, stderr.String())
	}
	if !errors.Is(err, io.EOF) {
		return err
	}
	if !complete {
		return fmt.Errorf("analysis worker exited without a completed map")
	}
	return nil
}

func defaultAssets(executable string) string {
	if path := os.Getenv("PICFETCH_SIMILARITY_ASSETS"); path != "" {
		return path
	}
	installed := filepath.Join(filepath.Dir(executable), "similarity-assets")
	if info, err := os.Stat(installed); err == nil && info.IsDir() {
		return installed
	}
	// Local developer trial; packaged installs use the directory above.
	root, err := os.Getwd()
	if err != nil {
		return installed
	}
	return filepath.Join(root, ".scratch", "visual-similarity-explorer", "assets")
}

// WorkerMain handles the private subprocess mode before any desktop startup.
// Normal application and test launches return false without side effects.
func WorkerMain() bool {
	if os.Getenv(workerEnvironment) != "1" {
		return false
	}
	_ = os.Unsetenv(workerEnvironment)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var req request
	err := json.NewDecoder(io.LimitReader(os.Stdin, 64<<20)).Decode(&req)
	if err == nil {
		output := json.NewEncoder(os.Stdout)
		err = analyzeLocal(ctx, req, func(event Event) error { return output.Encode(event) })
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return true
}
