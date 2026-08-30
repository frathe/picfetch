//go:build darwin

package ui

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#import <AppKit/AppKit.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

// Fyne's Darwin driver tags its NSMenuItems at menuTagMin (5000) plus an
// index into its callback table. clearNativeMenu then removes any subitem
// with tag >= 5000 from GLFW's untagged Window menu. Our merge separator
// must live in that range so Refresh does not leave a growing stack of
// separators, and must sit well above PicFetch's own callback IDs.
static const NSInteger fyneWindowMergeSeparatorTag = 105000;

static char *copyTitle(NSString *s) {
	const char *utf = [s UTF8String];
	return strdup(utf == NULL ? "" : utf);
}

int mergeWindowMenus(uintptr_t mainMenuPtr, uintptr_t systemWinPtr, const char *ourLabel) {
	NSMenu *main = (__bridge NSMenu *)(void *)mainMenuPtr;
	NSMenu *systemWin = (__bridge NSMenu *)(void *)systemWinPtr;
	if (main == nil || systemWin == nil || ourLabel == NULL) {
		return 0;
	}
	NSString *label = [NSString stringWithUTF8String:ourLabel];
	if (label == nil) {
		return 0;
	}

	NSArray<NSMenuItem *> *top = [[main itemArray] copy];
	for (NSMenuItem *item in top) {
		NSMenu *sub = item.submenu;
		if (sub == nil || sub == systemWin) {
			continue;
		}
		if (![sub.title isEqualToString:label] && ![sub.title isEqualToString:systemWin.title]) {
			continue;
		}

		NSInteger insert = 0;
		NSArray<NSMenuItem *> *ours = [[sub itemArray] copy];
		for (NSMenuItem *it in ours) {
			[sub removeItem:it];
			[systemWin insertItem:it atIndex:insert];
			insert++;
		}
		NSMenuItem *sep = [NSMenuItem separatorItem];
		[sep setTag:fyneWindowMergeSeparatorTag];
		[systemWin insertItem:sep atIndex:insert];
		[main removeItem:item];
		return 1;
	}
	return 0;
}

void mergeAppWindowMenus(const char *ourLabel) {
	NSMenu *main = [NSApp mainMenu];
	NSMenu *systemWin = [NSApp windowsMenu];
	if (main == nil || systemWin == nil) {
		return;
	}
	uintptr_t mainPtr = (uintptr_t)(__bridge void *)main;
	uintptr_t sysPtr = (uintptr_t)(__bridge void *)systemWin;
	while (mergeWindowMenus(mainPtr, sysPtr, ourLabel) != 0) {
	}
}

uintptr_t testNewMenu(const char *title) {
	NSString *t = [NSString stringWithUTF8String:title ? title : ""];
	NSMenu *m = [[NSMenu alloc] initWithTitle:t];
	return (uintptr_t)CFBridgingRetain(m);
}

void testReleaseMenu(uintptr_t menuPtr) {
	if (menuPtr != 0) {
		CFRelease((CFTypeRef)(void *)menuPtr);
	}
}

void testAddItem(uintptr_t menuPtr, const char *title, int isSep) {
	NSMenu *m = (__bridge NSMenu *)(void *)menuPtr;
	NSMenuItem *item;
	if (isSep) {
		item = [NSMenuItem separatorItem];
	} else {
		NSString *t = [NSString stringWithUTF8String:title ? title : ""];
		item = [[NSMenuItem alloc] initWithTitle:t action:nil keyEquivalent:@""];
	}
	[m addItem:item];
}

void testAddTopLevel(uintptr_t mainPtr, uintptr_t submenuPtr) {
	NSMenu *main = (__bridge NSMenu *)(void *)mainPtr;
	NSMenu *sub = (__bridge NSMenu *)(void *)submenuPtr;
	NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:@"" action:nil keyEquivalent:@""];
	[item setSubmenu:sub];
	[main addItem:item];
}

int testTopLevelCount(uintptr_t mainPtr) {
	NSMenu *main = (__bridge NSMenu *)(void *)mainPtr;
	return (int)[main numberOfItems];
}

char *testTopLevelSubmenuTitle(uintptr_t mainPtr, int index) {
	NSMenu *main = (__bridge NSMenu *)(void *)mainPtr;
	NSMenu *sub = [[main itemAtIndex:index] submenu];
	return copyTitle(sub == nil ? @"" : sub.title);
}

int testItemCount(uintptr_t menuPtr) {
	NSMenu *m = (__bridge NSMenu *)(void *)menuPtr;
	return (int)[m numberOfItems];
}

