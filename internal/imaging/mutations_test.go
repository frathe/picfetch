package imaging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/synctest"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/uitest"
)

type heldMutationPixels struct {
	image.Image
	once             sync.Once
	entered, release chan struct{}
}

func TestFileMutationCancellationLeavesOriginalBytes(t *testing.T) {
	for _, when := range []string{"before admission", "during encoding", "before rename"} {
		t.Run(when, func(t *testing.T) {
			source := uitest.TempJPEGURI(t, "source.jpg", 40, 20, color.White)
			before, err := os.ReadFile(source.Path())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var result WriteResult
			switch when {
			case "before admission":
				cancel()
				result, err = SaveRotatedContext(ctx, source, image.NewRGBA(image.Rect(0, 0, 20, 40)))
			case "during encoding":
				pixels := &heldMutationPixels{Image: image.NewRGBA(image.Rect(0, 0, 20, 40)), entered: make(chan struct{}), release: make(chan struct{})}
				done := make(chan struct{})
				go func() { defer close(done); result, err = SaveRotatedContext(ctx, source, pixels) }()
				<-pixels.entered
				cancel()
				close(pixels.release)
				<-done
			case "before rename":
				err = writeFileContext(ctx, source.Path(), 0o640, func(w io.Writer) error {
					if _, err := w.Write([]byte("finished encoding")); err != nil {
						return err
					}
					cancel()
					return nil
				})
			}
			if !errors.Is(err, context.Canceled) || result.Committed {
				t.Errorf("cancelled result = %+v, %v", result, err)
			}
			after, err := os.ReadFile(source.Path())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Error("cancelled write changed original bytes")
			}
			temps, err := filepath.Glob(filepath.Join(filepath.Dir(source.Path()), ".picfetch-save-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(temps) != 0 {
				t.Errorf("cancelled write left temporary files: %v", temps)
			}
		})
	}
}

func TestFileMutationCancelledWaiterExitsBeforeActiveEncoder(t *testing.T) {
	source := uitest.TempGPSJPEGURI(t, "source.jpg", 40, 20, 48.858222, 2.2945)
	synctest.Test(t, func(t *testing.T) {
		pixels := &heldMutationPixels{Image: image.NewRGBA(image.Rect(0, 0, 20, 40)), entered: make(chan struct{}), release: make(chan struct{})}
		firstDone := make(chan error, 1)
		go func() { firstDone <- SaveRotated(source, pixels) }()
		<-pixels.entered
		ctx, cancel := context.WithCancel(context.Background())
		secondDone := make(chan error, 1)
		go func() {
			result, err := StripJPEGMetadataContext(ctx, source)
			if result.Committed {
				t.Error("waiting strip committed")
			}
			secondDone <- err
		}()
		synctest.Wait()
		cancel()
		synctest.Wait()
		var finished bool
		select {
		case err := <-secondDone:
			finished = true
			if !errors.Is(err, context.Canceled) {
				t.Errorf("cancelled waiter = %v", err)
			}
		default:
			t.Error("cancelled waiter still waited for the active encoder")
		}
		close(pixels.release)
		if err := <-firstDone; err != nil {
			t.Fatal(err)
		}
		if !finished {
			<-secondDone
		}
		result, err := StripJPEGMetadataContext(context.Background(), source)
		if err != nil || !result.Committed {
			t.Errorf("retry after cancellation = %+v, %v", result, err)
		}
	})
}

