//go:build darwin && !no_native_menus

package ui

import "testing"

func TestInWindowMenuBarFollowsOSMenu(t *testing.T) {
	if inWindowMenuBar() {
		t.Fatal("the system menu bar should stay in place during picture-frame mode")
	}
}
