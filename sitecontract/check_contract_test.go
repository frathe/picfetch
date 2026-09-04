package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckGeneratedRejectsIncompleteLocaleFormatMatrix(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()

	build := exec.Command(
		"make",
		"build",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=en",
		"SITE_FORMATS=regular",
	)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare complete generated site: %v\n%s", err, combined)
	}
	writeStaticSiteFiles(t, output)

	check := exec.Command(
		"make",
		"check-generated",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=en",
		"SITE_FORMATS=regular",
	)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted an incomplete locale/format matrix")
	}
	if !strings.Contains(string(combined), "complete locale/format matrix") {
		t.Fatalf("incomplete-matrix diagnostic is not actionable:\n%s", combined)
	}
}

func TestCheckGeneratedRejectsUnexpectedGeneratedRoute(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()

	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare complete generated site: %v\n%s", err, combined)
	}
	writeStaticSiteFiles(t, output)

	staleRoute := filepath.Join(output, "fr", "index.html")
	if err := os.MkdirAll(filepath.Dir(staleRoute), 0o755); err != nil {
		t.Fatalf("create stale route directory: %v", err)
	}
	if err := os.WriteFile(staleRoute, []byte("<!doctype html><html lang=\"fr\"><body>stale</body></html>\n"), 0o600); err != nil {
		t.Fatalf("write stale route: %v", err)
	}

	check := exec.Command("make", "check-generated", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted an unexpected generated route")
	}
	if !strings.Contains(string(combined), "unexpected generated route: fr/index.html") {
		t.Fatalf("unexpected-route diagnostic is not actionable:\n%s", combined)
	}
}
