//go:build darwin && appleappstore && !microsoftstore

// Package distribution exposes immutable build-channel facts.
package distribution

// StoreManaged reports that the Mac App Store owns delivery and updates.
const StoreManaged = true

// AppleAppStore identifies the Mac App Store channel at compile time.
const AppleAppStore = true
