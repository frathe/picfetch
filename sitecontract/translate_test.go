package sitecontract_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMakeTranslateUsesDeepLContractAndWritesCache(t *testing.T) {
	repo := translationRepositoryRoot(t)
	translations := filepath.Join(t.TempDir(), "de.json")
	apiKey := "contract-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	var (
		mu                 sync.Mutex
		requestCount       int
		largestBatch       int
		requestError       string
		sawRenderedHTML    bool
		sawProtectedKeep   bool
		rawMarkdownHeading = regexp.MustCompile(`(?m)^\s*#{1,6}\s+`)
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/translate" {
			http.Error(w, "expected POST /v2/translate", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "DeepL-Auth-Key "+apiKey {
			http.Error(w, "unexpected authorization", http.StatusUnauthorized)
			return
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			http.Error(w, "expected JSON request", http.StatusBadRequest)
			return
		}

		var request struct {
			Text               []string `json:"text"`
			TargetLang         string   `json:"target_lang"`
			SourceLang         string   `json:"source_lang"`
			TagHandling        string   `json:"tag_handling"`
			TagHandlingVersion string   `json:"tag_handling_version"`
			IgnoreTags         []string `json:"ignore_tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid JSON request", http.StatusBadRequest)
			return
		}

		mu.Lock()
		requestCount++
		if len(request.Text) > largestBatch {
			largestBatch = len(request.Text)
		}
		defer mu.Unlock()
		if len(request.Text) > 50 {
			requestError = fmt.Sprintf("request contains %d texts, want at most 50", len(request.Text))
		}
		if request.TargetLang != "DE" {
			requestError = "target_lang must be DE"
		}
		if request.SourceLang != "EN" {
			requestError = "source_lang must be EN"
		}
		if request.TagHandling != "html" {
			requestError = "tag_handling must be html"
		}
		if request.TagHandlingVersion != "v2" {
			requestError = "tag_handling_version must be v2"
		}
		ignoreTags := make(map[string]bool)
		for _, tag := range request.IgnoreTags {
			ignoreTags[tag] = true
		}
		for _, tag := range []string{"keep", "code", "kbd"} {
			if !ignoreTags[tag] {
				requestError = "ignore_tags must contain " + tag
			}
		}

		translations := make([]map[string]string, len(request.Text))
		for index, text := range request.Text {
			if rawMarkdownHeading.MatchString(text) {
				requestError = "request contains a raw Markdown heading"
			}
			if strings.Contains(text, "<") && strings.Contains(text, ">") {
				sawRenderedHTML = true
			}
			if strings.Contains(text, "<keep>") && strings.Contains(text, "</keep>") {
				sawProtectedKeep = true
			}
			translations[index] = map[string]string{
				"detected_source_language": "EN",
				"text":                     fmt.Sprintf("German translation %d: %s", index, text),
			}
		}
		response := map[string]any{"translations": translations}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("write fake DeepL response: %v", err)
		}
	}))
	defer server.Close()

	cmd := exec.Command("make", "translate", "SITE_TRANSLATIONS="+translations)
	cmd.Dir = repo
	cmd.Env = withoutEnv(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+apiKey, "DEEPL_API_URL="+server.URL)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make translate failed: %v\n%s", err, combined)
	}
	if strings.Contains(string(combined), apiKey) {
		t.Fatalf("make translate leaked the API key in its output")
	}

	mu.Lock()
	gotRequestCount, gotLargestBatch, gotRequestError := requestCount, largestBatch, requestError
	gotRenderedHTML, gotProtectedKeep := sawRenderedHTML, sawProtectedKeep
	mu.Unlock()
	if gotRequestCount != 2 {
		t.Fatalf("make translate made %d DeepL requests, want two batches for more than 50 texts", gotRequestCount)
	}
	if gotLargestBatch > 50 {
		t.Fatalf("make translate sent %d texts in one request, want at most 50", gotLargestBatch)
	}
	if gotRequestError != "" {
		t.Fatal(gotRequestError)
	}
	if !gotRenderedHTML {
		t.Fatal("DeepL input did not contain rendered HTML")
	}
	if !gotProtectedKeep {
		t.Fatal("DeepL input did not contain protected <keep> content")
	}

	cache, err := os.ReadFile(translations)
	if err != nil {
		t.Fatalf("read generated German cache: %v", err)
	}
	var document struct {
		Version int    `json:"version"`
		Locale  string `json:"locale"`
		Entries map[string]struct {
			SourceHash  string `json:"source_hash"`
			RequestHash string `json:"request_hash"`
			Text        string `json:"text"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(cache, &document); err != nil {
		t.Fatalf("generated German cache is not complete JSON: %v", err)
	}
	if document.Version == 0 {
		t.Fatalf("generated German cache is missing a version")
	}
	if document.Locale != "de" {
		t.Fatalf("generated German cache locale = %q, want de", document.Locale)
	}
	if len(document.Entries) == 0 {
		t.Fatal("generated German cache has no entries")
	}
	for id, entry := range document.Entries {
		if strings.TrimSpace(entry.SourceHash) == "" {
			t.Errorf("cache entry %q has an empty source hash", id)
		}
		if strings.TrimSpace(entry.RequestHash) == "" {
			t.Errorf("cache entry %q has an empty request hash", id)
		}
		if strings.TrimSpace(entry.Text) == "" {
			t.Errorf("cache entry %q has empty translated text", id)
		}
	}
}

func TestMakeTranslateCanReadIgnoredLocalEnvironmentFile(t *testing.T) {
	repo := repositoryRoot(t)
	apiKey := "local-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	environmentPath := filepath.Join(t.TempDir(), ".env.local")
	if err := os.WriteFile(environmentPath, []byte("DEEPL_API_KEY="+apiKey+"\n"), 0o600); err != nil {
		t.Fatalf("write local environment file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "DeepL-Auth-Key "+apiKey {
			http.Error(w, "missing key from env file", http.StatusUnauthorized)
			return
		}
		var payload struct {
			Text []string `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		translations := make([]map[string]string, len(payload.Text))
		for index, text := range payload.Text {
			translations[index] = map[string]string{"text": text}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	cmd := exec.Command("make", "translate",
		"DEEPL_ENV_FILE="+environmentPath,
		"SITE_TRANSLATIONS="+filepath.Join(t.TempDir(), "de.json"),
	)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_URL="+server.URL)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make translate did not load the local environment file: %v\n%s", err, combined)
	}
	if strings.Contains(string(combined), apiKey) {
		t.Fatal("make translate printed the key loaded from the local environment file")
	}
}

func TestMakeTranslateReusesCurrentCacheWithoutCredentialOrNetwork(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var payload struct {
			Text []string `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		translations := make([]map[string]string, len(payload.Text))
		for index, text := range payload.Text {
			translations[index] = map[string]string{"text": text}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"translations": translations})
	}))

	first := exec.Command("make", "translate", "SITE_TRANSLATIONS="+cachePath)
	first.Dir = repo
	first.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	first.Env = append(first.Env, "DEEPL_API_KEY="+strings.Repeat("c", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := first.CombinedOutput(); err != nil {
		server.Close()
		t.Fatalf("create current cache: %v\n%s", err, combined)
	}
	server.Close()
	before, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read current cache: %v", err)
	}

	second := exec.Command("make", "translate",
		"SITE_TRANSLATIONS="+cachePath,
		"DEEPL_ENV_FILE="+filepath.Join(t.TempDir(), "missing.env"),
	)
	second.Dir = repo
	second.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	combined, err := second.CombinedOutput()
	if err != nil {
		t.Fatalf("reuse current cache without credentials failed: %v\n%s", err, combined)
	}
	after, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read reused cache: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("credential-free reuse changed an already-current cache")
	}
}

