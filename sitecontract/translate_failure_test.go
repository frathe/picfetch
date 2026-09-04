package sitecontract_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestMakeTranslateRequiresCredentialWithoutTouchingCache(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	prior := []byte("{\n  \"version\": 1,\n  \"locale\": \"de\",\n  \"entries\": {}\n}\n")
	if err := os.WriteFile(cachePath, prior, 0o600); err != nil {
		t.Fatalf("write prior translation cache: %v", err)
	}

	cmd := exec.Command("make", "translate",
		"SITE_TRANSLATIONS="+cachePath,
		"DEEPL_ENV_FILE="+filepath.Join(t.TempDir(), "missing.env"),
	)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make translate succeeded without DEEPL_API_KEY")
	}
	if !strings.Contains(string(combined), "DEEPL_API_KEY") {
		t.Fatalf("missing credential diagnostic does not name DEEPL_API_KEY:\n%s", combined)
	}
	after, readErr := os.ReadFile(cachePath)
	if readErr != nil {
		t.Fatalf("read cache after failed translation: %v", readErr)
	}
	if !bytes.Equal(after, prior) {
		t.Fatalf("failed translation changed the prior cache\nwant: %q\n got: %q", prior, after)
	}
}

func TestMakeTranslateRejectsEndpointQueryOrFragmentWithoutTouchingCache(t *testing.T) {
	for _, suffix := range []string{"?tenant=x", "?", "#tenant", "#"} {
		suffix := suffix
		t.Run(suffix, func(t *testing.T) {
			repo := repositoryRoot(t)
			cachePath := filepath.Join(t.TempDir(), "de.json")
			prior := []byte("{\n  \"version\": 1,\n  \"locale\": \"de\",\n  \"entries\": {}\n}\n")
			if err := os.WriteFile(cachePath, prior, 0o600); err != nil {
				t.Fatalf("write prior translation cache: %v", err)
			}

			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				http.Error(response, "unexpected request", http.StatusInternalServerError)
			}))
			defer server.Close()

			cmd := exec.Command(
				"make",
				"translate",
				"SITE_TRANSLATIONS="+cachePath,
				"DEEPL_ENV_FILE="+filepath.Join(t.TempDir(), "missing.env"),
			)
			cmd.Dir = repo
			cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
			cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("q", 24), "DEEPL_API_URL="+server.URL+suffix)
			combined, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("make translate accepted a DeepL endpoint with a query or fragment")
			}
			if !strings.Contains(string(combined), "must not include a query or fragment") {
				t.Fatalf("query-or-fragment diagnostic is not actionable:\n%s", combined)
			}
			if got := requests.Load(); got != 0 {
				t.Fatalf("invalid DeepL endpoint made %d network request(s)", got)
			}
			assertFileBytes(t, cachePath, prior)
		})
	}
}

