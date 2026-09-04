package sitecontract_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestLanguageSelectionInBrowser(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()

	build := exec.Command("make", "build",
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
		"SITE_LOCALES=en,de",
		"SITE_FORMATS=regular,amp",
	)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build browser-test site: %v\n%s", err, combined)
	}

	browser := exec.Command("make", "test-browser", "SITE_OUTPUT_DIR="+output)
	browser.Dir = repo
	combined, err := browser.CombinedOutput()
	if err != nil {
		t.Fatalf("browser language behavior failed: %v\n%s", err, combined)
	}
	if want := "browser language behavior: PASS"; !strings.Contains(string(combined), want) {
		t.Fatalf("browser test did not report %q:\n%s", want, combined)
	}
}
