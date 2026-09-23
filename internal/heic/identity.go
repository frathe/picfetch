package heic

import "runtime"

// SystemIdentity reads OS identity only; it never enumerates or loads codecs.
// An unreadable version is left empty so observations cannot be reused.
func SystemIdentity() Identity {
	return Identity{OS: runtime.GOOS, Architecture: runtime.GOARCH, OSVersion: systemVersion(), Revision: Revision}
}
