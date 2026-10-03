//go:build !microsoftstore && !appleappstore

// Package distribution exposes immutable build-channel facts.
package distribution

// StoreManaged reports whether an app store owns delivery and updates for
// this build. Ordinary direct-download builds keep PicFetch's GitHub updater.
const StoreManaged = false

// AppleAppStore identifies the Mac App Store channel at compile time.
const AppleAppStore = false
