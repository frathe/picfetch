package similarity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

const searchProtocolHelperMode = "PICFETCH_TEST_SEARCH_PROTOCOL"

func TestSearchProtocolInitialFlowAndControlClosure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req := searchProtocolRequest(t)
	queries := searchProtocolQueries(t, ctx, req)
	cmd := searchProtocolCommand(t, ctx, "clean")
	var events []SearchEvent
	err := searchCommand(ctx, cmd, req, queries, func(event SearchEvent) {
		events = append(events, event)
		if event.Kind == SearchReady {
			close(queries)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []SearchEvent{
		{SessionID: 77, QueryID: 1, Revision: 1, Kind: SearchProgress, Processed: 1, Total: 2},
		{SessionID: 77, QueryID: 1, Revision: 2, Kind: SearchFinal, Processed: 2, Total: 2, Matches: []Match{{Path: req.Search.Paths[1], Score: .75}}},
		{SessionID: 77, QueryID: 1, Revision: 3, Kind: SearchReady, Processed: 2, Total: 2},
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("transport events: %+v; want %+v", events, want)
	}
	// The helper exits successfully only after observing EOF on its controls.
	if cmd.ProcessState == nil || !cmd.ProcessState.Success() {
		t.Fatalf("provider returned without observing successful worker exit: %v", cmd.ProcessState)
	}
}

func TestSearchProtocolRejectsInvalidStreams(t *testing.T) {
	for _, mode := range []string{"truncated", "malformed", "revision", "session", "unknown_kind"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			req := searchProtocolRequest(t)
			queries := searchProtocolQueries(t, ctx, req)
			cmd := searchProtocolCommand(t, ctx, mode)
			var events []SearchEvent
			err := searchCommand(ctx, cmd, req, queries, func(event SearchEvent) { events = append(events, event) })
			if err == nil || errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("invalid stream was accepted or stalled until timeout: %v", err)
			}
			want := 2
			if mode == "revision" || mode == "session" {
				want = 1
			}
			if len(events) != want {
				t.Fatalf("invalid event reached the consumer: %+v", events)
			}
			searchProtocolAssertJoined(t, cmd, queries)
		})
	}
}

func TestSearchProtocolCancellationJoinsProcessAndControls(t *testing.T) {
	for _, mode := range []string{"cancel_idle", "cancel_backpressured"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			req := searchProtocolRequest(t)
			queries := make(chan SearchQuery)
			cmd := searchProtocolCommand(t, ctx, mode)
			delivered := 0
			err := searchCommand(ctx, cmd, req, queries, func(event SearchEvent) {
				delivered++
				if event.Kind != SearchProgress {
					t.Errorf("unexpected pre-cancellation event: %+v", event)
				}
				if mode == "cancel_backpressured" {
					// The helper stops reading after the request. Admit a control
					// larger than a native pipe so shutdown must also unblock writing.
					query := SearchQuery{ID: 1, ReferencePath: strings.Repeat("x", 8*1024*1024)}
					select {
					case queries <- query:
					case <-ctx.Done():
						t.Error("control writer did not admit the cancellation fixture")
					}
				}
				cancel()
			})
			if !errors.Is(err, context.Canceled) || delivered != 1 {
				t.Fatalf("cancellation did not retire the active worker: delivered=%d, %v", delivered, err)
			}
			searchProtocolAssertJoined(t, cmd, queries)
		})
	}
	t.Run("held_control_writer", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		req := searchProtocolRequest(t)
		queries := make(chan SearchQuery)
		cmd := searchProtocolCommand(t, ctx, "cancel_idle")
		resume := make(chan struct{})
		var resumeOnce sync.Once
		releaseWriter := func() { resumeOnce.Do(func() { close(resume) }) }
		checking := &searchProtocolHeldWriterContext{
			Context: ctx, cmd: cmd, writerEntered: make(chan struct{}),
			resume: resume, processJoined: make(chan struct{}),
		}
		completed := make(chan error, 1)
		done := make(chan struct{})
		go func() {
			defer close(done)
			completed <- searchCommand(checking, cmd, req, queries, func(_ SearchEvent) { cancel() })
		}()
		t.Cleanup(func() { cancel(); releaseWriter(); searchProtocolWait(t, done) })
		searchProtocolWait(t, checking.writerEntered)
		searchProtocolWait(t, checking.processJoined)
		select {
		case err := <-completed:
			t.Fatalf("provider returned while its control writer was still held: %v", err)
		default:
		}
		releaseWriter()
		select {
		case err := <-completed:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("held-writer cancellation: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("provider did not join its released control writer")
		}
		searchProtocolAssertJoined(t, cmd, queries)
	})
}

