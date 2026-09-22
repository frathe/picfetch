package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestArtifactRejectsMissingNotices(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "picfetch-windows-amd64.zip")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := inspectArtifact(filepath.Join("..", ".."), archive); err == nil {
		t.Fatal("accepted an archive without any notices")
	}
}

func TestNoticesRejectUnreviewedChanges(t *testing.T) {
	dir := t.TempDir()
	license := []byte("Copyright Example\nLicense conditions\n")
	writeTestFile(t, filepath.Join(dir, "LICENSE"), license)
	entry := noticeModule{
		Module: "example.org/Module", Version: "v1.0.0-RC1", Source: "https://proxy.golang.org/example.org/!module/@v/v1.0.0-!r!c1.zip", License: "MIT", Packages: []string{"example.org/Module"},
		Files: []noticeFile{{Path: "LICENSE", SHA256: fmt.Sprintf("%x", sha256.Sum256(license))}},
	}
	modules := map[string]goModule{entry.Module: {Version: entry.Version, Dir: dir, packages: entry.Packages}}
	good, err := renderNotices(dir, []noticeModule{entry}, modules)
	if err != nil || !bytes.Contains(good, license) {
		t.Fatalf("complete original notice missing: %v", err)
	}
	for _, link := range regexp.MustCompile(`]\(#updater-text-([a-f0-9]+)\)`).FindAllSubmatch(good, -1) {
		heading := []byte("### Updater text " + string(link[1]) + "\n")
		if !bytes.Contains(good, heading) {
			t.Fatalf("generated license link %s has no matching text heading", link[0])
		}
	}
	for _, change := range []string{"version", "missing module", "new module", "new package", "duplicate module", "license bytes", "missing license", "bad range", "stale source", "wrong source module", "unversioned source", "unescaped source", "missing linked text"} {
		t.Run(change, func(t *testing.T) {
			candidate := entry
			candidate.Files = slices.Clone(entry.Files)
			inventory := []noticeModule{candidate}
			switch change {
			case "version":
				inventory[0].Version = "v2.0.0"
			case "missing module":
				inventory = nil
			case "new module":
				inventory[0].Module = "example.org/unreviewed"
			case "new package":
				inventory[0].Packages = []string{"example.org/module/unreviewed"}
			case "duplicate module":
				inventory = append(inventory, candidate)
			case "license bytes":
				inventory[0].Files[0].SHA256 = strings.Repeat("0", 64)
			case "missing license":
				inventory[0].Files = nil
			case "bad range":
				inventory[0].Files[0].Start = 5
				inventory[0].Files[0].End = 2
			case "stale source":
				inventory[0].Source = "https://proxy.golang.org/example.org/!module/@v/v0.9.0.zip"
			case "wrong source module":
				inventory[0].Source = "https://proxy.golang.org/example.org/other/@v/v1.0.0-!r!c1.zip"
			case "unversioned source":
				inventory[0].Source = "https://example.org/Module"
			case "unescaped source":
				inventory[0].Source = "https://proxy.golang.org/example.org/Module/@v/v1.0.0-RC1.zip"
			case "missing linked text":
				inventory[0].Notes = "See the [shared Go BSD license](#updater-text-911f8f578293)."
			}
			if _, err := renderNotices(dir, inventory, modules); err == nil {
				t.Fatalf("accepted %s without a new license review", change)
			}
		})
	}
	before := []byte("existing non-updater notices\n")
	updated, err := replaceSection(before, good)
	if err != nil || !bytes.HasPrefix(updated, before) {
		t.Fatalf("lost existing notices: %v", err)
	}
	if again, err := replaceSection(updated, good); err != nil || !bytes.Equal(again, updated) {
		t.Fatalf("generation is not stable: %v", err)
	}
	broken := bytes.Replace(updated, []byte("License conditions"), []byte("removed conditions"), 1)
	if repaired, err := replaceSection(broken, good); err != nil || bytes.Equal(repaired, broken) || !bytes.Equal(repaired, updated) {
		t.Fatalf("stale license text was not detected: %v", err)
	}
}

