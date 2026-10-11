// Package session persists and restores the set of files that were open
// when the window last closed, via Fyne's app-scoped cache.
package session

import (
	"context"
	"encoding/json"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/fileaccess"
)

// cacheKey names the cache entry Save/Load read and write via app.Cache() -
// the same mechanism Fyne uses for any other app-scoped cache blob, keyed by
// name rather than a raw file path so it works the same under the test
// driver's in-memory cache as it does under the real on-disk one.
const cacheKey = "session.json"

// state is the on-disk representation of the file set that was loaded when
// the window last closed. A struct rather than a bare string slice so the
// format can grow a field later without breaking decode of what's already
// on disk.
type state struct {
	Files  []string             `json:"files"`
	Access *fileaccess.Manifest `json:"access,omitempty"`
}

// Save records files as the session offered on the next launch. An empty
// files removes any previously saved session instead of writing one, so
// quitting after Escape (reset) - a deliberate "start fresh" - doesn't leave
// a stale offer for a session the user already walked away from.
func Save(app fyne.App, files []fyne.URI) {
	cache := app.Cache()

	if len(files) == 0 {
		if cache.Exists(cacheKey) {
			if err := cache.Remove(cacheKey); err != nil {
				fyne.LogError("failed to clear session", err)
			}
		}
		return
	}

	uris := make([]string, len(files))
	for i, u := range files {
		uris[i] = u.String()
	}
	manifest, err := fileaccess.Pack(context.Background(), files)
	if err != nil {
		fyne.LogError("failed to capture session", err)
		return
	}
	var access *fileaccess.Manifest
	if len(manifest.Scopes) != 0 {
		access = &manifest
	}

	w, err := cache.Write(cacheKey)
	if err != nil {
		fyne.LogError("failed to save session", err)
		return
	}
	if err := json.NewEncoder(w).Encode(state{Files: uris, Access: access}); err != nil {
		_ = w.Close()
		fyne.LogError("failed to save session", err)
		return
	}
	if err := w.Close(); err != nil {
		fyne.LogError("failed to finish saving session", err)
	}
}

// Load returns the file set saved by the previous run's Save call, or nil
// if there is none - first launch, a cleared session, or a corrupt cache
// entry are all treated the same as "nothing to restore" rather than
// surfaced as an error, since there's no user action to report one to yet.
func Load(app fyne.App) []fyne.URI {
	cache := app.Cache()
	if !cache.Exists(cacheKey) {
		return nil
	}

	r, err := cache.Read(cacheKey)
	if err != nil {
		return nil
	}
	defer func() {
		if err := r.Close(); err != nil {
			fyne.LogError("failed to close session cache reader", err)
		}
	}()

	var s state
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return nil
	}

	if s.Access != nil {
		if len(s.Access.Sources) != len(s.Files) {
			return nil
		}
		for i, source := range s.Access.Sources {
			if source.URI != s.Files[i] {
				return nil
			}
		}
		uris, err := fileaccess.Unpack(context.Background(), *s.Access)
		if err != nil {
			return nil
		}
		return uris
	}

	uris := make([]fyne.URI, 0, len(s.Files))
	for _, u := range s.Files {
		if parsed, err := storage.ParseURI(u); err == nil {
			uris = append(uris, parsed)
		}
	}
	return uris
}