int testItemIsSeparator(uintptr_t menuPtr, int index) {
	NSMenu *m = (__bridge NSMenu *)(void *)menuPtr;
	return [[m itemAtIndex:index] isSeparatorItem] ? 1 : 0;
}

char *testItemTitle(uintptr_t menuPtr, int index) {
	NSMenu *m = (__bridge NSMenu *)(void *)menuPtr;
	return copyTitle([[m itemAtIndex:index] title]);
}

static int applyModifierMaskInSubmenu(NSMenu *sub, NSString *itemTitle, unsigned int mask) {
	if (sub == nil || itemTitle == nil) {
		return 0;
	}
	for (NSMenuItem *item in [sub itemArray]) {
		if ([item.title isEqualToString:itemTitle]) {
			[item setKeyEquivalentModifierMask:mask];
			return 1;
		}
		NSMenu *child = [item submenu];
		if (child != nil && applyModifierMaskInSubmenu(child, itemTitle, mask)) {
			return 1;
		}
	}
	return 0;
}

// Fyne's insertDarwinMenuItem only calls setKeyEquivalentModifierMask when
// the mask is non-zero, so an unmodified CustomShortcut keeps AppKit's
// default Command (⌘M = Minimize). Walk the live bar and clear that.
int setNativeMenuItemModifierMask(const char *menuTitle, const char *itemTitle, unsigned int mask) {
	if (itemTitle == NULL) {
		return 0;
	}
	NSString *it = [NSString stringWithUTF8String:itemTitle];
	if (it == nil) {
		return 0;
	}
	NSMenu *main = [NSApp mainMenu];
	if (main != nil && menuTitle != NULL) {
		NSString *mt = [NSString stringWithUTF8String:menuTitle];
		if (mt != nil) {
			for (NSMenuItem *top in [main itemArray]) {
				NSMenu *sub = [top submenu];
				if (sub == nil) {
					continue;
				}
				if (![sub.title isEqualToString:mt] && ![top.title isEqualToString:mt]) {
					continue;
				}
				if (applyModifierMaskInSubmenu(sub, it, mask)) {
					return 1;
				}
			}
		}
	}
	return applyModifierMaskInSubmenu([NSApp windowsMenu], it, mask);
}

void testAddItemWithKey(uintptr_t menuPtr, const char *title, const char *key) {
	NSMenu *m = (__bridge NSMenu *)(void *)menuPtr;
	if (m == nil) {
		return;
	}
	NSString *t = [NSString stringWithUTF8String:title ? title : ""];
	NSString *k = [NSString stringWithUTF8String:key ? key : ""];
	NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:t action:nil keyEquivalent:k];
	[m addItem:item];
}

unsigned long testItemModifierMask(uintptr_t menuPtr, int index) {
	NSMenu *m = (__bridge NSMenu *)(void *)menuPtr;
	return [[m itemAtIndex:index] keyEquivalentModifierMask];
}

int setMenuItemModifierMask(uintptr_t menuPtr, const char *itemTitle, unsigned int mask) {
	NSMenu *m = (__bridge NSMenu *)(void *)menuPtr;
	if (m == nil || itemTitle == NULL) {
		return 0;
	}
	return applyModifierMaskInSubmenu(m, [NSString stringWithUTF8String:itemTitle], mask);
}
*/
import "C"

import (
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
)

// mergeNativeWindowMenu folds the Fyne-created Window menu into AppKit's
// windowsMenu so macOS does not show two adjacent Window titles. No-op when
// GLFW has not installed a windowsMenu, or when there is no duplicate.
func mergeNativeWindowMenu() {
	c := C.CString(lang.L("Window"))
	defer C.free(unsafe.Pointer(c))
	C.mergeAppWindowMenus(c)
}

func mergeWindowMenus(main, system uintptr, label string) bool {
	c := C.CString(label)
	defer C.free(unsafe.Pointer(c))
	return C.mergeWindowMenus(C.uintptr_t(main), C.uintptr_t(system), c) != 0
}

func testNewMenu(title string) uintptr {
	c := C.CString(title)
	defer C.free(unsafe.Pointer(c))
	return uintptr(C.testNewMenu(c))
}

func testReleaseMenu(menu uintptr) {
	C.testReleaseMenu(C.uintptr_t(menu))
}

