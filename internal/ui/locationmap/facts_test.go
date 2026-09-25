package locationmap

import (
	"context"
	"sync"
	"testing"

	"github.com/frathe/picfetch/internal/imaging"
)

func TestFactCacheReusesCompletedFactsIncludingAbsentGPS(t *testing.T) {
	cache := NewFactCache()
	cache.Keep([]string{"located", "absent"})

	located := imaging.Metadata{HasGPS: true, Latitude: 48.858, Longitude: 2.294}
	if !cache.Capture("located", "v1").Store(context.Background(), located) {
		t.Fatal("current located source was not admitted")
	}
	if !cache.Capture("absent", "v1").Store(context.Background(), imaging.Metadata{}) {
		t.Fatal("successful absence of GPS was not admitted")
	}

	if got, ok := cache.Get("located", "v1"); !ok || got != located {
		t.Fatalf("located fact = (%+v, %t), want (%+v, true)", got, ok, located)
	}
	if got, ok := cache.Get("absent", "v1"); !ok || got != (imaging.Metadata{}) {
		t.Fatalf("absent GPS fact = (%+v, %t), want zero metadata and true", got, ok)
	}
	if _, ok := cache.Get("absent", "v2"); ok {
		t.Fatal("a different version reused an absent-GPS fact")
	}
	if cache.Capture("outside", "v1").Store(context.Background(), located) {
		t.Fatal("source outside current membership was admitted")
	}
	if cache.Capture("located", "").Store(context.Background(), located) {
		t.Fatal("empty source version was admitted")
	}
}

func TestFactCacheVersionChangeRetiresOlderWriters(t *testing.T) {
	cache := NewFactCache()
	cache.Keep([]string{"photo"})
	first := cache.Capture("photo", "v1")
	second := cache.Capture("photo", "v1")
	if !first.Store(context.Background(), imaging.Metadata{Latitude: 1}) ||
		!second.Store(context.Background(), imaging.Metadata{Latitude: 2}) {
		t.Fatal("two same-version writers should both remain authorized")
	}

	oldVersion := cache.Capture("photo", "v1")
	newVersion := cache.Capture("photo", "v2")
	if oldVersion.Store(context.Background(), imaging.Metadata{Latitude: 3}) {
		t.Fatal("old-version writer was admitted after a version change")
	}
	if _, ok := cache.Get("photo", "v1"); ok {
		t.Fatal("old-version fact survived a new-version capture")
	}
	if !newVersion.Store(context.Background(), imaging.Metadata{Latitude: 4}) {
		t.Fatal("new-version writer was refused")
	}
	cache.Capture("photo", "v1")
	if oldVersion.Store(context.Background(), imaging.Metadata{Latitude: 5}) {
		t.Fatal("old writer revived when its version became current again")
	}
}

func TestFactCacheInvalidateRetiresFactAndWriters(t *testing.T) {
	cache := NewFactCache()
	cache.Keep([]string{"photo"})
	stale := cache.Capture("photo", "v1")
	if !stale.Store(context.Background(), imaging.Metadata{HasGPS: true}) {
		t.Fatal("setup fact was refused")
	}
	cache.Invalidate([]string{"photo"})
	if _, ok := cache.Get("photo", "v1"); ok {
		t.Fatal("invalidated fact remained reusable")
	}
	if stale.Store(context.Background(), imaging.Metadata{HasGPS: true}) {
		t.Fatal("invalidated writer was still authorized")
	}
	if !cache.Capture("photo", "v1").Store(context.Background(), imaging.Metadata{}) {
		t.Fatal("invalidation removed live membership")
	}
}

func TestFactCacheKeepDeduplicatesAndRetiresRemovedMembers(t *testing.T) {
	cache := NewFactCache()
	cache.Keep([]string{"photo", "photo", "other"})
	writer := cache.Capture("photo", "v1")
	if !writer.Store(context.Background(), imaging.Metadata{HasGPS: true}) {
		t.Fatal("setup fact was refused")
	}
	if len(cache.Snapshot()) != 1 {
		t.Fatalf("duplicate membership produced %d completed facts, want one", len(cache.Snapshot()))
	}
	cache.Keep([]string{"photo", "photo"})
	if _, ok := cache.Get("photo", "v1"); !ok {
		t.Fatal("surviving member lost its fact")
	}
	cache.Keep(nil)
	if _, ok := cache.Get("photo", "v1"); ok {
		t.Fatal("removed member retained its fact")
	}
	cache.Keep([]string{"photo"})
	if writer.Store(context.Background(), imaging.Metadata{HasGPS: true}) {
		t.Fatal("old writer revived after membership was removed and readded")
	}
	if !cache.Capture("photo", "v1").Store(context.Background(), imaging.Metadata{}) {
		t.Fatal("readded member was not admitted")
	}
}

func TestFactCacheRejectsCancelledStore(t *testing.T) {
	cache := NewFactCache()
	cache.Keep([]string{"photo"})
	writer := cache.Capture("photo", "v1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if writer.Store(ctx, imaging.Metadata{HasGPS: true}) {
		t.Fatal("cancelled read was published")
	}
	if _, ok := cache.Get("photo", "v1"); ok {
		t.Fatal("cancelled read left a completed fact")
	}
}

func TestFactCacheSnapshotIsIndependent(t *testing.T) {
	cache := NewFactCache()
	cache.Keep([]string{"photo", "unfinished"})
	want := imaging.Metadata{HasGPS: true, Latitude: 51.5}
	if !cache.Capture("photo", "v1").Store(context.Background(), want) {
		t.Fatal("setup fact was refused")
	}
	cache.Capture("unfinished", "v1")
	snapshot := cache.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("snapshot has %d facts, want only one completed fact", len(snapshot))
	}
	snapshot["photo"] = Fact{Version: "changed", Metadata: imaging.Metadata{Latitude: 0}}
	snapshot["intruder"] = Fact{Version: "v1"}
	if got, ok := cache.Get("photo", "v1"); !ok || got != want {
		t.Fatalf("snapshot mutation changed the cache: (%+v, %t)", got, ok)
	}
	if len(cache.Snapshot()) != 1 {
		t.Fatal("snapshot mutation changed cache membership")
	}
}

func TestFactCacheConcurrentMembershipAndPublication(t *testing.T) {
	cache := NewFactCache()
	cache.Keep([]string{"photo"})
	start := make(chan struct{})
	var workers sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			<-start
			for i := 0; i < 500; i++ {
				switch worker {
				case 0:
					cache.Keep([]string{"photo", "photo"})
					cache.Keep(nil)
				case 1:
					cache.Capture("photo", "v1").Store(context.Background(), imaging.Metadata{Latitude: float64(i)})
				case 2:
					cache.Invalidate([]string{"photo"})
				case 3:
					_, _ = cache.Get("photo", "v1")
					for _, fact := range cache.Snapshot() {
						if fact.Version == "" {
							t.Error("snapshot published an empty version")
						}
					}
				}
			}
		}(worker)
	}
	close(start)
	workers.Wait()

	cache.Keep([]string{"photo"})
	cache.Invalidate([]string{"photo"})
	want := imaging.Metadata{HasGPS: true, Latitude: 7}
	if !cache.Capture("photo", "v2").Store(context.Background(), want) {
		t.Fatal("cache refused fresh publication after concurrent operations")
	}
	if got, ok := cache.Get("photo", "v2"); !ok || got != want {
		t.Fatalf("final fact = (%+v, %t), want (%+v, true)", got, ok, want)
	}
}
