package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMakeBuildRejectsSymlinkedOutputParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows hosts")
	}
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(output, "de")); err != nil {
		t.Fatalf("create output-parent symlink: %v", err)
	}

	cmd := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make build followed a symlinked output parent")
	}
	if !strings.Contains(string(combined), "symbolic link") {
		t.Fatalf("symlinked-output diagnostic is not actionable:\n%s", combined)
	}
	assertDirectoryEmpty(t, outside)
}

func TestMakeBuildRejectsSymlinkedOutputAncestor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows hosts")
	}
	container := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(container, "redirect")); err != nil {
		t.Fatalf("create output-ancestor symlink: %v", err)
	}

	cmd := exec.Command("make", "build",
		"SITE_OUTPUT_DIR="+filepath.Join(container, "redirect", "docs"),
		"SITE_LOCALES=en",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repositoryRoot(t)
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make build followed a symlink above the configured output root")
	}
	if !strings.Contains(string(combined), "symbolic link") {
		t.Fatalf("symlinked-output-ancestor diagnostic is not actionable:\n%s", combined)
	}
	assertDirectoryEmpty(t, outside)
}

func TestMakeTranslateRejectsSymlinkedCacheParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows hosts")
	}
	repo := repositoryRoot(t)
	cacheRoot := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(cacheRoot, "translations")); err != nil {
		t.Fatalf("create cache-parent symlink: %v", err)
	}
	server := echoingDeepLServer(t)
	defer server.Close()

	cmd := exec.Command("make", "translate", "SITE_TRANSLATIONS="+filepath.Join(cacheRoot, "translations", "de.json"))
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("y", 24), "DEEPL_API_URL="+server.URL)
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make translate followed a symlinked cache parent")
	}
	if !strings.Contains(string(combined), "symbolic link") {
		t.Fatalf("symlinked-cache diagnostic is not actionable:\n%s", combined)
	}
	assertDirectoryEmpty(t, outside)
}

func TestCheckGeneratedRejectsSymlinkedLocalAsset(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows hosts")
	}
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()
	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare generated site: %v\n%s", err, combined)
	}
	if err := os.WriteFile(filepath.Join(output, "favicon.ico"), []byte("test icon"), 0o600); err != nil {
		t.Fatalf("write local favicon: %v", err)
	}
	outsideAsset := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(outsideAsset, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write outside asset: %v", err)
	}
	if err := os.Symlink(outsideAsset, filepath.Join(output, "manifest.json")); err != nil {
		t.Fatalf("create local-asset symlink: %v", err)
	}

	check := exec.Command("make", "check-generated", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted a local asset symlink that escapes the deployment root")
	}
	if !strings.Contains(string(combined), "symbolic link") || !strings.Contains(string(combined), "manifest.json") {
		t.Fatalf("symlinked-local-asset diagnostic is not actionable:\n%s", combined)
	}
}

func TestCheckGeneratedRejectsSymlinkedExpectedRoute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows hosts")
	}
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()
	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("prepare generated site: %v\n%s", err, combined)
	}
	writeStaticSiteFiles(t, output)

	expectedRoute := filepath.Join(output, "de", "index.html")
	page, err := os.ReadFile(expectedRoute)
	if err != nil {
		t.Fatalf("read expected route fixture: %v", err)
	}
	outsideRoute := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(outsideRoute, page, 0o600); err != nil {
		t.Fatalf("write outside route fixture: %v", err)
	}
	if err := os.Remove(expectedRoute); err != nil {
		t.Fatalf("remove generated route before adding symlink: %v", err)
	}
	if err := os.Symlink(outsideRoute, expectedRoute); err != nil {
		t.Fatalf("create expected-route symlink: %v", err)
	}

	check := exec.Command("make", "check-generated", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	check.Dir = repo
	combined, err := check.CombinedOutput()
	if err == nil {
		t.Fatal("check-generated accepted a symlinked expected route")
	}
	if !strings.Contains(string(combined), "symbolic link") || !strings.Contains(string(combined), "de/index.html") {
		t.Fatalf("symlinked-route diagnostic is not actionable:\n%s", combined)
	}
}

func assertDirectoryEmpty(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read directory %s: %v", directory, err)
	}
	if len(entries) != 0 {
		t.Fatalf("write escaped through symlink into %s: %v", directory, entries)
	}
}
