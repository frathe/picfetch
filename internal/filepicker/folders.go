package filepicker

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/fileaccess"
)

const keySiblingFolderGrants = "appleSiblingFolderGrants"

// FolderAuthorizer remembers explicit sibling-folder approvals across launches.
// Its methods run on the caller's tracked opening worker.
type FolderAuthorizer struct {
	mu      sync.Mutex
	prefs   fyne.Preferences
	choose  func(context.Context, string) (fyne.URI, error)
	acquire func(context.Context, fyne.URI) (fyne.URI, func(), error)
}

func NewFolderAuthorizer(prefs fyne.Preferences) *FolderAuthorizer {
	return &FolderAuthorizer{prefs: prefs, choose: chooseSiblingFolderDarwin, acquire: fileaccess.Acquire}
}

// AuthorizeSiblingFolder is only for admitted single-image sibling discovery.
func (a *FolderAuthorizer) AuthorizeSiblingFolder(ctx context.Context, files []fyne.URI) ([]fyne.URI, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// The channel is fixed at build time, independently of the host inspector.
	//goland:noinspection GoBoolExpressions
	if !distribution.AppleAppStore {
		return files, nil
	}
	return a.authorize(ctx, files)
}

func (a *FolderAuthorizer) authorize(ctx context.Context, files []fyne.URI) ([]fyne.URI, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// A persisted file bookmark can now name a different path. Resolve its
	// current identity before suggesting or comparing a parent folder; the
	// recorded path is an index, never the authority for a new grant.
	if len(files) == 1 && fileaccess.HasScope(files[0]) && !fileaccess.Snapshot(files[0]).Directory {
		resolved, release, err := a.acquire(ctx, files[0])
		if err != nil {
			return nil, err
		}
		defer release()
		files = []fyne.URI{resolved}
	}
	var approved fyne.URI
	permitted, err := authorizeSiblingFolder(files, func(directory string) (fyne.URI, error) {
		if folder, err := a.recall(ctx, directory); folder != nil || err != nil {
			return folder, err
		}
		var err error
		approved, err = a.choose(ctx, directory)
		return approved, err
	})
	if cancelled := ctx.Err(); cancelled != nil {
		return nil, cancelled
	}
	if err != nil {
		return nil, err
	}
	if approved != nil {
		// The policy above validated the explicitly selected folder.
		a.remember(ctx, "", fileaccess.Snapshot(approved))
	}
	return permitted, nil
}

func (a *FolderAuthorizer) recall(ctx context.Context, directory string) (fyne.URI, error) {
	a.mu.Lock()
	records := a.load()
	a.mu.Unlock()
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Paths are indexes only: native resolution is the authority and can move.
		if !record.Directory || record.Relative != "" || len(record.Bookmark) == 0 {
			continue
		}
		saved, err := fileaccess.FromRecord(record)
		if err != nil {
			continue
		}
		resolved, release, err := a.acquire(ctx, saved)
		if err != nil {
			continue
		}
		directoryExists, statErr := storage.CanList(resolved)
		matches := statErr == nil && directoryExists && filepath.Clean(resolved.Path()) == filepath.Clean(directory)
		refreshed := fileaccess.Snapshot(resolved)
		release()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if matches {
			a.remember(ctx, record.URI, refreshed)
			return resolved, nil
		}
	}
	return nil, ctx.Err()
}

// load and remember serialize preference updates only; native calls never hold mu.
func (a *FolderAuthorizer) load() []fileaccess.Record {
	raw := a.prefs.String(keySiblingFolderGrants)
	if raw == "" {
		return nil
	}
	var records []fileaccess.Record
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		fyne.LogError("Could not read saved folder access", err)
		return nil
	}
	return records
}

func (a *FolderAuthorizer) remember(ctx context.Context, previous string, record fileaccess.Record) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx.Err() != nil {
		return
	}
	records := a.load()
	updated := make([]fileaccess.Record, 0, len(records)+1)
	for _, saved := range records {
		if saved.URI != previous && saved.URI != record.URI {
			updated = append(updated, saved)
		}
	}
	updated = append(updated, record)
	encoded, err := json.Marshal(updated)
	if err != nil {
		fyne.LogError("Could not save folder access", err)
		return
	}
	if string(encoded) != a.prefs.String(keySiblingFolderGrants) {
		a.prefs.SetString(keySiblingFolderGrants, string(encoded))
	}
}
