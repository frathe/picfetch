package sitegen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type secureWriteRoot struct {
	path   string
	handle *os.Root
}

func openSecureWriteRoot(path string) (*secureWriteRoot, error) {
	return openSecureRoot(path, true)
}

func openSecureReadRoot(path string) (*secureWriteRoot, error) {
	return openSecureRoot(path, false)
}

func openSecureRoot(path string, create bool) (*secureWriteRoot, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve write root %q: %w", path, err)
	}
	anchor, err := trustedPathAnchor(absolutePath)
	if err != nil {
		return nil, err
	}
	current, err := os.OpenRoot(anchor)
	if err != nil {
		return nil, fmt.Errorf("open trusted write anchor %q: %w", anchor, err)
	}
	closeCurrent := true
	defer func() {
		if closeCurrent {
			_ = current.Close()
		}
	}()
	if err := verifyOpenedDirectory(current, anchor); err != nil {
		return nil, err
	}

	relative, err := filepath.Rel(anchor, absolutePath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("write root %q is outside trusted anchor %q", path, anchor)
	}
	currentPath := anchor
	for part := range strings.SplitSeq(relative, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		info, err := current.Lstat(part)
		if os.IsNotExist(err) {
			if !create {
				return nil, fmt.Errorf("open root %q: component %q does not exist", path, filepath.Join(currentPath, part))
			}
			if err := current.Mkdir(part, 0o755); err != nil && !os.IsExist(err) {
				return nil, fmt.Errorf("create write-root component %q: %w", filepath.Join(currentPath, part), err)
			}
			info, err = current.Lstat(part)
		}
		if err != nil {
			return nil, fmt.Errorf("inspect write-root component %q: %w", filepath.Join(currentPath, part), err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("write-root component %q is a symbolic link", filepath.Join(currentPath, part))
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("write-root component %q is not a directory", filepath.Join(currentPath, part))
		}
		next, err := current.OpenRoot(part)
		if err != nil {
			return nil, fmt.Errorf("open write-root component %q: %w", filepath.Join(currentPath, part), err)
		}
		nextPath := filepath.Join(currentPath, part)
		if err := verifyOpenedChild(current, part, next, nextPath); err != nil {
			_ = next.Close()
			return nil, err
		}
		if err := current.Close(); err != nil {
			_ = next.Close()
			return nil, fmt.Errorf("close write-root component %q: %w", currentPath, err)
		}
		current = next
		currentPath = nextPath
	}
	closeCurrent = false
	return &secureWriteRoot{path: absolutePath, handle: current}, nil
}

func verifyOpenedDirectory(root *os.Root, path string) error {
	opened, err := root.Stat(".")
	if err != nil {
		return fmt.Errorf("inspect opened directory %q: %w", path, err)
	}
	current, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect directory path %q: %w", path, err)
	}
	if current.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("directory path %q is a symbolic link", path)
	}
	if !os.SameFile(opened, current) {
		return fmt.Errorf("directory path %q changed while it was opened", path)
	}
	return nil
}

func verifyOpenedChild(parent *os.Root, name string, child *os.Root, displayPath string) error {
	opened, err := child.Stat(".")
	if err != nil {
		return fmt.Errorf("inspect opened directory %q: %w", displayPath, err)
	}
	current, err := parent.Lstat(name)
	if err != nil {
		return fmt.Errorf("reinspect directory %q: %w", displayPath, err)
	}
	if current.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("directory path %q is a symbolic link", displayPath)
	}
	if !os.SameFile(opened, current) {
		return fmt.Errorf("directory path %q changed while it was opened", displayPath)
	}
	return nil
}

func (root *secureWriteRoot) relative(target string) (string, error) {
	absoluteTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve write target %q: %w", target, err)
	}
	relative, err := filepath.Rel(root.path, absoluteTarget)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("write target %q is outside root %q", target, root.path)
	}
	return relative, nil
}

func (root *secureWriteRoot) validateRelativePath(relative string) error {
	current := "."
	parts := strings.Split(relative, string(filepath.Separator))
	for index, part := range parts {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := root.handle.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect write path %q: %w", filepath.Join(root.path, current), err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("write path component %q is a symbolic link", filepath.Join(root.path, current))
		}
		if index < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("write path component %q is not a directory", filepath.Join(root.path, current))
		}
	}
	return nil
}

func createRootTemp(root *secureWriteRoot, directory, prefix string) (*os.File, string, error) {
	for range 100 {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, "", fmt.Errorf("generate temporary filename: %w", err)
		}
		name := filepath.Join(directory, prefix+hex.EncodeToString(random[:]))
		file, err := root.handle.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return file, name, nil
		}
		if !os.IsExist(err) {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("could not allocate a unique temporary filename in %q", filepath.Join(root.path, directory))
}

func trustedPathAnchor(absolutePath string) (string, error) {
	candidates := make([]string, 0, 2)
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine working directory for path validation: %w", err)
	}
	candidates = append(candidates, workingDirectory)
	temporaryDirectory, err := filepath.Abs(os.TempDir())
	if err != nil {
		return "", fmt.Errorf("resolve temporary directory for path validation: %w", err)
	}
	candidates = append(candidates, temporaryDirectory)

	best := ""
	for _, candidate := range candidates {
		candidate, err = filepath.Abs(candidate)
		if err != nil {
			return "", fmt.Errorf("resolve trusted path anchor: %w", err)
		}
		relative, err := filepath.Rel(candidate, absolutePath)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && len(candidate) > len(best) {
			best = candidate
		}
	}
	if best != "" {
		return best, nil
	}
	volume := filepath.VolumeName(absolutePath)
	return volume + string(filepath.Separator), nil
}
