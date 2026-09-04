package sitegen

import (
	"strings"
	"testing"
)

func TestCanonicalSafeMarkdownHTMLRejectsDuplicateAttributes(t *testing.T) {
	tests := []struct {
		name string
		html string
	}{
		{name: "ordinary", html: `<a href="https://example.test/one" href="https://example.test/two">Example</a>`},
		{name: "slash adjacent", html: `<a href="https://example.test/one" /href="https://example.test/two">Example</a>`},
		{name: "slash separated", html: `<a href="https://example.test/one" / href="https://example.test/two">Example</a>`},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := canonicalSafeMarkdownHTML("example", testCase.html)
			if err == nil {
				t.Fatal("canonical Markdown validation accepted duplicate href attributes")
			}
			if !strings.Contains(err.Error(), "duplicate attribute href") {
				t.Fatalf("duplicate-attribute diagnostic is not actionable: %v", err)
			}
		})
	}
}

func TestCanonicalSafeMarkdownHTMLRequiresPureInternalFragment(t *testing.T) {
	if _, err := canonicalSafeMarkdownHTML("example", `<p><a href="#details">Details</a></p>`); err != nil {
		t.Fatalf("canonical Markdown validation rejected a pure internal fragment: %v", err)
	}

	for _, href := range []string{"?preview=1#details", "?#details"} {
		t.Run(href, func(t *testing.T) {
			_, err := canonicalSafeMarkdownHTML("example", `<p><a href="`+href+`">Details</a></p>`)
			if err == nil {
				t.Fatalf("canonical Markdown validation accepted query-bearing internal fragment %q", href)
			}
			if !strings.Contains(err.Error(), "must be an HTTPS URL or internal fragment") {
				t.Fatalf("query-bearing-fragment diagnostic is not actionable: %v", err)
			}
		})
	}
}
