package fileaccess

import (
	"context"
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

func TestTransferRetainsOriginalScope(t *testing.T) {
	active, stopped := false, 0
	uri := &sourceURI{URI: storage.NewFileURI("/selected.png"), record: Record{URI: "file:///selected.png", Bookmark: []byte("persistent")},
		resolve: func(_ context.Context, _ Record) (resolution, func(), error) {
			active = true
			return resolution{path: "/resolved.png"}, func() { active = false; stopped++ }, nil
		}}
	sources := []fyne.URI{uri}
	ctx := WithSources(context.Background(), sources)
	sources[0] = storage.NewFileURI("/replacement.png")
	if SourceForPath(ctx, "/selected.png") != uri {
		t.Fatal("source authority was not captured")
	}
	transfer, release, err := exportTransfer(ctx, SourceForPath(ctx, "/selected.png"), func(path string) ([]byte, error) {
		if !active || path != "/resolved.png" {
			t.Fatal("bookmark capture outside resolved authority")
		}
		return []byte("implicit"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !active || string(transfer.Bookmark) != "implicit" || transfer.Path != "/resolved.png" {
		t.Fatal("transfer lost access")
	}
	release()
	release()
	if active || stopped != 1 {
		t.Fatalf("released %d times", stopped)
	}
}

func TestTransferCaptureFailureReleasesScope(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		ctx, stop := context.WithCancel(context.Background())
		released := false
		uri := &sourceURI{URI: storage.NewFileURI("/selected"), record: Record{URI: "file:///selected", Bookmark: []byte("scope")},
			resolve: func(_ context.Context, _ Record) (resolution, func(), error) {
				return resolution{path: "/selected"}, func() { released = true }, nil
			}}
		_, release, err := exportTransfer(ctx, uri, func(_ string) ([]byte, error) {
			if cancel {
				stop()
				return []byte("implicit"), nil
			}
			return nil, errors.New("refused")
		})
		stop()
		if err == nil || release != nil || !released {
			t.Fatal("failed capture retained access")
		}
	}
}

func TestTransferCancelledBeforeCapture(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := exportTransfer(ctx, storage.NewFileURI("/selected"), func(_ string) ([]byte, error) { t.Fatal("cancelled transfer created a grant"); return nil, nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
