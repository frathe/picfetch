package sitegen

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	stdhtml "html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/yuin/goldmark"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const translationCacheVersion = 1
const maxDeepLRequestBytes = 96 << 10
const maxDeepLTextsPerRequest = 50

type TranslateOptions struct {
	SourcePath string
	CachePath  string
	Endpoint   string
	APIKey     string
	Client     *http.Client
}

type TranslationCache struct {
	Version int                              `json:"version"`
	Locale  string                           `json:"locale"`
	Entries map[string]TranslationCacheEntry `json:"entries"`
}

type TranslationCacheEntry struct {
	SourceHash  string `json:"source_hash"`
	RequestHash string `json:"request_hash"`
	Format      string `json:"format"`
	Text        string `json:"text"`
}

type translationUnit struct {
	ID                string
	Source            string
	SourceHash        string
	RequestHash       string
	LegacyRequestHash string
	Format            string
	RequestHTML       string
	ProtectedTerms    []string
	MarkdownLinks     []markdownLinkBinding
}

type markdownLinkBinding struct {
	TitleID     string
	Marker      string
	Href        string
	HasHref     bool
	TitleSource string
}

type deepLRequest struct {
	Text               []string `json:"text"`
	TargetLang         string   `json:"target_lang"`
	SourceLang         string   `json:"source_lang"`
	TagHandling        string   `json:"tag_handling"`
	TagHandlingVersion string   `json:"tag_handling_version"`
	IgnoreTags         []string `json:"ignore_tags"`
}

type deepLResponse struct {
	Translations []struct {
		Text string `json:"text"`
	} `json:"translations"`
}

func Translate(options TranslateOptions) error {
	data, err := os.ReadFile(options.SourcePath)
	if err != nil {
		return fmt.Errorf("read website source %q: %w", options.SourcePath, err)
	}
	content, err := ParseContent(data)
	if err != nil {
		return err
	}
	units, err := collectTranslationUnits(content)
	if err != nil {
		return err
	}
	prior, err := readTranslationCache(options.CachePath)
	if err != nil {
		return err
	}

	next := TranslationCache{Version: translationCacheVersion, Locale: "de", Entries: make(map[string]TranslationCacheEntry, len(units))}
	pending := make([]translationUnit, 0)
	for _, unit := range units {
		entry, current := prior.Entries[unit.ID]
		if current && translationEntryCurrent(unit, entry) {
			legacy := translationEntryUsesLegacyRequest(unit, entry)
			if validated, err := validateCachedTranslation(unit, entry.Text, legacy); err == nil {
				entry.RequestHash = unit.RequestHash
				entry.Text = validated
				next.Entries[unit.ID] = entry
				continue
			}
		}
		pending = append(pending, unit)
	}

	if len(pending) > 0 {
		if strings.TrimSpace(options.APIKey) == "" {
			return fmt.Errorf("DEEPL_API_KEY is required for missing or stale translations; current offline builds and checks do not require it")
		}
		translations, err := requestTranslations(context.Background(), options, pending)
		if err != nil {
			return err
		}
		for index, unit := range pending {
			translated := stripProtection(translations[index])
			if unit.Format == "text" {
				translated = stdhtml.UnescapeString(translated)
			}
			validated, err := validateCachedTranslation(unit, translated, false)
			if err != nil {
				return err
			}
			next.Entries[unit.ID] = TranslationCacheEntry{
				SourceHash:  unit.SourceHash,
				RequestHash: unit.RequestHash,
				Format:      unit.Format,
				Text:        validated,
			}
		}
	}

	encoded, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode German translation cache: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := atomicWrite(options.CachePath, encoded); err != nil {
		return fmt.Errorf("write German translation cache: %w", err)
	}
	return nil
}