func TestFileMutationResultsDistinguishCommitFromNoopAndFailure(t *testing.T) {
	source := uitest.TempJPEGURI(t, "source.jpg", 40, 20, color.White)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result, err := SaveRotatedContext(ctx, source, image.NewRGBA(image.Rect(0, 0, 20, 40)))
	if err != nil || !result.Committed {
		t.Fatalf("save = %+v, %v", result, err)
	}
	cancel()
	if !result.Committed {
		t.Error("cancelling after replacement concealed the accomplished commit")
	}
	resolved, err := filepath.EvalSymlinks(source.Path())
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != resolved {
		t.Errorf("committed target = %q, want %q", result.Path, resolved)
	}
	result, err = StripJPEGMetadataContext(context.Background(), source)
	if err != nil || result.Committed {
		t.Errorf("metadata-free no-op = %+v, %v", result, err)
	}
	missing := storage.NewFileURI(filepath.Join(t.TempDir(), "missing.jpg"))
	result, err = SaveRotatedContext(context.Background(), missing, image.NewRGBA(image.Rect(0, 0, 20, 40)))
	if err == nil || result.Committed {
		t.Errorf("missing source = %+v, %v", result, err)
	}
}

func TestExportPreservesSymlinkParentsAndRejectsSymlinkLeaves(t *testing.T) {
	actual := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(actual, alias); err != nil {
		t.Fatal(err)
	}
	pixels := image.NewRGBA(image.Rect(0, 0, 13, 17))
	dest := storage.NewFileURI(filepath.Join(alias, "confirmed.extension"))
	result, err := ExportContext(context.Background(), dest, pixels, nil, ExportOptions{FallbackExt: ".png"})
	if err != nil || !result.Committed {
		t.Fatalf("export = %+v, %v", result, err)
	}
	if _, err := os.Stat(dest.Path()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest.Path() + ".png"); !os.IsNotExist(err) {
		t.Errorf("export appended an unconfirmed extension: %v", err)
	}
	for _, exists := range []bool{false, true} {
		t.Run(fmt.Sprintf("target-exists=%v", exists), func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "unrelated.txt")
			original := []byte("preserve the unrelated target")
			if exists {
				if err := os.WriteFile(target, original, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			link := filepath.Join(alias, fmt.Sprintf("leaf-%v.png", exists))
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			result, err := ExportContext(context.Background(), storage.NewFileURI(link), pixels, nil, ExportOptions{})
			if err == nil || result.Committed {
				t.Errorf("symlink leaf export = %+v, %v", result, err)
			}
			if got, err := os.Readlink(link); err != nil || got != target {
				t.Errorf("failed export changed the link: %q, %v", got, err)
			}
			data, err := os.ReadFile(target)
			if exists {
				if err != nil || !bytes.Equal(data, original) {
					t.Errorf("export changed the unrelated target: %v", err)
				}
			} else if !os.IsNotExist(err) {
				t.Errorf("export created the dangling target: %v", err)
			}
		})
	}
	t.Run("leaf introduced during encoding", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "unrelated.txt")
		original := []byte("preserve the unrelated target")
		if err := os.WriteFile(target, original, 0o600); err != nil {
			t.Fatal(err)
		}
		leaf := filepath.Join(alias, "late.jpg")
		pixels := &heldMutationPixels{Image: image.NewRGBA(image.Rect(0, 0, 13, 17)), entered: make(chan struct{}), release: make(chan struct{})}
		var result WriteResult
		var err error
		done := make(chan struct{})
		unpark := sync.OnceFunc(func() { close(pixels.release) })
		t.Cleanup(func() { unpark(); <-done })
		go func() {
			defer close(done)
			result, err = ExportContext(context.Background(), storage.NewFileURI(leaf), pixels, nil, ExportOptions{})
		}()
		<-pixels.entered
		if err := os.Symlink(target, leaf); err != nil {
			t.Fatal(err)
		}
		unpark()
		<-done
		if err != nil || !result.Committed {
			t.Fatalf("export = %+v, %v", result, err)
		}
		if data, err := os.ReadFile(target); err != nil || !bytes.Equal(data, original) {
			t.Errorf("export followed the late symlink: %v", err)
		}
		if info, err := os.Lstat(leaf); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("export did not replace the late symlink: %v", err)
		}
	})
}