type archiveMember struct {
	name string
	data []byte
	mode os.FileMode
}

func artifactNoticeFixture(t *testing.T) (string, []archiveMember) {
	t.Helper()
	root := t.TempDir()
	var members []archiveMember
	for _, name := range []string{"LICENSE", "THIRD-PARTY-NOTICES.md", "PRIVACY.md"} {
		data := []byte("current text: " + name)
		writeTestFile(t, filepath.Join(root, name), data)
		members = append(members, archiveMember{name: name, data: data})
	}
	return root, members
}

func artifactEntries(format string, notices []archiveMember, executable []byte) []archiveMember {
	entries := slices.Clone(notices)
	name := "picfetch.exe"
	if strings.Contains(format, "macos") {
		for i := range entries {
			entries[i].name = "PicFetch.app/Contents/Resources/" + entries[i].name
		}
		name = "PicFetch.app/Contents/MacOS/picfetch"
	} else if before, ok := strings.CutSuffix(format, ".tar.gz"); ok {
		name = before
	}
	return append(entries, archiveMember{name: name, data: executable})
}

func TestArtifactNoticeDelivery(t *testing.T) {
	root, members := artifactNoticeFixture(t)
	executable := append([]byte("executable bytes\x00"), members[1].data...)
	for _, format := range []string{"picfetch-windows-amd64.zip", "picfetch-macos-arm64.zip", "picfetch-linux-arm64.tar.gz", "picfetch-x64.msix"} {
		for _, change := range []string{"valid", "missing", "stale", "duplicate", "wrong directory"} {
			t.Run(format+"/"+change, func(t *testing.T) {
				entries := slices.Clone(members)
				switch change {
				case "missing":
					entries = entries[:2]
				case "stale":
					entries[1].data = []byte("outdated text")
				case "duplicate":
					entries = append(entries, entries[1])
				case "wrong directory":
					entries[1].name = "elsewhere/" + entries[1].name
				}
				entries = artifactEntries(format, entries, executable)
				path := filepath.Join(t.TempDir(), format)
				writeTestFile(t, path, archiveBytes(t, entries, strings.HasSuffix(format, ".tar.gz")))
				err := inspectArtifact(root, path)
				if (err == nil) != (change == "valid") {
					t.Fatalf("artifact check = %v", err)
				}
			})
		}
	}
	for _, change := range []string{"valid", "missing arch", "stale arm64", "duplicate arch", "unexpected arch"} {
		t.Run("bundle/"+change, func(t *testing.T) {
			payload := archiveBytes(t, artifactEntries("picfetch-x64.msix", members, executable), false)
			entries := []archiveMember{{name: "picfetch-x64.msix", data: payload}, {name: "picfetch-arm64.msix", data: payload}}
			switch change {
			case "missing arch":
				entries = entries[:1]
			case "stale arm64":
				bad := slices.Clone(members)
				bad[1].data = []byte("stale")
				entries[1].data = archiveBytes(t, artifactEntries("picfetch-arm64.msix", bad, executable), false)
			case "duplicate arch":
				entries = append(entries, entries[0])
			case "unexpected arch":
				entries[1].name = "picfetch-arm.msix"
			}
			path := filepath.Join(t.TempDir(), "picfetch-microsoft-store.msixbundle")
			writeTestFile(t, path, archiveBytes(t, entries, false))
			err := inspectArtifact(root, path)
			if (err == nil) != (change == "valid") {
				t.Fatalf("bundle check = %v", err)
			}
		})
	}
}

