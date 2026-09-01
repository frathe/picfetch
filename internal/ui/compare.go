package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/imaging"
)

// comparisonActive is the composition-layer fact used by every ordinary
// command entry. Feature packages stay independent: none of them needs to
// know comparison exists, and comparison itself never learns about the
// viewer, grid, menus, or OS integrations it temporarily excludes.
func (v *viewer) comparisonActive() bool {
	return v.compare != nil && v.compare.Visible()
}

// refuseOpenDuringComparison applies the one exceptional command policy:
// ordinary commands are silent no-ops, but an OS/file-dialog open request
// needs to explain why the supplied files were deliberately discarded.
func (v *viewer) refuseOpenDuringComparison() bool {
	if !v.comparisonActive() {
		return false
	}
	v.ShowToast(lang.L("Return to Grid View before opening files"))
	return true
}

// compareSelected is the only bridge from Grid View selection into the
// comparison feature. Selection, rather than Targets, is intentional: this
// command never falls back to the highlighted cell.
func (v *viewer) compareSelected() {
	if v.comparisonActive() {
		return
	}
	selected := v.grid.Selection()
	if !v.grid.Visible() || len(selected) != 2 {
		v.ShowToast(lang.L("Select exactly 2 images to compare"))
		return
	}

	var sources [2]fyne.URI
	for i, index := range selected {
		if index < 0 || index >= v.FileCount() {
			v.ShowToast(lang.L("Select exactly 2 images to compare"))
			return
		}
		sources[i] = v.FileAt(index)
	}
	v.compare.Open(sources)
}

// loadComparedImage keeps comparison on the viewer's canonical full-image
// path and cache without changing the displayed file or removing failures
// from the file set.
func (v *viewer) loadComparedImage(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
	if loaded, ok := v.imgCache.Get(uri.String()); ok {
		return loaded, nil
	}

	data, _, err := imaging.ReadAndProbe(ctx, uri)
	if err != nil {
		return nil, err
	}
	loaded, err := imaging.DecodeLoaded(ctx, data, v.imgCache.Budget())
	if err != nil {
		return nil, err
	}
	v.imgCache.Add(uri.String(), loaded)
	return loaded, nil
}

func (v *viewer) compareFailed(uri fyne.URI, err error) {
	v.ShowToast(fmt.Sprintf(lang.L("could not read %q: %v"), uri.Name(), err))
}

func (v *viewer) compareOrderChanged(left, right string) {
	v.win.SetTitle(fmt.Sprintf(lang.L("Compare: %s | %s - PicFetch"), left, right))
}

func (v *viewer) comparisonClosed() {
	v.applyTitle()
	v.syncMenus()
}
