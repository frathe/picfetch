package sitegen

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
	xhtml "golang.org/x/net/html"
)

type BuildOptions struct {
	SourcePath       string
	TemplatesPath    string
	TranslationsPath string
	OutputPath       string
	Locales          []string
	Formats          []string
}

type Alternate struct {
	Language string
	URL      string
}

type LanguageLink struct {
	Locale  string
	URL     string
	Flag    string
	Label   LocalizedText
	Current bool
}

type page struct {
	Content              *Content
	Locale               string
	CanonicalURL         string
	AMPURL               string
	Alternates           []Alternate
	LanguageLinks        []LanguageLink
	LocalPrefix          string
	OpenGraphLocale      string
	OpenGraphAlternate   string
	CSS                  template.CSS
	IsGerman             bool
	DetectGerman         bool
	ShowLanguageSelector bool
	translations         map[string]string
	markdownTranslations map[string]template.HTML
}

func Build(options BuildOptions) error {
	if len(options.Locales) == 0 {
		return fmt.Errorf("build requires at least one locale")
	}
	if len(options.Formats) == 0 {
		return fmt.Errorf("build requires at least one format")
	}
	data, err := os.ReadFile(options.SourcePath)
	if err != nil {
		return fmt.Errorf("read website source %q: %w", options.SourcePath, err)
	}
	content, err := ParseContent(data)
	if err != nil {
		return err
	}

	css, err := os.ReadFile(filepath.Join(options.TemplatesPath, "style.css"))
	if err != nil {
		return fmt.Errorf("read shared site styles: %w", err)
	}
	var germanText map[string]string
	var germanMarkdown map[string]template.HTML
	if contains(options.Locales, "de") {
		germanText, germanMarkdown, err = loadCurrentGermanTranslations(content, options.TranslationsPath)
		if err != nil {
			return err
		}
	}

	outputs := make(map[string][]byte)
	for _, locale := range options.Locales {
		for _, format := range options.Formats {
			if (locale != "en" && locale != "de") || (format != "regular" && format != "amp") {
				return fmt.Errorf("build locale %q format %q: not implemented", locale, format)
			}
			textTranslations := map[string]string{}
			markdownTranslations := map[string]template.HTML(nil)
			if locale == "de" {
				textTranslations = germanText
				markdownTranslations = germanMarkdown
			}
			pageData, err := newPage(content, locale, format, css, contains(options.Locales, "de"), textTranslations, markdownTranslations)
			if err != nil {
				return err
			}
			output, err := renderPage(options.TemplatesPath, format, pageData)
			if err != nil {
				return err
			}
			outputs[routePath(locale, format)] = output
		}
	}
	return writeOutputs(options.OutputPath, outputs)
}

func loadCurrentGermanTranslations(content *Content, cachePath string) (map[string]string, map[string]template.HTML, error) {
	units, err := collectTranslationUnits(content)
	if err != nil {
		return nil, nil, err
	}
	cache, err := readTranslationCache(cachePath)
	if err != nil {
		return nil, nil, err
	}
	missing := make([]string, 0)
	validatedTranslations := make(map[string]string, len(units))
	textTranslations := make(map[string]string)
	markdownTranslations := make(map[string]template.HTML)
	for _, unit := range units {
		entry, ok := cache.Entries[unit.ID]
		if !ok || !translationEntryCurrent(unit, entry) {
			missing = append(missing, unit.ID)
			continue
		}
		validated, err := validateCachedTranslation(unit, entry.Text, translationEntryUsesLegacyRequest(unit, entry))
		if err != nil {
			return nil, nil, err
		}
		validatedTranslations[unit.ID] = validated
	}
	if len(missing) > 0 {
		return nil, nil, fmt.Errorf("missing or stale German translation: %s", strings.Join(missing, ", "))
	}
	for _, unit := range units {
		validated := validatedTranslations[unit.ID]
		if unit.Format == "html" {
			assembled, err := restoreMarkdownLinkTitles(unit, validated, validatedTranslations)
			if err != nil {
				return nil, nil, err
			}
			markdownTranslations[unit.ID] = template.HTML(assembled)
		} else {
			textTranslations[unit.ID] = validated
		}
	}
	return textTranslations, markdownTranslations, nil
}

