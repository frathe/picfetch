package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/frathe/picfetch/internal/similarity"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/filescan"
	"github.com/frathe/picfetch/internal/imaging"
)

type configuration struct{ Assets, Library, Out, Provider string }

type item = similarity.Item

type evaluation struct {
	OfflineVerified bool
	Config          configuration
	ModelRevision   string
	Available       int
	Items           []item
	ElapsedSeconds  float64
	FirstMapSeconds float64
	PeakRSSBytes    int64
	MemoryScope     string
	SetupSeconds    float64
	DecodeSeconds   float64
	EncodeSeconds   float64
	InitialStages   map[string]float64
	FinalStages     map[string]float64
	ManifestSHA256  string
}

func evaluate(ctx context.Context, config configuration, output io.Writer) error {
	start := time.Now()
	if err := similarity.VerifyOffline(ctx); err != nil {
		return err
	}
	info, err := os.Stat(config.Library)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("library must be a directory")
	}
	scanError := similarity.RegisterLocalFiles()
	files, truncated := filescan.Images(ctx, []fyne.URI{storage.NewFileURI(config.Library)}, filescan.DefaultMax, nil)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := scanError(); err != nil {
		return fmt.Errorf("incomplete input scan: %w", err)
	}
	if truncated {
		return fmt.Errorf("input scan truncated; refusing an incomplete manifest")
	}
	if len(files) == 0 {
		return fmt.Errorf("library contains no supported images")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].String() < files[j].String() })
	result := evaluation{OfflineVerified: true, Config: config, ModelRevision: similarity.ModelRevision, Available: len(files)}
	files = smokeSample(files)
	if err := os.MkdirAll(filepath.Dir(config.Out), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(config.Out, 0o700); err != nil {
		return fmt.Errorf("evidence directory must be new: %w", err)
	}
	if err := os.Mkdir(filepath.Join(config.Out, "images"), 0o700); err != nil {
		return err
	}
	e, err := similarity.NewEncoder(config.Assets, config.Provider)
	if err != nil {
		return err
	}
	defer e.Close()
	result.SetupSeconds = time.Since(start).Seconds()
	var initial []item
	successful := 0
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		entry := item{Path: file.Path()}
		before, err := os.Stat(file.Path())
		if err == nil {
			entry.Size, entry.ModifiedNS = before.Size(), before.ModTime().UnixNano()
		}
		if err == nil {
			decodeStart := time.Now()
			var data []byte
			data, _, err = imaging.ReadAndProbe(ctx, file)
			if err == nil {
				entry.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
				var loaded *imaging.LoadedImage
				loaded, err = imaging.DecodeLoaded(ctx, data, 1)
				result.DecodeSeconds += time.Since(decodeStart).Seconds()
				if err == nil {
					encodeStart := time.Now()
					entry.Embedding, err = e.Encode(ctx, loaded.Frames[0])
					result.EncodeSeconds += time.Since(encodeStart).Seconds()
					if err == nil {
						entry.Thumbnail = fmt.Sprintf("images/%04d.jpg", i)
						f, writeErr := os.OpenFile(filepath.Join(config.Out, entry.Thumbnail), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
						if writeErr != nil {
							return writeErr
						}
						writeErr = jpeg.Encode(f, imaging.ScaleForExport(loaded.Frames[0], 240), &jpeg.Options{Quality: 85})
						closeErr := f.Close()
						if writeErr != nil {
							return writeErr
						}
						if closeErr != nil {
							return closeErr
						}
					}
				}
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err == nil {
			after, statErr := os.Stat(file.Path())
			if statErr != nil {
				err = statErr
			} else if !os.SameFile(before, after) || after.Size() != entry.Size || after.ModTime().UnixNano() != entry.ModifiedNS {
				err = fmt.Errorf("source changed during analysis")
			}
		}
		if err != nil {
			entry.Error = err.Error()
			entry.Embedding = nil
			entry.Thumbnail = ""
		} else {
			successful++
		}
		result.Items = append(result.Items, entry)
		if _, err := fmt.Fprintf(output, "processed %d/%d; represented %d; failed %d\n", i+1, len(files), successful, i+1-successful); err != nil {
			return err
		}
		if initial == nil && len(result.Items) >= max(1, len(files)/2) && successful > 0 {
			initial = append([]item(nil), result.Items...)
			result.InitialStages = map[string]float64{}
			if err := similarity.Group(ctx, initial, result.InitialStages); err != nil {
				return err
			}
			result.FirstMapSeconds = time.Since(start).Seconds()
			if err := writeJSON(filepath.Join(config.Out, "initial.json"), struct{ Items []item }{initial}); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(output, "initial map ready at %.2fs (%d inputs)\n", result.FirstMapSeconds, len(initial)); err != nil {
				return err
			}
		}
	}
	result.FinalStages = map[string]float64{}
	if err := similarity.Group(ctx, result.Items, result.FinalStages); err != nil {
		var failures []error
		for _, entry := range result.Items {
			if entry.Error != "" {
				failures = append(failures, fmt.Errorf("%s: %s", filepath.Base(entry.Path), entry.Error))
			}
		}
		return errors.Join(append([]error{err}, failures...)...)
	}
	manifest := make([]item, len(result.Items))
	for i, entry := range result.Items {
		entry.Embedding = nil
		entry.Position = nil
		entry.Cohort = ""
		entry.Thumbnail = ""
		manifest[i] = entry
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	result.ManifestSHA256 = fmt.Sprintf("%x", sha256.Sum256(manifestData))
	if err := writeJSON(filepath.Join(config.Out, "manifest.json"), manifest); err != nil {
		return err
	}
	result.ElapsedSeconds = time.Since(start).Seconds()
	result.PeakRSSBytes, err = peakRSS()
	if err != nil {
		return err
	}
	result.MemoryScope = "OS maximum RSS of the analysis worker (Go, native runtime and threads); excludes launcher and any external CoreML services"
	if err := writeReview(result, initial); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeJSON(filepath.Join(config.Out, "result.json"), result)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
