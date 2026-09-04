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

func TestTranslateRejectsChangedMarkdownStructure(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	const original = "A small desktop app for quickly viewing and browsing images. Drop one or more onto the window, and step through the set with the keyboard."
	const list = "- First\n- Second"
	modified := strings.Replace(string(source), original, list, 1)
	if modified == string(source) {
		t.Fatal("test setup did not add Markdown list structure")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
		t.Fatalf("write website source: %v", err)
	}

	var changed atomic.Bool
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
			if strings.Contains(text, "<ul>") && strings.Contains(text, "<li>First</li>") {
				translated = "<p>Erste</p><p>Zweite</p>"
				changed.Store(true)
			}
			translations[index] = map[string]string{"text": translated}
		}
		_ = json.NewEncoder(response).Encode(map[string]any{"translations": translations})
	}))
	defer server.Close()

	cachePath := filepath.Join(t.TempDir(), "de.json")
	translate := exec.Command("make", "translate", "SITE_SOURCE="+sourcePath, "SITE_TRANSLATIONS="+cachePath)
	translate.Dir = repo
	translate.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	translate.Env = append(translate.Env, "DEEPL_API_KEY="+strings.Repeat("t", 24), "DEEPL_API_URL="+server.URL)
	combined, err := translate.CombinedOutput()
	if err == nil {
		t.Fatal("make translate accepted changed Markdown structure")
	}
	if !changed.Load() {
		t.Fatal("fake translation did not change the Markdown list structure")
	}
	if !strings.Contains(string(combined), "hero.tagline") || !strings.Contains(string(combined), "Markdown structure") {
		t.Fatalf("structure-change diagnostic is not actionable:\n%s", combined)
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("failed translation wrote a partial cache: %v", err)
	}
}
