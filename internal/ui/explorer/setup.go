package explorer

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

	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/assets"
)

type explorerSetup struct {
	ready    func()
	panel    *widget.PopUp
	op       requestLifecycle
	status   *widget.Label
	progress *widget.ProgressBar
	primary  *widget.Button
}

func (f *Feature) prepareExplorer(ready func()) {
	if f.setup != nil {
		return
	}
	s := &explorerSetup{ready: ready}
	f.setup = s
	art := canvas.NewImageFromResource(fyne.NewStaticResource("explorer-intro.png", assets.ExplorerIntroPNG))
	art.FillMode = canvas.ImageFillContain
	art.SetMinSize(fyne.NewSize(320, 240))
	explanation := widget.NewLabel(lang.L("Find and group similar pictures with the SigLIP 2 AI model."))
	explanation.Wrapping = fyne.TextWrapWord
	download := widget.NewLabel(fmt.Sprintf(lang.L("One-time download: about %.0f MB. After that, everything works offline."), float64(similarity.AssetDownloadBytes())/1e6))
	download.Wrapping = fyne.TextWrapWord
	s.status = widget.NewLabel(lang.L("Checking AI model..."))
	s.status.Wrapping = fyne.TextWrapWord
	s.progress = widget.NewProgressBar()
	s.progress.Hide()
	s.primary = widget.NewButton(lang.L("Continue"), func() { f.finishExplorerSetup(s) })
	s.primary.Importance = widget.HighImportance
	s.primary.Disable()
	cancel := widget.NewButton(lang.L("Cancel"), f.closeExplorerSetup)
	discussions := widget.NewHyperlink(lang.L("GitHub Discussions"), nil)
	discussions.OnTapped = f.discussions
	policy := widget.NewHyperlink(lang.L("Privacy policy"), nil)
	policy.OnTapped = func() {
		address, err := url.Parse("https://github.com/frathe/picfetch/blob/main/PRIVACY.md")
		if err == nil {
			err = f.app.OpenURL(address)
		}
		if err != nil {
			fyne.LogError("open privacy policy", err)
		}
	}
	reading := container.NewVScroll(container.NewVBox(art, explanation, download))
	reading.SetMinSize(fyne.NewSize(320, 160))
	footer := container.NewVBox(s.status, s.progress,
		container.NewHBox(discussions, policy),
		container.NewHBox(layout.NewSpacer(), cancel, s.primary))
	title := widget.NewLabelWithStyle(lang.L("Similarity Explorer"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	body := container.NewPadded(container.NewBorder(title, footer, nil, nil, reading))
	s.panel = widget.NewModalPopUp(container.New(explorerSetupLayout{f.win.Canvas()}, body), f.win.Canvas())
	f.win.Resize(f.win.Canvas().Size().Max(fyne.NewSize(720, 660)))
	s.panel.Show()
	if !f.supported {
		s.status.SetText(lang.L("Similarity Explorer requires an Intel Mac with macOS 13.4 or newer, an Apple Silicon Mac, Linux x64/ARM64 or Windows 11 x64/ARM64."))
		s.primary.Hide()
		return
	}
	if f.assetsReady {
		f.explorerSetupReady(s)
		return
	}
	token := s.op.begin()
	client := f.client
	f.workers.Go(func() {
		err := client.CheckAssets(token.context())
		f.ui.Do(func() {
			if !token.current() {
				return
			}
			if err == nil {
				f.assetsReady = true
				f.explorerSetupReady(s)
				return
			}
			s.status.SetText(lang.L("Download the AI model to get started."))
			s.primary.SetText(lang.L("Download"))
			s.primary.OnTapped = func() { f.downloadExplorerAssets(s) }
			s.primary.Enable()
		})
	})
}

func (f *Feature) explorerSetupReady(s *explorerSetup) {
	if f.introSeen {
		f.finishExplorerSetup(s)
		return
	}
	s.status.SetText(lang.L("The AI model is ready. No download is needed."))
	s.primary.Enable()
}

func (f *Feature) finishExplorerSetup(s *explorerSetup) {
	if f.setup != s || !f.assetsReady || f.stopping {
		return
	}
	f.introSeen = true
	f.closeExplorerSetup()
	if s.ready != nil {
		s.ready()
	}
}

func (f *Feature) closeExplorerSetup() {
	if s := f.setup; s != nil {
		s.op.invalidate()
		s.panel.Hide()
		f.setup = nil
	}
}

// The page fills its modal overlay at every canvas size. The reading area can
// scroll independently while the title and actions remain in the viewport.
type explorerSetupLayout struct{ canvas fyne.Canvas }

func (l explorerSetupLayout) MinSize(_ []fyne.CanvasObject) fyne.Size { return l.canvas.Size() }
func (_ explorerSetupLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	layout.NewStackLayout().Layout(objects, size)
}

func (f *Feature) downloadExplorerAssets(s *explorerSetup) {
	if f.setup != s || f.stopping {
		return
	}
	token := s.op.begin()
	s.primary.Disable()
	s.status.SetText(lang.L("Downloading AI model..."))
	s.progress.SetValue(0)
	s.progress.Show()
	client := f.client
	f.workers.Go(func() {
		last := time.Time{}
		directory, err := client.InstallAssets(token.context(), func(p similarity.DownloadProgress) {
			if !token.current() || time.Since(last) < 100*time.Millisecond && p.Received < p.Total {
				return
			}
			last = time.Now()
			f.ui.Do(func() {
				if !token.current() {
					return
				}
				s.progress.SetValue(float64(p.Received) / float64(p.Total))
				s.status.SetText(fmt.Sprintf(lang.L("Downloading: %.1f of %.1f MB"), float64(p.Received)/1e6, float64(p.Total)/1e6))
				if p.Received == p.Total {
					s.status.SetText(lang.L("Verifying AI model..."))
				}
			})
		})
		f.ui.Do(func() {
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
			f.client.Assets = directory
			f.assetsReady = true
			f.finishExplorerSetup(s)
		})
	})
}

// EnsureReady presents setup only when needed. A retired setup cannot invoke ready.
func (f *Feature) EnsureReady(ready func()) bool {
	if f.stopping {
		return false
	}
	if f.introSeen && (f.assetsReady || f.analyze != nil) {
		return true
	}
	f.prepareExplorer(ready)
	return false
}
