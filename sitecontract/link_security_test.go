package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckGeneratedRejectsUnsupportedAndMalformedURLs(t *testing.T) {
	tests := []struct {
		name       string
		old        string
		invalid    string
		diagnostic string
	}{
		{
			name:       "unsupported scheme",
			old:        "https://player.vimeo.com/api/player.js",
			invalid:    "javascript:alert(1)",
			diagnostic: "unsupported URL scheme",
		},
		{
			name:       "malformed local URL",
			old:        "{{.LocalPrefix}}manifest.json",
			invalid:    "%zz",
			diagnostic: "invalid URL",
		},
		{
			name:       "HTTPS authority without hostname",
			old:        "https://player.vimeo.com/api/player.js",
			invalid:    "https://user@:443/file",
			diagnostic: "invalid external URL",
		},
		{
			name:       "HTTP authority with empty port",
			old:        "https://player.vimeo.com/api/player.js",
			invalid:    "http://example.test:/file",
			diagnostic: "invalid external URL",
		},
		{
			name:       "HTTPS authority with nonnumeric port",
			old:        "https://player.vimeo.com/api/player.js",
			invalid:    "https://example.test:notaport/file",
			diagnostic: "invalid external URL",
		},
		{
			name:       "HTTP authority with zero port",
			old:        "https://player.vimeo.com/api/player.js",
			invalid:    "http://example.test:0/file",
			diagnostic: "invalid external URL",
		},
		{
			name:       "HTTPS authority with out-of-range port",
			old:        "https://player.vimeo.com/api/player.js",
			invalid:    "https://example.test:99999/file",
			diagnostic: "invalid external URL",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := repositoryRoot(t)
			cachePath := createControlledGermanCache(t, repo)
			templates := copyTemplateDirectory(t, repo)
			regularPath := filepath.Join(templates, "regular.html.tmpl")
			regular, err := os.ReadFile(regularPath)
			if err != nil {
				t.Fatalf("read copied regular template: %v", err)
			}
			broken := strings.Replace(string(regular), testCase.old, testCase.invalid, 1)
			if broken == string(regular) {
				t.Fatalf("test setup did not inject %q", testCase.invalid)
			}
			if err := os.WriteFile(regularPath, []byte(broken), 0o600); err != nil {
				t.Fatalf("write invalid regular template: %v", err)
			}

			output := t.TempDir()
			build := exec.Command("make", "build", "SITE_TEMPLATES="+templates, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
			build.Dir = repo
			if combined, err := build.CombinedOutput(); err != nil {
				t.Fatalf("prepare generated site with invalid URL: %v\n%s", err, combined)
			}
			writeStaticSiteFiles(t, output)

			check := exec.Command("make", "check-generated", "SITE_TEMPLATES="+templates, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
			check.Dir = repo
			combined, err := check.CombinedOutput()
			if err == nil {
				t.Fatalf("check-generated accepted URL %q", testCase.invalid)
			}
			if !strings.Contains(string(combined), testCase.diagnostic) || !strings.Contains(string(combined), testCase.invalid) {
				t.Fatalf("invalid-URL diagnostic is not actionable:\n%s", combined)
			}
		})
	}
}
