package similarity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image/jpeg"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
)

func analyzeLocal(ctx context.Context, req request, controls <-chan Control, emit func(Event) error) error {
	start := time.Now()
	if err := ctx.Err(); err != nil {
		return err
	}
	offline := EnforcesNetworkIsolation()
	if offline {
		if err := VerifyOffline(ctx); err != nil {
			return err
		}
	}
	if err := VerifyAssets(ctx, req.Assets); err != nil {
		return err
	}
	tagger, err := NewTagger()
	if err != nil {
		return err
	}
	_ = RegisterLocalFiles()
	var encoder *Encoder
	defer func() {
		if encoder != nil {
			encoder.Close()
		}
	}()
	event := Event{Total: len(req.Paths), OfflineVerified: offline, Stage: "encoding"}
	event.Measurements.SetupSeconds = time.Since(start).Seconds()
	send := func(snapshot Event) error {
		snapshot.Measurements.ElapsedSeconds = time.Since(start).Seconds()
		return emit(snapshot)
	}
	cacheStart := time.Now()
	cache, cacheErr := openAnalysisCache(ctx, req.FavoritesDir)
	event.Measurements.CacheSeconds += time.Since(cacheStart).Seconds()
	defer cache.close()
	if cacheErr != nil {
		event.CacheWarning = cacheErr.Error()
	}
	items := make([]Item, 0, len(req.Paths))
	type representedSource struct {
		info os.FileInfo
		item Item
	}
	represented := make(map[string]representedSource)
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
		return publishAnalysisMap(ctx, &event, items, complete, send)
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
		reused, backfilled := false, false
		if sourceErr == nil {
			item.Size, item.ModifiedNS = before.Size(), before.ModTime().UnixNano()
			cacheStart := time.Now()
			previous, ok := represented[path]
			if ok && os.SameFile(previous.info, before) && previous.item.Size == item.Size && previous.item.ModifiedNS == item.ModifiedNS {
				item, reused = previous.item, true
			} else if cached, ok := cache.read(item); ok {
				item, reused = cached, true
			}
			event.Measurements.CacheSeconds += time.Since(cacheStart).Seconds()
		}
		if sourceErr == nil && reused && (item.Facts.Version != FactsVersion || item.Facts.Width <= 0 || item.Facts.Height <= 0) {
			factsStart := time.Now()
			data, bounds, readErr := imaging.ReadAndProbe(ctx, storage.NewFileURI(path))
			sourceErr = readErr
			if sourceErr == nil && fmt.Sprintf("%x", sha256.Sum256(data)) != item.SHA256 {
				sourceErr = fmt.Errorf("source changed since cached representation")
			}
			if sourceErr == nil {
				item.Facts = imageFacts(path, data, bounds)
				backfilled = true
			}
			event.Measurements.DecodeSeconds += time.Since(factsStart).Seconds()
		}
		if sourceErr == nil && !reused {
			if encoder == nil {
				var err error
				modelStart := time.Now()
				encoder, err = NewEncoder(req.Assets, "cpu")
				event.Measurements.ModelSeconds += time.Since(modelStart).Seconds()
				if err != nil {
					return err
				}
			}
			decodeStart := time.Now()
			data, bounds, readErr := imaging.ReadAndProbe(ctx, storage.NewFileURI(path))
			sourceErr = readErr
			if sourceErr == nil {
				item.Facts = imageFacts(path, data, bounds)
				item.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
				loaded, decodeErr := imaging.DecodeLoaded(ctx, data, 1)
				event.Measurements.DecodeSeconds += time.Since(decodeStart).Seconds()
				sourceErr = decodeErr
				if sourceErr == nil {
					encodeStart := time.Now()
					event.Measurements.InferenceAttempts++
					item.Embedding, sourceErr = encoder.Encode(ctx, loaded.Frames[0])
					event.Measurements.EncodeSeconds += time.Since(encodeStart).Seconds()
					if sourceErr == nil {
						previewStart := time.Now()
						var preview bytes.Buffer
						sourceErr = jpeg.Encode(&preview, imaging.ScaleForExport(loaded.Frames[0], 160), &jpeg.Options{Quality: 80})
						item.Preview = preview.Bytes()
						event.Measurements.PreviewSeconds += time.Since(previewStart).Seconds()
					}
				}
			} else {
				event.Measurements.DecodeSeconds += time.Since(decodeStart).Seconds()
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
			tagStart := time.Now()
			item.Tags = tagger.Tags(item.Embedding)
			event.Measurements.TagSeconds += time.Since(tagStart).Seconds()
			event.Successful++
			if reused {
				event.Reused++
			}
			represented[path] = representedSource{info: before, item: item}
			if !reused || backfilled {
				cacheStart := time.Now()
				err := cache.write(ctx, item)
				event.Measurements.CacheSeconds += time.Since(cacheStart).Seconds()
				if err != nil && event.CacheWarning == "" {
					event.CacheWarning = err.Error()
				}
			}
		}
		items = append(items, item)
		if err := send(event); err != nil {
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

func publishAnalysisMap(ctx context.Context, event *Event, items []Item, complete bool, send func(Event) error) error {
	event.Stage = "layout"
	if err := send(*event); err != nil {
		return err
	}
	var err error
	stages := map[string]float64{}
	groupStart := time.Now()
	event.Merges, err = Group(ctx, items, stages)
	event.Measurements.GroupingSeconds += time.Since(groupStart).Seconds()
	event.Measurements.ReductionSeconds += stages["reduction_seconds"]
	event.Measurements.HDBSCANSeconds += stages["hdbscan_seconds"]
	event.Measurements.ProjectionSeconds += stages["projection_seconds"]
	event.Measurements.HierarchySeconds += stages["hierarchy_seconds"]
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	event.Items = append([]Item(nil), items...)
	for i := range event.Items {
		// Inference vectors remain in the worker/cache. The viewer needs only
		// assignments, the compact hierarchy, labels and sampled previews.
		event.Items[i].Embedding = nil
	}
	event.Complete = complete
	event.Stage = "encoding"
	if complete {
		event.Stage = "complete"
	}
	event.Measurements.Publications++
	err = send(*event)
	event.Items = nil
	event.Merges = nil
	return err
}
