// Package displays inspects attached desktop displays in native pixel space.
package displays

import (
	"fmt"
	"image"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/lang"
)

// ID is an opaque platform identifier. Callers may compare and retain it for
// the current attached-display session but must not parse it.
type ID string

// Display describes one attached desktop display in native pixels.
type Display struct {
	ID     ID
	Name   string
	Bounds image.Rectangle
}

// Snapshot is one ordered inspection and its default target.
type Snapshot struct {
	Displays []Display
	Default  ID
}

// UnsupportedError reports that the active window/display backend cannot
// truthfully expose global native display topology.
type UnsupportedError struct {
	Reason string
}

func (e *UnsupportedError) Error() string {
	if e.Reason == "" {
		return "display inspection is unsupported"
	}

	return "display inspection is unsupported: " + e.Reason
}

// EmptyError reports a supported inspection that found no attached display.
type EmptyError struct{}

func (*EmptyError) Error() string { return "no attached displays found" }

// InvalidTopologyError reports malformed or internally inconsistent native
// display data.
type InvalidTopologyError struct {
	Reason string
}

func (e *InvalidTopologyError) Error() string { return "invalid display topology: " + e.Reason }

// Inspect is a dispatcher var so UI tests can replace native inspection with
// deterministic fixtures through internal/uitest.
var Inspect = inspectWindow

func inspectWindow(window fyne.Window) (Snapshot, error) {
	native, ok := window.(driver.NativeWindow)
	if !ok {
		return Snapshot{}, &UnsupportedError{Reason: "window has no native handle"}
	}

	var displays []Display
	var defaultID ID
	var inspectErr error
	native.RunNative(func(context any) {
		displays, defaultID, inspectErr = platformInspect(context)
	})
	if inspectErr != nil {
		return Snapshot{}, inspectErr
	}

	return newSnapshot(displays, defaultID)
}

func newSnapshot(displays []Display, defaultID ID) (Snapshot, error) {
	if len(displays) == 0 {
		return Snapshot{}, &EmptyError{}
	}
	seen := make(map[ID]struct{}, len(displays))
	for index := range displays {
		display := &displays[index]
		if display.ID == "" {
			return Snapshot{}, &InvalidTopologyError{Reason: fmt.Sprintf("display %d has an empty ID", index)}
		}
		if display.Bounds.Empty() {
			return Snapshot{}, &InvalidTopologyError{Reason: fmt.Sprintf("display %q has empty bounds", display.ID)}
		}
		if _, exists := seen[display.ID]; exists {
			return Snapshot{}, &InvalidTopologyError{Reason: fmt.Sprintf("display ID %q is duplicated", display.ID)}
		}
		seen[display.ID] = struct{}{}
		if display.Name == "" {
			display.Name = fmt.Sprintf(lang.L("Display %d"), index+1)
		}
	}
	if defaultID == "" {
		defaultID = displays[0].ID
	}
	if _, exists := seen[defaultID]; !exists {
		return Snapshot{}, &InvalidTopologyError{Reason: fmt.Sprintf("default display %q is not attached", defaultID)}
	}

	return Snapshot{Displays: slices.Clone(displays), Default: defaultID}, nil
}

func defaultForWindow(displays []Display, window image.Rectangle, windowKnown bool) ID {
	if len(displays) == 0 {
		return ""
	}
	best := displays[0].ID
	if !windowKnown {
		return best
	}
	var bestArea int64 = -1
	for _, display := range displays {
		intersection := display.Bounds.Intersect(window)
		area := int64(intersection.Dx()) * int64(intersection.Dy())
		if area > bestArea {
			best, bestArea = display.ID, area
		}
	}

	return best
}
