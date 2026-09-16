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

	"github.com/frathe/picfetch/internal/imaging"
)

const workerEnvironment = "PICFETCH_SIMILARITY_WORKER"

const (
	workerRequestLimit       = 64 * 1024 * 1024
	DefaultAnalysisMemoryMB  = 512
	DefaultAnalysisItemLimit = 10000
	MaxAnalysisMemoryMB      = 1024 * 1024
)

// AnalysisLimits bounds collection work and each serialized result. MemoryMB
// does not bound total process memory or the native inference runtime.
type AnalysisLimits struct {
	MemoryMB int
	Items    int
}

func (l AnalysisLimits) Normalized() AnalysisLimits {
	if l.MemoryMB <= 0 || l.MemoryMB > MaxAnalysisMemoryMB {
		l.MemoryMB = DefaultAnalysisMemoryMB
	}
	if l.Items <= 0 {
		l.Items = DefaultAnalysisItemLimit
	}
	return l
}

var (
	ErrAnalysisItemLimit   = errors.New("similarity explorer item limit exceeded")
	ErrAnalysisMemoryLimit = errors.New("similarity explorer memory limit exceeded")
)

func (l AnalysisLimits) validateSourceCount(count int) error {
	limit := l.Normalized().Items
	// Grouping is quadratic in represented sources; keep its worst case finite.
	if count > limit {
		return fmt.Errorf("%w: accepts at most %d images, got %d", ErrAnalysisItemLimit, limit, count)
	}
	return nil
}

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
	Assets         string
	AnalysisLimits AnalysisLimits
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
	Search               *SearchRequest `json:",omitempty"`
	Assets               string
	FavoritesDir         string
	GeneralAnalysisDir   string
	DisableFavoriteCache bool
	Paths                []string
	MaxEncodedBytes      int64
	AnalysisLimits       AnalysisLimits
}

// Analyze streams serialized immutable snapshots and waits for worker exit.
// Cancellation kills even a batch algorithm that does not accept a context.
func (c Client) Analyze(ctx context.Context, paths []string, controls <-chan Control, emit func(Event)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	limits := c.AnalysisLimits.Normalized()
	if err := limits.validateSourceCount(len(paths)); err != nil {
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
	req := request{Assets: assets, FavoritesDir: c.FavoritesDir, GeneralAnalysisDir: c.GeneralAnalysisDir, DisableFavoriteCache: c.DisableFavoriteCache, Paths: paths, MaxEncodedBytes: imaging.MaxEncodedBytes(), AnalysisLimits: limits}
	cmd := workerCommand(ctx, executable)
	cmd.Env = append(os.Environ(), workerEnvironment+"=1")
	return analyzeCommand(ctx, cmd, req, controls, emit)
}

func analyzeCommand(ctx context.Context, cmd *exec.Cmd, req request, controls <-chan Control, emit func(Event)) error {
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
	decoder := newWorkerEventDecoder(stdout, req.AnalysisLimits.Normalized().MemoryMB*1024*1024, req.AnalysisLimits.Normalized().Items)
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
	// Killing the worker after a decode failure must not hide the resource error.
	if err != nil && !errors.Is(err, io.EOF) {
		return err
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
	var req request
	input, err := controlInput()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = input.Close() }()
	decoder := newWorkerDecoder(input)
	err = decoder.Decode(&req)
	if err == nil && req.Search == nil {
		err = req.AnalysisLimits.validateSourceCount(len(req.Paths))
	}
	if err == nil {
		imaging.SetMaxEncodedBytes(req.MaxEncodedBytes)
		if req.Search != nil {
			if err := searchWorker(ctx, req, decoder, input); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return true
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
		output := newWorkerEventEncoder(os.Stdout, req.AnalysisLimits.Normalized().MemoryMB*1024*1024)
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
