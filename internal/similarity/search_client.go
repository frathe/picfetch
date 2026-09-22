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
	"slices"
	"time"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/imaging"
)

// Search retains a prepared worker until cancellation or query-channel closure.
// Its return observes subprocess exit and the control writer's completion.
func (c Client) Search(ctx context.Context, search SearchRequest, queries <-chan SearchQuery, emit func(SearchEvent)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !SupportedPlatform() {
		return fmt.Errorf("local similarity search is unavailable on this platform")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	assets := c.Assets
	if assets == "" {
		assets = defaultAssets(executable)
	}
	search.Paths = slices.Clone(search.Paths)
	req := request{Assets: assets, MaxEncodedBytes: imaging.MaxEncodedBytes(), Search: &search}
	req.captureHEIC(heic.FromContext(ctx))
	cmd := workerCommand(ctx, executable)
	cmd.Env = append(os.Environ(), workerEnvironment+"=1")
	return searchCommand(ctx, cmd, req, queries, emit)
}

func searchCommand(ctx context.Context, cmd *exec.Cmd, req request, queries <-chan SearchQuery, emit func(SearchEvent)) error {
	// Reject large path inventories before JSON encoding can duplicate them.
	if req.Search != nil {
		remaining := workerRequestLimit
		for _, path := range req.Search.Paths {
			if len(path) > remaining {
				return fmt.Errorf("search request exceeds the %d-byte worker request limit", workerRequestLimit)
			}
			remaining -= len(path)
		}
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if len(payload) > workerRequestLimit {
		return fmt.Errorf("search request exceeds the %d-byte worker request limit", workerRequestLimit)
	}
	payload = append(payload, '\n')
	input, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	output, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.WaitDelay = 3 * time.Second
	if err := cmd.Start(); err != nil {
		return err
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = input.Close() }()
		encoder := json.NewEncoder(input)
		if _, err := input.Write(payload); err != nil {
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case query, ok := <-queries:
				if !ok {
					return
				}
				if encoder.Encode(query) != nil {
					return
				}
			}
		}
	}()
	defer func() {
		close(stop)
		_ = input.Close()
		<-done
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	decoder := json.NewDecoder(output)
	var revision uint64
	ready := false
	for {
		var event SearchEvent
		err = decoder.Decode(&event)
		if err != nil {
			break
		}
		if ctx.Err() != nil {
			break
		}
		if event.SessionID != req.Search.SessionID || event.Revision <= revision || event.Processed < 0 || event.Processed > event.Total || event.Total < 0 || event.Failed < 0 || event.Failed > event.Processed || len(event.Matches) > 30 {
			err = fmt.Errorf("invalid search worker event")
			break
		}
		switch event.Kind {
		case SearchProgress, SearchPartial, SearchFinal, SearchReady, SearchQueryFailure, SearchFailure:
		default:
			err = fmt.Errorf("unknown search worker event")
		}
		if err != nil {
			break
		}
		revision = event.Revision
		ready = ready || event.Kind == SearchReady
		if event.HEICUnavailable {
			heic.ReportUnavailable(ctx)
		}
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
		return fmt.Errorf("search worker: %w: %s", waitErr, stderr.String())
	}
	if !errors.Is(err, io.EOF) {
		return err
	}
	if !ready {
		return fmt.Errorf("search worker exited before readiness")
	}
	return nil
}

func searchWorker(ctx context.Context, req request, decoder *workerDecoder, input io.Closer) error {
	readCtx, cancel := context.WithCancel(ctx)
	queries := make(chan SearchQuery, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(queries)
		for {
			var query SearchQuery
			if decoder.Decode(&query) != nil {
				return
			}
			select {
			case queries <- query:
			case <-readCtx.Done():
				return
			}
		}
	}()
	output := json.NewEncoder(os.Stdout)
	err := searchLocal(ctx, req, queries, func(event SearchEvent) error {
		event.HEICUnavailable = workerHEICUnavailable(ctx)
		return output.Encode(event)
	})
	cancel()
	_ = input.Close()
	<-done
	return err
}
