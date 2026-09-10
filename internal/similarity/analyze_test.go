package similarity

import (
	"context"
	"strings"
	"testing"
)

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
