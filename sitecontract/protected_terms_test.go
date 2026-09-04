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

func TestMakeTranslatePreservesHTMLSignificantProtectedTextTerm(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	modified := strings.Replace(string(source), "    - Fyne\n", "    - Fyne\n    - 'AT&T'\n", 1)
	modified = strings.Replace(modified,
		"PicFetch — a small, fast image viewer for macOS, Windows and Linux",
		"PicFetch — a small, fast image viewer for macOS, Windows and Linux for AT&T",
		1,
	)
	if modified == string(source) || !strings.Contains(modified, "for AT&T") {
		t.Fatal("test setup did not add an HTML-significant protected term")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
		t.Fatalf("write modified source: %v", err)
	}
	server := echoingDeepLServer(t)
	defer server.Close()

	cachePath := filepath.Join(t.TempDir(), "de.json")
	cmd := exec.Command("make", "translate",
		"SITE_SOURCE="+sourcePath,
		"SITE_TRANSLATIONS="+cachePath,
	)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("h", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make translate rejected unchanged protected term AT&T: %v\n%s", err, combined)
	}
}

func TestMakeTranslateHandlesProtectedTermsInMarkdownAttributes(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	needle := "A small desktop app for quickly viewing and browsing images. Drop one or more onto the window, and step through the set with the keyboard."
	replacement := needle + ` Read the [PicFetch guide](https://example.test/PicFetch.pdf "PicFetch JPEGs manual").`
	modified := strings.Replace(string(source), needle, replacement, 1)
	if modified == string(source) || !strings.Contains(modified, `https://example.test/PicFetch.pdf "PicFetch JPEGs manual"`) {
		t.Fatal("test setup did not add protected terms to Markdown link attributes")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
		t.Fatalf("write modified source: %v", err)
	}
	server := echoingDeepLServer(t)
	defer server.Close()

	cachePath := filepath.Join(t.TempDir(), "de.json")
	cmd := exec.Command("make", "translate",
		"SITE_SOURCE="+sourcePath,
		"SITE_TRANSLATIONS="+cachePath,
	)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("h", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make translate rejected unchanged protected terms in Markdown attributes: %v\n%s", err, combined)
	}

	cache, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read translated cache: %v", err)
	}
	changed := strings.Replace(string(cache), "PicFetch JPEGs manual", "OtherProduct JPEGs manual", 1)
	if changed == string(cache) {
		t.Fatal("test setup did not change the protected term in the Markdown title attribute")
	}
	if err := os.WriteFile(cachePath, []byte(changed), 0o600); err != nil {
		t.Fatalf("write cache with changed Markdown title attribute: %v", err)
	}
	build := exec.Command("make", "build",
		"SITE_SOURCE="+sourcePath,
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+t.TempDir(),
		"SITE_LOCALES=de",
		"SITE_FORMATS=regular",
	)
	build.Dir = repo
	combined, err := build.CombinedOutput()
	if err == nil {
		t.Fatal("make build accepted a changed protected term in a Markdown title attribute")
	}
	if !strings.Contains(string(combined), "hero.tagline") || !strings.Contains(string(combined), "protected term") {
		t.Fatalf("changed Markdown title attribute diagnostic is not actionable:\n%s", combined)
	}
}

func TestMakeTranslateHandlesProtectedTermsThatOccurInInternalMarkerNames(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	modified := strings.Replace(string(source), "    - PicFetch\n", "    - PicFetch\n    - PICFETCH\n    - TOKEN\n", 1)
	modified = strings.Replace(modified,
		"PicFetch — a small, fast image viewer for macOS, Windows and Linux",
		"PicFetch — a small, fast image viewer for macOS, Windows and Linux — PICFETCH TOKEN",
		1,
	)
	if modified == string(source) || !strings.Contains(modified, "— PICFETCH TOKEN") {
		t.Fatal("test setup did not add protected terms that collide with internal marker names")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
		t.Fatalf("write modified source: %v", err)
	}

	var mu sync.Mutex
	sawWrappedTerms := false
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
			if strings.Contains(text, "PICFETCHPROTECTED") {
				http.Error(response, "internal protection marker leaked into request", http.StatusBadRequest)
				return
			}
			if strings.Contains(text, "<keep>PICFETCH</keep>") && strings.Contains(text, "<keep>TOKEN</keep>") {
				mu.Lock()
				sawWrappedTerms = true
				mu.Unlock()
			}
			translations[index] = map[string]string{"text": "Deutsch: " + text}
		}
		if err := json.NewEncoder(response).Encode(map[string]any{"translations": translations}); err != nil {
			t.Errorf("write fake DeepL response: %v", err)
		}
	}))
	defer server.Close()

	cachePath := filepath.Join(t.TempDir(), "de.json")
	cmd := exec.Command("make", "translate",
		"SITE_SOURCE="+sourcePath,
		"SITE_TRANSLATIONS="+cachePath,
	)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("h", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make translate rejected collision-prone protected terms: %v\n%s", err, combined)
	}
	mu.Lock()
	gotWrappedTerms := sawWrappedTerms
	mu.Unlock()
	if !gotWrappedTerms {
		t.Fatal("translation request did not preserve both collision-prone terms with protection tags")
	}
	cache, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read translated cache: %v", err)
	}
	if strings.Contains(string(cache), "PICFETCHPROTECTED") {
		t.Fatal("translated cache contains an internal protection marker")
	}
	if !strings.Contains(string(cache), "PICFETCH TOKEN") {
		t.Fatal("translated cache did not preserve the collision-prone protected terms")
	}
}
