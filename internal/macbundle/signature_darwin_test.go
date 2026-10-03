//go:build darwin && cgo

package macbundle

import (
	"context"
	"debug/macho"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSignedRuntimeRequiresLibraryAndOuterSeals(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Fixture.app")
	contents := filepath.Join(app, "Contents")
	frameworks := filepath.Join(contents, "Frameworks")
	for _, dir := range []string{frameworks, filepath.Join(contents, "MacOS"), filepath.Join(contents, "Resources")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	put := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	put(filepath.Join(contents, "Info.plist"), `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>io.github.frathe.picfetch</string><key>CFBundleExecutable</key><string>PicFetch</string><key>CFBundlePackageType</key><string>APPL</string></dict></plist>`)
	source := filepath.Join(t.TempDir(), "fixture.c")
	put(source, "int fixture(void) { return 42; }\nint main(void) {return 0;}\n")
	command := func(name string, args ...string) {
		t.Helper()
		if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", name, err, out)
		}
	}
	library := filepath.Join(frameworks, "libfixture.dylib")
	command("xcrun", "clang", "-dynamiclib", source, "-o", library)
	command("xcrun", "clang", source, "-o", filepath.Join(contents, "MacOS", "PicFetch"))
	notice := filepath.Join(contents, "Resources", "notice.txt")
	put(notice, "sealed notice")
	command("codesign", "--force", "--sign", "-", library)
	command("codesign", "--force", "--sign", "-", app)
	verify := func() error { return VerifyLibrary(context.Background(), frameworks, library, runtime.GOARCH) }
	if err := verify(); err != nil {
		t.Fatalf("valid signed fixture: %v", err)
	}
	other := "amd64"
	if runtime.GOARCH == other {
		other = "arm64"
	}
	if err := VerifyLibrary(context.Background(), frameworks, library, other); err == nil {
		t.Fatal("accepted wrong architecture")
	}
	put(notice, "modified notice")
	if err := verify(); err == nil {
		t.Fatal("accepted modified outer bundle")
	}
	put(notice, "sealed notice")
	original, err := os.ReadFile(library)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := macho.Open(library)
	if err != nil {
		t.Fatal(err)
	}
	section := parsed.Section("__text")
	_ = parsed.Close()
	if section == nil {
		t.Fatal("native fixture has no text section")
	}
	offset := section.Offset
	modified := append([]byte(nil), original...)
	modified[offset] ^= 1
	if err := os.WriteFile(library, modified, 0700); err != nil {
		t.Fatal(err)
	}
	if err := verify(); err == nil {
		t.Fatal("accepted tampered library")
	}
	command("codesign", "--force", "--sign", "-", library)
	if err := verify(); err == nil {
		t.Fatal("accepted newly signed library outside the outer seal")
	}
	if err := os.WriteFile(library, original, 0700); err != nil {
		t.Fatal(err)
	}
	if err := verify(); err != nil {
		t.Fatalf("restored library: %v", err)
	}
	command("codesign", "--remove-signature", library)
	if err := verify(); err == nil {
		t.Fatal("accepted unsigned library")
	}
}

func TestRuntimeVerifierRejectsNonBundleAndCancelledInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := VerifyLibrary(ctx, "/missing", "/missing/lib.dylib", runtime.GOARCH); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
	if err := VerifyLibrary(context.Background(), t.TempDir(), "/missing/lib.dylib", runtime.GOARCH); err == nil || !strings.Contains(err.Error(), "Frameworks") {
		t.Fatalf("non-bundle root: %v", err)
	}
}

func TestRuntimeVerifierRejectsEscapingLinks(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Fixture.app")
	contents := filepath.Join(app, "Contents")
	if err := os.MkdirAll(contents, 0700); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	target := filepath.Join(outside, "libfixture.dylib")
	if err := os.WriteFile(target, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	frameworks := filepath.Join(contents, "Frameworks")
	if err := os.Symlink(outside, frameworks); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLibrary(t.Context(), frameworks, filepath.Join(frameworks, "libfixture.dylib"), runtime.GOARCH); err == nil || !strings.Contains(err.Error(), "directory escapes") {
		t.Fatalf("escaping Frameworks: %v", err)
	}
	if err := os.Remove(frameworks); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(frameworks, 0700); err != nil {
		t.Fatal(err)
	}
	library := filepath.Join(frameworks, "libfixture.dylib")
	if err := os.Symlink(target, library); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLibrary(t.Context(), frameworks, library, runtime.GOARCH); err == nil || !strings.Contains(err.Error(), "library escapes") {
		t.Fatalf("escaping library: %v", err)
	}
}

func TestNativeExecutablePathIsAbsolute(t *testing.T) {
	path, err := executablePath()
	if err != nil || !filepath.IsAbs(path) {
		t.Fatalf("native executable: %q, %v", path, err)
	}
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("native executable not a file: %v", err)
	}
}
