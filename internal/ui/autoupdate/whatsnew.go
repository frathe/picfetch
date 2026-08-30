package autoupdate

import "fyne.io/fyne/v2"

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
	return saveCacheJSON(app, WhatsNewCacheKey, WhatsNew{Version: version, Body: body})
}

// LoadWhatsNew reads the cached payload, or nil if nothing is cached.
func LoadWhatsNew(app fyne.App) (*WhatsNew, error) {
	return loadCacheJSON[WhatsNew](app, WhatsNewCacheKey)
}

// ClearWhatsNew removes the cached payload, if any.
func ClearWhatsNew(app fyne.App) error {
	return clearCacheJSON(app, WhatsNewCacheKey)
}
