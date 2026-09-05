package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeZip(t *testing.T, dir, name string, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTarGz(t *testing.T, dir, name string, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// archiveEntry is one ordered member of an archive fixture. A non-empty
// linkTarget makes it a symlink entry, or a hard link when hard is set;
// otherwise it is a regular file holding body. Ordering matters for hostile
// fixtures, where a link entry has to precede the entry that would be written
// through it, so this cannot use writeZip/writeTarGz's name-keyed maps.
type archiveEntry struct {
	name       string
	body       []byte
	linkTarget string
	hard       bool
}

func zipArchiveBytes(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		body := e.body
		if e.linkTarget != "" {
			hdr.SetMode(os.ModeSymlink | 0o777)
			body = []byte(e.linkTarget)
		} else {
			hdr.SetMode(0o755)
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func tarGzArchiveBytes(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: 0o755, Typeflag: tar.TypeReg, Size: int64(len(e.body))}
		if e.linkTarget != "" {
			hdr.Typeflag = tar.TypeSymlink
			if e.hard {
				hdr.Typeflag = tar.TypeLink
			}
			hdr.Linkname = e.linkTarget
			hdr.Size = 0
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if hdr.Size > 0 {
			if _, err := tw.Write(e.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtract_MacOSApp(t *testing.T) {
	zipPath := writeZip(t, t.TempDir(), "picfetch-macos-arm64.zip", map[string][]byte{
		"PicFetch.app/Contents/MacOS/picfetch": []byte("newbin"),
		"PicFetch.app/Contents/Info.plist":     []byte("plist"),
	})
	dest := t.TempDir()
	bin, plist, err := extract(context.Background(), zipPath, dest)
	if err != nil {
		t.Fatal(err)
	}
	gotBin, err := os.ReadFile(filepath.Join(dest, "PicFetch.app", "Contents", "MacOS", "picfetch"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBin) != "newbin" {
		t.Errorf("binary contents = %q, want newbin", gotBin)
	}
	gotPlist, err := os.ReadFile(filepath.Join(dest, "PicFetch.app", "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotPlist) != "plist" {
		t.Errorf("plist contents = %q, want plist", gotPlist)
	}
	if bin != filepath.Join(dest, "PicFetch.app", "Contents", "MacOS", "picfetch") {
		t.Errorf("BinaryPath = %q", bin)
	}
	if plist != filepath.Join(dest, "PicFetch.app", "Contents", "Info.plist") {
		t.Errorf("PlistPath = %q", plist)
	}
}

func TestSafeJoin(t *testing.T) {
	dest := t.TempDir()
	tests := []struct {
		entry   string
		wantErr bool
	}{
		{entry: "picfetch.exe", wantErr: false},
		{entry: "PicFetch.app/Contents/MacOS/picfetch", wantErr: false},
		{entry: "PicFetch.app/", wantErr: false},
		{entry: "..foo", wantErr: false},
		{entry: "../escape", wantErr: true},
		{entry: "/tmp/x", wantErr: true},
		{entry: "foo/../../etc/passwd", wantErr: true},
		{entry: "..", wantErr: true},
		{entry: "", wantErr: true},
	}
	for _, tc := range tests {
		got, err := safeJoin(dest, tc.entry)
		if tc.wantErr {
			if err == nil {
				t.Errorf("safeJoin(%q) = %q, want error", tc.entry, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("safeJoin(%q): %v", tc.entry, err)
			continue
		}
		rel, err := filepath.Rel(dest, got)
		if err != nil || !filepath.IsLocal(rel) {
			t.Errorf("safeJoin(%q) = %q, escapes dest", tc.entry, got)
		}
	}
}

func TestExtract_ZipSlipDotDot(t *testing.T) {
	dest := t.TempDir()
	parent := filepath.Dir(dest)
	outside := filepath.Join(parent, "escape")
	zipPath := writeZip(t, t.TempDir(), "slip.zip", map[string][]byte{
		"../escape": []byte("nope"),
	})
	_, _, err := extract(context.Background(), zipPath, dest)
	if err == nil {
		t.Fatal("want zip-slip error")
	}
	if !strings.Contains(err.Error(), "zip slip") {
		t.Fatalf("err = %v, want zip slip", err)
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("wrote ../escape outside destDir")
	}
}

func TestExtract_TarSlipDotDot(t *testing.T) {
	dest := t.TempDir()
	parent := filepath.Dir(dest)
	outside := filepath.Join(parent, "escape")
	tarPath := writeTarGz(t, t.TempDir(), "slip.tar.gz", map[string][]byte{
		"../escape": []byte("nope"),
	})
	_, _, err := extract(context.Background(), tarPath, dest)
	if err == nil {
		t.Fatal("want zip-slip error")
	}
	if !strings.Contains(err.Error(), "zip slip") {
		t.Fatalf("err = %v, want zip slip", err)
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("wrote ../escape outside destDir")
	}
}

func TestExtract_ZipSlipAbsolute(t *testing.T) {
	dest := t.TempDir()
	outside := filepath.Join("/tmp", "x")
	_, existed := os.Stat(outside)
	zipPath := writeZip(t, t.TempDir(), "slip.zip", map[string][]byte{
		"/tmp/x": []byte("nope"),
	})
	_, _, err := extract(context.Background(), zipPath, dest)
	if err == nil {
		t.Fatal("want zip-slip error")
	}
	if !strings.Contains(err.Error(), "zip slip") {
		t.Fatalf("err = %v, want zip slip", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "tmp", "x")); err == nil {
		t.Fatal("extracted absolute name into dest")
	}
	if existed != nil {
		if _, err := os.Stat(outside); err == nil {
			t.Fatal("wrote /tmp/x outside destDir")
		}
	} else if b, err := os.ReadFile(outside); err == nil && string(b) == "nope" {
		_ = os.Remove(outside)
		t.Fatal("overwrote /tmp/x")
	}
}

func TestExtract_TarSlipAbsolute(t *testing.T) {
	dest := t.TempDir()
	outside := filepath.Join("/tmp", "x")
	_, existed := os.Stat(outside)
	tarPath := writeTarGz(t, t.TempDir(), "slip.tar.gz", map[string][]byte{
		"/tmp/x": []byte("nope"),
	})
	_, _, err := extract(context.Background(), tarPath, dest)
	if err == nil {
		t.Fatal("want zip-slip error")
	}
	if !strings.Contains(err.Error(), "zip slip") {
		t.Fatalf("err = %v, want zip slip", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "tmp", "x")); err == nil {
		t.Fatal("extracted absolute name into dest")
	}
	if existed != nil {
		if _, err := os.Stat(outside); err == nil {
			t.Fatal("wrote /tmp/x outside destDir")
		}
	} else if b, err := os.ReadFile(outside); err == nil && string(b) == "nope" {
		_ = os.Remove(outside)
		t.Fatal("overwrote /tmp/x")
	}
}

func TestExtract_LinuxTarball(t *testing.T) {
	tarPath := writeTarGz(t, t.TempDir(), "picfetch-linux-amd64.tar.gz", map[string][]byte{
		"picfetch-linux-amd64": []byte("elf"),
	})
	dest := t.TempDir()
	bin, plist, err := extract(context.Background(), tarPath, dest)
	if err != nil {
		t.Fatal(err)
	}
	if plist != "" {
		t.Errorf("PlistPath = %q, want empty", plist)
	}
	if filepath.Base(bin) != "picfetch-linux-amd64" {
		t.Errorf("BinaryPath = %q, want picfetch-linux-amd64", bin)
	}
	got, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "elf" {
		t.Errorf("contents = %q, want elf", got)
	}
}

func TestExtractTarEntry_NULTypeflag(t *testing.T) {
	dest := t.TempDir()
	const body = "elf"
	hdr := &tar.Header{
		Name:     "picfetch-linux-amd64",
		Mode:     0o755,
		Size:     int64(len(body)),
		Typeflag: 0,
	}
	if err := extractTarEntry(dest, hdr, strings.NewReader(body)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, hdr.Name))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Errorf("contents = %q, want %q", got, body)
	}
}

func TestExtract_WindowsZip(t *testing.T) {
	zipPath := writeZip(t, t.TempDir(), "picfetch-windows-amd64.zip", map[string][]byte{
		"picfetch.exe": []byte("mz"),
	})
	dest := t.TempDir()
	bin, plist, err := extract(context.Background(), zipPath, dest)
	if err != nil {
		t.Fatal(err)
	}
	if plist != "" {
		t.Errorf("PlistPath = %q, want empty", plist)
	}
	if filepath.Base(bin) != "picfetch.exe" {
		t.Errorf("BinaryPath = %q, want picfetch.exe", bin)
	}
	got, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "mz" {
		t.Errorf("contents = %q, want mz", got)
	}
}

func TestExtract_NoPayload(t *testing.T) {
	zipPath := writeZip(t, t.TempDir(), "empty.zip", map[string][]byte{
		"README.txt": []byte("no"),
	})
	_, _, err := extract(context.Background(), zipPath, t.TempDir())
	if err == nil {
		t.Fatal("want error when no payload matches")
	}
}
