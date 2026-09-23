//go:build linux || darwin

package similarity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestHEICAnalysisWorkerCancellationJoinsRetirement(t *testing.T) {
	t.Setenv("PICFETCH_TEST_HEIC_CAPTURE", "1")
	for _, kind := range []string{"finite", "retained"} {
		t.Run(kind, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "retired")
			t.Setenv("PICFETCH_TEST_HEIC_RETIREMENT", marker)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var err error
			if kind == "finite" {
				err = (Client{}).Analyze(ctx, nil, nil, func(_ Event) { cancel() })
			} else {
				err = (Client{}).Search(ctx, SearchRequest{SessionID: 1}, nil, func(_ SearchEvent) { cancel() })
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("producer cancellation: %v", err)
			}
			data, err := os.ReadFile(marker)
			if err != nil || string(data) != "joined" {
				t.Fatalf("producer returned before child-local retirement: %q, %v", data, err)
			}
		})
	}
}
