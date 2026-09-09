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

func analyzeLocal(ctx context.Context, req request, emit func(Event) error) error {
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
	encoder, err := NewEncoder(req.Assets, "cpu")
	if err != nil {
		return err
	}
	defer encoder.Close()
	event := Event{Total: len(req.Paths), OfflineVerified: true, Stage: "encoding"}
	items := make([]Item, 0, len(req.Paths))
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
		if sourceErr == nil {
			item.Size, item.ModifiedNS = before.Size(), before.ModTime().UnixNano()
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
		}
		items = append(items, item)
		if err := emit(event); err != nil {
			return err
		}
	}
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
	event.Items = items
	event.Stage = "complete"
	event.Complete = true
	return emit(event)
}