func TestMakeTranslateRejectsMalformedResponseWithoutTouchingCache(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	prior := []byte("{\n  \"version\": 1,\n  \"locale\": \"de\",\n  \"entries\": {}\n}\n")
	if err := os.WriteFile(cachePath, prior, 0o600); err != nil {
		t.Fatalf("write prior translation cache: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var body struct {
			Text []string `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		translations := make([]map[string]string, len(body.Text))
		for index, text := range body.Text {
			translations[index] = map[string]string{"text": text}
		}
		encoded, err := json.Marshal(map[string]any{"translations": translations})
		if err != nil {
			t.Errorf("encode fake response: %v", err)
			return
		}
		_, _ = fmt.Fprintf(w, "%s trailing garbage", encoded)
	}))
	defer server.Close()

	cmd := exec.Command("make", "translate", "SITE_TRANSLATIONS="+cachePath)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("x", 24), "DEEPL_API_URL="+server.URL)
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make translate accepted a response with trailing malformed data")
	}
	if !strings.Contains(string(combined), "malformed") {
		t.Fatalf("malformed-response diagnostic is unclear:\n%s", combined)
	}
	after, readErr := os.ReadFile(cachePath)
	if readErr != nil {
		t.Fatalf("read cache after failed translation: %v", readErr)
	}
	if !bytes.Equal(after, prior) {
		t.Fatalf("malformed translation response changed the prior cache")
	}
}

func TestMakeTranslateRejectsVisiblyEmptyContentWithoutTouchingCache(t *testing.T) {
	tests := []struct {
		name        string
		matches     func(string) bool
		translation string
		identity    string
	}{
		{
			name: "Markdown HTML",
			matches: func(text string) bool {
				return strings.Contains(text, "A small desktop app for quickly viewing")
			},
			translation: "<p> \t </p>",
			identity:    "hero.tagline",
		},
		{
			name:        "plain text with zero-width character",
			matches:     func(text string) bool { return text == "Close" },
			translation: "\u200b",
			identity:    "labels.lightbox-close",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := repositoryRoot(t)
			cachePath := filepath.Join(t.TempDir(), "de.json")
			prior := []byte("{\n  \"version\": 1,\n  \"locale\": \"de\",\n  \"entries\": {}\n}\n")
			if err := os.WriteFile(cachePath, prior, 0o600); err != nil {
				t.Fatalf("write prior translation cache: %v", err)
			}

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				var payload struct {
					Text []string `json:"text"`
				}
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					http.Error(response, "bad request", http.StatusBadRequest)
					return
				}
				translations := make([]map[string]string, len(payload.Text))
				for index, text := range payload.Text {
					translated := text
					if test.matches(text) {
						translated = test.translation
					}
					translations[index] = map[string]string{"text": translated}
				}
				_ = json.NewEncoder(response).Encode(map[string]any{"translations": translations})
			}))
			defer server.Close()

			cmd := exec.Command("make", "translate", "SITE_TRANSLATIONS="+cachePath)
			cmd.Dir = repo
			cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
			cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("x", 24), "DEEPL_API_URL="+server.URL)
			combined, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("make translate accepted visibly empty content")
			}
			if !strings.Contains(string(combined), test.identity) || !strings.Contains(string(combined), "no visible text") {
				t.Fatalf("empty-content diagnostic is not actionable:\n%s", combined)
			}
			assertFileBytes(t, cachePath, prior)
		})
	}
}

func TestBuildRejectsMissingGermanTranslations(t *testing.T) {
	repo := repositoryRoot(t)
	output := t.TempDir()
	missingCache := filepath.Join(t.TempDir(), "missing-de.json")

	cmd := exec.Command("make", "build", "SITE_TRANSLATIONS="+missingCache, "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=de", "SITE_FORMATS=regular")
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make build generated German output without a translation cache")
	}
	diagnostic := string(combined)
	if !strings.Contains(diagnostic, "missing or stale German translation") || !strings.Contains(diagnostic, "metadata.title") {
		t.Fatalf("missing-cache diagnostic is not actionable:\n%s", combined)
	}
	if _, statErr := os.Stat(filepath.Join(output, "de", "index.html")); !os.IsNotExist(statErr) {
		t.Fatalf("failed German build created output: %v", statErr)
	}
}

func TestMakeTranslateServiceFailuresLeaveCacheUntouched(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		diagnostic string
	}{
		{name: "authentication", status: http.StatusUnauthorized, diagnostic: "authentication failed"},
		{name: "rate limit", status: http.StatusTooManyRequests, diagnostic: "rate limit exceeded"},
		{name: "quota", status: 456, diagnostic: "quota exceeded"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := repositoryRoot(t)
			cachePath := filepath.Join(t.TempDir(), "de.json")
			prior := []byte("{\n  \"version\": 1,\n  \"locale\": \"de\",\n  \"entries\": {}\n}\n")
			if err := os.WriteFile(cachePath, prior, 0o600); err != nil {
				t.Fatalf("write prior cache: %v", err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				response.WriteHeader(testCase.status)
			}))
			defer server.Close()

			cmd := exec.Command("make", "translate", "SITE_TRANSLATIONS="+cachePath)
			cmd.Dir = repo
			cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
			cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("s", 24), "DEEPL_API_URL="+server.URL)
			combined, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("make translate accepted HTTP %d", testCase.status)
			}
			if !strings.Contains(string(combined), testCase.diagnostic) {
				t.Fatalf("HTTP %d diagnostic is unclear:\n%s", testCase.status, combined)
			}
			assertFileBytes(t, cachePath, prior)
		})
	}
}

func TestMakeTranslateRejectsPartialResponseWithoutTouchingCache(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	prior := []byte("{\n  \"version\": 1,\n  \"locale\": \"de\",\n  \"entries\": {}\n}\n")
	if err := os.WriteFile(cachePath, prior, 0o600); err != nil {
		t.Fatalf("write prior cache: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(response).Encode(map[string]any{
			"translations": []map[string]string{{"text": "only one result"}},
		})
	}))
	defer server.Close()

	cmd := exec.Command("make", "translate", "SITE_TRANSLATIONS="+cachePath)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("r", 24), "DEEPL_API_URL="+server.URL)
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make translate accepted a partial DeepL response")
	}
	if !strings.Contains(string(combined), "translations for") {
		t.Fatalf("partial-response diagnostic is unclear:\n%s", combined)
	}
	assertFileBytes(t, cachePath, prior)
}

func TestMakeTranslateNetworkFailureLeavesCacheUntouched(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	prior := []byte("{\n  \"version\": 1,\n  \"locale\": \"de\",\n  \"entries\": {}\n}\n")
	if err := os.WriteFile(cachePath, prior, 0o600); err != nil {
		t.Fatalf("write prior cache: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := server.URL
	server.Close()

	cmd := exec.Command("make", "translate", "SITE_TRANSLATIONS="+cachePath)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("n", 24), "DEEPL_API_URL="+endpoint)
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make translate accepted a network failure")
	}
	if !strings.Contains(string(combined), "translation request failed") {
		t.Fatalf("network-failure diagnostic is unclear:\n%s", combined)
	}
	assertFileBytes(t, cachePath, prior)
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s changed after failed translation", path)
	}
}

func withoutEnvironment(environment []string, names ...string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		remove := false
		for _, name := range names {
			if strings.HasPrefix(entry, name+"=") {
				remove = true
				break
			}
		}
		if !remove {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
