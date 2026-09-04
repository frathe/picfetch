package sitegen

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

func Check(options BuildOptions) error {
	if !sameValues(options.Locales, []string{"en", "de"}) || !sameValues(options.Formats, []string{"regular", "amp"}) {
		return fmt.Errorf("check requires the complete locale/format matrix: locales en,de and formats regular,amp")
	}
	if err := rejectObsoleteGermanTranslations(options.SourcePath, options.TranslationsPath); err != nil {
		return err
	}
	siteBasePath, err := configuredSiteBasePath(options.SourcePath)
	if err != nil {
		return err
	}
	first, err := os.MkdirTemp("", "picfetch-site-check-first-*")
	if err != nil {
		return fmt.Errorf("create first isolated check directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(first) }()
	second, err := os.MkdirTemp("", "picfetch-site-check-second-*")
	if err != nil {
		return fmt.Errorf("create second isolated check directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(second) }()

	firstOptions := options
	firstOptions.OutputPath = first
	if err := Build(firstOptions); err != nil {
		return fmt.Errorf("isolated generated-site build: %w", err)
	}
	secondOptions := options
	secondOptions.OutputPath = second
	if err := Build(secondOptions); err != nil {
		return fmt.Errorf("repeated generated-site build: %w", err)
	}

	paths := generatedPaths(options.Locales, options.Formats)
	if err := rejectUnexpectedGeneratedRoutes(options.OutputPath, paths); err != nil {
		return err
	}
	if err := validateGeneratedLinks(first, options.OutputPath, paths, siteBasePath); err != nil {
		return err
	}
	for _, relative := range paths {
		committedExists, err := pathExistsWithoutSymlinks(options.OutputPath, relative)
		if err != nil {
			return fmt.Errorf("invalid generated artifact %s: %w", relative, err)
		}
		if !committedExists {
			return fmt.Errorf("stale generated artifact: %s (file does not exist)", relative)
		}
		firstData, err := os.ReadFile(filepath.Join(first, relative))
		if err != nil {
			return fmt.Errorf("read isolated artifact %s: %w", relative, err)
		}
		secondData, err := os.ReadFile(filepath.Join(second, relative))
		if err != nil {
			return fmt.Errorf("read repeated artifact %s: %w", relative, err)
		}
		if !bytes.Equal(firstData, secondData) {
			return fmt.Errorf("nondeterministic generated artifact: %s", relative)
		}
		committedData, err := readFileWithoutSymlinks(options.OutputPath, relative)
		if err != nil {
			return fmt.Errorf("stale generated artifact: %s (%w)", relative, err)
		}
		if !bytes.Equal(firstData, committedData) {
			return fmt.Errorf("stale generated artifact: %s", relative)
		}
	}
	return nil
}

func rejectObsoleteGermanTranslations(sourcePath, cachePath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read website source %q for translation-cache validation: %w", sourcePath, err)
	}
	content, err := ParseContent(data)
	if err != nil {
		return err
	}
	units, err := collectTranslationUnits(content)
	if err != nil {
		return err
	}
	cache, err := readTranslationCache(cachePath)
	if err != nil {
		return err
	}
	current := make(map[string]struct{}, len(units))
	for _, unit := range units {
		current[unit.ID] = struct{}{}
	}
	obsolete := make([]string, 0)
	for id := range cache.Entries {
		if _, ok := current[id]; !ok {
			obsolete = append(obsolete, id)
		}
	}
	if len(obsolete) == 0 {
		return nil
	}
	sort.Strings(obsolete)
	return fmt.Errorf("obsolete German translation cache entries: %s; run make translate or make update", strings.Join(obsolete, ", "))
}

func rejectUnexpectedGeneratedRoutes(root string, expectedPaths []string) error {
	expected := make(map[string]struct{}, len(expectedPaths)+1)
	for _, path := range expectedPaths {
		expected[filepath.ToSlash(filepath.Clean(path))] = struct{}{}
	}
	// GitHub Pages serves this ownership-verification token as a standalone
	// static artifact. It is deliberately the only HTML file outside the four
	// generated locale/format routes.
	expected["google3c65c0a6b3b51cc2.html"] = struct{}{}
	openedRoot, err := openSecureReadRoot(root)
	if err != nil {
		return fmt.Errorf("inspect generated routes: %w", err)
	}
	defer func() { _ = openedRoot.handle.Close() }()
	var unexpected []string
	err = fs.WalkDir(openedRoot.handle.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("generated deployment contains symbolic link: %s", strings.TrimPrefix(path, "./"))
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".html") {
			return nil
		}
		relative := strings.TrimPrefix(path, "./")
		if _, ok := expected[relative]; !ok {
			unexpected = append(unexpected, relative)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("inspect generated routes: %w", err)
	}
	if len(unexpected) > 0 {
		sort.Strings(unexpected)
		return fmt.Errorf("unexpected generated route: %s", unexpected[0])
	}
	return nil
}

func configuredSiteBasePath(sourcePath string) (string, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("read website source %q for link validation: %w", sourcePath, err)
	}
	content, err := ParseContent(data)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(content.Site.BaseURL)
	if err != nil {
		return "", fmt.Errorf("parse site base URL for link validation: %w", err)
	}
	return strings.TrimRight(parsed.Path, "/") + "/", nil
}

func validateGeneratedLinks(generatedRoot, deployedRoot string, paths []string, siteBasePath string) error {
	for _, relative := range paths {
		data, err := readFileWithoutSymlinks(generatedRoot, relative)
		if err != nil {
			return fmt.Errorf("open generated page %s for link validation: %w", relative, err)
		}
		document, parseErr := html.Parse(bytes.NewReader(data))
		if parseErr != nil {
			return fmt.Errorf("parse generated page %s for link validation: %w", relative, parseErr)
		}

		ids := make(map[string]struct{})
		var fragments []string
		type localReference struct {
			element            string
			attribute          string
			requireRegularFile bool
			raw                string
			path               string
			fragment           string
		}
		var localReferences []localReference
		var invalidURL string
		var invalidURLReason string
		walkHTML(document, func(node *html.Node) {
			for _, attribute := range node.Attr {
				switch attribute.Key {
				case "id":
					ids[attribute.Val] = struct{}{}
				case "href", "src":
					requireRegularFile := localReferenceRequiresRegularFile(node, attribute.Key)
					parsed, parseErr := url.Parse(attribute.Val)
					if parseErr != nil {
						if invalidURL == "" {
							invalidURL = attribute.Val
							invalidURLReason = "invalid URL"
							if strings.Contains(attribute.Val, "://") {
								invalidURLReason = "invalid external URL"
							}
						}
						continue
					}
					if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
						if invalidURL == "" {
							invalidURL = attribute.Val
							invalidURLReason = "unsupported URL scheme"
						}
						continue
					}
					if parsed.Scheme == "http" || parsed.Scheme == "https" {
						if authorityErr := validateURLAuthority(parsed); authorityErr != nil {
							if invalidURL == "" {
								invalidURL = attribute.Val
								invalidURLReason = "invalid external URL (" + authorityErr.Error() + ")"
							}
							continue
						}
					} else if parsed.Scheme == "" && parsed.Host != "" {
						if invalidURL == "" {
							invalidURL = attribute.Val
							invalidURLReason = "invalid external URL"
						}
						continue
					}
					if !requireRegularFile && parsed.Scheme == "" && parsed.Host == "" && parsed.Path == "" && parsed.Fragment != "" {
						fragments = append(fragments, "#"+parsed.Fragment)
						continue
					}
					if attribute.Val != "" && parsed.Scheme == "" && parsed.Host == "" && (parsed.Path != "" || requireRegularFile) {
						localReferences = append(localReferences, localReference{
							element:            node.Data,
							attribute:          attribute.Key,
							requireRegularFile: requireRegularFile,
							raw:                attribute.Val,
							path:               parsed.Path,
							fragment:           parsed.Fragment,
						})
					}
				}
			}
		})
		if invalidURL != "" {
			return fmt.Errorf("%s %q in %s", invalidURLReason, invalidURL, relative)
		}
		for _, fragment := range fragments {
			if _, ok := ids[strings.TrimPrefix(fragment, "#")]; !ok {
				return fmt.Errorf("broken internal anchor %s in %s", fragment, relative)
			}
		}
		for _, reference := range localReferences {
			referenceKind := "link"
			switch {
			case reference.attribute == "src":
				referenceKind = reference.element + "[src]"
			case reference.element == "link" && reference.requireRegularFile:
				referenceKind = "link[href]"
			}
			if reference.requireRegularFile && reference.path == "" {
				return fmt.Errorf("broken local %s %q in %s: regular-file URL must include a path", referenceKind, reference.raw, relative)
			}
			if reference.requireRegularFile && localFileURLHasDirectoryForm(reference.path) {
				return fmt.Errorf("broken local %s %q in %s: regular-file URL must not end in a slash or dot segment", referenceKind, reference.raw, relative)
			}
			target, err := resolveLocalPath(relative, reference.path, siteBasePath, !reference.requireRegularFile)
			if err != nil {
				return fmt.Errorf("broken local %s %q in %s: %w", referenceKind, reference.raw, relative, err)
			}
			resolvedTarget, generatedExists, err := resolveExistingLocalTarget(generatedRoot, target, reference.requireRegularFile)
			if err != nil {
				return fmt.Errorf("broken local %s %q in %s: %w", referenceKind, reference.raw, relative, err)
			}
			targetRoot := generatedRoot
			targetExists := generatedExists
			if !generatedExists {
				deployedTarget, deployedExists, deployedErr := resolveExistingLocalTarget(deployedRoot, target, reference.requireRegularFile)
				targetRoot = deployedRoot
				targetExists = deployedExists
				err = deployedErr
				if err != nil {
					return fmt.Errorf("broken local %s %q in %s: %w", referenceKind, reference.raw, relative, err)
				}
				// Prefer a directory route's concrete index path in diagnostics,
				// even if neither root contains that index yet.
				if deployedExists || resolvedTarget == target {
					resolvedTarget = deployedTarget
				}
			}
			if !targetExists {
				return fmt.Errorf("broken local %s %q in %s: target %s does not exist", referenceKind, reference.raw, relative, resolvedTarget)
			}
			if reference.fragment != "" {
				if err := validateLinkedFragment(targetRoot, resolvedTarget, reference.fragment); err != nil {
					return fmt.Errorf("%w (referenced from %s)", err, relative)
				}
			}
		}
	}
	return nil
}

func localFileURLHasDirectoryForm(path string) bool {
	lastSlash := strings.LastIndex(path, "/")
	lastSegment := path[lastSlash+1:]
	return lastSegment == "" || lastSegment == "." || lastSegment == ".."
}

func localReferenceRequiresRegularFile(node *html.Node, attribute string) bool {
	if attribute == "src" {
		// Frame sources are page navigations and may use directory routes;
		// other src attributes identify file-valued resources.
		return node.Data != "iframe" && node.Data != "frame" && node.Data != "amp-iframe"
	}
	if attribute != "href" || node.Data != "link" {
		return false
	}
	for _, candidate := range node.Attr {
		if candidate.Key != "rel" {
			continue
		}
		for _, relationship := range strings.Fields(strings.ToLower(candidate.Val)) {
			switch relationship {
			case "apple-touch-icon", "apple-touch-icon-precomposed", "icon", "manifest", "mask-icon", "modulepreload", "preload", "stylesheet":
				return true
			}
		}
	}
	return false
}

func validateLinkedFragment(root, target, fragment string) error {
	data, err := readFileWithoutSymlinks(root, target)
	if err != nil {
		return fmt.Errorf("open linked page %s: %w", target, err)
	}
	document, parseErr := html.Parse(bytes.NewReader(data))
	if parseErr != nil {
		return fmt.Errorf("parse linked page %s: %w", target, parseErr)
	}
	found := false
	walkHTML(document, func(node *html.Node) {
		for _, attribute := range node.Attr {
			if attribute.Key == "id" && attribute.Val == fragment {
				found = true
			}
		}
	})
	if !found {
		return fmt.Errorf("broken linked anchor #%s in %s", fragment, filepath.ToSlash(target))
	}
	return nil
}

func resolveLocalPath(pagePath, reference, siteBasePath string, resolveDirectoryRoute bool) (string, error) {
	var target string
	if strings.HasPrefix(reference, "/") {
		var siteRelativeReference string
		siteRoot := strings.TrimSuffix(siteBasePath, "/")
		switch {
		case siteBasePath == "/":
			siteRelativeReference = strings.TrimPrefix(reference, "/")
		case reference == siteRoot:
			siteRelativeReference = ""
		case strings.HasPrefix(reference, siteBasePath):
			siteRelativeReference = strings.TrimPrefix(reference, siteBasePath)
		default:
			return "", fmt.Errorf("root-relative target %q is outside configured site base %q", reference, siteBasePath)
		}
		target = filepath.FromSlash(strings.TrimLeft(siteRelativeReference, "/"))
	} else {
		target = filepath.Join(filepath.Dir(pagePath), filepath.FromSlash(reference))
	}
	target = filepath.Clean(target)
	if target == ".." || strings.HasPrefix(target, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("target escapes the generated site")
	}
	if resolveDirectoryRoute && (strings.HasSuffix(reference, "/") || target == ".") {
		target = filepath.Join(target, "index.html")
	}
	return target, nil
}

func pathExistsWithoutSymlinks(root, relative string) (bool, error) {
	return localTargetExists(root, relative, false)
}

func localTargetExists(root, relative string, requireRegularFile bool) (bool, error) {
	openedRoot, err := openSecureReadRoot(root)
	if err != nil {
		return false, err
	}
	defer func() { _ = openedRoot.handle.Close() }()
	if err := openedRoot.validateRelativePath(relative); err != nil {
		return false, fmt.Errorf("local-link target %q: %w", relative, err)
	}
	info, err := openedRoot.handle.Lstat(relative)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect local-link target %q: %w", relative, err)
	}
	if requireRegularFile && !info.Mode().IsRegular() {
		return false, fmt.Errorf("target %s is not a regular file", filepath.ToSlash(relative))
	}
	return true, nil
}

