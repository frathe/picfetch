package sitegen

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func renderSitemap(baseURL string, locales []string) ([]byte, error) {
	type sitemapURL struct {
		Location string `xml:"loc"`
	}
	sitemap := struct {
		XMLName xml.Name     `xml:"urlset"`
		XMLNS   string       `xml:"xmlns,attr"`
		URLs    []sitemapURL `xml:"url"`
	}{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	base := strings.TrimRight(baseURL, "/") + "/"
	locations := make(map[string]struct{}, len(locales))
	for _, locale := range locales {
		path := strings.TrimSuffix(filepath.ToSlash(routePath(locale, "regular")), "index.html")
		locations[base+path] = struct{}{}
	}
	sorted := make([]string, 0, len(locations))
	for location := range locations {
		sorted = append(sorted, location)
	}
	sort.Strings(sorted)
	for _, location := range sorted {
		sitemap.URLs = append(sitemap.URLs, sitemapURL{Location: location})
	}
	data, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render sitemap: %w", err)
	}
	return []byte(xml.Header + string(data) + "\n"), nil
}
