package sitegen

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateCachedTranslationRejectsMarkdownStructureChanges(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		translated string
	}{
		{
			name:       "element hierarchy",
			source:     "<ul><li>First</li><li>Second</li></ul>",
			translated: "<p>Erste</p><p>Zweite</p>",
		},
		{
			name:       "dropped block",
			source:     "<p>First</p><p>Second</p>",
			translated: "<p>Erste und zweite</p>",
		},
		{
			name:       "ordered list start",
			source:     `<ol start="3"><li>First</li></ol>`,
			translated: `<ol start="4"><li>Erste</li></ol>`,
		},
		{
			name:       "list item value",
			source:     `<ol><li value="3">First</li></ol>`,
			translated: `<ol><li value="4">Erste</li></ol>`,
		},
		{
			name:       "emptied paragraph",
			source:     `<p>First</p><p>Second</p>`,
			translated: `<p></p><p>Erste und zweite</p>`,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			unit := newTranslationUnit("test.markdown", testCase.source, "html", testCase.source, nil)
			_, err := validateCachedTranslation(unit, testCase.translated, false)
			if err == nil {
				t.Fatal("validateCachedTranslation accepted changed Markdown structure")
			}
			if !strings.Contains(err.Error(), "Markdown structure") {
				t.Fatalf("structure-change diagnostic is not actionable: %v", err)
			}
		})
	}
}

func TestValidateCachedTranslationAllowsTextChangesWithinMarkdownStructure(t *testing.T) {
	const source = `<ol start="3"><li value="4"><strong>First</strong></li><li>Second</li></ol>`
	const translated = `<ol start="3"><li value="4"><strong>Erste</strong></li><li>Zweite</li></ol>`
	unit := newTranslationUnit("test.markdown", source, "html", source, nil)

	if _, err := validateCachedTranslation(unit, translated, false); err != nil {
		t.Fatalf("validateCachedTranslation rejected translated text with unchanged structure: %v", err)
	}
}

func TestValidateCachedTranslationAllowsBoundMarkdownLinksToReorder(t *testing.T) {
	const source = `<p><a href="https://one.test/" title="First title"><strong>One</strong></a> and <a href="https://two.test/">Two</a></p>`
	legacy, requestHTML, links, err := extractMarkdownLinkBindings("test.markdown", source)
	if err != nil {
		t.Fatalf("extract Markdown link bindings: %v", err)
	}
	if len(links) != 2 || links[0].TitleID == "" || links[1].TitleID != "" {
		t.Fatalf("unexpected Markdown link bindings: %#v", links)
	}
	unit := newTranslationUnit("test.markdown", legacy, "html", requestHTML, nil)
	unit.MarkdownLinks = links
	translated := fmt.Sprintf(
		`<p><a href="%s" title="%s">Zwei</a> und <a href="%s" title="%s"><strong>Eins</strong></a></p>`,
		links[1].Href,
		links[1].Marker,
		links[0].Href,
		links[0].Marker,
	)

	if _, err := validateCachedTranslation(unit, translated, false); err != nil {
		t.Fatalf("validateCachedTranslation rejected reordered bound links: %v", err)
	}
}

func TestValidateCachedTranslationRejectsEmptiedBoundMarkdownLink(t *testing.T) {
	const source = `<p><a href="https://one.test/">One</a></p>`
	legacy, requestHTML, links, err := extractMarkdownLinkBindings("test.markdown", source)
	if err != nil {
		t.Fatalf("extract Markdown link bindings: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("unexpected Markdown link bindings: %#v", links)
	}
	unit := newTranslationUnit("test.markdown", legacy, "html", requestHTML, nil)
	unit.MarkdownLinks = links
	translated := fmt.Sprintf(`<p><a href="%s" title="%s"></a>Eins</p>`, links[0].Href, links[0].Marker)

	_, err = validateCachedTranslation(unit, translated, false)
	if err == nil {
		t.Fatal("validateCachedTranslation accepted an emptied bound link")
	}
	if !strings.Contains(err.Error(), "Markdown structure") {
		t.Fatalf("empty-bound-link diagnostic is not actionable: %v", err)
	}
}
