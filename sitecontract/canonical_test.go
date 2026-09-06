package sitecontract_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frathe/picfetch/internal/sitegen"
)

func TestBuildRejectsInvalidCanonicalBeforePublishing(t *testing.T) {
	repo := repositoryRoot(t)
	canonical := `<link rel="canonical" href="{{.CanonicalURL}}">`
	for _, testCase := range []struct {
		name        string
		format      string
		locale      string
		replacement string
		inBody      bool
	}{
		{name: "missing regular", format: "regular", locale: "en"},
		{name: "missing AMP", format: "amp", locale: "en"},
		{name: "duplicate", format: "regular", locale: "en", replacement: canonical + canonical},
		{name: "language alternate", format: "regular", locale: "en", replacement: `<link rel="canonical" hreflang="en" href="{{.CanonicalURL}}">`},
		{name: "language attribute", format: "regular", locale: "en", replacement: `<link rel="canonical" lang="en" href="{{.CanonicalURL}}">`},
		{name: "media alternate", format: "regular", locale: "en", replacement: `<link rel="canonical" media="screen" href="{{.CanonicalURL}}">`},
		{name: "format alternate", format: "regular", locale: "en", replacement: `<link rel="canonical" type="text/html" href="{{.CanonicalURL}}">`},
		{name: "relative URL", format: "regular", locale: "en", replacement: `<link rel="canonical" href="./">`},
		{name: "index alias", format: "regular", locale: "en", replacement: `<link rel="canonical" href="{{.CanonicalURL}}index.html">`},
		{name: "German points to English", format: "regular", locale: "de", replacement: `<link rel="canonical" href="{{.Content.Site.BaseURL}}">`},
		{name: "AMP points to itself", format: "amp", locale: "en", replacement: `<link rel="canonical" href="{{.AMPURL}}">`},
		{name: "canonical in body", format: "regular", locale: "en", inBody: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			templates := copyTemplateDirectory(t, repo)
			templatePath := filepath.Join(templates, testCase.format+".html.tmpl")
			original, err := os.ReadFile(templatePath)
			if err != nil {
				t.Fatal(err)
			}
			changed := strings.Replace(string(original), canonical, testCase.replacement, 1)
			if testCase.inBody {
				changed = strings.Replace(changed, "</body>", canonical+"</body>", 1)
			}
			if changed == string(original) {
				t.Fatal("test did not change the canonical tag")
			}
			if err := os.WriteFile(templatePath, []byte(changed), 0o600); err != nil {
				t.Fatal(err)
			}
			output := t.TempDir()
			priorPath := filepath.Join(output, "index.html")
			prior := "previous published page\n"
			if err := os.WriteFile(priorPath, []byte(prior), 0o600); err != nil {
				t.Fatal(err)
			}
			err = sitegen.Build(sitegen.BuildOptions{
				SourcePath:       filepath.Join(repo, "website.md"),
				TemplatesPath:    templates,
				TranslationsPath: filepath.Join(repo, "site", "translations", "de.json"),
				OutputPath:       output,
				Locales:          []string{testCase.locale},
				Formats:          []string{testCase.format},
			})
			if err == nil || !strings.Contains(err.Error(), "canonical") {
				t.Fatalf("build should reject invalid canonical metadata, got %v", err)
			}
			after, err := os.ReadFile(priorPath)
			if err != nil || string(after) != prior {
				t.Fatalf("rejected build changed the published page: %q, %v", after, err)
			}
		})
	}
}
