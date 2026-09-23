package similarity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/imaging"
)

type heicCaptureReport struct {
	Available, HasBackend      bool
	Generation                 uint64
	MaxEncodedBytes, MaxPixels int64
}

// This stub replaces only inference at the real producer protocol boundary.
// Native decoder and isolation qualification remain separate opt-in cases.
func heicCaptureWorker() error {
	decoder := newWorkerDecoder(os.Stdin)
	var req request
	if err := decoder.Decode(&req); err != nil {
		return err
	}
	imaging.SetMaxEncodedBytes(req.MaxEncodedBytes)
	ctx, release := workerHEICContext(context.Background(), req)
	defer release()
	snapshot := heic.FromContext(ctx)
	if code := os.Getenv("PICFETCH_TEST_HEIC_LOSS"); code != "" {
		backend := &boundedHEIC{Backend: heicFailureBackend{code: code}}
		ctx = heic.WithSnapshot(ctx, heic.Snapshot{Available: true, Backend: backend, Generation: snapshot.Generation})
		_, _ = backend.Read(ctx, nil, heic.Request{})
	}
	report := heicCaptureReport{Available: snapshot.Available, HasBackend: snapshot.Backend != nil, Generation: snapshot.Generation, MaxEncodedBytes: req.MaxEncodedBytes, MaxPixels: req.MaxPixels}
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if marker := os.Getenv("PICFETCH_TEST_HEIC_RETIREMENT"); marker != "" {
		stopping, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
		defer stop()
		if req.Search == nil {
			err = newWorkerEventEncoder(os.Stdout, workerRequestLimit).Encode(Event{Stage: "encoding"})
		} else {
			err = json.NewEncoder(os.Stdout).Encode(SearchEvent{SessionID: req.Search.SessionID, Revision: 1, Kind: SearchReady})
		}
		if err != nil {
			return err
		}
		<-stopping.Done()
		return os.WriteFile(marker, []byte("joined"), 0600)
	}
	if req.Search == nil {
		return newWorkerEventEncoder(os.Stdout, workerRequestLimit).Encode(Event{Complete: true, CacheWarning: string(data), HEICUnavailable: workerHEICUnavailable(ctx)})
	}
	encoder := json.NewEncoder(os.Stdout)
	event := SearchEvent{SessionID: req.Search.SessionID, Revision: 1, Kind: SearchReady, CacheWarning: string(data), HEICUnavailable: workerHEICUnavailable(ctx)}
	if err := encoder.Encode(event); err != nil {
		return err
	}
	for {
		var query SearchQuery
		if err := decoder.Decode(&query); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		event.Revision++
		event.Kind, event.QueryID = SearchFinal, query.ID
		if err := encoder.Encode(event); err != nil {
			return err
		}
	}
}

type heicFailureBackend struct{ code string }

func (heicFailureBackend) Check(_ context.Context) error { return nil }
func (b heicFailureBackend) Read(_ context.Context, _ []byte, _ heic.Request) (heic.Result, error) {
	return heic.Result{}, map[string]error{"unavailable": heic.ErrUnavailable, "unsupported": heic.ErrUnsupported, "invalid": heic.ErrInvalid, "protocol": heic.ErrProtocol, "crash": io.ErrUnexpectedEOF, "timeout": context.DeadlineExceeded}[b.code]
}