func resolveExistingLocalTarget(root, relative string, requireRegularFile bool) (string, bool, error) {
	openedRoot, err := openSecureReadRoot(root)
	if err != nil {
		return relative, false, err
	}
	defer func() { _ = openedRoot.handle.Close() }()
	if err := openedRoot.validateRelativePath(relative); err != nil {
		return relative, false, fmt.Errorf("local-link target %q: %w", relative, err)
	}
	info, err := openedRoot.handle.Lstat(relative)
	if os.IsNotExist(err) {
		return relative, false, nil
	}
	if err != nil {
		return relative, false, fmt.Errorf("inspect local-link target %q: %w", relative, err)
	}

	resolved := relative
	if info.IsDir() && !requireRegularFile {
		resolved = filepath.Join(relative, "index.html")
		if err := openedRoot.validateRelativePath(resolved); err != nil {
			return resolved, false, fmt.Errorf("local-link target %q: %w", resolved, err)
		}
		info, err = openedRoot.handle.Lstat(resolved)
		if os.IsNotExist(err) {
			return resolved, false, nil
		}
		if err != nil {
			return resolved, false, fmt.Errorf("inspect local-link target %q: %w", resolved, err)
		}
	}
	if !info.Mode().IsRegular() {
		return resolved, false, fmt.Errorf("target %s is not a regular file", filepath.ToSlash(resolved))
	}
	return resolved, true, nil
}

func readFileWithoutSymlinks(root, relative string) ([]byte, error) {
	openedRoot, err := openSecureReadRoot(root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = openedRoot.handle.Close() }()
	if err := openedRoot.validateRelativePath(relative); err != nil {
		return nil, err
	}
	return openedRoot.handle.ReadFile(relative)
}

func walkHTML(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walkHTML(child, visit)
	}
}

func generatedPaths(locales, formats []string) []string {
	paths := make([]string, 0, len(locales)*len(formats))
	for _, locale := range locales {
		for _, format := range formats {
			paths = append(paths, routePath(locale, format))
		}
	}
	sort.Strings(paths)
	return paths
}
