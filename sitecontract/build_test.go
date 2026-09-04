package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMakeBuildGeneratesEnglishRegularPage(t *testing.T) {
	repo := repositoryRoot(t)
	output := t.TempDir()

	cmd := exec.Command("make", "build", "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en", "SITE_FORMATS=regular")
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}

	page, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatalf("read generated English regular page: %v", err)
	}
	html := string(page)
	for _, want := range []string{
		`<html lang="en">`,
		`<title>PicFetch — a small, fast image viewer for macOS, Windows and Linux</title>`,
		`1220283616`,
		`href="#downloads"`,
		`picfetch-linux-arm64.tar.gz`,
		`class="lightbox"`,
		`prefers-color-scheme: dark`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("generated English regular page does not contain %q", want)
		}
	}
}

func TestInvalidSourceReportsAffectedField(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	broken := strings.Replace(string(source), "id: metadata.description", "id: ''", 1)
	if broken == string(source) {
		t.Fatal("test setup did not remove metadata.description identity")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write invalid website source: %v", err)
	}

	cmd := exec.Command("make", "build", "SITE_SOURCE="+sourcePath, "SITE_OUTPUT_DIR="+t.TempDir(), "SITE_LOCALES=en", "SITE_FORMATS=regular")
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("make build succeeded with a missing metadata.description identity")
	}
	if !strings.Contains(string(combined), "metadata.description") {
		t.Fatalf("diagnostic does not identify metadata.description:\n%s", combined)
	}
}

func TestInvalidSourceRejectsTextAndMarkdownIdentityCollision(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	broken := strings.Replace(string(source), "id: metadata.title", "id: hero.tagline", 1)
	if broken == string(source) {
		t.Fatal("test setup did not create a cross-namespace translation ID collision")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write invalid website source: %v", err)
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
		t.Fatal("make build accepted a shared text/Markdown translation identity")
	}
	if !strings.Contains(string(combined), `duplicate translatable identity "hero.tagline"`) || !strings.Contains(string(combined), "metadata.title") {
		t.Fatalf("cross-namespace identity diagnostic is not actionable:\n%s", combined)
	}
}

func TestMakeBuildGeneratesEnglishAMPFromSharedSource(t *testing.T) {
	repo := repositoryRoot(t)
	output := t.TempDir()

	cmd := exec.Command("make", "build", "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en", "SITE_FORMATS=regular,amp")
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}

	page, err := os.ReadFile(filepath.Join(output, "amp", "index.html"))
	if err != nil {
		t.Fatalf("read generated English AMP page: %v", err)
	}
	html := string(page)
	for _, want := range []string{
		`<html amp lang="en">`,
		`<link rel="canonical" href="https://frathe.github.io/picfetch/">`,
		`https://cdn.ampproject.org/v0.js`,
		`custom-element="amp-vimeo"`,
		`custom-element="amp-lightbox-gallery"`,
		`<amp-vimeo data-videoid="1220283616"`,
		`<amp-img src="https://raw.githubusercontent.com/frathe/picfetch/main/assets/screens/main_screen.png" width="520" height="372" layout="responsive" lightbox="screenshots"`,
		`picfetch-linux-arm64.tar.gz`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("generated English AMP page does not contain %q", want)
		}
	}
	if strings.Contains(html, "<iframe") {
		t.Error("generated English AMP page contains a regular iframe")
	}
}

func TestMakeValidateAMPRunsPinnedValidatorOffline(t *testing.T) {
	repo := repositoryRoot(t)
	output := t.TempDir()

	build := exec.Command("make", "build", "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en", "SITE_FORMATS=amp")
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}

	validate := exec.Command("make", "validate-amp", "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en")
	validate.Dir = repo
	validate.Env = append(os.Environ(), "npm_config_offline=true")
	combined, err := validate.CombinedOutput()
	if err != nil {
		t.Fatalf("offline make validate-amp failed: %v\n%s", err, combined)
	}
	if !strings.Contains(string(combined), filepath.Join(output, "amp", "index.html")+": PASS") {
		t.Fatalf("validator did not report the English AMP artifact passed:\n%s", combined)
	}
}

func TestMakeBuildPublicationFailureRollsBackAllPages(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()
	prior := []byte("prior generated page\n")
	pagePaths := []string{
		filepath.Join("amp", "index.html"),
		filepath.Join("de", "amp", "index.html"),
		filepath.Join("de", "index.html"),
	}
	for _, relative := range pagePaths {
		target := filepath.Join(output, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("create prior page directory: %v", err)
		}
		if err := os.WriteFile(target, prior, 0o600); err != nil {
			t.Fatalf("write prior page %s: %v", relative, err)
		}
	}
	conflictingTarget := filepath.Join(output, "index.html")
	if err := os.Mkdir(conflictingTarget, 0o755); err != nil {
		t.Fatalf("create conflicting output directory: %v", err)
	}

	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	combined, err := build.CombinedOutput()
	if err == nil {
		t.Fatal("make build replaced a directory where a generated page should be")
	}
	if !strings.Contains(string(combined), "publish generated pages") {
		t.Fatalf("publication failure diagnostic is not actionable:\n%s", combined)
	}
	for _, relative := range pagePaths {
		after, err := os.ReadFile(filepath.Join(output, relative))
		if err != nil {
			t.Fatalf("read prior page %s after failed build: %v", relative, err)
		}
		if string(after) != string(prior) {
			t.Errorf("failed build changed prior page %s", relative)
		}
	}
	info, err := os.Stat(conflictingTarget)
	if err != nil {
		t.Fatalf("inspect conflicting target after failed build: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("failed build replaced the conflicting target directory")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate contract test source")
	}
	return filepath.Dir(filepath.Dir(filename))
}
