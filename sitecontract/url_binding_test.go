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

func TestTranslateAllowsMarkdownLinksToReorderWithTheirDestinations(t *testing.T) {
	repo := repositoryRoot(t)
	const licenseURL = "https://github.com/frathe/picfetch/blob/main/LICENSE"
	const fyneURL = "https://fyne.io/"
	var reordered atomic.Bool
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
			if strings.Contains(text, licenseURL) && strings.Contains(text, fyneURL) {
				licenseAnchor := htmlAnchorWithHref(t, text, licenseURL)
				fyneAnchor := htmlAnchorWithHref(t, text, fyneURL)
				translated = "<p>Mit " + fyneAnchor + " gebaut; <keep>PicFetch</keep> steht unter der " + licenseAnchor + ".</p>"
				reordered.Store(true)
			}
			translations[index] = map[string]string{"text": translated}
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	cachePath := filepath.Join(t.TempDir(), "de.json")
	translate := exec.Command("make", "translate", "SITE_TRANSLATIONS="+cachePath)
	translate.Dir = repo
	translate.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	translate.Env = append(translate.Env, "DEEPL_API_KEY="+strings.Repeat("r", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := translate.CombinedOutput(); err != nil {
		t.Fatalf("make translate rejected reordered Markdown links: %v\n%s", err, combined)
	}
	if !reordered.Load() {
		t.Fatal("fake translation did not reorder the linked footer content")
	}

	output := t.TempDir()
	build := exec.Command("make", "build",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=de",
		"SITE_FORMATS=regular",
	)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("make build rejected cached reordered Markdown links: %v\n%s", err, combined)
	}
	page, err := os.ReadFile(filepath.Join(output, "de", "index.html"))
	if err != nil {
		t.Fatalf("read generated German page: %v", err)
	}
	generated := string(page)
	fyneOffset := strings.Index(generated, `href="`+fyneURL+`"`)
	licenseOffset := strings.LastIndex(generated, `href="`+licenseURL+`"`)
	if fyneOffset < 0 || licenseOffset < 0 || fyneOffset >= licenseOffset {
		t.Fatalf("generated page did not retain reordered anchors with their destinations (Fyne=%d LICENSE=%d)", fyneOffset, licenseOffset)
	}
	if strings.Contains(generated, "sitegen-link-") {
		t.Fatal("generated page leaked an internal URL-binding marker")
	}
}

func TestTranslateRejectsSwappedMarkdownLinkDestinations(t *testing.T) {
	repo := repositoryRoot(t)
	const licenseURL = "https://github.com/frathe/picfetch/blob/main/LICENSE"
	const fyneURL = "https://fyne.io/"
	var swapped atomic.Bool
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
			if strings.Contains(text, licenseURL) && strings.Contains(text, fyneURL) {
				translated = strings.NewReplacer(licenseURL, fyneURL, fyneURL, licenseURL).Replace(text)
				swapped.Store(true)
			}
			translations[index] = map[string]string{"text": translated}
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	cachePath := filepath.Join(t.TempDir(), "de.json")
	translate := exec.Command("make", "translate", "SITE_TRANSLATIONS="+cachePath)
	translate.Dir = repo
	translate.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	translate.Env = append(translate.Env, "DEEPL_API_KEY="+strings.Repeat("s", 24), "DEEPL_API_URL="+server.URL)
	combined, err := translate.CombinedOutput()
	if err == nil {
		t.Fatal("make translate accepted swapped Markdown link destinations")
	}
	if !swapped.Load() {
		t.Fatal("fake translation did not swap the linked footer destinations")
	}
	if !strings.Contains(string(combined), "footer.colophon") || !strings.Contains(string(combined), "protected URL") {
		t.Fatalf("swapped-link diagnostic is not actionable:\n%s", combined)
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("failed translation wrote a partial cache: %v", err)
	}
}

func TestTranslateMigratesLegacyMarkdownURLBindingsOffline(t *testing.T) {
	repo := repositoryRoot(t)
	legacy, err := os.ReadFile(filepath.Join(repo, "site", "translations", "de.json"))
	if err != nil {
		t.Fatalf("read checked-in German cache: %v", err)
	}
	if strings.Contains(string(legacy), "sitegen-link-binding-") {
		t.Fatal("checked-in German cache is no longer a legacy URL-binding fixture")
	}
	cachePath := filepath.Join(t.TempDir(), "de.json")
	if err := os.WriteFile(cachePath, legacy, 0o600); err != nil {
		t.Fatalf("copy checked-in German cache: %v", err)
	}
	runTranslate := func() []byte {
		t.Helper()
		translate := exec.Command("make", "translate",
			"SITE_TRANSLATIONS="+cachePath,
			"DEEPL_ENV_FILE="+filepath.Join(t.TempDir(), "missing.env"),
		)
		translate.Dir = repo
		translate.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
		if combined, err := translate.CombinedOutput(); err != nil {
			t.Fatalf("migrate legacy German cache without credentials: %v\n%s", err, combined)
		}
		migrated, err := os.ReadFile(cachePath)
		if err != nil {
			t.Fatalf("read migrated German cache: %v", err)
		}
		return migrated
	}

	first := runTranslate()
	if !strings.Contains(string(first), "sitegen-link-binding-") {
		t.Fatal("offline cache migration did not persist URL-binding markers")
	}
	if !strings.Contains(string(first), "PicFetch ist kostenlos und Open Source") {
		t.Fatal("offline cache migration did not preserve the checked-in German prose")
	}
	second := runTranslate()
	if string(second) != string(first) {
		t.Fatal("URL-binding cache migration is not deterministic")
	}

	output := t.TempDir()
	build := exec.Command("make", "build",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=de",
		"SITE_FORMATS=regular",
	)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build with migrated German cache: %v\n%s", err, combined)
	}
	page, err := os.ReadFile(filepath.Join(output, "de", "index.html"))
	if err != nil {
		t.Fatalf("read German page built from migrated cache: %v", err)
	}
	if strings.Contains(string(page), "sitegen-link-") {
		t.Fatal("German page leaked an internal URL-binding marker")
	}
}

