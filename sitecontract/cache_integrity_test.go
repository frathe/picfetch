package sitecontract_test

import (
	"encoding/json"
	"os"
	"os/exec"
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
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := repositoryRoot(t)
			cachePath := createControlledGermanCache(t, repo)
			var cache struct {
				Version int    `json:"version"`
				Locale  string `json:"locale"`
				Entries map[string]struct {
					SourceHash string `json:"source_hash"`
					Format     string `json:"format"`
					Text       string `json:"text"`
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
