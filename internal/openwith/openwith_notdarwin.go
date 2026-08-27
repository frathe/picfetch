//go:build !darwin

package openwith

// Install is a no-op everywhere but macOS, and reports so.
//
// Linux and Windows hand a double-clicked or "Open with"-ed file to the
// process in argv, which main.go already turns into URIs before the viewer
// starts, so there is no delegate to graft anything onto and nothing for
// this package's queue to buffer. false is the accurate answer rather than
// an error: nothing failed and there is nothing for the caller to recover
// from, it just logs that no bridge is in place.
func Install() bool { return false }

// DelegateRespondsToOpen is always false off macOS: there is no application
// delegate to graft anything onto. It exists here so a caller can ask the
// question without a build tag of its own.
func DelegateRespondsToOpen() bool { return false }
