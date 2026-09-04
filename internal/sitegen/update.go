package sitegen

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type UpdateOptions struct {
	Build            BuildOptions
	Endpoint         string
	APIKey           string
	NodeCommand      string
	AMPValidatorPath string
}

// Update refreshes translations, renders the complete site, validates the
// staged output, and only then replaces the committed derived files.
func Update(options UpdateOptions) error {
	if !sameValues(options.Build.Locales, []string{"en", "de"}) || !sameValues(options.Build.Formats, []string{"regular", "amp"}) {
		return fmt.Errorf("update requires the complete locale/format matrix: locales en,de and formats regular,amp")
	}
	stage, err := os.MkdirTemp("", "picfetch-site-update-*")
	if err != nil {
		return fmt.Errorf("create update staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stage) }()

	stagedCache := filepath.Join(stage, "translations", "de.json")
	if err := copyFileIfExists(options.Build.TranslationsPath, stagedCache); err != nil {
		return fmt.Errorf("stage German translation cache: %w", err)
	}
	if err := Translate(TranslateOptions{
		SourcePath: options.Build.SourcePath,
		CachePath:  stagedCache,
		Endpoint:   options.Endpoint,
		APIKey:     options.APIKey,
	}); err != nil {
		return fmt.Errorf("translation refresh failed: %w", err)
	}

	stagedOutput := filepath.Join(stage, "docs")
	stagedBuild := options.Build
	stagedBuild.TranslationsPath = stagedCache
	stagedBuild.OutputPath = stagedOutput
	if err := Build(stagedBuild); err != nil {
		return fmt.Errorf("site generation failed: %w", err)
	}
	paths := generatedPaths(stagedBuild.Locales, stagedBuild.Formats)
	if _, err := os.Lstat(options.Build.OutputPath); err == nil {
		if err := rejectUnexpectedGeneratedRoutes(options.Build.OutputPath, paths); err != nil {
			return fmt.Errorf("generated-site validation failed: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect current generated-site root: %w", err)
	}
	if err := validateGeneratedLinks(stagedOutput, options.Build.OutputPath, paths); err != nil {
		return fmt.Errorf("generated-site validation failed: %w", err)
	}
	if err := validateStagedAMP(options, stagedOutput); err != nil {
		return err
	}

	files := make(map[string][]byte, len(paths)+1)
	cache, err := os.ReadFile(stagedCache)
	if err != nil {
		return fmt.Errorf("read staged German translation cache: %w", err)
	}
	files[options.Build.TranslationsPath] = cache
	for _, relative := range paths {
		data, err := os.ReadFile(filepath.Join(stagedOutput, relative))
		if err != nil {
			return fmt.Errorf("read staged generated page %s: %w", relative, err)
		}
		files[filepath.Join(options.Build.OutputPath, relative)] = data
	}
	if err := commitFileSet(files, options.Build.OutputPath, filepath.Dir(options.Build.TranslationsPath)); err != nil {
		return fmt.Errorf("publish validated website update: %w", err)
	}
	return nil
}

func sameValues(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(got))
	for _, value := range got {
		seen[value]++
	}
	for _, value := range want {
		seen[value]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

func copyFileIfExists(source, destination string) error {
	data, err := os.ReadFile(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return atomicWrite(destination, data)
}

func validateStagedAMP(options UpdateOptions, stagedOutput string) error {
	node := strings.TrimSpace(options.NodeCommand)
	if node == "" {
		node = "node"
	}
	validator := strings.TrimSpace(options.AMPValidatorPath)
	if validator == "" {
		validator = filepath.Join("site", "tools", "validate-amp.cjs")
	}
	arguments := []string{validator}
	for _, locale := range options.Build.Locales {
		arguments = append(arguments, filepath.Join(stagedOutput, routePath(locale, "amp")))
	}
	command := exec.Command(node, arguments...)
	command.Env = environmentWithout("DEEPL_API_KEY")
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message == "" {
			return fmt.Errorf("AMP validation failed: %w", err)
		}
		return fmt.Errorf("AMP validation failed: %w\n%s", err, message)
	}
	return nil
}

func environmentWithout(names ...string) []string {
	environment := os.Environ()
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		keep := true
		for _, name := range names {
			if strings.HasPrefix(entry, name+"=") {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

type preparedFile struct {
	root           *secureWriteRoot
	target         string
	displayTarget  string
	temporary      string
	backup         string
	hadOriginal    bool
	published      bool
	preserveBackup bool
}

func commitFileSet(files map[string][]byte, roots ...string) (resultErr error) {
	committed := false
	targets := make([]string, 0, len(files))
	for target := range files {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	rootPaths := make([]string, 0, len(roots))
	seenRoots := make(map[string]bool, len(roots))
	for _, root := range roots {
		absoluteRoot, err := filepath.Abs(root)
		if err != nil {
			return fmt.Errorf("resolve write root %q: %w", root, err)
		}
		if !seenRoots[absoluteRoot] {
			seenRoots[absoluteRoot] = true
			rootPaths = append(rootPaths, absoluteRoot)
		}
	}
	sort.Strings(rootPaths)
	openedRoots := make(map[string]*secureWriteRoot, len(rootPaths))
	defer func() {
		for _, rootPath := range rootPaths {
			root := openedRoots[rootPath]
			if root == nil {
				continue
			}
			if err := root.handle.Close(); err != nil && !committed {
				resultErr = errors.Join(resultErr, fmt.Errorf("close write root %q: %w", rootPath, err))
			}
		}
	}()
	for _, rootPath := range rootPaths {
		root, err := openSecureWriteRoot(rootPath)
		if err != nil {
			return err
		}
		openedRoots[rootPath] = root
	}

	prepared := make([]preparedFile, 0, len(targets))
	cleanup := func() error {
		var cleanupErrors []error
		for index := range prepared {
			file := &prepared[index]
			if file.temporary != "" {
				if err := file.root.handle.Remove(file.temporary); err != nil && !os.IsNotExist(err) {
					cleanupErrors = append(cleanupErrors, fmt.Errorf("remove temporary file for %q: %w", file.displayTarget, err))
				}
			}
			if file.backup != "" && !file.preserveBackup {
				if err := file.root.handle.Remove(file.backup); err != nil && !os.IsNotExist(err) {
					cleanupErrors = append(cleanupErrors, fmt.Errorf("remove backup file for %q: %w", file.displayTarget, err))
				}
			}
		}
		return errors.Join(cleanupErrors...)
	}
	defer func() {
		if err := cleanup(); err != nil && !committed {
			resultErr = errors.Join(resultErr, fmt.Errorf("clean up website update files: %w", err))
		}
	}()

	for _, target := range targets {
		rootPath, err := containingWriteRoot(target, roots)
		if err != nil {
			return err
		}
		root := openedRoots[rootPath]
		relativeTarget, err := root.relative(target)
		if err != nil {
			return err
		}
		if err := root.validateRelativePath(relativeTarget); err != nil {
			return err
		}
		relativeDirectory := filepath.Dir(relativeTarget)
		if err := root.handle.MkdirAll(relativeDirectory, 0o755); err != nil {
			return fmt.Errorf("create parent directory for %q: %w", target, err)
		}
		if err := root.validateRelativePath(relativeTarget); err != nil {
			return err
		}
		temporary, temporaryName, err := createRootTemp(root, relativeDirectory, ".sitegen-update-")
		if err != nil {
			return fmt.Errorf("prepare replacement for %q: %w", target, err)
		}
		entry := preparedFile{root: root, target: relativeTarget, displayTarget: target, temporary: temporaryName}
		prepared = append(prepared, entry)
		if _, err := temporary.Write(files[target]); err != nil {
			_ = temporary.Close()
			return fmt.Errorf("write replacement for %q: %w", target, err)
		}
		if err := temporary.Chmod(0o644); err != nil {
			_ = temporary.Close()
			return fmt.Errorf("set replacement permissions for %q: %w", target, err)
		}
		if err := temporary.Close(); err != nil {
			return fmt.Errorf("close replacement for %q: %w", target, err)
		}
	}

	rollback := func(last int) error {
		var rollbackErrors []error
		for index := last; index >= 0; index-- {
			file := &prepared[index]
			var removeErr error
			if file.published {
				if err := file.root.handle.Remove(file.target); err != nil && !os.IsNotExist(err) {
					removeErr = fmt.Errorf("remove published target %q: %w", file.displayTarget, err)
				}
			}
			if file.hadOriginal && file.backup != "" {
				if err := file.root.handle.Rename(file.backup, file.target); err != nil {
					file.preserveBackup = true
					rollbackErrors = append(rollbackErrors, removeErr, fmt.Errorf("restore backup for %q: %w", file.displayTarget, err))
					continue
				}
				file.backup = ""
				file.hadOriginal = false
				file.published = false
				continue
			}
			if removeErr != nil {
				rollbackErrors = append(rollbackErrors, removeErr)
			} else {
				file.published = false
			}
		}
		return errors.Join(rollbackErrors...)
	}
	rollbackFailure := func(cause error, last int) error {
		if err := rollback(last); err != nil {
			return errors.Join(cause, fmt.Errorf("website update rollback failed: %w", err))
		}
		return cause
	}

	for index := range prepared {
		file := &prepared[index]
		if err := file.root.validateRelativePath(file.target); err != nil {
			return rollbackFailure(err, index-1)
		}
		if info, err := file.root.handle.Lstat(file.target); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return rollbackFailure(fmt.Errorf("target %q is a symbolic link", file.displayTarget), index-1)
			}
			if info.IsDir() {
				return rollbackFailure(fmt.Errorf("target %q is a directory", file.displayTarget), index-1)
			}
			backupFile, backupName, err := createRootTemp(file.root, filepath.Dir(file.target), ".sitegen-backup-")
			if err != nil {
				return rollbackFailure(fmt.Errorf("prepare backup for %q: %w", file.displayTarget, err), index-1)
			}
			file.backup = backupName
			if err := backupFile.Close(); err != nil {
				return rollbackFailure(fmt.Errorf("close backup placeholder for %q: %w", file.displayTarget, err), index-1)
			}
			if err := file.root.handle.Remove(file.backup); err != nil {
				return rollbackFailure(fmt.Errorf("remove backup placeholder for %q: %w", file.displayTarget, err), index-1)
			}
			if err := file.root.handle.Rename(file.target, file.backup); err != nil {
				return rollbackFailure(fmt.Errorf("back up %q: %w", file.displayTarget, err), index-1)
			}
			file.hadOriginal = true
		} else if !os.IsNotExist(err) {
			return rollbackFailure(fmt.Errorf("inspect existing target %q: %w", file.displayTarget, err), index-1)
		}
		if err := file.root.handle.Rename(file.temporary, file.target); err != nil {
			return rollbackFailure(fmt.Errorf("replace %q: %w", file.displayTarget, err), index)
		}
		file.temporary = ""
		file.published = true
	}
	committed = true
	for index := range prepared {
		if prepared[index].backup != "" {
			if err := prepared[index].root.handle.Remove(prepared[index].backup); err != nil && !os.IsNotExist(err) {
				continue
			}
			prepared[index].backup = ""
		}
	}
	return nil
}

func containingWriteRoot(target string, roots []string) (string, error) {
	absoluteTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve write target %q: %w", target, err)
	}
	best := ""
	for _, root := range roots {
		absoluteRoot, err := filepath.Abs(root)
		if err != nil {
			return "", fmt.Errorf("resolve write root %q: %w", root, err)
		}
		relative, err := filepath.Rel(absoluteRoot, absoluteTarget)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		if len(absoluteRoot) > len(best) {
			best = absoluteRoot
		}
	}
	if best == "" {
		return "", fmt.Errorf("write target %q is outside the configured output roots", target)
	}
	return best, nil
}