func TestBuildRejectsDuplicateMarkdownURLAttributes(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	type cacheEntry struct {
		SourceHash  string `json:"source_hash"`
		RequestHash string `json:"request_hash"`
		Format      string `json:"format"`
		Text        string `json:"text"`
	}
	var cache struct {
		Version int                   `json:"version"`
		Locale  string                `json:"locale"`
		Entries map[string]cacheEntry `json:"entries"`
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read controlled cache: %v", err)
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("parse controlled cache: %v", err)
	}
	entry := cache.Entries["footer.colophon"]
	const licenseAttribute = `href="https://github.com/frathe/picfetch/blob/main/LICENSE"`
	entry.Text = strings.Replace(entry.Text, licenseAttribute, licenseAttribute+` href="https://fyne.io/"`, 1)
	if strings.Count(entry.Text, licenseAttribute) != 1 || !strings.Contains(entry.Text, licenseAttribute+` href="https://fyne.io/"`) {
		t.Fatalf("test setup did not add duplicate href attributes: %q", entry.Text)
	}
	cache.Entries["footer.colophon"] = entry
	data, err = json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("encode controlled cache: %v", err)
	}
	if err := os.WriteFile(cachePath, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write controlled cache: %v", err)
	}

	build := exec.Command("make", "build",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+t.TempDir(),
		"SITE_LOCALES=de",
		"SITE_FORMATS=regular",
	)
	build.Dir = repo
	combined, err := build.CombinedOutput()
	if err == nil {
		t.Fatal("make build accepted duplicate Markdown href attributes")
	}
	if !strings.Contains(string(combined), "footer.colophon") || !strings.Contains(string(combined), "duplicate attribute href") {
		t.Fatalf("duplicate-attribute diagnostic is not actionable:\n%s", combined)
	}
}

func htmlAnchorWithHref(t *testing.T, value, href string) string {
	t.Helper()
	start := strings.Index(value, `<a href="`+href+`"`)
	if start < 0 {
		t.Fatalf("translation request has no anchor for %q: %q", href, value)
	}
	end := strings.Index(value[start:], "</a>")
	if end < 0 {
		t.Fatalf("translation request has no closing anchor for %q: %q", href, value)
	}
	return value[start : start+end+len("</a>")]
}
