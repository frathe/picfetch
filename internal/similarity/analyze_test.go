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
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAnalysisLimitsAllowQualifiedLibrary(t *testing.T) {
	limits := AnalysisLimits{MemoryMB: 2048, Items: 50655}.Normalized()
	if err := limits.validateSourceCount(50655); err != nil {
		t.Fatal(err)
	}
	if err := limits.validateSourceCount(50656); !errors.Is(err, ErrAnalysisItemLimit) {
		t.Fatalf("limit error = %v", err)
	}
	if got := (AnalysisLimits{}).Normalized(); got.MemoryMB != 512 || got.Items != 10000 {
		t.Fatalf("defaults = %+v", got)
	}
	if got := (AnalysisLimits{MemoryMB: MaxAnalysisMemoryMB + 1, Items: -1}).Normalized(); got.MemoryMB != 512 || got.Items != 10000 {
		t.Fatalf("invalid limits = %+v", got)
	}
	err := (Client{AnalysisLimits: AnalysisLimits{Items: 2}}).Analyze(context.Background(), []string{"a", "b", "c"}, nil, func(_ Event) { t.Fatal("over-limit collection published") })
	if !errors.Is(err, ErrAnalysisItemLimit) {
		t.Fatalf("client lost configured limit: %v", err)
	}
}

func TestAnalyzerStorePersistsLooseRepresentations(t *testing.T) {
	for _, looseEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("loose-enabled=%t", looseEnabled), func(t *testing.T) {
			policy := cacheTestPolicy(t)
			policy.GeneralLimitBytes = 64 * 1024
			req := request{FavoritesDir: policy.Roots.FavoritesDir, GeneralAnalysisLimitBytes: policy.GeneralLimitBytes}
			if looseEnabled {
				req.GeneralAnalysisDir = policy.Roots.GeneralDir
			}
			store, err := openAnalyzerRepresentationStore(context.Background(), req)
			if store != nil {
				defer store.close()
			}
			if err != nil {
				t.Fatal(err)
			}
			if looseEnabled && store.policy.GeneralLimitBytes != policy.GeneralLimitBytes {
				t.Fatalf("analyzer cache limit = %d, want %d", store.policy.GeneralLimitBytes, policy.GeneralLimitBytes)
			}
			if err := store.write(context.Background(), cacheFixtureItem(t, "loose.jpg")); err != nil {
				t.Fatal(err)
			}
			wantGeneralRecords := 0
			if looseEnabled {
				wantGeneralRecords = 1
			}
			usage, err := (CacheManager{}).Inspect(context.Background(), policy.Roots, nil)
			if err != nil || usage.Favorite.Records != 0 || usage.General.Records != wantGeneralRecords {
				t.Fatalf("analyzer cache effects: %+v, %v", usage, err)
			}
		})
	}
}

func TestAnalyzerCacheWriteReportsPressureAfterCompletion(t *testing.T) {
	event := Event{}
	recordAnalyzerCacheWrite(&event, CachePressureError{NeedBytes: 64})
	recordAnalyzerCacheWrite(&event, CachePressureError{NeedBytes: 128})
	if event.CachePressureBytes != 128 || event.CacheWarning != "" {
		t.Fatalf("cache pressure = %d, warning = %q", event.CachePressureBytes, event.CacheWarning)
	}
	recordAnalyzerCacheWrite(&event, errors.New("cache unavailable"))
	if event.CacheWarning != "cache unavailable" {
		t.Fatalf("cache failure warning = %q", event.CacheWarning)
	}
	for _, complete := range []bool{false, true} {
		delivered := analysisEventForDelivery(Event{Complete: complete, CachePressureBytes: event.CachePressureBytes}, time.Second)
		want := uint64(0)
		if complete {
			want = event.CachePressureBytes
		}
		if delivered.CachePressureBytes != want {
			t.Fatalf("complete=%t delivered pressure = %d, want %d", complete, delivered.CachePressureBytes, want)
		}
	}
}

