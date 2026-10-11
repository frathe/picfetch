package fileaccess

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
)

func TestSourceAcquireKeepsIndependentOperationLifetimes(t *testing.T) {
	var active atomic.Int32
	source, err := FromRecord(Record{URI: "file:///original/photo.jpg", Bookmark: []byte("bookmark")})
	if err != nil {
		t.Fatal(err)
	}
	source.(*sourceURI).resolve = func(_ context.Context, _ Record) (resolution, func(), error) {
		active.Add(1)
		return resolution{path: "/moved/photo.jpg"}, func() { active.Add(-1) }, nil
	}
	first, releaseFirst, err := Acquire(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	second, releaseSecond, err := Acquire(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if first.Path() != "/moved/photo.jpg" || second.Path() != first.Path() || source.Path() != "/original/photo.jpg" {
		t.Fatal("lost resolved path or changed captured identity")
	}
	releaseFirst()
	releaseFirst()
	if active.Load() != 1 {
		t.Fatal("release retired another operation")
	}
	releaseSecond()
	if active.Load() != 0 {
		t.Fatal("access leaked")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := Acquire(ctx, source); !errors.Is(err, context.Canceled) || active.Load() != 0 {
		t.Fatalf("cancelled acquire: %v", err)
	}
}

func TestChildCannotBroadenCapturedDirectory(t *testing.T) {
	directory, err := FromRecord(Record{URI: "file:///selected", Bookmark: []byte("folder"), Directory: true})
	if err != nil {
		t.Fatal(err)
	}
	child, err := Child(directory, storage.NewFileURI("/selected/sub/photo.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	record := Snapshot(child)
	if record.Relative != "sub/photo.jpg" {
		t.Fatal(record.Relative)
	}
	record.Bookmark[0] = 'X'
	if string(Snapshot(directory).Bookmark) != "folder" || string(Snapshot(child).Bookmark) != "folder" {
		t.Fatal("mutable bookmark escaped")
	}
	for _, path := range []string{"/other/photo.jpg", "/selected/../other.jpg", "/selected-sibling/photo.jpg"} {
		if _, err := Child(directory, storage.NewFileURI(path)); err == nil {
			t.Fatalf("broadened to %q", path)
		}
	}
	file, err := FromRecord(Record{URI: "file:///selected/photo.jpg", Bookmark: []byte("file")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Child(file, storage.NewFileURI("/selected/sibling.jpg")); err == nil {
		t.Fatal("file grant authorized sibling")
	}
}

func TestRecordRejectsRelativeEscape(t *testing.T) {
	for _, relative := range []string{"../other", "sub/../../other", "/absolute", "sub/../other", "sub//other"} {
		if _, err := FromRecord(Record{URI: "file:///selected/photo.jpg", Bookmark: []byte("folder"), Directory: true, Relative: relative}); err == nil {
			t.Fatalf("accepted %q", relative)
		}
	}
}

func TestReaderRetainsScopeUntilCloseAndRollsBackFailure(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	path := filepath.Join(t.TempDir(), "source.dat")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	source, err := FromRecord(Record{URI: "file:///selected/source.dat", Bookmark: []byte("grant")})
	if err != nil {
		t.Fatal(err)
	}
	var active atomic.Int32
	source.(*sourceURI).resolve = func(_ context.Context, _ Record) (resolution, func(), error) {
		active.Add(1)
		return resolution{path: path}, func() { active.Add(-1) }, nil
	}
	reader, err := Reader(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(reader)
	if err != nil || string(content) != "fixture" || active.Load() != 1 {
		t.Fatalf("read %q, err %v, active %d", content, err, active.Load())
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if active.Load() != 0 {
		t.Fatal("reader leaked or released twice")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := Reader(context.Background(), source); err == nil || active.Load() != 0 {
		t.Fatal("failed open leaked its scope")
	}
}

func TestAcquiredDirectoryPropagatesMovedScopeToChild(t *testing.T) {
	directory, err := FromRecord(Record{URI: "file:///selected", Bookmark: []byte("folder"), Directory: true})
	if err != nil {
		t.Fatal(err)
	}
	directory.(*sourceURI).resolve = func(_ context.Context, _ Record) (resolution, func(), error) {
		return resolution{path: "/moved"}, func() {}, nil
	}
	resolved, release, err := Acquire(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	child, err := Child(resolved, storage.NewFileURI("/moved/photo.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	record := Snapshot(child)
	if record.URI != child.String() || record.Relative != "photo.jpg" || string(record.Bookmark) != "folder" {
		t.Fatalf("child lost moved scope: %+v", record)
	}
}

func TestParentRequiresCapturedDirectoryAuthority(t *testing.T) {
	directory, err := FromRecord(Record{URI: "file:///selected", Bookmark: []byte("folder"), Directory: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parent(directory); err == nil {
		t.Fatal("selected directory authorized its parent")
	}
	child, err := Child(directory, storage.NewFileURI("/selected/sub/photo.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	parent, err := Parent(child)
	if err != nil {
		t.Fatal(err)
	}
	if Snapshot(parent).Relative != "sub" || filepath.ToSlash(filepath.Clean(parent.Path())) != "/selected/sub" {
		t.Fatalf("lost parent inside selected scope: path=%q, relative=%q", parent.Path(), Snapshot(parent).Relative)
	}
	file, err := FromRecord(Record{URI: "file:///selected/photo.jpg", Bookmark: []byte("file")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parent(file); err == nil {
		t.Fatal("file grant authorized its parent")
	}
}

func TestAcquireReleasesScopeWhenCancelledDuringResolution(t *testing.T) {
	source, err := FromRecord(Record{URI: "file:///photo.jpg", Bookmark: []byte("grant")})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	released := 0
	source.(*sourceURI).resolve = func(_ context.Context, _ Record) (resolution, func(), error) {
		cancel()
		return resolution{path: "/photo.jpg"}, func() { released++ }, nil
	}
	uri, release, err := Acquire(ctx, source)
	if !errors.Is(err, context.Canceled) || uri != nil || release != nil || released != 1 {
		t.Fatalf("cancelled acquisition returned %v, %v; released %d", uri, err, released)
	}
}

func TestOpeningRefreshesImmutableBookmarkOccurrences(t *testing.T) {
	original, err := FromRecord(Record{URI: "file:///old/sub/photo.jpg", Bookmark: []byte("old"), Directory: true, Relative: "sub/photo.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	calls, released := 0, 0
	renewed := []byte("renewed")
	original.(*sourceURI).resolve = func(_ context.Context, record Record) (resolution, func(), error) {
		calls++
		if string(record.Bookmark) != "old" {
			t.Fatal("captured input changed")
		}
		return resolution{path: "/moved", bookmark: renewed}, func() { released++ }, nil
	}
	opened, err := CaptureSelected(context.Background(), []fyne.URI{original, original})
	if err != nil {
		t.Fatal(err)
	}
	record := Snapshot(opened[0])
	if record.URI != "file:///moved/sub/photo.jpg" || string(record.Bookmark) != "renewed" || record.Relative != "sub/photo.jpg" || !record.Directory {
		t.Fatalf("opening retained stale authority: %+v", record)
	}
	if calls != 1 || released != 1 || opened[0] != opened[1] {
		t.Fatalf("duplicate opening: calls %d, releases %d", calls, released)
	}
	manifest, err := Pack(context.Background(), opened)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Unpack(context.Background(), manifest)
	if err != nil || len(restored) != 2 || restored[1].String() != record.URI || string(Snapshot(restored[1]).Bookmark) != "renewed" {
		t.Fatalf("renewed authority did not survive persistence: %v %v", restored, err)
	}
	renewed[0] = 'X'
	if string(Snapshot(opened[0]).Bookmark) != "renewed" || string(Snapshot(original).Bookmark) != "old" || original.Path() != "/old/sub/photo.jpg" {
		t.Fatal("renewal changed immutable metadata")
	}
}

func TestOpeningPreservesUnavailableSourceButAbortsCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		source, err := FromRecord(Record{URI: "file:///offline/photo.jpg", Bookmark: []byte("saved")})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		source.(*sourceURI).resolve = func(_ context.Context, _ Record) (resolution, func(), error) {
			if cancelled {
				cancel()
			}
			return resolution{}, nil, errors.New("offline")
		}
		opened, err := CaptureSelected(ctx, []fyne.URI{source})
		cancel()
		if cancelled {
			if !errors.Is(err, context.Canceled) || opened != nil {
				t.Fatalf("cancelled opening: %v %v", opened, err)
			}
		} else if err != nil || len(opened) != 1 || opened[0] != source {
			t.Fatalf("lost offline source: %v %v", opened, err)
		}
	}
}