func TestFileTransactionClaimsAreReleasedAfterSuccessFailureAndCancellation(t *testing.T) {
	var transactions pathTransactions
	path := filepath.Join(t.TempDir(), "source")
	release, err := transactions.acquire(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for range 100 {
		if unlock, err := transactions.acquire(ctx, path); !errors.Is(err, context.Canceled) || unlock != nil {
			t.Fatalf("cancelled claim = %v", err)
		}
	}
	transactions.mu.Lock()
	remaining := len(transactions.paths)
	transactions.mu.Unlock()
	if remaining != 1 {
		t.Errorf("held paths = %d, want only the active owner", remaining)
	}
	release()
	for _, fail := range []bool{false, true} {
		for range 100 {
			_, _ = transactions.write(context.Background(), path, true, func(_ string) (bool, error) {
				if fail {
					return false, errors.New("write failed")
				}
				return true, nil
			})
		}
	}
	if got := len(transactions.paths); got != 0 {
		t.Errorf("finished transaction paths retained = %d", got)
	}
}

func TestFileTransactionAliasAdmissionUsesLiveEntries(t *testing.T) {
	for _, scenario := range []string{"missing case alias", "replaced case alias", "distinct files", "distinct case files"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			first, second := filepath.Join(dir, "Photo.jpg"), filepath.Join(dir, "photo.jpg")
			shared := scenario == "missing case alias" || scenario == "replaced case alias"
			if scenario != "missing case alias" {
				if err := os.WriteFile(first, []byte("first"), 0o600); err != nil {
					t.Fatal(err)
				}
				if scenario == "distinct files" {
					second = filepath.Join(dir, "other.jpg")
				}
				if err := os.WriteFile(second, []byte("second"), 0o600); err != nil {
					t.Fatal(err)
				}
				a, aErr := os.Stat(first)
				b, bErr := os.Stat(second)
				if aErr != nil || bErr != nil {
					t.Fatalf("stat fixtures: %v, %v", aErr, bErr)
				}
				if os.SameFile(a, b) != shared {
					t.Skip("requires the other filesystem case behavior")
				}
			}
			synctest.Test(t, func(t *testing.T) {
				var transactions pathTransactions
				release, err := transactions.acquire(context.Background(), first)
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "replaced case alias" {
					if err := writeFileContext(context.Background(), first, 0o600, func(w io.Writer) error {
						_, err := io.WriteString(w, "new inode")
						return err
					}); err != nil {
						release()
						t.Fatal(err)
					}
				}
				admitted := make(chan struct{}, 1)
				done := make(chan error, 1)
				go func() {
					unlock, err := transactions.acquire(context.Background(), second)
					if err == nil {
						admitted <- struct{}{}
						unlock()
					}
					done <- err
				}()
				synctest.Wait()
				if early := len(admitted) != 0; early == shared {
					t.Errorf("second path admitted early=%v, shared=%v", early, shared)
				}
				release()
				if err := <-done; err != nil {
					t.Fatal(err)
				}
				if len(transactions.paths) != 0 {
					t.Fatal("completed alias claims were retained")
				}
			})
		})
	}
}

func (p *heldMutationPixels) At(x, y int) color.Color {
	p.once.Do(func() { close(p.entered); <-p.release })
	return p.Image.At(x, y)
}