func TestArtifactEmbeddedNotices(t *testing.T) {
	root, notices := artifactNoticeFixture(t)
	for _, format := range []string{"picfetch-windows-amd64.zip", "picfetch-macos-arm64.zip", "picfetch-linux-arm64.tar.gz", "picfetch-x64.msix", "bundle x64", "bundle arm64"} {
		for _, change := range []string{"valid", "missing executable", "missing embedding", "stale embedding", "partial embedding", "duplicate executable", "wrong directory", "non-regular executable"} {
			t.Run(format+"/"+change, func(t *testing.T) {
				executable := append([]byte("executable prefix\x00"), notices[1].data...)
				executable = append(executable, []byte("\x00executable suffix")...)
				entries := artifactEntries(format, notices, executable)
				index := len(entries) - 1
				switch change {
				case "missing executable":
					entries = entries[:index]
				case "missing embedding":
					entries[index].data = []byte("executable without notices")
				case "stale embedding":
					entries[index].data = bytes.Replace(executable, []byte("current"), []byte("outdated"), 1)
				case "partial embedding":
					entries[index].data = notices[1].data[:len(notices[1].data)-1]
				case "duplicate executable":
					entries = append(entries, entries[index])
				case "wrong directory":
					entries[index].name = "elsewhere/" + entries[index].name
				case "non-regular executable":
					entries[index].mode = os.ModeSymlink | 0o755
				}
				archiveName := format
				if strings.HasPrefix(format, "bundle ") {
					good := archiveBytes(t, artifactEntries("picfetch-x64.msix", notices, executable), false)
					bad := archiveBytes(t, entries, false)
					entries = []archiveMember{{name: "picfetch-x64.msix", data: good}, {name: "picfetch-arm64.msix", data: good}}
					if format == "bundle x64" {
						entries[0].data = bad
					} else {
						entries[1].data = bad
					}
					archiveName = "picfetch-microsoft-store.msixbundle"
				}
				path := filepath.Join(t.TempDir(), archiveName)
				writeTestFile(t, path, archiveBytes(t, entries, strings.HasSuffix(archiveName, ".tar.gz")))
				err := inspectArtifact(root, path)
				if (err == nil) != (change == "valid") {
					t.Fatalf("artifact check = %v", err)
				}
			})
		}
	}
}

func TestArtifactEmbeddedNoticesAcrossReadBoundaries(t *testing.T) {
	for _, size := range []int{37, 100 * 1024} {
		root, notices := artifactNoticeFixture(t)
		notices[1].data = append(bytes.Repeat([]byte("notice text\n"), size/12), []byte("end of notice\n")...)
		writeTestFile(t, filepath.Join(root, "THIRD-PARTY-NOTICES.md"), notices[1].data)
		for _, format := range []string{"picfetch-windows-amd64.zip", "picfetch-macos-arm64.zip", "picfetch-linux-arm64.tar.gz", "picfetch-x64.msix"} {
			for _, offset := range []int{0, 32*1024 - 1, 64*1024 - 1} {
				t.Run(fmt.Sprintf("%s/%d bytes/offset %d", format, size, offset), func(t *testing.T) {
					executable := bytes.Repeat([]byte{0xa5}, offset)
					executable = append(executable, notices[1].data...)
					executable = append(executable, bytes.Repeat([]byte{0x5a}, 64*1024)...)
					entries := artifactEntries(format, notices, executable)
					path := filepath.Join(t.TempDir(), format)
					writeTestFile(t, path, archiveBytes(t, entries, strings.HasSuffix(format, ".tar.gz")))
					if err := inspectArtifact(root, path); err != nil {
						t.Fatalf("complete embedded notice rejected: %v", err)
					}
				})
			}
		}
	}
}

