package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func inspectArtifact(root, artifact string) error {
	want := make(map[string][]byte)
	for _, name := range []string{"LICENSE", "THIRD-PARTY-NOTICES.md", "PRIVACY.md"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return err
		}
		want[name] = data
	}
	if strings.HasSuffix(artifact, ".tar.gz") {
		return inspectTar(artifact, want)
	}
	reader, err := zip.OpenReader(artifact)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	if strings.HasSuffix(artifact, ".msixbundle") {
		return inspectBundle(&reader.Reader, want)
	}
	prefix := ""
	if strings.HasPrefix(filepath.Base(artifact), "picfetch-macos-") {
		prefix = "PicFetch.app/Contents/Resources/"
	}
	return inspectZip(&reader.Reader, prefix, want)
}

func inspectBundle(reader *zip.Reader, want map[string][]byte) error {
	seen := make(map[string]bool)
	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, ".msix") {
			continue
		}
		if (file.Name != "picfetch-x64.msix" && file.Name != "picfetch-arm64.msix") || seen[file.Name] {
			return fmt.Errorf("unexpected or duplicate bundle payload %s", file.Name)
		}
		seen[file.Name] = true
		// Current payloads are below 100 MiB. Bound allocation when inspecting
		// an invalid archive without extracting any files to the filesystem.
		const maxPackageBytes = 512 * 1024 * 1024
		data, err := readZipEntry(file, maxPackageBytes)
		if err != nil {
			return err
		}
		payload, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return err
		}
		if err := inspectZip(payload, "", want); err != nil {
			return fmt.Errorf("%s: %w", file.Name, err)
		}
	}
	if len(seen) != 2 {
		return fmt.Errorf("bundle must contain both picfetch-x64.msix and picfetch-arm64.msix")
	}
	return nil
}

func inspectZip(reader *zip.Reader, prefix string, want map[string][]byte) error {
	seen := make(map[string]bool)
	for _, file := range reader.File {
		name, ok := strings.CutPrefix(file.Name, prefix)
		if !ok {
			continue
		}
		expected, ok := want[name]
		if !ok {
			continue
		}
		if seen[name] || !file.Mode().IsRegular() {
			return fmt.Errorf("duplicate or non-regular notice %s", file.Name)
		}
		data, err := readZipEntry(file, int64(len(expected)))
		if err != nil {
			return err
		}
		if !bytes.Equal(data, expected) {
			return fmt.Errorf("stale notice %s", file.Name)
		}
		seen[name] = true
	}
	return requireNotices(seen, want)
}

func readZipEntry(file *zip.File, limit int64) ([]byte, error) {
	if file.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("oversized archive entry %s", file.Name)
	}
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("oversized archive entry %s", file.Name)
	}
	return data, nil
}

func inspectTar(artifact string, want map[string][]byte) error {
	file, err := os.Open(artifact)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer func() { _ = compressed.Close() }()
	reader := tar.NewReader(compressed)
	seen := make(map[string]bool)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		expected, ok := want[header.Name]
		if !ok {
			continue
		}
		if seen[header.Name] || header.Typeflag != tar.TypeReg || header.Size != int64(len(expected)) {
			return fmt.Errorf("duplicate, non-regular or wrong-size notice %s", header.Name)
		}
		data, err := io.ReadAll(io.LimitReader(reader, int64(len(expected))+1))
		if err != nil {
			return err
		}
		if !bytes.Equal(data, expected) {
			return fmt.Errorf("stale notice %s", header.Name)
		}
		seen[header.Name] = true
	}
	// Read the gzip trailer as well as the tar payload, so corruption cannot
	// hide behind tar's earlier end-of-archive marker.
	if _, err := io.Copy(io.Discard, compressed); err != nil {
		return err
	}
	return requireNotices(seen, want)
}

func requireNotices(seen map[string]bool, want map[string][]byte) error {
	for name := range want {
		if !seen[name] {
			return fmt.Errorf("missing notice %s", name)
		}
	}
	return nil
}
