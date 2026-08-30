package autoupdate

import (
	"encoding/json"

	"fyne.io/fyne/v2"
)

// This file holds the Fyne-cache plumbing shared by whatsnew.go and
// applyfailure.go. Both persist one small JSON document across the process
// boundary that ApplyStagedUpdate creates - it runs from Fyne's stopped
// callback, so whatever it learns has to reach the next launch - and both
// were carrying their own copy of the same three subtleties: the writer has
// to be closed even when encoding fails, a missing entry is "nothing
// cached" rather than an error, and Remove is only called on an entry that
// exists. Worth having once.

// saveCacheJSON writes v to key, replacing whatever was there.
func saveCacheJSON[T any](app fyne.App, key string, v T) error {
	w, err := app.Cache().Write(key)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

// loadCacheJSON reads key, or returns nil if nothing is cached. An entry
// that exists but cannot be read or decoded returns an error rather than
// nil: callers act on the absence of a record, so the two cases have to stay
// distinguishable.
func loadCacheJSON[T any](app fyne.App, key string) (*T, error) {
	cache := app.Cache()
	if !cache.Exists(key) {
		return nil, nil
	}
	r, err := cache.Read(key)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	var v T
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// clearCacheJSON removes key, if it is there.
func clearCacheJSON(app fyne.App, key string) error {
	cache := app.Cache()
	if !cache.Exists(key) {
		return nil
	}
	return cache.Remove(key)
}
