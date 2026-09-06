package sitegen

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// validateCanonical keeps every alias and AMP page consolidated under its
// language's regular URL, even when a template is changed independently.
func validateCanonical(data []byte, expected string) error {
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("parse canonical metadata: %w", err)
	}
	var canonicals []*html.Node
	walkHTML(document, func(node *html.Node) {
		if node.Type != html.ElementNode || node.Data != "link" {
			return
		}
		for _, attribute := range node.Attr {
			if attribute.Key != "rel" {
				continue
			}
			for _, relation := range strings.Fields(attribute.Val) {
				if strings.EqualFold(relation, "canonical") {
					canonicals = append(canonicals, node)
				}
			}
		}
	})
	if len(canonicals) != 1 {
		return fmt.Errorf("expected exactly one canonical link in head, found %d", len(canonicals))
	}
	canonical := canonicals[0]
	if canonical.Parent == nil || canonical.Parent.Data != "head" {
		return fmt.Errorf("canonical link must be in head")
	}
	var hrefs []string
	for _, attribute := range canonical.Attr {
		switch attribute.Key {
		case "href":
			hrefs = append(hrefs, attribute.Val)
		case "hreflang", "lang", "media", "type":
			return fmt.Errorf("canonical link must not have %s: Google ignores canonical annotations with alternate attributes", attribute.Key)
		}
	}
	if len(hrefs) != 1 || hrefs[0] != expected {
		return fmt.Errorf("canonical link must point to %q, found %q", expected, hrefs)
	}
	return nil
}
