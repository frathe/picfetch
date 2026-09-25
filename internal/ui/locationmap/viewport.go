package locationmap

import (
	"context"
	"math"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"
)

type tileView struct {
	x, y, scale float64
	size        fyne.Size
}
type tilePlacement struct {
	key    TileKey
	images []*canvas.Image
}

// ConfigureTiles replaces transport, budgets and time before a browsing session.
// A nil wait uses a cancellable timer. The store survives close/reopen.
func (f *Feature) ConfigureTiles(options TileOptions, wait func(context.Context, time.Duration) error) {
	f.suspendTiles()
	f.tiles = NewTileStore(options)
	if wait == nil {
		wait = waitForTileRetry
	}
	f.tileWait = wait
}

// TileUsage reports cache-owned encoded/decoded bytes, not mounted references.
func (f *Feature) TileUsage() (int64, int64) { return f.tiles.Usage() }

func waitForTileRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (f *Feature) suspendTiles() {
	if f.tileCancel != nil {
		f.tileCancel()
		f.tileCancel = nil
	}
	f.tileRevision++
	if f.surface != nil {
		for _, object := range f.surface.tileLayer.Objects {
			if img, ok := object.(*canvas.Image); ok {
				img.Image = nil
			}
		}
		f.surface.tileLayer.RemoveAll()
	}
}

func (f *Feature) updateTiles() {
	s := f.surface
	view := tileView{s.centerX, s.centerY, s.scale, s.Size()}
	if f.tileCancel != nil && view == f.tileView {
		return
	}
	f.suspendTiles()
	f.tileView = view
	if view.size.Width <= 0 || view.size.Height <= 0 {
		return
	}
	ctx, cancel := context.WithCancel(f.ctx)
	f.tileCancel = cancel
	z := max(0, min(19, int(math.Floor(math.Log2(view.scale/256)))))
	n := int(math.Exp2(float64(z)))
	edge := view.scale / float64(n)
	left := view.x*float64(n) - float64(view.size.Width)/(2*edge)
	top := view.y*float64(n) - float64(view.size.Height)/(2*edge)
	byKey := map[TileKey]int{}
	var placements []tilePlacement
	count := 0
	for y := int(math.Floor(top)); y <= int(math.Floor(top+float64(view.size.Height)/edge)); y++ {
		if y < 0 || y >= n {
			continue
		}
		for x := int(math.Floor(left)); x <= int(math.Floor(left+float64(view.size.Width)/edge)); x++ {
			// Bound rendered references as well as the store, even for enormous windows.
			if count >= 128 {
				break
			}
			count++
			key := TileKey{z, ((x % n) + n) % n, y}
			img := canvas.NewImageFromImage(nil)
			img.FillMode = canvas.ImageFillStretch
			img.Move(fyne.NewPos(float32((float64(x)-left)*edge), float32((float64(y)-top)*edge)))
			img.Resize(fyne.NewSize(float32(edge+1), float32(edge+1)))
			s.tileLayer.Add(img)
			index, ok := byKey[key]
			if !ok {
				index = len(placements)
				byKey[key] = index
				placements = append(placements, tilePlacement{key: key})
			}
			placements[index].images = append(placements[index].images, img)
		}
	}
	f.requestTiles(ctx, placements, f.tileRevision)
}

func (f *Feature) requestTiles(ctx context.Context, placements []tilePlacement, revision uint64) {
	queue, store, wait := f.ui, f.tiles, f.tileWait
	f.workers.Go(func() {
		var workers sync.WaitGroup
		var mu sync.Mutex
		var retry time.Time
		for lane := range min(4, len(placements)) {
			workers.Go(func() {
				select {
				case f.tileSlots <- struct{}{}:
				case <-ctx.Done():
					return
				}
				defer func() { <-f.tileSlots }()
				for index := lane; index < len(placements); index += 4 {
					if ctx.Err() != nil {
						return
					}
					placement := placements[index]
					pixels, next, err := store.Fetch(ctx, placement.key)
					if ctx.Err() != nil {
						return
					}
					if err != nil {
						mu.Lock()
						if retry.IsZero() || next.Before(retry) {
							retry = next
						}
						mu.Unlock()
					}
					queue.Do(func() {
						if ctx.Err() != nil || revision != f.tileRevision || !f.Visible() {
							return
						}
						if err == nil {
							for _, img := range placement.images {
								img.Image = pixels
								img.Refresh()
							}
							return
						}
						now := store.now()
						if f.lastTileToast.IsZero() || now.Sub(f.lastTileToast) >= time.Minute {
							f.lastTileToast = now
							f.host.ShowToast(lang.L("Could not load map tiles. An internet connection is required."))
						}
					})
				}
			})
		}
		workers.Wait()
		if retry.IsZero() || ctx.Err() != nil {
			return
		}
		f.retryWorkers.Go(func() {
			if wait(ctx, max(0, retry.Sub(store.now()))) != nil || ctx.Err() != nil {
				return
			}
			queue.Do(func() {
				if ctx.Err() == nil && revision == f.tileRevision && f.Visible() {
					f.requestTiles(ctx, placements, revision)
				}
			})
		})
	})
}
