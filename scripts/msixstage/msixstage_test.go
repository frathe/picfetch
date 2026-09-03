package main

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frathe/picfetch/internal/imaging"
)

func TestRenderManifest_UsesStoreIdentityVersionAndArchitecture(t *testing.T) {
	manifest, err := renderManifest(appMetadata{Version: "1.0.0"}, "amd64")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		`Name="OpenSourceDeveloperFloria.PicFetch"`,
		`Publisher="CN=D9654E56-586C-4C1E-ABC8-71CCDC33B78F"`,
		`Version="1.0.0.0"`,
		`ProcessorArchitecture="x64"`,
		`Name="Windows.Desktop"`,
		`MinVersion="10.0.19041.0"`,
		`uap10:RuntimeBehavior="packagedClassicApp"`,
		`uap10:TrustLevel="mediumIL"`,
		`<rescap:Capability Name="runFullTrust" />`,
	} {
		if !strings.Contains(manifest, want) {
			t.Errorf("manifest missing %q", want)
		}
	}

	decoder := xml.NewDecoder(strings.NewReader(manifest))
	for {
		if _, err := decoder.Token(); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("manifest is not well-formed XML: %v", err)
		}
	}
}

func TestRenderManifest_MapsArm64AndRejectsUnsupportedArchitecture(t *testing.T) {
	manifest, err := renderManifest(appMetadata{Version: "1.0.0"}, "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(manifest, `ProcessorArchitecture="arm64"`) {
		t.Error("ARM64 manifest has no arm64 package architecture")
	}

	if _, err := renderManifest(appMetadata{Version: "1.0.0"}, "386"); err == nil {
		t.Fatal("unsupported architecture was accepted")
	}
}

func TestRenderManifest_EmitsEverySupportedImageExtension(t *testing.T) {
	manifest, err := renderManifest(appMetadata{Version: "1.0.0"}, "amd64")
	if err != nil {
		t.Fatal(err)
	}

	for _, ext := range imaging.SupportedExtensions() {
		want := "<uap:FileType>" + ext + "</uap:FileType>"
		if strings.Count(manifest, want) != 1 {
			t.Errorf("manifest count for %q = %d, want 1", ext, strings.Count(manifest, want))
		}
	}
}

func TestStoreVersion_UsesSemanticVersionAndStoreReservedRevision(t *testing.T) {
	for _, tc := range []struct {
		version string
		want    string
	}{
		{"1.0.0", "1.0.0.0"},
		{"1.2.3", "1.2.3.0"},
		{"65535.65535.65535", "65535.65535.65535.0"},
	} {
		got, err := storeVersion(tc.version)
		if err != nil {
			t.Fatalf("storeVersion(%q): %v", tc.version, err)
		}
		if got != tc.want {
			t.Errorf("storeVersion(%q) = %q, want %q", tc.version, got, tc.want)
		}
	}

	for _, version := range []string{
		"0.2.17",
		"1.2",
		"1.2.3.4",
		"1.2.3-beta.1",
		"1.02.3",
		"1.-1.0",
		"1.65536.0",
		"65536.0.0",
	} {
		if _, err := storeVersion(version); err == nil {
			t.Errorf("storeVersion(%q) succeeded, want error", version)
		}
	}
}

func TestReadAppMetadata(t *testing.T) {
	got, err := readAppMetadata(strings.NewReader(`[Details]
Name = "PicFetch"
Version = "1.0.0"
Build = 440
`))
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "1.0.0" {
		t.Fatalf("metadata = %+v", got)
	}
}

