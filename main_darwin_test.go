//go:build darwin

package main

import (
	"testing"

	"github.com/frathe/picfetch/internal/openwith"
)

// TestInstall_GraftsOntoGLFWsDelegate is here, in package main, rather than
// in internal/openwith, because this is the only test binary in the module
// that links Fyne's Cocoa driver (through fyne.io/fyne/v2/app, imported by
// main.go) and therefore the only one where GLFWApplicationDelegate exists
// to be grafted onto at all. internal/openwith's own tests call the method
// implementations as plain C functions; this is what proves the graft
// actually lands on the class the running app's delegate is an instance of.
//
// The assertion is on DelegateRespondsToOpen, not on Install's own bool:
// Install reports false both when the class is missing and when an earlier
// call already added the methods, so it cannot tell "no bridge" from "the
// bridge is already there" and is no use as an oracle.
func TestInstall_GraftsOntoGLFWsDelegate(t *testing.T) {
	openwith.Install()

	if !openwith.DelegateRespondsToOpen() {
		t.Error("DelegateRespondsToOpen() = false after Install - the open-document methods are not on GLFW's delegate class, so macOS will keep ignoring \"Open With\"")
	}
}