func TestWorkerEventProtocolChunksSnapshots(t *testing.T) {
	const budget = 64 * 1024
	event := Event{Complete: true, Total: 3, Successful: 3, Stage: "complete",
		Items:  []Item{{Path: "/a.jpg", Preview: bytes.Repeat([]byte{1}, 1024)}, {Path: "/b.jpg", Preview: bytes.Repeat([]byte{2}, 1024)}, {Path: "/c.jpg", Preview: bytes.Repeat([]byte{3}, 1024)}},
		Merges: []CohortMerge{{Left: "a", Right: "b"}},
	}
	var output bytes.Buffer
	encoder := newWorkerEventEncoder(&output, budget)
	if err := encoder.Encode(event); err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte{'\n'})
	if len(lines) < 5 {
		t.Fatalf("snapshot sent as %d message(s); want independent bounded chunks", len(lines))
	}
	for _, line := range lines {
		if len(line) > 2048 {
			t.Fatalf("frame contains multiple previews: %d bytes", len(line))
		}
	}
	if err := encoder.Encode(Event{Stage: "next"}); err != nil {
		t.Fatal(err)
	}
	decoder := newWorkerEventDecoder(&output, budget, 10000)
	var got Event
	if err := decoder.Decode(&got); err != nil || !reflect.DeepEqual(got, event) {
		t.Fatalf("snapshot changed: %+v, %v", got, err)
	}
	if err := decoder.Decode(&got); err != nil || got.Stage != "next" || got.Items != nil || got.Complete {
		t.Fatalf("next snapshot retained old state: %+v, %v", got, err)
	}
	if err := decoder.Decode(&got); !errors.Is(err, io.EOF) {
		t.Fatalf("end of stream = %v", err)
	}
}

func TestWorkerEventProtocolSnapshotBudget(t *testing.T) {
	event := Event{Stage: "complete", Complete: true, Items: []Item{{Path: "/a.jpg", Preview: bytes.Repeat([]byte{1}, 512)}, {Path: "/b.jpg", Preview: bytes.Repeat([]byte{2}, 512)}}}
	var output bytes.Buffer
	if err := newWorkerEventEncoder(&output, 16*1024).Encode(event); err != nil {
		t.Fatal(err)
	}
	wire := bytes.Clone(output.Bytes())
	for _, budget := range []int{len(wire), len(wire) - 1} {
		var encoded bytes.Buffer
		err := newWorkerEventEncoder(&encoded, budget).Encode(event)
		if budget == len(wire) && err != nil || budget < len(wire) && !errors.Is(err, ErrAnalysisMemoryLimit) {
			t.Fatalf("encoder budget=%d, bytes=%d: %v", budget, len(wire), err)
		}
		got := Event{Stage: "previous"}
		err = newWorkerEventDecoder(bytes.NewReader(wire), budget, 2).Decode(&got)
		if budget == len(wire) {
			if err != nil || !reflect.DeepEqual(got, event) {
				t.Fatalf("exact budget rejected: %v", err)
			}
		} else if !errors.Is(err, ErrAnalysisMemoryLimit) || got.Stage != "previous" {
			t.Fatalf("aggregate overflow published a partial map or lost cause: stage=%q, items=%d, %v", got.Stage, len(got.Items), err)
		}
	}
	var got Event
	if err := newWorkerEventDecoder(bytes.NewReader(wire), len(wire), 1).Decode(&got); !errors.Is(err, ErrAnalysisItemLimit) {
		t.Fatalf("item budget = %v", err)
	}
	// Removing the final marker must never publish a partial map.
	end := bytes.LastIndex(wire[:len(wire)-1], []byte{'\n'}) + 1
	if err := newWorkerEventDecoder(bytes.NewReader(wire[:end]), len(wire), 2).Decode(&got); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("truncated map = %v", err)
	}
}

func TestWorkerEventProtocolRejectsInvalidFrames(t *testing.T) {
	for _, wire := range []string{
		"null\n",
		"{\"End\":true}\n",
		"{\"Snapshot\":{},\"End\":true}\n",
		"{\"Snapshot\":{\"Items\":[{}]}}\n",
		"{\"Snapshot\":{}}\n{\"Snapshot\":{}}\n",
		"{\"Snapshot\":{}}\n{\"Item\":{}}\n",
		"{\"Snapshot\":{},\"HasItems\":true}\n{\"Merge\":{}}\n",
	} {
		got := Event{Stage: "previous"}
		if err := newWorkerEventDecoder(strings.NewReader(wire), 4096, 2).Decode(&got); err == nil || got.Stage != "previous" {
			t.Fatalf("invalid stream %q published a result: stage=%q, %v", wire, got.Stage, err)
		}
	}
}