func newPage(content *Content, locale, format string, css []byte, multilingual bool, textTranslations map[string]string, translatedMarkdown map[string]template.HTML) (*page, error) {
	markdown := translatedMarkdown
	if locale == "en" {
		markdown = make(map[string]template.HTML, len(content.Markdown))
		engine := goldmark.New(goldmark.WithRendererOptions(html.WithUnsafe()))
		for id, source := range content.Markdown {
			var rendered bytes.Buffer
			if err := engine.Convert([]byte(source), &rendered); err != nil {
				return nil, fmt.Errorf("render Markdown section %q: %w", id, err)
			}
			renderedHTML := strings.TrimSpace(rendered.String())
			validated, err := canonicalSafeMarkdownHTML(id, renderedHTML)
			if err != nil {
				return nil, err
			}
			markdown[id] = template.HTML(validated)
		}
	}
	base := strings.TrimRight(content.Site.BaseURL, "/") + "/"
	canonicalURL := base
	ampURL := base + "amp/"
	alternateEnglish := base
	alternateGerman := base + "de/"
	localPrefix := "./"
	languageEnglish := base
	languageGerman := base + "de/"
	openGraphLocale := "en_GB"
	openGraphAlternate := "de_DE"
	if locale == "de" {
		canonicalURL = base + "de/"
		ampURL = base + "de/amp/"
		localPrefix = "../"
		openGraphLocale = "de_DE"
		openGraphAlternate = "en_GB"
	}
	if format == "amp" {
		alternateEnglish = base + "amp/"
		alternateGerman = base + "de/amp/"
		localPrefix = "../"
		if locale == "de" {
			localPrefix = "../../"
		}
		languageEnglish = alternateEnglish
		languageGerman = alternateGerman
	}
	return &page{
		Content:      content,
		Locale:       locale,
		CanonicalURL: canonicalURL,
		AMPURL:       ampURL,
		Alternates: []Alternate{
			{Language: "en", URL: alternateEnglish},
			{Language: "de", URL: alternateGerman},
			{Language: "x-default", URL: alternateEnglish},
		},
		LanguageLinks: []LanguageLink{
			{Locale: "en", URL: languageEnglish, Flag: content.LanguageFlags.English, Label: content.Labels.English, Current: locale == "en"},
			{Locale: "de", URL: languageGerman, Flag: content.LanguageFlags.German, Label: content.Labels.German, Current: locale == "de"},
		},
		LocalPrefix:          localPrefix,
		OpenGraphLocale:      openGraphLocale,
		OpenGraphAlternate:   openGraphAlternate,
		CSS:                  template.CSS(css),
		IsGerman:             locale == "de",
		DetectGerman:         multilingual && locale == "en" && format == "regular",
		ShowLanguageSelector: multilingual,
		translations:         textTranslations,
		markdownTranslations: markdown,
	}, nil
}

func contains(values []string, want string) bool {
	return slices.Contains(values, want)
}

func renderPage(templatesPath, format string, data *page) ([]byte, error) {
	functions := template.FuncMap{
		"text": func(value LocalizedText) (string, error) {
			if translated, ok := data.translations[value.ID]; ok {
				return translated, nil
			}
			if data.IsGerman {
				return "", fmt.Errorf("missing German translation %s", value.ID)
			}
			return value.Text, nil
		},
		"markdown": func(id string) (template.HTML, error) {
			translated, ok := data.markdownTranslations[id]
			if !ok {
				return "", fmt.Errorf("missing %s Markdown section %s", data.Locale, id)
			}
			return translated, nil
		},
		"videoAspectRatioPadding": func(video *Video) (template.CSS, error) {
			if video == nil || video.Width <= 0 || video.Height <= 0 {
				return "", fmt.Errorf("video dimensions must be positive")
			}
			percentage := float64(video.Height) * 100 / float64(video.Width)
			return template.CSS(strconv.FormatFloat(percentage, 'f', -1, 64) + "%"), nil
		},
	}
	filename := format + ".html.tmpl"
	tmpl, err := template.New(filename).Funcs(functions).ParseFiles(filepath.Join(templatesPath, filename))
	if err != nil {
		return nil, fmt.Errorf("parse %s template: %w", format, err)
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return nil, fmt.Errorf("render %s %s page: %w", data.Locale, format, err)
	}
	return normalizeHTML(rendered.Bytes()), nil
}

func normalizeHTML(data []byte) []byte {
	data = bytes.TrimSpace(data)
	preformatted := preformattedHTMLBytes(data)
	var normalized bytes.Buffer
	for lineStart := 0; lineStart < len(data); {
		lineEnd := bytes.IndexByte(data[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(data)
		} else {
			lineEnd += lineStart
		}
		trimmedEnd := lineEnd
		for trimmedEnd > lineStart && !preformatted[trimmedEnd-1] && (data[trimmedEnd-1] == ' ' || data[trimmedEnd-1] == '\t') {
			trimmedEnd--
		}
		normalized.Write(data[lineStart:trimmedEnd])
		if lineEnd == len(data) {
			break
		}
		normalized.WriteByte('\n')
		lineStart = lineEnd + 1
	}
	normalized.WriteByte('\n')
	return normalized.Bytes()
}

// preformattedHTMLBytes marks bytes whose whitespace is rendered verbatim by
// HTML. The normalizer may clean up template indentation around the document,
// but must not rewrite fenced code embedded in a <pre> element.
func preformattedHTMLBytes(data []byte) []bool {
	preformatted := make([]bool, len(data))
	tokenizer := xhtml.NewTokenizer(bytes.NewReader(data))
	offset := 0
	preDepth := 0
	for {
		tokenType := tokenizer.Next()
		if tokenType == xhtml.ErrorToken {
			break
		}
		raw := tokenizer.Raw()
		protected := preDepth > 0
		token := xhtml.Token{}
		if tokenType == xhtml.StartTagToken || tokenType == xhtml.EndTagToken {
			token = tokenizer.Token()
		}
		if tokenType == xhtml.EndTagToken && token.Data == "pre" && preDepth > 0 {
			preDepth--
			protected = preDepth > 0
		}
		if protected {
			end := min(offset+len(raw), len(preformatted))
			for index := offset; index < end; index++ {
				preformatted[index] = true
			}
		}
		offset += len(raw)
		if tokenType == xhtml.StartTagToken && token.Data == "pre" {
			preDepth++
		}
	}
	return preformatted
}

func routePath(locale, format string) string {
	parts := make([]string, 0, 2)
	if locale == "de" {
		parts = append(parts, "de")
	}
	if format == "amp" {
		parts = append(parts, "amp")
	}
	parts = append(parts, "index.html")
	return filepath.Join(parts...)
}

func writeOutputs(root string, outputs map[string][]byte) error {
	files := make(map[string][]byte, len(outputs))
	for relative, data := range outputs {
		files[filepath.Join(root, relative)] = data
	}
	if err := commitFileSet(files, root); err != nil {
		return fmt.Errorf("publish generated pages: %w", err)
	}
	return nil
}
