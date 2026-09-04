package sitecontract_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMakeBuildGeneratesEnglishRegularPage(t *testing.T) {
	repo := repositoryRoot(t)
	output := t.TempDir()

	cmd := exec.Command("make", "build", "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en", "SITE_FORMATS=regular")
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}

	page, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatalf("read generated English regular page: %v", err)
	}
	html := string(page)
	for _, want := range []string{
		`<html lang="en">`,
		`<title>PicFetch — a small, fast image viewer for macOS, Windows and Linux</title>`,
		`1220283616`,
		`href="#downloads"`,
		`picfetch-linux-arm64.tar.gz`,
		`class="lightbox"`,
		`prefers-color-scheme: dark`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("generated English regular page does not contain %q", want)
		}
	}
}

func TestRegularVideoAspectRatioComesFromAuthoredDimensions(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	changed := strings.Replace(string(source), "      width: 1000\n      height: 660", "      width: 1600\n      height: 900", 1)
	if changed == string(source) {
		t.Fatal("test setup did not change the first video dimensions")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(changed), 0o600); err != nil {
		t.Fatalf("write website source with a 16:9 video: %v", err)
	}
	output := t.TempDir()

	cmd := exec.Command("make", "build", "SITE_SOURCE="+sourcePath, "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en", "SITE_FORMATS=regular")
	cmd.Dir = repo
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}
	page, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatalf("read generated English regular page: %v", err)
	}
	if !strings.Contains(string(page), `style="padding:56.25% 0 0 0;position:relative;"`) {
		t.Fatal("regular video wrapper does not use the authored 16:9 aspect ratio")
	}
}

func TestVideoIdentityAndAutoplayDriveBothFormats(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	changed := strings.Replace(string(source), "      video_id: '1220283616'", "      video_id: '987654321'", 1)
	changed = strings.Replace(changed, "      autoplay: true", "      autoplay: false", 1)
	if changed == string(source) || strings.Contains(changed, "      video_id: '1220283616'") {
		t.Fatal("test setup did not change the first video identity and autoplay behavior")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(changed), 0o600); err != nil {
		t.Fatalf("write website source with changed video: %v", err)
	}
	output := t.TempDir()

	cmd := exec.Command("make", "build", "SITE_SOURCE="+sourcePath, "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en", "SITE_FORMATS=regular,amp")
	cmd.Dir = repo
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}
	regular, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatalf("read generated English regular page: %v", err)
	}
	amp, err := os.ReadFile(filepath.Join(output, "amp", "index.html"))
	if err != nil {
		t.Fatalf("read generated English AMP page: %v", err)
	}
	if !strings.Contains(string(regular), `src="https://player.vimeo.com/video/987654321?badge=0&amp;autopause=0&amp;player_id=0&amp;app_id=58479"`) {
		t.Fatal("regular page did not derive its URL from the authored video identity and disabled autoplay")
	}
	if !strings.Contains(string(amp), `<amp-vimeo data-videoid="987654321" width="1000" height="660" layout="responsive" aria-label="PicFetch"></amp-vimeo>`) {
		t.Fatal("AMP page did not use the same authored video identity and disabled autoplay")
	}
}

func TestMarkdownSectionHeadingInsideFenceIsLiteralContent(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	needle := "A small desktop app for quickly viewing and browsing images. Drop one or more onto the window, and step through the set with the keyboard.\n\n## Drop almost anything {#features.drop-anything.body}"
	replacement := "A small desktop app for quickly viewing and browsing images. Drop one or more onto the window, and step through the set with the keyboard.\n\n```markdown\n## Example {#not-a-section}\n```\n\n## Drop almost anything {#features.drop-anything.body}"
	changed := strings.Replace(string(source), needle, replacement, 1)
	if changed == string(source) {
		t.Fatal("test setup did not add a fenced Markdown example")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(changed), 0o600); err != nil {
		t.Fatalf("write website source with fenced example: %v", err)
	}
	output := t.TempDir()

	cmd := exec.Command("make", "build", "SITE_SOURCE="+sourcePath, "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en", "SITE_FORMATS=regular")
	cmd.Dir = repo
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("make build rejected a heading inside fenced code: %v\n%s", err, combined)
	}
	page, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatalf("read generated English regular page: %v", err)
	}
	if !strings.Contains(string(page), `<pre><code class="language-markdown">## Example {#not-a-section}`) {
		t.Fatal("heading inside fenced code was not rendered as literal code")
	}
}

func TestInvalidSourceReportsAffectedField(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	broken := strings.Replace(string(source), "id: metadata.description", "id: ''", 1)
	if broken == string(source) {
		t.Fatal("test setup did not remove metadata.description identity")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write invalid website source: %v", err)
	}

	cmd := exec.Command("make", "build", "SITE_SOURCE="+sourcePath, "SITE_OUTPUT_DIR="+t.TempDir(), "SITE_LOCALES=en", "SITE_FORMATS=regular")
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("make build succeeded with a missing metadata.description identity")
	}
	if !strings.Contains(string(combined), "metadata.description") {
		t.Fatalf("diagnostic does not identify metadata.description:\n%s", combined)
	}
}

