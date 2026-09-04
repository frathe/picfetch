package sitecontract_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeUpdateTranslatesBuildsAndValidatesFourPages(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	output := t.TempDir()
	writeStaticSiteFiles(t, output)
	server := echoingDeepLServer(t)
	defer server.Close()

	cmd := exec.Command("make", "update", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("u", 24), "DEEPL_API_URL="+server.URL, "npm_config_offline=true")
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make update failed: %v\n%s", err, combined)
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("make update did not publish the translation cache: %v", err)
	}
	for _, relative := range []string{
		"index.html",
		filepath.Join("amp", "index.html"),
		filepath.Join("de", "index.html"),
		filepath.Join("de", "amp", "index.html"),
	} {
		if _, err := os.Stat(filepath.Join(output, relative)); err != nil {
			t.Errorf("make update did not publish %s: %v", relative, err)
		}
	}
}

func TestMakeUpdateValidationFailureLeavesCacheAndPagesUntouched(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := filepath.Join(t.TempDir(), "de.json")
	priorCache := []byte("{\n  \"version\": 1,\n  \"locale\": \"de\",\n  \"entries\": {}\n}\n")
	if err := os.WriteFile(cachePath, priorCache, 0o600); err != nil {
		t.Fatalf("write prior cache: %v", err)
	}
	output := t.TempDir()
	writeStaticSiteFiles(t, output)
	priorPage := []byte("prior deployment\n")
	priorPagePath := filepath.Join(output, "index.html")
	if err := os.WriteFile(priorPagePath, priorPage, 0o600); err != nil {
		t.Fatalf("write prior page: %v", err)
	}

	templates := copyTemplateDirectory(t, repo)
	ampPath := filepath.Join(templates, "amp.html.tmpl")
	amp, err := os.ReadFile(ampPath)
	if err != nil {
		t.Fatalf("read copied AMP template: %v", err)
	}
	broken := strings.Replace(string(amp), `<script async src="https://cdn.ampproject.org/v0.js"></script>`, "", 1)
	if broken == string(amp) {
		t.Fatal("test setup did not invalidate the AMP template")
	}
	if err := os.WriteFile(ampPath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write invalid AMP template: %v", err)
	}

	server := echoingDeepLServer(t)
	defer server.Close()
	cmd := exec.Command("make", "update",
		"SITE_TEMPLATES="+templates,
		"SITE_TRANSLATIONS="+cachePath,
		"SITE_OUTPUT_DIR="+output,
	)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("v", 24), "DEEPL_API_URL="+server.URL, "npm_config_offline=true")
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make update accepted invalid staged AMP output")
	}
	if !strings.Contains(string(combined), "AMP validation failed") {
		t.Fatalf("validation failure diagnostic is not actionable:\n%s", combined)
	}
	afterCache, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache after failed update: %v", err)
	}
	if !bytes.Equal(afterCache, priorCache) {
		t.Fatal("failed update changed the prior translation cache")
	}
	afterPage, err := os.ReadFile(priorPagePath)
	if err != nil {
		t.Fatalf("read page after failed update: %v", err)
	}
	if !bytes.Equal(afterPage, priorPage) {
		t.Fatal("failed update changed the prior deployment page")
	}
	for _, relative := range []string{
		filepath.Join("amp", "index.html"),
		filepath.Join("de", "index.html"),
		filepath.Join("de", "amp", "index.html"),
	} {
		if _, err := os.Stat(filepath.Join(output, relative)); !os.IsNotExist(err) {
			t.Errorf("failed update published %s: %v", relative, err)
		}
	}
}

func TestMakeUpdateRejectsUnexpectedExistingRouteBeforePublishing(t *testing.T) {
	repo := repositoryRoot(t)
	fixtureRoot := t.TempDir()
	cachePath := filepath.Join(fixtureRoot, "de.json")
	output := filepath.Join(fixtureRoot, "docs")
	if err := os.MkdirAll(filepath.Join(output, "fr"), 0o755); err != nil {
		t.Fatalf("create deployment fixture: %v", err)
	}
	writeStaticSiteFiles(t, output)
	priorPage := []byte("prior deployment\n")
	if err := os.WriteFile(filepath.Join(output, "index.html"), priorPage, 0o600); err != nil {
		t.Fatalf("write prior root page: %v", err)
	}
	if err := os.WriteFile(filepath.Join(output, "fr", "index.html"), []byte("stale French route\n"), 0o600); err != nil {
		t.Fatalf("write stale route: %v", err)
	}
	server := echoingDeepLServer(t)
	defer server.Close()

	cmd := exec.Command("make", "update", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	cmd.Dir = repo
	cmd.Env = withoutEnvironment(os.Environ(), "DEEPL_API_KEY", "DEEPL_API_URL")
	cmd.Env = append(cmd.Env, "DEEPL_API_KEY="+strings.Repeat("w", 24), "DEEPL_API_URL="+server.URL, "npm_config_offline=true")
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make update accepted an unexpected existing generated route")
	}
	if !strings.Contains(string(combined), "unexpected generated route: fr/index.html") {
		t.Fatalf("unexpected-route update diagnostic is not actionable:\n%s", combined)
	}
	assertFileBytes(t, filepath.Join(output, "index.html"), priorPage)
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("rejected update published translation cache: %v", err)
	}
}

func echoingDeepLServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var payload struct {
			Text []string `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			http.Error(response, "bad request", http.StatusBadRequest)
			return
		}
		translations := make([]map[string]string, len(payload.Text))
		for index, source := range payload.Text {
			translations[index] = map[string]string{"text": "Deutsch: " + source}
		}
		if err := json.NewEncoder(response).Encode(map[string]any{"translations": translations}); err != nil {
			t.Errorf("write fake DeepL response: %v", err)
		}
	}))
}
