package sitecontract_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestRegularAndAMPVariantsPreserveSemanticContentParity(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()
	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build four-page parity fixture: %v\n%s", err, combined)
	}

	for _, locale := range []string{"en", "de"} {
		t.Run(locale, func(t *testing.T) {
			regularPath := filepath.Join(output, "index.html")
			ampPath := filepath.Join(output, "amp", "index.html")
			if locale == "de" {
				regularPath = filepath.Join(output, "de", "index.html")
				ampPath = filepath.Join(output, "de", "amp", "index.html")
			}
			regular := readPageSemantics(t, regularPath)
			amp := readPageSemantics(t, ampPath)
			if !reflect.DeepEqual(regular, amp) {
				t.Fatalf("regular/AMP semantic content differs for %s\nregular: %#v\nAMP:     %#v", locale, regular, amp)
			}
		})
	}
}

type pageSemantics struct {
	VisibleSections []string
	ContentLinks    []string
	ImageAlts       []string
	VideoLabels     []string
	SelectorLabels  []string
}

func readPageSemantics(t *testing.T, path string) pageSemantics {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open generated page %s: %v", path, err)
	}
	document, parseErr := html.Parse(file)
	closeErr := file.Close()
	if parseErr != nil {
		t.Fatalf("parse generated page %s: %v", path, parseErr)
	}
	if closeErr != nil {
		t.Fatalf("close generated page %s: %v", path, closeErr)
	}

	result := pageSemantics{}
	walkPageNodes(document, func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}
		if node.Data == "header" || node.Data == "main" || node.Data == "footer" {
			result.VisibleSections = append(result.VisibleSections, normalizedNodeText(node))
			walkPageNodes(node, func(descendant *html.Node) {
				if descendant.Type != html.ElementNode {
					return
				}
				switch descendant.Data {
				case "a":
					result.ContentLinks = append(result.ContentLinks, attribute(descendant, "href")+" | "+normalizedNodeText(descendant))
				case "img", "amp-img":
					if alt := attribute(descendant, "alt"); alt != "" {
						result.ImageAlts = append(result.ImageAlts, alt)
					}
				case "iframe":
					result.VideoLabels = append(result.VideoLabels, attribute(descendant, "title"))
				case "amp-vimeo":
					result.VideoLabels = append(result.VideoLabels, attribute(descendant, "aria-label"))
				}
			})
		}
		if node.Data == "nav" && strings.Contains(" "+attribute(node, "class")+" ", " language-selector ") {
			result.SelectorLabels = append(result.SelectorLabels, attribute(node, "aria-label"))
			walkPageNodes(node, func(descendant *html.Node) {
				if descendant.Type == html.ElementNode && descendant.Data == "a" {
					result.SelectorLabels = append(result.SelectorLabels,
						fmt.Sprintf("%s | %s", attribute(descendant, "lang"), attribute(descendant, "aria-label")))
				}
			})
		}
	})
	return result
}

func walkPageNodes(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walkPageNodes(child, visit)
	}
}

func normalizedNodeText(node *html.Node) string {
	var text []string
	walkPageNodes(node, func(descendant *html.Node) {
		if descendant.Type == html.TextNode {
			text = append(text, descendant.Data)
		}
	})
	return strings.Join(strings.Fields(strings.Join(text, " ")), " ")
}

func attribute(node *html.Node, name string) string {
	for _, value := range node.Attr {
		if value.Key == name {
			return value.Val
		}
	}
	return ""
}
