//go:build heicnative && (darwin || linux || windows) && (amd64 || arm64)

package ui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/heicdecode"
	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/uitest"
)

// Only the test executable understands these fixture environment variables.
// The production entry point has no HEIC override or qualification dispatcher.
func TestNativePackagedHEICActivation(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_ACTIVATION_CHILD") != "1" {
		runOwnedHEICApplication(t)
		return
	}
	verifyNativeHEICIdentity(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root, err := heicclient.InstallationRoot(executable, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	helper, digest, err := heicclient.LoadPackage(root, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	unchanged := preserveNativeHEICInstallation(t, helper)
	defer unchanged()
	t.Logf("application=%s helper=%s sha256=%x os=%s arch=%s store=%v", executable, helper, digest, runtime.GOOS, runtime.GOARCH, distribution.StoreManaged)
	before := testApp
	testApp = nativeHEICApp{App: before, cache: nativeHEICCache{Cache: before.Cache(), root: storage.NewFileURI(t.TempDir())}}
	t.Cleanup(func() { testApp = before; fyne.SetCurrentApp(before) })
	preferences.Save(testApp, preferences.State{})
	uri := storage.NewFileURI(filepath.Join(t.TempDir(), "owned.heic"))
	fixture, err := os.ReadFile(os.Getenv("PICFETCH_HEIC_ACTIVATION_FIXTURE"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(uri.Path(), fixture, 0600); err != nil {
		t.Fatal(err)
	}
	heif := storage.NewFileURI(filepath.Join(filepath.Dir(uri.Path()), "owned.heif"))
	if err = os.WriteFile(heif.Path(), fixture, 0600); err != nil {
		t.Fatal(err)
	}
	t.Run("enable stays inactive until restart", func(t *testing.T) {
		v, _, _ := newTestUI(t)
		if v.images.owner != nil || v.images.foreground.IsSupportedImage(uri) {
			t.Fatal("default startup enabled HEIC")
		}
		v.showSettings()
		check := experimentalHEICCheckbox(t)
		if check.Checked {
			t.Fatal("checkbox did not default off")
		}
		test.Tap(check)
		preferences.Save(testApp, v.currentPreferences())
		if v.images.owner != nil || !preferences.Load(testApp).ExperimentalHEIC {
			t.Fatal("checkbox changed this session or lost intent")
		}
	})
	t.Run("restarted viewer decodes through native helper", func(t *testing.T) {
		v, _, _ := newTestUI(t)
		if v.images.owner == nil {
			t.Fatalf("packaged startup unavailable: %v", v.images.startupError)
		}
		if v.storeManaged != distribution.StoreManaged || v.explorer.Options().Client.HEIC != v.images.owner {
			t.Fatal("distribution or shared ownership changed")
		}
		dropAndWait(t, v, uri, heif)
		if v.img.Image == nil || v.img.Image.Bounds().Dx() != 16 || v.img.Image.Bounds().Dy() != 16 {
			t.Fatal("native HEIC viewing did not produce the owned fixture")
		}
		if err := v.grid.Warm(); err != nil {
			t.Fatal(err)
		}
		if !v.grid.Cached(uri) || !v.grid.Cached(heif) {
			t.Fatal("native HEIC did not reach grid previews")
		}
		entered, done := make(chan struct{}), make(chan error, 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			_, decodeErr := v.images.owner.Do(ctx, heicdecode.Decode, func(ctx context.Context, _ int64) ([]byte, error) {
				close(entered)
				<-ctx.Done()
				return nil, ctx.Err()
			})
			done <- decodeErr
		}()
		select {
		case <-entered:
		case decodeErr := <-done:
			t.Fatalf("native readiness failed before cancellation: %v", decodeErr)
		case <-time.After(30 * time.Second):
			t.Fatal("native cancellation input was not admitted")
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("native cancellation: %v", err)
		}
		owner := v.images.owner
		v.showSettings()
		test.Tap(experimentalHEICCheckbox(t))
		preferences.Save(testApp, v.currentPreferences())
		if v.images.owner != owner || !v.images.foreground.IsSupportedImage(uri) {
			t.Fatal("disabling replaced the running decoder")
		}
	})
	t.Run("disabled restart and ordinary viewing", func(t *testing.T) {
		v, _, _ := newTestUI(t)
		if v.images.owner != nil || v.images.foreground.IsSupportedImage(uri) {
			t.Fatal("restart did not disable HEIC")
		}
		ordinary := storage.NewFileURI(uitest.WriteTempFile(t, "ordinary.png", uitest.EncodePNG(t, 3, 2, color.White)))
		dropAndWait(t, v, uri, ordinary)
		if len(v.state.files) != 1 || v.img.Image == nil || v.img.Image.Bounds().Dx() != 3 {
			t.Fatal("ordinary viewing changed with HEIC disabled")
		}
	})
	_, after, err := heicclient.LoadPackage(root, runtime.GOOS, runtime.GOARCH)
	if err != nil || after != digest {
		t.Fatal("activation modified the installed helper package")
	}
	t.Log("native application startup, admission, cancellation and shutdown completed")
}

type nativeHEICApp struct {
	fyne.App
	cache fyne.Cache
}

func (a nativeHEICApp) Cache() fyne.Cache { return a.cache }

type nativeHEICCache struct {
	fyne.Cache
	root fyne.URI
}

func (c nativeHEICCache) RootURI() fyne.URI { return c.root }

func runOwnedHEICApplication(t *testing.T) {
	t.Helper()
	verifyNativeHEICIdentity(t)
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	destination := filepath.Join(root, "picfetch")
	if runtime.GOOS == "windows" {
		destination += ".exe"
	}
	if runtime.GOOS == "darwin" {
		root = filepath.Join(root, "PicFetch.app")
		destination = filepath.Join(root, "Contents", "MacOS", "picfetch")
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = source.Close() }()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(output, source)
	closeErr := output.Close()
	if err = errors.Join(copyErr, closeErr); err != nil {
		t.Fatal(err)
	}
	run := func(command *exec.Cmd) {
		t.Helper()
		command.Dir = repository
		output, err := command.CombinedOutput()
		t.Logf("%s\n%s", command.String(), output)
		if err != nil {
			t.Fatalf("native application fixture: %v", err)
		}
	}
	run(exec.Command("go", "run", "./scripts/heicpackage", "-os", runtime.GOOS, "-arch", runtime.GOARCH, "-out", root))
	if runtime.GOOS == "darwin" {
		plist := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>com.frathe.PicFetch.HEICQualification</string><key>CFBundleExecutable</key><string>picfetch</string><key>CFBundlePackageType</key><string>APPL</string></dict></plist>`
		if err = os.WriteFile(filepath.Join(root, "Contents", "Info.plist"), []byte(plist), 0600); err != nil {
			t.Fatal(err)
		}
		run(exec.Command("/usr/bin/codesign", "--force", "--sign", "-", "--options", "runtime", root))
		run(exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", "--verbose=2", root))
	}
	bytes, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(bytes)
	t.Logf("owned application sha256=%s", hex.EncodeToString(digest[:]))
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, destination, "-test.run=^TestNativePackagedHEICActivation$", "-test.v", "-test.timeout=3m")
	command.Env = append(os.Environ(), "PICFETCH_HEIC_ACTIVATION_CHILD=1", fmt.Sprintf("PICFETCH_HEIC_ACTIVATION_FIXTURE=%s", filepath.Join(repository, "scripts", "heicbuild", "testdata", "tenbit.heic")))
	run(command)
}