func TestSearchProtocolExitBeforeReady(t *testing.T) {
	for _, mode := range []string{"early_exit", "final_without_ready", "provider_error"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			req := searchProtocolRequest(t)
			queries := searchProtocolQueries(t, ctx, req)
			cmd := searchProtocolCommand(t, ctx, mode)
			delivered := 0
			err := searchCommand(ctx, cmd, req, queries, func(_ SearchEvent) { delivered++ })
			wantError := "before readiness"
			if mode == "provider_error" {
				wantError = "fixture worker setup failed"
			}
			if err == nil || !strings.Contains(err.Error(), wantError) {
				t.Fatalf("early worker exit lost its failure: %v", err)
			}
			wantEvents := 0
			if mode == "final_without_ready" {
				wantEvents = 1
			}
			if delivered != wantEvents {
				t.Fatalf("early exit delivered %d events; want %d", delivered, wantEvents)
			}
			searchProtocolAssertJoined(t, cmd, queries)
		})
	}
}

// The exact helper test is the subprocess entry point. Ordinary suite execution
// leaves it inert; os.Exit keeps the testing runner's PASS text off the protocol.
func TestSearchProtocolHelperProcess(_ *testing.T) {
	mode := os.Getenv(searchProtocolHelperMode)
	if mode == "" {
		return
	}
	if err := searchProtocolWorker(mode); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func searchProtocolWorker(mode string) error {
	decoder := json.NewDecoder(os.Stdin)
	var req request
	if err := decoder.Decode(&req); err != nil {
		return err
	}
	if req.Search == nil || req.Search.SessionID != 77 || len(req.Search.Paths) != 2 || req.Search.Limit != 30 || req.MaxEncodedBytes != 8192 {
		return fmt.Errorf("worker received a changed request: %+v", req)
	}
	encoder := json.NewEncoder(os.Stdout)
	progress := SearchEvent{SessionID: 77, QueryID: 1, Revision: 1, Kind: SearchProgress, Processed: 1, Total: 2}
	ready := SearchEvent{SessionID: 77, QueryID: 1, Revision: 2, Kind: SearchReady, Processed: 2, Total: 2}
	if mode == "cancel_idle" || mode == "cancel_backpressured" {
		if err := encoder.Encode(progress); err != nil {
			return err
		}
		if mode == "cancel_idle" {
			_, err := io.Copy(io.Discard, os.Stdin)
			return err
		}
		// This child deliberately never drains controls. The deadline bounds a
		// broken cancellation harness; normal execution ends by parent cancellation.
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		<-ctx.Done()
		return ctx.Err()
	}
	var query SearchQuery
	if err := decoder.Decode(&query); err != nil {
		return err
	}
	if query.ID != 1 || query.ReferencePath != req.Search.Paths[0] {
		return fmt.Errorf("worker received a changed query: %+v", query)
	}
	switch mode {
	case "early_exit":
		return nil
	case "final_without_ready":
		return encoder.Encode(SearchEvent{SessionID: 77, QueryID: 1, Revision: 1, Kind: SearchFinal, Processed: 2, Total: 2})
	case "provider_error":
		return errors.New("fixture worker setup failed")
	case "revision", "session":
		if err := encoder.Encode(progress); err != nil {
			return err
		}
		if mode == "revision" {
			ready.Revision = progress.Revision
		} else {
			ready.SessionID++
		}
		return encoder.Encode(ready)
	case "malformed", "truncated", "unknown_kind":
		if err := encoder.Encode(progress); err != nil {
			return err
		}
		if err := encoder.Encode(ready); err != nil {
			return err
		}
		if mode == "unknown_kind" {
			ready.Revision++
			ready.Kind = "unknown"
			return encoder.Encode(ready)
		}
		raw := "not-json\n"
		if mode == "truncated" {
			raw = `{"SessionID":`
		}
		_, err := io.WriteString(os.Stdout, raw)
		return err
	case "clean":
	default:
		return fmt.Errorf("unknown search helper mode %q", mode)
	}
	for _, event := range []SearchEvent{
		{SessionID: 77, QueryID: 1, Revision: 1, Kind: SearchProgress, Processed: 1, Total: 2},
		{SessionID: 77, QueryID: 1, Revision: 2, Kind: SearchFinal, Processed: 2, Total: 2, Matches: []Match{{Path: req.Search.Paths[1], Score: .75}}},
		{SessionID: 77, QueryID: 1, Revision: 3, Kind: SearchReady, Processed: 2, Total: 2},
	} {
		if err := encoder.Encode(event); err != nil {
			return err
		}
	}
	if err := decoder.Decode(&query); !errors.Is(err, io.EOF) {
		return fmt.Errorf("controls did not close after readiness: %v", err)
	}
	return nil
}

func searchProtocolAssertJoined(t *testing.T, cmd *exec.Cmd, queries chan SearchQuery) {
	t.Helper()
	if cmd.ProcessState == nil {
		t.Fatal("provider returned before joining the subprocess")
	}
	select {
	case queries <- SearchQuery{ID: 2}:
		t.Fatal("control writer still accepted a query after provider return")
	default:
	}
}

// Only the control writer uses Done on the context passed to searchCommand;
// exec.CommandContext owns the ordinary parent context separately. Err observes
// cmd.ProcessState on the transport goroutine that calls cmd.Wait.
type searchProtocolHeldWriterContext struct {
	context.Context
	cmd           *exec.Cmd
	writerEntered chan struct{}
	resume        <-chan struct{}
	processJoined chan struct{}
	writerOnce    sync.Once
	joinedOnce    sync.Once
}

func (c *searchProtocolHeldWriterContext) Done() <-chan struct{} {
	c.writerOnce.Do(func() { close(c.writerEntered); <-c.resume })
	return c.Context.Done()
}

func (c *searchProtocolHeldWriterContext) Err() error {
	if c.cmd.ProcessState != nil {
		c.joinedOnce.Do(func() { close(c.processJoined) })
	}
	return c.Context.Err()
}

func searchProtocolWait(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("search transport did not reach its observable completion")
	}
}

