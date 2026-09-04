package sitecontract_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeBuildRejectsUnsafeHTMLInEnglishMarkdown(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	unsafeSource := strings.Replace(string(source),
		"Played back frame by frame at their encoded speed,",
		`<img src=x onerror="alert('unsafe')"> Played back frame by frame at their encoded speed,`,
		1,
	)
	if unsafeSource == string(source) {
		t.Fatal("test setup did not add unsafe English HTML")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(unsafeSource), 0o600); err != nil {
		t.Fatalf("write unsafe source: %v", err)
	}

	cmd := exec.Command("make", "build",
		"SITE_SOURCE="+sourcePath,
		"SITE_OUTPUT_DIR="+t.TempDir(),
		"SITE_LOCALES=en",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make build accepted an event handler in English Markdown")
	}
	if !strings.Contains(string(combined), "features.animated-gifs.body") || !strings.Contains(string(combined), "unsafe HTML") {
		t.Fatalf("unsafe-English-HTML diagnostic is not actionable:\n%s", combined)
	}
}

func TestMakeBuildRejectsUnsafeHTMLInGermanCache(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	type entry struct {
		SourceHash  string `json:"source_hash"`
		RequestHash string `json:"request_hash"`
		Format      string `json:"format"`
		Text        string `json:"text"`
	}
	var cache struct {
		Version int              `json:"version"`
		Locale  string           `json:"locale"`
		Entries map[string]entry `json:"entries"`
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read controlled cache: %v", err)
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("parse controlled cache: %v", err)
	}
	malicious := cache.Entries["features.animated-gifs.body"]
	malicious.Text += `<img src=x onerror="alert('unsafe')">`
	cache.Entries["features.animated-gifs.body"] = malicious
	data, err = json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("encode unsafe cache: %v", err)
	}
	if err := os.WriteFile(cachePath, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write unsafe cache: %v", err)
	}

	cmd := exec.Command("make", "build",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+t.TempDir(),
		"SITE_LOCALES=de",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make build accepted an event handler in cached German HTML")
	}
	if !strings.Contains(string(combined), "features.animated-gifs.body") || !strings.Contains(string(combined), "unsafe HTML") {
		t.Fatalf("unsafe-German-HTML diagnostic is not actionable:\n%s", combined)
	}
}

func TestMakeBuildCanonicalizesStructurallyEquivalentParserRepairedGermanHTML(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	type entry struct {
		SourceHash  string `json:"source_hash"`
		RequestHash string `json:"request_hash"`
		Format      string `json:"format"`
		Text        string `json:"text"`
	}
	var cache struct {
		Version int              `json:"version"`
		Locale  string           `json:"locale"`
		Entries map[string]entry `json:"entries"`
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read controlled cache: %v", err)
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("parse controlled cache: %v", err)
	}
	repaired := cache.Entries["hero.tagline"]
	repaired.Text = `<p>Repaired text`
	cache.Entries["hero.tagline"] = repaired
	data, err = json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("encode repaired cache: %v", err)
	}
	if err := os.WriteFile(cachePath, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write repaired cache: %v", err)
	}

	output := t.TempDir()
	cmd := exec.Command("make", "build",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=de",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build rejected repairable cached HTML: %v\n%s", err, combined)
	}
	page, err := os.ReadFile(filepath.Join(output, "de", "index.html"))
	if err != nil {
		t.Fatalf("read generated German page: %v", err)
	}
	html := string(page)
	if !strings.Contains(html, `<div class="tagline"><p>Repaired text</p></div>`) {
		t.Fatalf("generated page did not contain canonical repaired HTML:\n%s", html)
	}
	if strings.Contains(html, `<p>Repaired text</div>`) {
		t.Fatal("generated page emitted the unvalidated cache bytes")
	}
}
