package update

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"

	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
)

// Package roots are derived from the authenticated main executable, never
// supplied by a companion manifest. macOS installs the complete signed bundle;
// other platforms replace only the dedicated helper directory beside the app.
func companionRoots(binary, system string) (installation, companions string, err error) {
	if !filepath.IsAbs(binary) {
		return "", "", errors.New("update package binary is not absolute")
	}
	installation = filepath.Dir(binary)
	if system == "darwin" {
		if filepath.Base(installation) != "MacOS" || filepath.Base(filepath.Dir(installation)) != "Contents" {
			return "", "", errors.New("update package is not a macOS app bundle")
		}
		installation = filepath.Dir(filepath.Dir(installation))
		if filepath.Ext(installation) != ".app" {
			return "", "", errors.New("update package has no app bundle root")
		}
		return installation, installation, nil
	}
	if system != "linux" && system != "windows" {
		return "", "", errors.New("unsupported update companion platform")
	}
	return installation, filepath.Join(installation, "heic"), nil
}

func captureCompanions(binary, system, architecture string) (map[string]string, error) {
	installation, root, err := companionRoots(binary, system)
	if err != nil {
		// Legacy stages can have no helper or .app layout; their existing binary
		// and plist verification remains authoritative until a complete package.
		if system == "darwin" {
			return nil, nil
		}
		return nil, err
	}
	_, manifest, err := heicclient.PackagePaths(installation, system)
	if err != nil {
		return nil, err
	}
	if _, err = os.Lstat(manifest); errors.Is(err, os.ErrNotExist) {
		helper, _, pathErr := heicclient.PackagePaths(installation, system)
		if pathErr != nil {
			return nil, pathErr
		}
		if _, helperErr := os.Lstat(helper); errors.Is(helperErr, os.ErrNotExist) {
			return nil, nil
		}
		return nil, errors.New("update helper has no manifest")
	} else if err != nil {
		return nil, err
	}
	if _, _, err = heicclient.LoadPackage(installation, system, architecture); err != nil {
		return nil, fmt.Errorf("update helper package: %w", err)
	}
	return companionDigests(root)
}

func companionDigests(root string) (map[string]string, error) {
	files := make(map[string]string)
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("update companion is a symbolic link: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() < 0 {
			return errors.New("update companion is not a regular file")
		}
		total += info.Size()
		if len(files) >= 4096 || total > 256*1024*1024 {
			return errors.New("update companion package exceeds finite limits")
		}
		name, err := filepath.Rel(root, path)
		if err != nil || !filepath.IsLocal(name) {
			return errors.New("invalid update companion path")
		}
		digest, err := fileSHA256(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(name)] = digest
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("empty update companion package")
	}
	return files, nil
}

func validateCompanions(stage Stage) error {
	expected := stage.verification.CompanionDigests
	if len(expected) == 0 {
		actual, err := captureCompanions(stage.BinaryPath, stage.verification.GOOS, stage.verification.GOARCH)
		if err != nil {
			return err
		}
		if len(actual) != 0 {
			return errors.New("update helper package has no verified companion digests")
		}
		return nil
	}
	_, root, err := companionRoots(stage.BinaryPath, stage.verification.GOOS)
	if err != nil {
		return err
	}
	actual, err := companionDigests(root)
	if err != nil {
		return err
	}
	if !maps.Equal(actual, expected) {
		return errors.New("update companion package changed after archive verification")
	}
	return nil
}
