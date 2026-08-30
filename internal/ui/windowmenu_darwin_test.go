//go:build darwin

package ui

import "testing"

func newTestMenu(t *testing.T, title string) uintptr {
	t.Helper()
	menu := testNewMenu(title)
	t.Cleanup(func() { testReleaseMenu(menu) })
	return menu
}

func TestNativeMenuTitleCopiesHaveIndependentLifetime(t *testing.T) {
	menu := newTestMenu(t, "Window")
	testAddItem(menu, "Viewer", false)
	testAddItem(menu, "EXIF Data", false)

	first, second := testHeldItemTitles(menu, 0, 1)
	if first != "Viewer" || second != "EXIF Data" {
		t.Fatalf("held titles = %q, %q; want Viewer, EXIF Data", first, second)
	}
}

func TestMergeWindowMenus_MovesItemsOntoSystemWindowMenu(t *testing.T) {
	main := newTestMenu(t, "")
	file := newTestMenu(t, "File")
	testAddItem(file, "Open", false)
	testAddTopLevel(main, file)

	ours := newTestMenu(t, "Window")
	testAddItem(ours, "Viewer", false)
	testAddItem(ours, "EXIF Data", false)
	testAddTopLevel(main, ours)

	system := newTestMenu(t, "Window")
	testAddItem(system, "Minimize", false)
	testAddItem(system, "Zoom", false)
	testAddTopLevel(main, system)

	if got := testTopLevelCount(main); got != 3 {
		t.Fatalf("top-level menus before merge = %d, want 3", got)
	}

	if !mergeWindowMenus(main, system, "Window") {
		t.Fatal("merge should move the PicFetch Window menu onto the system one")
	}
	if got := testTopLevelCount(main); got != 2 {
		t.Fatalf("top-level menus after merge = %d, want 2 (File + system Window)", got)
	}
	if title := testTopLevelSubmenuTitle(main, 0); title != "File" {
		t.Errorf("first remaining menu = %q, want File", title)
	}

	want := []struct {
		title string
		sep   bool
	}{
		{"Viewer", false},
		{"EXIF Data", false},
		{"", true},
		{"Minimize", false},
		{"Zoom", false},
	}
	if n := testItemCount(system); n != len(want) {
		t.Fatalf("system Window items = %d, want %d", n, len(want))
	}
	for i, w := range want {
		sep := testItemIsSeparator(system, i)
		if sep != w.sep {
			t.Errorf("item %d separator = %v, want %v", i, sep, w.sep)
		}
		if !w.sep {
			if title := testItemTitle(system, i); title != w.title {
				t.Errorf("item %d title = %q, want %q", i, title, w.title)
			}
		}
	}
}

func TestMergeWindowMenus_MatchesLocalizedPicFetchTitle(t *testing.T) {
	main := newTestMenu(t, "")
	ours := newTestMenu(t, "Fenster")
	testAddItem(ours, "Bildanzeige", false)
	testAddTopLevel(main, ours)

	system := newTestMenu(t, "Window")
	testAddItem(system, "Minimize", false)
	testAddTopLevel(main, system)

	if !mergeWindowMenus(main, system, "Fenster") {
		t.Fatal("merge should find the PicFetch menu by lang.L label even when GLFW left the system title in English")
	}
	if got := testTopLevelCount(main); got != 1 {
		t.Fatalf("top-level menus after merge = %d, want 1", got)
	}
	if title := testItemTitle(system, 0); title != "Bildanzeige" {
		t.Errorf("first system item = %q, want Bildanzeige", title)
	}
}

func TestMergeWindowMenus_SecondCallIsNoop(t *testing.T) {
	main := newTestMenu(t, "")
	ours := newTestMenu(t, "Window")
	testAddItem(ours, "Viewer", false)
	testAddTopLevel(main, ours)
	system := newTestMenu(t, "Window")
	testAddItem(system, "Minimize", false)
	testAddTopLevel(main, system)

	if !mergeWindowMenus(main, system, "Window") {
		t.Fatal("first merge should succeed")
	}
	n := testItemCount(system)
	if mergeWindowMenus(main, system, "Window") {
		t.Fatal("second merge should be a no-op once the duplicate is gone")
	}
	if got := testItemCount(system); got != n {
		t.Fatalf("second merge changed item count %d -> %d (extra separator?)", n, got)
	}
}

func TestMergeWindowMenus_FoldsEveryDuplicate(t *testing.T) {
	main := newTestMenu(t, "")
	first := newTestMenu(t, "Window")
	testAddItem(first, "Viewer", false)
	testAddTopLevel(main, first)
	second := newTestMenu(t, "Window")
	testAddItem(second, "EXIF Data", false)
	testAddTopLevel(main, second)
	system := newTestMenu(t, "Window")
	testAddItem(system, "Minimize", false)
	testAddTopLevel(main, system)

	folded := 0
	for mergeWindowMenus(main, system, "Window") {
		folded++
	}
	if folded != 2 {
		t.Fatalf("folded %d Window menus, want 2", folded)
	}
	if got := testTopLevelCount(main); got != 1 {
		t.Fatalf("top-level after folding every duplicate = %d, want 1", got)
	}
}

func TestSetMenuItemModifierMask_ClearsDefaultCommand(t *testing.T) {
	m := newTestMenu(t, "Actions")
	testAddItemWithKey(m, "Toggle merge mode", "m")

	const nsEventModifierFlagCommand = 1 << 20
	if got := testItemModifierMask(m, 0); got&nsEventModifierFlagCommand == 0 {
		t.Fatalf("default keyEquivalentModifierMask = %d, want Command set", got)
	}
	if !setMenuItemModifierMask(m, "Toggle merge mode", 0) {
		t.Fatal("should find the item by title")
	}
	if got := testItemModifierMask(m, 0); got != 0 {
		t.Errorf("mask after clear = %d, want 0 (unmodified M, not ⌘M)", got)
	}
}

func TestMergeWindowMenus_NoDuplicateIsNoop(t *testing.T) {
	main := newTestMenu(t, "")
	system := newTestMenu(t, "Window")
	testAddItem(system, "Minimize", false)
	testAddTopLevel(main, system)

	if mergeWindowMenus(main, system, "Window") {
		t.Fatal("merge should no-op when there is only the system Window menu")
	}
	if got := testItemCount(system); got != 1 {
		t.Fatalf("system Window items = %d, want 1", got)
	}
}
