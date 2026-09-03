//go:build !microsoftstore

// Package distribution exposes immutable build-channel facts.
package distribution

// StoreManaged reports whether Microsoft Store owns delivery and updates for
// this build. Ordinary direct-download builds keep PicFetch's GitHub updater.
const StoreManaged = false
