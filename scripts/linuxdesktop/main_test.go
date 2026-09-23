package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/frathe/picfetch/internal/imaging"
)

func TestHEICStaticDeclarations(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			out := t.TempDir()
			executable := "picfetch-linux-" + arch
			args := []string{"-root", root, "-executable", executable, "-out", out}
			if err := run(args); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(out, "io.github.frathe.picfetch.desktop")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			entry := desktopFields(t, data)
			for key, value := range map[string]string{"Name": "PicFetch", "Icon": "io.github.frathe.picfetch", "Exec": executable + " %F", "Terminal": "false", "Type": "Application", "Categories": "Graphics;Photography;Viewer;"} {
				if entry[key] != value {
					t.Errorf("%s=%q, want %q", key, entry[key], value)
				}
			}
			mimes := strings.Split(entry["MimeType"], ";")
			for _, mime := range []string{"image/heic", "image/heif", "image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp", "image/tiff", "image/avif", "image/svg+xml", "image/vnd.microsoft.icon", "image/x-xpixmap", "image/x-canon-cr2", "image/x-adobe-dng"} {
				if !slices.Contains(mimes, mime) || strings.Count(entry["MimeType"], mime+";") != 1 {
					t.Errorf("missing or duplicate static type %s: %q", mime, entry["MimeType"])
				}
			}
			for _, ext := range []string{".heic", ".heif"} {
				if slices.Contains(imaging.SupportedExtensions(), ext) {
					t.Fatalf("optional decoder unexpectedly in unconditional extensions: %s", ext)
				}
			}
			icon, err := os.ReadFile(filepath.Join(out, "io.github.frathe.picfetch.png"))
			if err != nil {
				t.Fatal(err)
			}
			wantIcon, err := os.ReadFile(filepath.Join(root, "assets", "appIcon.png"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(icon, wantIcon) {
				t.Fatal("staged icon differs from the official app icon")
			}
			if err := run(args); err != nil {
				t.Fatal(err)
			}
			again, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(data, again) {
				t.Fatalf("desktop declaration is nondeterministic: %v", err)
			}
		})
	}
	t.Run("safe_executable", func(t *testing.T) {
		out := t.TempDir()
		if err := run([]string{"-root", root, "-executable", "Pic Fetch-linux-amd64", "-out", out}); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(out, "io.github.frathe.picfetch.desktop"))
		if err != nil {
			t.Fatal(err)
		}
		if got := desktopFields(t, data)["Exec"]; got != `"Pic Fetch-linux-amd64" %F` {
			t.Fatalf("executable spaces escaped the one-argument boundary: %q", got)
		}
	})
	t.Run("reject_unsafe_executable", func(t *testing.T) {
		for _, executable := range []string{"/usr/bin/picfetch", "../picfetch", "", "picfetch;touch x", "picfetch$HOME", "picfetch`id`", "picfetch%F", "picfetch\nHidden=true", "picfetch\tother", `picfetch\name`, `picfetch"name`, "picfetch=arg", "<picfetch>", "picfetch|other", "picfetch&", "picfetch#comment", "picfetch?", "picfetch*", "picfetch'quote", "..", "-picfetch", " picfetch", "picfetch ", "PicFétch"} {
			out := t.TempDir()
			if err := run([]string{"-root", root, "-executable", executable, "-out", out}); err == nil {
				t.Errorf("unsafe executable accepted: %q", executable)
			}
			files, err := os.ReadDir(out)
			if err != nil || len(files) != 0 {
				t.Errorf("unsafe executable produced files: %q, %v, %v", executable, files, err)
			}
		}
	})
}

func TestLinuxDesktopMetadataBoundary(t *testing.T) {
	for _, input := range []struct{ name, id, icon string }{
		{"PicFetch\nHidden=true", "io.github.frathe.picfetch", "icon.png"},
		{"PicFetch", "../escape", "icon.png"},
		{"PicFetch", "io.github.frathe.picfetch", "../icon.png"},
	} {
		parent := t.TempDir()
		root := filepath.Join(parent, "project")
		if err := os.Mkdir(root, 0755); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{filepath.Join(root, "icon.png"), filepath.Join(parent, "icon.png")} {
			if err := os.WriteFile(path, []byte("owned test icon"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		metadata := fmt.Sprintf("[Details]\nName = %q\nID = %q\nIcon = %q\n", input.name, input.id, input.icon)
		if err := os.WriteFile(filepath.Join(root, "FyneApp.toml"), []byte(metadata), 0644); err != nil {
			t.Fatal(err)
		}
		if err := run([]string{"-root", root, "-executable", "picfetch", "-out", filepath.Join(parent, "out")}); err == nil {
			t.Errorf("unsafe metadata accepted: %+v", input)
		}
	}
}

func desktopFields(t *testing.T, data []byte) map[string]string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] != "[Desktop Entry]" {
		t.Fatal("missing desktop entry group")
	}
	fields := map[string]string{}
	for _, line := range lines[1:] {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("invalid desktop entry: %s", line)
		}
		if _, duplicate := fields[key]; duplicate {
			t.Fatalf("duplicate key: %s", key)
		}
		fields[key] = value
	}
	return fields
}
