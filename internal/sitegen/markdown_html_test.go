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
