package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type extractedPayload struct {
	BinaryPath, PlistPath     string
	BinaryDigest, PlistDigest string
}

// extract consumes the same in-memory bytes that passed release verification.
// The manifest hashes entry streams, never files in the writable stage directory.
func extract(ctx context.Context, archiveName string, data []byte, destDir string) (extractedPayload, error) {
	if err := ctx.Err(); err != nil {
		return extractedPayload{}, err
	}
	digests := make(map[string]string)
	var err error
	switch {
	case strings.HasSuffix(archiveName, ".tar.gz"):
		err = extractTarGz(ctx, data, destDir, digests)
	case strings.HasSuffix(strings.ToLower(archiveName), ".zip"):
		err = extractZip(ctx, data, destDir, digests)
	default:
		return extractedPayload{}, fmt.Errorf("unsupported archive %q", filepath.Base(archiveName))
	}
	if err != nil {
		return extractedPayload{}, err
	}
	return pickPayload(digests)
}

func safeJoin(dest, entry string) (string, error) {
	dest = filepath.Clean(dest)
	entry = strings.TrimSuffix(entry, "/")
	if !filepath.IsLocal(entry) {
		return "", fmt.Errorf("zip slip: %q", entry)
	}
	return filepath.Join(dest, entry), nil
}

func extractZip(ctx context.Context, data []byte, destDir string, digests map[string]string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := extractZipFile(destDir, f, digests); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(destDir string, f *zip.File, digests map[string]string) error {
	name := f.Name
	if name == "" || name == "." {
		return nil
	}
	if f.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink %q", name)
	}
	isDir := f.FileInfo().IsDir() || strings.HasSuffix(name, "/")
	name = strings.TrimSuffix(name, "/")
	if !filepath.IsLocal(name) {
		return fmt.Errorf("zip slip: %q", name)
	}
	target, err := safeJoin(destDir, name)
	if err != nil {
		return err
	}
	if isDir {
		return os.MkdirAll(target, 0o755)
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	return writeExtractedFile(target, f.Mode().Perm(), rc, digests)
}

func extractTarGz(ctx context.Context, data []byte, destDir string, digests map[string]string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
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
		if err := extractTarEntry(destDir, hdr, tr, digests); err != nil {
			return err
		}
	}
}

func extractTarEntry(destDir string, hdr *tar.Header, r io.Reader, digests map[string]string) error {
	name := hdr.Name
	if name == "" || name == "." {
		return nil
	}
	name = strings.TrimSuffix(name, "/")
	if !filepath.IsLocal(name) {
		return fmt.Errorf("zip slip: %q", name)
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
	case tar.TypeReg, '\x00':
		target, err := safeJoin(destDir, name)
		if err != nil {
			return err
		}
		mode := os.FileMode(hdr.Mode).Perm()
		return writeExtractedFile(target, mode, r, digests)
	default:
		return fmt.Errorf("refusing tar entry %q type %v", name, hdr.Typeflag)
	}
}

func writeExtractedFile(path string, mode os.FileMode, r io.Reader, digests map[string]string) error {
	h := sha256.New()
	if err := writeNewFile(path, mode, io.TeeReader(r, h)); err != nil {
		return err
	}
	digests[path] = hex.EncodeToString(h.Sum(nil))
	return nil
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

func pickPayload(digests map[string]string) (extractedPayload, error) {
	var macosBins, winExes, linuxOrBare []string
	paths := make([]string, 0, len(digests))
	for path := range digests {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		name := filepath.Base(path)
		if isMacOSBinary(path) {
			macosBins = append(macosBins, path)
		}
		if name == "picfetch.exe" {
			winExes = append(winExes, path)
		}
		if name == "picfetch" || strings.HasPrefix(name, "picfetch-linux-") {
			linuxOrBare = append(linuxOrBare, path)
		}
	}
	var bin, plist string
	if len(macosBins) > 0 {
		bin = macosBins[0]
		candidate := filepath.Join(filepath.Dir(filepath.Dir(bin)), "Info.plist")
		if _, ok := digests[candidate]; ok {
			plist = candidate
		}
	} else if len(winExes) > 0 {
		bin = winExes[0]
	} else if len(linuxOrBare) == 1 {
		bin = linuxOrBare[0]
	} else {
		return extractedPayload{}, fmt.Errorf("no update payload in archive")
	}
	return extractedPayload{
		BinaryPath: bin, PlistPath: plist,
		BinaryDigest: digests[bin], PlistDigest: digests[plist],
	}, nil
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
