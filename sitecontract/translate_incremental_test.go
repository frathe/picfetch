package sitecontract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestMakeTranslateRequestsOnlyChangedUnitsAndPrunesObsoleteEntries(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	var (
		mu        sync.Mutex
		requested []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var payload struct {
			Text []string `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		requested = append(requested, payload.Text...)
		mu.Unlock()
		translations := make([]map[string]string, len(payload.Text))
		for index, source := range payload.Text {
			translations[index] = map[string]string{"text": "Deutsch: " + source}
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	runTranslate := func(sourcePath string) {
		t.Helper()
		cmd := exec.Command("make", "translate", "SITE_SOURCE="+sourcePath, "SITE_TRANSLATIONS="+cachePath)
		cmd.Dir = repo
		cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
		cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("i", 24), "DEEPL_API_URL="+server.URL)
		if combined, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("make translate failed: %v\n%s", err, combined)
		}
	}
	runTranslate("website.md")

	type cacheEntry struct {
		SourceHash string `json:"source_hash"`
		Format     string `json:"format"`
		Text       string `json:"text"`
	}
	type cacheDocument struct {
		Version int                   `json:"version"`
		Locale  string                `json:"locale"`
		Entries map[string]cacheEntry `json:"entries"`
	}
	cacheData, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read initial cache: %v", err)
	}
	var cache cacheDocument
	if err := json.Unmarshal(cacheData, &cache); err != nil {
		t.Fatalf("parse initial cache: %v", err)
	}
	initialCount := len(cache.Entries)
	cache.Entries["obsolete.unit"] = cacheEntry{SourceHash: strings.Repeat("0", 64), Format: "text", Text: "Alt"}
	cacheData, err = json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("encode cache with obsolete entry: %v", err)
	}
	if err := os.WriteFile(cachePath, append(cacheData, '\n'), 0o600); err != nil {
		t.Fatalf("write cache with obsolete entry: %v", err)
	}

	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	changed := strings.Replace(string(source),
		"PicFetch — a small, fast image viewer for macOS, Windows and Linux",
		"PicFetch — a small, fast image viewer for macOS, Windows and Linux today",
		1,
	)
	if changed == string(source) {
		t.Fatal("test setup did not change one translation unit")
	}
	changedSource := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(changedSource, []byte(changed), 0o600); err != nil {
		t.Fatalf("write changed source: %v", err)
	}
	mu.Lock()
	requested = nil
	mu.Unlock()
	runTranslate(changedSource)

	mu.Lock()
	secondRequest := append([]string(nil), requested...)
	mu.Unlock()
	if len(secondRequest) != 1 {
		t.Fatalf("changed-source refresh requested %d units, want exactly one", len(secondRequest))
	}
	if !strings.Contains(secondRequest[0], "today") {
		t.Fatalf("changed-source refresh requested the wrong unit: %q", secondRequest[0])
	}
	after, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read refreshed cache: %v", err)
	}
	cache = cacheDocument{}
	if err := json.Unmarshal(after, &cache); err != nil {
		t.Fatalf("parse refreshed cache: %v", err)
	}
	if _, exists := cache.Entries["obsolete.unit"]; exists {
		t.Fatal("translation refresh retained an obsolete derived entry")
	}
	if len(cache.Entries) != initialCount {
		t.Fatalf("refreshed cache has %d entries, want %d", len(cache.Entries), initialCount)
	}
}

func TestMakeTranslateRepairsCacheWhenProtectionConfigurationChanges(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	var (
		mu        sync.Mutex
		requested []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var payload struct {
			Text []string `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		requested = append(requested, payload.Text...)
		mu.Unlock()
		translations := make([]map[string]string, len(payload.Text))
		for index, source := range payload.Text {
			translated := source
			if !strings.Contains(source, "<keep>viewer</keep>") {
				translated = strings.ReplaceAll(source, "viewer", "Betrachter")
			}
			translations[index] = map[string]string{"text": translated}
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	runTranslate := func(sourcePath string) {
		t.Helper()
		cmd := exec.Command("make", "translate", "SITE_SOURCE="+sourcePath, "SITE_TRANSLATIONS="+cachePath)
		cmd.Dir = repo
		cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
		cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("i", 24), "DEEPL_API_URL="+server.URL)
		if combined, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("make translate failed: %v\n%s", err, combined)
		}
	}
	runTranslate("website.md")

	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	changed := strings.Replace(string(source), "    - PicFetch\n", "    - PicFetch\n    - viewer\n", 1)
	if changed == string(source) {
		t.Fatal("test setup did not add a protected term")
	}
	changedSource := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(changedSource, []byte(changed), 0o600); err != nil {
		t.Fatalf("write source with changed protection configuration: %v", err)
	}
	mu.Lock()
	requested = nil
	mu.Unlock()
	runTranslate(changedSource)

	mu.Lock()
	repairRequest := append([]string(nil), requested...)
	mu.Unlock()
	if len(repairRequest) == 0 {
		t.Fatal("protection change did not request repair of invalid cached translations")
	}
	for _, text := range repairRequest {
		if !strings.Contains(text, "<keep>viewer</keep>") {
			t.Fatalf("repair request did not protect the newly configured term: %q", text)
		}
	}
}
