package sitecontract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestTranslateAndBuildPreserveIndentedCodeAtMarkdownSectionBoundaries(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	const original = "## Tagline {#hero.tagline}\n\nA small desktop app for quickly viewing and browsing images. Drop one or more onto the window, and step through the set with the keyboard.\n\n## Drop almost anything {#features.drop-anything.body}"
	const replacement = "## Tagline {#hero.tagline}\n\n    first line  \n    last line\t\n\n## Drop almost anything {#features.drop-anything.body}"
	modified := strings.Replace(string(source), original, replacement, 1)
	if modified == string(source) {
		t.Fatal("test setup did not put an indented code block at both section boundaries")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
		t.Fatalf("write website source with boundary code block: %v", err)
	}

	const requestedCode = "<pre><code><keep>first line  \nlast line\t\n</keep></code></pre>"
	var sawCode atomic.Bool
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
			if text == requestedCode {
				sawCode.Store(true)
			}
			translations[index] = map[string]string{"text": text}
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	cachePath := filepath.Join(t.TempDir(), "de.json")
	translate := exec.Command("make", "translate", "SITE_SOURCE="+sourcePath, "SITE_TRANSLATIONS="+cachePath)
	translate.Dir = repo
	translate.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	translate.Env = append(translate.Env, "DEEPL_API_KEY="+strings.Repeat("b", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := translate.CombinedOutput(); err != nil {
		t.Fatalf("make translate failed: %v\n%s", err, combined)
	}
	if !sawCode.Load() {
		t.Fatalf("make translate did not preserve boundary code whitespace in its request; want %q", requestedCode)
	}

	output := t.TempDir()
	build := exec.Command("make", "build",
		"SITE_SOURCE="+sourcePath,
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=de",
		"SITE_FORMATS=regular",
	)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}
	page, err := os.ReadFile(filepath.Join(output, "de", "index.html"))
	if err != nil {
		t.Fatalf("read generated German page: %v", err)
	}
	const renderedCode = "<pre><code>first line  \nlast line\t\n</code></pre>"
	if !strings.Contains(string(page), renderedCode) {
		t.Fatalf("make build did not preserve boundary code whitespace; want exact fragment %q", renderedCode)
	}
}

func TestInvalidSourceRejectsMalformedStructuredURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "empty hostname", url: "https://user@:443/file"},
		{name: "empty port", url: "https://example.test:/file"},
		{name: "nonnumeric port", url: "https://example.test:notaport/file"},
		{name: "zero port", url: "https://example.test:0/file"},
		{name: "out-of-range port", url: "https://example.test:99999/file"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := repositoryRoot(t)
			source, err := os.ReadFile(filepath.Join(repo, "website.md"))
			if err != nil {
				t.Fatalf("read website source: %v", err)
			}
			const current = "  open_graph_image: https://raw.githubusercontent.com/frathe/picfetch/main/assets/social_logo.jpg"
			modified := strings.Replace(string(source), current, "  open_graph_image: '"+test.url+"'", 1)
			if modified == string(source) {
				t.Fatal("test setup did not change metadata.open_graph_image")
			}
			sourcePath := filepath.Join(t.TempDir(), "website.md")
			if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
				t.Fatalf("write website source with malformed structured URL: %v", err)
			}

			command := exec.Command("make", "build",
				"SITE_SOURCE="+sourcePath,
				"SITE_OUTPUT_DIR="+t.TempDir(),
				"SITE_LOCALES=en",
				"SITE_FORMATS=regular",
			)
			command.Dir = repo
			combined, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("make build accepted malformed structured URL %q", test.url)
			}
			if !strings.Contains(string(combined), "metadata.open_graph_image") || !strings.Contains(string(combined), "URL") {
				t.Fatalf("malformed structured URL diagnostic is not actionable:\n%s", combined)
			}
		})
	}
}

func TestInvalidSourceRejectsMalformedMarkdownURLs(t *testing.T) {
	urls := []string{
		"https://user@:443/file",
		"https://example.test:/file",
		"https://example.test:99999/file",
	}

	for _, invalidURL := range urls {
		t.Run(invalidURL, func(t *testing.T) {
			repo := repositoryRoot(t)
			source, err := os.ReadFile(filepath.Join(repo, "website.md"))
			if err != nil {
				t.Fatalf("read website source: %v", err)
			}
			const current = "[build instructions](https://github.com/frathe/picfetch#building)"
			modified := strings.Replace(string(source), current, "[build instructions]("+invalidURL+")", 1)
			if modified == string(source) {
				t.Fatal("test setup did not change the Markdown link")
			}
			sourcePath := filepath.Join(t.TempDir(), "website.md")
			if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
				t.Fatalf("write website source with malformed Markdown URL: %v", err)
			}

			command := exec.Command("make", "build",
				"SITE_SOURCE="+sourcePath,
				"SITE_OUTPUT_DIR="+t.TempDir(),
				"SITE_LOCALES=en",
				"SITE_FORMATS=regular",
			)
			command.Dir = repo
			combined, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("make build accepted malformed Markdown URL %q", invalidURL)
			}
			if !strings.Contains(string(combined), "downloads.introduction") || !strings.Contains(string(combined), invalidURL) {
				t.Fatalf("malformed Markdown URL diagnostic is not actionable:\n%s", combined)
			}
		})
	}
}

func TestBuildPreservesTrailingWhitespaceInUnclosedFencedSection(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	const heading = "## Colophon {#footer.colophon}\n\n"
	sectionStart := strings.Index(string(source), heading)
	if sectionStart < 0 {
		t.Fatal("test setup could not find the final Markdown section")
	}
	modified := string(source[:sectionStart]) + heading + "```text\nlast visible line\n \t \n"
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
		t.Fatalf("write website source with unclosed fence: %v", err)
	}

	output := t.TempDir()
	command := exec.Command("make", "build",
		"SITE_SOURCE="+sourcePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=en",
		"SITE_FORMATS=regular",
	)
	command.Dir = repo
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("make build rejected an unclosed final fence: %v\n%s", err, combined)
	}
	page, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatalf("read generated English page: %v", err)
	}
	const rendered = "<pre><code class=\"language-text\">last visible line\n \t \n</code></pre>"
	if !strings.Contains(string(page), rendered) {
		t.Fatalf("make build dropped trailing whitespace from the unclosed fence; want exact fragment %q", rendered)
	}
}
