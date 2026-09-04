package sitegen

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

var markdownHeading = regexp.MustCompile(`^##\s+.+\s+\{#([a-z][a-z0-9.-]*)\}\s*$`)
var stableIdentity = regexp.MustCompile(`^[a-z][a-z0-9.-]*$`)
var vimeoVideoID = regexp.MustCompile(`^[0-9]+$`)

var reservedSectionAnchors = map[string]string{
	"lightbox": "regular-page lightbox",
}

type Content struct {
	Site          Site              `yaml:"site"`
	Metadata      Metadata          `yaml:"metadata"`
	Icons         []Icon            `yaml:"icons"`
	LanguageFlags LanguageFlags     `yaml:"language_flags"`
	Labels        Labels            `yaml:"labels"`
	Hero          Hero              `yaml:"hero"`
	Sections      []Section         `yaml:"sections"`
	Footer        Footer            `yaml:"footer"`
	Markdown      map[string]string `yaml:"-"`
}

type Site struct {
	BaseURL        string   `yaml:"base_url"`
	ProductName    string   `yaml:"product_name"`
	ProtectedTerms []string `yaml:"protected_terms"`
}

type LocalizedText struct {
	ID   string `yaml:"id"`
	Text string `yaml:"text"`
}

type Metadata struct {
	Title                LocalizedText `yaml:"title"`
	Description          LocalizedText `yaml:"description"`
	OpenGraphTitle       LocalizedText `yaml:"open_graph_title"`
	OpenGraphDescription LocalizedText `yaml:"open_graph_description"`
	OpenGraphImage       string        `yaml:"open_graph_image"`
}

type Icon struct {
	Rel   string `yaml:"rel"`
	Type  string `yaml:"type"`
	Sizes string `yaml:"sizes"`
	Href  string `yaml:"href"`
}

type LanguageFlags struct {
	English string `yaml:"english"`
	German  string `yaml:"german"`
}

type Labels struct {
	LanguageSelector LocalizedText `yaml:"language_selector"`
	English          LocalizedText `yaml:"english"`
	German           LocalizedText `yaml:"german"`
	LightboxClose    LocalizedText `yaml:"lightbox_close"`
	DeepLDisclosure  LocalizedText `yaml:"deepl_disclosure"`
}

type Hero struct {
	Image   Asset         `yaml:"image"`
	Alt     LocalizedText `yaml:"alt"`
	Tagline string        `yaml:"tagline"`
	Actions []Link        `yaml:"actions"`
}

type Asset struct {
	URL    string `yaml:"url"`
	Width  int    `yaml:"width"`
	Height int    `yaml:"height"`
}

type Link struct {
	ID      string        `yaml:"id"`
	Label   LocalizedText `yaml:"label"`
	Href    string        `yaml:"href"`
	Primary bool          `yaml:"primary"`
}

type Section struct {
	ID             string          `yaml:"id"`
	Kind           string          `yaml:"kind"`
	Anchor         string          `yaml:"anchor"`
	Heading        LocalizedText   `yaml:"heading"`
	Body           string          `yaml:"body"`
	Video          *Video          `yaml:"video"`
	Screenshots    []Screenshot    `yaml:"screenshots"`
	Features       []Feature       `yaml:"features"`
	DownloadGroups []DownloadGroup `yaml:"download_groups"`
	Notice         *Notice         `yaml:"notice"`
}

type Video struct {
	ID       string        `yaml:"id"`
	VideoID  string        `yaml:"video_id"`
	Width    int           `yaml:"width"`
	Height   int           `yaml:"height"`
	Autoplay bool          `yaml:"autoplay"`
	Title    LocalizedText `yaml:"title"`
}

func (v *Video) RegularURL() string {
	result := "https://player.vimeo.com/video/" + url.PathEscape(v.VideoID) + "?badge=0&autopause=0&player_id=0&app_id=58479"
	if v.Autoplay {
		result += "&autoplay=1&muted=1&loop=1"
	}
	return result
}

type Screenshot struct {
	ID      string        `yaml:"id"`
	Image   Asset         `yaml:"image"`
	Alt     LocalizedText `yaml:"alt"`
	Caption LocalizedText `yaml:"caption"`
}

type Feature struct {
	ID    string        `yaml:"id"`
	Title LocalizedText `yaml:"title"`
	Body  string        `yaml:"body"`
}

type DownloadGroup struct {
	ID    string        `yaml:"id"`
	Title LocalizedText `yaml:"title"`
	Links []Link        `yaml:"links"`
}

type Notice struct {
	Title LocalizedText `yaml:"title"`
	Body  string        `yaml:"body"`
}

type Footer struct {
	Image    Asset         `yaml:"image"`
	Alt      LocalizedText `yaml:"alt"`
	Links    []Link        `yaml:"links"`
	Colophon string        `yaml:"colophon"`
}

