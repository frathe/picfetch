package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckGeneratedRejectsStaleDeploymentArtifact(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	committed := t.TempDir()
	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+committed)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare committed site: %v\n%s", err, combined)
	}
	writeStaticSiteFiles(t, committed)

	check := func() ([]byte, error) {
		cmd := exec.Command("make", "check-generated", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+committed)
		cmd.Dir = repo
		return cmd.CombinedOutput()
	}
	if combined, err := check(); err != nil {
		t.Fatalf("current generated site failed check: %v\n%s", err, combined)
	}

	indexPath := filepath.Join(committed, "index.html")
	index, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read generated index: %v", err)
	}
	index = append(index, []byte("<!-- manual edit -->\n")...)
	if err := os.WriteFile(indexPath, index, 0o600); err != nil {
		t.Fatalf("make deployment artifact stale: %v", err)
	}
	combined, err := check()
	if err == nil {
		t.Fatal("check-generated accepted a manually edited deployment artifact")
	}
	if !strings.Contains(string(combined), "stale generated artifact: index.html") {
		t.Fatalf("stale-output diagnostic is not actionable:\n%s", combined)
	}
}

func TestCheckGeneratedRejectsBrokenInternalAnchor(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	broken := strings.Replace(string(source), "href: '#downloads'", "href: '#missing'", 1)
	if broken == string(source) {
		t.Fatal("test setup did not create a broken anchor")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write source with broken anchor: %v", err)
	}
	committed := t.TempDir()
	build := exec.Command("make", "build", "SITE_SOURCE="+sourcePath, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+committed)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare broken generated site: %v\n%s", err, combined)
	}
	writeStaticSiteFiles(t, committed)

	check := exec.Command("make", "check-generated", "SITE_SOURCE="+sourcePath, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+committed)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted a missing internal anchor")
	}
	if !strings.Contains(string(combined), "broken internal anchor #missing") {
		t.Fatalf("broken-anchor diagnostic is not actionable:\n%s", combined)
	}
}

func TestCheckGeneratedRejectsBrokenAnchorOnLinkedPage(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	templates := copyTemplateDirectory(t, repo)
	regularPath := filepath.Join(templates, "regular.html.tmpl")
	regular, err := os.ReadFile(regularPath)
	if err != nil {
		t.Fatalf("read copied regular template: %v", err)
	}
	broken := strings.Replace(string(regular), `href="{{.URL}}"`, `href="/picfetch/de/#missing"`, 1)
	if broken == string(regular) {
		t.Fatal("test setup did not create a broken cross-page anchor")
	}
	if err := os.WriteFile(regularPath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write template with broken linked-page anchor: %v", err)
	}
	output := t.TempDir()
	build := exec.Command("make", "build", "SITE_TEMPLATES="+templates, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare generated site with broken linked-page anchor: %v\n%s", err, combined)
	}
	writeStaticSiteFiles(t, output)

	check := exec.Command("make", "check-generated", "SITE_TEMPLATES="+templates, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted a missing anchor on another generated page")
	}
	if !strings.Contains(string(combined), "broken linked anchor #missing") || !strings.Contains(string(combined), "de/index.html") {
		t.Fatalf("broken-linked-anchor diagnostic is not actionable:\n%s", combined)
	}
}

func TestCheckGeneratedRejectsRootRelativeLinkOutsideSiteBase(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	templates := copyTemplateDirectory(t, repo)
	regularPath := filepath.Join(templates, "regular.html.tmpl")
	regular, err := os.ReadFile(regularPath)
	if err != nil {
		t.Fatalf("read copied regular template: %v", err)
	}
	broken := strings.Replace(string(regular), `href="{{.URL}}"`, `href="/de/"`, 1)
	if broken == string(regular) {
		t.Fatal("test setup did not create a root-relative link outside the site base")
	}
	if err := os.WriteFile(regularPath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write template with root-relative link outside the site base: %v", err)
	}
	output := t.TempDir()
	build := exec.Command("make", "build", "SITE_TEMPLATES="+templates, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare generated site with root-relative link outside the site base: %v\n%s", err, combined)
	}
	writeStaticSiteFiles(t, output)

	check := exec.Command("make", "check-generated", "SITE_TEMPLATES="+templates, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted a root-relative link outside the configured site base")
	}
	if !strings.Contains(string(combined), `root-relative target "/de/" is outside configured site base "/picfetch/"`) {
		t.Fatalf("outside-site-base diagnostic is not actionable:\n%s", combined)
	}
}

func TestCheckGeneratedRejectsMissingLocalAsset(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	committed := t.TempDir()
	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+committed)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare generated site: %v\n%s", err, combined)
	}
	if err := os.WriteFile(filepath.Join(committed, "favicon.ico"), []byte("test icon"), 0o600); err != nil {
		t.Fatalf("write static favicon: %v", err)
	}

	check := exec.Command("make", "check-generated", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+committed)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted a missing local manifest")
	}
	if !strings.Contains(string(combined), "broken local link") || !strings.Contains(string(combined), "manifest.json") {
		t.Fatalf("missing-local-link diagnostic is not actionable:\n%s", combined)
	}
}

func TestCheckGeneratedRejectsMalformedExternalURL(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	templates := copyTemplateDirectory(t, repo)
	regularPath := filepath.Join(templates, "regular.html.tmpl")
	regular, err := os.ReadFile(regularPath)
	if err != nil {
		t.Fatalf("read copied regular template: %v", err)
	}
	broken := strings.Replace(string(regular), "https://player.vimeo.com/api/player.js", "https://[broken", 1)
	if broken == string(regular) {
		t.Fatal("test setup did not create a malformed external URL")
	}
	if err := os.WriteFile(regularPath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write malformed regular template: %v", err)
	}

	committed := t.TempDir()
	build := exec.Command("make", "build", "SITE_TEMPLATES="+templates, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+committed)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare generated site with malformed URL: %v\n%s", err, combined)
	}
	writeStaticSiteFiles(t, committed)

	check := exec.Command("make", "check-generated", "SITE_TEMPLATES="+templates, "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+committed)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted a malformed external URL")
	}
	if !strings.Contains(string(combined), "invalid external URL") || !strings.Contains(string(combined), "https://[broken") {
		t.Fatalf("malformed-external-URL diagnostic is not actionable:\n%s", combined)
	}
}

func copyTemplateDirectory(t *testing.T, repo string) string {
	t.Helper()
	destination := t.TempDir()
	for _, name := range []string{"regular.html.tmpl", "amp.html.tmpl", "style.css"} {
		data, err := os.ReadFile(filepath.Join(repo, "site", "templates", name))
		if err != nil {
			t.Fatalf("read template fixture %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(destination, name), data, 0o600); err != nil {
			t.Fatalf("copy template fixture %s: %v", name, err)
		}
	}
	return destination
}

func writeStaticSiteFiles(t *testing.T, root string) {
	t.Helper()
	for name, data := range map[string]string{
		"favicon.ico":   "test icon",
		"manifest.json": "{}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0o600); err != nil {
			t.Fatalf("write static test file %s: %v", name, err)
		}
	}
}
