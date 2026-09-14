package explorer_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestFeatureSetupUnsupported(t *testing.T) {
	for _, ready := range []bool{false, true} {
		t.Run(fmt.Sprintf("assets_ready_%v", ready), func(t *testing.T) {
			app := test.NewApp()
			t.Cleanup(app.Quit)
			host := &featureHost{win: app.NewWindow("Explorer")}
			f := explorer.NewFeature(host, explorer.Options{App: app, Queue: &uitest.UIQueue{}, AssetsReady: ready})
			t.Cleanup(func() { f.Stop(); f.Settle() })
			accepted := false
			if f.EnsureReady(func() { accepted = true }) {
				t.Fatal("unsupported setup admitted the caller")
			}
			f.Settle()
			offered := false
			walkFeature(host.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
				if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Continue") {
					offered = true
				}
			})
			if !f.State().SetupOpen || offered || accepted {
				t.Fatal("unsupported platform offered continuation")
			}
		})
	}
}

func TestFeatureSetupDownloadRetryCancel(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	host := &featureHost{win: app.NewWindow("Explorer")}
	requests := 0
	started := make(chan struct{})
	assets := filepath.Join(t.TempDir(), "assets")
	client := similarity.Client{Assets: assets, HTTPClient: &http.Client{Transport: featureAssetTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return nil, fmt.Errorf("connection unavailable")
		}
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}}
	f := explorer.NewFeature(host, explorer.Options{App: app, Queue: &uitest.UIQueue{}, Client: client, Supported: true})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	accepted := false
	f.EnsureReady(func() { accepted = true })
	f.Settle()
	if requests != 0 {
		t.Fatal("showing setup made an implicit network request")
	}
	test.Tap(featureButton(t, host, "Download"))
	f.Settle()
	// Store binaries preserve their bundled-runtime repair path.
	//goland:noinspection GoBoolExpressions
	if distribution.StoreManaged && requests == 0 {
		explained := false
		walkFeature(host.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
			if l, ok := o.(*widget.Label); ok && l.Text == lang.L("The bundled analysis runtime is missing or damaged. Repair or update PicFetch through Microsoft Store, then retry.") {
				explained = true
			}
		})
		if !explained {
			t.Fatal("missing Store runtime did not explain repair")
		}
		test.Tap(featureButton(t, host, "Retry"))
		f.Settle()
		test.Tap(featureButton(t, host, "Cancel"))
		f.Settle()
		if requests != 0 || f.State().SetupOpen || accepted {
			t.Fatal("Store repair admitted analysis or a download")
		}
		return
	}
	if requests != 1 || accepted {
		t.Fatal("failed download did not remain available for retry")
	}
	test.Tap(featureButton(t, host, "Retry"))
	<-started
	test.Tap(featureButton(t, host, "Cancel"))
	f.Settle()
	if f.State().SetupOpen || f.Settings().IntroSeen || accepted {
		t.Fatal("cancelled setup continued into analysis")
	}
	entries, err := os.ReadDir(filepath.Dir(assets))
	if err != nil || len(entries) != 0 {
		t.Fatalf("cancelled setup left staged assets: %v, %v", entries, err)
	}
}

type featureAssetTransport func(*http.Request) (*http.Response, error)

func (f featureAssetTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