func ParseContent(data []byte) (*Content, error) {
	frontMatter, body, err := splitDocument(data)
	if err != nil {
		return nil, err
	}

	var content Content
	decoder := yaml.NewDecoder(bytes.NewReader(frontMatter))
	decoder.KnownFields(true)
	if err := decoder.Decode(&content); err != nil {
		return nil, fmt.Errorf("website front matter: %w", err)
	}
	content.Markdown, err = parseMarkdownSections(body)
	if err != nil {
		return nil, err
	}
	if err := validateContent(&content); err != nil {
		return nil, err
	}
	return &content, nil
}

func splitDocument(data []byte) ([]byte, []byte, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, nil, fmt.Errorf("website: YAML front matter must start with ---")
	}
	rest := text[4:]
	before, after, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return nil, nil, fmt.Errorf("website: YAML front matter is missing its closing ---")
	}
	return []byte(before), []byte(after), nil
}

func parseMarkdownSections(body []byte) (map[string]string, error) {
	sections := make(map[string]string)
	var id string
	var lines []string
	var fenceMarker byte
	var fenceLength int
	flush := func() error {
		if id == "" {
			return nil
		}
		value := strings.TrimSpace(strings.Join(lines, "\n"))
		if value == "" {
			return fmt.Errorf("markdown section %q: content is required", id)
		}
		sections[id] = value
		return nil
	}

	for lineNumber, line := range strings.Split(string(body), "\n") {
		if fenceMarker != 0 {
			lines = append(lines, line)
			if closesMarkdownFence(line, fenceMarker, fenceLength) {
				fenceMarker = 0
				fenceLength = 0
			}
			continue
		}
		match := markdownHeading.FindStringSubmatch(line)
		if match == nil {
			if id == "" && strings.TrimSpace(line) != "" {
				return nil, fmt.Errorf("markdown line %d: content must follow a level-two heading with a stable {#id}", lineNumber+1)
			}
			lines = append(lines, line)
			if marker, length, ok := opensMarkdownFence(line); ok {
				fenceMarker = marker
				fenceLength = length
			}
			continue
		}
		if err := flush(); err != nil {
			return nil, err
		}
		id = match[1]
		if _, duplicate := sections[id]; duplicate {
			return nil, fmt.Errorf("markdown section %q: duplicate stable identity", id)
		}
		lines = nil
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return sections, nil
}

func opensMarkdownFence(line string) (byte, int, bool) {
	start := 0
	for start < len(line) && start < 4 && line[start] == ' ' {
		start++
	}
	if start > 3 || start == len(line) || (line[start] != '`' && line[start] != '~') {
		return 0, 0, false
	}
	marker := line[start]
	end := start
	for end < len(line) && line[end] == marker {
		end++
	}
	if end-start < 3 || (marker == '`' && strings.ContainsRune(line[end:], '`')) {
		return 0, 0, false
	}
	return marker, end - start, true
}

func closesMarkdownFence(line string, marker byte, minimumLength int) bool {
	start := 0
	for start < len(line) && start < 4 && line[start] == ' ' {
		start++
	}
	if start > 3 || start == len(line) || line[start] != marker {
		return false
	}
	end := start
	for end < len(line) && line[end] == marker {
		end++
	}
	return end-start >= minimumLength && strings.TrimSpace(line[end:]) == ""
}

func validateContent(content *Content) error {
	validator := contentValidator{
		content:        content,
		translationIDs: make(map[string]string),
		markdownRefs:   make(map[string]string),
		itemIDs:        make(map[string]string),
		sectionAnchors: make(map[string]string),
	}
	return validator.validate()
}

type contentValidator struct {
	content        *Content
	translationIDs map[string]string
	markdownRefs   map[string]string
	itemIDs        map[string]string
	sectionAnchors map[string]string
}

func (v *contentValidator) validate() error {
	if err := v.requireBaseURL("site.base_url", v.content.Site.BaseURL); err != nil {
		return err
	}
	if strings.TrimSpace(v.content.Site.ProductName) == "" {
		return fmt.Errorf("site.product_name: value is required")
	}
	if len(v.content.Site.ProtectedTerms) == 0 {
		return fmt.Errorf("site.protected_terms: at least one explicit term is required")
	}

	texts := []struct {
		path  string
		value LocalizedText
	}{
		{"metadata.title", v.content.Metadata.Title},
		{"metadata.description", v.content.Metadata.Description},
		{"metadata.open_graph_title", v.content.Metadata.OpenGraphTitle},
		{"metadata.open_graph_description", v.content.Metadata.OpenGraphDescription},
		{"labels.language_selector", v.content.Labels.LanguageSelector},
		{"labels.english", v.content.Labels.English},
		{"labels.german", v.content.Labels.German},
		{"labels.lightbox_close", v.content.Labels.LightboxClose},
		{"labels.deepl_disclosure", v.content.Labels.DeepLDisclosure},
		{"hero.alt", v.content.Hero.Alt},
		{"footer.alt", v.content.Footer.Alt},
	}
	for _, text := range texts {
		if err := v.requireText(text.path, text.value); err != nil {
			return err
		}
	}
	if err := v.requireURL("metadata.open_graph_image", v.content.Metadata.OpenGraphImage, false); err != nil {
		return err
	}
	for index, icon := range v.content.Icons {
		path := fmt.Sprintf("icons[%d]", index)
		if strings.TrimSpace(icon.Rel) == "" || strings.TrimSpace(icon.Sizes) == "" {
			return fmt.Errorf("%s: rel and sizes are required", path)
		}
		if err := v.requireURL(path+".href", icon.Href, false); err != nil {
			return err
		}
	}
	if len(v.content.Icons) == 0 {
		return fmt.Errorf("icons: at least one icon is required")
	}
	if strings.TrimSpace(v.content.LanguageFlags.English) == "" || strings.TrimSpace(v.content.LanguageFlags.German) == "" {
		return fmt.Errorf("language_flags: English and German indicators are required")
	}
	if err := v.requireAsset("hero.image", v.content.Hero.Image); err != nil {
		return err
	}
	if err := v.requireMarkdown("hero.tagline", v.content.Hero.Tagline); err != nil {
		return err
	}
	for index, action := range v.content.Hero.Actions {
		if err := v.requireLink(fmt.Sprintf("hero.actions[%d]", index), action); err != nil {
			return err
		}
	}
	if len(v.content.Hero.Actions) == 0 {
		return fmt.Errorf("hero.actions: at least one action is required")
	}
	if len(v.content.Sections) == 0 {
		return fmt.Errorf("sections: at least one section is required")
	}
	for index, section := range v.content.Sections {
		path := fmt.Sprintf("sections[%d]", index)
		if err := v.requireItemID(path+".id", section.ID); err != nil {
			return err
		}
		if err := v.requireText(path+".heading", section.Heading); err != nil {
			return err
		}
		switch section.Kind {
		case "video":
			if section.Video == nil {
				return fmt.Errorf("%s.video: value is required for kind video", path)
			}
			if err := v.requireItemID(path+".video.id", section.Video.ID); err != nil {
				return err
			}
			if !vimeoVideoID.MatchString(section.Video.VideoID) {
				return fmt.Errorf("%s.video.video_id: a decimal Vimeo video ID is required", path)
			}
			if section.Video.Width <= 0 || section.Video.Height <= 0 {
				return fmt.Errorf("%s.video: positive width and height are required", path)
			}
			if err := v.requireText(path+".video.title", section.Video.Title); err != nil {
				return err
			}
		case "screenshots":
			if len(section.Screenshots) == 0 {
				return fmt.Errorf("%s.screenshots: at least one screenshot is required", path)
			}
			for itemIndex, screenshot := range section.Screenshots {
				itemPath := fmt.Sprintf("%s.screenshots[%d]", path, itemIndex)
				if err := v.requireItemID(itemPath+".id", screenshot.ID); err != nil {
					return err
				}
				if err := v.requireAsset(itemPath+".image", screenshot.Image); err != nil {
					return err
				}
				if err := v.requireText(itemPath+".alt", screenshot.Alt); err != nil {
					return err
				}
				if err := v.requireText(itemPath+".caption", screenshot.Caption); err != nil {
					return err
				}
			}
		case "features":
			if len(section.Features) == 0 {
				return fmt.Errorf("%s.features: at least one feature is required", path)
			}
			for itemIndex, feature := range section.Features {
				itemPath := fmt.Sprintf("%s.features[%d]", path, itemIndex)
				if err := v.requireItemID(itemPath+".id", feature.ID); err != nil {
					return err
				}
				if err := v.requireText(itemPath+".title", feature.Title); err != nil {
					return err
				}
				if err := v.requireMarkdown(itemPath+".body", feature.Body); err != nil {
					return err
				}
			}
		case "downloads":
			if err := v.requireSectionAnchor(path+".anchor", section.Anchor); err != nil {
				return err
			}
			if err := v.requireMarkdown(path+".body", section.Body); err != nil {
				return err
			}
			if len(section.DownloadGroups) == 0 {
				return fmt.Errorf("%s.download_groups: at least one group is required", path)
			}
			for groupIndex, group := range section.DownloadGroups {
				groupPath := fmt.Sprintf("%s.download_groups[%d]", path, groupIndex)
				if err := v.requireItemID(groupPath+".id", group.ID); err != nil {
					return err
				}
				if err := v.requireText(groupPath+".title", group.Title); err != nil {
					return err
				}
				for linkIndex, link := range group.Links {
					if err := v.requireLink(fmt.Sprintf("%s.links[%d]", groupPath, linkIndex), link); err != nil {
						return err
					}
				}
			}
			if section.Notice == nil {
				return fmt.Errorf("%s.notice: value is required", path)
			}
			if err := v.requireText(path+".notice.title", section.Notice.Title); err != nil {
				return err
			}
			if err := v.requireMarkdown(path+".notice.body", section.Notice.Body); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s.kind: unsupported value %q", path, section.Kind)
		}
	}
	if err := v.requireAsset("footer.image", v.content.Footer.Image); err != nil {
		return err
	}
	for index, link := range v.content.Footer.Links {
		if err := v.requireLink(fmt.Sprintf("footer.links[%d]", index), link); err != nil {
			return err
		}
	}
	if len(v.content.Footer.Links) == 0 {
		return fmt.Errorf("footer.links: at least one link is required")
	}
	if err := v.requireMarkdown("footer.colophon", v.content.Footer.Colophon); err != nil {
		return err
	}
	for id := range v.content.Markdown {
		if _, referenced := v.markdownRefs[id]; !referenced {
			return fmt.Errorf("markdown section %q: no schema field references this content", id)
		}
	}
	return nil
}

func (v *contentValidator) requireText(path string, value LocalizedText) error {
	if !stableIdentity.MatchString(value.ID) {
		return fmt.Errorf("%s: id must be a stable lowercase identity", path)
	}
	if strings.TrimSpace(value.Text) == "" {
		return fmt.Errorf("%s: text is required", path)
	}
	return v.registerTranslationID(path, value.ID)
}

func (v *contentValidator) requireMarkdown(path, id string) error {
	if !stableIdentity.MatchString(id) {
		return fmt.Errorf("%s: Markdown section identity is required", path)
	}
	if _, ok := v.content.Markdown[id]; !ok {
		return fmt.Errorf("%s: Markdown section %q was not found", path, id)
	}
	if err := v.registerTranslationID(path, id); err != nil {
		return err
	}
	v.markdownRefs[id] = path
	return nil
}

func (v *contentValidator) registerTranslationID(path, id string) error {
	if firstPath, exists := v.translationIDs[id]; exists {
		return fmt.Errorf("%s: duplicate translatable identity %q (already used by %s)", path, id, firstPath)
	}
	v.translationIDs[id] = path
	return nil
}

func (v *contentValidator) requireItemID(path, id string) error {
	if !stableIdentity.MatchString(id) {
		return fmt.Errorf("%s: stable lowercase identity is required", path)
	}
	if firstPath, exists := v.itemIDs[id]; exists {
		return fmt.Errorf("%s: duplicate item identity %q (already used by %s)", path, id, firstPath)
	}
	v.itemIDs[id] = path
	return nil
}

func (v *contentValidator) requireSectionAnchor(path, anchor string) error {
	if !stableIdentity.MatchString(anchor) {
		return fmt.Errorf("%s: stable identity is required", path)
	}
	if owner, reserved := reservedSectionAnchors[anchor]; reserved {
		return fmt.Errorf("%s: reserved anchor %q is owned by the %s", path, anchor, owner)
	}
	if firstPath, exists := v.sectionAnchors[anchor]; exists {
		return fmt.Errorf("%s: duplicate anchor %q (already used by %s)", path, anchor, firstPath)
	}
	v.sectionAnchors[anchor] = path
	return nil
}

func (v *contentValidator) requireAsset(path string, asset Asset) error {
	if err := v.requireURL(path+".url", asset.URL, false); err != nil {
		return err
	}
	if asset.Width <= 0 || asset.Height <= 0 {
		return fmt.Errorf("%s: positive width and height are required", path)
	}
	return nil
}

func (v *contentValidator) requireLink(path string, link Link) error {
	if err := v.requireItemID(path+".id", link.ID); err != nil {
		return err
	}
	if err := v.requireText(path+".label", link.Label); err != nil {
		return err
	}
	return v.requireURL(path+".href", link.Href, true)
}

func (v *contentValidator) requireURL(path, raw string, allowFragment bool) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s: invalid URL: %w", path, err)
	}
	if allowFragment && parsed.Scheme == "" && parsed.Host == "" && parsed.Path == "" && parsed.Fragment != "" {
		return nil
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%s: absolute HTTPS URL is required", path)
	}
	return nil
}

func (v *contentValidator) requireBaseURL(path, raw string) error {
	if err := v.requireURL(path, raw, false); err != nil {
		return err
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s: invalid URL: %w", path, err)
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || strings.Contains(raw, "#") {
		return fmt.Errorf("%s: query or fragment is not allowed", path)
	}
	return nil
}
