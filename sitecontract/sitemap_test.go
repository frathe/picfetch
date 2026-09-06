package sitecontract_test

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/frathe/picfetch/internal/sitegen"
)

func TestBuildSitemapListsRegularCanonicalPages(t *testing.T) {
	repo := repositoryRoot(t)
	options := sitegen.BuildOptions{
		SourcePath:       filepath.Join(repo, "website.md"),
		TemplatesPath:    filepath.Join(repo, "site", "templates"),
		TranslationsPath: createControlledGermanCache(t, repo),
		OutputPath:       t.TempDir(),
		Locales:          []string{"en", "de"},
		Formats:          []string{"regular", "amp"},
	}
	if err := sitegen.Build(options); err != nil {
		t.Fatalf("build website: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(options.OutputPath, "sitemap.xml"))
	if err != nil {
		t.Fatalf("read generated sitemap: %v", err)
	}
	locations := sitemapLocations(t, data)
	want := []string{"https://frathe.github.io/picfetch/", "https://frathe.github.io/picfetch/de/"}
	if !reflect.DeepEqual(locations, want) {
		t.Fatalf("sitemap locations = %v, want regular canonical URLs only: %v", locations, want)
	}
	options.Locales = []string{"de", "en", "de"}
	options.Formats = []string{"amp", "regular", "amp"}
	options.OutputPath = t.TempDir()
	if err := sitegen.Build(options); err != nil {
		t.Fatalf("rebuild with reordered and repeated locales/formats: %v", err)
	}
	reordered, err := os.ReadFile(filepath.Join(options.OutputPath, "sitemap.xml"))
	if err != nil {
		t.Fatalf("read reordered sitemap: %v", err)
	}
	if !bytes.Equal(data, reordered) {
		t.Fatalf("locale/format ordering and repetition changed sitemap bytes:\n%s", reordered)
	}
}

func TestBuildSitemapFollowsConfiguredBaseAndSelectedRoutes(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	changed := strings.Replace(string(source), "base_url: https://frathe.github.io/picfetch/", "base_url: https://example.com/a&b", 1)
	if changed == string(source) {
		t.Fatal("test setup did not change the base URL")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(changed), 0o600); err != nil {
		t.Fatalf("write custom website source: %v", err)
	}
	options := sitegen.BuildOptions{
		SourcePath:       sourcePath,
		TemplatesPath:    filepath.Join(repo, "site", "templates"),
		TranslationsPath: createControlledGermanCacheForSource(t, repo, sourcePath),
	}
	cases := []struct {
		name    string
		locales []string
		formats []string
		want    []string
	}{
		{"english", []string{"en"}, []string{"regular"}, []string{"https://example.com/a&b/"}},
		{"german", []string{"de"}, []string{"regular"}, []string{"https://example.com/a&b/de/"}},
		{"both", []string{"en", "de"}, []string{"regular", "amp"}, []string{"https://example.com/a&b/", "https://example.com/a&b/de/"}},
		{"amp only", []string{"en", "de"}, []string{"amp"}, nil},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			options.OutputPath = t.TempDir()
			options.Locales = testCase.locales
			options.Formats = testCase.formats
			if err := sitegen.Build(options); err != nil {
				t.Fatalf("build website: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(options.OutputPath, "sitemap.xml"))
			if testCase.want == nil {
				if !os.IsNotExist(err) {
					t.Fatalf("AMP-only build sitemap read error = %v, want no sitemap", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("read generated sitemap: %v", err)
			}
			if locations := sitemapLocations(t, data); !reflect.DeepEqual(locations, testCase.want) {
				t.Fatalf("sitemap locations = %v, want %v", locations, testCase.want)
			}
			if !strings.Contains(string(data), "https://example.com/a&amp;b/") {
				t.Fatalf("sitemap does not XML-escape authored URLs:\n%s", data)
			}
		})
	}
}

func sitemapLocations(t *testing.T, data []byte) []string {
	t.Helper()
	var sitemap struct {
		XMLName xml.Name `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
		URLs    []struct {
			Location string `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 loc"`
		} `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 url"`
	}
	if err := xml.Unmarshal(data, &sitemap); err != nil {
		t.Fatalf("parse sitemap XML: %v", err)
	}
	var locations []string
	for _, entry := range sitemap.URLs {
		locations = append(locations, entry.Location)
	}
	return locations
}

func TestCheckRejectsMissingOrStaleSitemap(t *testing.T) {
	repo := repositoryRoot(t)
	options := sitegen.BuildOptions{
		SourcePath:       filepath.Join(repo, "website.md"),
		TemplatesPath:    filepath.Join(repo, "site", "templates"),
		TranslationsPath: createControlledGermanCache(t, repo),
		Locales:          []string{"en", "de"},
		Formats:          []string{"regular", "amp"},
	}
	for _, missing := range []bool{true, false} {
		name := "stale"
		if missing {
			name = "missing"
		}
		t.Run(name, func(t *testing.T) {
			options.OutputPath = t.TempDir()
			if err := sitegen.Build(options); err != nil {
				t.Fatalf("build website: %v", err)
			}
			writeStaticSiteFiles(t, options.OutputPath)
			if err := sitegen.Check(options); err != nil {
				t.Fatalf("check current website: %v", err)
			}
			path := filepath.Join(options.OutputPath, "sitemap.xml")
			if missing {
				if err := os.Remove(path); err != nil {
					t.Fatalf("remove sitemap: %v", err)
				}
			} else if err := os.WriteFile(path, []byte("<urlset/>\n"), 0o600); err != nil {
				t.Fatalf("replace sitemap: %v", err)
			}
			err := sitegen.Check(options)
			if err == nil || !strings.Contains(err.Error(), "stale generated artifact: sitemap.xml") {
				t.Fatalf("check %s sitemap error = %v, want actionable stale artifact diagnostic", name, err)
			}
		})
	}
}
