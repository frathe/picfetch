package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/frathe/picfetch/internal/imaging"
)

func TestHEICStaticDeclarations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Info.plist")
	if err := os.WriteFile(path, []byte(fyneTemplateFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{path}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"heic", "heif", "public.heic", "public.heif", "jpg", "public.jpeg", "public.folder"} {
		if strings.Count(string(data), "<string>"+value+"</string>") != 1 {
			t.Errorf("static declaration must contain %q exactly once", value)
		}
	}
	for _, ext := range []string{".heic", ".heif"} {
		if slices.Contains(imaging.SupportedExtensions(), ext) {
			t.Errorf("optional codec %s entered unconditional runtime extensions", ext)
		}
	}
}

func TestRun_Usage(t *testing.T) {
	cases := [][]string{
		nil,
		{},
		{"a", "b"},
	}

	for _, args := range cases {
		err := run(args)
		if err == nil || !strings.Contains(err.Error(), "usage:") {
			t.Errorf("run(%v) = %v, want a usage error", args, err)
		}
	}
}

func TestRun_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.plist")

	if err := run([]string{path}); err == nil {
		t.Fatal("expected an error reading a missing file, got nil")
	}
}

func TestRun_PatchesFileInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Info.plist")
	if err := os.WriteFile(path, []byte(fyneTemplateFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{path}); err != nil {
		t.Fatalf("run: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "<key>CFBundleDocumentTypes</key>") {
		t.Fatal("patched file is missing CFBundleDocumentTypes")
	}
}

func TestRun_FailsLoudlyOnSecondInvocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Info.plist")
	if err := os.WriteFile(path, []byte(fyneTemplateFixture), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{path}); err != nil {
		t.Fatalf("first run: %v", err)
	}

	if err := run([]string{path}); err == nil {
		t.Fatal("expected the second run against an already-patched Info.plist to fail, got nil")
	}
}