func collectTranslationUnits(content *Content) ([]translationUnit, error) {
	units := make([]translationUnit, 0)
	protectedTerms := effectiveProtectedTerms(content.Site)
	addText := func(value LocalizedText) error {
		protected, err := protectTerms(value.Text, protectedTerms, true)
		if err != nil {
			return fmt.Errorf("prepare translation unit %s: %w", value.ID, err)
		}
		units = append(units, newTranslationUnit(value.ID, value.Text, "text", protected, protectedTerms))
		return nil
	}
	addMarkdown := func(id string) error {
		source := content.Markdown[id]
		var rendered bytes.Buffer
		engine := goldmark.New(goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()))
		if err := engine.Convert([]byte(source), &rendered); err != nil {
			return fmt.Errorf("render translation unit %s: %w", id, err)
		}
		validated, err := canonicalSafeMarkdownHTML(id, strings.TrimSpace(rendered.String()))
		if err != nil {
			return err
		}
		legacyHTML, markedHTML, links, err := extractMarkdownLinkBindings(id, validated)
		if err != nil {
			return fmt.Errorf("prepare translation unit %s links: %w", id, err)
		}
		legacyRequestHTML, err := protectMarkdownHTML(legacyHTML, protectedTerms)
		if err != nil {
			return fmt.Errorf("prepare translation unit %s: %w", id, err)
		}
		requestHTML, err := protectMarkdownHTML(markedHTML, protectedTerms)
		if err != nil {
			return fmt.Errorf("prepare translation unit %s: %w", id, err)
		}
		parentSource := source
		for _, link := range links {
			if link.TitleID != "" {
				parentSource = legacyHTML
				break
			}
		}
		unit := newTranslationUnit(id, parentSource, "html", requestHTML, protectedTerms)
		legacyRequestHash := sha256.Sum256([]byte(legacyRequestHTML))
		unit.LegacyRequestHash = hex.EncodeToString(legacyRequestHash[:])
		unit.MarkdownLinks = links
		units = append(units, unit)
		for _, link := range links {
			if link.TitleID == "" {
				continue
			}
			protected, err := protectTerms(link.TitleSource, protectedTerms, true)
			if err != nil {
				return fmt.Errorf("prepare translation unit %s: %w", link.TitleID, err)
			}
			units = append(units, newTranslationUnit(link.TitleID, link.TitleSource, "text", protected, protectedTerms))
		}
		return nil
	}

	texts := []LocalizedText{
		content.Metadata.Title,
		content.Metadata.Description,
		content.Metadata.OpenGraphTitle,
		content.Metadata.OpenGraphDescription,
		content.Labels.LanguageSelector,
		content.Labels.English,
		content.Labels.German,
		content.Labels.LightboxClose,
		content.Labels.DeepLDisclosure,
		content.Hero.Alt,
	}
	for _, text := range texts {
		if err := addText(text); err != nil {
			return nil, err
		}
	}
	if err := addMarkdown(content.Hero.Tagline); err != nil {
		return nil, err
	}
	for _, action := range content.Hero.Actions {
		if err := addText(action.Label); err != nil {
			return nil, err
		}
	}
	for _, section := range content.Sections {
		if err := addText(section.Heading); err != nil {
			return nil, err
		}
		switch section.Kind {
		case "video":
			if err := addText(section.Video.Title); err != nil {
				return nil, err
			}
		case "screenshots":
			for _, screenshot := range section.Screenshots {
				if err := addText(screenshot.Alt); err != nil {
					return nil, err
				}
				if err := addText(screenshot.Caption); err != nil {
					return nil, err
				}
			}
		case "features":
			for _, feature := range section.Features {
				if err := addText(feature.Title); err != nil {
					return nil, err
				}
				if err := addMarkdown(feature.Body); err != nil {
					return nil, err
				}
			}
		case "downloads":
			if err := addMarkdown(section.Body); err != nil {
				return nil, err
			}
			for _, group := range section.DownloadGroups {
				if err := addText(group.Title); err != nil {
					return nil, err
				}
				for _, link := range group.Links {
					if err := addText(link.Label); err != nil {
						return nil, err
					}
				}
			}
			if err := addText(section.Notice.Title); err != nil {
				return nil, err
			}
			if err := addMarkdown(section.Notice.Body); err != nil {
				return nil, err
			}
		}
	}
	if err := addText(content.Footer.Alt); err != nil {
		return nil, err
	}
	for _, link := range content.Footer.Links {
		if err := addText(link.Label); err != nil {
			return nil, err
		}
	}
	if err := addMarkdown(content.Footer.Colophon); err != nil {
		return nil, err
	}
	sort.Slice(units, func(i, j int) bool { return units[i].ID < units[j].ID })
	for index := 1; index < len(units); index++ {
		if units[index-1].ID == units[index].ID {
			return nil, fmt.Errorf("duplicate derived translation identity %q", units[index].ID)
		}
	}
	return units, nil
}

func effectiveProtectedTerms(site Site) []string {
	terms := make([]string, 0, len(site.ProtectedTerms)+1)
	productNameIncluded := false
	for _, term := range site.ProtectedTerms {
		if term == site.ProductName {
			if productNameIncluded {
				continue
			}
			productNameIncluded = true
		}
		terms = append(terms, term)
	}
	if !productNameIncluded {
		terms = append(terms, site.ProductName)
	}
	return terms
}

