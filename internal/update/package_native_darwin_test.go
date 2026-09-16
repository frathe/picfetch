//go:build heicnative && darwin

package update

import (
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// This owns complete signed fixture bundles. It never launches the fixture app
// or touches an installed application, preferences or a user's image files.
func TestNativeMacUpdatePreservesSignedBundle(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	unrelated := filepath.Join(parent, "owned-user-data")
	if err = os.WriteFile(unrelated, []byte("preserved"), 0600); err != nil {
		t.Fatal(err)
	}
	previous := nativeSignedBundle(t, root, filepath.Join(parent, "installed", "PicFetch.app"), "1")
	candidate := nativeSignedBundle(t, root, filepath.Join(parent, "staged", "PicFetch.app"), "2")
	before, err := companionDigests(previous)
	if err != nil {
		t.Fatal(err)
	}
	after, err := companionDigests(candidate)
	if err != nil {
		t.Fatal(err)
	}
	staged := Stage{BinaryPath: filepath.Join(candidate, "Contents", "MacOS", "picfetch"), PlistPath: filepath.Join(candidate, "Contents", "Info.plist"), verification: stageVerification{GOOS: "darwin", CompanionDigests: after}}
	failedInstall := false
	_, err = installDirectory(candidate, previous, after, func(from, to string) error {
		if strings.HasPrefix(filepath.Base(from), ".picfetch-package-new-") && to == previous {
			failedInstall = true
			return os.ErrPermission
		}
		return os.Rename(from, to)
	})
	if !failedInstall || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("commit failure not observed: %v", err)
	}
	restored, err := companionDigests(previous)
	if err != nil || !maps.Equal(before, restored) {
		t.Fatalf("rollback changed signed bundle files: %v", err)
	}
	verifyNativeBundle(t, previous)
	if err = Apply(staged, filepath.Join(previous, "Contents", "MacOS", "picfetch"), ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	installed, err := companionDigests(previous)
	if err != nil || !maps.Equal(after, installed) {
		t.Fatalf("updated bundle differs from verified candidate: %v", err)
	}
	verifyNativeBundle(t, previous)
	data, err := os.ReadFile(unrelated)
	if err != nil || string(data) != "preserved" {
		t.Fatalf("neighboring user data changed: %q (%v)", data, err)
	}
}

func nativeSignedBundle(t *testing.T, root, bundle, version string) string {
	t.Helper()
	main := filepath.Join(bundle, "Contents", "MacOS", "picfetch")
	if err := os.MkdirAll(filepath.Dir(main), 0755); err != nil {
		t.Fatal(err)
	}
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err = copyFile(source, main); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>io.github.frathe.picfetch.owned-update</string><key>CFBundleExecutable</key><string>picfetch</string><key>CFBundlePackageType</key><string>APPL</string><key>CFBundleVersion</key><string>` + version + `</string></dict></plist>`
	if err = os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), []byte(plist), 0644); err != nil {
		t.Fatal(err)
	}
	packageHelper := exec.Command("go", "run", "./scripts/heicpackage", "-os", "darwin", "-arch", runtime.GOARCH, "-out", bundle)
	packageHelper.Dir = root
	if output, buildErr := packageHelper.CombinedOutput(); buildErr != nil {
		t.Fatalf("package owned helper: %v: %s", buildErr, output)
	}
	// Distinguish both helper and enclosing sealed resources between versions.
	// Otherwise a binary/plist-only update could accidentally produce the same
	// bytes as the candidate because its unchanged helper still matched.
	helperBundle := filepath.Join(bundle, "Contents", "Helpers", "HEICWorker.app")
	helperPlist := filepath.Join(helperBundle, "Contents", "Info.plist")
	helperMetadata, err := os.ReadFile(helperPlist)
	if err != nil {
		t.Fatal(err)
	}
	helperMetadata = []byte(strings.Replace(string(helperMetadata), "<string>1</string>", "<string>"+version+"</string>", 1))
	if err = os.WriteFile(helperPlist, helperMetadata, 0644); err != nil {
		t.Fatal(err)
	}
	helperSign := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", "--options", "runtime", "--entitlements", filepath.Join(root, "packaging", "heic", "macos.entitlements.plist"), helperBundle)
	if output, signErr := helperSign.CombinedOutput(); signErr != nil {
		t.Fatalf("sign changed owned helper: %v: %s", signErr, output)
	}
	finalize := exec.Command("go", "run", "./scripts/heicpackage", "-mode", "finalize", "-os", "darwin", "-arch", runtime.GOARCH, "-out", bundle)
	finalize.Dir = root
	if output, finalizeErr := finalize.CombinedOutput(); finalizeErr != nil {
		t.Fatalf("finalize changed owned helper: %v: %s", finalizeErr, output)
	}
	if err = os.WriteFile(filepath.Join(bundle, "Contents", "Resources", "owned-version"), []byte(version), 0644); err != nil {
		t.Fatal(err)
	}
	sign := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", bundle)
	if output, signErr := sign.CombinedOutput(); signErr != nil {
		t.Fatalf("sign owned bundle: %v: %s", signErr, output)
	}
	verifyNativeBundle(t, bundle)
	return bundle
}

func verifyNativeBundle(t *testing.T, bundle string) {
	t.Helper()
	command := exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", "--verbose=2", bundle)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("complete bundle signature: %v: %s", err, output)
	}
}

func TestNativeMacLegacyUpdateRequiresCompleteReinstall(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	previous := nativeSignedBundle(t, root, filepath.Join(parent, "installed", "PicFetch.app"), "1")
	for _, path := range []string{filepath.Join(previous, "Contents", "Helpers"), filepath.Join(previous, "Contents", "Resources", "heic")} {
		if err = os.RemoveAll(path); err != nil {
			t.Fatal(err)
		}
	}
	sign := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", previous)
	if output, signErr := sign.CombinedOutput(); signErr != nil {
		t.Fatalf("sign owned legacy bundle: %v: %s", signErr, output)
	}
	verifyNativeBundle(t, previous)
	stageDirectory := filepath.Join(parent, "legacy-stage")
	candidate := nativeSignedBundle(t, root, filepath.Join(stageDirectory, "PicFetch.app"), "2")
	// The released updater knows only BinaryPath/PlistPath and removes its
	// stage after a successful application. Exercise that retained legacy path.
	staged := Stage{BinaryPath: filepath.Join(candidate, "Contents", "MacOS", "picfetch"), PlistPath: filepath.Join(candidate, "Contents", "Info.plist")}
	if err = applyUnix(staged, filepath.Join(previous, "Contents", "MacOS", "picfetch"), ApplyOptions{}); err != nil {
		t.Fatal(err)
	}
	if err = RemoveStage(stageDirectory); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(previous, "Contents", "Resources", "heic", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("legacy updater unexpectedly installed helper manifest: %v", err)
	}
	if _, err = os.Stat(stageDirectory); !os.IsNotExist(err) {
		t.Fatalf("legacy stage remains recoverable: %v", err)
	}
	command := exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", previous)
	if output, verifyErr := command.CombinedOutput(); verifyErr == nil {
		t.Fatal("legacy binary/plist replacement unexpectedly preserved the new enclosing signature")
	} else {
		t.Logf("owned legacy update requires complete reinstall: %v: %s", verifyErr, output)
	}
}
