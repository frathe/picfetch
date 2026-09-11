package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/displays"
	"github.com/frathe/picfetch/internal/imaging"
	mosaiccore "github.com/frathe/picfetch/internal/mosaic"
	"github.com/frathe/picfetch/internal/ui/mosaicwin"
)

// mosaicSources snapshots the loaded collection, or the current Grid subject
// when Grid View is open. Explicit selection is exclusive except
// that a selected duplicate currently hidden by the Grid resolves to the
// group's highest-resolution representative. Without a selection, every
// member of the filtered Grid result is used.
func (v *viewer) mosaicSources() ([]fyne.URI, error) {
	if !v.grid.Visible() {
		sources := make([]fyne.URI, v.FileCount())
		for i := range sources {
			sources[i] = v.FileAt(i)
		}
		return sources, nil
	}
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

func (v *viewer) canMosaic() bool {
	if v.comparisonActive() {
		return false
	}
	if v.grid.Visible() {
		return len(v.grid.ResultIndexes()) > 0
	}
	return v.FileCount() > 0
}

// showMosaic is the guarded Window-menu and keyboard entry. An already-open window is
// raised before resolving anything so its original command-entry snapshot can
// never be silently retargeted.
func (v *viewer) showMosaic() {
	if v.comparisonActive() || !v.yieldCopySelection() {
		return
	}
	if v.mosaicWin.Opened() {
		v.mosaicWin.Show(mosaicwin.Snapshot{})
		return
	}
	if !v.canMosaic() {
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
	kind := mosaicwin.SourceFiles
	if v.grid.Visible() {
		kind = mosaicwin.SourceResult
		if v.grid.SelectionCount() > 0 {
			kind = mosaicwin.SourceSelection
		}
	}
	snapshot, err := mosaicwin.NewSnapshot(sources, kind, topology)
	if err != nil {
		fyne.LogError("could not build mosaic snapshot", err)
		v.ShowToast(fmt.Sprintf(lang.L("Could not open image mosaic: %v"), err))
		return
	}
	v.mosaicWin.Show(snapshot)
}

func (v *viewer) GenerateMosaic(ctx context.Context, request mosaiccore.Request, report func(mosaiccore.Progress)) (mosaiccore.Result, error) {
	return mosaiccore.GenerateWithProgress(ctx, request, report)
}

func (v *viewer) InspectMosaicDisplays() (displays.Snapshot, error) {
	return displays.Inspect(v.win)
}

// AfterFileExported reconciles a mosaic destination that may alias a source.
func (v *viewer) AfterFileExported(result imaging.WriteResult) {
	if result.Committed {
		v.imgCache.Purge()
	}
	v.afterFileWrite(result, true, true, func() {})
}
