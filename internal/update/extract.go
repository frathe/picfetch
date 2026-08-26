package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func extract(ctx context.Context, archivePath, destDir string) (binaryPath, plistPath string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz"):
		err = extractTarGz(ctx, archivePath, destDir)
	case strings.HasSuffix(strings.ToLower(archivePath), ".zip"):
		err = extractZip(ctx, archivePath, destDir)
	default:
		return "", "", fmt.Errorf("unsupported archive %q", filepath.Base(archivePath))
	}
	if err != nil {
		return "", "", err
	}
	return pickPayload(destDir)
}

func safeJoin(dest, entry string) (string, error) {
	dest = filepath.Clean(dest)
	target := filepath.Clean(filepath.Join(dest, entry))
	rel, err := filepath.Rel(dest, target)
	if err != nil {
		return "", fmt.Errorf("zip slip: %q: %w", entry, err)
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("zip slip: %q", entry)
	}
	return target, nil
}

func extractZip(ctx context.Context, zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := extractZipFile(destDir, f); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(destDir string, f *zip.File) error {
	name := f.Name
	if name == "" || name == "." {
		return nil
	}
	if f.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink %q", name)
	}
	target, err := safeJoin(destDir, name)
	if err != nil {
		return err
	}
	if f.FileInfo().IsDir() || strings.HasSuffix(name, "/") {
		return os.MkdirAll(target, 0o755)
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	return writeNewFile(target, f.Mode().Perm(), rc)
}

func extractTarGz(ctx context.Context, path, destDir string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gr.Close() }()
	tr := tar.NewReader(gr)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := extractTarEntry(destDir, hdr, tr); err != nil {
			return err
		}
	}
}

func extractTarEntry(destDir string, hdr *tar.Header, r io.Reader) error {
	name := hdr.Name
	if name == "" || name == "." {
		return nil
	}
	switch hdr.Typeflag {
	case tar.TypeXHeader, tar.TypeXGlobalHeader, tar.TypeGNULongName, tar.TypeGNULongLink:
		return nil
	case tar.TypeDir:
		target, err := safeJoin(destDir, name)
		if err != nil {
			return err
		}
		return os.MkdirAll(target, 0o755)
	case tar.TypeReg, tar.TypeRegA:
		target, err := safeJoin(destDir, name)
		if err != nil {
			return err
		}
		mode := os.FileMode(hdr.Mode).Perm()
		return writeNewFile(target, mode, r)
	default:
		return fmt.Errorf("refusing tar entry %q type %v", name, hdr.Typeflag)
	}
}

func writeNewFile(path string, mode os.FileMode, r io.Reader) error {
	if mode == 0 {
		mode = 0o644
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func pickPayload(dest string) (binaryPath, plistPath string, err error) {
	var macosBins, winExes, linuxOrBare []string
	err = filepath.WalkDir(dest, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if isMacOSBinary(path) {
			macosBins = append(macosBins, path)
		}
		if name == "picfetch.exe" {
			winExes = append(winExes, path)
		}
		if name == "picfetch" || strings.HasPrefix(name, "picfetch-linux-") {
			linuxOrBare = append(linuxOrBare, path)
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}
	if len(macosBins) > 0 {
		bin := macosBins[0]
		plist := filepath.Join(filepath.Dir(filepath.Dir(bin)), "Info.plist")
		if st, err := os.Stat(plist); err == nil && !st.IsDir() {
			return bin, plist, nil
		}
		return bin, "", nil
	}
	if len(winExes) > 0 {
		return winExes[0], "", nil
	}
	if len(linuxOrBare) == 1 {
		return linuxOrBare[0], "", nil
	}
	return "", "", fmt.Errorf("no update payload in archive")
}

func isMacOSBinary(path string) bool {
	if filepath.Base(path) != "picfetch" {
		return false
	}
	macosDir := filepath.Dir(path)
	if filepath.Base(macosDir) != "MacOS" {
		return false
	}
	contentsDir := filepath.Dir(macosDir)
	if filepath.Base(contentsDir) != "Contents" {
		return false
	}
	for dir := filepath.Dir(contentsDir); ; dir = filepath.Dir(dir) {
		if strings.HasSuffix(dir, ".app") {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
	}
}
