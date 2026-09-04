package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestFourPageMetadataAndSelectorMatrix(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()
	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build four-page site: %v\n%s", err, combined)
	}

	base := "https://frathe.github.io/picfetch/"
	cases := []struct {
		path        string
		language    string
		canonical   string
		amp         string
		englishPeer string
		germanPeer  string
		xDefault    string
		ogLocale    string
		localPrefix string
		isAMP       bool
	}{
		{"index.html", "en", base, base + "amp/", base, base + "de/", base, "en_GB", "./", false},
		{filepath.Join("de", "index.html"), "de", base + "de/", base + "de/amp/", base, base + "de/", base, "de_DE", "../", false},
		{filepath.Join("amp", "index.html"), "en", base, "", base + "amp/", base + "de/amp/", base + "amp/", "en_GB", "../", true},
		{filepath.Join("de", "amp", "index.html"), "de", base + "de/", "", base + "amp/", base + "de/amp/", base + "amp/", "de_DE", "../../", true},
	}
	for _, testCase := range cases {
		page, err := os.ReadFile(filepath.Join(output, testCase.path))
		if err != nil {
			t.Errorf("read %s: %v", testCase.path, err)
			continue
		}
		html := string(page)
		labelPrefix := ""
		if testCase.language == "de" {
			labelPrefix = "Deutsch: "
		}
		for _, want := range []string{
			`lang="` + testCase.language + `"`,
			`rel="canonical" href="` + testCase.canonical + `"`,
			`rel="alternate" hreflang="en" href="` + testCase.englishPeer + `"`,
			`rel="alternate" hreflang="de" href="` + testCase.germanPeer + `"`,
			`rel="alternate" hreflang="x-default" href="` + testCase.xDefault + `"`,
			`property="og:locale" content="` + testCase.ogLocale + `"`,
			`href="` + testCase.localPrefix + `favicon.ico"`,
			`href="` + testCase.localPrefix + `manifest.json"`,
			`href="` + testCase.englishPeer + `" hreflang="en"`,
			`href="` + testCase.germanPeer + `" hreflang="de"`,
			`aria-label="` + labelPrefix + `English"`,
			`aria-label="` + labelPrefix + `German"`,
		} {
			if !strings.Contains(html, want) {
				t.Errorf("%s does not contain %q", testCase.path, want)
			}
		}
		for _, unwanted := range []string{
			`href="` + testCase.englishPeer + `" lang=`,
			`href="` + testCase.germanPeer + `" lang=`,
		} {
			if strings.Contains(html, unwanted) {
				t.Errorf("%s contains misleading selector language metadata %q", testCase.path, unwanted)
			}
		}
		if testCase.amp != "" && !strings.Contains(html, `rel="amphtml" href="`+testCase.amp+`"`) {
			t.Errorf("%s does not advertise %s", testCase.path, testCase.amp)
		}
		if testCase.isAMP && strings.Contains(html, "window.location.replace") {
			t.Errorf("%s contains automatic redirect logic", testCase.path)
		}
	}

	var generated []string
	err := filepath.WalkDir(output, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && entry.Name() == "index.html" {
			relative, err := filepath.Rel(output, path)
			if err != nil {
				return err
			}
			generated = append(generated, relative)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("enumerate generated routes: %v", err)
	}
	sort.Strings(generated)
	wantRoutes := []string{filepath.Join("amp", "index.html"), filepath.Join("de", "amp", "index.html"), filepath.Join("de", "index.html"), "index.html"}
	if strings.Join(generated, "\n") != strings.Join(wantRoutes, "\n") {
		t.Fatalf("generated route files = %v, want %v", generated, wantRoutes)
	}

	validate := exec.Command("make", "validate-amp", "SITE_OUTPUT_DIR="+output)
	validate.Dir = repo
	validate.Env = append(os.Environ(), "npm_config_offline=true")
	if combined, err := validate.CombinedOutput(); err != nil {
		t.Fatalf("validate both AMP pages offline: %v\n%s", err, combined)
	}
}
