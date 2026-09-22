package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestHEICStaticDeclarationsDelivery(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(makefile), "PACKAGE_ID := io.github.frathe.picfetch\n") {
		t.Error("Makefile package identity differs from the desktop identity")
	}
	for _, target := range []string{"package-linux", "package-linux-debug"} {
		_, body, found := strings.Cut(string(makefile), "\n"+target+":")
		body, _, _ = strings.Cut(body, "\n\n")
		if !found || !strings.Contains(body, "./scripts/linuxdesktop") || !strings.Contains(body, `-executable "$(BIN_NAME)-`) {
			t.Errorf("%s does not stage the matching desktop launcher", target)
		}
	}
	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var archiveLine string
	for _, line := range strings.Split(string(workflow), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), `tar -czf "picfetch-linux-$arch.tar.gz"`) {
			archiveLine = strings.TrimSpace(line)
		}
	}
	if archiveLine == "" {
		t.Fatal("Linux release archive command is missing")
	}
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			staging := t.TempDir()
			bin := filepath.Join(staging, "bin")
			if err := os.Mkdir(bin, 0755); err != nil {
				t.Fatal(err)
			}
			executable := "picfetch-linux-" + arch
			binary := []byte("representative bundled executable for archive membership")
			if err := os.WriteFile(filepath.Join(bin, executable), binary, 0755); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"LICENSE", "THIRD-PARTY-NOTICES.md", "PRIVACY.md"} {
				data, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(staging, name), data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := run([]string{"-root", root, "-executable", executable, "-out", filepath.Join(bin, "linux-"+arch)}); err != nil {
				t.Fatal(err)
			}
			// Execute the production tar argument vector without a shell. Every
			// token in this narrowly scoped release command is one simple argument.
			words := strings.Fields(strings.ReplaceAll(archiveLine, "$arch", arch))
			for i := range words {
				if strings.HasPrefix(words[i], `"`) {
					words[i], err = strconv.Unquote(words[i])
					if err != nil {
						t.Fatalf("archive argument is no longer a simple quoted token: %v", err)
					}
				}
			}
			cmd := exec.Command(words[0], words[1:]...)
			cmd.Dir = bin
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("release archive command: %v\n%s", err, output)
			}
			files := archiveFiles(t, filepath.Join(bin, executable+".tar.gz"))
			for _, name := range []string{executable, "io.github.frathe.picfetch.desktop", "io.github.frathe.picfetch.png", "LICENSE", "THIRD-PARTY-NOTICES.md", "PRIVACY.md"} {
				if len(files[name]) == 0 {
					t.Errorf("release archive is missing %s", name)
				}
			}
			if !bytes.Equal(files[executable], binary) {
				t.Error("archive changed the executable")
			}
			if data := files["io.github.frathe.picfetch.desktop"]; len(data) != 0 && desktopFields(t, data)["Exec"] != executable+" %F" {
				t.Error("archive launcher targets another architecture")
			}
		})
	}
}

func archiveFiles(t *testing.T, path string) map[string][]byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	zipped, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zipped.Close() }()
	reader := tar.NewReader(zipped)
	files := map[string][]byte{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return files
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.ToSlash(filepath.Clean(header.Name))
		if _, duplicate := files[name]; duplicate {
			t.Fatalf("duplicate archive member: %s", name)
		}
		files[name] = data
	}
}
