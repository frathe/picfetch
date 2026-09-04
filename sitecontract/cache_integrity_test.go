package sitecontract_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRejectsModifiedOpaqueValuesInReusedGermanCache(t *testing.T) {
	tests := []struct {
		name        string
		entryID     string
		original    string
		replacement string
		appendText  string
	}{
		{
			name:        "Markdown URL",
			entryID:     "downloads.introduction",
			original:    "https://github.com/frathe/picfetch#building",
			replacement: "https://evil.invalid",
		},
		{
			name:        "protected product name",
			entryID:     "metadata.title",
			original:    "PicFetch",
			replacement: "OtherProduct",
		},
		{
			name:       "duplicated protected term",
			entryID:    "metadata.title",
			appendText: " PicFetch",
		},
		{
			name:        "protected keyboard action",
			entryID:     "downloads.warning.body",
			original:    "Control-click",
			replacement: "Strg-Klick",
		},
		{
			name:        "protected architecture family",
			entryID:     "downloads.macos.arm64",
			original:    "Apple Silicon",
			replacement: "Apple-Chip",
		},
		{
			name:        "protected processor vendor",
			entryID:     "downloads.macos.x86-64",
			original:    "Intel",
			replacement: "Prozessor",
		},
		{
			name:        "protected architecture identifier",
			entryID:     "downloads.macos.arm64",
			original:    "arm64",
			replacement: "armv8",
		},
		{
			name:        "protected camera identifier",
			entryID:     "features.exif-aware.body",
			original:    "ISO",
			replacement: "Lichtempfindlichkeit",
		},
		{
			name:        "protected translation provider",
			entryID:     "labels.deepl-disclosure",
			original:    "DeepL",
			replacement: "Übersetzungsdienst",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := repositoryRoot(t)
			cachePath := createControlledGermanCache(t, repo)
			var cache struct {
				Version int    `json:"version"`
				Locale  string `json:"locale"`
				Entries map[string]struct {
					SourceHash  string `json:"source_hash"`
					RequestHash string `json:"request_hash"`
					Format      string `json:"format"`
					Text        string `json:"text"`
				} `json:"entries"`
			}
			data, err := os.ReadFile(cachePath)
			if err != nil {
				t.Fatalf("read controlled cache: %v", err)
			}
			if err := json.Unmarshal(data, &cache); err != nil {
				t.Fatalf("parse controlled cache: %v", err)
			}
			entry, ok := cache.Entries[testCase.entryID]
			if !ok {
				t.Fatalf("controlled cache has no %s entry", testCase.entryID)
			}
			originalText := entry.Text
			if testCase.appendText != "" {
				entry.Text += testCase.appendText
			} else {
				entry.Text = strings.Replace(entry.Text, testCase.original, testCase.replacement, 1)
			}
			if entry.Text == originalText {
				t.Fatalf("test setup did not modify %s", testCase.entryID)
			}
			cache.Entries[testCase.entryID] = entry
			data, err = json.MarshalIndent(cache, "", "  ")
			if err != nil {
				t.Fatalf("encode modified cache: %v", err)
			}
			if err := os.WriteFile(cachePath, append(data, '\n'), 0o600); err != nil {
				t.Fatalf("write modified cache: %v", err)
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
				t.Fatalf("make build accepted modified opaque content in %s", testCase.entryID)
			}
			if !strings.Contains(string(combined), testCase.entryID) || !strings.Contains(string(combined), "protected") {
				t.Fatalf("opaque-cache diagnostic is not actionable:\n%s", combined)
			}
		})
	}
}

func TestBuildRejectsSwappedMarkdownLinksInReusedGermanCache(t *testing.T) {
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
	entry, ok := cache.Entries["footer.colophon"]
	if !ok {
		t.Fatal("controlled cache has no footer.colophon entry")
	}
	const licenseURL = "https://github.com/frathe/picfetch/blob/main/LICENSE"
	const fyneURL = "https://fyne.io/"
	if strings.Count(entry.Text, licenseURL) != 1 || strings.Count(entry.Text, fyneURL) != 1 {
		t.Fatalf("test setup expected one LICENSE and one Fyne link in footer.colophon: %q", entry.Text)
	}
	entry.Text = strings.NewReplacer(licenseURL, fyneURL, fyneURL, licenseURL).Replace(entry.Text)
	cache.Entries["footer.colophon"] = entry
	data, err = json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("encode modified cache: %v", err)
	}
	if err := os.WriteFile(cachePath, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write modified cache: %v", err)
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
		t.Fatal("make build accepted Markdown links whose destinations were swapped")
	}
	if !strings.Contains(string(combined), "footer.colophon") || !strings.Contains(string(combined), "protected URL") {
		t.Fatalf("swapped-link diagnostic is not actionable:\n%s", combined)
	}
}

