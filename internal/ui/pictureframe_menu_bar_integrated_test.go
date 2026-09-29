//go:build !darwin || no_native_menus

package ui

import "testing"

func TestInWindowMenuBarFollowsOSMenu(t *testing.T) {
	if !inWindowMenuBar() {
		t.Fatal("in-window menus should slide during picture-frame mode")
	}
}
