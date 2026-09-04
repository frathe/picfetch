package sitegen

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestDeepLRequestIgnoresRawPreformattedMarkdown(t *testing.T) {
	const requestHTML = `<pre>literal text without nested code</pre><p>Translate me.</p>`
	body, err := marshalDeepLRequest([]translationUnit{{RequestHTML: requestHTML}})
	if err != nil {
		t.Fatalf("marshal DeepL request: %v", err)
	}
	var request deepLRequest
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("decode DeepL request: %v", err)
	}
	if len(request.Text) != 1 || request.Text[0] != requestHTML {
		t.Fatalf("DeepL request changed Markdown HTML: %#v", request.Text)
	}
	if !slices.Contains(request.IgnoreTags, "pre") {
		t.Fatalf("DeepL request does not ignore raw <pre> content: %#v", request.IgnoreTags)
	}
}

func TestRawPreformattedMarkdownIsValidatedAsOpaque(t *testing.T) {
	const requestHTML = `<pre>literal &amp; <em>nested markup</em></pre><p>Translate me.</p>`
	unit := newTranslationUnit("example.body", requestHTML, "html", requestHTML, nil)

	if _, err := validateCachedTranslation(unit, `<pre>literal &amp; <em>nested markup</em></pre><p>Übersetze mich.</p>`, false); err != nil {
		t.Fatalf("unchanged raw <pre> content was rejected: %v", err)
	}

	_, err := validateCachedTranslation(unit, `<pre>übersetzt &amp; <em>nested markup</em></pre><p>Übersetze mich.</p>`, false)
	if err == nil {
		t.Fatal("changed raw <pre> content was accepted")
	}
	if !strings.Contains(err.Error(), "preformatted block") || !strings.Contains(err.Error(), unit.ID) {
		t.Fatalf("raw <pre> mutation diagnostic is not actionable: %v", err)
	}
}

func TestFencedCodeProtectionBytesRemainStable(t *testing.T) {
	const markdownHTML = `<pre><code>literal text</code></pre>`
	protected, err := protectMarkdownHTML(markdownHTML, nil)
	if err != nil {
		t.Fatalf("protect fenced code: %v", err)
	}
	const expected = `<pre><code><keep>literal text</keep></code></pre>`
	if protected != expected {
		t.Fatalf("fenced-code protection changed: got %q, want %q", protected, expected)
	}
}
