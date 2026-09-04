package sitecontract_test

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestBrowserCheckClosesServerWhenChromiumLaunchFails(t *testing.T) {
	repo := repositoryRoot(t)
	output := t.TempDir()
	script := `
const playwright = require('playwright-core');
playwright.chromium.launch = async () => { throw new Error('synthetic Chromium launch failure'); };
process.argv[2] = ` + strconv.Quote(output) + `;
require('./site/tools/test-language.cjs');
`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "-e", script)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatal("browser check hung after Chromium launch failed")
	}
	if err == nil {
		t.Fatal("browser check succeeded after Chromium launch failed")
	}
	if !strings.Contains(string(combined), "browser language behavior: FAIL: synthetic Chromium launch failure") {
		t.Fatalf("browser launch diagnostic is not actionable:\n%s", combined)
	}
}
