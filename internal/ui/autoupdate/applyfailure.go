package autoupdate

import "fyne.io/fyne/v2"

// ApplyFailureCacheKey is the Fyne app cache entry ApplyStagedUpdate writes
// when it could not replace the running binary, and internal/ui reads on the
// next launch. The apply runs from Fyne's stopped callback, where there is
// no window left to report into, so the reason has to outlive the process
// that learned it.
const ApplyFailureCacheKey = "updatefailure.json"

// ApplyFailure is the cached account of an update that was downloaded and
// verified but could not be installed - on Windows most often because
// Defender's Controlled Folder Access refused the write to an unsigned
// executable.
type ApplyFailure struct {
	Version string `json:"version"`
	Reason  string `json:"reason"` // update.FailureReason
	Op      string `json:"op"`     // update.ApplyError.Op
	Path    string `json:"path"`   // the executable that could not be replaced

	// Detail is the raw error text, kept so the log can say exactly what the
	// OS refused. What the user is told is derived from Reason instead: an
	// errno-flavoured sentence explains nothing and cannot be translated.
	Detail string `json:"detail"`
}

// SaveApplyFailure stores f in app's cache, replacing any earlier record -
// only the most recent attempt is worth reporting.
func SaveApplyFailure(app fyne.App, f ApplyFailure) error {
	return saveCacheJSON(app, ApplyFailureCacheKey, f)
}

// LoadApplyFailure reads the cached record, or nil if nothing is cached. An
// unreadable record returns an error rather than nil: callers act on the
// absence of a record, so the two cases must stay distinguishable.
func LoadApplyFailure(app fyne.App) (*ApplyFailure, error) {
	return loadCacheJSON[ApplyFailure](app, ApplyFailureCacheKey)
}

// ClearApplyFailure removes the cached record, if any.
func ClearApplyFailure(app fyne.App) error {
	return clearCacheJSON(app, ApplyFailureCacheKey)
}
