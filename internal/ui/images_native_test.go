//go:build heicnative && (darwin || linux || windows) && (amd64 || arm64)

package ui

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/heicdecode"
	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/imaging"
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
	if failure := os.Getenv("PICFETCH_HEIC_ACTIVATION_FAILURE"); failure != "" {
		if failure == "sandbox readiness" {
			t.Run("TestExperimentalHEICReadinessRefusal", testNativeHEICReadinessRefusal)
		} else {
			t.Run("TestExperimentalHEICUnavailable/"+failure, testNativeHEICUnavailable)
		}
		return
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
	preferences.Save(testApp, preferences.State{MaxFileSizeMB: 1})
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
	ordinary := storage.NewFileURI(filepath.Join(filepath.Dir(uri.Path()), "ordinary.png"))
	if err = os.WriteFile(ordinary.Path(), uitest.EncodePNG(t, 3, 2, color.White), 0600); err != nil {
		t.Fatal(err)
	}
	t.Run("enable stays inactive until restart", func(t *testing.T) {
		v, _, _ := newTestUI(t)
		if v.images.owner != nil || v.images.foreground.IsSupportedImage(uri) {
			t.Fatal("default startup enabled HEIC")
		}
		check := experimentalHEICCheckbox(t, v)
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
		t.Run("TestExperimentalHEICMixedDirectoryNavigation", func(t *testing.T) {
			dropAndWait(t, v, storage.NewFileURI(filepath.Dir(uri.Path())))
			if len(v.state.files) != 3 {
				t.Fatalf("mixed directory admitted %d images, want 3", len(v.state.files))
			}
			for i, file := range v.state.files {
				v.ShowImage(i)
				waitUntilLoaded(t, v)
				want := image.Pt(16, 16)
				if file.String() == ordinary.String() {
					want = image.Pt(3, 2)
				}
				if v.img.Image == nil || v.img.Image.Bounds().Size() != want || v.display.Snapshot().Displayed.Source.String() != file.String() {
					t.Fatalf("mixed directory navigation did not display %s", file.Name())
				}
			}
			v.display.Settle()
		})
		t.Run("live file-size limit", func(t *testing.T) {
			// A valid free-space box grows the owned image above the initial
			// one-MiB preference without changing its decoded pixels.
			free := make([]byte, 1024*1024)
			binary.BigEndian.PutUint32(free, uint32(len(free)))
			copy(free[4:], "free")
			large := storage.NewFileURI(uitest.WriteTempFile(t, "larger.heic", append(slices.Clone(fixture), free...)))
			owner := v.images.owner
			for _, reader := range []imaging.Reader{v.images.foreground, v.images.background} {
				v.SetMaxFileSizeMB(1)
				var tooLarge *imaging.InputTooLargeError
				if _, err := reader.Read(context.Background(), large); !errors.As(err, &tooLarge) {
					t.Fatalf("lowered live limit admitted the image: %v", err)
				}
				v.SetMaxFileSizeMB(2)
				source, err := reader.Read(context.Background(), large)
				if err != nil {
					t.Fatalf("raised live limit still rejected the image: %v", err)
				}
				if source.Bounds().Dx() != 16 || source.Bounds().Dy() != 16 || v.images.owner != owner {
					t.Fatal("live limit changed native pixels or replaced the shared owner")
				}
			}
			v.SetMaxFileSizeMB(128)
			checked := errors.New("input ceiling checked")
			_, err := owner.Do(context.Background(), heicdecode.Decode, func(_ context.Context, maxBytes int64) ([]byte, error) {
				if maxBytes != 64*1024*1024 {
					t.Errorf("native hard input ceiling changed: %d", maxBytes)
				}
				return nil, checked
			})
			if !errors.Is(err, checked) {
				t.Fatalf("native hard input ceiling was not checked: %v", err)
			}
		})
		entered, closed := make(chan struct{}), make(chan struct{})
		var readOnce, closeOnce sync.Once
		unblock := func() { closeOnce.Do(func() { close(closed) }) }
		defer unblock()
		held := uitest.ReaderURI(storage.NewFileURI(uitest.WriteTempFile(t, "a-held.heic", fixture)), func() (io.ReadCloser, error) {
			return uitest.ReadCloser{
				ReadFunc: func(_ []byte) (int, error) {
					readOnce.Do(func() { close(entered) })
					<-closed
					return 0, os.ErrClosed
				},
				CloseFunc: func() error { unblock(); return nil },
			}, nil
		})
		// The source's first byte read follows actual native readiness. Hold
		// that foreground read until navigation closes it through cancellation.
		v.handleDrop([]fyne.URI{held, ordinary})
		waitForScan(t, v)
		waitForSort(t, v)
		select {
		case <-entered:
		case <-time.After(testTimeout):
			t.Fatal("native foreground source was not admitted")
		}
		owner := v.images.owner
		test.Tap(experimentalHEICCheckbox(t, v))
		preferences.Save(testApp, v.currentPreferences())
		if v.images.owner != owner || !v.images.foreground.IsSupportedImage(uri) {
			t.Fatal("disabling replaced the running decoder")
		}
		select {
		case <-closed:
			t.Fatal("Settings retired active native work")
		default:
		}
		retired := v.display.LoadDone()
		replacement := storage.NewFileURI(uitest.WriteTempFile(t, "replacement.png", uitest.EncodePNG(t, 5, 4, color.White)))
		dropAndWait(t, v, replacement)
		waitHandle(t, "retired native HEIC load", retired)
		select {
		case <-closed:
		default:
			t.Fatal("navigation did not close the retired native source")
		}
		v.display.Settle()
		if len(v.state.files) != 1 || v.img.Image == nil || v.img.Image.Bounds().Size() != image.Pt(5, 4) || v.display.Snapshot().Displayed.Source.String() != replacement.String() {
			t.Fatal("retired native work replaced ordinary viewing")
		}
		// Disabling remains restart-only even after cancellation and replacement.
		dropAndWait(t, v, uri, heif)
		if v.img.Image == nil || v.img.Image.Bounds().Size() != image.Pt(16, 16) || v.images.owner != owner {
			t.Fatal("active capability did not survive Settings and source replacement")
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

func testNativeHEICUnavailable(t *testing.T) {
	preferences.Save(testApp, preferences.State{ExperimentalHEIC: true})
	v, _, _ := newTestUI(t)
	if v.images.owner != nil || v.images.startupError == nil || !v.images.unavailable() || v.settingsWin.Open() {
		t.Fatal("invalid installed package did not fail closed without opening Settings")
	}
	dir := t.TempDir()
	heic := storage.NewFileURI(filepath.Join(dir, "unavailable.heic"))
	if err := os.WriteFile(heic.Path(), []byte("owned source must remain unadmitted"), 0600); err != nil {
		t.Fatal(err)
	}
	ordinary := storage.NewFileURI(filepath.Join(dir, "ordinary.png"))
	if err := os.WriteFile(ordinary.Path(), uitest.EncodePNG(t, 5, 4, color.White), 0600); err != nil {
		t.Fatal(err)
	}
	dropAndWait(t, v, storage.NewFileURI(dir))
	if v.FileCount() != 1 || v.state.files[0].String() != ordinary.String() || v.img.Image == nil || v.img.Image.Bounds().Size() != image.Pt(5, 4) {
		t.Fatal("invalid native package disrupted ordinary directory viewing")
	}
	if !experimentalHEICCheckbox(t, v).Checked || !preferences.Load(testApp).ExperimentalHEIC || v.images.foreground.IsSupportedImage(heic) {
		t.Fatal("invalid native package lost intent or admitted HEIC")
	}
}

func testNativeHEICReadinessRefusal(t *testing.T) {
	preferences.Save(testApp, preferences.State{ExperimentalHEIC: true})
	v, _, _ := newTestUI(t)
	if v.images.owner == nil || v.images.unavailable() {
		t.Fatalf("readiness fixture failed before native launch: %v", v.images.startupError)
	}
	owner := v.images.owner
	var bulkRead atomic.Bool
	heic := uitest.ReaderURI(storage.NewFileURI(uitest.WriteTempFile(t, "a-unready.heic", []byte("owned"))), func() (io.ReadCloser, error) {
		return uitest.ReadCloser{
			ReadFunc:  func(_ []byte) (int, error) { bulkRead.Store(true); return 0, io.EOF },
			CloseFunc: func() error { return nil },
		}, nil
	})
	ordinary := storage.NewFileURI(uitest.WriteTempFile(t, "ordinary.png", uitest.EncodePNG(t, 5, 4, color.White)))
	dropAndWait(t, v, heic, ordinary)
	if bulkRead.Load() || !v.images.unavailable() || v.images.owner != owner {
		t.Fatal("native readiness refusal read source bytes, lost status or replaced ownership")
	}
	if v.FileCount() != 1 || v.state.files[0].String() != ordinary.String() || v.img.Image == nil || v.img.Image.Bounds().Size() != image.Pt(5, 4) || v.settingsWin.Open() {
		t.Fatal("native readiness refusal disrupted ordinary viewing or opened Settings")
	}
	if !experimentalHEICCheckbox(t, v).Checked || !v.images.foreground.IsSupportedImage(heic) {
		t.Fatal("native readiness refusal changed saved intent or immutable capability")
	}
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
	run := func(t *testing.T, command *exec.Cmd) {
		t.Helper()
		command.Dir = repository
		output, err := command.CombinedOutput()
		t.Logf("%s\n%s", command.String(), output)
		if err != nil {
			t.Fatalf("native application fixture: %v", err)
		}
	}
	run(t, exec.Command("go", "run", "./scripts/heicpackage", "-os", runtime.GOOS, "-arch", runtime.GOARCH, "-out", root))
	if runtime.GOOS == "darwin" {
		plist := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>com.frathe.PicFetch.HEICQualification</string><key>CFBundleExecutable</key><string>picfetch</string><key>CFBundlePackageType</key><string>APPL</string></dict></plist>`
		if err = os.WriteFile(filepath.Join(root, "Contents", "Info.plist"), []byte(plist), 0600); err != nil {
			t.Fatal(err)
		}
		run(t, exec.Command("/usr/bin/codesign", "--force", "--sign", "-", "--options", "runtime", root))
		run(t, exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", "--verbose=2", root))
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
	run(t, command)

	// These mutations apply only to this disposable standalone fixture after
	// its successful child has exited. Installed MSIX bytes are never changed.
	helper, manifest, err := heicclient.PackagePaths(root, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	helperBytes, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	runFailure := func(t *testing.T, failure string) {
		t.Helper()
		failureContext, stop := context.WithTimeout(context.Background(), time.Minute)
		defer stop()
		probe := exec.CommandContext(failureContext, destination, "-test.run=^TestNativePackagedHEICActivation$", "-test.v", "-test.timeout=45s")
		probe.Env = append(os.Environ(), "PICFETCH_HEIC_ACTIVATION_CHILD=1", "PICFETCH_HEIC_ACTIVATION_FAILURE="+failure)
		output, err := probe.CombinedOutput()
		t.Logf("failure=%s\n%s", failure, output)
		if err != nil {
			t.Fatalf("native package refusal: %v", err)
		}
	}
	for _, failure := range []string{"missing helper", "missing manifest", "invalid manifest", "wrong target", "helper identity"} {
		t.Run(failure, func(t *testing.T) {
			// Restore both inputs before each independent failure scenario.
			if err := os.WriteFile(manifest, manifestBytes, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(helper, helperBytes, 0700); err != nil {
				t.Fatal(err)
			}
			var mutationErr error
			switch failure {
			case "missing helper":
				mutationErr = os.Remove(helper)
			case "missing manifest":
				mutationErr = os.Remove(manifest)
			case "invalid manifest":
				mutationErr = os.WriteFile(manifest, []byte("invalid owned manifest"), 0600)
			case "wrong target":
				var record heicclient.PackageManifest
				mutationErr = json.Unmarshal(manifestBytes, &record)
				if mutationErr == nil {
					record.GOARCH = "amd64"
					if runtime.GOARCH == record.GOARCH {
						record.GOARCH = "arm64"
					}
					var data []byte
					data, mutationErr = json.Marshal(record)
					if mutationErr == nil {
						mutationErr = os.WriteFile(manifest, data, 0600)
					}
				}
			case "helper identity":
				mutationErr = os.WriteFile(helper, []byte("changed owned helper"), 0700)
			}
			if mutationErr != nil {
				t.Fatal(mutationErr)
			}
			runFailure(t, failure)
		})
	}
	if runtime.GOOS == "darwin" {
		t.Run("sandbox readiness", func(t *testing.T) {
			if err := os.WriteFile(helper, helperBytes, 0700); err != nil {
				t.Fatal(err)
			}
			// Independently sign this owned negative helper without App Sandbox.
			// The real worker must refuse readiness; no production policy changes.
			entitlements := filepath.Join(t.TempDir(), "empty.plist")
			if err := os.WriteFile(entitlements, []byte(`<?xml version="1.0"?><plist version="1.0"><dict/></plist>`), 0600); err != nil {
				t.Fatal(err)
			}
			bundle := filepath.Dir(filepath.Dir(filepath.Dir(helper)))
			run(t, exec.Command("/usr/bin/codesign", "--force", "--sign", "-", "--options", "runtime", "--entitlements", entitlements, bundle))
			data, err := os.ReadFile(helper)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(data)
			var record heicclient.PackageManifest
			if err = json.Unmarshal(manifestBytes, &record); err != nil {
				t.Fatal(err)
			}
			record.ExecutableSHA256 = hex.EncodeToString(digest[:])
			data, err = json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(manifest, data, 0600); err != nil {
				t.Fatal(err)
			}
			run(t, exec.Command("/usr/bin/codesign", "--force", "--sign", "-", "--options", "runtime", root))
			run(t, exec.Command("/usr/bin/codesign", "--verify", "--deep", "--strict", "--verbose=2", root))
			runFailure(t, "sandbox readiness")
		})
	}
}