func TestStage_CopiesExecutableAndRendersAssets(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "FyneApp.toml"), []byte("Version = \"1.0.0\"\nBuild = 441\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestIcon(t, filepath.Join(root, "assets", "appIcon.png"))

	exe := filepath.Join(root, "input.exe")
	wantExe := []byte("test executable")
	if err := os.WriteFile(exe, wantExe, 0o755); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(root, "stage")
	if err := stage(stageOptions{Root: root, Arch: "amd64", Executable: exe, Out: out}); err != nil {
		t.Fatal(err)
	}

	gotExe, err := os.ReadFile(filepath.Join(out, "picfetch.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotExe, wantExe) {
		t.Fatalf("staged executable = %q, want %q", gotExe, wantExe)
	}
	manifest, err := os.ReadFile(filepath.Join(out, "AppxManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(manifest, []byte(`Version="1.0.0.0"`)) {
		t.Fatalf("staged manifest has wrong version:\n%s", manifest)
	}

	wantSizes := map[string]image.Point{
		"StoreLogo.png":                                       {X: 50, Y: 50},
		"Square44x44Logo.png":                                 {X: 44, Y: 44},
		"Square44x44Logo.scale-200.png":                       {X: 88, Y: 88},
		"Square44x44Logo.scale-400.png":                       {X: 176, Y: 176},
		"Square44x44Logo.targetsize-16_altform-unplated.png":  {X: 16, Y: 16},
		"Square44x44Logo.targetsize-24_altform-unplated.png":  {X: 24, Y: 24},
		"Square44x44Logo.targetsize-32_altform-unplated.png":  {X: 32, Y: 32},
		"Square44x44Logo.targetsize-48_altform-unplated.png":  {X: 48, Y: 48},
		"Square44x44Logo.targetsize-256_altform-unplated.png": {X: 256, Y: 256},
		"Square150x150Logo.png":                               {X: 150, Y: 150},
		"Square150x150Logo.scale-200.png":                     {X: 300, Y: 300},
		"Square150x150Logo.scale-400.png":                     {X: 600, Y: 600},
	}
	for name, want := range wantSizes {
		f, err := os.Open(filepath.Join(out, "Assets", name))
		if err != nil {
			t.Errorf("open %s: %v", name, err)
			continue
		}
		cfg, _, err := image.DecodeConfig(f)
		_ = f.Close()
		if err != nil {
			t.Errorf("decode %s: %v", name, err)
			continue
		}
		if got := image.Pt(cfg.Width, cfg.Height); got != want {
			t.Errorf("%s size = %v, want %v", name, got, want)
		}
	}
}

func TestStoreListingAssets(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	listing, err := os.ReadFile(filepath.Join(root, "packaging", "microsoft-store", "listing.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"## English (United States)",
		"## German (Germany)",
		"https://frathe.github.io/picfetch/",
		"https://github.com/frathe/picfetch/issues",
		"https://github.com/frathe/picfetch/blob/main/PRIVACY.md",
		"https://github.com/frathe/picfetch/blob/main/LICENSE",
		"runFullTrust",
		"assets/screens/picture_galery.png",
		"assets/screens/viewer.png",
	} {
		if !bytes.Contains(listing, []byte(want)) {
			t.Errorf("listing handoff missing %q", want)
		}
	}

	assertImageSize(t, filepath.Join(root, "packaging", "microsoft-store", "StoreLogo-300.png"), image.Pt(300, 300))
	for _, name := range []string{"picture_galery.png", "viewer.png"} {
		path := filepath.Join(root, "assets", "screens", name)
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		cfg, _, err := image.DecodeConfig(f)
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Width < 1366 || cfg.Height < 768 {
			t.Errorf("%s is %dx%d, below the Store desktop minimum", name, cfg.Width, cfg.Height)
		}
	}
}

func TestMicrosoftStoreWorkflowAndBuildTarget(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "microsoft-store.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"workflow_dispatch:",
		"tags:",
		"make package-windows-store",
		"-arch amd64",
		"-arch arm64",
		"MakeAppx.exe",
		"/h SHA256",
		"SignTool.exe",
		"X509Store('TrustedPeople', 'LocalMachine')",
		"appcert test",
		"Resolve-Path -LiteralPath 'dist/picfetch-microsoft-store.msixbundle'",
		"Join-Path (Resolve-Path -LiteralPath 'dist').Path 'wack-report.xml'",
		"picfetch-microsoft-store.msixbundle",
		"wack-report.xml",
	} {
		if !bytes.Contains(workflow, []byte(want)) {
			t.Errorf("Microsoft Store workflow missing %q", want)
		}
	}
	if bytes.Contains(workflow, []byte("X509Store('TrustedPeople', 'CurrentUser')")) {
		t.Error("Microsoft Store workflow trusts its test certificate only for the current user")
	}

	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"package-windows-store:", "-tags microsoftstore", "$(BIN_NAME)-microsoft-store-$$arch.exe"} {
		if !bytes.Contains(makefile, []byte(want)) {
			t.Errorf("Microsoft Store build target missing %q", want)
		}
	}

	ci, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(ci, []byte("ci-${{ github.workflow }}-${{ github.ref }}")) {
		t.Error("reusable CI concurrency does not distinguish the Release and Microsoft Store callers")
	}
}

func assertImageSize(t *testing.T, path string, want image.Point) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, err := image.DecodeConfig(f)
	_ = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if got := image.Pt(cfg.Width, cfg.Height); got != want {
		t.Errorf("%s size = %v, want %v", path, got, want)
	}
}

func writeTestIcon(t *testing.T, path string) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 8), G: uint8(y * 8), B: 80, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
