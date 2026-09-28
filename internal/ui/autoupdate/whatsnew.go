package autoupdate

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
func (u *Updater) SaveWhatsNew(version, body string) error {
	if !u.updates.Allowed() {
		return unavailableUpdateError()
	}
	return saveCacheJSON(u.app, WhatsNewCacheKey, WhatsNew{Version: version, Body: body})
}

// LoadWhatsNew reads the cached payload, or nil if nothing is cached.
func (u *Updater) LoadWhatsNew() (*WhatsNew, error) {
	if !u.updates.Allowed() {
		return nil, unavailableUpdateError()
	}
	return loadCacheJSON[WhatsNew](u.app, WhatsNewCacheKey)
}

// ClearWhatsNew removes the cached payload, if any.
func (u *Updater) ClearWhatsNew() error {
	if !u.updates.Allowed() {
		return unavailableUpdateError()
	}
	return clearCacheJSON(u.app, WhatsNewCacheKey)
}
