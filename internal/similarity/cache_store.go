package similarity

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// CachePressureError asks the parent to make space after retiring the producer.
// A source's usable in-memory representation is independent of this disk effect.
type CachePressureError struct{ NeedBytes uint64 }

func (CachePressureError) Error() string { return "analysis cache needs space" }

type representationStore struct {
	policy             CachePolicy
	favorites          analysisCache
	general            *os.Root
	lease              *cacheLease
	accountedBytes     uint64
	accountedRevision  string
	accountedDirectory os.FileInfo
	accounted          bool
}

func openRepresentationStore(ctx context.Context, policy CachePolicy) (*representationStore, error) {
	store := &representationStore{policy: policy}
	if store.policy.GeneralLimitBytes == 0 {
		store.policy.GeneralLimitBytes = DefaultAnalysisCacheBytes
	}
	// Membership remains available even when Favorite writes are disabled.
	favorites, err := openAnalysisCache(ctx, policy.Roots.FavoritesDir)
	store.favorites = favorites
	if err != nil {
		// Healthy Favorite records remain usable, but unknown membership must
		// not bypass Favorite preferences through general reads or writes.
		return store, err
	}
	if policy.LooseEnabled && policy.Roots.GeneralDir != "" {
		lease, err := openCacheLease(ctx, CacheRoots{GeneralDir: policy.Roots.GeneralDir}, true)
		if err != nil {
			return store, err
		}
		store.lease = lease
		store.general, err = os.OpenRoot(policy.Roots.GeneralDir)
		return store, err
	}
	return store, nil
}
func (s *representationStore) close() {
	if s == nil {
		return
	}
	s.favorites.close()
	if s.general != nil {
		_ = s.general.Close()
	}
	s.lease.close()
}
func (s *representationStore) read(ctx context.Context, source Item) (Item, bool) {
	if s == nil {
		return Item{}, false
	}
	if !s.policy.FavoriteEnabled && len(s.favorites[filepath.Clean(source.Path)]) > 0 {
		return Item{}, false
	}
	if s.policy.FavoriteEnabled {
		if item, ok := s.favorites.read(source); ok {
			return item, true
		}
	}
	if s.general == nil {
		return Item{}, false
	}
	var item Item
	err := s.lease.write(ctx, func() error {
		name := filepath.Join("v1", filepath.Base(analysisName(source.Path)))
		file, err := s.general.Open(name)
		if err != nil {
			return err
		}
		info, err := file.Stat()
		if err != nil || info.Size() > maximumAnalysisRecordBytes {
			_ = file.Close()
			return os.ErrInvalid
		}
		item, err = decodeRepresentation(file)
		_ = file.Close()
		if err != nil {
			return err
		}
		if item.Path != source.Path || item.Size != source.Size || item.ModifiedNS != source.ModifiedNS {
			return os.ErrInvalid
		}
		now := time.Now()
		_ = s.general.Chtimes(name, now, now)
		return nil
	})
	if err != nil {
		return Item{}, false
	}
	if s.policy.FavoriteEnabled && len(s.favorites[filepath.Clean(source.Path)]) > 0 {
		_ = s.favorites.write(ctx, item)
	}
	return item, true
}
func (s *representationStore) write(ctx context.Context, item Item) error {
	if s == nil {
		return nil
	}
	if len(s.favorites[filepath.Clean(item.Path)]) > 0 {
		if s.policy.FavoriteEnabled {
			return s.favorites.write(ctx, item)
		}
		return nil
	}
	if s.general == nil {
		return nil
	}
	item.Cohort, item.Position, item.Thumbnail = "", nil, ""
	item.Tags = nil
	data, err := json.Marshal(cachedRepresentation{Version: RepresentationVersion, Item: item})
	if err != nil {
		return err
	}
	if uint64(len(data)) > s.policy.GeneralLimitBytes || len(data) > maximumAnalysisRecordBytes {
		return nil
	}
	if _, err := decodeRepresentation(bytes.NewReader(data)); err != nil {
		return err
	}
	return s.lease.write(ctx, func() error {
		if err := s.general.Mkdir("v1", 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		usage, err := s.generalUsage(ctx)
		if err != nil {
			return err
		}
		// The budget includes the temporary file. Even a replacement needs room
		// for both records until rename; credit the old bytes only after commit.
		if usage > s.policy.GeneralLimitBytes || uint64(len(data)) > s.policy.GeneralLimitBytes-usage {
			return CachePressureError{NeedBytes: uint64(len(data))}
		}
		destination := filepath.Join("v1", filepath.Base(analysisName(item.Path)))
		var replaced uint64
		if info, err := s.general.Lstat(destination); err == nil {
			if info.Mode().IsRegular() {
				replaced = uint64(info.Size())
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		// Invalidate every other producer's accounting before any disk effect.
		// A failed or interrupted write leaves its successor to inventory again.
		revision := rand.Text()
		s.accounted = false
		if err := s.general.WriteFile(".analysis-write-revision", []byte(revision), 0600); err != nil {
			return err
		}
		temporary := filepath.Join("v1", "."+rand.Text()+".tmp")
		file, err := s.general.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer func() { _ = s.general.Remove(temporary) }()
		_, err = file.Write(data)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.general.Rename(temporary, destination); err != nil {
			return err
		}
		directory, err := s.general.Stat("v1")
		if err != nil {
			return err
		}
		s.accountedBytes = usage - min(usage, replaced) + uint64(len(data))
		s.accountedRevision, s.accountedDirectory, s.accounted = revision, directory, true
		return nil
	})
}

// generalUsage runs under the cache lease. One producer inventories once, then
// updates its byte total after each commit. A cooperating writer changes the
// revision first; directory changes also invalidate the snapshot, including
// leftover managed temporary files. Maintenance retires the lease altogether.
func (s *representationStore) generalUsage(ctx context.Context) (uint64, error) {
	revision, err := s.general.ReadFile(".analysis-write-revision")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	directory, err := s.general.Stat("v1")
	if err != nil {
		return 0, err
	}
	if s.accounted && string(revision) == s.accountedRevision && directory.ModTime().Equal(s.accountedDirectory.ModTime()) && directory.Size() == s.accountedDirectory.Size() {
		return s.accountedBytes, nil
	}
	usage, err := (CacheManager{}).Inspect(ctx, CacheRoots{GeneralDir: s.policy.Roots.GeneralDir}, nil)
	if err != nil {
		return 0, err
	}
	s.accountedBytes, s.accountedRevision, s.accountedDirectory, s.accounted = usage.General.Bytes, string(revision), directory, true
	return s.accountedBytes, nil
}
