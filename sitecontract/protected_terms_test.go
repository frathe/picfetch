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

	cmd := exec.Command("make", "translate",
		"SITE_SOURCE="+sourcePath,
		"SITE_TRANSLATIONS="+filepath.Join(t.TempDir(), "de.json"),
	)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("h", 24), "DEEPL_API_URL="+server.URL)
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make translate rejected unchanged protected term AT&T: %v\n%s", err, combined)
	}
}