func extractMarkdownLinkBindings(id, value string) (string, string, []markdownLinkBinding, error) {
	nodes, err := parseMarkdownFragment(value)
	if err != nil {
		return "", "", nil, err
	}
	titleOccurrences := make(map[string]int)
	linkOccurrences := make(map[string]int)
	links := make([]markdownLinkBinding, 0)
	type syntheticMarker struct {
		node   *xhtml.Node
		marker string
	}
	syntheticMarkers := make([]syntheticMarker, 0)
	var visit func(*xhtml.Node) error
	visit = func(node *xhtml.Node) error {
		if node.Type == xhtml.ElementNode && node.Data == "a" {
			href, hasHref := markdownAttributeValue(node, "href")
			titleAttribute := -1
			for index, attribute := range node.Attr {
				if attribute.Namespace == "" && attribute.Key == "title" {
					if titleAttribute >= 0 {
						return fmt.Errorf("link has more than one title attribute")
					}
					titleAttribute = index
				}
			}
			if titleAttribute >= 0 {
				titleOccurrences[href]++
				hrefHash := sha256.Sum256([]byte(href))
				titleID := id + "#link-title-" + hex.EncodeToString(hrefHash[:]) + "-" + strconv.Itoa(titleOccurrences[href])
				markerHash := sha256.Sum256([]byte(titleID))
				marker := "sitegen-link-title-" + hex.EncodeToString(markerHash[:])
				links = append(links, markdownLinkBinding{
					TitleID:     titleID,
					Marker:      marker,
					Href:        href,
					HasHref:     hasHref,
					TitleSource: node.Attr[titleAttribute].Val,
				})
				node.Attr[titleAttribute].Val = marker
			} else if hasHref {
				linkOccurrences[href]++
				hrefHash := sha256.Sum256([]byte(href))
				bindingID := id + "#link-binding-" + hex.EncodeToString(hrefHash[:]) + "-" + strconv.Itoa(linkOccurrences[href])
				markerHash := sha256.Sum256([]byte(bindingID))
				marker := "sitegen-link-binding-" + hex.EncodeToString(markerHash[:])
				links = append(links, markdownLinkBinding{Marker: marker, Href: href, HasHref: true})
				syntheticMarkers = append(syntheticMarkers, syntheticMarker{node: node, marker: marker})
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, node := range nodes {
		if err := visit(node); err != nil {
			return "", "", nil, err
		}
	}
	rendered, err := renderMarkdownFragment(nodes)
	if err != nil {
		return "", "", nil, err
	}
	legacyHTML, err := canonicalSafeMarkdownHTML(id, rendered)
	if err != nil {
		return "", "", nil, err
	}
	for _, synthetic := range syntheticMarkers {
		synthetic.node.Attr = append(synthetic.node.Attr, xhtml.Attribute{Key: "title", Val: synthetic.marker})
	}
	rendered, err = renderMarkdownFragment(nodes)
	if err != nil {
		return "", "", nil, err
	}
	markedHTML, err := canonicalSafeMarkdownHTML(id, rendered)
	if err != nil {
		return "", "", nil, err
	}
	return legacyHTML, markedHTML, links, nil
}

func parseMarkdownFragment(value string) ([]*xhtml.Node, error) {
	context := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(strings.NewReader(value), context)
	if err != nil {
		return nil, fmt.Errorf("parse Markdown HTML fragment: %w", err)
	}
	return nodes, nil
}

func renderMarkdownFragment(nodes []*xhtml.Node) (string, error) {
	var rendered bytes.Buffer
	for _, node := range nodes {
		if err := xhtml.Render(&rendered, node); err != nil {
			return "", fmt.Errorf("render Markdown HTML fragment: %w", err)
		}
	}
	return rendered.String(), nil
}

func markdownAttribute(node *xhtml.Node, name string) string {
	value, _ := markdownAttributeValue(node, name)
	return value
}

func markdownAttributeValue(node *xhtml.Node, name string) (string, bool) {
	for _, attribute := range node.Attr {
		if attribute.Namespace == "" && attribute.Key == name {
			return attribute.Val, true
		}
	}
	return "", false
}

func protectMarkdownHTML(value string, terms []string) (string, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(value))
	var protected strings.Builder
	ignoredDepth := 0
	for {
		tokenType := tokenizer.Next()
		if tokenType == xhtml.ErrorToken {
			if errors.Is(tokenizer.Err(), io.EOF) {
				return protected.String(), nil
			}
			return "", tokenizer.Err()
		}
		raw := tokenizer.Raw()
		switch tokenType {
		case xhtml.StartTagToken:
			token := tokenizer.Token()
			protected.Write(raw)
			if token.Data == "code" || token.Data == "kbd" {
				protected.WriteString("<keep>")
				ignoredDepth++
			}
		case xhtml.EndTagToken:
			token := tokenizer.Token()
			if token.Data == "code" || token.Data == "kbd" {
				protected.WriteString("</keep>")
				ignoredDepth--
			}
			protected.Write(raw)
		case xhtml.TextToken:
			if ignoredDepth > 0 {
				protected.Write(raw)
				continue
			}
			text, err := protectTerms(tokenizer.Token().Data, terms, true)
			if err != nil {
				return "", err
			}
			protected.WriteString(text)
		default:
			protected.Write(raw)
		}
	}
}

func newTranslationUnit(id, source, format, requestHTML string, protectedTerms []string) translationUnit {
	sourceHash := sha256.Sum256([]byte(source))
	requestHash := sha256.Sum256([]byte(requestHTML))
	return translationUnit{
		ID:             id,
		Source:         source,
		SourceHash:     hex.EncodeToString(sourceHash[:]),
		RequestHash:    hex.EncodeToString(requestHash[:]),
		Format:         format,
		RequestHTML:    requestHTML,
		ProtectedTerms: append([]string(nil), protectedTerms...),
	}
}

func translationEntryCurrent(unit translationUnit, entry TranslationCacheEntry) bool {
	requestCurrent := entry.RequestHash == unit.RequestHash ||
		(unit.LegacyRequestHash != "" && entry.RequestHash == unit.LegacyRequestHash)
	return entry.SourceHash == unit.SourceHash &&
		requestCurrent &&
		entry.Format == unit.Format &&
		strings.TrimSpace(entry.Text) != ""
}

func translationEntryUsesLegacyRequest(unit translationUnit, entry TranslationCacheEntry) bool {
	return unit.LegacyRequestHash != "" &&
		unit.LegacyRequestHash != unit.RequestHash &&
		entry.RequestHash == unit.LegacyRequestHash
}

func protectTerms(source string, terms []string, escape bool) (string, error) {
	sortedTerms := append([]string(nil), terms...)
	sort.SliceStable(sortedTerms, func(i, j int) bool { return len(sortedTerms[i]) > len(sortedTerms[j]) })
	for index, term := range sortedTerms {
		if term == "" {
			return "", fmt.Errorf("protected term at index %d is empty", index)
		}
	}

	var protected strings.Builder
	protected.Grow(len(source))
	writeText := func(value string) {
		if escape {
			protected.WriteString(stdhtml.EscapeString(value))
			return
		}
		protected.WriteString(value)
	}
	plainStart := 0
	for position := 0; position < len(source); {
		matched := ""
		for _, term := range sortedTerms {
			if strings.HasPrefix(source[position:], term) {
				matched = term
				break
			}
		}
		if matched == "" {
			position++
			continue
		}
		writeText(source[plainStart:position])
		protected.WriteString("<keep>")
		writeText(matched)
		protected.WriteString("</keep>")
		position += len(matched)
		plainStart = position
	}
	writeText(source[plainStart:])
	return protected.String(), nil
}

func requestTranslations(ctx context.Context, options TranslateOptions, units []translationUnit) ([]string, error) {
	endpoint, err := translationEndpoint(options.Endpoint, options.APIKey)
	if err != nil {
		return nil, err
	}
	batches, err := translationBatches(units)
	if err != nil {
		return nil, err
	}
	translations := make([]string, 0, len(units))
	for _, batch := range batches {
		translated, err := requestTranslationBatch(ctx, options, endpoint, batch)
		if err != nil {
			return nil, err
		}
		translations = append(translations, translated...)
	}
	return translations, nil
}

func translationBatches(units []translationUnit) ([][]translationUnit, error) {
	var batches [][]translationUnit
	current := make([]translationUnit, 0)
	for _, unit := range units {
		candidate := append(append([]translationUnit(nil), current...), unit)
		body, err := marshalDeepLRequest(candidate)
		if err != nil {
			return nil, err
		}
		if len(candidate) <= maxDeepLTextsPerRequest && len(body) <= maxDeepLRequestBytes {
			current = candidate
			continue
		}
		if len(current) == 0 {
			return nil, fmt.Errorf("translation unit %s exceeds the DeepL request-size limit", unit.ID)
		}
		batches = append(batches, current)
		current = []translationUnit{unit}
		body, err = marshalDeepLRequest(current)
		if err != nil {
			return nil, err
		}
		if len(body) > maxDeepLRequestBytes {
			return nil, fmt.Errorf("translation unit %s exceeds the DeepL request-size limit", unit.ID)
		}
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches, nil
}

func marshalDeepLRequest(units []translationUnit) ([]byte, error) {
	payload := deepLRequest{
		Text:               make([]string, len(units)),
		TargetLang:         "DE",
		SourceLang:         "EN",
		TagHandling:        "html",
		TagHandlingVersion: "v2",
		IgnoreTags:         []string{"keep", "code", "kbd"},
	}
	for index, unit := range units {
		payload.Text[index] = unit.RequestHTML
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode DeepL request: %w", err)
	}
	return body, nil
}

func requestTranslationBatch(ctx context.Context, options TranslateOptions, endpoint string, units []translationUnit) ([]string, error) {
	body, err := marshalDeepLRequest(units)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create DeepL request: %w", err)
	}
	request.Header.Set("Authorization", "DeepL-Auth-Key "+options.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "picfetch-sitegen/1")
	client := options.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("DeepL translation request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil, deepLStatusError(response.StatusCode)
	}
	var decoded deepLResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 8<<20))
	if err := decoder.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("DeepL returned malformed JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("DeepL returned malformed JSON: multiple response documents")
		}
		return nil, fmt.Errorf("DeepL returned malformed trailing data: %w", err)
	}
	if len(decoded.Translations) != len(units) {
		return nil, fmt.Errorf("DeepL returned %d translations for %d requested units", len(decoded.Translations), len(units))
	}
	translations := make([]string, len(units))
	for index, translated := range decoded.Translations {
		translations[index] = translated.Text
	}
	return translations, nil
}

