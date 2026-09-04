package sitecontract_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeTranslateTreatsLiteralHTMLLikeTextAsPlainText(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	const literal = "Press <code>Esc</code> or use <a href='https://example.test'>help</a>"
	modified := strings.Replace(string(source), "    text: Close\n", `    text: "`+literal+`"`+"\n", 1)
	if modified == string(source) || !strings.Contains(modified, literal) {
		t.Fatal("test setup did not add literal HTML-like text")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
		t.Fatalf("write modified source: %v", err)
	}

	server := echoingDeepLServer(t)
	defer server.Close()
	cachePath := filepath.Join(t.TempDir(), "de.json")
	cmd := exec.Command("make", "translate", "SITE_SOURCE="+sourcePath, "SITE_TRANSLATIONS="+cachePath)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("l", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make translate treated literal plain text as Markdown markup: %v\n%s", err, combined)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read translated cache: %v", err)
	}
	var cache struct {
		Entries map[string]struct {
			Text string `json:"text"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("parse translated cache: %v", err)
	}
	if got, want := cache.Entries["labels.lightbox-close"].Text, "Deutsch: "+literal; got != want {
		t.Fatalf("literal plain text changed during translation: got %q, want %q", got, want)
	}
}
