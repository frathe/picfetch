package similarity

import (
	"context"
	"strings"
	"testing"
)

func TestAnalysisRejectsOversizedCollection(t *testing.T) {
	if err := validateAnalysisSourceCount(maxAnalysisSources); err != nil {
		t.Fatalf("limit boundary rejected: %v", err)
	}
	paths := make([]string, maxAnalysisSources+1)
	err := (Client{}).Analyze(context.Background(), paths, nil, func(Event) {})
	if err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("Analyze error = %v; want collection limit rejection", err)
	}
}

func TestWorkerEventDecoderRejectsOversizedMessage(t *testing.T) {
	decoder := newWorkerEventDecoder(strings.NewReader(`{"Stage":"`+strings.Repeat("x", 32)+`"}`+"\n"), 16)
	var event Event
	err := decoder.Decode(&event)
	if err == nil || !strings.Contains(err.Error(), "token too long") {
		t.Fatalf("Decode error = %v; want oversized token rejection", err)
	}
}

func TestWorkerEventDecoderStreamsMessages(t *testing.T) {
	decoder := newWorkerEventDecoder(strings.NewReader("{\"Stage\":\"first\"}\n{\"Stage\":\"second\"}\n"), 1024)
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
