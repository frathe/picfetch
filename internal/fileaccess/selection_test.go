package fileaccess

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

func TestSelectedInputCapturesBeforeReleaseAndPreservesOccurrences(t *testing.T) {
	selected := storage.NewFileURI("/selected/photo.jpg")
	released, captured := 0, 0
	input := NewSelection(selected, func(_ context.Context) (Record, error) {
		if released != 0 {
			t.Fatal("URL released before capture")
		}
		captured++
		return Record{URI: selected.String(), Bookmark: []byte("scope")}, nil
	}, func() { released++ })
	files := []fyne.URI{input, selected, input}
	got, err := CaptureSelected(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || string(Snapshot(got[0]).Bookmark) != "scope" || got[0] != got[2] || got[1] != selected {
		t.Fatalf("captured occurrences: %v", got)
	}
	ReleaseSelected(files)
	if captured != 1 || released != 1 {
		t.Fatalf("captured=%d released=%d", captured, released)
	}
}

func TestSelectedInputCloseWaitsForActiveNativeCapture(t *testing.T) {
	entered, finish := make(chan struct{}), make(chan struct{})
	var released atomic.Int32
	selected := storage.NewFileURI("/selected/photo.jpg")
	input := NewSelection(selected, func(_ context.Context) (Record, error) {
		close(entered)
		<-finish
		if released.Load() != 0 {
			return Record{}, errors.New("native URL retired during capture")
		}
		return Record{URI: selected.String(), Bookmark: []byte("scope")}, nil
	}, func() { released.Add(1) })
	done := make(chan error, 1)
	go func() { _, err := CaptureSelected(context.Background(), []fyne.URI{input}); done <- err }()
	<-entered
	ReleaseSelected([]fyne.URI{input, input})
	activeRelease := released.Load()
	close(finish)
	err := <-done
	if activeRelease != 0 || released.Load() != 1 || !errors.Is(err, errSelectionClosed) {
		t.Fatalf("active releases=%d final releases=%d result=%v", activeRelease, released.Load(), err)
	}
}

func TestSelectedInputFailureDiscardsCompleteBatch(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if cancelled {
			cancel()
		}
		released, calls := 0, 0
		var files []fyne.URI
		for range 3 {
			files = append(files, NewSelection(storage.NewFileURI("/photo.jpg"), func(_ context.Context) (Record, error) {
				calls++
				return Record{}, errors.New("capture failed")
			}, func() { released++ }))
		}
		got, err := CaptureSelected(ctx, files)
		cancel()
		releasedAtReturn := released
		ReleaseSelected(files)
		if err == nil || got != nil || releasedAtReturn != 3 || released != 3 || (cancelled && calls != 0) || (!cancelled && calls != 1) {
			t.Fatalf("cancelled=%v result=%v err=%v calls=%d release at return=%d final=%d", cancelled, got, err, calls, releasedAtReturn, released)
		}
	}
}
