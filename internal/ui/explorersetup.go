package ui

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/assets"
)

type explorerSetup struct {
	panel    *widget.PopUp
	op       requestLifecycle
	status   *widget.Label
	progress *widget.ProgressBar
	primary  *widget.Button
}

func (v *viewer) prepareExplorer() {
	if v.explorer.setup != nil {
		return
	}
	s := &explorerSetup{}
	v.explorer.setup = s
	art := canvas.NewImageFromResource(fyne.NewStaticResource("explorer-intro.png", assets.ExplorerIntroPNG))
	art.FillMode = canvas.ImageFillContain
	art.SetMinSize(fyne.NewSize(320, 240))
	explanation := widget.NewLabel(lang.L("Find pictures that belong together. Explore similar images, filter by subjects, and organize your own cohorts."))
	explanation.Wrapping = fyne.TextWrapWord
	privacy := widget.NewLabel(lang.L("Your pictures stay on your computer. Analysis runs locally, without uploads or analytics."))
	privacy.Wrapping = fyne.TextWrapWord
	download := widget.NewLabel(fmt.Sprintf(lang.L("First-time setup downloads about %.0f MB from Hugging Face and Microsoft GitHub. After setup, analysis works offline."), float64(similarity.AssetDownloadBytes())/1e6))
	if distribution.StoreManaged {
		download.SetText(fmt.Sprintf(lang.L("First-time setup downloads about %.0f MB of model data from Hugging Face. The runtime is included and updated through Microsoft Store. After setup, analysis works offline."), float64(similarity.AssetDownloadBytes())/1e6))
	}
	download.Wrapping = fyne.TextWrapWord
	s.status = widget.NewLabel(lang.L("Checking local model..."))
	s.status.Wrapping = fyne.TextWrapWord
	s.progress = widget.NewProgressBar()
	s.progress.Hide()
	s.primary = widget.NewButton(lang.L("Continue"), func() { v.finishExplorerSetup(s) })
	s.primary.Importance = widget.HighImportance
	s.primary.Disable()
	cancel := widget.NewButton(lang.L("Cancel"), v.closeExplorerSetup)
	discussions := widget.NewHyperlink(lang.L("GitHub Discussions"), nil)
	discussions.OnTapped = v.help.ShowDiscussions
	policy := widget.NewHyperlink(lang.L("Privacy policy"), nil)
	policy.OnTapped = func() {
		address, err := url.Parse("https://github.com/frathe/picfetch/blob/main/PRIVACY.md")
		if err == nil {
			err = v.app.OpenURL(address)
		}
		if err != nil {
			fyne.LogError("open privacy policy", err)
		}
	}
	reading := container.NewVScroll(container.NewVBox(art, explanation, privacy, download))
	reading.SetMinSize(fyne.NewSize(320, 160))
	footer := container.NewVBox(s.status, s.progress,
		container.NewHBox(discussions, policy),
		container.NewHBox(layout.NewSpacer(), cancel, s.primary))
	if !v.explorer.networkIsolation {
		notice := widget.NewLabel(lang.L("Windows: Analysis runs locally with ONNX Runtime telemetry disabled. No network port is opened. Windows does not block the analysis process from accessing the network."))
		notice.Wrapping = fyne.TextWrapWord
		footer.Objects = append([]fyne.CanvasObject{notice}, footer.Objects...)
	}
	title := widget.NewLabelWithStyle(lang.L("Visual Similarity Explorer"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	body := container.NewPadded(container.NewBorder(title, footer, nil, nil, reading))
	s.panel = widget.NewModalPopUp(container.New(explorerSetupLayout{v.win.Canvas()}, body), v.win.Canvas())
	v.win.Resize(v.win.Canvas().Size().Max(fyne.NewSize(720, 660)))
	s.panel.Show()
	if v.explorer.assetsReady {
		v.explorerSetupReady(s)
		return
	}
	if !v.explorer.supported {
		s.status.SetText(lang.L("Visual Similarity Explorer requires an Apple Silicon Mac, 64-bit x86 Linux or Windows 11 x64/ARM64."))
		s.primary.Hide()
		return
	}
	token := s.op.begin()
	client := v.explorer.client
	v.explorer.workers.Go(func() {
		err := client.CheckAssets(token.context())
		v.explorer.ui.Do(func() {
			if !token.current() {
				return
			}
			if err == nil {
				v.explorer.assetsReady = true
				v.explorerSetupReady(s)
				return
			}
			s.status.SetText(lang.L("Download the local model to get started."))
			s.primary.SetText(lang.L("Download"))
			s.primary.OnTapped = func() { v.downloadExplorerAssets(s) }
			s.primary.Enable()
		})
	})
}

func (v *viewer) explorerSetupReady(s *explorerSetup) {
	if v.explorer.introSeen {
		v.finishExplorerSetup(s)
		return
	}
	s.status.SetText(lang.L("The local model is ready. No download is needed."))
	s.primary.Enable()
}

func (v *viewer) finishExplorerSetup(s *explorerSetup) {
	if v.explorer.setup != s || !v.explorer.assetsReady || v.stopping {
		return
	}
	v.explorer.introSeen = true
	preferences.Save(v.app, v.currentPreferences())
	v.closeExplorerSetup()
	v.showExplorer()
}

func (v *viewer) closeExplorerSetup() {
	if s := v.explorer.setup; s != nil {
		s.op.invalidate()
		s.panel.Hide()
		v.explorer.setup = nil
	}
}

// The page fills its modal overlay at every canvas size. The reading area can
// scroll independently while the title and actions remain in the viewport.
type explorerSetupLayout struct{ canvas fyne.Canvas }

func (l explorerSetupLayout) MinSize(_ []fyne.CanvasObject) fyne.Size { return l.canvas.Size() }
func (_ explorerSetupLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	layout.NewStackLayout().Layout(objects, size)
}

func (v *viewer) downloadExplorerAssets(s *explorerSetup) {
	if v.explorer.setup != s || v.stopping {
		return
	}
	token := s.op.begin()
	s.primary.Disable()
	s.status.SetText(lang.L("Downloading local model..."))
	s.progress.SetValue(0)
	s.progress.Show()
	client := v.explorer.client
	v.explorer.workers.Go(func() {
		last := time.Time{}
		directory, err := client.InstallAssets(token.context(), func(p similarity.DownloadProgress) {
			if !token.current() || time.Since(last) < 100*time.Millisecond && p.Received < p.Total {
				return
			}
			last = time.Now()
			v.explorer.ui.Do(func() {
				if !token.current() {
					return
				}
				s.progress.SetValue(float64(p.Received) / float64(p.Total))
				s.status.SetText(fmt.Sprintf(lang.L("Downloading: %.1f of %.1f MB"), float64(p.Received)/1e6, float64(p.Total)/1e6))
				if p.Received == p.Total {
					s.status.SetText(lang.L("Verifying local model..."))
				}
			})
		})
		v.explorer.ui.Do(func() {
			if !token.current() {
				return
			}
			if err != nil {
				fyne.LogError("install similarity assets", err)
				s.status.SetText(lang.L("Setup could not finish. Check your connection and available disk space, then retry."))
				if errors.Is(err, similarity.ErrBundledRuntimeUnavailable) {
					s.status.SetText(lang.L("The bundled analysis runtime is missing or damaged. Repair or update PicFetch through Microsoft Store, then retry."))
				}
				s.primary.SetText(lang.L("Retry"))
				s.primary.Enable()
				return
			}
			v.explorer.client.Assets = directory
			v.explorer.assetsReady = true
			v.finishExplorerSetup(s)
		})
	})
}
