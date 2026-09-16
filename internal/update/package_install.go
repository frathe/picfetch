package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

type directorySwap struct {
	destination string
	backup      string
	hadPrevious bool
}

// installDirectory prepares and verifies every file before renaming any
// installed directory. All temporary/backup directories share its filesystem.
func installDirectory(source, destination string, expected map[string]string, rename func(string, string) error) (*directorySwap, error) {
	if len(expected) == 0 {
		return nil, errors.New("update directory has no verified files")
	}
	parent := filepath.Dir(destination)
	prepared, err := os.MkdirTemp(parent, ".picfetch-package-new-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(prepared) }()
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		local := filepath.FromSlash(name)
		if !filepath.IsLocal(local) || filepath.ToSlash(filepath.Clean(local)) != name || !validSHA256(expected[name]) {
			return nil, errors.New("invalid verified companion path or digest")
		}
		src, target := filepath.Join(source, local), filepath.Join(prepared, local)
		info, statErr := os.Lstat(src)
		if statErr != nil {
			return nil, statErr
		}
		if !info.Mode().IsRegular() {
			return nil, errors.New("update companion source is not a regular file")
		}
		if err = copyFile(src, target); err != nil {
			return nil, err
		}
		if err = os.Chmod(target, info.Mode().Perm()); err != nil {
			return nil, err
		}
		if err = verifyFileSHA256(target, expected[name]); err != nil {
			return nil, err
		}
	}
	// A package root must remain traversable by its existing consumers, including
	// an AppContainer with explicit read/execute provisioning after installation.
	if err = os.Chmod(prepared, 0755); err != nil {
		return nil, err
	}
	backup, err := os.MkdirTemp(parent, ".picfetch-package-old-")
	if err != nil {
		return nil, err
	}
	if err = os.Remove(backup); err != nil {
		return nil, err
	}
	transaction := &directorySwap{destination: destination, backup: backup}
	if info, statErr := os.Lstat(destination); statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("installed companion root is not an ordinary directory")
		}
		if err = rename(destination, backup); err != nil {
			return nil, err
		}
		transaction.hadPrevious = true
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, statErr
	}
	if err = rename(prepared, destination); err != nil {
		if transaction.hadPrevious {
			err = errors.Join(err, rename(backup, destination))
		}
		return nil, err
	}
	return transaction, nil
}

func (s *directorySwap) rollback() error {
	if err := os.RemoveAll(s.destination); err != nil {
		return err
	}
	if s.hadPrevious {
		return os.Rename(s.backup, s.destination)
	}
	return nil
}

func (s *directorySwap) commit() {
	if s.hadPrevious {
		_ = os.RemoveAll(s.backup)
	}
}

func applyWithCompanions(stage Stage, destination string, apply func() error) error {
	if len(stage.verification.CompanionDigests) == 0 || stage.verification.GOOS == "darwin" {
		return apply()
	}
	if err := validateCompanions(stage); err != nil {
		return err
	}
	_, source, err := companionRoots(stage.BinaryPath, stage.verification.GOOS)
	if err != nil {
		return err
	}
	destination, err = filepath.EvalSymlinks(destination)
	if err != nil {
		return err
	}
	_, target, err := companionRoots(destination, stage.verification.GOOS)
	if err != nil {
		return err
	}
	transaction, err := installDirectory(source, target, stage.verification.CompanionDigests, os.Rename)
	if err != nil {
		return &ApplyError{Op: "companions", Path: target, Err: err}
	}
	if err = apply(); err != nil {
		// Relaunch errors occur after the executable has committed. Its companion
		// package must stay paired with that newly installed executable.
		if failure, ok := errors.AsType[*ApplyError](err); ok && failure.Op == "relaunch" {
			transaction.commit()
			return err
		}
		if rollbackErr := transaction.rollback(); rollbackErr != nil {
			return &ApplyError{Op: "restore", Path: target, Err: errors.Join(err, rollbackErr)}
		}
		return err
	}
	transaction.commit()
	return nil
}

func applyMacBundle(stage Stage, destination string, options ApplyOptions, launch func(string) error) error {
	if err := validateCompanions(stage); err != nil {
		return err
	}
	_, source, err := companionRoots(stage.BinaryPath, "darwin")
	if err != nil {
		return err
	}
	destination, err = filepath.EvalSymlinks(destination)
	if err != nil {
		return err
	}
	_, target, err := companionRoots(destination, "darwin")
	if err != nil {
		return err
	}
	transaction, err := installDirectory(source, target, stage.verification.CompanionDigests, os.Rename)
	if err != nil {
		return fmt.Errorf("install verified app bundle: %w", err)
	}
	transaction.commit()
	if options.Relaunch {
		if err = launch(destination); err != nil {
			return &ApplyError{Op: "relaunch", Path: destination, Err: err}
		}
	}
	return nil
}
