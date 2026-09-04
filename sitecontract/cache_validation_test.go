package sitecontract_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckGeneratedRejectsObsoleteGermanCacheEntry(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read controlled cache: %v", err)
	}
	var cache map[string]any
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("parse controlled cache: %v", err)
	}
	entries, ok := cache["entries"].(map[string]any)
	if !ok {
		t.Fatal("controlled cache has no entries object")
	}
	entries["obsolete.unit"] = map[string]any{
		"source_hash":  strings.Repeat("0", 64),
		"request_hash": strings.Repeat("1", 64),
		"format":       "text",
		"text":         "Obsolete translation",
	}
	data, err = json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatalf("encode cache with obsolete entry: %v", err)
	}
	if err := os.WriteFile(cachePath, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write cache with obsolete entry: %v", err)
	}

	output := t.TempDir()
	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build with otherwise-current cache: %v\n%s", err, combined)
	}
	writeStaticSiteFiles(t, output)
	check := exec.Command("make", "check-generated", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted an obsolete German cache entry")
	}
	if !strings.Contains(string(combined), "obsolete German translation") || !strings.Contains(string(combined), "obsolete.unit") {
		t.Fatalf("obsolete-cache diagnostic is not actionable:\n%s", combined)
	}
}

func TestBuildRejectsTrailingDataInGermanCache(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	cache, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read controlled cache: %v", err)
	}
	cache = append(cache, []byte("{}\n")...)
	if err := os.WriteFile(cachePath, cache, 0o600); err != nil {
		t.Fatalf("append a second JSON document to cache: %v", err)
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
		t.Fatal("make build accepted trailing data in the German cache")
	}
	if !strings.Contains(string(combined), "trailing data") {
		t.Fatalf("trailing-cache diagnostic is not actionable:\n%s", combined)
	}
	if strings.Contains(string(combined), filepath.Base(cachePath)+": no such file") {
		t.Fatalf("test fixture disappeared unexpectedly:\n%s", combined)
	}
}
