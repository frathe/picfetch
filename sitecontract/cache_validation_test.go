package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