func translationEndpoint(override, key string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(override), "/")
	if base == "" {
		base = "https://api.deepl.com"
		if strings.HasSuffix(key, ":fx") {
			base = "https://api-free.deepl.com"
		}
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("DeepL endpoint %q is not a valid URL", base)
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "localhost" || parsed.Hostname() == "::1")) {
		return "", fmt.Errorf("DeepL endpoint must use HTTPS (HTTP is allowed only for loopback tests)")
	}
	return base + "/v2/translate", nil
}

func deepLStatusError(status int) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("DeepL authentication failed (HTTP %d)", status)
	case http.StatusTooManyRequests:
		return fmt.Errorf("DeepL rate limit exceeded (HTTP %d)", status)
	case 456:
		return fmt.Errorf("DeepL quota exceeded (HTTP %d)", status)
	default:
		return fmt.Errorf("DeepL translation failed (HTTP %d)", status)
	}
}

func validateOpaqueContent(unit translationUnit, translated string) error {
	expected := stripProtection(unit.RequestHTML)
	actual := stripProtection(translated)
	if unit.Format == "html" {
		expectedOpaque, err := opaqueMarkdownSubtrees(expected)
		if err != nil {
			return fmt.Errorf("inspect protected code or keyboard content in source %s: %w", unit.ID, err)
		}
		actualOpaque, err := opaqueMarkdownSubtrees(actual)
		if err != nil {
			return fmt.Errorf("inspect protected code or keyboard content in translation %s: %w", unit.ID, err)
		}
		if !slices.Equal(expectedOpaque, actualOpaque) {
			return fmt.Errorf("translation changed protected code or keyboard content in %s", unit.ID)
		}
		expectedURLs, err := opaqueMarkdownURLBindings(expected)
		if err != nil {
			return fmt.Errorf("inspect protected URL content in source %s: %w", unit.ID, err)
		}
		actualURLs, err := opaqueMarkdownURLBindings(actual)
		if err != nil {
			return fmt.Errorf("inspect protected URL content in translation %s: %w", unit.ID, err)
		}
		if !sameOpaqueMarkdownURLBindings(expectedURLs, actualURLs) {
			return fmt.Errorf("translation changed protected URL content in %s", unit.ID)
		}
	}
	terms := append([]string(nil), unit.ProtectedTerms...)
	sort.SliceStable(terms, func(i, j int) bool { return len(terms[i]) > len(terms[j]) })
	seen := make(map[string]bool, len(terms))
	requestText := unit.RequestHTML
	remaining := actual
	requestAttributes := ""
	remainingAttributes := ""
	if unit.Format == "html" {
		requestContent, err := inspectTranslatableHTML(unit.RequestHTML, true)
		if err != nil {
			return fmt.Errorf("inspect protected terms in source HTML for %s: %w", unit.ID, err)
		}
		translatedContent, err := inspectTranslatableHTML(actual, false)
		if err != nil {
			return fmt.Errorf("inspect protected terms in translated HTML for %s: %w", unit.ID, err)
		}
		requestText = requestContent.Text
		remaining = translatedContent.Text
		requestAttributes = requestContent.Attributes
		remainingAttributes = translatedContent.Attributes
	}
	for _, term := range terms {
		if seen[term] {
			continue
		}
		seen[term] = true
		requestTerm := stdhtml.EscapeString(term)
		renderedTerm := term
		if unit.Format == "html" {
			requestTerm = term
		}
		required := strings.Count(requestText, "<keep>"+requestTerm+"</keep>")
		requiredInAttributes := 0
		if unit.Format == "html" {
			requiredInAttributes = strings.Count(requestAttributes, term)
		}
		if required == 0 && requiredInAttributes == 0 {
			remaining = strings.ReplaceAll(remaining, renderedTerm, strings.Repeat("\x00", len(renderedTerm)))
			if unit.Format == "html" {
				remainingAttributes = strings.ReplaceAll(remainingAttributes, term, strings.Repeat("\x00", len(term)))
			}
			continue
		}
		if actualCount := strings.Count(remaining, renderedTerm); actualCount != required {
			return fmt.Errorf("translation changed protected term %q in %s (got %d occurrences, want %d)", term, unit.ID, actualCount, required)
		}
		remaining = strings.ReplaceAll(remaining, renderedTerm, strings.Repeat("\x00", len(renderedTerm)))
		if unit.Format == "html" {
			if actualCount := strings.Count(remainingAttributes, term); actualCount != requiredInAttributes {
				return fmt.Errorf("translation changed protected term %q in Markdown attributes for %s (got %d occurrences, want %d)", term, unit.ID, actualCount, requiredInAttributes)
			}
			requestAttributes = strings.ReplaceAll(requestAttributes, term, strings.Repeat("\x00", len(term)))
			remainingAttributes = strings.ReplaceAll(remainingAttributes, term, strings.Repeat("\x00", len(term)))
		}
	}
	return nil
}

