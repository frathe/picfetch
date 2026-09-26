package ui

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestLocationMapReleaseQualification(t *testing.T) {
	t.Run("repeated_occurrences", func(t *testing.T) {
		v := newTestViewer(t)
		base := uitest.TempGPSJPEGURI(t, "repeated.jpg", 24, 16, 52.52, 13.405)
		var reads atomic.Int32
		source := uitest.ReaderURI(base, func() (io.ReadCloser, error) {
			reads.Add(1)
			return os.Open(base.Path())
		})
		dropAndWait(t, v, source)
		v.state.SetMergeMode(true)
		dropAndWait(t, v, source)
		v.display.Settle()
		warmThumbs(t, v)
		reads.Store(0)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		points := v.locationMap.Points()
		if len(points) != 2 || v.locationMap.Counts().Located != 2 {
			t.Fatalf("repeated source lost a collection occurrence: %+v", points)
		}
		for i, point := range points {
			if point.Source.Identity != (fileidentity.Occurrence{Path: base.Path(), Ordinal: i}) {
				t.Fatalf("occurrence %d has identity %+v", i, point.Source.Identity)
			}
		}
		if reads.Load() != 1 || len(v.locationMap.RawFacts()) != 1 {
			t.Fatalf("repeated occurrences did not share one raw fact: reads=%d facts=%d", reads.Load(), len(v.locationMap.RawFacts()))
		}
		fynetest.Tap(locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2)))
		v.grid.Settle()
		if len(v.grid.ResultIndexes()) != 2 {
			t.Fatal("cluster collapsed repeated occurrences")
		}
		v.RemoveFile(v.grid.ResultIndexes()[0])
		v.grid.Settle()
		v.locationMap.Settle()
		if len(v.grid.ResultIndexes()) != 1 || len(v.locationMap.RawFacts()) != 1 {
			t.Fatal("removing one occurrence retired its surviving source fact")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.locationMap.Settle()
		if !v.locationMap.Visible() || len(v.locationMap.Points()) != 1 || reads.Load() != 1 {
			t.Fatalf("surviving occurrence lost map/reuse: reads=%d counts=%+v", reads.Load(), v.locationMap.Counts())
		}
		v.RemoveFile(0)
		v.grid.Settle()
		v.locationMap.Settle()
		if len(v.locationMap.RawFacts()) != 0 || len(v.locationMap.Points()) != 0 {
			t.Fatal("last occurrence retained its retired fact or point")
		}
		locationButton(t, v, lang.L("Back to Viewer"))
	})
	t.Run("metadata_limits_xmp_read_only", func(t *testing.T) {
		xmp := []byte("http://ns.adobe.com/xap/1.0/\x00" +
			`<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"><rdf:Description xmlns:exif="http://ns.adobe.com/exif/1.0/" exif:GPSLatitude="52,31.2N" exif:GPSLongitude="13,24.3E"/></rdf:RDF></x:xmpmeta>`)
		jpeg := uitest.EncodeJPEG(t, 24, 16, color.White)
		segment := binary.BigEndian.AppendUint16([]byte{0xff, 0xe1}, uint16(len(xmp)+2))
		xmpJPEG := append(append(append(bytes.Clone(jpeg[:2]), segment...), xmp...), jpeg[2:]...)
		for _, tc := range []struct {
			name    string
			data    []byte
			limited bool
		}{
			{"xmp-only.jpg", xmpJPEG, false},
			{"malformed-exif.png", uitest.PNGWithEXIF(t, []byte("invalid TIFF")), false},
			{"limited.jpg", uitest.GPSJPEG(t, 24, 16, 52.52, 13.405), true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				v := newTestViewer(t)
				base := storage.NewFileURI(uitest.WriteTempFile(t, tc.name, tc.data))
				if err := os.Chmod(base.Path(), 0o444); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(base.Path(), 0o600) })
				source := uitest.ReaderURI(base, func() (io.ReadCloser, error) { return os.Open(base.Path()) })
				dropAndWait(t, v, source)
				v.display.Settle()
				if tc.limited {
					limit := imaging.MaxEncodedBytes()
					t.Cleanup(func() { imaging.SetMaxEncodedBytes(limit) })
					imaging.SetMaxEncodedBytes(int64(len(tc.data) - 1))
				}
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				counts := v.locationMap.Counts()
				if !counts.Complete || counts.Total != 1 || counts.Located != 0 || len(v.locationMap.Points()) != 0 {
					t.Fatalf("unsupported metadata or limited source produced a location: %+v", counts)
				}
				if tc.limited {
					if counts.Failed != 1 || counts.Unlocated != 0 || len(v.locationMap.RawFacts()) != 0 {
						t.Fatalf("size-limit failure was cached as absent GPS: %+v", counts)
					}
					v.LeaveLocationMap()
					imaging.SetMaxEncodedBytes(0)
					locationMenu(t, v).Action()
					v.locationMap.Settle()
					if v.locationMap.Counts().Located != 1 || v.locationMap.Counts().Failed != 0 {
						t.Fatal("restoring read limit did not recover the uncached GPS source")
					}
				} else if counts.Unlocated != 1 || counts.Failed != 0 {
					t.Fatalf("unsupported metadata became an operational failure: %+v", counts)
				}
				locationButton(t, v, lang.L("Back to Viewer"))
				if after, err := os.ReadFile(base.Path()); err != nil || !bytes.Equal(after, tc.data) {
					t.Fatal("metadata browsing changed read-only source bytes")
				}
			})
		}
	})
	t.Run("unchanged_entry_and_return", func(t *testing.T) {
		for _, route := range []string{"entry", "image", "cluster"} {
			t.Run(route, func(t *testing.T) {
				v := newTestViewer(t)
				base := uitest.TempGPSJPEGURI(t, "reused.jpg", 24, 16, 52.52, 13.405)
				var reads atomic.Int32
				source := uitest.ReaderURI(base, func() (io.ReadCloser, error) {
					reads.Add(1)
					return os.Open(base.Path())
				})
				sources := []fyne.URI{source}
				if route == "cluster" {
					sources = append(sources, uitest.TempGPSJPEGURI(t, "neighbor.jpg", 24, 16, 52.52, 13.405))
				}
				dropAndWait(t, v, sources...)
				v.display.Settle()
				warmThumbs(t, v)
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				for _, changed := range []bool{false, true} {
					switch route {
					case "entry":
						v.LeaveLocationMap()
					case "image":
						fynetest.Tap(locationPhoto(t, v, source.Name()))
						waitUntilLoaded(t, v)
						v.display.Settle()
					case "cluster":
						fynetest.Tap(locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2)))
						v.grid.Settle()
					}
					before := reads.Load()
					if changed {
						if err := os.WriteFile(base.Path(), uitest.GPSJPEG(t, 60, 40, 51.507, -.128), 0o600); err != nil {
							t.Fatal(err)
						}
					}
					if route == "entry" {
						locationMenu(t, v).Action()
					} else {
						v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
					}
					v.grid.Settle()
					v.locationMap.Settle()
					if !v.locationMap.Visible() || !v.locationMap.Counts().Complete {
						t.Fatal("entry/return did not settle on the map")
					}
					if !changed && reads.Load() != before {
						t.Fatalf("unchanged return reread the source: %d -> %d", before, reads.Load())
					}
					if changed && reads.Load() == before {
						t.Fatal("changed source reused obsolete metadata")
					}
					wantLatitude := 52.52
					if changed {
						wantLatitude = 51.507
					}
					fact, ok := v.locationMap.RawFacts()[source.String()]
					if !ok || !fact.Metadata.HasGPS || math.Abs(fact.Metadata.Latitude-wantLatitude) > .000001 {
						t.Fatalf("entry/return has wrong source fact: %+v", fact)
					}
					found := false
					for _, point := range v.locationMap.Points() {
						if point.Source.URI.String() == source.String() {
							found = math.Abs(point.Metadata.Latitude-wantLatitude) < .000001
						}
					}
					if !found {
						t.Fatal("current raw location did not reach map points")
					}
				}
			})
		}
	})
	t.Run("rebuilt_thumbnail_and_camera", func(t *testing.T) {
		v := newTestViewer(t)
		v.win.Resize(fyne.NewSize(1000, 700))
		source := uitest.TempGPSJPEGURI(t, "changed.jpg", 24, 16, 52.52, 13.405)
		anchor := uitest.TempGPSJPEGURI(t, "anchor.jpg", 24, 16, 40.7, -74)
		dropAndWait(t, v, source, anchor)
		warmThumbs(t, v)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(35, 20)})
		v.locationMap.Settle()
		position := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, anchor.Name()))
		oldVersion := v.locationMap.RawFacts()[source.String()].Version
		updated := uitest.TempGPSJPEGURI(t, "replacement.jpg", 60, 40, 51.507, -.128)
		pixels, err := imaging.LoadImage(updated, imaging.DefaultImgCacheBytes)
		if err != nil {
			t.Fatal(err)
		}
		result, err := imaging.ExportContext(context.Background(), source, pixels.Frames[0], updated, imaging.ExportOptions{})
		if err != nil || !result.Committed {
			t.Fatalf("replacement export did not commit: %v", err)
		}
		completed := false
		v.afterFileWrite(result, false, false, func() { completed = true })
		drainFileWork(t, v)
		v.grid.Settle()
		v.locationMap.Settle()
		if !completed || !v.locationMap.Visible() {
			t.Fatal("committed replacement did not finish with the map visible")
		}
		if got := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, anchor.Name())); got != position {
			t.Fatalf("source rebuild moved manual camera: %v -> %v", position, got)
		}
		fact := v.locationMap.RawFacts()[source.String()]
		if fact.Version == oldVersion || math.Abs(fact.Metadata.Latitude-51.507) > .000001 {
			t.Fatalf("replacement retained old metadata/version: %+v", fact)
		}
		found := false
		explorerWalk(locationPhoto(t, v, source.Name()), func(object fyne.CanvasObject) {
			if img, ok := object.(*canvas.Image); ok && img.Image != nil {
				found = true
				if got := img.Image.Bounds().Size(); got.X != 60 || got.Y != 40 {
					t.Errorf("rebuilt card retained old preview: %v", got)
				}
			}
		})
		if !found {
			t.Fatal("rebuilt card did not mount the current preview")
		}
		cached, ok := v.grid.CachedThumb(source)
		if !ok || cached.Bounds().Dx() != 60 {
			t.Fatal("map rebuild did not refresh shared thumbnail cache")
		}
	})
	t.Run("competing_writes_and_replacement", func(t *testing.T) {
		for _, retainOld := range []bool{false, true} {
			for _, closeMap := range []bool{false, true} {
				t.Run(fmt.Sprintf("retained=%t/closed=%t", retainOld, closeMap), func(t *testing.T) {
					v := newTestViewer(t)
					old := uitest.TempGPSJPEGURI(t, "old.jpg", 24, 16, 52.52, 13.405)
					current := uitest.TempGPSJPEGURI(t, "current.jpg", 24, 16, 40.7, -74)
					dropAndWait(t, v, old)
					locationMenu(t, v).Action()
					v.locationMap.Settle()
					first, err := imaging.StripJPEGMetadataContext(context.Background(), old)
					if err != nil || !first.Committed {
						t.Fatalf("first write did not commit: %v", err)
					}
					retiredQueue := &uitest.UIQueue{}
					v.fileWork.ui = retiredQueue
					var firstDone, secondDone atomic.Int32
					v.afterFileWrite(first, false, false, func() { firstDone.Add(1) })
					v.fileWork.workers.Wait() // The old snapshot is resolved; its UI delivery remains held.
					if firstDone.Load() != 0 {
						t.Fatal("first reconciliation completed before its UI delivery")
					}
					v.fileWork.ui = &uitest.UIQueue{}
					// Cleanup must also deliver the held callback if a later assertion fails.
					t.Cleanup(func() { retiredQueue.Drain(); drainFileWork(t, v) })
					sources := []fyne.URI{current}
					if retainOld {
						sources = append(sources, old)
					}
					dropAndWait(t, v, sources...)
					locationMenu(t, v).Action()
					v.locationMap.Settle()
					updated := uitest.TempGPSJPEGURI(t, "latest.jpg", 60, 40, 51.507, -.128)
					pixels, err := imaging.LoadImage(updated, imaging.DefaultImgCacheBytes)
					if err != nil {
						t.Fatal(err)
					}
					second, err := imaging.ExportContext(context.Background(), current, pixels.Frames[0], updated, imaging.ExportOptions{})
					if err != nil || !second.Committed {
						t.Fatalf("second write did not commit: %v", err)
					}
					v.afterFileWrite(second, false, false, func() { secondDone.Add(1) })
					drainFileWork(t, v)
					v.grid.Settle()
					v.locationMap.Settle()
					if secondDone.Load() != 1 || firstDone.Load() != 0 {
						t.Fatal("fixture did not complete newer reconciliation before the older delivery")
					}
					if closeMap {
						v.LeaveLocationMap()
					}
					retiredQueue.Drain()
					drainFileWork(t, v)
					v.grid.Settle()
					v.locationMap.Settle()
					if firstDone.Load() != 1 || secondDone.Load() != 1 || v.FileCount() != len(sources) {
						t.Fatal("late reconciliation lost completion or changed collection membership")
					}
					facts := v.locationMap.RawFacts()
					if fact, ok := facts[current.String()]; !ok || !fact.Metadata.HasGPS || math.Abs(fact.Metadata.Latitude-51.507) > .000001 {
						t.Fatalf("older delivery replaced the newer source fact: %+v", facts)
					}
					if closeMap {
						if v.locationMap.Active() || v.locationMap.Visible() {
							t.Fatal("late committed write reopened the closed map")
						}
					} else if counts := v.locationMap.Counts(); !counts.Complete || counts.Located != 1 || counts.Total != len(sources) {
						t.Fatalf("late write corrupted current map outcomes: %+v", counts)
					}
					if !retainOld {
						if _, ok := facts[old.String()]; ok {
							t.Fatal("retired source fact reappeared in the replacement collection")
						}
					} else if fact, ok := facts[old.String()]; ok && fact.Metadata.HasGPS {
						t.Fatal("retained source recovered stripped GPS from the earlier collection")
					}
				})
			}
		}
	})
	t.Run("mutation_to_empty_read_only", func(t *testing.T) {
		for _, route := range []string{"map", "image", "cluster"} {
			t.Run(route, func(t *testing.T) {
				v := newTestViewer(t)
				source := uitest.TempGPSJPEGURI(t, "last.jpg", 24, 16, 52.52, 13.405)
				dropAndWait(t, v, source)
				if route == "cluster" {
					v.state.SetMergeMode(true)
					dropAndWait(t, v, source)
				}
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				if route == "image" {
					fynetest.Tap(locationPhoto(t, v, source.Name()))
					waitUntilLoaded(t, v)
				} else if route == "cluster" {
					fynetest.Tap(locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2)))
					v.grid.Settle()
				}
				result, err := imaging.StripJPEGMetadataContext(context.Background(), source)
				if err != nil || !result.Committed {
					t.Fatalf("metadata removal did not commit: %v", err)
				}
				stripped, err := os.ReadFile(source.Path())
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(source.Path(), 0o444); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(source.Path(), 0o600) })
				var completed atomic.Bool
				v.afterFileWrite(result, false, false, func() { completed.Store(true) })
				drainFileWork(t, v)
				v.grid.Settle()
				v.locationMap.Settle()
				if route == "cluster" {
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
					v.locationMap.Settle()
				}
				counts := v.locationMap.Counts()
				if !completed.Load() || !counts.Complete || counts.Unlocated != v.FileCount() || counts.Located != 0 || len(v.locationMap.Points()) != 0 || !v.locationMap.Visible() {
					t.Fatalf("last GPS removal did not reach the empty map: %+v", counts)
				}
				assertLocationStatus(t, v, fmt.Sprintf(lang.L("No usable image locations. %d without GPS, %d unreadable, %d conflicts"), v.FileCount(), 0, 0))
				indices := []int{0}
				if route == "cluster" {
					indices = append(indices, 1)
				}
				v.RemoveFiles(indices)
				v.grid.Settle()
				v.locationMap.Settle()
				counts = v.locationMap.Counts()
				if !counts.Complete || counts.Total != 0 || v.FileCount() != 0 || len(v.locationMap.RawFacts()) != 0 || len(v.locationMap.Points()) != 0 {
					t.Fatalf("final removal retained retired map state: %+v", counts)
				}
				assertLocationStatus(t, v, lang.L("Open images to browse their locations."))
				fynetest.Tap(locationButton(t, v, lang.L("Back to Viewer")))
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				assertLocationStatus(t, v, lang.L("Open images to browse their locations."))
				if after, err := os.ReadFile(source.Path()); err != nil || !bytes.Equal(stripped, after) {
					t.Fatal("read-only browsing or collection removal modified the remaining file")
				}
			})
		}
	})
}

func assertLocationStatus(t *testing.T, v *viewer, want string) {
	t.Helper()
	if !v.locationMap.Visible() {
		t.Fatal("location status belongs to a hidden map")
	}
	found := false
	explorerWalk(v.win.Content(), func(object fyne.CanvasObject) {
		if label, ok := object.(*widget.Label); ok && label.Visible() && label.Text == want {
			found = true
		}
	})
	if !found {
		t.Fatalf("location status %q is not mounted", want)
	}
}