func TestInvalidSourceRejectsTextAndMarkdownIdentityCollision(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	broken := strings.Replace(string(source), "id: metadata.title", "id: hero.tagline", 1)
	if broken == string(source) {
		t.Fatal("test setup did not create a cross-namespace translation ID collision")
	}
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write invalid website source: %v", err)
	}

	cmd := exec.Command("make", "build",
		"SITE_SOURCE="+sourcePath,
		"SITE_OUTPUT_DIR="+t.TempDir(),
		"SITE_LOCALES=en",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make build accepted a shared text/Markdown translation identity")
	}
	if !strings.Contains(string(combined), `duplicate translatable identity "hero.tagline"`) || !strings.Contains(string(combined), "metadata.title") {
		t.Fatalf("cross-namespace identity diagnostic is not actionable:\n%s", combined)
	}
}

func TestInvalidSourceRejectsQueryAndFragmentInBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{name: "query", baseURL: "https://example.test/site?preview=1"},
		{name: "empty query", baseURL: "https://example.test/site?"},
		{name: "fragment", baseURL: "https://example.test/site#preview"},
		{name: "empty fragment", baseURL: "https://example.test/site#"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := repositoryRoot(t)
			source, err := os.ReadFile(filepath.Join(repo, "website.md"))
			if err != nil {
				t.Fatalf("read website source: %v", err)
			}
			broken := strings.Replace(string(source),
				"  base_url: https://frathe.github.io/picfetch/",
				"  base_url: '"+testCase.baseURL+"'",
				1,
			)
			if broken == string(source) {
				t.Fatal("test setup did not change site.base_url")
			}
			sourcePath := filepath.Join(t.TempDir(), "website.md")
			if err := os.WriteFile(sourcePath, []byte(broken), 0o600); err != nil {
				t.Fatalf("write website source with invalid base URL: %v", err)
			}

			cmd := exec.Command("make", "build",
				"SITE_SOURCE="+sourcePath,
				"SITE_OUTPUT_DIR="+t.TempDir(),
				"SITE_LOCALES=en",
				"SITE_FORMATS=regular",
			)
			cmd.Dir = repo
			combined, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("make build accepted site.base_url %q", testCase.baseURL)
			}
			if !strings.Contains(string(combined), "site.base_url") || !strings.Contains(string(combined), "query or fragment") {
				t.Fatalf("invalid-base-URL diagnostic is not actionable:\n%s", combined)
			}
		})
	}
}

func TestInvalidSourceRejectsReservedAndDuplicateDownloadAnchors(t *testing.T) {
	tests := []struct {
		name       string
		change     func(string) string
		diagnostic string
	}{
		{
			name: "template-owned ID",
			change: func(source string) string {
				return strings.Replace(source, "    anchor: downloads", "    anchor: lightbox", 1)
			},
			diagnostic: `reserved anchor "lightbox"`,
		},
		{
			name: "duplicate downloads anchor",
			change: func(source string) string {
				duplicate := `  - id: downloads-copy
    kind: downloads
    anchor: downloads
    heading:
      id: sections.downloads-copy.heading
      text: Download copy
`
				return strings.Replace(source, "footer:\n", duplicate+"footer:\n", 1)
			},
			diagnostic: `duplicate anchor "downloads"`,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := repositoryRoot(t)
			source, err := os.ReadFile(filepath.Join(repo, "website.md"))
			if err != nil {
				t.Fatalf("read website source: %v", err)
			}
			broken := testCase.change(string(source))
			if broken == string(source) {
				t.Fatal("test setup did not change a downloads anchor")
			}
			sourcePath := filepath.Join(t.TempDir(), "website.md")
			if err := os.WriteFile(sourcePath, []byte(broken), 0o600); err != nil {
				t.Fatalf("write website source with conflicting anchor: %v", err)
			}

			cmd := exec.Command("make", "build",
				"SITE_SOURCE="+sourcePath,
				"SITE_OUTPUT_DIR="+t.TempDir(),
				"SITE_LOCALES=en",
				"SITE_FORMATS=regular",
			)
			cmd.Dir = repo
			combined, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("make build accepted a conflicting downloads anchor")
			}
			if !strings.Contains(string(combined), testCase.diagnostic) {
				t.Fatalf("conflicting-anchor diagnostic is not actionable:\n%s", combined)
			}
		})
	}
}

