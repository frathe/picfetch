package similarity

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalysisCacheUsageAndClearConfinement(t *testing.T) {
	roots := CacheRoots{GeneralDir: t.TempDir(), FavoritesDir: t.TempDir()}
	records := filepath.Join(roots.GeneralDir, "v1")
	if err := os.Mkdir(records, 0700); err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(records, strings.Repeat("a", 64)+".json")
	untouched := filepath.Join(roots.GeneralDir, "original.jpg")
	for _, p := range []string{managed, untouched} {
		if err := os.WriteFile(p, []byte("12345"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	quiesced := false
	manager := CacheManager{Quiesce: func(_ context.Context, _ CacheRoots) error { quiesced = true; return nil }}
	usage, err := manager.Inspect(context.Background(), roots, nil)
	if err != nil || usage.General.Bytes != 5 || usage.General.Records != 1 || quiesced {
		t.Fatalf("usage %+v quiesced=%v: %v", usage, quiesced, err)
	}
	report, err := manager.Clean(context.Background(), CacheCleanRequest{Roots: roots, Mode: ClearAll}, nil)
	if err != nil || !quiesced || report.RemovedBytes != 5 || report.RemovedRecords != 1 || report.Remaining.General.Bytes != 0 {
		t.Fatalf("cleanup %+v quiesced=%v: %v", report, quiesced, err)
	}
	if _, err := os.Stat(untouched); err != nil {
		t.Fatalf("cleanup touched original: %v", err)
	}
}