func TestHEICAnalysisWorkerReportsBackendLoss(t *testing.T) {
	t.Setenv("PICFETCH_TEST_HEIC_CAPTURE", "1")
	for _, kind := range []string{"finite", "retained"} {
		for _, code := range []string{"unavailable", "unsupported", "invalid", "protocol", "crash", "timeout"} {
			t.Run(kind+"/"+code, func(t *testing.T) {
				t.Setenv("PICFETCH_TEST_HEIC_LOSS", code)
				capability := heic.NewCapability(heicFailureBackend{}, heic.Identity{}, heic.Observation{})
				t.Cleanup(func() { capability.Stop(); capability.Wait() })
				<-capability.Check(context.Background())
				ctx, cancel := context.WithTimeout(capability.CaptureContext(context.Background()), 10*time.Second)
				defer cancel()
				run := func() error {
					if kind == "finite" {
						return (Client{}).Analyze(ctx, nil, nil, func(_ Event) {})
					}
					queries := make(chan SearchQuery)
					close(queries)
					return (Client{}).Search(ctx, SearchRequest{SessionID: 1}, queries, func(_ SearchEvent) {})
				}
				if err := run(); err != nil {
					t.Fatal(err)
				}
				state := capability.State()
				if code == "unavailable" {
					if state.Available || state.Generation != 2 {
						t.Fatalf("producer lost decisive backend-loss notice: %+v", state)
					}
					<-capability.Check(context.Background())
					if err := run(); err != nil {
						t.Fatal(err)
					}
					if current := capability.State(); !current.Available || current.Generation != 3 {
						t.Fatalf("stale producer notice invalidated a later support observation: %+v", current)
					}
				} else if !state.Available || state.Generation != 1 {
					t.Fatalf("per-file failure invalidated global support: %+v", state)
				}
			})
		}
	}
}

func TestHEICAnalysisWorkerCapturedPolicy(t *testing.T) {
	t.Setenv("PICFETCH_TEST_HEIC_CAPTURE", "1")
	before := imaging.MaxEncodedBytes()
	imaging.SetMaxEncodedBytes(1234567)
	t.Cleanup(func() { imaging.SetMaxEncodedBytes(before) })
	backend := heic.NewClient("")
	t.Cleanup(func() { backend.Stop(); backend.Wait() })
	for _, snapshot := range []heic.Snapshot{{}, {Available: true, Generation: 2}, {Backend: backend, Available: false, Generation: 3}, {Backend: backend, Available: true, Generation: 4}} {
		ctx, cancel := context.WithTimeout(heic.WithSnapshot(context.Background(), snapshot), 10*time.Second)
		var actual heicCaptureReport
		err := (Client{}).Analyze(ctx, nil, nil, func(event Event) {
			if err := json.Unmarshal([]byte(event.CacheWarning), &actual); err != nil {
				t.Error(err)
			}
		})
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		available := snapshot.Available && snapshot.Backend != nil
		want := heicCaptureReport{Available: available, HasBackend: available, Generation: snapshot.Generation, MaxEncodedBytes: 1234567, MaxPixels: imaging.MaxImagePixels()}
		if actual != want {
			t.Fatalf("finite producer lost captured policy: got %+v want %+v", actual, want)
		}
	}
	ctx, cancel := context.WithTimeout(heic.WithSnapshot(context.Background(), heic.Snapshot{Backend: backend, Available: true, Generation: 9}), 10*time.Second)
	defer cancel()
	queries := make(chan SearchQuery, 1)
	var finals int
	err := (Client{}).Search(ctx, SearchRequest{SessionID: 77}, queries, func(event SearchEvent) {
		var actual heicCaptureReport
		if err := json.Unmarshal([]byte(event.CacheWarning), &actual); err != nil {
			t.Error(err)
			return
		}
		want := heicCaptureReport{Available: true, HasBackend: true, Generation: 9, MaxEncodedBytes: 1234567, MaxPixels: imaging.MaxImagePixels()}
		if actual != want {
			t.Errorf("retained producer lost its captured policy: got %+v want %+v", actual, want)
		}
		if event.Kind == SearchReady {
			// A later application limit must not change the retained request.
			imaging.SetMaxEncodedBytes(7654321)
			queries <- SearchQuery{ID: 1}
		} else if event.QueryID == 1 {
			finals++
			queries <- SearchQuery{ID: 2}
		} else {
			finals++
			close(queries)
		}
	})
	if err != nil || finals != 2 {
		t.Fatalf("retained request did not settle both references: finals=%d error=%v", finals, err)
	}
}
