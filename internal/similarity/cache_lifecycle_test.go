package similarity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/favstore"
)

func TestAnalysisCacheClearRetiresOpenWriters(t *testing.T) {
	for _, favorite := range []bool{false, true} {
		name := "general"
		if favorite {
			name = "favorite"
		}
		t.Run(name, func(t *testing.T) {
			item := cacheFixtureItem(t, "source.jpg")
			policy := cacheTestPolicy(t)
			if favorite {
				cacheTestFavorite(t, policy.Roots, item)
			}
			old := cacheTestStore(t, policy)
			if err := old.write(context.Background(), item); err != nil {
				t.Fatal(err)
			}
			report, err := (CacheManager{}).Clean(context.Background(), CacheCleanRequest{Roots: policy.Roots, Mode: ClearAll}, nil)
			if err != nil || report.RemovedRecords != 1 {
				t.Fatalf("clear: %+v, %v", report, err)
			}
			if err := old.write(context.Background(), item); !errors.Is(err, ErrCacheRetired) {
				t.Fatalf("old writer was not retired: %v", err)
			}
			fresh := cacheTestStore(t, policy)
			if _, ok := fresh.read(context.Background(), item); ok {
				t.Fatal("old writer repopulated the cleared cache")
			}
			if err := fresh.write(context.Background(), item); err != nil {
				t.Fatalf("new writer was not admitted: %v", err)
			}
			if _, ok := fresh.read(context.Background(), item); !ok {
				t.Fatal("fresh writer did not persist a reusable record")
			}
		})
	}
}

