package sitecontract_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	xhtml "golang.org/x/net/html"
)

func TestMakeBuildGeneratesGermanRegularFromCurrentCache(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()

	cmd := exec.Command("make", "build",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=en,de",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build failed with a current German cache: %v\n%s", err, combined)
	}

	german, err := os.ReadFile(filepath.Join(output, "de", "index.html"))
	if err != nil {
		t.Fatalf("read German regular page: %v", err)
	}
	html := string(german)
	for _, want := range []string{
		`<html lang="de">`,
		`<link rel="canonical" href="https://frathe.github.io/picfetch/de/">`,
		`<link rel="amphtml" href="https://frathe.github.io/picfetch/de/amp/">`,
		`<meta property="og:locale" content="de_DE">`,
		`class="translation-disclosure"`,
		`Deutsch: This page was translated with DeepL and has not been edited.`,
		`🇬🇧`,
		`🇩🇪`,
		`https://player.vimeo.com/video/1220283616?badge=0&amp;autopause=0`,
		`picfetch-windows-arm64.zip`,
		`xattr -cr &quot;/path/to/PicFetch.app&quot;`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("German regular page does not contain %q", want)
		}
	}
	if strings.Contains(html, "window.location.replace") {
		t.Error("explicit German route contains automatic language redirection")
	}

	english, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatalf("read English regular page: %v", err)
	}
	if !strings.Contains(string(english), "window.location.replace('./de/')") {
		t.Error("English root does not contain first-visit German detection")
	}
}

func TestLanguageIndicatorsComeFromAuthoredSource(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	custom := string(source)
	if strings.Contains(custom, "language_flags:\n") {
		custom = strings.Replace(custom, "english: '🇬🇧'", "english: '[EN]'", 1)
		custom = strings.Replace(custom, "german: '🇩🇪'", "german: '[DE]'", 1)
	} else {
		custom = strings.Replace(custom, "labels:\n", "language_flags:\n  english: '[EN]'\n  german: '[DE]'\nlabels:\n", 1)
	}
	if custom == string(source) {
		t.Fatal("test setup did not add custom language indicators")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(custom), 0o600); err != nil {
		t.Fatalf("write website source with custom indicators: %v", err)
	}
	cachePath := createControlledGermanCacheForSource(t, repo, sourcePath)
	output := t.TempDir()

	cmd := exec.Command("make", "build",
		"SITE_SOURCE="+sourcePath,
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=en,de",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build with authored language indicators: %v\n%s", err, combined)
	}
	for _, pagePath := range []string{"index.html", filepath.Join("de", "index.html")} {
		page, err := os.ReadFile(filepath.Join(output, pagePath))
		if err != nil {
			t.Fatalf("read %s: %v", pagePath, err)
		}
		if !strings.Contains(string(page), "[EN]") || !strings.Contains(string(page), "[DE]") {
			t.Errorf("%s does not contain both authored language indicators", pagePath)
		}
	}
}

func createControlledGermanCache(t *testing.T, repo string) string {
	t.Helper()
	return createControlledGermanCacheForSource(t, repo, "website.md")
}

func createControlledGermanCacheForSource(t *testing.T, repo, sourcePath string) string {
	t.Helper()
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
			translations[index] = map[string]string{"text": prefixPlainTextTranslation("Deutsch: ", text)}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	cachePath := filepath.Join(t.TempDir(), "de.json")
	cmd := exec.Command("make", "translate", "SITE_SOURCE="+sourcePath, "SITE_TRANSLATIONS="+cachePath)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("g", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create controlled German cache: %v\n%s", err, combined)
	}
	return cachePath
}

// prefixPlainTextTranslation gives controlled text units an observable fake
// translation without changing the root or text-slot structure of rendered
// Markdown HTML. Plain-text requests may contain <keep> protection elements,
// so the first significant token—not merely the presence of markup—identifies
// the request shape.
func prefixPlainTextTranslation(prefix, source string) string {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(source))
	for {
		switch tokenizer.Next() {
		case xhtml.ErrorToken:
			return prefix + source
		case xhtml.TextToken:
			if strings.TrimSpace(tokenizer.Token().Data) != "" {
				return prefix + source
			}
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			if tokenizer.Token().Data == "keep" {
				return prefix + source
			}
			return source
		}
	}
}
