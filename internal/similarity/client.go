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
	"syscall"
	"time"

	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/imaging"
)

const workerEnvironment = "PICFETCH_SIMILARITY_WORKER"
const workerRequestLimit = 64 * 1024 * 1024

// Bound each decoded message independently; a retained search can accept an
// arbitrary number of small reference queries without exhausting a lifetime cap.
type workerDecoder struct {
	reader  *io.LimitedReader
	decoder *json.Decoder
}

func newWorkerDecoder(input io.Reader) *workerDecoder {
	reader := &io.LimitedReader{R: input, N: workerRequestLimit}
	return &workerDecoder{reader: reader, decoder: json.NewDecoder(reader)}
}
func (d *workerDecoder) Decode(value any) error {
	// Count prefetched bytes toward the next message's budget.
	buffered, err := io.Copy(io.Discard, d.decoder.Buffered())
	if err != nil {
		return err
	}
	d.reader.N = workerRequestLimit - buffered
	return d.decoder.Decode(value)
}

// Client runs native inference and batch algorithms outside the viewer process.
// Assets may override the installed assets directory for a local trial.
type Client struct {
	Assets string
	// HEIC is the application's shared admission owner. Nil preserves the
	// existing unavailable-format behavior; workers never launch their own.
	HEIC *heicclient.Client
	// HTTPClient configures asset downloads; analysis never uses it.
	HTTPClient *http.Client
	// FavoritesDir supplies membership and, unless disabled, Favorite analysis.
	FavoritesDir string
	// DisableFavoriteCache preserves membership for loose-cache routing while
	// disabling Favorite analysis reuse and persistence.
	DisableFavoriteCache bool
	// GeneralAnalysisDir enables reuse of compatible loose-image representations.
	// Analyzer hits are promoted into enabled Favorites; misses are not written here.
	GeneralAnalysisDir string
}

type request struct {
	HEIC                 *heicclient.PipeConfig `json:",omitempty"`
	Search               *SearchRequest         `json:",omitempty"`
	Assets               string
	FavoritesDir         string
	GeneralAnalysisDir   string
	DisableFavoriteCache bool
	Paths                []string
	MaxEncodedBytes      int64
	reader               imaging.Reader
}

// Analyze streams serialized immutable snapshots and waits for worker exit.
// Cancellation kills even a batch algorithm that does not accept a context.
func (c Client) Analyze(ctx context.Context, paths []string, controls <-chan Control, emit func(Event)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !SupportedPlatform() {
		return fmt.Errorf("local similarity analysis requires macOS/Linux/Windows amd64/arm64")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	assets := c.Assets
	if assets == "" {
		assets = defaultAssets(executable)
	}
	req := request{Assets: assets, FavoritesDir: c.FavoritesDir, GeneralAnalysisDir: c.GeneralAnalysisDir, DisableFavoriteCache: c.DisableFavoriteCache, Paths: paths, MaxEncodedBytes: imaging.MaxEncodedBytes()}
	cmd := workerCommand(ctx, executable)
	cmd.Env = append(os.Environ(), workerEnvironment+"=1")
	return c.analyzeCommand(ctx, cmd, req, controls, emit)
}

func (c Client) analyzeCommand(ctx context.Context, cmd *exec.Cmd, req request, controls <-chan Control, emit func(Event)) error {
	link, err := c.attachHEIC(ctx, cmd, &req)
	if err != nil {
		return err
	}
	defer closeHEICAttachment(link)
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
	if link != nil {
		link.Started()
	}
	stopWriter := make(chan struct{})
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		defer func() { _ = input.Close() }()
		encoder := json.NewEncoder(input)
		if err := encoder.Encode(req); err != nil {
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
	if err := isolateWorker(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	input, err := controlInput()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = input.Close() }()
	if err = runWorker(ctx, input); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return true
}

func runWorker(ctx context.Context, input io.ReadCloser) error {
	var req request
	decoder := newWorkerDecoder(input)
	if err := decoder.Decode(&req); err != nil {
		return err
	}
	cleanup, err := openWorkerSource(&req)
	if err != nil {
		return err
	}
	defer cleanup()
	imaging.SetMaxEncodedBytes(req.MaxEncodedBytes)
	if req.Search != nil {
		return searchWorker(ctx, req, decoder, input)
	}
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
	return err
}
