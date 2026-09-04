package sitegen

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var safeCodeClass = regexp.MustCompile(`^language-[A-Za-z0-9_-]+$`)
var safeInteger = regexp.MustCompile(`^[0-9]+$`)

var markdownElementAttributes = map[string]map[string]bool{
	"a":          {"href": true, "title": true},
	"blockquote": {},
	"br":         {},
	"code":       {"class": true},
	"em":         {},
	"hr":         {},
	"kbd":        {},
	"li":         {"value": true},
	"ol":         {"start": true},
	"p":          {},
	"pre":        {},
	"strong":     {},
	"ul":         {},
}

func validateSafeMarkdownHTML(id, value string) error {
	context := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(strings.NewReader(value), context)
	if err != nil {
		return fmt.Errorf("unsafe HTML in %s: parse fragment: %w", id, err)
	}
	for _, node := range nodes {
		if err := validateMarkdownNode(id, node); err != nil {
			return err
		}
	}
	return nil
}

func validateMarkdownNode(id string, node *html.Node) error {
	switch node.Type {
	case html.TextNode:
		// Text is escaped by Goldmark or supplied by the translation service.
	case html.ElementNode:
		allowedAttributes, ok := markdownElementAttributes[node.Data]
		if !ok {
			return fmt.Errorf("unsafe HTML in %s: element <%s> is not allowed", id, node.Data)
		}
		for _, attribute := range node.Attr {
			if attribute.Namespace != "" || !allowedAttributes[attribute.Key] {
				return fmt.Errorf("unsafe HTML in %s: attribute %s on <%s> is not allowed", id, attribute.Key, node.Data)
			}
			switch {
			case node.Data == "a" && attribute.Key == "href":
				if err := validateMarkdownLink(attribute.Val); err != nil {
					return fmt.Errorf("unsafe HTML in %s: %w", id, err)
				}
			case node.Data == "code" && attribute.Key == "class":
				if !safeCodeClass.MatchString(attribute.Val) {
					return fmt.Errorf("unsafe HTML in %s: code class %q is not allowed", id, attribute.Val)
				}
			case (node.Data == "ol" && attribute.Key == "start") || (node.Data == "li" && attribute.Key == "value"):
				if !safeInteger.MatchString(attribute.Val) {
					return fmt.Errorf("unsafe HTML in %s: %s value %q is not an integer", id, attribute.Key, attribute.Val)
				}
			}
		}
	default:
		return fmt.Errorf("unsafe HTML in %s: node type %d is not allowed", id, node.Type)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if err := validateMarkdownNode(id, child); err != nil {
			return err
		}
	}
	return nil
}

func validateMarkdownLink(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("link %q is invalid: %w", raw, err)
	}
	if parsed.Scheme == "" && parsed.Host == "" && parsed.Path == "" && parsed.Fragment != "" {
		return nil
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("link %q must be an HTTPS URL or internal fragment", raw)
	}
	return nil
}
