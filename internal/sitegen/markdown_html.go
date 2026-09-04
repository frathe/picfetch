package sitegen

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

func canonicalSafeMarkdownHTML(id, value string) (string, error) {
	if err := validateNoDuplicateMarkdownAttributes(id, value); err != nil {
		return "", err
	}
	context := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(strings.NewReader(value), context)
	if err != nil {
		return "", fmt.Errorf("unsafe HTML in %s: parse fragment: %w", id, err)
	}
	for _, node := range nodes {
		if err := validateMarkdownNode(id, node); err != nil {
			return "", err
		}
	}
	var canonical bytes.Buffer
	for _, node := range nodes {
		if err := html.Render(&canonical, node); err != nil {
			return "", fmt.Errorf("render validated HTML in %s: %w", id, err)
		}
	}
	// html.Render uses the numeric form for quotes in text nodes. Keep the
	// existing, equally safe named entity so generated pages remain stable.
	return strings.ReplaceAll(strings.TrimSpace(canonical.String()), "&#34;", "&quot;"), nil
}

func validateNoDuplicateMarkdownAttributes(id, value string) error {
	tokenizer := html.NewTokenizer(strings.NewReader(value))
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			if errors.Is(tokenizer.Err(), io.EOF) {
				return nil
			}
			return fmt.Errorf("unsafe HTML in %s: inspect attributes: %w", id, tokenizer.Err())
		}
		if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
			continue
		}
		element, attribute, duplicate := duplicateRawHTMLAttribute(tokenizer.Raw())
		if duplicate {
			return fmt.Errorf("unsafe HTML in %s: duplicate attribute %s on <%s>", id, attribute, element)
		}
	}
}

// x/net/html follows the browser parser and discards later duplicate
// attributes. Inspect each raw start tag first so canonicalization cannot hide
// an ambiguous attribute binding from validation.
func duplicateRawHTMLAttribute(raw []byte) (string, string, bool) {
	if len(raw) < 3 || raw[0] != '<' {
		return "", "", false
	}
	position := 1
	elementStart := position
	for position < len(raw) && !rawHTMLAttributeDelimiter(raw[position]) {
		position++
	}
	element := strings.ToLower(string(raw[elementStart:position]))
	seen := make(map[string]bool)
	for position < len(raw) {
		for position < len(raw) && rawHTMLSpace(raw[position]) {
			position++
		}
		if position >= len(raw) || raw[position] == '>' {
			return "", "", false
		}
		if raw[position] == '/' {
			closing := position + 1
			for closing < len(raw) && rawHTMLSpace(raw[closing]) {
				closing++
			}
			if closing >= len(raw) || raw[closing] == '>' {
				return "", "", false
			}
			position++
			continue
		}
		attributeStart := position
		for position < len(raw) && !rawHTMLAttributeDelimiter(raw[position]) && raw[position] != '=' {
			position++
		}
		attribute := strings.ToLower(string(raw[attributeStart:position]))
		if seen[attribute] {
			return element, attribute, true
		}
		seen[attribute] = true
		for position < len(raw) && rawHTMLSpace(raw[position]) {
			position++
		}
		if position >= len(raw) || raw[position] != '=' {
			continue
		}
		position++
		for position < len(raw) && rawHTMLSpace(raw[position]) {
			position++
		}
		if position >= len(raw) {
			return "", "", false
		}
		if raw[position] == '\'' || raw[position] == '"' {
			quote := raw[position]
			position++
			for position < len(raw) && raw[position] != quote {
				position++
			}
			if position < len(raw) {
				position++
			}
			continue
		}
		for position < len(raw) && !rawHTMLSpace(raw[position]) && raw[position] != '>' {
			position++
		}
	}
	return "", "", false
}

func rawHTMLAttributeDelimiter(value byte) bool {
	return rawHTMLSpace(value) || value == '/' || value == '>'
}

func rawHTMLSpace(value byte) bool {
	return value == ' ' || value == '\n' || value == '\r' || value == '\t' || value == '\f'
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
		seenAttributes := make(map[string]bool, len(node.Attr))
		for _, attribute := range node.Attr {
			attributeIdentity := attribute.Namespace + "\x00" + attribute.Key
			if seenAttributes[attributeIdentity] {
				return fmt.Errorf("unsafe HTML in %s: duplicate attribute %s on <%s>", id, attribute.Key, node.Data)
			}
			seenAttributes[attributeIdentity] = true
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
	if parsed.Scheme != "https" {
		return fmt.Errorf("link %q must be an HTTPS URL or internal fragment", raw)
	}
	if err := validateURLAuthority(parsed); err != nil {
		return fmt.Errorf("link %q has invalid URL authority: %w", raw, err)
	}
	return nil
}