func TestBuildRejectsModifiedMultilineFencedCodeInGermanCache(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	needle := "A small desktop app for quickly viewing and browsing images. Drop one or more onto the window, and step through the set with the keyboard.\n\n## Drop almost anything {#features.drop-anything.body}"
	replacement := "A small desktop app for quickly viewing and browsing images. Drop one or more onto the window, and step through the set with the keyboard. Run `PicFetch --help` for details.\n\n```shell\nPicFetch --safe\nPicFetch --readonly\n```\n\n## Drop almost anything {#features.drop-anything.body}"
	changed := strings.Replace(string(source), needle, replacement, 1)
	if changed == string(source) {
		t.Fatal("test setup did not add multiline fenced code")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(changed), 0o600); err != nil {
		t.Fatalf("write source with multiline fenced code: %v", err)
	}
	cachePath := createControlledGermanCacheForSource(t, repo, sourcePath)

	var cache struct {
		Version int    `json:"version"`
		Locale  string `json:"locale"`
		Entries map[string]struct {
			SourceHash  string `json:"source_hash"`
			RequestHash string `json:"request_hash"`
			Format      string `json:"format"`
			Text        string `json:"text"`
		} `json:"entries"`
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read controlled cache: %v", err)
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("parse controlled cache: %v", err)
	}
	entry := cache.Entries["hero.tagline"]
	entry.Text = strings.Replace(entry.Text, "PicFetch --readonly", "PicFetch --write", 1)
	if !strings.Contains(entry.Text, "PicFetch --write") {
		t.Fatal("test setup did not modify fenced code in the cache")
	}
	cache.Entries["hero.tagline"] = entry
	data, err = json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("encode modified cache: %v", err)
	}
	if err := os.WriteFile(cachePath, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write modified cache: %v", err)
	}

	cmd := exec.Command("make", "build",
		"SITE_SOURCE="+sourcePath,
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+t.TempDir(),
		"SITE_LOCALES=de",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make build accepted modified multiline fenced code")
	}
	if !strings.Contains(string(combined), "protected code") || !strings.Contains(string(combined), "hero.tagline") {
		t.Fatalf("multiline-code diagnostic is not actionable:\n%s", combined)
	}
}

func TestBuildRejectsModifiedTailOfNestedKeyboardMarkup(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	needle := "A small desktop app for quickly viewing and browsing images."
	withNestedKeyboard := strings.Replace(string(source), needle,
		needle+` Press <kbd><code>Ctrl</code>+C</kbd> to continue.`, 1)
	if withNestedKeyboard == string(source) {
		t.Fatal("test setup did not add nested keyboard markup")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(withNestedKeyboard), 0o600); err != nil {
		t.Fatalf("write source with nested keyboard markup: %v", err)
	}
	cachePath := createControlledGermanCacheForSource(t, repo, sourcePath)

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
	entry := cache.Entries["hero.tagline"]
	entry.Text = strings.Replace(entry.Text, "</code>+C</kbd>", "</code>+V</kbd>", 1)
	if !strings.Contains(entry.Text, "</code>+V</kbd>") {
		t.Fatal("test setup did not modify the tail of the nested keyboard markup")
	}
	cache.Entries["hero.tagline"] = entry
	data, err = json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("encode modified cache: %v", err)
	}
	if err := os.WriteFile(cachePath, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write modified cache: %v", err)
	}

	cmd := exec.Command("make", "build",
		"SITE_SOURCE="+sourcePath,
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+t.TempDir(),
		"SITE_LOCALES=de",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make build accepted a changed tail in nested keyboard markup")
	}
	if !strings.Contains(string(combined), "hero.tagline") || !strings.Contains(string(combined), "protected code or keyboard content") {
		t.Fatalf("nested-keyboard diagnostic is not actionable:\n%s", combined)
	}
}