func opaqueMarkdownSubtrees(value string) ([]string, error) {
	nodes, err := parseMarkdownFragment(value)
	if err != nil {
		return nil, err
	}
	var subtrees []string
	var visit func(*xhtml.Node) error
	visit = func(node *xhtml.Node) error {
		if node.Type == xhtml.ElementNode && (node.Data == "code" || node.Data == "kbd") {
			var rendered bytes.Buffer
			if err := xhtml.Render(&rendered, node); err != nil {
				return err
			}
			subtrees = append(subtrees, rendered.String())
			return nil
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, node := range nodes {
		if err := visit(node); err != nil {
			return nil, err
		}
	}
	return subtrees, nil
}

type opaqueMarkdownURLBinding struct {
	Element   string
	Attribute string
	Value     string
}

func opaqueMarkdownURLBindings(value string) ([]opaqueMarkdownURLBinding, error) {
	nodes, err := parseMarkdownFragment(value)
	if err != nil {
		return nil, err
	}
	var bindings []opaqueMarkdownURLBinding
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode {
			for _, attribute := range node.Attr {
				if attribute.Namespace == "" && (attribute.Key == "href" || attribute.Key == "src") {
					bindings = append(bindings, opaqueMarkdownURLBinding{
						Element:   node.Data,
						Attribute: attribute.Key,
						Value:     attribute.Val,
					})
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	for _, node := range nodes {
		visit(node)
	}
	return bindings, nil
}

func sameOpaqueMarkdownURLBindings(expected, actual []opaqueMarkdownURLBinding) bool {
	remaining := make(map[opaqueMarkdownURLBinding]int, len(expected))
	for _, binding := range expected {
		remaining[binding]++
	}
	for _, binding := range actual {
		if remaining[binding] == 0 {
			return false
		}
		remaining[binding]--
	}
	for _, count := range remaining {
		if count != 0 {
			return false
		}
	}
	return true
}

type translatableHTMLContent struct {
	Text       string
	Attributes string
}

// inspectTranslatableHTML separates visible text from translatable title
// attributes. Protection markers are optionally retained in visible text so
// source counts can be compared without mistaking opaque URLs for prose.
func inspectTranslatableHTML(value string, retainProtection bool) (translatableHTMLContent, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(value))
	var text strings.Builder
	var attributes strings.Builder
	ignoredDepth := 0
	separate := func() {
		if text.Len() > 0 {
			text.WriteByte(0)
		}
	}
	appendAttribute := func(value string) {
		if attributes.Len() > 0 {
			attributes.WriteByte(0)
		}
		attributes.WriteString(value)
	}
	for {
		tokenType := tokenizer.Next()
		if tokenType == xhtml.ErrorToken {
			if errors.Is(tokenizer.Err(), io.EOF) {
				return translatableHTMLContent{Text: text.String(), Attributes: attributes.String()}, nil
			}
			return translatableHTMLContent{}, tokenizer.Err()
		}
		switch tokenType {
		case xhtml.StartTagToken:
			token := tokenizer.Token()
			for _, attribute := range token.Attr {
				if attribute.Namespace == "" && attribute.Key == "title" {
					appendAttribute(attribute.Val)
				}
			}
			if token.Data == "code" || token.Data == "kbd" {
				separate()
				ignoredDepth++
				continue
			}
			if ignoredDepth > 0 {
				continue
			}
			if retainProtection && token.Data == "keep" {
				text.WriteString("<keep>")
			} else {
				separate()
			}
		case xhtml.EndTagToken:
			token := tokenizer.Token()
			if token.Data == "code" || token.Data == "kbd" {
				if ignoredDepth > 0 {
					ignoredDepth--
				}
				separate()
				continue
			}
			if ignoredDepth > 0 {
				continue
			}
			if retainProtection && token.Data == "keep" {
				text.WriteString("</keep>")
			} else {
				separate()
			}
		case xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			for _, attribute := range token.Attr {
				if attribute.Namespace == "" && attribute.Key == "title" {
					appendAttribute(attribute.Val)
				}
			}
			if ignoredDepth == 0 {
				separate()
			}
		case xhtml.CommentToken, xhtml.DoctypeToken:
			if ignoredDepth == 0 {
				separate()
			}
		case xhtml.TextToken:
			if ignoredDepth == 0 {
				text.WriteString(tokenizer.Token().Data)
			}
		}
	}
}

func validateCachedTranslation(unit translationUnit, translated string, legacyRequest bool) (string, error) {
	if unit.Format == "html" {
		canonical, err := canonicalSafeMarkdownHTML(unit.ID, translated)
		if err != nil {
			return "", err
		}
		translated = canonical
		visible, err := hasVisibleMarkdownContent(translated)
		if err != nil {
			return "", fmt.Errorf("inspect translated Markdown in %s: %w", unit.ID, err)
		}
		if !visible {
			return "", fmt.Errorf("translation for %s has no visible text", unit.ID)
		}
		translated, err = validateMarkdownLinkBindings(unit, translated, legacyRequest)
		if err != nil {
			return "", err
		}
	} else if !hasVisibleText(translated) {
		return "", fmt.Errorf("translation for %s has no visible text", unit.ID)
	}
	if err := validateOpaqueContent(unit, translated); err != nil {
		return "", err
	}
	return translated, nil
}

func hasVisibleMarkdownContent(value string) (bool, error) {
	nodes, err := parseMarkdownFragment(value)
	if err != nil {
		return false, err
	}
	var visible bool
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		if visible {
			return
		}
		if node.Type == xhtml.TextNode && hasVisibleText(node.Data) {
			visible = true
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	for _, node := range nodes {
		visit(node)
	}
	return visible, nil
}

func hasVisibleText(value string) bool {
	for _, character := range value {
		if unicode.IsGraphic(character) && !unicode.IsSpace(character) {
			return true
		}
	}
	return false
}

func validateMarkdownLinkBindings(unit translationUnit, value string, legacyRequest bool) (string, error) {
	expected := make(map[string]markdownLinkBinding, len(unit.MarkdownLinks))
	for _, link := range unit.MarkdownLinks {
		if _, duplicate := expected[link.Marker]; duplicate {
			return "", fmt.Errorf("prepare protected URL bindings in %s: duplicate marker", unit.ID)
		}
		expected[link.Marker] = link
	}
	if legacyRequest {
		return migrateLegacyMarkdownLinkBindings(unit, value)
	}
	found := make(map[string]int, len(expected))
	nodes, err := parseMarkdownFragment(value)
	if err != nil {
		return "", fmt.Errorf("inspect protected URL bindings in %s: %w", unit.ID, err)
	}
	var validationErr error
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		if validationErr != nil {
			return
		}
		if node.Type == xhtml.ElementNode && node.Data == "a" {
			href, hasHref := markdownAttributeValue(node, "href")
			marker, hasMarker := markdownAttributeValue(node, "title")
			if hasHref && !hasMarker {
				validationErr = fmt.Errorf("translation changed protected URL binding marker in %s", unit.ID)
				return
			}
			if hasMarker {
				link, ok := expected[marker]
				if !ok {
					validationErr = fmt.Errorf("translation changed protected URL binding marker in %s", unit.ID)
					return
				}
				if hasHref != link.HasHref || href != link.Href {
					validationErr = fmt.Errorf("translation changed protected URL content in %s", unit.ID)
					return
				}
				found[marker]++
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	for _, node := range nodes {
		visit(node)
	}
	if validationErr != nil {
		return "", validationErr
	}
	for marker := range expected {
		if found[marker] != 1 {
			return "", fmt.Errorf("translation changed protected URL binding marker in %s (got %d occurrences, want 1)", unit.ID, found[marker])
		}
	}
	return value, nil
}

func migrateLegacyMarkdownLinkBindings(unit translationUnit, value string) (string, error) {
	expectedURLs, err := opaqueMarkdownURLBindings(stripProtection(unit.RequestHTML))
	if err != nil {
		return "", fmt.Errorf("inspect protected URL content in source %s: %w", unit.ID, err)
	}
	actualURLs, err := opaqueMarkdownURLBindings(value)
	if err != nil {
		return "", fmt.Errorf("inspect protected URL content in translation %s: %w", unit.ID, err)
	}
	if !slices.Equal(expectedURLs, actualURLs) {
		return "", fmt.Errorf("translation changed protected URL content in %s", unit.ID)
	}
	nodes, err := parseMarkdownFragment(value)
	if err != nil {
		return "", fmt.Errorf("inspect legacy protected URL bindings in %s: %w", unit.ID, err)
	}
	actualLinks := make([]*xhtml.Node, 0, len(unit.MarkdownLinks))
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode && node.Data == "a" {
			_, hasHref := markdownAttributeValue(node, "href")
			_, hasTitle := markdownAttributeValue(node, "title")
			if hasHref || hasTitle {
				actualLinks = append(actualLinks, node)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	for _, node := range nodes {
		visit(node)
	}
	if len(actualLinks) != len(unit.MarkdownLinks) {
		return "", fmt.Errorf("translation changed protected URL binding markers in %s", unit.ID)
	}
	for index, node := range actualLinks {
		expected := unit.MarkdownLinks[index]
		href, hasHref := markdownAttributeValue(node, "href")
		marker, hasMarker := markdownAttributeValue(node, "title")
		if hasHref != expected.HasHref || href != expected.Href {
			return "", fmt.Errorf("translation changed protected URL content in %s", unit.ID)
		}
		if expected.TitleID != "" {
			if !hasMarker || marker != expected.Marker {
				return "", fmt.Errorf("translation changed protected URL binding marker in %s", unit.ID)
			}
			continue
		}
		if hasMarker {
			return "", fmt.Errorf("translation changed protected URL binding marker in %s", unit.ID)
		}
		node.Attr = append(node.Attr, xhtml.Attribute{Key: "title", Val: expected.Marker})
	}
	rendered, err := renderMarkdownFragment(nodes)
	if err != nil {
		return "", fmt.Errorf("migrate protected URL bindings in %s: %w", unit.ID, err)
	}
	canonical, err := canonicalSafeMarkdownHTML(unit.ID, rendered)
	if err != nil {
		return "", err
	}
	return canonical, nil
}

func restoreMarkdownLinkTitles(unit translationUnit, value string, translations map[string]string) (string, error) {
	if len(unit.MarkdownLinks) == 0 {
		return value, nil
	}
	linksByMarker := make(map[string]markdownLinkBinding, len(unit.MarkdownLinks))
	for _, link := range unit.MarkdownLinks {
		linksByMarker[link.Marker] = link
	}
	nodes, err := parseMarkdownFragment(value)
	if err != nil {
		return "", fmt.Errorf("restore Markdown link titles in %s: %w", unit.ID, err)
	}
	var restoreErr error
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		if restoreErr != nil {
			return
		}
		if node.Type == xhtml.ElementNode && node.Data == "a" {
			attributes := node.Attr[:0]
			for _, attribute := range node.Attr {
				if attribute.Namespace != "" || attribute.Key != "title" {
					attributes = append(attributes, attribute)
					continue
				}
				link, ok := linksByMarker[attribute.Val]
				if !ok {
					restoreErr = fmt.Errorf("restore Markdown links in %s: unknown binding marker", unit.ID)
					return
				}
				if link.TitleID == "" {
					continue
				}
				translated, ok := translations[link.TitleID]
				if !ok {
					restoreErr = fmt.Errorf("restore Markdown link titles in %s: missing German translation %s", unit.ID, link.TitleID)
					return
				}
				attribute.Val = translated
				attributes = append(attributes, attribute)
			}
			node.Attr = attributes
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	for _, node := range nodes {
		visit(node)
	}
	if restoreErr != nil {
		return "", restoreErr
	}
	rendered, err := renderMarkdownFragment(nodes)
	if err != nil {
		return "", fmt.Errorf("restore Markdown link titles in %s: %w", unit.ID, err)
	}
	return canonicalSafeMarkdownHTML(unit.ID, rendered)
}

func stripProtection(value string) string {
	value = strings.ReplaceAll(value, "<keep>", "")
	return strings.ReplaceAll(value, "</keep>", "")
}

func readTranslationCache(path string) (TranslationCache, error) {
	cache := TranslationCache{Version: translationCacheVersion, Locale: "de", Entries: map[string]TranslationCacheEntry{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cache, nil
	}
	if err != nil {
		return TranslationCache{}, fmt.Errorf("read German translation cache: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cache); err != nil {
		return TranslationCache{}, fmt.Errorf("parse German translation cache: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return TranslationCache{}, fmt.Errorf("parse German translation cache: unexpected trailing data")
		}
		return TranslationCache{}, fmt.Errorf("parse German translation cache trailing data: %w", err)
	}
	if cache.Version != translationCacheVersion {
		return TranslationCache{}, fmt.Errorf("German translation cache version %d is unsupported", cache.Version)
	}
	if cache.Locale != "de" {
		return TranslationCache{}, fmt.Errorf("German translation cache locale must be de, got %q", cache.Locale)
	}
	if cache.Entries == nil {
		cache.Entries = map[string]TranslationCacheEntry{}
	}
	return cache, nil
}

func atomicWrite(path string, data []byte) (resultErr error) {
	committed := false
	root, err := openSecureWriteRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() {
		if err := root.handle.Close(); err != nil && !committed {
			resultErr = errors.Join(resultErr, fmt.Errorf("close write root for %q: %w", path, err))
		}
	}()
	target, err := root.relative(path)
	if err != nil {
		return err
	}
	if err := root.validateRelativePath(target); err != nil {
		return err
	}
	temporary, temporaryPath, err := createRootTemp(root, ".", ".sitegen-")
	if err != nil {
		return err
	}
	cleanup := func() { _ = root.handle.Remove(temporaryPath) }
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		cleanup()
		return err
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		cleanup()
		return err
	}
	if err := temporary.Close(); err != nil {
		cleanup()
		return err
	}
	if err := root.validateRelativePath(target); err != nil {
		cleanup()
		return err
	}
	if err := root.handle.Rename(temporaryPath, target); err != nil {
		cleanup()
		return err
	}
	committed = true
	return nil
}