func TestArtifactRejectsExecutableChecksumErrors(t *testing.T) {
	root, notices := artifactNoticeFixture(t)
	executable := append(slices.Clone(notices[1].data), bytes.Repeat([]byte{0xa5}, 96*1024)...)
	for _, format := range []string{"picfetch-windows-amd64.zip", "picfetch-macos-arm64.zip", "picfetch-x64.msix", "bundle x64", "bundle arm64"} {
		t.Run(format, func(t *testing.T) {
			entries := artifactEntries(format, notices, executable)
			data := archiveBytes(t, entries, false)
			reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			var corrupt bytes.Buffer
			writer := zip.NewWriter(&corrupt)
			for _, file := range reader.File {
				if file.Name == entries[len(entries)-1].name {
					file.CRC32 ^= 1
				}
				if err := writer.Copy(file); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			data = corrupt.Bytes()
			archiveName := format
			if strings.HasPrefix(format, "bundle ") {
				good := archiveBytes(t, entries, false)
				payloads := []archiveMember{{name: "picfetch-x64.msix", data: good}, {name: "picfetch-arm64.msix", data: good}}
				if format == "bundle x64" {
					payloads[0].data = data
				} else {
					payloads[1].data = data
				}
				data = archiveBytes(t, payloads, false)
				archiveName = "picfetch-microsoft-store.msixbundle"
			}
			path := filepath.Join(t.TempDir(), archiveName)
			writeTestFile(t, path, data)
			if err := inspectArtifact(root, path); !errors.Is(err, zip.ErrChecksum) {
				t.Fatalf("executable checksum error = %v, want %v", err, zip.ErrChecksum)
			}
		})
	}
	t.Run("gzip trailer", func(t *testing.T) {
		format := "picfetch-linux-arm64.tar.gz"
		data := archiveBytes(t, artifactEntries(format, notices, executable), true)
		data[len(data)-8] ^= 1
		path := filepath.Join(t.TempDir(), format)
		writeTestFile(t, path, data)
		if err := inspectArtifact(root, path); !errors.Is(err, gzip.ErrChecksum) {
			t.Fatalf("gzip trailer error = %v, want %v", err, gzip.ErrChecksum)
		}
	})
}

func archiveBytes(t *testing.T, entries []archiveMember, compressedTar bool) []byte {
	t.Helper()
	var out bytes.Buffer
	if compressedTar {
		gz := gzip.NewWriter(&out)
		writer := tar.NewWriter(gz)
		for _, entry := range entries {
			header := &tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.data))}
			if entry.mode&os.ModeSymlink != 0 {
				header.Typeflag = tar.TypeSymlink
				header.Linkname = "other-executable"
				header.Size = 0
			}
			if err := writer.WriteHeader(header); err != nil {
				t.Fatal(err)
			}
			if header.Typeflag == tar.TypeSymlink {
				continue
			}
			if _, err := writer.Write(entry.data); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		writer := zip.NewWriter(&out)
		for _, entry := range entries {
			header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
			if entry.mode != 0 {
				header.SetMode(entry.mode)
			}
			file, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write(entry.data); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return out.Bytes()
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowChecksFinalNotices(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, check := range []struct{ file, command, before string }{
		{".github/workflows/ci.yml", "make check-updater-notices", "- name: Vet"},
		{".github/workflows/release.yml", "-artifact dist/picfetch-macos-arm64.zip -artifact dist/picfetch-macos-x86_64.zip", "- name: Create GitHub release"},
		{".github/workflows/release.yml", "-artifact dist/picfetch-windows-amd64.zip -artifact dist/picfetch-windows-arm64.zip", "- name: Create GitHub release"},
		{".github/workflows/release.yml", "-artifact dist/picfetch-linux-amd64.tar.gz -artifact dist/picfetch-linux-arm64.tar.gz", "- name: Create GitHub release"},
		{".github/workflows/microsoft-store.yml", "-artifact dist/picfetch-microsoft-store.msixbundle", "- name: Record validated stable-release provenance"},
	} {
		data, err := os.ReadFile(filepath.Join(root, check.file))
		if err != nil {
			t.Fatal(err)
		}
		command, before := bytes.Index(data, []byte(check.command)), bytes.Index(data, []byte(check.before))
		if command < 0 || before < 0 || command >= before {
			t.Errorf("%s must run %s before %s", check.file, check.command, check.before)
		}
	}
}

func TestNoticesMatchReviewedSources(t *testing.T) {
	if err := notices(filepath.Join("..", ".."), false); err != nil {
		t.Fatal(err)
	}
}

func TestNoticesRejectMissingInventory(t *testing.T) {
	if err := notices(t.TempDir(), false); err == nil {
		t.Fatal("accepted a repository with no reviewed dependency inventory or notices")
	}
}
