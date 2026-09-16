package ui

import (
	"context"
	"fmt"
	"image"

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
	v.Unfocus()
	v.compare.Open(sources)
}

// loadComparedImage keeps comparison on the viewer's canonical full-image
// path and cache without changing the displayed file or removing failures
// from the file set.
func (v *viewer) loadComparedImage(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		writer := v.imgCache.Capture()
		if loaded, ok := v.imgCache.Get(uri.String()); ok {
			if writer.Current() {
				if loaded == nil || len(loaded.Frames) == 0 {
					return loaded, nil
				}
				if err := comparisonImageFits(loaded, v.imgCache.Budget()); err != nil {
					return nil, err
				}
				// Comparison displays only the first frame. Do not retain an
				// animation's unused frames outside their existing cache owner.
				frozen := *loaded
				frozen.Frames = []image.Image{loaded.Frames[0]}
				frozen.Delays = nil
				return &frozen, nil
			}
			continue
		}
		data, bounds, err := imaging.ReadAndProbe(ctx, uri)
		if err != nil {
			if !writer.Current() {
				continue
			}
			return nil, err
		}
		if imaging.EstimateDecodedBytes(bounds) > v.imgCache.Budget()/2 {
			return nil, fmt.Errorf("comparison memory budget exceeded")
		}
		// Comparison never animates, so decoding additional GIF frames would
		// consume memory that cannot contribute to either pane.
		loaded, err := imaging.DecodeRecord(ctx, data, 0)
		if !writer.Current() {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !imaging.IsAnimatedGIF(data) {
			writer.Add(uri.String(), loaded)
		}
		return loaded, nil
	}
}

func comparisonImageFits(loaded *imaging.LoadedImage, budget int64) error {
	if loaded == nil || len(loaded.Frames) == 0 {
		return nil
	}
	if imaging.EstimateDecodedBytes(loaded.Frames[0].Bounds()) > budget/2 {
		return fmt.Errorf("comparison memory budget exceeded")
	}
	return nil
}

func (v *viewer) compareFailed(uri fyne.URI, err error) {
	v.ShowToast(fmt.Sprintf(lang.L("could not read %q: %v"), uri.Name(), err))
}

func (v *viewer) compareOrderChanged(left, right string) {
	v.win.SetTitle(fmt.Sprintf(lang.L("Compare: %s | %s - PicFetch"), left, right))
}

func (v *viewer) comparisonClosed() {
	if v.searchActive() && v.searchView.pending != nil {
		v.applySearchDelivery(*v.searchView.pending)
	}
	v.applyTitle()
	v.syncMenus()
}
