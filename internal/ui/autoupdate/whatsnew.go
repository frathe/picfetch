package autoupdate

import (
	"encoding/json"

	"fyne.io/fyne/v2"
)

// WhatsNewCacheKey is the Fyne app cache entry ApplyStagedUpdate writes to
// right before it replaces the running binary, and internal/ui's
// maybeShowWhatsNew reads on the next launch.
const WhatsNewCacheKey = "whatsnew.json"

// WhatsNew is the cached release-notes payload for the version that was
// just applied.
type WhatsNew struct {
	Version string `json:"version"`
	Body    string `json:"body"`
}

// SaveWhatsNew stores version/body in app's cache, ready for the next
// launch's maybeShowWhatsNew.
func SaveWhatsNew(app fyne.App, version, body string) error {
	w, err := app.Cache().Write(WhatsNewCacheKey)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(WhatsNew{Version: version, Body: body}); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

// LoadWhatsNew reads the cached payload, or nil if nothing is cached.
func LoadWhatsNew(app fyne.App) (*WhatsNew, error) {
	cache := app.Cache()
	if !cache.Exists(WhatsNewCacheKey) {
		return nil, nil
	}
	r, err := cache.Read(WhatsNewCacheKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	var s WhatsNew
	if err := json.NewDecoder(r).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ClearWhatsNew removes the cached payload, if any.
func ClearWhatsNew(app fyne.App) error {
	cache := app.Cache()
	if !cache.Exists(WhatsNewCacheKey) {
		return nil
	}
	return cache.Remove(WhatsNewCacheKey)
}
