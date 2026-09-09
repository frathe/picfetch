package similarity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
)

func analyzeLocal(ctx context.Context, req request, controls <-chan Control, emit func(Event) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := VerifyOffline(ctx); err != nil {
		return err
	}
	if err := VerifyAssets(ctx, req.Assets); err != nil {
		return err
	}
	_ = RegisterLocalFiles()
	var encoder *Encoder
	defer func() {
		if encoder != nil {
			encoder.Close()
		}
	}()
	event := Event{Total: len(req.Paths), OfflineVerified: true, Stage: "encoding"}
	cache, cacheErr := openAnalysisCache(ctx, req.FavoritesDir)
	defer cache.close()
	if cacheErr != nil {
		event.CacheWarning = cacheErr.Error()
	}
	items := make([]Item, 0, len(req.Paths))
	automatic, requested := false, false
	readControls := func() {
		for controls != nil {
			select {
			case control, ok := <-controls:
				if !ok {
					controls = nil
					return
				}
				automatic = control.Automatic
				requested = requested || control.Update
			default:
				return
			}
		}
	}
	publishMap := func(complete bool) error {
		event.Stage = "layout"
		if err := emit(event); err != nil {
			return err
		}
		if event.Successful > 0 {
			if err := Group(ctx, items, map[string]float64{}); err != nil {
				return err
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		event.Items = append([]Item(nil), items...)
		event.Complete = complete
		event.Stage = "encoding"
		if complete {
			event.Stage = "complete"
		}
		err := emit(event)
		event.Items = nil
		return err
	}
	for _, path := range req.Paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		item := Item{Path: path}
		var sourceErr error
		if !filepath.IsAbs(path) {
			sourceErr = fmt.Errorf("source must be an absolute local path")
		}
		var before os.FileInfo
		if sourceErr == nil {
			before, sourceErr = os.Stat(path)
		}
		reused := false
		if sourceErr == nil {
			item.Size, item.ModifiedNS = before.Size(), before.ModTime().UnixNano()
			if cached, ok := cache.read(item); ok {
				item, reused = cached, true
			}
		}
		if sourceErr == nil && !reused {
			if encoder == nil {
				var err error
				encoder, err = NewEncoder(req.Assets, "cpu")
				if err != nil {
					return err
				}
			}
			data, _, readErr := imaging.ReadAndProbe(ctx, storage.NewFileURI(path))
			sourceErr = readErr
			if sourceErr == nil {
				item.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
				loaded, decodeErr := imaging.DecodeLoaded(ctx, data, 1)
				sourceErr = decodeErr
				if sourceErr == nil {
					item.Embedding, sourceErr = encoder.Encode(ctx, loaded.Frames[0])
					if sourceErr == nil {
						var preview bytes.Buffer
						sourceErr = jpeg.Encode(&preview, imaging.ScaleForExport(loaded.Frames[0], 160), &jpeg.Options{Quality: 80})
						item.Preview = preview.Bytes()
					}
				}
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if sourceErr == nil {
			after, statErr := os.Stat(path)
			if statErr != nil {
				sourceErr = statErr
			} else if !os.SameFile(before, after) || after.Size() != item.Size || after.ModTime().UnixNano() != item.ModifiedNS {
				sourceErr = fmt.Errorf("source changed during analysis")
			}
		}
		if sourceErr != nil {
			item.Error = sourceErr.Error()
			item.Embedding = nil
			item.Preview = nil
			event.Failed++
		} else {
			event.Successful++
			if reused {
				event.Reused++
			} else if err := cache.write(ctx, item); err != nil && event.CacheWarning == "" {
				event.CacheWarning = err.Error()
			}
		}
		items = append(items, item)
		if err := emit(event); err != nil {
			return err
		}
		readControls()
		if (requested || automatic && len(items)%30 == 0) && len(items) < len(req.Paths) && event.Successful > 0 {
			if err := publishMap(false); err != nil {
				return err
			}
			requested = false
		}
	}
	return publishMap(true)
}
