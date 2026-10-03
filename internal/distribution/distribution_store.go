//go:build microsoftstore && !appleappstore

// Package distribution exposes immutable build-channel facts.
package distribution

// StoreManaged reports whether Microsoft Store owns delivery and updates for
// this build.
const StoreManaged = true

// AppleAppStore identifies the Mac App Store channel at compile time.
const AppleAppStore = false
