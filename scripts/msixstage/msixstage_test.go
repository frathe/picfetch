package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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
		`<PackageDependency Name="Microsoft.VCLibs.140.00.UWPDesktop" MinVersion="14.0.33728.0" Publisher="CN=Microsoft Corporation, O=Microsoft Corporation, L=Redmond, S=Washington, C=US" />`,
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
	for _, name := range []string{"LICENSE", "THIRD-PARTY-NOTICES.md", "PRIVACY.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("test "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	exe := filepath.Join(root, "input.exe")
	wantExe := []byte("test executable")
	if err := os.WriteFile(exe, wantExe, 0o755); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(root, "stage")
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch+"_requires_runtime", func(t *testing.T) {
			if err := stage(stageOptions{Root: root, Arch: arch, Executable: exe, Out: filepath.Join(root, "missing-runtime")}); err == nil {
				t.Fatal("Store staging accepted an absent bundled runtime")
			}
		})
	}
	options := stageOptions{Root: root, Arch: "arm64", Executable: exe, Out: out, RuntimeArchive: filepath.Join(root, "runtime.zip")}
	t.Run("runtime_failure_stops_packaging", func(t *testing.T) {
		wantErr := errors.New("runtime checksum mismatch")
		if err := stageWithRuntime(options, func(_ context.Context, _, _, _ string) error { return wantErr }); !errors.Is(err, wantErr) {
			t.Fatalf("runtime failure was lost: %v", err)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatalf("failed runtime produced a package: %v", err)
		}
	})
	runtimeStaged := false
	if err := stageWithRuntime(options, func(_ context.Context, arch, archive, destination string) error {
		if arch != options.Arch || archive != options.RuntimeArchive || destination != out {
			t.Fatal("runtime staging lost its architecture, pinned input or destination")
		}
		runtimeStaged = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !runtimeStaged {
		t.Fatal("Store package omitted runtime staging")
	}
	t.Run("rejects_stale_package", func(t *testing.T) {
		if err := stageWithRuntime(options, func(_ context.Context, _, _, _ string) error {
			t.Fatal("attempted runtime extraction into a stale package")
			return nil
		}); err == nil || !strings.Contains(err.Error(), "must be empty") {
			t.Fatalf("stale package accepted: %v", err)
		}
	})
	for _, name := range []string{"LICENSE", "THIRD-PARTY-NOTICES.md", "PRIVACY.md"} {
		got, err := os.ReadFile(filepath.Join(out, name))
		if err != nil || string(got) != "test "+name {
			t.Fatalf("staged notice %s: %q, %v", name, got, err)
		}
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

func TestStandaloneArchivesRetainNotices(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	workflow, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"picfetch.exe" ../LICENSE ../THIRD-PARTY-NOTICES.md ../PRIVACY.md -j`,
		`-C .. LICENSE THIRD-PARTY-NOTICES.md PRIVACY.md`,
		`@('LICENSE', 'THIRD-PARTY-NOTICES.md', 'PRIVACY.md')`,
		`Compress-Archive -Path $packageFiles`,
	} {
		if !bytes.Contains(workflow, []byte(want)) {
			t.Errorf("standalone release loses notices: missing %q", want)
		}
	}
	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(makefile, []byte(`cp LICENSE THIRD-PARTY-NOTICES.md PRIVACY.md "$(APP_NAME).app/Contents/Resources/"`)) {
		t.Fatal("macOS bundle omits license documents")
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
		"-runtime-archive dist/onnxruntime-win-x64.zip",
		"-runtime-archive dist/onnxruntime-win-arm64.zip",
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
	for _, want := range []string{
		"warm-fyne-cross-windows:",
		`-v "$(FYNE_CROSS_CACHE):/go"`,
		"package-windows-store: warm-fyne-cross-windows",
		`-cache "$(FYNE_CROSS_CACHE)"`,
		`-tags "$(APP_TAGS),microsoftstore"`,
		"$(BIN_NAME)-microsoft-store-$$arch.exe",
	} {
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

func TestStoreWorkflowPublishingContract(t *testing.T) {
	t.Run("environment policy", testStoreEnvironmentPolicy)
	root := filepath.Join("..", "..", ".github", "workflows")
	producer, err := os.ReadFile(filepath.Join(root, "microsoft-store.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Index(producer, []byte("go run ./scripts/storepublish record")) <= bytes.Index(producer, []byte("appcert test")) {
		t.Error("release evidence must be recorded after WACK succeeds")
	}
	for _, want := range []string{"dist/store-release.json", "retention-days: 90", "refs/tags/v[0-9]+"} {
		if !bytes.Contains(producer, []byte(want)) {
			t.Errorf("producer missing %q", want)
		}
	}
	publisher, err := os.ReadFile(filepath.Join(root, "microsoft-store-publish.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"workflow_run:", "workflows: [Microsoft Store package]", "types: [completed]", "workflow_dispatch:",
		"prepare:", "needs: prepare", "deployment-branch-policies", "deployment_protection_rules", "environment-policy.jq",
		"github.repository == 'frathe/picfetch'", "github.ref == 'refs/heads/main'", "github.event.workflow_run.conclusion == 'success'", "github.event.workflow_run.event == 'push'", "github.event.workflow_run.head_repository.full_name == github.repository",
		"group: microsoft-store-publisher", "cancel-in-progress: false", "name: microsoft-store", "contents: read", "actions: read", "deployments: write",
		"ref: ${{ github.sha }}", "fetch-depth: 0", "persist-credentials: false", "PICFETCH_STORE_SERIALIZED: '1'", "timeout-minutes: 15",
		"secrets.MSSTORE_TENANT_ID", "secrets.MSSTORE_CLIENT_ID", "secrets.MSSTORE_CLIENT_SECRET", "go run ./scripts/storepublish",
		"GITHUB_STEP_SUMMARY", "store-result.json", "--state-dir", "--approval-sha256", "--approval approval/approval.json", "artifact-ids: ${{ needs.prepare.outputs.artifact_id }}", "needs.prepare.outputs.sha256", "--run-id", "github.event.workflow_run.id", "steps.record.outputs.artifact-id",
	} {
		if !bytes.Contains(publisher, []byte(want)) {
			t.Errorf("publisher missing %q", want)
		}
	}
	for _, forbidden := range []string{"schedule:", "cron:", "ref: main", "secrets: inherit", "contents: write", "pull_request_target:", "github.event.workflow_run.head_sha", "make package", "make release", "gh release", "release.yml", "inputs.tag }}"} {
		if bytes.Contains(publisher, []byte(forbidden)) {
			t.Errorf("publisher contains unsafe or coupled source %q", forbidden)
		}
	}
	prepare := bytes.Split(publisher, []byte("\n  publish:"))[0]
	for _, forbidden := range []string{"environment:", "secrets.MSSTORE_", "deployments: write"} {
		if bytes.Contains(prepare, []byte(forbidden)) {
			t.Errorf("unapproved preparation contains %q", forbidden)
		}
	}
}

func testStoreEnvironmentPolicy(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required to execute the GitHub environment policy")
	}
	const policy = `{"name":"microsoft-store","can_admins_bypass":false,"deployment_branch_policy":{"custom_branch_policies":true,"protected_branches":false},"protection_rules":[{"type":"branch_policy"},{"type":"required_reviewers","prevent_self_review":false,"reviewers":[{"type":"User","reviewer":{"login":"frathe"}}]}]}`
	for _, change := range []string{"valid", "missing reviewer", "other reviewer", "bypass", "self review blocked", "timer", "tag", "extra branch", "custom rule"} {
		t.Run(change, func(t *testing.T) {
			environment := policy
			branches := `{"total_count":1,"branch_policies":[{"name":"main","type":"branch"}]}`
			custom := `{"total_count":0,"custom_deployment_protection_rules":[]}`
			switch change {
			case "missing reviewer":
				environment = strings.ReplaceAll(environment, "required_reviewers", "disabled_reviewers")
			case "other reviewer":
				environment = strings.ReplaceAll(environment, "frathe", "someone-else")
			case "bypass":
				environment = strings.ReplaceAll(environment, `"can_admins_bypass":false`, `"can_admins_bypass":true`)
			case "self review blocked":
				environment = strings.ReplaceAll(environment, `"prevent_self_review":false`, `"prevent_self_review":true`)
			case "timer":
				environment = strings.ReplaceAll(environment, `{"type":"branch_policy"}`, `{"type":"branch_policy"},{"type":"wait_timer","wait_timer":15}`)
			case "tag":
				branches = strings.ReplaceAll(branches, `"type":"branch"`, `"type":"tag"`)
			case "extra branch":
				branches = `{"total_count":2,"branch_policies":[{"name":"main","type":"branch"},{"name":"*","type":"branch"}]}`
			case "custom rule":
				custom = `{"total_count":1,"custom_deployment_protection_rules":[{"enabled":true}]}`
			}
			for _, document := range []string{environment, branches, custom} {
				if !json.Valid([]byte(document)) {
					t.Fatal("invalid policy fixture")
				}
			}
			cmd := exec.Command("jq", "-e", "-s", "-f", "../storepublish/environment-policy.jq")
			cmd.Stdin = strings.NewReader(environment + "\n" + branches + "\n" + custom)
			out, err := cmd.CombinedOutput()
			if (err == nil) != (change == "valid") || (err != nil && string(out) != "false\n") {
				t.Fatalf("policy %s: %s (%v)", change, out, err)
			}
		})
	}
}

func TestWindowsToolchainWarmupUsesPackagingUser(t *testing.T) {
	if os.Getuid() < 0 {
		t.Skip("requires POSIX user IDs")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("requires a POSIX shell")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("requires make")
	}
	dir := t.TempDir()
	engine := filepath.Join(dir, "engine")
	argsFile := filepath.Join(dir, "args")
	if err := os.WriteFile(engine, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"$PICFETCH_ENGINE_ARGS\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(dir, "cache with spaces")
	cmd := exec.Command("make", "--no-print-directory", "warm-fyne-cross-windows",
		"FYNE_CROSS_ENGINE="+engine, "FYNE_CROSS_CACHE="+cache, "FYNE_CROSS_WINDOWS_IMAGE=test-image")
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "PICFETCH_ENGINE_ARGS="+argsFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("warm toolchain: %v\n%s", err, output)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	// Check the executed container boundary: the warm-up must use the same
	// UID as fyne-cross, or Go creates a root-owned module cache on Linux.
	for _, want := range []string{
		"--user\n" + strconv.Itoa(os.Getuid()) + "\n",
		"-e\nHOME=/tmp\n",
		"-v\n" + cache + ":/go\n",
		"-e\nGOTOOLCHAIN=auto\n",
		"test-image\ngo\nversion\n",
	} {
		if !bytes.Contains(args, []byte(want)) {
			t.Errorf("warm-up container arguments missing %q:\n%s", want, args)
		}
	}
}

func TestPackagingToolsUseCurrentFyneCLI(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	for _, tc := range []struct {
		name     string
		required []string
	}{
		{"Makefile", []string{"include packaging/tools.mk", "go install fyne.io/tools/cmd/fyne@$(FYNE_VERSION)", "go install github.com/fyne-io/fyne-cross@$(FYNE_CROSS_VERSION)", "package-mac: install-fyne", `"$(FYNE_BIN)" package`}},
		{filepath.Join(".github", "workflows", "release.yml"), []string{"make install-fyne", "make install-fyne-cross"}},
		{filepath.Join(".github", "workflows", "microsoft-store.yml"), []string{"make install-fyne-cross"}},
	} {
		content, err := os.ReadFile(filepath.Join(root, tc.name))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(content, []byte("fyne.io/fyne/v2/cmd/fyne")) || regexp.MustCompile(`(?:fyne.io/tools/cmd/fyne|github.com/fyne-io/fyne-cross)@latest`).Match(content) {
			t.Errorf("%s installs a deprecated or floating packaging CLI", tc.name)
		}
		for _, want := range tc.required {
			if !bytes.Contains(content, []byte(want)) {
				t.Errorf("%s omits shared packaging input %q", tc.name, want)
			}
		}
	}
	inputs, err := os.ReadFile(filepath.Join(root, "packaging", "tools.mk"))
	if err != nil {
		t.Fatal(err)
	}
	inputs = bytes.ReplaceAll(inputs, []byte("\r\n"), []byte("\n"))
	for _, pattern := range []string{
		`(?m)^FYNE_VERSION := v[0-9]+\.[0-9]+\.[0-9]+$`,
		`(?m)^FYNE_CROSS_VERSION := v[0-9]+\.[0-9]+\.[0-9]+$`,
		`(?m)^FYNE_CROSS_WINDOWS_IMAGE \?= fyneio/fyne-cross-images:windows@sha256:[0-9a-f]{64}$`,
		`(?m)^FYNE_CROSS_LINUX_IMAGE \?= fyneio/fyne-cross-images:linux@sha256:[0-9a-f]{64}$`,
	} {
		if !regexp.MustCompile(pattern).Match(inputs) {
			t.Errorf("packaging input is not versioned/pinned: %s", pattern)
		}
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
	for y := range 32 {
		for x := range 32 {
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

func TestCrossPackagingUsesReviewedInputs(t *testing.T) {
	if os.Getuid() < 0 {
		t.Skip("packaging Make targets require a POSIX host")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("requires make")
	}
	for _, route := range []struct {
		target, platform, artifact, flags string
	}{
		{"package-linux", "linux", "picfetch-linux-", ""},
		{"package-linux-debug", "linux", "picfetch-debug-linux-", "-no-strip-debug\n"},
		{"package-windows", "windows", "picfetch-windows-", ""},
		{"package-windows-store", "windows", "picfetch-microsoft-store-", "-tags\nno_emoji,microsoftstore\n"},
		{"package-windows-debug", "windows", "picfetch-debug-windows-", "-console\n-no-strip-debug\n"},
	} {
		t.Run(route.target, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"Makefile", "packaging/tools.mk"} {
				data, err := os.ReadFile(filepath.Join("..", "..", name))
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			toolsDir := filepath.Join(dir, "tools")
			if err := os.MkdirAll(toolsDir, 0o755); err != nil {
				t.Fatal(err)
			}
			writeTool := func(name, body string) string {
				path := filepath.Join(toolsDir, name)
				if err := os.WriteFile(path, []byte("#!/bin/sh\nset -eu\n"+body), 0o700); err != nil {
					t.Fatal(err)
				}
				return path
			}
			cross := writeTool("fyne-cross", `printf '%s\n' CROSS "$@" END >> "$PICFETCH_PACKAGING_LOG"
if [ "${PICFETCH_PACKAGING_FAIL:-}" = cross ]; then exit 23; fi
kind=$1
shift
arch=''
name=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    -arch=*) arch=${1#-arch=} ;;
    -name) shift; name=$1 ;;
  esac
  shift
done
mkdir -p "fyne-cross/bin/$kind-$arch"
if [ "$kind" = windows ]; then name="$name.exe"; fi
printf '%s\n' "$kind $arch" > "fyne-cross/bin/$kind-$arch/$name"
`)
			engine := writeTool("engine", `printf '%s\n' ENGINE "$@" END >> "$PICFETCH_PACKAGING_LOG"
if [ "${PICFETCH_PACKAGING_FAIL:-}" = engine ]; then exit 23; fi
`)
			writeTool("cp", `if [ "${PICFETCH_PACKAGING_FAIL:-}" = copy ]; then
  case "$1" in *-amd64/*) exit 23 ;; esac
fi
exec /bin/cp "$@"
`)
			writeTool("go", `printf '%s\n' GO "$@" END >> "$PICFETCH_PACKAGING_LOG"
`)
			logPath := filepath.Join(dir, "commands")
			cache := filepath.Join(dir, "cache with spaces")
			cmd := exec.Command("make", "--no-print-directory", route.target, "FYNE_CROSS_ENGINE="+engine,
				"FYNE_CROSS_BIN="+cross, "FYNE_CROSS_CACHE="+cache)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "PATH="+toolsDir+string(os.PathListSeparator)+os.Getenv("PATH"), "PICFETCH_PACKAGING_LOG="+logPath)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("package route: %v\n%s", err, output)
			}
			data, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			log := string(data)
			for _, call := range strings.Split(log, "ENGINE\n")[1:] {
				if !strings.HasPrefix(call, "run\n") {
					continue
				}
				args, _, _ := strings.Cut(call, "END\n")
				if !strings.Contains(args, "--user\n"+strconv.Itoa(os.Getuid())+"\n") {
					t.Errorf("container call omitted packaging UID:\n%s", args)
				}
			}
			for _, want := range []string{
				"GO\nrun\n./scripts/tagvectors\n",
				"-engine\n" + engine + "\n",
				"-cache\n" + cache + "\n",
				"-image\nfyneio/fyne-cross-images:" + route.platform + "@sha256:",
				"GOTOOLCHAIN=auto\n",
				"--user\n" + strconv.Itoa(os.Getuid()) + "\n",
				"GO\nversion\n-m\n" + cross + "\n",
				"ENGINE\nimage\ninspect\n",
				"/usr/local/bin/fyne\nversion\n",
			} {
				if !strings.Contains(log, want) {
					t.Errorf("packaging omitted %q\n%s", want, log)
				}
			}
			if route.flags != "" && !strings.Contains(log, route.flags) {
				t.Errorf("route omitted distribution/debug flags %q", route.flags)
			}
			for _, call := range strings.Split(log, "CROSS\n")[1:] {
				args, _, _ := strings.Cut(call, "END\n")
				if !strings.Contains(args, "-tags\nno_emoji") {
					t.Errorf("packaged app includes the unused emoji font:\n%s", args)
				}
			}
			if route.target != "package-windows-store" && strings.Contains(log, "microsoftstore") {
				t.Error("ordinary route selected Store distribution")
			}
			for _, arch := range []string{"amd64", "arm64"} {
				name := route.artifact + arch
				if route.platform == "windows" {
					name += ".exe"
				}
				artifact, err := os.ReadFile(filepath.Join(dir, "bin", name))
				if err != nil || string(artifact) != route.platform+" "+arch+"\n" {
					t.Errorf("%s artifact = %q, %v", arch, artifact, err)
				}
			}
			for _, failure := range []string{"engine", "cross", "copy"} {
				t.Run(failure+" failure", func(t *testing.T) {
					if err := os.WriteFile(logPath, nil, 0o600); err != nil {
						t.Fatal(err)
					}
					failed := exec.Command(cmd.Path, cmd.Args[1:]...)
					failed.Dir = dir
					failed.Env = append(cmd.Env, "PICFETCH_PACKAGING_FAIL="+failure)
					if output, err := failed.CombinedOutput(); err == nil {
						t.Errorf("packaging accepted %s failure:\n%s", failure, output)
					}
					data, err := os.ReadFile(logPath)
					if err != nil {
						t.Fatal(err)
					}
					if bytes.Contains(data, []byte("-arch=arm64\n")) {
						t.Error("packaging continued after the first architecture failed")
					}
				})
			}
		})
	}
}
