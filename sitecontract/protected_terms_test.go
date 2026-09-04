package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