func testAddItem(menu uintptr, title string, sep bool) {
	c := C.CString(title)
	defer C.free(unsafe.Pointer(c))
	isSep := C.int(0)
	if sep {
		isSep = 1
	}
	C.testAddItem(C.uintptr_t(menu), c, isSep)
}

func testAddTopLevel(main, submenu uintptr) {
	C.testAddTopLevel(C.uintptr_t(main), C.uintptr_t(submenu))
}

func testTopLevelCount(main uintptr) int {
	return int(C.testTopLevelCount(C.uintptr_t(main)))
}

func testTopLevelSubmenuTitle(main uintptr, index int) string {
	return stringFromOwnedCString(C.testTopLevelSubmenuTitle(C.uintptr_t(main), C.int(index)))
}

func testItemCount(menu uintptr) int {
	return int(C.testItemCount(C.uintptr_t(menu)))
}

func testItemIsSeparator(menu uintptr, index int) bool {
	return C.testItemIsSeparator(C.uintptr_t(menu), C.int(index)) != 0
}

func testItemTitle(menu uintptr, index int) string {
	return stringFromOwnedCString(C.testItemTitle(C.uintptr_t(menu), C.int(index)))
}

func testHeldItemTitles(menu uintptr, first, second int) (string, string) {
	firstTitle := C.testItemTitle(C.uintptr_t(menu), C.int(first))
	secondTitle := C.testItemTitle(C.uintptr_t(menu), C.int(second))
	defer C.free(unsafe.Pointer(firstTitle))
	defer C.free(unsafe.Pointer(secondTitle))
	return C.GoString(firstTitle), C.GoString(secondTitle)
}

func stringFromOwnedCString(c *C.char) string {
	defer C.free(unsafe.Pointer(c))
	return C.GoString(c)
}

func testAddItemWithKey(menu uintptr, title, key string) {
	ct := C.CString(title)
	defer C.free(unsafe.Pointer(ct))
	ck := C.CString(key)
	defer C.free(unsafe.Pointer(ck))
	C.testAddItemWithKey(C.uintptr_t(menu), ct, ck)
}

func testItemModifierMask(menu uintptr, index int) uint64 {
	return uint64(C.testItemModifierMask(C.uintptr_t(menu), C.int(index)))
}

func setMenuItemModifierMask(menu uintptr, itemTitle string, mask uint) bool {
	c := C.CString(itemTitle)
	defer C.free(unsafe.Pointer(c))
	return C.setMenuItemModifierMask(C.uintptr_t(menu), c, C.uint(mask)) != 0
}

func setNativeMenuItemModifierMask(menuTitle, itemTitle string, mask uint) {
	cm := C.CString(menuTitle)
	defer C.free(unsafe.Pointer(cm))
	ci := C.CString(itemTitle)
	defer C.free(unsafe.Pointer(ci))
	C.setNativeMenuItemModifierMask(cm, ci, C.uint(mask))
}

// applyUnmodifiedNativeAccelerators clears AppKit's default Command mask on
// Fyne menu items whose CustomShortcut asked for no modifiers. Items that
// set KeyModifierShortcutDefault (Copy, Open, …) are left alone. Window
// items are searched in NSApp.windowsMenu after mergeNativeWindowMenu
// removes the duplicate Fyne Window submenu.
func applyUnmodifiedNativeAccelerators(bar *fyne.MainMenu) {
	if bar == nil {
		return
	}
	for _, menu := range bar.Items {
		if menu == nil {
			continue
		}
		applyUnmodifiedNativeItems(menu.Label, menu.Items)
	}
}

func applyUnmodifiedNativeItems(menuTitle string, items []*fyne.MenuItem) {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.ChildMenu != nil {
			title := item.ChildMenu.Label
			if title == "" {
				title = menuTitle
			}
			applyUnmodifiedNativeItems(title, item.ChildMenu.Items)
		}
		sc, ok := item.Shortcut.(*desktop.CustomShortcut)
		if !ok || sc.Modifier != 0 {
			continue
		}
		setNativeMenuItemModifierMask(menuTitle, item.Label, 0)
	}
}

// Referenced so `go build` (which skips tests) still compiles the AppKit
// harness windowmenu_darwin_test.go needs. Go forbids cgo in _test.go files.
var _ = []any{
	testNewMenu, testReleaseMenu, testAddItem, testAddTopLevel,
	testTopLevelCount, testTopLevelSubmenuTitle,
	testItemCount, testItemIsSeparator, testItemTitle, testHeldItemTitles,
	testAddItemWithKey, testItemModifierMask, setMenuItemModifierMask,
}