func TestAnalysisProtocolPreservesLimitErrorsAndConfiguration(t *testing.T) {
	for _, mode := range []string{"limit", "complete"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			// Allow the large streamed result and race-detector shutdown the
			// same bounded budget as the search protocol helper.
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestAnalysisProtocolHelperProcess$", "-test.timeout=20s")
			cmd.Env = append(os.Environ(), "PICFETCH_TEST_ANALYSIS_PROTOCOL="+mode)
			paths := make([]string, 50655)
			for i := range paths {
				paths[i] = fmt.Sprintf("/library/%d.jpg", i)
			}
			req := request{Paths: paths, AnalysisLimits: AnalysisLimits{MemoryMB: 2048, Items: 50655}}
			var events []Event
			err = analyzeCommand(ctx, cmd, req, nil, func(e Event) { events = append(events, e) })
			if cmd.ProcessState == nil {
				t.Fatal("analysis returned before worker exit")
			}
			if mode == "limit" {
				if !errors.Is(err, ErrAnalysisMemoryLimit) || len(events) != 0 {
					t.Fatalf("worker failure hid limit or published partial map: %v, %v", err, events)
				}
			} else if err != nil || len(events) != 1 || !events[0].Complete || len(events[0].Items) != 50655 {
				t.Fatalf("configured worker did not complete: %v, events=%d", err, len(events))
			}
		})
	}
}

func TestAnalysisProtocolHelperProcess(_ *testing.T) {
	mode := os.Getenv("PICFETCH_TEST_ANALYSIS_PROTOCOL")
	if mode == "" {
		return
	}
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil || req.AnalysisLimits.MemoryMB != 2048 || req.AnalysisLimits.Items != 50655 || len(req.Paths) != 50655 {
		_, _ = fmt.Fprintln(os.Stderr, "analysis configuration not captured", err)
		os.Exit(2)
	}
	if mode == "limit" {
		// A tiny synthetic budget exercises error transport without a large allocation.
		err := newWorkerEventEncoder(os.Stdout, 32).Encode(Event{Stage: "complete", Complete: true})
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	items := make([]Item, len(req.Paths))
	for i := range items {
		items[i] = Item{Path: req.Paths[i]}
	}
	if err := newWorkerEventEncoder(os.Stdout, 32*1024*1024).Encode(Event{Complete: true, Stage: "complete", Total: len(items), Items: items}); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestAnalysisRejectsOversizedCollection(t *testing.T) {
	if err := (AnalysisLimits{}).validateSourceCount(10000); err != nil {
		t.Fatalf("limit boundary rejected: %v", err)
	}
	paths := make([]string, 10001)
	err := (Client{}).Analyze(context.Background(), paths, nil, func(Event) {})
	if err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("Analyze error = %v; want collection limit rejection", err)
	}
}

func TestWorkerEventDecoderRejectsOversizedMessage(t *testing.T) {
	decoder := newWorkerEventDecoder(strings.NewReader(`{"Stage":"`+strings.Repeat("x", 32)+`"}`+"\n"), 16, 10000)
	var event Event
	err := decoder.Decode(&event)
	if err == nil || !strings.Contains(err.Error(), "token too long") {
		t.Fatalf("Decode error = %v; want oversized token rejection", err)
	}
}

func TestWorkerEventDecoderStreamsMessages(t *testing.T) {
	var output bytes.Buffer
	encoder := newWorkerEventEncoder(&output, 1024)
	for _, stage := range []string{"first", "second"} {
		if err := encoder.Encode(Event{Stage: stage}); err != nil {
			t.Fatal(err)
		}
	}
	decoder := newWorkerEventDecoder(&output, 1024, 10000)
	for _, want := range []string{"first", "second"} {
		var event Event
		if err := decoder.Decode(&event); err != nil || event.Stage != want {
			t.Fatalf("Decode = stage %q, error %v; want %q", event.Stage, err, want)
		}
	}
}

func TestPublishAnalysisMapRequiresSuccessfulSources(t *testing.T) {
	for _, complete := range []bool{false, true} {
		event := Event{Total: 2, Failed: 2}
		publications := 0
		err := publishAnalysisMap(context.Background(), &event, []Item{{Path: "a.jpg", Error: "decode failed"}, {Path: "b.jpg", Error: "encode failed"}}, complete, func(e Event) error {
			if e.Complete || e.Items != nil {
				publications++
			}
			return nil
		})
		if err == nil || !strings.Contains(err.Error(), "no images were represented successfully") || publications != 0 {
			t.Fatalf("complete=%v: error=%v publications=%d; want failed analysis without a map", complete, err, publications)
		}
	}
}
