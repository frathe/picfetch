package sitecontract_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestMakeTranslateBatchesRequestsWithinDeepLLimits(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	largeSource := strings.Replace(string(source),
		"Played back frame by frame at their encoded speed,",
		"Played back frame by frame at their encoded speed,"+strings.Repeat(" extra", 12_000), 1)
	largeSource = strings.Replace(largeSource,
		"Drop folders to scan them recursively,",
		"Drop folders to scan them recursively,"+strings.Repeat(" more", 12_000), 1)
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(largeSource), 0o600); err != nil {
		t.Fatalf("write large website source: %v", err)
	}

	var mu sync.Mutex
	requestCount := 0
	largestBatch := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		body, readErr := io.ReadAll(request.Body)
		if readErr != nil {
			http.Error(w, "read request", http.StatusBadRequest)
			return
		}
		if len(body) > 100*1024 {
			http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
			return
		}
		var payload struct {
			Text []string `json:"text"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "bad JSON", http.StatusBadRequest)
			return
		}
		if len(payload.Text) > 50 {
			http.Error(w, "too many texts", http.StatusBadRequest)
			return
		}
		mu.Lock()
		requestCount++
		if len(payload.Text) > largestBatch {
			largestBatch = len(payload.Text)
		}
		mu.Unlock()
		translations := make([]map[string]string, len(payload.Text))
		for index, text := range payload.Text {
			translations[index] = map[string]string{"text": text}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	cmd := exec.Command("make", "translate", "SITE_SOURCE="+sourcePath, "SITE_TRANSLATIONS="+filepath.Join(t.TempDir(), "de.json"))
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("k", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make translate did not batch large input: %v\n%s", err, combined)
	}
	mu.Lock()
	gotRequestCount, gotLargestBatch := requestCount, largestBatch
	mu.Unlock()
	if gotRequestCount < 2 {
		t.Fatalf("large translation used %d request, want at least two bounded batches", gotRequestCount)
	}
	if gotLargestBatch > 50 {
		t.Fatalf("large translation sent %d texts in one request, want at most 50", gotLargestBatch)
	}
}
