package similarity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
type Client struct {
	Assets string
	// HTTPClient configures asset downloads; analysis never uses it.
	HTTPClient *http.Client
	// FavoritesDir enables per-favorite representation reuse. Empty disables disk caching.
	FavoritesDir string
}

type request struct {
	Assets       string
	FavoritesDir string
	Paths        []string
}

// Analyze streams serialized immutable snapshots and waits for worker exit.
// Cancellation kills even a batch algorithm that does not accept a context.
func (c Client) Analyze(ctx context.Context, paths []string, controls <-chan Control, emit func(Event)) error {
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
	cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1) (allow default) (deny network*)", executable)
	cmd.Env = append(os.Environ(), workerEnvironment+"=1")
	input, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
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
	stopWriter := make(chan struct{})
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		defer func() { _ = input.Close() }()
		encoder := json.NewEncoder(input)
		if err := encoder.Encode(request{Assets: assets, FavoritesDir: c.FavoritesDir, Paths: paths}); err != nil {
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopWriter:
				return
			case control, ok := <-controls:
				if !ok {
					return
				}
				if err := encoder.Encode(control); err != nil {
					return
				}
			}
		}
	}()
	defer func() {
		close(stopWriter)
		_ = input.Close()
		<-writerDone
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
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
	userAssets, userErr := userAssetDirectory()
	if userErr == nil {
		if info, err := os.Stat(userAssets); err == nil && info.IsDir() {
			return userAssets
		}
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
	trialAssets := filepath.Join(root, ".scratch", "visual-similarity-explorer", "assets")
	if info, err := os.Stat(trialAssets); err == nil && info.IsDir() {
		return trialAssets
	}
	if userErr == nil {
		return userAssets
	}
	return installed
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
	input, err := controlInput()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = input.Close() }()
	decoder := json.NewDecoder(io.LimitReader(input, 64<<20))
	err = decoder.Decode(&req)
	if err == nil {
		readCtx, cancelRead := context.WithCancel(ctx)
		controls := make(chan Control, 1)
		readDone := make(chan struct{})
		go func() {
			defer close(readDone)
			defer close(controls)
			for {
				var control Control
				if err := decoder.Decode(&control); err != nil {
					return
				}
				select {
				case controls <- control:
				case <-readCtx.Done():
					return
				}
			}
		}()
		output := json.NewEncoder(os.Stdout)
		err = analyzeLocal(ctx, req, controls, func(event Event) error { return output.Encode(event) })
		cancelRead()
		_ = input.Close()
		<-readDone
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return true
}
