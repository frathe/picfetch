package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeBuildPreservesTrailingWhitespaceInFencedCode(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	marker := "\n\n## Drop almost anything {#features.drop-anything.body}"
	fenced := "\n\n```text\nfirst line  \nsecond line\t\n \t \n```"
	modified := strings.Replace(string(source), marker, fenced+marker, 1)
	if modified == string(source) {
		t.Fatal("test setup did not add the fenced code block")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(modified), 0o600); err != nil {
		t.Fatalf("write website source with fenced code: %v", err)
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
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}

	page, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatalf("read generated page: %v", err)
	}
	want := "<pre><code class=\"language-text\">first line  \nsecond line\t\n \t \n</code></pre>"
	if !strings.Contains(string(page), want) {
		t.Fatalf("generated page changed preformatted trailing whitespace; want exact fragment %q", want)
	}
}
