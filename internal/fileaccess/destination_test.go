package fileaccess

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/storage"
)

func TestDestinationCloseRetainsActiveAccess(t *testing.T) {
	released := 0
	uri := storage.NewFileURI("/selected/not-created.png")
	destination := NewDestination(uri, func() { released++ })
	first, endFirst, err := Acquire(context.Background(), destination)
	if err != nil || first.Path() != uri.Path() {
		t.Fatalf("acquire: %v %v", first, err)
	}
	_, endSecond, err := Acquire(context.Background(), destination)
	if err != nil {
		t.Fatal(err)
	}
	ReleaseDestination(destination)
	ReleaseDestination(destination)
	if released != 0 {
		t.Fatal("closed an active destination")
	}
	if _, release, err := Acquire(context.Background(), destination); err == nil {
		release()
		t.Error("closed destination admitted access")
	}
	endFirst()
	endFirst()
	if released != 0 {
		t.Fatal("released before final borrower")
	}
	endSecond()
	endSecond()
	if released != 1 {
		t.Fatalf("release count = %d", released)
	}
}

func TestDestinationDiscardAndCancelledAdmission(t *testing.T) {
	released := 0
	destination := NewDestination(storage.NewFileURI("/selected/missing.png"), func() { released++ })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, release, err := Acquire(ctx, destination); err == nil {
		release()
		t.Fatal("cancelled access admitted")
	}
	ReleaseDestination(destination)
	ReleaseDestination(destination)
	if released != 1 {
		t.Fatalf("release count = %d", released)
	}
}