func TestMakeTranslateProtectsKeyboardAndCodeContents(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	var sawNestedProtection bool
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var payload struct {
			Text []string `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}
		translations := make([]map[string]string, len(payload.Text))
		for index, source := range payload.Text {
			if strings.Contains(source, "<kbd><keep>d</keep></kbd>") && strings.Contains(source, "<kbd><keep>Shift</keep></kbd>") {
				sawNestedProtection = true
			}
			// Simulate the production behavior that exposed this regression: a
			// short key label inside an ordinary kbd tag is normalized.
			translated := strings.ReplaceAll(source, "<kbd>d</kbd>", "<kbd>D</kbd>")
			translations[index] = map[string]string{"text": translated}
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	cmd := exec.Command("make", "translate", "SITE_TRANSLATIONS="+cachePath)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("p", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make translate did not protect keyboard contents: %v\n%s", err, combined)
	}
	if !sawNestedProtection {
		t.Fatal("DeepL request did not independently protect keyboard contents")
	}
	cache, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read generated cache: %v", err)
	}
	var document struct {
		Entries map[string]struct {
			Text string `json:"text"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(cache, &document); err != nil {
		t.Fatalf("parse generated cache: %v", err)
	}
	translation := document.Entries["features.hide-duplicates.body"].Text
	if !strings.Contains(translation, "<kbd>d</kbd> and <kbd>Shift</kbd>+<kbd>D</kbd>") {
		t.Fatal("generated cache did not preserve the exact keyboard labels")
	}
}

func withoutEnv(environment []string, names ...string) []string {
	blocked := make(map[string]bool, len(names))
	for _, name := range names {
		blocked[name] = true
	}
	filtered := make([]string, 0, len(environment))
	for _, value := range environment {
		name, _, ok := strings.Cut(value, "=")
		if !ok || !blocked[name] {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func translationRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate contract test source")
	}
	return filepath.Dir(filepath.Dir(filename))
}
