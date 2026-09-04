package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/displays"
	mosaiccore "github.com/frathe/picfetch/internal/mosaic"
	"github.com/frathe/picfetch/internal/ui/mosaicwin"
)

// mosaicSources resolves the current Grid subject on the UI goroutine and
// immediately snapshots its URIs. Explicit selection is exclusive except
// that a selected duplicate currently hidden by the Grid resolves to the
// group's highest-resolution representative. Without a selection, every
// member of the filtered Grid result is used.
func (v *viewer) mosaicSources() ([]fyne.URI, error) {
	indices := v.grid.Selection()
	if len(indices) == 0 {
		indices = v.grid.ResultIndexes()
	} else if !v.grid.BrowsingDuplicates() {
		visibility := v.dupes.Visibility()
		if visibility.Hide {
			resolved := make([]int, 0, len(indices))
			seen := make(map[int]struct{}, len(indices))
			for _, index := range indices {
				if visibility.HiddenExtra(index) {
					index = visibility.RepresentativeOf(index)
				}
				if _, exists := seen[index]; exists {
					continue
				}
				seen[index] = struct{}{}
				resolved = append(resolved, index)
			}
			indices = resolved
		}
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("mosaic source pool is empty")
	}

	sources := make([]fyne.URI, 0, len(indices))
	for _, index := range indices {
		if index < 0 || index >= v.FileCount() {
			return nil, fmt.Errorf("mosaic source index %d is outside the file set", index)
		}
		uri := v.FileAt(index)
		if uri == nil {
			return nil, fmt.Errorf("mosaic source index %d has no URI", index)
		}
		sources = append(sources, uri)
	}

	return sources, nil
}

// showMosaic is the guarded Actions-menu entry. An already-open window is
// raised before resolving anything so its original command-entry snapshot can
// never be silently retargeted.
func (v *viewer) showMosaic() {
	if v.mosaicWin.Opened() {
		v.mosaicWin.Show(mosaicwin.Snapshot{})
		return
	}
	if !v.grid.Visible() || len(v.grid.ResultIndexes()) == 0 {
		return
	}
	sources, err := v.mosaicSources()
	if err != nil {
		fyne.LogError("could not resolve mosaic sources", err)
		v.ShowToast(fmt.Sprintf(lang.L("Could not open image mosaic: %v"), err))
		return
	}
	topology, err := displays.Inspect(v.win)
	if err != nil {
		fyne.LogError("could not inspect displays for mosaic", err)
		v.ShowToast(fmt.Sprintf(lang.L("Could not inspect displays: %v"), err))
		return
	}
	kind := mosaicwin.SourceResult
	if v.grid.SelectionCount() > 0 {
		kind = mosaicwin.SourceSelection
	}
	snapshot, err := mosaicwin.NewSnapshot(sources, kind, topology)
	if err != nil {
		fyne.LogError("could not build mosaic snapshot", err)
		v.ShowToast(fmt.Sprintf(lang.L("Could not open image mosaic: %v"), err))
		return
	}
	v.mosaicWin.Show(snapshot)
}

func (v *viewer) GenerateMosaic(ctx context.Context, request mosaiccore.Request) (mosaiccore.Result, error) {
	return mosaiccore.Generate(ctx, request)
}

func (v *viewer) InspectMosaicDisplays() (displays.Snapshot, error) {
	return displays.Inspect(v.win)
}