func searchProtocolCommand(t *testing.T, ctx context.Context, mode string) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestSearchProtocolHelperProcess$", "-test.timeout=20s")
	cmd.Env = append(os.Environ(), searchProtocolHelperMode+"="+mode)
	return cmd
}

func searchProtocolRequest(t *testing.T) request {
	t.Helper()
	dir := t.TempDir()
	return request{MaxEncodedBytes: 8192, Search: &SearchRequest{SessionID: 77, Paths: []string{filepath.Join(dir, "reference.jpg"), filepath.Join(dir, "match.jpg")}, Limit: 30}}
}

func searchProtocolQueries(t *testing.T, ctx context.Context, req request) chan SearchQuery {
	t.Helper()
	queries := make(chan SearchQuery)
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case queries <- SearchQuery{ID: 1, ReferencePath: req.Search.Paths[0]}:
		case <-ctx.Done():
		}
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("initial query sender did not stop")
		}
	})
	return queries
}

func TestSearchProtocolRejectsOversizedRequestBeforeLaunch(t *testing.T) {
	req := searchProtocolRequest(t)
	path := filepath.Join(t.TempDir(), strings.Repeat("a", 4000))
	req.Search.Paths = make([]string, 20000)
	for i := range req.Search.Paths {
		req.Search.Paths[i] = fmt.Sprintf("%s%d", path, i)
	}
	cmd := searchProtocolCommand(t, context.Background(), "early_exit")
	err := searchCommand(context.Background(), cmd, req, nil, func(_ SearchEvent) {})
	if err == nil || !strings.Contains(err.Error(), "request exceeds") || cmd.Process != nil {
		t.Fatalf("oversized scope launched worker instead of rejecting admission: process=%v err=%v", cmd.Process, err)
	}
}

func TestSearchProtocolWorkerBudgetResetsBetweenMessages(t *testing.T) {
	payload := `"` + strings.Repeat("a", 32*1024*1024) + "\"\n"
	decoder := newWorkerDecoder(io.MultiReader(strings.NewReader(payload), strings.NewReader(payload), strings.NewReader(`{"ID":3,"ReferencePath":"/next.jpg"}`)))
	for i := range 2 {
		var value string
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
	}
	var query SearchQuery
	if err := decoder.Decode(&query); err != nil || query.ID != 3 {
		t.Fatalf("retained worker exhausted its lifetime input budget: %+v, %v", query, err)
	}
}
