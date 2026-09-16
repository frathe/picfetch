package ui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/heicdecode"
	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/imaging"
	mosaiccore "github.com/frathe/picfetch/internal/mosaic"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/uitest"
)

func ownedHEICInstallation(t *testing.T) (executable, helper, private string) {
	t.Helper()
	root := t.TempDir()
	executable = filepath.Join(root, "picfetch")
	if runtime.GOOS == "darwin" {
		root = filepath.Join(root, "PicFetch.app")
		executable = filepath.Join(root, "Contents", "MacOS", "picfetch")
	}
	if err := os.MkdirAll(filepath.Dir(executable), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("owned main executable"), 0700); err != nil {
		t.Fatal(err)
	}
	private = t.TempDir()
	helper, manifest, err := heicclient.PackagePaths(root, runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{filepath.Dir(helper), filepath.Dir(manifest)} {
		if err = os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	data := []byte("owned helper, never executed by this test")
	if err = os.WriteFile(helper, data, 0700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	record, err := json.Marshal(heicclient.PackageManifest{Version: 1, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		ExecutableSHA256: hex.EncodeToString(digest[:]), GuestSHA256: hex.EncodeToString(digest[:])})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(manifest, record, 0600); err != nil {
		t.Fatal(err)
	}
	return executable, helper, private
}

func TestExperimentalHEICStartup(t *testing.T) {
	executable, helper, private := ownedHEICInstallation(t)
	if got := installedImageServices(preferences.State{}, "missing", private); got.owner != nil || got.startupError != nil {
		t.Fatal("disabled startup attempted HEIC activation")
	}
	prefs := preferences.State{ExperimentalHEIC: true}
	missing := installedImageServices(prefs, filepath.Join(t.TempDir(), "missing"), private)
	if missing.owner != nil || missing.startupError == nil {
		t.Fatal("enabled missing package did not report unavailability")
	}
	services := installedImageServices(prefs, executable, private)
	defer func() { services.Stop(); services.Wait() }()
	if services.owner == nil || services.startupError != nil || !services.foreground.IsSupportedImage(storage.NewFileURI("photo.heic")) {
		t.Fatalf("complete opt-in package not activated: %v", services.startupError)
	}
	if runtime.GOOS == "windows" {
		// Native client tests cover the copied executable's protected identity.
		return
	}
	if err := os.WriteFile(helper, []byte("changed"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err := services.owner.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
		t.Fatal("changed helper read source input")
		return nil, nil
	})
	if !errors.Is(err, heicclient.ErrUnavailable) || !services.unavailable() {
		t.Fatalf("changed helper not reported unavailable: %v", err)
	}
}

func TestExperimentalHEICAdmission(t *testing.T) {
	reader := imaging.NewReader(func(ctx context.Context, op heicdecode.Operation, input heicclient.Input) (heicdecode.Response, error) {
		if _, err := input(ctx, 128); err != nil {
			return heicdecode.Response{}, err
		}
		result := heicdecode.Response{}
		if op == heicdecode.Decode {
			result.Image = image.NewNRGBA(image.Rect(0, 0, 3, 2))
			result.Config = image.Config{Width: 3, Height: 2}
		}
		return result, nil
	})
	for _, active := range []bool{false, true} {
		for _, route := range []string{"direct", "folder", "siblings", "restored", "favorite"} {
			name := route + "/disabled"
			services := imageServices{}
			if active {
				name = route + "/active"
				services = imageServices{foreground: reader, background: reader}
			}
			t.Run(name, func(t *testing.T) {
				v, _, _ := newTestUIWithImages(t, services)
				dir := t.TempDir()
				var files []fyne.URI
				for _, filename := range []string{"still.heic", "still.heif", "ordinary.png"} {
					data := []byte("owned source")
					if filename == "ordinary.png" {
						data = uitest.EncodePNG(t, 3, 2, color.White)
					}
					path := filepath.Join(dir, filename)
					if err := os.WriteFile(path, data, 0600); err != nil {
						t.Fatal(err)
					}
					files = append(files, storage.NewFileURI(path))
				}
				switch route {
				case "direct":
					v.handleDrop(files)
				case "folder":
					v.handleDrop([]fyne.URI{storage.NewFileURI(dir)})
				case "siblings":
					v.handleDrop(files[:1])
				case "restored":
					v.savedSession = files
					v.restoreSession()
				case "favorite":
					v.OpenFavorite(t.TempDir(), files)
				}
				waitForScan(t, v)
				want := 1
				if active {
					want = 3
				} else if route == "siblings" {
					want = 0
				}
				if want > 0 {
					waitForSort(t, v)
					waitUntilLoaded(t, v)
				}
				if len(v.state.files) != want {
					t.Fatalf("admitted %d sources, want %d", len(v.state.files), want)
				}
			})
		}
	}
}

func TestHEICSourceConsumers(t *testing.T) {
	var foreground, background atomic.Int64
	reader := func(calls *atomic.Int64) imaging.Reader {
		return imaging.NewReader(func(ctx context.Context, op heicdecode.Operation, input heicclient.Input) (heicdecode.Response, error) {
			calls.Add(1)
			data, err := input(ctx, 128)
			if err != nil {
				return heicdecode.Response{}, err
			}
			date := "2026:09:16 12:34:56"
			if string(data) == "early" {
				date = "2026:09:15 12:34:56"
			}
			result := heicdecode.Response{Metadata: &heicdecode.Metadata{Make: "Owned camera", DateTimeOriginal: date}}
			if op == heicdecode.Decode {
				result.Image = image.NewNRGBA64(image.Rect(0, 0, 8, 6))
				result.Config = image.Config{Width: 8, Height: 6}
			}
			return result, nil
		})
	}
	v, _, _ := newTestUIWithImages(t, imageServices{foreground: reader(&foreground), background: reader(&background)})
	late := storage.NewFileURI(uitest.WriteTempFile(t, "late.heic", []byte("late")))
	early := storage.NewFileURI(uitest.WriteTempFile(t, "early.heic", []byte("early")))
	files := []fyne.URI{late, early}
	dropAndWait(t, v, files...)
	v.display.Settle()
	if foreground.Load() == 0 || !v.display.Snapshot().HasEXIF {
		t.Fatal("viewer did not inject its foreground reader")
	}
	v.imgCache.Purge()
	compared, err := v.loadComparedImage(context.Background(), early)
	if err != nil || compared.FileSize != 5 || !compared.HasEXIF {
		t.Fatalf("comparison source = %v", err)
	}
	v.exif.Show()
	v.exif.Settle()
	if !strings.Contains(v.exif.Text().Text, "Owned camera") || v.exif.StripButton().Visible() {
		t.Fatal("EXIF source values or editing capability were lost")
	}
	if err := v.grid.Warm(); err != nil {
		t.Fatal(err)
	}
	if !v.grid.Cached(early) || background.Load() == 0 {
		t.Fatal("grid did not use the background reader")
	}
	v.SetSortMode(filesort.ByCaptureDate)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	if v.state.files[0].String() != early.String() {
		t.Fatal("capture-date sort ignored isolated metadata")
	}
	request, err := mosaiccore.NewRequest(files, image.Pt(320, 180), mosaiccore.DefaultSettings(), 1)
	if err != nil {
		t.Fatal(err)
	}
	result, err := v.GenerateMosaic(context.Background(), request, nil)
	if err != nil || result.Bounds().Size() != image.Pt(320, 180) {
		t.Fatalf("mosaic source = %v", err)
	}
	// Use an uncached third source so Favorite persistence must decode.
	favorite := storage.NewFileURI(uitest.WriteTempFile(t, "favorite.heic", []byte("favorite")))
	favDir := t.TempDir()
	v.SetFavoritePreviewCache(true)
	v.SyncFavoritePreviews(favDir, []fyne.URI{favorite})
	settleFavoritePreviews(t, v)
	if got := previewNames(t, favDir); len(got) != 1 {
		t.Fatalf("Favorite previews = %v", got)
	}
}

func TestHEICOwnerStopsWithViewer(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	owner, err := heicclient.New(heicclient.Config{Executable: executable, SHA256: [32]byte{1}, Limits: heicdecode.DefaultLimits(0)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { owner.Stop(); owner.Wait() }()
	v, _, _ := newTestUIWithImages(t, newImageServices(owner))
	if v.explorer.Options().Client.HEIC != owner {
		t.Fatal("Explorer received a different admission owner")
	}
	link, err := owner.Attach(context.Background(), exec.Command(executable))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { link.Stop(); link.Wait() }()
	lifecycle, ok := testApp.Lifecycle().(interface{ OnStopped() func() })
	if !ok {
		t.Fatal("test lifecycle has no stopped hook")
	}
	previous := lifecycle.OnStopped()
	t.Cleanup(func() { testApp.Lifecycle().SetOnStopped(previous) })
	registerShutdown(testApp, v)
	lifecycle.OnStopped()()
	v.waitForShutdown()
	_, err = owner.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
		t.Fatal("stopped owner read a source")
		return nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("viewer left owner admission open: %v", err)
	}
}

func experimentalHEICCheckbox(t *testing.T, v *viewer) *widget.Check {
	t.Helper()
	before := slices.Clone(testApp.Driver().AllWindows())
	if v.settingsWin.Open() {
		t.Fatal("fixture requires this viewer's Settings to be closed")
	}
	v.showSettings()
	for _, win := range testApp.Driver().AllWindows() {
		if slices.Contains(before, win) || win.Title() != lang.L("Settings") {
			continue
		}
		surface := win.Content()
		if wrapper, ok := surface.(*fyne.Container); ok && len(wrapper.Objects) == 1 {
			surface = wrapper.Objects[0]
		}
		tabs, ok := surface.(*container.AppTabs)
		if !ok {
			t.Fatal("Settings surface has no tabs")
		}
		tab := tabs.Items[len(tabs.Items)-1]
		if tab.Text != lang.L("Experimental") {
			t.Fatal("Experimental is not the last Settings tab")
		}
		t.Cleanup(func() {
			if slices.Contains(testApp.Driver().AllWindows(), win) {
				win.Close()
			}
		})
		tabs.Select(tab)
		var found *widget.Check
		var walk func(fyne.CanvasObject)
		walk = func(obj fyne.CanvasObject) {
			switch value := obj.(type) {
			case *fyne.Container:
				for _, child := range value.Objects {
					walk(child)
				}
			case *container.Scroll:
				walk(value.Content)
			case *widget.Check:
				if value.Text == lang.L("Experimental HEIC support") {
					found = value
				}
			}
		}
		walk(tab.Content)
		if found == nil {
			t.Fatal("Experimental checkbox is not in the Settings surface")
		}
		return found
	}
	t.Fatal("Settings window was not opened")
	return nil
}

func TestExperimentalHEICRestartOnly(t *testing.T) {
	// Another viewer's Settings surface must not receive this viewer's edits.
	prior := testApp.NewWindow(lang.L("Settings"))
	prior.SetContent(container.NewAppTabs(container.NewTabItem(lang.L("Experimental"), widget.NewCheck(lang.L("Experimental HEIC support"), nil))))
	prior.Show()
	t.Cleanup(func() {
		if slices.Contains(testApp.Driver().AllWindows(), prior) {
			prior.Close()
		}
	})
	before := preferences.Load(testApp)
	t.Cleanup(func() { preferences.Save(testApp, before) })
	preferences.Save(testApp, preferences.State{})
	executable, _, private := ownedHEICInstallation(t)
	heic := storage.NewFileURI("owned.heic")
	t.Run("enable requires restart", func(t *testing.T) {
		v, _, _ := newTestUI(t)
		check := experimentalHEICCheckbox(t, v)
		if check.Checked || v.images.foreground.IsSupportedImage(heic) {
			t.Fatal("HEIC enabled by default")
		}
		test.Tap(check)
		preferences.Save(testApp, v.currentPreferences())
		if !preferences.Load(testApp).ExperimentalHEIC || v.images.foreground.IsSupportedImage(heic) || v.images.owner != nil {
			t.Fatal("enabling changed the running capability or lost the preference")
		}
	})
	t.Run("disable requires restart", func(t *testing.T) {
		services := installedImageServices(preferences.Load(testApp), executable, private)
		if services.owner == nil {
			t.Fatalf("restart did not activate saved preference: %v", services.startupError)
		}
		v, _, _ := newTestUIWithImages(t, services)
		check := experimentalHEICCheckbox(t, v)
		if !check.Checked {
			t.Fatal("saved preference was not restored to the checkbox")
		}
		test.Tap(check)
		preferences.Save(testApp, v.currentPreferences())
		if preferences.Load(testApp).ExperimentalHEIC || v.images.owner != services.owner || !v.images.foreground.IsSupportedImage(heic) {
			t.Fatal("disabling replaced the running owner or lost the preference")
		}
	})
	t.Run("restart disables", func(t *testing.T) {
		services := installedImageServices(preferences.Load(testApp), executable, private)
		v, _, _ := newTestUIWithImages(t, services)
		if v.images.owner != nil || v.images.foreground.IsSupportedImage(heic) {
			t.Fatal("disabled preference did not take effect after restart")
		}
	})
	t.Run("unavailable retains intent and ordinary viewing", func(t *testing.T) {
		prefs := preferences.Load(testApp)
		prefs.ExperimentalHEIC = true
		preferences.Save(testApp, prefs)
		services := installedImageServices(prefs, filepath.Join(t.TempDir(), "missing"), private)
		v, _, _ := newTestUIWithImages(t, services)
		if !experimentalHEICCheckbox(t, v).Checked || !v.images.unavailable() {
			t.Fatal("unavailable package lost saved intent or explanation state")
		}
		ordinary := storage.NewFileURI(uitest.WriteTempFile(t, "ordinary.png", uitest.EncodePNG(t, 3, 2, color.White)))
		dropAndWait(t, v, ordinary)
		if !v.currentPreferences().ExperimentalHEIC || v.images.owner != nil || v.img.Image == nil {
			t.Fatal("ordinary viewing did not survive unavailable HEIC")
		}
	})
}
