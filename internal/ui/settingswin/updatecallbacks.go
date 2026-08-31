package settingswin

// UpdateCallbacks is the Settings window's consumer-side vocabulary for a
// manual update request. Host.CheckForUpdatesNow delivers every callback on Fyne's UI thread.
// Values stay primitive so settingswin does not depend on the updater or
// internal/update packages.
type UpdateCallbacks struct {
	Downloading func(version string)
	Progress    func(downloaded, total int64)
	Current     func()
	Ready       func(version string)
	Failed      func(error)
}
