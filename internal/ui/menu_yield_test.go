package ui

import (
	"image/color"
	"reflect"
	"testing"

	"github.com/frathe/picfetch/internal/ui/menus"
	"github.com/frathe/picfetch/internal/uitest"
)

// Every menus.Callbacks field must run through the viewer's yield unless it
// is one of the exempt keep-the-mode commands. Enumerated by reflection so
// a field added to menus.Callbacks later fails here until it is either
// wrapped in yieldingMenuCallbacks or added to this exempt list on purpose.
func TestYieldingMenuCallbacksWrapsEveryField(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))

	exempt := map[string]bool{
		"ZoomIn":        true, // zoom keeps the mode — see yieldingMenuCallbacks
		"ZoomOut":       true,
		"CopyImage":     true, // context-routes an active selection before falling back
		"CopySelection": true, // Copy Selection itself must not cancel the mode
	}

	var c menus.Callbacks
	cv := reflect.ValueOf(&c).Elem()
	ct := cv.Type()

	ran := make(map[string]int, ct.NumField())
	for i := 0; i < ct.NumField(); i++ {
		field := ct.Field(i)
		if field.Type.Kind() != reflect.Func {
			continue
		}
		name := field.Name
		cv.Field(i).Set(reflect.MakeFunc(field.Type, func([]reflect.Value) []reflect.Value {
			ran[name]++
			return nil
		}))
	}

	wrapped := reflect.ValueOf(v.yieldingMenuCallbacks(c))

	for i := 0; i < ct.NumField(); i++ {
		field := ct.Field(i)
		if field.Type.Kind() != reflect.Func {
			continue
		}

		v.startRegionCopy()
		if !v.regionCopy.State().Active {
			t.Fatalf("%s: could not start Copy Selection for the probe", field.Name)
		}

		args := make([]reflect.Value, field.Type.NumIn())
		for j := range args {
			args[j] = reflect.Zero(field.Type.In(j))
		}
		wrapped.Field(i).Call(args)

		active := v.regionCopy.State().Active
		switch {
		case exempt[field.Name] && !active:
			t.Errorf("%s is exempt from the yield but ended Copy Selection", field.Name)
		case !exempt[field.Name] && active:
			t.Errorf("%s does not yield Copy Selection — wrap it in yieldingMenuCallbacks or, if it must keep the mode, add it to this test's exempt list", field.Name)
		}
		v.cancelRegionCopy()

		if ran[field.Name] != 1 {
			t.Errorf("%s ran %d times, want exactly once", field.Name, ran[field.Name])
		}
	}
}