func TestAnalysisCacheMaintenanceLocksInitiallyAbsentRoots(t *testing.T) {
	for _, retune := range []bool{false, true} {
		name := "clear"
		if retune {
			name = "retune"
		}
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			roots := CacheRoots{GeneralDir: filepath.Join(base, "general"), FavoritesDir: filepath.Join(base, "favorites")}
			manager := CacheManager{Quiesce: func(_ context.Context, _ CacheRoots) error {
				for _, path := range []string{roots.GeneralDir, roots.FavoritesDir} {
					// Mimic another process opening a root after maintenance started.
					if err := os.MkdirAll(path, 0700); err != nil {
						return err
					}
					file, err := os.OpenFile(filepath.Join(path, ".analysis-lock"), os.O_CREATE|os.O_RDWR, 0600)
					if err != nil {
						return err
					}
					ctx, cancel := context.WithCancel(context.Background())
					err = lockCacheFile(&cacheTestCancelAfterLockAttempt{Context: ctx, cancel: cancel}, file)
					if err == nil {
						unlockCacheFile(file)
					}
					cancel()
					_ = file.Close()
					if !errors.Is(err, context.Canceled) {
						return errors.New("another writer acquired a root during maintenance: " + path)
					}
				}
				return nil
			}}
			var err error
			if retune {
				_, err = manager.Retune(context.Background(), CacheRetuneRequest{Roots: roots, LimitBytes: 1, RetireWriters: true}, nil)
			} else {
				_, err = manager.Clean(context.Background(), CacheCleanRequest{Roots: roots, Mode: ClearAll}, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Permit one nonblocking native lock attempt, then cancel a contended wait.
// This distinguishes held/free locks without a sleep or a scheduling guess.
type cacheTestCancelAfterLockAttempt struct {
	context.Context
	cancel context.CancelFunc
}

func (c *cacheTestCancelAfterLockAttempt) Err() error {
	err := c.Context.Err()
	c.cancel()
	return err
}

func TestAnalysisCacheMaintenanceQuiescenceJoinsProducer(t *testing.T) {
	policy := cacheTestPolicy(t)
	item := cacheFixtureItem(t, "source.jpg")
	store := cacheTestStore(t, policy)
	if err := store.write(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	producerCtx, cancelProducer := context.WithCancel(context.Background())
	defer cancelProducer()
	resume := make(chan struct{})
	var resumeOnce sync.Once
	resumeProducer := func() { resumeOnce.Do(func() { close(resume) }) }
	defer resumeProducer()
	producerDone := make(chan struct{})
	var producerErr error
	go func() {
		defer close(producerDone)
		<-resume
		producerErr = store.write(producerCtx, item)
	}()
	t.Cleanup(func() { resumeProducer(); cacheTestReceive(t, producerDone) })
	quiescing := make(chan struct{})
	manager := CacheManager{Quiesce: func(ctx context.Context, roots CacheRoots) error {
		if roots != policy.Roots {
			return errors.New("quiescence received another cache scope")
		}
		cancelProducer()
		close(quiescing)
		select {
		case <-producerDone:
			_, err := os.Stat(cacheTestGeneralPath(policy.Roots, item))
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	completed := make(chan cacheTestCleanResult, 1)
	go func() {
		report, err := manager.Clean(ctx, CacheCleanRequest{Roots: policy.Roots, Mode: ClearAll}, nil)
		completed <- cacheTestCleanResult{report, err}
	}()
	cacheTestReceive(t, quiescing)
	if _, err := os.Stat(cacheTestGeneralPath(policy.Roots, item)); err != nil {
		t.Fatalf("record was removed before the producer joined: %v", err)
	}
	resumeProducer()
	result := cacheTestReceive(t, completed)
	if result.err != nil || result.report.RemovedRecords != 1 || !errors.Is(producerErr, context.Canceled) {
		t.Fatalf("quiescence result: %+v, %v; producer %v", result.report, result.err, producerErr)
	}
}

func TestAnalysisCacheMaintenanceCancellationReportsRemaining(t *testing.T) {
	t.Run("after_quiescence", func(t *testing.T) {
		policy := cacheTestPolicy(t)
		store := cacheTestStore(t, policy)
		if err := store.write(context.Background(), cacheFixtureItem(t, "source.jpg")); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		manager := CacheManager{Quiesce: func(_ context.Context, _ CacheRoots) error { cancel(); return nil }}
		report, err := manager.Clean(ctx, CacheCleanRequest{Roots: policy.Roots, Mode: ClearAll}, func(progress CacheProgress) {
			if progress.Phase == "inspect" {
				// This writer commits after the preliminary directory snapshot
				// and before maintenance locks/invalidate its producer lease.
				if err := store.write(context.Background(), cacheFixtureItem(t, "arrived.jpg")); err != nil {
					t.Fatal(err)
				}
			}
		})
		if !errors.Is(err, context.Canceled) || !report.Canceled || report.Remaining.General.Records != 0 || !report.Remaining.Incomplete || report.RemovedRecords != 0 {
			t.Fatalf("canceled second inventory did not report its partial observation: %+v, %v", report, err)
		}
	})

	t.Run("after_one_removal", func(t *testing.T) {
		policy := cacheTestPolicy(t)
		store := cacheTestStore(t, policy)
		for _, name := range []string{"first.jpg", "second.jpg", "third.jpg"} {
			if err := store.write(context.Background(), cacheFixtureItem(t, name)); err != nil {
				t.Fatal(err)
			}
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		report, err := (CacheManager{}).Clean(ctx, CacheCleanRequest{Roots: policy.Roots, Mode: ClearAll}, func(progress CacheProgress) {
			if progress.Phase == "remove" {
				cancel()
				// A cancellation result must use its tracked remainder, without
				// starting another inventory after cancellation.
				path := filepath.Join(policy.Roots.GeneralDir, "v1", strings.Repeat("f", 64)+".json")
				if err := os.WriteFile(path, []byte("outside the canceled inventory"), 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
		if !errors.Is(err, context.Canceled) || !report.Canceled || report.RemovedRecords != 1 || report.Remaining.General.Records != 2 || report.Remaining.Incomplete {
			t.Fatalf("cancelled cleanup: %+v, %v", report, err)
		}
		if report.Before.General.Bytes != report.RemovedBytes+report.Remaining.General.Bytes {
			t.Fatalf("cancelled cleanup lost byte accounting: %+v", report)
		}
	})
	t.Run("waiting_for_writer", func(t *testing.T) {
		policy := cacheTestPolicy(t)
		item := cacheFixtureItem(t, "source.jpg")
		store := cacheTestStore(t, policy)
		if err := store.write(context.Background(), item); err != nil {
			t.Fatal(err)
		}
		resume := make(chan struct{})
		var resumeOnce sync.Once
		resumeWriter := func() { resumeOnce.Do(func() { close(resume) }) }
		writing := &cacheTestHeldWriteContext{Context: context.Background(), directory: filepath.Join(policy.Roots.GeneralDir, "v1"), ready: make(chan struct{}), resume: resume}
		writerDone := make(chan struct{})
		writerResult := make(chan error, 1)
		go func() {
			defer close(writerDone)
			writerResult <- store.write(writing, item)
		}()
		t.Cleanup(func() { resumeWriter(); cacheTestReceive(t, writerDone) })
		cacheTestReceive(t, writing.ready)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		waiting := &cacheTestAdmissionContext{Context: ctx, admission: make(chan struct{})}
		completed := make(chan cacheTestCleanResult, 1)
		go func() {
			report, err := (CacheManager{}).Clean(waiting, CacheCleanRequest{Roots: policy.Roots, Mode: ClearAll}, nil)
			completed <- cacheTestCleanResult{report, err}
		}()
		cacheTestReceive(t, waiting.admission)
		cancel()
		result := cacheTestReceive(t, completed)
		resumeWriter()
		if err := cacheTestReceive(t, writerResult); err != nil {
			t.Fatalf("writer did not finish after maintenance cancellation: %v", err)
		}
		if !errors.Is(result.err, context.Canceled) || !result.report.Canceled || !result.report.Remaining.Incomplete || result.report.RemovedRecords != 0 || result.report.Remaining.General.Records != 1 {
			t.Fatalf("cancelled admission: %+v, %v", result.report, result.err)
		}
		if _, ok := store.read(context.Background(), item); !ok {
			t.Fatal("cancelled lock admission invalidated the active writer")
		}
	})
}

func TestAnalysisCacheMaintenancePartialFailure(t *testing.T) {
	policy := cacheTestPolicy(t)
	item := cacheFixtureItem(t, "source.jpg")
	cacheTestFavorite(t, policy.Roots, item)
	store := cacheTestStore(t, policy)
	if err := store.write(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	cacheTestWriteFile(t, filepath.Join(policy.Roots.GeneralDir, "v1"), []byte("not a directory"))
	usage, err := (CacheManager{}).Inspect(context.Background(), policy.Roots, nil)
	// Inspection returns the measured partial inventory together with its error.
	//goland:noinspection GoDfaErrorMayBeNotNil
	if err == nil || !usage.Incomplete || usage.Favorite.Records != 1 {
		t.Fatalf("partial inspection: %+v, %v", usage, err)
	}
	report, err := (CacheManager{}).Clean(context.Background(), CacheCleanRequest{Roots: policy.Roots, Mode: ClearAll}, nil)
	// Cleanup reports completed removals even when another root could not be read.
	//goland:noinspection GoDfaErrorMayBeNotNil
	if err == nil || !report.Before.Incomplete || !report.Remaining.Incomplete || report.RemovedRecords != 1 || report.Remaining.Favorite.Records != 0 {
		t.Fatalf("partial cleanup: %+v, %v", report, err)
	}
	if report.RemovedBytes != usage.Favorite.Bytes {
		t.Fatalf("partial cleanup lost the successful disk effect: %+v", report)
	}
}

func TestAnalysisCacheMaintenanceUnreadableFavoriteKeepsHealthyPeers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating the unreadable symlink fixture requires Unix")
	}
	for name, mode := range map[string]CacheCleanMode{"clear": ClearAll, "stale": RemoveStale} {
		t.Run(name, func(t *testing.T) {
			policy := cacheTestPolicy(t)
			item := cacheFixtureItem(t, "source.jpg")
			cacheTestFavorite(t, policy.Roots, item)
			store := cacheTestStore(t, policy)
			if err := store.write(context.Background(), item); err != nil {
				t.Fatal(err)
			}
			blocked := filepath.Join(policy.Roots.FavoritesDir, "Blocked")
			if err := os.Mkdir(blocked, 0700); err != nil {
				t.Fatal(err)
			}
			// A self-referencing definition fails Stat even when CI runs as root.
			if err := os.Symlink("file-list.json", filepath.Join(blocked, "file-list.json")); err != nil {
				t.Fatal(err)
			}
			manager := CacheManager{}
			usage, err := manager.Inspect(context.Background(), policy.Roots, nil)
			// An unreadable peer accompanies a useful partial inventory.
			//goland:noinspection GoDfaErrorMayBeNotNil
			if err == nil || !usage.Incomplete || usage.Favorite.Records != 1 {
				t.Fatalf("healthy Favorite missing from partial inventory: %+v, %v", usage, err)
			}
			cacheTestWriteFile(t, item.Path, []byte("changed source version"))
			report, err := manager.Clean(context.Background(), CacheCleanRequest{Roots: policy.Roots, Mode: mode}, nil)
			// Healthy peers still have completed removals despite the partial error.
			//goland:noinspection GoDfaErrorMayBeNotNil
			if err == nil || !report.Remaining.Incomplete || report.RemovedRecords != 1 || report.Remaining.Favorite.Records != 0 {
				t.Fatalf("healthy Favorite skipped during maintenance: %+v, %v", report, err)
			}
		})
	}
}

func TestAnalysisCacheConfinementManagedUsageAndTemps(t *testing.T) {
	policy := cacheTestPolicy(t)
	item := cacheFixtureItem(t, "source.jpg")
	cacheTestFavorite(t, policy.Roots, item)
	general := filepath.Join(policy.Roots.GeneralDir, "v1")
	favorite := filepath.Join(favstore.Dir(policy.Roots.FavoritesDir, "Trip"), "analysis")
	for _, path := range []string{
		filepath.Join(general, strings.Repeat("a", 64)+".json"),
		filepath.Join(general, "."+strings.Repeat("b", 26)+".tmp"),
		filepath.Join(favorite, strings.Repeat("c", 64)+".json"),
		filepath.Join(favorite, "."+strings.Repeat("d", 26)+".tmp"),
	} {
		cacheTestWriteFile(t, path, []byte("12345"))
	}
	protected := []string{
		filepath.Join(general, "notes.json"),
		filepath.Join(general, ".short.tmp"),
		filepath.Join(general, strings.Repeat("E", 64)+".json"),
		filepath.Join(general, "nested", strings.Repeat("f", 64)+".json"),
		filepath.Join(policy.Roots.GeneralDir, "vision_model.onnx"),
		filepath.Join(favstore.Dir(policy.Roots.FavoritesDir, "Trip"), "thumbs", "preview.jpg"),
		filepath.Join(favstore.Dir(policy.Roots.FavoritesDir, "Trip"), "cohorts.json"),
	}
	for _, path := range protected {
		cacheTestWriteFile(t, path, []byte("keep"))
	}
	manager := CacheManager{Quiesce: func(_ context.Context, _ CacheRoots) error { return nil }}
	usage, err := manager.Inspect(context.Background(), policy.Roots, nil)
	if err != nil || usage.General != (CacheSize{Bytes: 10, Records: 1}) || usage.Favorite != (CacheSize{Bytes: 10, Records: 1}) {
		t.Fatalf("managed usage including temps: %+v, %v", usage, err)
	}
	report, err := manager.Clean(context.Background(), CacheCleanRequest{Roots: policy.Roots, Mode: ClearAll}, nil)
	if err != nil || report.RemovedBytes != 20 || report.RemovedRecords != 2 || report.Remaining.General.Bytes != 0 || report.Remaining.Favorite.Bytes != 0 {
		t.Fatalf("managed clear: %+v, %v", report, err)
	}
	for _, path := range protected {
		if data, err := os.ReadFile(path); err != nil || string(data) != "keep" {
			t.Fatalf("cleanup changed an excluded file: %s: %q, %v", path, data, err)
		}
	}
	files, err := favstore.Load(policy.Roots.FavoritesDir, "Trip")
	if err != nil || len(files) != 1 || files[0].Path() != item.Path {
		t.Fatalf("cleanup changed Favorite membership: %v, %v", files, err)
	}
	if data, err := os.ReadFile(item.Path); err != nil || string(data) != "source" {
		t.Fatalf("cleanup changed the source image: %q, %v", data, err)
	}
	t.Run("symlink_target", func(t *testing.T) {
		roots := CacheRoots{GeneralDir: t.TempDir()}
		target := filepath.Join(t.TempDir(), "outside.jpg")
		cacheTestWriteFile(t, target, []byte("outside"))
		dir := filepath.Join(roots.GeneralDir, "v1")
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, strings.Repeat("a", 64)+".json")
		if err := os.Symlink(target, link); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("symlink creation needs Windows privilege: %v", err)
			}
			t.Fatal(err)
		}
		result, err := (CacheManager{}).Clean(context.Background(), CacheCleanRequest{Roots: roots, Mode: ClearAll}, nil)
		if err != nil || result.RemovedRecords != 0 || result.Before.General.Bytes != 0 {
			t.Fatalf("symlink was admitted as a managed record: %+v, %v", result, err)
		}
		if data, err := os.ReadFile(target); err != nil || string(data) != "outside" {
			t.Fatalf("cleanup followed a symlink: %q, %v", data, err)
		}
	})
}

func TestAnalysisCacheStaleSourcesPayloadsAndMembership(t *testing.T) {
	for _, kind := range []string{"valid", "changed", "missing", "disconnected", "disconnected_mountpoint", "old_version", "corrupt", "zero_vector", "oversized", "removed_membership"} {
		t.Run(kind, func(t *testing.T) {
			policy := cacheTestPolicy(t)
			item := cacheFixtureItem(t, "source.jpg")
			if kind == "removed_membership" {
				cacheTestFavorite(t, policy.Roots, item)
			}
			store := cacheTestStore(t, policy)
			if err := store.write(context.Background(), item); err != nil {
				t.Fatal(err)
			}
			record := cacheTestGeneralPath(policy.Roots, item)
			switch kind {
			case "changed":
				cacheTestWriteFile(t, item.Path, []byte("changed source"))
			case "missing":
				if err := os.Remove(item.Path); err != nil {
					t.Fatal(err)
				}
			case "disconnected", "disconnected_mountpoint":
				if err := os.Rename(filepath.Dir(item.Path), filepath.Join(t.TempDir(), "offline")); err != nil {
					t.Fatal(err)
				}
				if kind == "disconnected_mountpoint" {
					if err := os.Mkdir(filepath.Dir(item.Path), 0700); err != nil {
						t.Fatal(err)
					}
				}
			case "old_version":
				data := strings.Replace(string(cacheTestPayload(t, item)), RepresentationVersion, "old-model/oriented-v0", 1)
				cacheTestWriteFile(t, record, []byte(data))
			case "corrupt":
				cacheTestWriteFile(t, record, []byte("{broken"))
			case "zero_vector":
				item.Embedding = make([]float32, 768)
				cacheTestWriteFile(t, record, cacheTestPayload(t, item))
			case "oversized":
				cacheTestWriteFile(t, record, []byte(strings.Repeat(" ", 1024*1024+1)))
			case "removed_membership":
				record = filepath.Join(favstore.Dir(policy.Roots.FavoritesDir, "Trip"), analysisName(item.Path))
				cacheTestFavorite(t, policy.Roots)
			}
			report, err := (CacheManager{}).Clean(context.Background(), CacheCleanRequest{Roots: policy.Roots, Mode: RemoveStale}, nil)
			if err != nil {
				t.Fatal(err)
			}
			_, statErr := os.Stat(record)
			if kind == "valid" || kind == "missing" || strings.HasPrefix(kind, "disconnected") {
				if statErr != nil || report.RemovedRecords != 0 || report.Skipped != 1 {
					t.Fatalf("usable or unavailable source lost: %+v, %v", report, statErr)
				}
				if kind != "valid" && report.Unavailable != 1 {
					t.Fatalf("disconnected source was not reported unavailable: %+v", report)
				}
			} else if !errors.Is(statErr, os.ErrNotExist) || report.RemovedRecords != 1 || report.Remaining.General.Records+report.Remaining.Favorite.Records != 0 {
				t.Fatalf("stale record was retained: %+v, %v", report, statErr)
			}
		})
	}
}

func cacheTestWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestAnalysisCacheStaleRevalidatesFavoriteAfterInventory(t *testing.T) {
	policy := cacheTestPolicy(t)
	item := cacheFixtureItem(t, "restored-member.jpg")
	cacheTestFavorite(t, policy.Roots, item)
	store := cacheTestStore(t, policy)
	if err := store.write(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	cacheTestFavorite(t, policy.Roots)
	// General records are visited before Favorites. Its removal callback lets
	// another instance replace the Favorite after the full inventory is captured.
	marker := filepath.Join(policy.Roots.GeneralDir, "v1", strings.Repeat("a", 64)+".json")
	cacheTestWriteFile(t, marker, []byte("corrupt"))
	replaced := false
	report, err := (CacheManager{}).Clean(context.Background(), CacheCleanRequest{Roots: policy.Roots, Mode: RemoveStale}, func(progress CacheProgress) {
		if progress.Phase == "remove" && !replaced {
			replaced = true
			cacheTestFavorite(t, policy.Roots, item)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(favstore.Dir(policy.Roots.FavoritesDir, "Trip"), analysisName(item.Path))
	if _, err := os.Stat(record); err != nil {
		t.Fatalf("stale membership deleted a newly restored Favorite member: %v", err)
	}
	if !replaced || report.RemovedRecords != 1 || report.Skipped != 1 || report.Unavailable != 1 {
		t.Fatalf("membership replacement was not conservatively reported: %+v", report)
	}
}

type cacheTestCleanResult struct {
	report CacheReport
	err    error
}

// The temporary record is an observable disk effect. Hold its writer before
// publication without changing the production implementation or using a delay.
type cacheTestHeldWriteContext struct {
	context.Context
	directory string
	ready     chan struct{}
	resume    <-chan struct{}
	once      sync.Once
}

func (c *cacheTestHeldWriteContext) Err() error {
	entries, _ := os.ReadDir(c.directory)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			c.once.Do(func() { close(c.ready); <-c.resume })
			break
		}
	}
	return c.Context.Err()
}

type cacheTestAdmissionContext struct {
	context.Context
	admission chan struct{}
	once      sync.Once
}

func (c *cacheTestAdmissionContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.admission) })
	return c.Context.Done()
}

func cacheTestReceive[T any](t *testing.T, channel <-chan T) T {
	t.Helper()
	var result T
	select {
	case value := <-channel:
		result = value
	case <-time.After(5 * time.Second):
		t.Fatal("cache operation did not reach its observable completion")
	}
	return result
}
