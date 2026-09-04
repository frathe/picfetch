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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	xhtml "golang.org/x/net/html"
)

const translationCacheVersion = 1
const maxDeepLRequestBytes = 96 << 10
const maxDeepLTextsPerRequest = 50

var ignoredElement = regexp.MustCompile(`(?s)<(?:code|kbd)(?:\s[^>]*)?>(.*?)</(?:code|kbd)>`)
var opaqueAttribute = regexp.MustCompile(`(?:href|src)="([^"]+)"`)
var protectedElement = regexp.MustCompile(`<keep>(.*?)</keep>`)

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
	ID             string
	Source         string
	SourceHash     string
	RequestHash    string
	Format         string
	RequestHTML    string
	ProtectedTerms []string
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
			if validated, err := validateCachedTranslation(unit, entry.Text); err == nil {
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
			validated, err := validateCachedTranslation(unit, translated)
			if err != nil {
				return err
			}
			if strings.TrimSpace(validated) == "" {
				return fmt.Errorf("DeepL returned an empty translation for %s", unit.ID)
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
	addText := func(value LocalizedText) error {
		protected, err := protectTerms(value.Text, content.Site.ProtectedTerms, true)
		if err != nil {
			return fmt.Errorf("prepare translation unit %s: %w", value.ID, err)
		}
		units = append(units, newTranslationUnit(value.ID, value.Text, "text", protected, content.Site.ProtectedTerms))
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
		requestHTML, err := protectMarkdownHTML(validated, content.Site.ProtectedTerms)
		if err != nil {
			return fmt.Errorf("prepare translation unit %s: %w", id, err)
		}
		units = append(units, newTranslationUnit(id, source, "html", requestHTML, content.Site.ProtectedTerms))
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
	return units, nil
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
	return entry.SourceHash == unit.SourceHash &&
		entry.RequestHash == unit.RequestHash &&
		entry.Format == unit.Format &&
		strings.TrimSpace(entry.Text) != ""
}

func protectTerms(source string, terms []string, escape bool) (string, error) {
	sortedTerms := append([]string(nil), terms...)
	sort.SliceStable(sortedTerms, func(i, j int) bool { return len(sortedTerms[i]) > len(sortedTerms[j]) })
	protected := source
	placeholders := make([]string, len(sortedTerms))
	for index, term := range sortedTerms {
		if term == "" {
			return "", fmt.Errorf("protected term at index %d is empty", index)
		}
		placeholder := fmt.Sprintf("PICFETCHPROTECTED%04dTOKEN", index)
		if strings.Contains(source, placeholder) {
			return "", fmt.Errorf("source contains reserved protection marker %q", placeholder)
		}
		placeholders[index] = placeholder
		protected = strings.ReplaceAll(protected, term, placeholder)
	}
	if escape {
		protected = stdhtml.EscapeString(protected)
	}
	for index, placeholder := range placeholders {
		protected = strings.ReplaceAll(protected, placeholder, "<keep>"+stdhtml.EscapeString(sortedTerms[index])+"</keep>")
	}
	return protected, nil
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
	if !sameCapturedValues(ignoredElement, expected, actual, 0) {
		return fmt.Errorf("translation changed protected code or keyboard content in %s", unit.ID)
	}
	if !sameCapturedValues(opaqueAttribute, expected, actual, 1) {
		return fmt.Errorf("translation changed protected URL content in %s", unit.ID)
	}
	terms := append([]string(nil), unit.ProtectedTerms...)
	sort.SliceStable(terms, func(i, j int) bool { return len(terms[i]) > len(terms[j]) })
	seen := make(map[string]bool, len(terms))
	requestText := ignoredElement.ReplaceAllString(unit.RequestHTML, "")
	remaining := ignoredElement.ReplaceAllString(actual, "")
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
		if actualCount := strings.Count(remaining, renderedTerm); actualCount != required {
			return fmt.Errorf("translation changed protected term %q in %s (got %d occurrences, want %d)", term, unit.ID, actualCount, required)
		}
		remaining = strings.ReplaceAll(remaining, renderedTerm, strings.Repeat("\x00", len(renderedTerm)))
		if unit.Format == "html" {
			requiredInAttributes := strings.Count(requestAttributes, term)
			if actualCount := strings.Count(remainingAttributes, term); actualCount != requiredInAttributes {
				return fmt.Errorf("translation changed protected term %q in Markdown attributes for %s (got %d occurrences, want %d)", term, unit.ID, actualCount, requiredInAttributes)
			}
			requestAttributes = strings.ReplaceAll(requestAttributes, term, strings.Repeat("\x00", len(term)))
			remainingAttributes = strings.ReplaceAll(remainingAttributes, term, strings.Repeat("\x00", len(term)))
		}
	}
	return nil
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

func sameCapturedValues(pattern *regexp.Regexp, expected, actual string, group int) bool {
	counts := func(value string) map[string]int {
		captured := make(map[string]int)
		for _, match := range pattern.FindAllStringSubmatch(value, -1) {
			if group < len(match) {
				captured[match[group]]++
			}
		}
		return captured
	}
	expectedCounts := counts(expected)
	actualCounts := counts(actual)
	if len(expectedCounts) != len(actualCounts) {
		return false
	}
	for value, count := range expectedCounts {
		if actualCounts[value] != count {
			return false
		}
	}
	return true
}

func validateCachedTranslation(unit translationUnit, translated string) (string, error) {
	if unit.Format == "html" {
		canonical, err := canonicalSafeMarkdownHTML(unit.ID, translated)
		if err != nil {
			return "", err
		}
		translated = canonical
	}
	if err := validateOpaqueContent(unit, translated); err != nil {
		return "", err
	}
	return translated, nil
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
