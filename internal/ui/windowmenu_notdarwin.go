//go:build !darwin

package ui

import "fyne.io/fyne/v2"

func mergeNativeWindowMenu() {}

func applyUnmodifiedNativeAccelerators(_ *fyne.MainMenu) {}
