package locationmap

import (
	"context"
	"math"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/ui/mapstyle"
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
	f.cancelTileRequests()
	if f.surface != nil {
		clearTileLayer(f.surface.tileLayer)
	}
}

func (f *Feature) cancelTileRequests() {
	if f.tileCancel != nil {
		f.tileCancel()
		f.tileCancel = nil
	}
	f.tileRevision++
}

func clearTileLayer(layer *fyne.Container) {
	for _, object := range layer.Objects {
		if img, ok := object.(*canvas.Image); ok {
			img.Image = nil
			img.Refresh()
		}
	}
	layer.RemoveAll()
}

func (f *Feature) updateTiles() {
	s := f.surface
	view := tileView{s.centerX, s.centerY, s.scale, s.Size()}
	if f.tileCancel != nil && view == f.tileView {
		return
	}
	// Retire network demand without removing the last painted scene. New tiles
	// remain detached until the complete current viewport can replace it at once.
	f.cancelTileRequests()
	f.movePaintedTiles(view)
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

// Move the retained scene through the same camera transform as the photos.
// Updating its pose after each input avoids accumulating a second drag offset.
func (f *Feature) movePaintedTiles(view tileView) {
	previous := f.paintedTileView
	if previous.scale > 0 {
		ratio := float32(view.scale / previous.scale)
		dx := float32((previous.x-view.x)*view.scale) + view.size.Width/2 - previous.size.Width/2*ratio
		dy := float32((previous.y-view.y)*view.scale) + view.size.Height/2 - previous.size.Height/2*ratio
		for _, object := range f.surface.tileLayer.Objects {
			position, size := object.Position(), object.Size()
			object.Move(fyne.NewPos(position.X*ratio+dx, position.Y*ratio+dy))
			object.Resize(fyne.NewSize(size.Width*ratio, size.Height*ratio))
		}
	}
	f.paintedTileView = view
}

func (f *Feature) requestTiles(ctx context.Context, placements []tilePlacement, revision uint64) {
	queue, store, wait := f.ui, f.tiles, f.tileWait
	// Called on UI: successful detached tiles survive a retry. Workers only read
	// this captured demand; they never inspect UI-owned canvas image state.
	var remaining []tilePlacement
	for _, placement := range placements {
		if placement.images[0].Image == nil {
			remaining = append(remaining, placement)
		}
	}
	f.workers.Go(func() {
		var workers sync.WaitGroup
		var mu sync.Mutex
		var retry time.Time
		for lane := range min(4, len(remaining)) {
			workers.Go(func() {
				select {
				case f.tileSlots <- struct{}{}:
				case <-ctx.Done():
					return
				}
				defer func() { <-f.tileSlots }()
				for index := lane; index < len(remaining); index += 4 {
					if ctx.Err() != nil {
						return
					}
					placement := remaining[index]
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
								img.Image = mapstyle.ForTheme(pixels)
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
		queue.Do(func() {
			if ctx.Err() != nil || revision != f.tileRevision || !f.Visible() {
				return
			}
			var objects []fyne.CanvasObject
			for _, placement := range placements {
				for _, img := range placement.images {
					if img.Image == nil {
						return
					}
					// Detached tiles may have arrived before a theme change.
					img.Image = mapstyle.ForTheme(img.Image)
					objects = append(objects, img)
				}
			}
			clearTileLayer(f.surface.tileLayer)
			f.surface.tileLayer.Objects = objects
			f.paintedTileView = f.tileView
			f.surface.tileLayer.Refresh()
		})
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
