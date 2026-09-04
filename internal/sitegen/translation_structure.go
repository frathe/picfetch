package sitegen

import (
	"fmt"
	"slices"
	"sort"

	xhtml "golang.org/x/net/html"
)

type markdownStructureToken struct {
	Kind           byte
	Name           string
	Value          string
	HasVisibleText bool
}

type markdownStructure struct {
	Document   []markdownStructureToken
	BoundLinks map[string][]markdownStructureToken
}

const (
	markdownStructureElementStart byte = iota
	markdownStructureAttribute
	markdownStructureElementEnd
	markdownStructureBoundLink
	markdownStructureVisibleText
)

func validateMarkdownStructure(unit translationUnit, translated string) error {
	boundMarkers := make(map[string]struct{}, len(unit.MarkdownLinks))
	for _, link := range unit.MarkdownLinks {
		boundMarkers[link.Marker] = struct{}{}
	}
	expected, err := markdownStructureSignature(stripProtection(unit.RequestHTML), boundMarkers)
	if err != nil {
		return fmt.Errorf("inspect Markdown structure in source %s: %w", unit.ID, err)
	}
	actual, err := markdownStructureSignature(translated, boundMarkers)
	if err != nil {
		return fmt.Errorf("inspect Markdown structure in translation %s: %w", unit.ID, err)
	}
	if !slices.Equal(expected.Document, actual.Document) {
		return fmt.Errorf("translation changed Markdown structure or structural attributes in %s", unit.ID)
	}
	for marker, expectedLink := range expected.BoundLinks {
		actualLink, ok := actual.BoundLinks[marker]
		if !ok || !slices.Equal(expectedLink, actualLink) {
			return fmt.Errorf("translation changed Markdown structure or structural attributes in %s", unit.ID)
		}
	}
	return nil
}

func markdownStructureSignature(value string, boundMarkers map[string]struct{}) (markdownStructure, error) {
	nodes, err := parseMarkdownFragment(value)
	if err != nil {
		return markdownStructure{}, err
	}
	structure := markdownStructure{BoundLinks: make(map[string][]markdownStructureToken, len(boundMarkers))}
	var visit func(*xhtml.Node, *[]markdownStructureToken, bool) bool
	visit = func(node *xhtml.Node, tokens *[]markdownStructureToken, captureBoundLink bool) bool {
		if node.Type == xhtml.TextNode {
			visible := hasVisibleText(node.Data)
			if visible {
				// The words may change, but retaining their position relative to
				// inline children prevents text from silently leaving a link or
				// emphasized span while keeping the same element skeleton.
				*tokens = append(*tokens, markdownStructureToken{Kind: markdownStructureVisibleText})
			}
			return visible
		}
		if node.Type != xhtml.ElementNode {
			return false
		}
		if captureBoundLink && node.Data == "a" {
			marker, hasMarker := markdownAttributeValue(node, "title")
			if _, bound := boundMarkers[marker]; hasMarker && bound {
				linkTokens := make([]markdownStructureToken, 0)
				hasVisibleDescendant := visit(node, &linkTokens, false)
				structure.BoundLinks[marker] = linkTokens
				*tokens = append(*tokens, markdownStructureToken{Kind: markdownStructureBoundLink, Name: "a"})
				return hasVisibleDescendant
			}
		}
		start := len(*tokens)
		*tokens = append(*tokens, markdownStructureToken{Kind: markdownStructureElementStart, Name: node.Data})
		attributes := append([]xhtml.Attribute(nil), node.Attr...)
		sort.Slice(attributes, func(i, j int) bool {
			if attributes[i].Namespace != attributes[j].Namespace {
				return attributes[i].Namespace < attributes[j].Namespace
			}
			if attributes[i].Key != attributes[j].Key {
				return attributes[i].Key < attributes[j].Key
			}
			return attributes[i].Val < attributes[j].Val
		})
		for _, attribute := range attributes {
			value := attribute.Val
			if node.Data == "a" && (attribute.Key == "href" || attribute.Key == "title") {
				// Link destinations and title markers are validated by
				// validateMarkdownLinkBindings. Retaining only their shape here
				// lets links move with translated prose without weakening their
				// binding to the original URL.
				value = ""
			}
			*tokens = append(*tokens, markdownStructureToken{
				Kind:  markdownStructureAttribute,
				Name:  attribute.Namespace + "\x00" + attribute.Key,
				Value: value,
			})
		}
		hasVisibleDescendant := false
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if visit(child, tokens, true) {
				hasVisibleDescendant = true
			}
		}
		(*tokens)[start].HasVisibleText = hasVisibleDescendant
		*tokens = append(*tokens, markdownStructureToken{Kind: markdownStructureElementEnd, Name: node.Data})
		return hasVisibleDescendant
	}
	for _, node := range nodes {
		visit(node, &structure.Document, true)
	}
	return structure, nil
}