func TestInvalidSourceRejectsDownloadGroupWithoutLinks(t *testing.T) {
	repo := repositoryRoot(t)
	source, err := os.ReadFile(filepath.Join(repo, "website.md"))
	if err != nil {
		t.Fatalf("read website source: %v", err)
	}
	const groupStart = "      - id: macos\n"
	const linksStart = "        links:\n"
	const nextGroup = "      - id: windows\n"
	groupOffset := strings.Index(string(source), groupStart)
	if groupOffset < 0 {
		t.Fatal("test setup could not find the macOS download group")
	}
	linksOffset := strings.Index(string(source[groupOffset:]), linksStart)
	nextGroupOffset := strings.Index(string(source[groupOffset:]), nextGroup)
	if linksOffset < 0 || nextGroupOffset < 0 || linksOffset >= nextGroupOffset {
		t.Fatal("test setup could not isolate the macOS download links")
	}
	linksOffset += groupOffset
	nextGroupOffset += groupOffset
	broken := string(source[:linksOffset]) + "        links: []\n" + string(source[nextGroupOffset:])
	sourcePath := filepath.Join(t.TempDir(), "website.md")
	if err := os.WriteFile(sourcePath, []byte(broken), 0o600); err != nil {
		t.Fatalf("write website source with empty download group: %v", err)
	}

	cmd := exec.Command("make", "build",
		"SITE_SOURCE="+sourcePath,
		"SITE_OUTPUT_DIR="+t.TempDir(),
		"SITE_LOCALES=en",
		"SITE_FORMATS=regular",
	)
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("make build accepted a download group without links")
	}
	if !strings.Contains(string(combined), "sections[4].download_groups[0].links: at least one link is required") {
		t.Fatalf("empty-download-group diagnostic is not actionable:\n%s", combined)
	}
}

func TestMakeBuildGeneratesEnglishAMPFromSharedSource(t *testing.T) {
	repo := repositoryRoot(t)
	output := t.TempDir()

	cmd := exec.Command("make", "build", "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en", "SITE_FORMATS=regular,amp")
	cmd.Dir = repo
	combined, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}

	page, err := os.ReadFile(filepath.Join(output, "amp", "index.html"))
	if err != nil {
		t.Fatalf("read generated English AMP page: %v", err)
	}
	html := string(page)
	for _, want := range []string{
		`<html amp lang="en">`,
		`<link rel="canonical" href="https://frathe.github.io/picfetch/">`,
		`https://cdn.ampproject.org/v0.js`,
		`custom-element="amp-vimeo"`,
		`custom-element="amp-lightbox-gallery"`,
		`<amp-vimeo data-videoid="1220283616"`,
		`<amp-img src="https://raw.githubusercontent.com/frathe/picfetch/main/assets/screens/main_screen.png" width="520" height="372" layout="responsive" lightbox="screenshots"`,
		`picfetch-linux-arm64.tar.gz`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("generated English AMP page does not contain %q", want)
		}
	}
	if strings.Contains(html, "<iframe") {
		t.Error("generated English AMP page contains a regular iframe")
	}
}

func TestMakeValidateAMPRunsPinnedValidatorOffline(t *testing.T) {
	repo := repositoryRoot(t)
	output := t.TempDir()

	build := exec.Command("make", "build", "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en", "SITE_FORMATS=amp")
	build.Dir = repo
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("make build failed: %v\n%s", err, combined)
	}

	validate := exec.Command("make", "validate-amp", "SITE_OUTPUT_DIR="+output, "SITE_LOCALES=en")
	validate.Dir = repo
	validate.Env = append(os.Environ(), "npm_config_offline=true")
	combined, err := validate.CombinedOutput()
	if err != nil {
		t.Fatalf("offline make validate-amp failed: %v\n%s", err, combined)
	}
	if !strings.Contains(string(combined), filepath.Join(output, "amp", "index.html")+": PASS") {
		t.Fatalf("validator did not report the English AMP artifact passed:\n%s", combined)
	}
}

func TestMakeBuildPublicationFailureRollsBackAllPages(t *testing.T) {
	repo := repositoryRoot(t)
	cachePath := createControlledGermanCache(t, repo)
	output := t.TempDir()
	prior := []byte("prior generated page\n")
	pagePaths := []string{
		filepath.Join("amp", "index.html"),
		filepath.Join("de", "amp", "index.html"),
		filepath.Join("de", "index.html"),
	}
	for _, relative := range pagePaths {
		target := filepath.Join(output, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatalf("create prior page directory: %v", err)
		}
		if err := os.WriteFile(target, prior, 0o600); err != nil {
			t.Fatalf("write prior page %s: %v", relative, err)
		}
	}
	conflictingTarget := filepath.Join(output, "index.html")
	if err := os.Mkdir(conflictingTarget, 0o755); err != nil {
		t.Fatalf("create conflicting output directory: %v", err)
	}

	build := exec.Command("make", "build", "SITE_TRANSLATIONS="+cachePath, "SITE_OUTPUT_DIR="+output)
	build.Dir = repo
	combined, err := build.CombinedOutput()
	if err == nil {
		t.Fatal("make build replaced a directory where a generated page should be")
	}
	if !strings.Contains(string(combined), "publish generated pages") {
		t.Fatalf("publication failure diagnostic is not actionable:\n%s", combined)
	}
	for _, relative := range pagePaths {
		after, err := os.ReadFile(filepath.Join(output, relative))
		if err != nil {
			t.Fatalf("read prior page %s after failed build: %v", relative, err)
		}
		if string(after) != string(prior) {
			t.Errorf("failed build changed prior page %s", relative)
		}
	}
	info, err := os.Stat(conflictingTarget)
	if err != nil {
		t.Fatalf("inspect conflicting target after failed build: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("failed build replaced the conflicting target directory")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate contract test source")
	}
	return filepath.Dir(filepath.Dir(filename))
}
