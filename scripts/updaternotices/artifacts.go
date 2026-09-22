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

const maxPackageBytes = 512 * 1024 * 1024

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
	executable := "picfetch.exe"
	if strings.HasPrefix(filepath.Base(artifact), "picfetch-macos-") {
		prefix = "PicFetch.app/Contents/Resources/"
		executable = "PicFetch.app/Contents/MacOS/picfetch"
	}
	return inspectZip(&reader.Reader, prefix, executable, want)
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
		data, err := readZipEntry(file, maxPackageBytes)
		if err != nil {
			return err
		}
		payload, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return err
		}
		if err := inspectZip(payload, "", "picfetch.exe", want); err != nil {
			return fmt.Errorf("%s: %w", file.Name, err)
		}
	}
	if len(seen) != 2 {
		return fmt.Errorf("bundle must contain both picfetch-x64.msix and picfetch-arm64.msix")
	}
	return nil
}

func inspectZip(reader *zip.Reader, prefix, executable string, want map[string][]byte) error {
	seen := make(map[string]bool)
	executableSeen := false
	for _, file := range reader.File {
		if file.Name == executable {
			if executableSeen || !file.Mode().IsRegular() {
				return fmt.Errorf("duplicate or non-regular executable %s", file.Name)
			}
			if err := inspectZipExecutable(file, want["THIRD-PARTY-NOTICES.md"]); err != nil {
				return err
			}
			executableSeen = true
			continue
		}
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
	if err := requireNotices(seen, want); err != nil {
		return err
	}
	if !executableSeen {
		return fmt.Errorf("missing executable %s", executable)
	}
	return nil
}

func inspectZipExecutable(file *zip.File, want []byte) error {
	if file.UncompressedSize64 > maxPackageBytes {
		return fmt.Errorf("oversized archive entry %s", file.Name)
	}
	r, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	return inspectEmbeddedNotices(r, file.Name, want)
}

func inspectEmbeddedNotices(reader io.Reader, name string, want []byte) error {
	// Keep enough overlap to find the complete notice across read boundaries.
	// Memory depends on the checkout's notice size, never on executable size.
	chunkSize := max(32*1024, len(want))
	buffer := make([]byte, chunkSize+max(0, len(want)-1))
	reader = io.LimitReader(reader, maxPackageBytes+1)
	var total int64
	retained := 0
	found := len(want) == 0
	for {
		n, err := reader.Read(buffer[retained:])
		total += int64(n)
		if total > maxPackageBytes {
			return fmt.Errorf("oversized archive entry %s", name)
		}
		window := buffer[:retained+n]
		if !found {
			found = bytes.Contains(window, want)
		}
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("%s: %w", name, err)
			}
			if !found {
				return fmt.Errorf("missing or stale embedded THIRD-PARTY-NOTICES.md in %s", name)
			}
			return nil
		}
		// Continue to EOF after matching: ZIP checksums may fail on the final read.
		retained = 0
		if !found {
			retained = min(len(want)-1, len(window))
			copy(buffer, window[len(window)-retained:])
		}
	}
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
	executable := strings.TrimSuffix(filepath.Base(artifact), ".tar.gz")
	executableSeen := false
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if header.Name == executable {
			if executableSeen || header.Typeflag != tar.TypeReg {
				return fmt.Errorf("duplicate or non-regular executable %s", header.Name)
			}
			if header.Size > maxPackageBytes {
				return fmt.Errorf("oversized archive entry %s", header.Name)
			}
			if err := inspectEmbeddedNotices(reader, header.Name, want["THIRD-PARTY-NOTICES.md"]); err != nil {
				return err
			}
			executableSeen = true
			continue
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
	if err := requireNotices(seen, want); err != nil {
		return err
	}
	if !executableSeen {
		return fmt.Errorf("missing executable %s", executable)
	}
	return nil
}

func requireNotices(seen map[string]bool, want map[string][]byte) error {
	for name := range want {
		if !seen[name] {
			return fmt.Errorf("missing notice %s", name)
		}
	}
	return nil
}