func TestFileMutationsSerializeWholeTransactionsAcrossAliases(t *testing.T) {
	for _, alias := range []string{"direct", "symlink", "case"} {
		for _, next := range []string{"strip", "save", "export"} {
			t.Run(fmt.Sprintf("alias=%s/next=%s", alias, next), func(t *testing.T) {
				source := uitest.TempGPSJPEGURI(t, "source.jpg", 40, 20, 48.858222, 2.2945)
				destination := source
				if alias == "case" {
					destination = storage.NewFileURI(filepath.Join(filepath.Dir(source.Path()), "SOURCE.JPG"))
					first, firstErr := os.Stat(source.Path())
					second, secondErr := os.Stat(destination.Path())
					if firstErr != nil || secondErr != nil || !os.SameFile(first, second) {
						t.Skip("requires a case-insensitive filesystem")
					}
				}
				if alias == "symlink" {
					link := filepath.Join(t.TempDir(), "alias.jpg")
					target := source.Path()
					if next == "export" {
						target = filepath.Dir(target)
					}
					if err := os.Symlink(target, link); err != nil {
						t.Fatal(err)
					}
					if next == "export" {
						link = filepath.Join(link, filepath.Base(source.Path()))
					}
					destination = storage.NewFileURI(link)
				}
				synctest.Test(t, func(t *testing.T) {
					pixels := &heldMutationPixels{Image: image.NewRGBA(image.Rect(0, 0, 20, 40)), entered: make(chan struct{}), release: make(chan struct{})}
					firstDone := make(chan error, 1)
					go func() { firstDone <- SaveRotated(source, pixels) }()
					<-pixels.entered
					secondDone := make(chan error, 1)
					go func() {
						switch next {
						case "strip":
							secondDone <- StripJPEGMetadata(destination)
						case "save":
							secondDone <- SaveRotated(destination, image.NewRGBA(image.Rect(0, 0, 13, 17)))
						case "export":
							secondDone <- Export(destination, image.NewRGBA(image.Rect(0, 0, 13, 17)), source, ExportOptions{OmitMetadata: true})
						}
					}()
					synctest.Wait()
					var early bool
					select {
					case err := <-secondDone:
						early = true
						t.Errorf("second %s transaction completed while the first still encoded: %v", next, err)
					default:
					}
					close(pixels.release)
					if err := <-firstDone; err != nil {
						t.Fatal(err)
					}
					if !early {
						if err := <-secondDone; err != nil {
							t.Fatal(err)
						}
					}
					data, err := os.ReadFile(source.Path())
					if err != nil {
						t.Fatal(err)
					}
					cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
					if err != nil {
						t.Fatal(err)
					}
					want := image.Pt(13, 17)
					if next == "strip" {
						want = image.Pt(20, 40)
					}
					if image.Pt(cfg.Width, cfg.Height) != want {
						t.Errorf("final source dimensions = %dx%d, want %v; later transaction used old bytes or missed the target", cfg.Width, cfg.Height, want)
					}
					if next != "save" && !ReadMetadata(data).Empty() {
						t.Error("later metadata removal was overwritten by the earlier save")
					}
					if alias == "symlink" {
						link := destination.Path()
						if next == "export" {
							link = filepath.Dir(link)
						}
						info, err := os.Lstat(link)
						if err != nil {
							t.Fatal(err)
						}
						if info.Mode()&os.ModeSymlink == 0 {
							t.Error("writing through the confirmed alias replaced the link")
						}
					}
				})
			})
		}
	}
}

func TestFileMutationsAllowIndependentDestinations(t *testing.T) {
	source := uitest.TempJPEGURI(t, "first.jpg", 40, 20, color.White)
	other := uitest.TempJPEGURI(t, "second.jpg", 40, 20, color.Black)
	synctest.Test(t, func(t *testing.T) {
		pixels := &heldMutationPixels{Image: image.NewRGBA(image.Rect(0, 0, 20, 40)), entered: make(chan struct{}), release: make(chan struct{})}
		firstDone := make(chan error, 1)
		go func() { firstDone <- SaveRotated(source, pixels) }()
		<-pixels.entered
		secondDone := make(chan error, 1)
		go func() { secondDone <- SaveRotated(other, image.NewRGBA(image.Rect(0, 0, 13, 17))) }()
		synctest.Wait()
		var ready bool
		select {
		case err := <-secondDone:
			ready = true
			if err != nil {
				t.Error(err)
			}
		default:
			t.Error("an independent destination waited for the held first file")
		}
		close(pixels.release)
		if err := <-firstDone; err != nil {
			t.Fatal(err)
		}
		if !ready {
			if err := <-secondDone; err != nil {
				t.Fatal(err)
			}
		}
	})
}
